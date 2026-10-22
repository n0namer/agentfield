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

func newIdempotencyTestStorage(t *testing.T) *LocalStorage {
	t.Helper()
	ctx := context.Background()
	storage := NewLocalStorage(LocalStorageConfig{})
	config := StorageConfig{Mode: "local", Local: LocalStorageConfig{
		DatabasePath: filepath.Join(t.TempDir(), "agentfield.db"),
		KVStorePath:  filepath.Join(t.TempDir(), "agentfield.bolt"),
	}}
	if err := storage.Initialize(ctx, config); err != nil {
		if strings.Contains(err.Error(), "no such module: fts5") {
			t.Skip("sqlite3 compiled without FTS5")
		}
		t.Fatalf("initialize storage: %v", err)
	}
	t.Cleanup(func() { _ = storage.Close(ctx) })
	return storage
}

func TestCreateAsyncExecutionIdempotent_ReplaysAndConflicts(t *testing.T) {
	storage := newIdempotencyTestStorage(t)
	ctx := context.Background()
	now := time.Now().UTC()
	execution := &types.Execution{ExecutionID: "exec-idem-1", RunID: "run-idem-1", AgentNodeID: "agent-1", NodeID: "agent-1", ReasonerID: "echo", Status: types.ExecutionStatusRunning, StartedAt: now}

	created, err := storage.CreateAsyncExecutionIdempotent(ctx, "caller\x00tenant", "agent-1.echo", "request-1", "digest-a", execution)
	if err != nil {
		t.Fatalf("create idempotent execution: %v", err)
	}
	if !created.Created || created.Execution == nil {
		t.Fatalf("expected a newly created binding: %#v", created)
	}

	replayed, err := storage.CreateAsyncExecutionIdempotent(ctx, "caller\x00tenant", "agent-1.echo", "request-1", "digest-a", &types.Execution{ExecutionID: "exec-idem-2", RunID: "run-idem-2"})
	if err != nil {
		t.Fatalf("replay idempotent execution: %v", err)
	}
	if replayed.Created || replayed.Conflict || replayed.Execution.ExecutionID != "exec-idem-1" {
		t.Fatalf("expected canonical replay, got %#v", replayed)
	}

	conflict, err := storage.CreateAsyncExecutionIdempotent(ctx, "caller\x00tenant", "agent-1.echo", "request-1", "digest-b", &types.Execution{ExecutionID: "exec-idem-3", RunID: "run-idem-3"})
	if err != nil {
		t.Fatalf("conflict lookup: %v", err)
	}
	if !conflict.Conflict {
		t.Fatalf("expected digest conflict, got %#v", conflict)
	}

	targetConflict, err := storage.CreateAsyncExecutionIdempotent(ctx, "caller\x00tenant", "agent-2.echo", "request-1", "digest-a", &types.Execution{ExecutionID: "exec-idem-4", RunID: "run-idem-4"})
	if err != nil {
		t.Fatalf("target conflict lookup: %v", err)
	}
	if !targetConflict.Conflict {
		t.Fatalf("expected target conflict, got %#v", targetConflict)
	}
}

func TestCreateAsyncExecutionIdempotent_ConcurrentWinnerOnly(t *testing.T) {
	storage := newIdempotencyTestStorage(t)
	ctx := context.Background()
	const contenders = 8
	var wait sync.WaitGroup
	wait.Add(contenders)
	results := make(chan *AsyncExecutionIdempotencyResult, contenders)
	errors := make(chan error, contenders)
	for index := 0; index < contenders; index++ {
		index := index
		go func() {
			defer wait.Done()
			execution := &types.Execution{ExecutionID: "exec-race-" + strconv.Itoa(index), RunID: "run-race", AgentNodeID: "agent-1", NodeID: "agent-1", ReasonerID: "echo", Status: types.ExecutionStatusRunning}
			result, err := storage.CreateAsyncExecutionIdempotent(ctx, "caller\x00tenant", "agent-1.echo", "request-race", "digest", execution)
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

	created := 0
	canonical := ""
	for result := range results {
		if result.Created {
			created++
			canonical = result.Execution.ExecutionID
		} else if result.Execution != nil && canonical == "" {
			canonical = result.Execution.ExecutionID
		}
	}
	for err := range errors {
		t.Fatalf("concurrent idempotency error: %v", err)
	}
	if created != 1 || canonical == "" {
		t.Fatalf("expected one winner and canonical execution, created=%d canonical=%q", created, canonical)
	}
}
