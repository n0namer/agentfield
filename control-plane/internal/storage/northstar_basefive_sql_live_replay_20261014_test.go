package storage

import (
	"context"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Agent-Field/agentfield/control-plane/pkg/types"
)

func TestBaseFiveSQLLiveReplay(t *testing.T) {
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
		ExecutionID: "basefive-execution-1",
		RunID:       "basefive-run-1",
		AgentNodeID: "agent-1",
		NodeID:      "agent-1",
		ReasonerID:  "echo",
		Status:      types.ExecutionStatusQueued,
		StartedAt:   time.Now().UTC(),
	}
	created, err := storage.CreateAsyncExecutionIdempotent(ctx, "caller-a\x00tenant-a", "agent-1.echo", "basefive-request", "digest-a", canonical)
	if err != nil {
		t.Fatalf("create canonical execution: %v", err)
	}
	if created == nil || !created.Created || created.Execution == nil || created.Execution.ExecutionID != canonical.ExecutionID {
		t.Fatalf("unexpected canonical create result: %+v", created)
	}

	const contenders = 2
	start := make(chan struct{})
	results := make(chan *AsyncExecutionIdempotencyResult, contenders)
	errors := make(chan error, contenders)
	var wait sync.WaitGroup
	for index := 0; index < contenders; index++ {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			<-start
			result, callErr := storage.CreateAsyncExecutionIdempotent(ctx, "caller-a\x00tenant-a", "agent-1.echo", "basefive-race", "digest-race", &types.Execution{
				ExecutionID: "basefive-race-execution-" + strconv.Itoa(index),
				RunID:       "basefive-race-run",
				AgentNodeID: "agent-1",
				NodeID:      "agent-1",
				ReasonerID:  "echo",
				Status:      types.ExecutionStatusQueued,
				StartedAt:   time.Now().UTC(),
			})
			if callErr != nil {
				errors <- callErr
				return
			}
			results <- result
		}(index)
	}
	close(start)
	wait.Wait()
	close(results)
	close(errors)

	createdCount := 0
	canonicalRaceID := ""
	for result := range results {
		if result == nil || result.Conflict || result.Execution == nil {
			t.Fatalf("unexpected concurrent replay result: %+v", result)
		}
		if result.Created {
			createdCount++
		}
		if canonicalRaceID == "" {
			canonicalRaceID = result.Execution.ExecutionID
		} else if result.Execution.ExecutionID != canonicalRaceID {
			t.Fatalf("concurrent same-key calls returned different executions: %q and %q", canonicalRaceID, result.Execution.ExecutionID)
		}
	}
	for callErr := range errors {
		t.Fatalf("concurrent same-key call failed: %v", callErr)
	}
	if createdCount != 1 || canonicalRaceID == "" {
		t.Fatalf("expected one concurrent winner and one replay, created=%d canonical=%q", createdCount, canonicalRaceID)
	}

	foreignDigest, err := storage.LookupAsyncExecutionIdempotency(ctx, "caller-a\x00tenant-a", "agent-1.echo", "basefive-request", "digest-foreign")
	if err != nil {
		t.Fatalf("lookup foreign digest: %v", err)
	}
	if foreignDigest == nil || !foreignDigest.Conflict || foreignDigest.Execution != nil {
		t.Fatalf("foreign digest was not rejected: %+v", foreignDigest)
	}

	if err := storage.Close(ctx); err != nil {
		t.Fatalf("close native SQL storage before restart: %v", err)
	}
	restarted := NewLocalStorage(LocalStorageConfig{})
	if err := restarted.Initialize(ctx, config); err != nil {
		t.Fatalf("reinitialize native SQL storage: %v", err)
	}
	applyBaseFiveIdempotencyMigration(t, restarted)
	defer restarted.Close(ctx)

	replayed, err := restarted.LookupAsyncExecutionIdempotency(ctx, "caller-a\x00tenant-a", "agent-1.echo", "basefive-request", "digest-a")
	if err != nil {
		t.Fatalf("lookup canonical execution after restart: %v", err)
	}
	if replayed == nil || replayed.Conflict || replayed.Execution == nil || replayed.Execution.ExecutionID != canonical.ExecutionID {
		t.Fatalf("restart did not return canonical execution: %+v", replayed)
	}
}

func applyBaseFiveIdempotencyMigration(t *testing.T, storage *LocalStorage) {
	t.Helper()
	ctx := context.Background()
	if _, err := storage.requireSQLDB().ExecContext(ctx, `
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
	if _, err := storage.requireSQLDB().ExecContext(ctx, `
		CREATE INDEX IF NOT EXISTS idx_execution_request_idempotency_execution
		ON execution_request_idempotency(execution_id)`); err != nil {
		t.Fatalf("apply execution idempotency index migration: %v", err)
	}
}
