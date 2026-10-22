package storage

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/Agent-Field/agentfield/control-plane/pkg/types"
)

func TestR24DurableFailure(t *testing.T) {
	ctx := context.Background()
	for _, scenario := range []string{"canonical result", "wrong digest", "wrong target", "invalid source run"} {
		t.Run(scenario, func(t *testing.T) {
			database := openRuntimeSQLite(t, runtimeSQLiteDriver(t), filepath.Join(t.TempDir(), "r24.db"))
			t.Cleanup(func() { _ = database.Close() })
			createRuntimeIdempotencySchema(t, database)
			store := NewLocalStorage(LocalStorageConfig{})
			store.db = newSQLDatabase(database, "local")
			execution := runtimeExecution("exec-r24-canonical", "run-r24-canonical")
			execution.Status = types.ExecutionStatusSucceeded
			execution.ResultPayload = []byte(`{"result":"durable-r24"}`)
			created, err := store.CreateAsyncExecutionIdempotent(ctx, "scope-r24", "agent.echo", "request-r24", "digest-r24", execution)
			if err != nil || created == nil || !created.Created || created.Execution == nil {
				t.Fatalf("create canonical result: result=%+v error=%v", created, err)
			}

			target, digest := "agent.echo", "digest-r24"
			switch scenario {
			case "wrong digest":
				digest = "changed-digest"
			case "wrong target":
				target = "other.echo"
			case "invalid source run":
				if _, err := database.ExecContext(ctx, `UPDATE execution_request_idempotency SET run_id = ? WHERE scope_key = ? AND request_key = ?`, "foreign-run", "scope-r24", "request-r24"); err != nil {
					t.Fatalf("invalidate source run identity: %v", err)
				}
			}

			replay, err := store.CreateAsyncExecutionIdempotent(ctx, "scope-r24", target, "request-r24", digest, runtimeExecution("exec-r24-rejected", "run-r24-rejected"))
			switch scenario {
			case "invalid source run":
				if err == nil || replay != nil {
					t.Fatalf("invalid source run must fail closed: result=%+v error=%v", replay, err)
				}
			case "wrong digest", "wrong target":
				if err != nil || replay == nil || !replay.Conflict || replay.Created || replay.Execution != nil {
					t.Fatalf("conflicting request must fail closed: result=%+v error=%v", replay, err)
				}
			default:
				if err != nil || replay == nil || replay.Created || replay.Conflict || replay.Execution == nil {
					t.Fatalf("canonical result replay: result=%+v error=%v", replay, err)
				}
				if replay.Execution.ExecutionID != execution.ExecutionID || replay.Execution.RunID != execution.RunID || replay.Execution.Status != types.ExecutionStatusSucceeded || string(replay.Execution.ResultPayload) != string(execution.ResultPayload) {
					t.Fatalf("replay changed canonical durable outcome: %+v", replay.Execution)
				}
			}

			var executions, bindings int
			if err := database.QueryRowContext(ctx, `SELECT COUNT(*) FROM executions`).Scan(&executions); err != nil {
				t.Fatalf("count execution rows: %v", err)
			}
			if err := database.QueryRowContext(ctx, `SELECT COUNT(*) FROM execution_request_idempotency`).Scan(&bindings); err != nil {
				t.Fatalf("count binding rows: %v", err)
			}
			if executions != 1 || bindings != 1 {
				t.Fatalf("replay created extra SQL rows: executions=%d bindings=%d", executions, bindings)
			}
		})
	}
}
