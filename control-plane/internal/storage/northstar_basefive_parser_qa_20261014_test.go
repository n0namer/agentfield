package storage

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Agent-Field/agentfield/control-plane/pkg/types"
)

func TestBaseFiveSQLParserQA(t *testing.T) {
	ctx := context.Background()
	databasePath := filepath.Join(t.TempDir(), "basefive-parser-qa.db")
	config := StorageConfig{Mode: "local", Local: LocalStorageConfig{
		DatabasePath: databasePath,
		KVStorePath:  filepath.Join(t.TempDir(), "basefive-parser-qa.bolt"),
	}}

	storage := NewLocalStorage(LocalStorageConfig{})
	if err := storage.Initialize(ctx, config); err != nil {
		if strings.Contains(err.Error(), "no such module: fts5") {
			t.Skip("sqlite3 compiled without FTS5")
		}
		t.Fatalf("initialize native SQL storage: %v", err)
	}
	if _, err := storage.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS execution_request_idempotency (
			scope_key TEXT NOT NULL,
			target TEXT NOT NULL,
			request_key TEXT NOT NULL,
			body_digest TEXT NOT NULL,
			execution_id TEXT NOT NULL,
			run_id TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			PRIMARY KEY (scope_key, request_key),
			UNIQUE (execution_id)
		)`); err != nil {
		t.Fatalf("apply execution idempotency migration: %v", err)
	}
	if _, err := storage.db.ExecContext(ctx, `
		CREATE INDEX IF NOT EXISTS idx_execution_request_idempotency_execution
		ON execution_request_idempotency(execution_id)`); err != nil {
		t.Fatalf("create execution idempotency migration index: %v", err)
	}

	canonical := &types.Execution{
		ExecutionID: "basefive-parser-execution",
		RunID:       "basefive-parser-run",
		AgentNodeID: "agent-basefive",
		NodeID:      "agent-basefive",
		ReasonerID:  "echo",
		Status:      types.ExecutionStatusQueued,
		StartedAt:   time.Now().UTC(),
	}
	created, err := storage.CreateAsyncExecutionIdempotent(ctx, "basefive-owner", "agent-basefive.echo", "basefive-request", "digest-a", canonical)
	if err != nil {
		t.Fatalf("create canonical execution: %v", err)
	}
	if created == nil || !created.Created || created.Execution == nil || created.Execution.ExecutionID != canonical.ExecutionID {
		t.Fatalf("positive create assertion failed: %+v", created)
	}

	replayed, err := storage.LookupAsyncExecutionIdempotency(ctx, "basefive-owner", "agent-basefive.echo", "basefive-request", "digest-a")
	if err != nil {
		t.Fatalf("lookup canonical execution: %v", err)
	}
	if replayed == nil || replayed.Conflict || replayed.Execution == nil || replayed.Execution.ExecutionID != canonical.ExecutionID {
		t.Fatalf("positive replay assertion failed: %+v", replayed)
	}

	for name, target := range map[string]string{
		"conflicting digest": "agent-basefive.echo",
		"conflicting target": "agent-other.echo",
	} {
		digestValue := "digest-a"
		if name == "conflicting digest" {
			digestValue = "digest-other"
		}
		conflict, lookupErr := storage.LookupAsyncExecutionIdempotency(ctx, "basefive-owner", target, "basefive-request", digestValue)
		if lookupErr != nil {
			t.Fatalf("%s lookup: %v", name, lookupErr)
		}
		if conflict == nil || !conflict.Conflict || conflict.Execution != nil {
			t.Fatalf("%s was not rejected fail-closed: %+v", name, conflict)
		}
	}

	if err := storage.Close(ctx); err != nil {
		t.Fatalf("close storage before restart: %v", err)
	}

	restarted := NewLocalStorage(LocalStorageConfig{})
	if err := restarted.Initialize(ctx, config); err != nil {
		t.Fatalf("reinitialize storage after restart: %v", err)
	}
	defer restarted.Close(ctx)

	restartedReplay, err := restarted.LookupAsyncExecutionIdempotency(ctx, "basefive-owner", "agent-basefive.echo", "basefive-request", "digest-a")
	if err != nil {
		t.Fatalf("lookup canonical execution after restart: %v", err)
	}
	if restartedReplay == nil || restartedReplay.Conflict || restartedReplay.Execution == nil || restartedReplay.Execution.ExecutionID != canonical.ExecutionID {
		t.Fatalf("restart persistence assertion failed: %+v", restartedReplay)
	}
}
