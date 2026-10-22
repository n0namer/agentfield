package handlers

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Agent-Field/agentfield/control-plane/internal/storage"
	"github.com/Agent-Field/agentfield/control-plane/pkg/types"
)

type liveIdempotencyCanaryStore struct {
	mu          sync.Mutex
	records     map[string]liveIdempotencyCanaryRecord
	createCount int
}

type liveIdempotencyCanaryRecord struct {
	scopeKey   string
	target     string
	requestKey string
	bodyDigest string
	execution  *types.Execution
}

func newLiveIdempotencyCanaryStore() *liveIdempotencyCanaryStore {
	return &liveIdempotencyCanaryStore{records: make(map[string]liveIdempotencyCanaryRecord)}
}

func (s *liveIdempotencyCanaryStore) CreateAsyncExecutionIdempotent(ctx context.Context, scopeKey, target, requestKey, bodyDigest string, execution *types.Execution) (*storage.AsyncExecutionIdempotencyResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	key := scopeKey + "\x00" + requestKey
	if existing, ok := s.records[key]; ok {
		if existing.target != target || existing.bodyDigest != bodyDigest {
			return &storage.AsyncExecutionIdempotencyResult{Conflict: true}, nil
		}
		return &storage.AsyncExecutionIdempotencyResult{Execution: cloneCanaryExecution(existing.execution)}, nil
	}
	s.createCount++
	s.records[key] = liveIdempotencyCanaryRecord{
		scopeKey:   scopeKey,
		target:     target,
		requestKey: requestKey,
		bodyDigest: bodyDigest,
		execution:  cloneCanaryExecution(execution),
	}
	return &storage.AsyncExecutionIdempotencyResult{Created: true, Execution: cloneCanaryExecution(execution)}, nil
}

func (s *liveIdempotencyCanaryStore) LookupAsyncExecutionIdempotency(ctx context.Context, scopeKey, target, requestKey, bodyDigest string) (*storage.AsyncExecutionIdempotencyResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.records[scopeKey+"\x00"+requestKey]
	if !ok {
		return &storage.AsyncExecutionIdempotencyResult{}, nil
	}
	if existing.target != target || existing.bodyDigest != bodyDigest {
		return &storage.AsyncExecutionIdempotencyResult{Conflict: true}, nil
	}
	return &storage.AsyncExecutionIdempotencyResult{Execution: cloneCanaryExecution(existing.execution)}, nil
}

func cloneCanaryExecution(execution *types.Execution) *types.Execution {
	if execution == nil {
		return nil
	}
	copy := *execution
	return &copy
}

func TestMVPLiveIdempotency(t *testing.T) {
	ctx := context.Background()
	store := newLiveIdempotencyCanaryStore()
	request := asyncIdempotencyRequest{
		scopeKey:   "caller-a\x00tenant-a",
		target:     "agent-1.echo",
		requestKey: "canary-request",
		bodyDigest: "digest-a",
	}
	canonical := &types.Execution{
		ExecutionID: "canary-execution-1",
		RunID:       "canary-run-1",
		Status:      types.ExecutionStatusQueued,
		CreatedAt:   time.Date(2026, time.October, 14, 10, 0, 0, 0, time.UTC),
	}

	t.Run("positive replay is durable and single-write", func(t *testing.T) {
		adapter := &asyncIdempotentExecutionStore{store: store, request: request}
		if err := adapter.CreateExecutionRecord(ctx, canonical); err != nil {
			t.Fatalf("create canonical execution: %v", err)
		}
		var replay *asyncIdempotencyReplayError
		if err := adapter.CreateExecutionRecord(ctx, &types.Execution{ExecutionID: "duplicate", RunID: "duplicate-run"}); !errors.As(err, &replay) {
			t.Fatalf("expected replay error, got %v", err)
		}
		if replay.execution == nil || replay.execution.ExecutionID != canonical.ExecutionID || replay.execution.RunID != canonical.RunID {
			t.Fatalf("replay returned wrong canonical execution: %+v", replay.execution)
		}
		store.mu.Lock()
		defer store.mu.Unlock()
		if store.createCount != 1 {
			t.Fatalf("expected one durable write, got %d", store.createCount)
		}
	})

	t.Run("wrong identity or payload fails closed", func(t *testing.T) {
		wrongTarget := request
		wrongTarget.target = "agent-2.echo"
		adapter := &asyncIdempotentExecutionStore{store: store, request: wrongTarget}
		var conflict *asyncIdempotencyConflictError
		if err := adapter.CreateExecutionRecord(ctx, &types.Execution{ExecutionID: "foreign-target"}); !errors.As(err, &conflict) {
			t.Fatalf("expected target conflict, got %v", err)
		}

		wrongDigest := request
		wrongDigest.bodyDigest = "digest-foreign"
		adapter = &asyncIdempotentExecutionStore{store: store, request: wrongDigest}
		if err := adapter.CreateExecutionRecord(ctx, &types.Execution{ExecutionID: "foreign-payload"}); !errors.As(err, &conflict) {
			t.Fatalf("expected payload conflict, got %v", err)
		}
	})

	t.Run("cancelled request performs no durable write", func(t *testing.T) {
		cancelled, cancel := context.WithCancel(ctx)
		cancel()
		adapter := &asyncIdempotentExecutionStore{store: store, request: asyncIdempotencyRequest{
			scopeKey: "caller-b\x00tenant-b", target: request.target, requestKey: "cancelled", bodyDigest: request.bodyDigest,
		}}
		before := store.createCount
		err := adapter.CreateExecutionRecord(cancelled, &types.Execution{ExecutionID: "cancelled"})
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context cancellation, got %v", err)
		}
		store.mu.Lock()
		defer store.mu.Unlock()
		if store.createCount != before {
			t.Fatalf("cancelled request created a durable record: before=%d after=%d", before, store.createCount)
		}
	})

	t.Run("replay survives controller restart", func(t *testing.T) {
		restartedAdapter := &asyncIdempotentExecutionStore{store: store, request: request}
		var replay *asyncIdempotencyReplayError
		if err := restartedAdapter.CreateExecutionRecord(ctx, &types.Execution{ExecutionID: "after-restart"}); !errors.As(err, &replay) {
			t.Fatalf("expected restart replay, got %v", err)
		}
		if replay.execution == nil || replay.execution.ExecutionID != canonical.ExecutionID {
			t.Fatalf("restart returned wrong execution: %+v", replay.execution)
		}
	})
}
