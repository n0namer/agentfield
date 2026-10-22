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

func TestNorthstarLongBundleSQLNativeReplay(t *testing.T) {
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
		t.Fatalf("initialize local storage: %v", err)
	}

	canonical := &types.Execution{
		ExecutionID: "northstar-execution-1",
		RunID:       "northstar-run-1",
		AgentNodeID: "agent-1",
		NodeID:      "agent-1",
		ReasonerID:  "echo",
		Status:      types.ExecutionStatusQueued,
		StartedAt:   time.Now().UTC(),
	}
	created, err := storage.CreateAsyncExecutionIdempotent(ctx, "caller-a\\x00tenant-a", "agent-1.echo", "northstar-request-1", "digest-a", canonical)
	if err != nil {
		t.Fatalf("create canonical execution: %v", err)
	}
	if !created.Created || created.Execution == nil || created.Execution.ExecutionID != canonical.ExecutionID {
		t.Fatalf("unexpected create result: %+v", created)
	}

	replayed, err := storage.CreateAsyncExecutionIdempotent(ctx, "caller-a\\x00tenant-a", "agent-1.echo", "northstar-request-1", "digest-a", &types.Execution{ExecutionID: "northstar-execution-2", RunID: "northstar-run-2"})
	if err != nil {
		t.Fatalf("replay canonical execution: %v", err)
	}
	if replayed.Created || replayed.Conflict || replayed.Execution == nil || replayed.Execution.ExecutionID != canonical.ExecutionID {
		t.Fatalf("expected canonical replay, got %+v", replayed)
	}

	digestConflict, err := storage.LookupAsyncExecutionIdempotency(ctx, "caller-a\\x00tenant-a", "agent-1.echo", "northstar-request-1", "digest-b")
	if err != nil {
		t.Fatalf("lookup digest conflict: %v", err)
	}
	if !digestConflict.Conflict || digestConflict.Execution != nil {
		t.Fatalf("expected digest conflict, got %+v", digestConflict)
	}

	targetConflict, err := storage.LookupAsyncExecutionIdempotency(ctx, "caller-a\\x00tenant-a", "agent-2.echo", "northstar-request-1", "digest-a")
	if err != nil {
		t.Fatalf("lookup target conflict: %v", err)
	}
	if !targetConflict.Conflict || targetConflict.Execution != nil {
		t.Fatalf("expected target conflict, got %+v", targetConflict)
	}

	const contenders = 2
	var wait sync.WaitGroup
	wait.Add(contenders)
	results := make(chan *AsyncExecutionIdempotencyResult, contenders)
	errors := make(chan error, contenders)
	for index, digest := range []string{"digest-c", "digest-d"} {
		index, digest := index, digest
		go func() {
			defer wait.Done()
			result, err := storage.CreateAsyncExecutionIdempotent(ctx, "caller-a\\x00tenant-a", "agent-1.echo", "northstar-race", digest, &types.Execution{
				ExecutionID: "northstar-race-execution-" + strconv.Itoa(index),
				RunID:       "northstar-race-run",
				AgentNodeID: "agent-1",
				NodeID:      "agent-1",
				ReasonerID:  "echo",
				Status:      types.ExecutionStatusQueued,
				StartedAt:   time.Now().UTC(),
			})
			if err != nil {
				errors <- err
				return
			}
			results <- result
		}()
	}
	wait.Wait()
	close(results)
	close(errors)

	createdCount := 0
	conflictCount := 0
	canonicalRaceID := ""
	for result := range results {
		if result == nil {
			t.Fatalf("nil concurrent result")
		}
		if result.Created {
			createdCount++
			canonicalRaceID = result.Execution.ExecutionID
		}
		if result.Conflict {
			conflictCount++
		}
	}
	for err := range errors {
		t.Fatalf("concurrent mixed-digest request: %v", err)
	}
	if createdCount != 1 || conflictCount != 1 || canonicalRaceID == "" {
		t.Fatalf("expected one winner and one digest conflict, created=%d conflicts=%d canonical=%q", createdCount, conflictCount, canonicalRaceID)
	}

	if err := storage.Close(ctx); err != nil {
		t.Fatalf("close storage for restart: %v", err)
	}
	restarted := NewLocalStorage(LocalStorageConfig{})
	if err := restarted.Initialize(ctx, config); err != nil {
		t.Fatalf("reinitialize local storage: %v", err)
	}
	defer restarted.Close(ctx)

	restartReplay, err := restarted.LookupAsyncExecutionIdempotency(ctx, "caller-a\\x00tenant-a", "agent-1.echo", "northstar-request-1", "digest-a")
	if err != nil {
		t.Fatalf("lookup after restart: %v", err)
	}
	if restartReplay.Conflict || restartReplay.Execution == nil || restartReplay.Execution.ExecutionID != canonical.ExecutionID {
		t.Fatalf("expected durable canonical replay after restart, got %+v", restartReplay)
	}

	restartConflict, err := restarted.LookupAsyncExecutionIdempotency(ctx, "caller-a\\x00tenant-a", "agent-1.echo", "northstar-request-1", "digest-b")
	if err != nil {
		t.Fatalf("lookup restart digest conflict: %v", err)
	}
	if !restartConflict.Conflict || restartConflict.Execution != nil {
		t.Fatalf("expected durable digest conflict after restart, got %+v", restartConflict)
	}
}
