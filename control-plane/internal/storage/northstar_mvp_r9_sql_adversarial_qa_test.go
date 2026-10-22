package storage

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Agent-Field/agentfield/control-plane/pkg/types"
)

func TestMVPR9SQLAtomicReplayAdversarialQA(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	config := StorageConfig{Mode: "local", Local: LocalStorageConfig{
		DatabasePath: filepath.Join(tempDir, "agentfield.db"),
		KVStorePath:  filepath.Join(tempDir, "agentfield.bolt"),
	}}

	storage := NewLocalStorage(LocalStorageConfig{})
	if err := storage.Initialize(ctx, config); err != nil {
		if strings.Contains(err.Error(), "no such module: fts5") {
			t.Skip("sqlite3 compiled without FTS5")
		}
		t.Fatalf("initialize native SQL storage: %v", err)
	}
	applyBaseFiveIdempotencyMigration(t, storage)

	canonical := &types.Execution{
		ExecutionID: "r9-canonical-execution",
		RunID:       "r9-canonical-run",
		AgentNodeID: "agent-r9",
		NodeID:      "agent-r9",
		ReasonerID:  "echo",
		Status:      types.ExecutionStatusQueued,
		StartedAt:   time.Now().UTC(),
	}
	created, err := storage.CreateAsyncExecutionIdempotent(ctx, "caller-a\x00tenant-a", "agent-r9.echo", "r9-request", "digest-a", canonical)
	if err != nil {
		t.Fatalf("create canonical execution: %v", err)
	}
	if created == nil || !created.Created || created.Execution == nil {
		t.Fatalf("expected canonical SQL winner: %+v", created)
	}

	t.Run("nil payload fails closed without mutation", func(t *testing.T) {
		_, err := storage.CreateAsyncExecutionIdempotent(ctx, "caller-a\x00tenant-a", "agent-r9.echo", "r9-nil", "digest-nil", nil)
		if err == nil || !strings.Contains(err.Error(), "nil execution payload") {
			t.Fatalf("nil payload was not rejected: %v", err)
		}
		assertR9Counts(t, storage.requireSQLDB().DB, "caller-a\x00tenant-a", "r9-request", 1, 1)
	})

	t.Run("canceled request fails closed without mutation", func(t *testing.T) {
		canceled, cancel := context.WithCancel(ctx)
		cancel()
		_, err := storage.CreateAsyncExecutionIdempotent(canceled, "caller-a\x00tenant-a", "agent-r9.echo", "r9-canceled", "digest-canceled", &types.Execution{
			ExecutionID: "r9-canceled-execution",
			RunID:       "r9-canceled-run",
			AgentNodeID: "agent-r9",
			NodeID:      "agent-r9",
			ReasonerID:  "echo",
			Status:      types.ExecutionStatusQueued,
		})
		if err == nil {
			t.Fatal("canceled request unexpectedly created an execution")
		}
		assertR9Counts(t, storage.requireSQLDB().DB, "caller-a\x00tenant-a", "r9-request", 1, 1)
	})

	t.Run("foreign scope cannot replay or inspect canonical execution", func(t *testing.T) {
		foreign, err := storage.LookupAsyncExecutionIdempotency(ctx, "caller-b\x00tenant-b", "agent-r9.echo", "r9-request", "digest-a")
		if err != nil {
			t.Fatalf("lookup from foreign scope: %v", err)
		}
		if foreign == nil || foreign.Execution != nil || foreign.Conflict || foreign.Created {
			t.Fatalf("foreign scope crossed the durable fence: %+v", foreign)
		}
	})

	t.Run("different digest conflicts without replacing canonical execution", func(t *testing.T) {
		conflict, err := storage.CreateAsyncExecutionIdempotent(ctx, "caller-a\x00tenant-a", "agent-r9.echo", "r9-request", "digest-b", &types.Execution{
			ExecutionID: "r9-foreign-execution",
			RunID:       "r9-foreign-run",
			AgentNodeID: "agent-r9",
			NodeID:      "agent-r9",
			ReasonerID:  "echo",
			Status:      types.ExecutionStatusQueued,
		})
		if err != nil {
			t.Fatalf("different digest conflict lookup: %v", err)
		}
		if conflict == nil || !conflict.Conflict || conflict.Execution != nil || conflict.Created {
			t.Fatalf("different digest did not fail closed: %+v", conflict)
		}
		assertR9Counts(t, storage.requireSQLDB().DB, "caller-a\x00tenant-a", "r9-request", 1, 1)
	})

	if err := storage.Close(ctx); err != nil {
		t.Fatalf("close storage before restart: %v", err)
	}
	restarted := NewLocalStorage(LocalStorageConfig{})
	if err := restarted.Initialize(ctx, config); err != nil {
		t.Fatalf("restart native SQL storage: %v", err)
	}
	applyBaseFiveIdempotencyMigration(t, restarted)
	defer restarted.Close(ctx)

	replayed, err := restarted.CreateAsyncExecutionIdempotent(ctx, "caller-a\x00tenant-a", "agent-r9.echo", "r9-request", "digest-a", &types.Execution{
		ExecutionID: "r9-after-restart-execution",
		RunID:       "r9-after-restart-run",
		AgentNodeID: "agent-r9",
		NodeID:      "agent-r9",
		ReasonerID:  "echo",
		Status:      types.ExecutionStatusQueued,
	})
	if err != nil {
		t.Fatalf("replay after restart: %v", err)
	}
	if replayed == nil || replayed.Created || replayed.Conflict || replayed.Execution == nil || replayed.Execution.ExecutionID != canonical.ExecutionID {
		t.Fatalf("restart replay changed canonical execution: %+v", replayed)
	}
	assertR9Counts(t, restarted.requireSQLDB().DB, "caller-a\x00tenant-a", "r9-request", 1, 1)
}

func assertR9Counts(t *testing.T, db *sql.DB, scopeKey, requestKey string, expectedBindingCount, expectedExecutionCount int) {
	t.Helper()
	var bindingCount, executionCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM execution_request_idempotency WHERE scope_key = ? AND request_key = ?`, scopeKey, requestKey).Scan(&bindingCount); err != nil {
		t.Fatalf("count idempotency binding: %v", err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM executions`).Scan(&executionCount); err != nil {
		t.Fatalf("count executions: %v", err)
	}
	if bindingCount != expectedBindingCount || executionCount != expectedExecutionCount {
		t.Fatalf("unexpected durable counts: bindings=%d executions=%d, want bindings=%d executions=%d", bindingCount, executionCount, expectedBindingCount, expectedExecutionCount)
	}
}
