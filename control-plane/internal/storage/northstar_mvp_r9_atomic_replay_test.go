package storage

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Agent-Field/agentfield/control-plane/pkg/types"
)

type mvpR9StoragePaths struct {
	databasePath string
	kvStorePath  string
}

func newMVPR9Storage(t *testing.T) (*LocalStorage, mvpR9StoragePaths) {
	t.Helper()
	root := t.TempDir()
	paths := mvpR9StoragePaths{
		databasePath: filepath.Join(root, "agentfield.db"),
		kvStorePath:  filepath.Join(root, "agentfield.bolt"),
	}
	storage := NewLocalStorage(LocalStorageConfig{})
	config := StorageConfig{Mode: "local", Local: LocalStorageConfig{
		DatabasePath: paths.databasePath,
		KVStorePath:  paths.kvStorePath,
	}}
	if err := storage.Initialize(context.Background(), config); err != nil {
		if strings.Contains(err.Error(), "no such module: fts5") {
			t.Skip("sqlite3 compiled without FTS5")
		}
		t.Fatalf("initialize isolated SQLite storage: %v", err)
	}
	applyMVPR11GCanonicalMigration(t, storage)
	t.Cleanup(func() { _ = storage.Close(context.Background()) })
	return storage, paths
}

func reopenMVPR9Storage(t *testing.T, paths mvpR9StoragePaths) *LocalStorage {
	t.Helper()
	storage := NewLocalStorage(LocalStorageConfig{})
	config := StorageConfig{Mode: "local", Local: LocalStorageConfig{
		DatabasePath: paths.databasePath,
		KVStorePath:  paths.kvStorePath,
	}}
	if err := storage.Initialize(context.Background(), config); err != nil {
		t.Fatalf("reopen isolated SQLite storage: %v", err)
	}
	t.Cleanup(func() { _ = storage.Close(context.Background()) })
	return storage
}

func mvpR9Execution(id string) *types.Execution {
	now := time.Now().UTC()
	return &types.Execution{
		ExecutionID: id,
		RunID:       "run-" + id,
		AgentNodeID: "agent-r9",
		NodeID:      "agent-r9",
		ReasonerID:  "echo",
		Status:      types.ExecutionStatusRunning,
		StartedAt:   now,
	}
}

func TestMVPR9_SQLAtomicReplayNative(t *testing.T) {
	storage, paths := newMVPR9Storage(t)
	ctx := context.Background()

	t.Run("migration 036 bootstraps durable request table", func(t *testing.T) {
		var tableName string
		err := storage.requireSQLDB().QueryRowContext(ctx, `
			SELECT name FROM sqlite_master
			WHERE type = 'table' AND name = 'execution_request_idempotency'`).Scan(&tableName)
		if err != nil {
			t.Fatalf("migration 036 did not bootstrap request table: %v", err)
		}
		if tableName != "execution_request_idempotency" {
			t.Fatalf("unexpected request table name %q", tableName)
		}
	})

	t.Run("positive replay and wrong digest or target fail closed", func(t *testing.T) {
		first, err := storage.CreateAsyncExecutionIdempotent(ctx, "tenant-a", "agent-r9.echo", "request-r9", "digest-a", mvpR9Execution("exec-r9-primary"))
		if err != nil {
			t.Fatalf("create native execution: %v", err)
		}
		if !first.Created || first.Conflict || first.Execution == nil {
			t.Fatalf("expected typed created result, got %#v", first)
		}

		replay, err := storage.CreateAsyncExecutionIdempotent(ctx, "tenant-a", "agent-r9.echo", "request-r9", "digest-a", mvpR9Execution("exec-r9-duplicate"))
		if err != nil {
			t.Fatalf("replay native execution: %v", err)
		}
		if replay.Created || replay.Conflict || replay.Execution == nil || replay.Execution.ExecutionID != "exec-r9-primary" {
			t.Fatalf("expected canonical persisted replay, got %#v", replay)
		}

		for name, pair := range map[string][2]string{
			"wrong digest": {"agent-r9.echo", "digest-b"},
			"wrong target": {"other-agent.echo", "digest-a"},
		} {
			target, digest := pair[0], pair[1]
			result, err := storage.CreateAsyncExecutionIdempotent(ctx, "tenant-a", target, "request-r9", digest, mvpR9Execution("exec-r9-"+strings.ReplaceAll(name, " ", "-")))
			if err != nil {
				t.Fatalf("%s lookup: %v", name, err)
			}
			if !result.Conflict || result.Execution != nil || result.Created {
				t.Fatalf("%s should fail closed, got %#v", name, result)
			}
		}
	})

	t.Run("canceled request creates no durable effect", func(t *testing.T) {
		canceled, cancel := context.WithCancel(ctx)
		cancel()
		_, err := storage.CreateAsyncExecutionIdempotent(canceled, "tenant-a", "agent-r9.echo", "request-canceled", "digest-c", mvpR9Execution("exec-r9-canceled"))
		if err == nil {
			t.Fatal("expected canceled request to fail")
		}

		var count int
		if err := storage.requireSQLDB().QueryRowContext(ctx, `SELECT COUNT(*) FROM execution_request_idempotency WHERE request_key = ?`, "request-canceled").Scan(&count); err != nil {
			t.Fatalf("check canceled request binding: %v", err)
		}
		if count != 0 {
			t.Fatalf("canceled request left %d durable bindings", count)
		}
	})

	t.Run("concurrent exact key has one winner", func(t *testing.T) {
		const contenders = 6
		var wait sync.WaitGroup
		wait.Add(contenders)
		results := make(chan *AsyncExecutionIdempotencyResult, contenders)
		errors := make(chan error, contenders)
		for index := 0; index < contenders; index++ {
			index := index
			go func() {
				defer wait.Done()
				result, err := storage.CreateAsyncExecutionIdempotent(ctx, "tenant-race", "agent-r9.echo", "request-race", "digest-race", mvpR9Execution("exec-race-"+strconv.Itoa(index)))
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
			}
			if result.Execution != nil {
				if canonical == "" {
					canonical = result.Execution.ExecutionID
				} else if canonical != result.Execution.ExecutionID {
					t.Fatalf("concurrent replay returned different executions: %q and %q", canonical, result.Execution.ExecutionID)
				}
			}
		}
		for err := range errors {
			t.Fatalf("concurrent exact-key request failed: %v", err)
		}
		if created != 1 || canonical == "" {
			t.Fatalf("expected one winner and one canonical execution, created=%d canonical=%q", created, canonical)
		}
	})

	t.Run("close and reopen replays without duplicate execution", func(t *testing.T) {
		if err := storage.Close(ctx); err != nil {
			t.Fatalf("close storage before restart: %v", err)
		}
		storage = reopenMVPR9Storage(t, paths)

		replayed, err := storage.CreateAsyncExecutionIdempotent(ctx, "tenant-a", "agent-r9.echo", "request-r9", "digest-a", mvpR9Execution("exec-r9-after-restart"))
		if err != nil {
			t.Fatalf("replay after restart: %v", err)
		}
		if replayed.Created || replayed.Conflict || replayed.Execution == nil || replayed.Execution.ExecutionID != "exec-r9-primary" {
			t.Fatalf("expected durable canonical replay after restart, got %#v", replayed)
		}

		var count int
		if err := storage.requireSQLDB().QueryRowContext(ctx, `SELECT COUNT(*) FROM executions WHERE execution_id = ?`, "exec-r9-primary").Scan(&count); err != nil {
			t.Fatalf("count canonical execution: %v", err)
		}
		if count != 1 {
			t.Fatalf("expected one canonical execution after restart, got %d", count)
		}
	})
}

func applyMVPR11GCanonicalMigration(t *testing.T, storage *LocalStorage) {
	t.Helper()
	migration, err := os.ReadFile(filepath.Join("..", "..", "migrations", "036_execution_request_idempotency.sql"))
	if err != nil {
		t.Fatalf("read canonical idempotency migration: %v", err)
	}
	upSection, _, foundDown := strings.Cut(string(migration), "-- +goose Down")
	if !foundDown {
		t.Fatal("canonical idempotency migration is missing its Down marker")
	}
	upSection = strings.Replace(upSection, "-- +goose Up", "", 1)
	if _, err := storage.requireSQLDB().Exec(upSection); err != nil {
		t.Fatalf("apply canonical idempotency migration: %v", err)
	}
}
