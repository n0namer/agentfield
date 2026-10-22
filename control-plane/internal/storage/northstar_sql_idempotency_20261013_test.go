package storage

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"sync"
	"testing"
)

func TestNorthstarSQLIdempotencyRaceRestartAndConflicts(t *testing.T) {
	ctx := context.Background()
	databasePath := filepath.Join(t.TempDir(), "northstar-idempotency.db")
	driverName := runtimeSQLiteDriver(t)

	setupDB := openRuntimeSQLite(t, driverName, databasePath)
	createRuntimeIdempotencySchema(t, setupDB)
	if _, err := setupDB.Exec(`PRAGMA busy_timeout = 5000`); err != nil {
		t.Fatalf("set setup busy timeout: %v", err)
	}
	if err := setupDB.Close(); err != nil {
		t.Fatalf("close setup database: %v", err)
	}

	const contenders = 2
	contenderDBs := make([]interface{ Close() error }, contenders)
	contenderStores := make([]*LocalStorage, contenders)
	for index := 0; index < contenders; index++ {
		db := openRuntimeSQLite(t, driverName, databasePath)
		if _, err := db.Exec(`PRAGMA busy_timeout = 5000`); err != nil {
			db.Close()
			t.Fatalf("set contender %d busy timeout: %v", index, err)
		}
		contenderDBs[index] = db
		contenderStores[index] = runtimeLocalStorage(t, db)
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
			result, err := contenderStores[index].CreateAsyncExecutionIdempotent(
				ctx,
				"caller-a\x00tenant-a",
				"agent-a.reasoner-a",
				"request-race",
				"digest-a",
				runtimeExecution(fmt.Sprintf("exec-race-%d", index), fmt.Sprintf("run-race-%d", index)),
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

	var raceErrors []string
	for err := range errors {
		raceErrors = append(raceErrors, err.Error())
	}
	if len(raceErrors) > 0 {
		sort.Strings(raceErrors)
		t.Fatalf("concurrent same-payload requests must replay the SQL winner, got errors: %v", raceErrors)
	}

	createdCount := 0
	canonicalExecutionID := ""
	resultCount := 0
	for result := range results {
		resultCount++
		if result == nil || result.Execution == nil || result.Conflict {
			t.Fatalf("unexpected race result: %+v", result)
		}
		if result.Created {
			createdCount++
		}
		if canonicalExecutionID == "" {
			canonicalExecutionID = result.Execution.ExecutionID
		} else if result.Execution.ExecutionID != canonicalExecutionID {
			t.Fatalf("same request returned different executions: %q and %q", canonicalExecutionID, result.Execution.ExecutionID)
		}
	}
	if resultCount != contenders || createdCount != 1 {
		t.Fatalf("expected %d results and one SQL winner, got results=%d created=%d", contenders, resultCount, createdCount)
	}

	for index, db := range contenderDBs {
		if err := db.Close(); err != nil {
			t.Fatalf("close contender %d database: %v", index, err)
		}
	}

	restartedDB := openRuntimeSQLite(t, driverName, databasePath)
	defer restartedDB.Close()
	restartedStore := runtimeLocalStorage(t, restartedDB)

	replayed, err := restartedStore.CreateAsyncExecutionIdempotent(
		ctx,
		"caller-a\x00tenant-a",
		"agent-a.reasoner-a",
		"request-race",
		"digest-a",
		runtimeExecution("exec-after-restart", "run-after-restart"),
	)
	if err != nil {
		t.Fatalf("replay after storage restart: %v", err)
	}
	if replayed.Created || replayed.Conflict || replayed.Execution == nil || replayed.Execution.ExecutionID != canonicalExecutionID {
		t.Fatalf("restart replay did not return the canonical execution: %+v", replayed)
	}

	digestConflict, err := restartedStore.LookupAsyncExecutionIdempotency(ctx, "caller-a\x00tenant-a", "agent-a.reasoner-a", "request-race", "digest-b")
	if err != nil {
		t.Fatalf("lookup conflicting digest: %v", err)
	}
	if !digestConflict.Conflict || digestConflict.Execution != nil {
		t.Fatalf("different payload digest must conflict: %+v", digestConflict)
	}

	foreignScope, err := restartedStore.LookupAsyncExecutionIdempotency(ctx, "caller-b\x00tenant-b", "agent-a.reasoner-a", "request-race", "digest-a")
	if err != nil {
		t.Fatalf("lookup foreign scope: %v", err)
	}
	if foreignScope.Conflict || foreignScope.Execution != nil || foreignScope.Created {
		t.Fatalf("request key leaked across scope boundary: %+v", foreignScope)
	}

	var bindingCount, executionCount int
	if err := restartedDB.QueryRow(`SELECT COUNT(*) FROM execution_request_idempotency WHERE scope_key = ? AND request_key = ?`, "caller-a\x00tenant-a", "request-race").Scan(&bindingCount); err != nil {
		t.Fatalf("count canonical binding: %v", err)
	}
	if err := restartedDB.QueryRow(`SELECT COUNT(*) FROM executions`).Scan(&executionCount); err != nil {
		t.Fatalf("count execution records: %v", err)
	}
	if bindingCount != 1 || executionCount != 1 {
		t.Fatalf("idempotency fence invariant failed: bindings=%d executions=%d", bindingCount, executionCount)
	}
}
