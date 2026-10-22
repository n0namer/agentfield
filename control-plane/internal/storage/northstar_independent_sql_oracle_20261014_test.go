package storage

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestNorthstarIndependentSQLOracleConcurrentDuplicate(t *testing.T) {
	ctx := context.Background()
	databasePath := filepath.Join(t.TempDir(), "independent-concurrent.db")
	const contenders = 2

	stores := make([]*LocalStorage, contenders)
	for index := 0; index < contenders; index++ {
		stores[index] = initializeNorthstarStorage(t, databasePath, filepath.Join(t.TempDir(), fmt.Sprintf("independent-concurrent-%d.bolt", index)))
	}

	start := make(chan struct{})
	results := make(chan *AsyncExecutionIdempotencyResult, contenders)
	errors := make(chan error, contenders)
	var wait sync.WaitGroup
	for index := 0; index < contenders; index++ {
		index := index
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			result, err := stores[index].CreateAsyncExecutionIdempotent(
				ctx,
				"independent-owner",
				"agent.reasoner",
				"request-concurrent",
				"digest-same",
				runtimeExecution(fmt.Sprintf("independent-exec-%d", index), fmt.Sprintf("independent-run-%d", index)),
			)
			if err != nil {
				errors <- err
				return
			}
			results <- result
		}()
	}
	close(start)
	wait.Wait()
	close(results)
	close(errors)

	for err := range errors {
		t.Fatalf("concurrent duplicate request failed closed with storage error: %v", err)
	}

	createdCount := 0
	canonicalID := ""
	resultCount := 0
	for result := range results {
		resultCount++
		if result == nil || result.Execution == nil || result.Conflict {
			t.Fatalf("unexpected concurrent result: %+v", result)
		}
		if result.Created {
			createdCount++
		}
		if canonicalID == "" {
			canonicalID = result.Execution.ExecutionID
		} else if result.Execution.ExecutionID != canonicalID {
			t.Fatalf("duplicate request returned different executions: %q and %q", canonicalID, result.Execution.ExecutionID)
		}
	}
	if resultCount != contenders || createdCount != 1 {
		t.Fatalf("expected two results and exactly one creator, got results=%d creators=%d", resultCount, createdCount)
	}

	db := stores[0].requireSQLDB()
	var bindings, executions int
	if err := db.QueryRow(`SELECT COUNT(*) FROM execution_request_idempotency`).Scan(&bindings); err != nil {
		t.Fatalf("count idempotency bindings: %v", err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM executions`).Scan(&executions); err != nil {
		t.Fatalf("count execution records: %v", err)
	}
	if bindings != 1 || executions != 1 {
		t.Fatalf("database fence invariant failed: bindings=%d executions=%d", bindings, executions)
	}
}

func TestNorthstarIndependentSQLOracleRejectsConflictingDigestAndTarget(t *testing.T) {
	ctx := context.Background()
	store := initializeNorthstarStorage(t, filepath.Join(t.TempDir(), "independent-conflict.db"), filepath.Join(t.TempDir(), "independent-conflict.bolt"))

	created, err := store.CreateAsyncExecutionIdempotent(
		ctx, "independent-owner", "agent.reasoner", "request-conflict", "digest-original", runtimeExecution("independent-conflict-exec", "independent-conflict-run"),
	)
	if err != nil || !created.Created {
		t.Fatalf("create conflict fixture: result=%+v error=%v", created, err)
	}

	for name, target := range map[string]string{
		"different digest": "agent.reasoner",
		"different target": "other.reasoner",
	} {
		digestValue := "digest-original"
		if name == "different digest" {
			digestValue = "digest-other"
		}
		conflict, lookupErr := store.LookupAsyncExecutionIdempotency(ctx, "independent-owner", target, "request-conflict", digestValue)
		if lookupErr != nil {
			t.Fatalf("%s lookup: %v", name, lookupErr)
		}
		if conflict == nil || !conflict.Conflict || conflict.Execution != nil {
			t.Fatalf("%s was not rejected fail-closed: %+v", name, conflict)
		}
	}
}

func TestNorthstarIndependentSQLOraclePersistsAcrossRestart(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	config := StorageConfig{Mode: "local", Local: LocalStorageConfig{
		DatabasePath: filepath.Join(tempDir, "independent-restart.db"),
		KVStorePath:  filepath.Join(tempDir, "independent-restart.bolt"),
	}}

	store := NewLocalStorage(LocalStorageConfig{})
	if err := store.Initialize(ctx, config); err != nil {
		if strings.Contains(err.Error(), "no such module: fts5") {
			t.Skip("sqlite3 compiled without FTS5")
		}
		t.Fatalf("initialize pre-restart storage: %v", err)
	}
	applyNorthstarIdempotencyMigration(t, store)
	canonicalID := "independent-restart-exec"
	created, err := store.CreateAsyncExecutionIdempotent(
		ctx, "independent-owner", "agent.reasoner", "request-restart", "digest-restart", runtimeExecution(canonicalID, "independent-restart-run"),
	)
	if err != nil || !created.Created {
		_ = store.Close(ctx)
		t.Fatalf("create restart fixture: result=%+v error=%v", created, err)
	}
	if err := store.Close(ctx); err != nil {
		t.Fatalf("close pre-restart storage: %v", err)
	}

	restarted := NewLocalStorage(LocalStorageConfig{})
	if err := restarted.Initialize(ctx, config); err != nil {
		t.Fatalf("reinitialize storage after restart: %v", err)
	}
	applyNorthstarIdempotencyMigration(t, restarted)
	defer restarted.Close(ctx)

	replayed, err := restarted.CreateAsyncExecutionIdempotent(
		ctx, "independent-owner", "agent.reasoner", "request-restart", "digest-restart", runtimeExecution("must-not-win", "must-not-win"),
	)
	if err != nil {
		t.Fatalf("replay after restart: %v", err)
	}
	if replayed == nil || replayed.Created || replayed.Conflict || replayed.Execution == nil || replayed.Execution.ExecutionID != canonicalID {
		t.Fatalf("restart replay did not return canonical execution: %+v", replayed)
	}
}

func initializeNorthstarStorage(t *testing.T, databasePath, kvStorePath string) *LocalStorage {
	t.Helper()
	ctx := context.Background()
	config := StorageConfig{Mode: "local", Local: LocalStorageConfig{
		DatabasePath: databasePath,
		KVStorePath:  kvStorePath,
	}}
	store := NewLocalStorage(LocalStorageConfig{})
	if err := store.Initialize(ctx, config); err != nil {
		if strings.Contains(err.Error(), "no such module: fts5") {
			t.Skip("sqlite3 compiled without FTS5")
		}
		t.Fatalf("initialize native SQL storage: %v", err)
	}
	applyNorthstarIdempotencyMigration(t, store)
	t.Cleanup(func() {
		_ = store.Close(ctx)
	})
	return store
}

func applyNorthstarIdempotencyMigration(t *testing.T, store *LocalStorage) {
	t.Helper()
	migration := readQAArtifact(t, "../../migrations/036_execution_request_idempotency.sql")
	upSection, _, foundDown := strings.Cut(migration, "-- +goose Down")
	if !foundDown {
		t.Fatal("canonical idempotency migration is missing its Down marker")
	}
	upSection = strings.Replace(upSection, "-- +goose Up", "", 1)
	if _, err := store.requireSQLDB().Exec(upSection); err != nil {
		t.Fatalf("apply canonical idempotency migration: %v", err)
	}
}
