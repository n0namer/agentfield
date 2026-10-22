package storage

import (
	"context"
	"strconv"
	"sync"
	"testing"
)

func TestMVPR12_SQLReplay(t *testing.T) {
	storage, paths := newMVPR9Storage(t)
	ctx := context.Background()

	t.Run("migration 036 creates the durable request binding table", func(t *testing.T) {
		var tableName string
		if err := storage.requireSQLDB().QueryRowContext(ctx, `
			SELECT name FROM sqlite_master
			WHERE type = 'table' AND name = 'execution_request_idempotency'`).Scan(&tableName); err != nil {
			t.Fatalf("migration 036 request table: %v", err)
		}
		if tableName != "execution_request_idempotency" {
			t.Fatalf("unexpected migration 036 table %q", tableName)
		}
	})

	t.Run("concurrent same request has exactly one durable execution", func(t *testing.T) {
		const contenders = 8
		results := make(chan *AsyncExecutionIdempotencyResult, contenders)
		errors := make(chan error, contenders)
		var group sync.WaitGroup
		group.Add(contenders)

		for index := 0; index < contenders; index++ {
			index := index
			go func() {
				defer group.Done()
				result, err := storage.CreateAsyncExecutionIdempotent(
					ctx,
					"tenant-r12",
					"agent-r12.echo",
					"request-r12-race",
					"digest-r12-race",
					mvpR9Execution("exec-r12-race-"+strconv.Itoa(index)),
				)
				if err != nil {
					errors <- err
					return
				}
				results <- result
			}()
		}
		group.Wait()
		close(results)
		close(errors)

		created := 0
		canonicalID := ""
		for result := range results {
			if result.Created {
				created++
			}
			if result.Execution == nil {
				t.Fatalf("exact-key call returned no execution: %#v", result)
			}
			if canonicalID == "" {
				canonicalID = result.Execution.ExecutionID
			} else if canonicalID != result.Execution.ExecutionID {
				t.Fatalf("same request returned multiple executions: %q and %q", canonicalID, result.Execution.ExecutionID)
			}
		}
		for err := range errors {
			t.Fatalf("concurrent exact-key call failed: %v", err)
		}
		if created != 1 || canonicalID == "" {
			t.Fatalf("expected exactly one creator and canonical execution, created=%d canonical=%q", created, canonicalID)
		}

		var executionCount int
		if err := storage.requireSQLDB().QueryRowContext(ctx, `SELECT COUNT(*) FROM executions WHERE execution_id = ?`, canonicalID).Scan(&executionCount); err != nil {
			t.Fatalf("count canonical execution: %v", err)
		}
		if executionCount != 1 {
			t.Fatalf("expected one persisted external effect, got %d", executionCount)
		}
	})

	t.Run("digest conflict fails closed without creating another execution", func(t *testing.T) {
		first, err := storage.CreateAsyncExecutionIdempotent(ctx, "tenant-r12", "agent-r12.echo", "request-r12-conflict", "digest-r12-a", mvpR9Execution("exec-r12-conflict-primary"))
		if err != nil || !first.Created || first.Execution == nil {
			t.Fatalf("create canonical request: result=%#v err=%v", first, err)
		}

		conflict, err := storage.CreateAsyncExecutionIdempotent(ctx, "tenant-r12", "agent-r12.echo", "request-r12-conflict", "digest-r12-b", mvpR9Execution("exec-r12-conflict-rejected"))
		if err != nil {
			t.Fatalf("conflicting digest lookup: %v", err)
		}
		if !conflict.Conflict || conflict.Created || conflict.Execution != nil {
			t.Fatalf("digest conflict did not fail closed: %#v", conflict)
		}

		var executionCount int
		if err := storage.requireSQLDB().QueryRowContext(ctx, `SELECT COUNT(*) FROM executions WHERE execution_id IN (?, ?)`, "exec-r12-conflict-primary", "exec-r12-conflict-rejected").Scan(&executionCount); err != nil {
			t.Fatalf("count digest-conflict executions: %v", err)
		}
		if executionCount != 1 {
			t.Fatalf("digest conflict created an extra execution: %d persisted", executionCount)
		}
	})

	t.Run("canceled request persists neither binding nor execution", func(t *testing.T) {
		canceled, cancel := context.WithCancel(ctx)
		cancel()
		_, err := storage.CreateAsyncExecutionIdempotent(canceled, "tenant-r12", "agent-r12.echo", "request-r12-canceled", "digest-r12-canceled", mvpR9Execution("exec-r12-canceled"))
		if err == nil {
			t.Fatal("canceled request unexpectedly succeeded")
		}

		var bindingCount int
		if err := storage.requireSQLDB().QueryRowContext(ctx, `SELECT COUNT(*) FROM execution_request_idempotency WHERE request_key = ?`, "request-r12-canceled").Scan(&bindingCount); err != nil {
			t.Fatalf("count canceled request bindings: %v", err)
		}
		if bindingCount != 0 {
			t.Fatalf("canceled request persisted %d bindings", bindingCount)
		}
	})

	t.Run("restart retains canonical replay without duplicate execution", func(t *testing.T) {
		created, err := storage.CreateAsyncExecutionIdempotent(ctx, "tenant-r12", "agent-r12.echo", "request-r12-restart", "digest-r12-restart", mvpR9Execution("exec-r12-restart-primary"))
		if err != nil || !created.Created || created.Execution == nil {
			t.Fatalf("create restart candidate: result=%#v err=%v", created, err)
		}
		if err := storage.Close(ctx); err != nil {
			t.Fatalf("close storage before restart: %v", err)
		}
		storage = reopenMVPR9Storage(t, paths)

		replay, err := storage.CreateAsyncExecutionIdempotent(ctx, "tenant-r12", "agent-r12.echo", "request-r12-restart", "digest-r12-restart", mvpR9Execution("exec-r12-restart-replayed"))
		if err != nil {
			t.Fatalf("replay after restart: %v", err)
		}
		if replay.Created || replay.Conflict || replay.Execution == nil || replay.Execution.ExecutionID != "exec-r12-restart-primary" {
			t.Fatalf("restart replay did not return the canonical execution: %#v", replay)
		}

		var executionCount int
		if err := storage.requireSQLDB().QueryRowContext(ctx, `SELECT COUNT(*) FROM executions WHERE execution_id IN (?, ?)`, "exec-r12-restart-primary", "exec-r12-restart-replayed").Scan(&executionCount); err != nil {
			t.Fatalf("count restart executions: %v", err)
		}
		if executionCount != 1 {
			t.Fatalf("restart replay created an extra execution: %d persisted", executionCount)
		}
	})
}
