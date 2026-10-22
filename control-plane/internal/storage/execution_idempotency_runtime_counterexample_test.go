package storage

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
	"unsafe"

	"github.com/Agent-Field/agentfield/control-plane/pkg/types"
)

func TestIdempotencyRuntimeCounterexample(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "idempotency-runtime.db")
	driverName := runtimeSQLiteDriver(t)
	db := openRuntimeSQLite(t, driverName, databasePath)
	createRuntimeIdempotencySchema(t, db)
	store := runtimeLocalStorage(t, db)

	ctx := context.Background()
	base := runtimeExecution("exec-one", "run-one")
	created, err := store.CreateAsyncExecutionIdempotent(ctx, "caller-a\x00target-did", "agent.reasoner", "request-1", "digest-a", base)
	if err != nil {
		t.Fatalf("create initial idempotent execution: %v", err)
	}
	if !created.Created || created.Conflict || created.Execution == nil || created.Execution.ExecutionID != base.ExecutionID {
		t.Fatalf("unexpected create result: %+v", created)
	}

	t.Run("same scope key and digest replays after database reopen", func(t *testing.T) {
		if err := db.Close(); err != nil {
			t.Fatalf("close initial database: %v", err)
		}
		reopened := openRuntimeSQLite(t, driverName, databasePath)
		t.Cleanup(func() { _ = reopened.Close() })
		restartedStore := runtimeLocalStorage(t, reopened)

		replay, err := restartedStore.LookupAsyncExecutionIdempotency(ctx, "caller-a\x00target-did", "agent.reasoner", "request-1", "digest-a")
		if err != nil {
			t.Fatalf("lookup after reopen: %v", err)
		}
		if replay.Conflict || replay.Execution == nil || replay.Execution.ExecutionID != base.ExecutionID {
			t.Fatalf("restart lookup did not return canonical execution: %+v", replay)
		}

		mismatch, err := restartedStore.LookupAsyncExecutionIdempotency(ctx, "caller-a\x00target-did", "agent.reasoner", "request-1", "digest-b")
		if err != nil {
			t.Fatalf("lookup payload mismatch: %v", err)
		}
		if !mismatch.Conflict || mismatch.Execution != nil {
			t.Fatalf("payload mismatch was not rejected: %+v", mismatch)
		}

		targetMismatch, err := restartedStore.LookupAsyncExecutionIdempotency(ctx, "caller-a\x00target-did", "other.reasoner", "request-1", "digest-a")
		if err != nil {
			t.Fatalf("lookup target mismatch: %v", err)
		}
		if !targetMismatch.Conflict {
			t.Fatalf("target mismatch was not rejected: %+v", targetMismatch)
		}

		otherScope, err := restartedStore.LookupAsyncExecutionIdempotency(ctx, "caller-b\x00target-did", "agent.reasoner", "request-1", "digest-a")
		if err != nil {
			t.Fatalf("lookup other owner scope: %v", err)
		}
		if otherScope.Conflict || otherScope.Execution != nil {
			t.Fatalf("request key leaked across owner scope: %+v", otherScope)
		}
	})
}

func TestIdempotencyConcurrentSameKeyCreatesOneCanonicalExecution(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "idempotency-race.db")
	driverName := runtimeSQLiteDriver(t)
	setupDB := openRuntimeSQLite(t, driverName, databasePath)
	createRuntimeIdempotencySchema(t, setupDB)
	if _, err := setupDB.Exec(`PRAGMA busy_timeout = 5000`); err != nil {
		t.Fatalf("set SQLite busy timeout: %v", err)
	}
	if err := setupDB.Close(); err != nil {
		t.Fatalf("close setup database: %v", err)
	}

	const contenders = 2
	contenderDBs := make([]*sql.DB, contenders)
	contenderStores := make([]*LocalStorage, contenders)
	for index := 0; index < contenders; index++ {
		contenderDBs[index] = openRuntimeSQLite(t, driverName, databasePath)
		t.Cleanup(func() { _ = contenderDBs[index].Close() })
		if _, err := contenderDBs[index].Exec(`PRAGMA busy_timeout = 5000`); err != nil {
			t.Fatalf("set contender SQLite busy timeout: %v", err)
		}
		contenderStores[index] = runtimeLocalStorage(t, contenderDBs[index])
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
				context.Background(),
				"caller-a\x00target-did",
				"agent.reasoner",
				"request-race",
				"digest-race",
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
		t.Fatalf("concurrent same-key request returned storage errors instead of replaying the canonical execution: %v", raceErrors)
	}

	var canonicalID string
	createdCount := 0
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
			t.Fatalf("same key returned different executions: %q and %q", canonicalID, result.Execution.ExecutionID)
		}
	}
	if resultCount != contenders || createdCount != 1 {
		t.Fatalf("expected %d successful results with exactly one creator; got results=%d creators=%d", contenders, resultCount, createdCount)
	}

	verifyDB := openRuntimeSQLite(t, driverName, databasePath)
	defer verifyDB.Close()
	var bindings, executions int
	if err := verifyDB.QueryRow(`SELECT COUNT(*) FROM execution_request_idempotency`).Scan(&bindings); err != nil {
		t.Fatalf("count idempotency bindings: %v", err)
	}
	if err := verifyDB.QueryRow(`SELECT COUNT(*) FROM executions`).Scan(&executions); err != nil {
		t.Fatalf("count executions: %v", err)
	}
	if bindings != 1 || executions != 1 {
		t.Fatalf("concurrent request escaped database fence: bindings=%d executions=%d", bindings, executions)
	}
}

func TestIdempotencySQLiteStaleCompareAndSwapRowsAffected(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "idempotency-cas.db")
	driverName := runtimeSQLiteDriver(t)
	db := openRuntimeSQLite(t, driverName, databasePath)
	defer db.Close()
	createRuntimeIdempotencySchema(t, db)
	store := runtimeLocalStorage(t, db)

	created, err := store.CreateAsyncExecutionIdempotent(
		context.Background(),
		"caller-a\x00target-did",
		"agent.reasoner",
		"request-cas",
		"digest-cas",
		runtimeExecution("exec-cas", "run-cas"),
	)
	if err != nil || !created.Created {
		t.Fatalf("create CAS fixture: result=%+v error=%v", created, err)
	}

	var originalUpdatedAt any
	if err := db.QueryRow(`SELECT updated_at FROM executions WHERE execution_id = ?`, "exec-cas").Scan(&originalUpdatedAt); err != nil {
		t.Fatalf("read original update token: %v", err)
	}
	firstWrite, err := db.Exec(
		`UPDATE executions SET status = ?, updated_at = ? WHERE execution_id = ? AND updated_at = ?`,
		types.ExecutionStatusSucceeded,
		time.Now().UTC().Add(time.Second),
		"exec-cas",
		originalUpdatedAt,
	)
	if err != nil {
		t.Fatalf("apply first compare-and-swap write: %v", err)
	}
	firstRows, err := firstWrite.RowsAffected()
	if err != nil {
		t.Fatalf("read first compare-and-swap row count: %v", err)
	}
	if firstRows != 1 {
		t.Fatalf("first compare-and-swap write affected %d rows, want 1", firstRows)
	}

	staleWrite, err := db.Exec(
		`UPDATE executions SET status = ? WHERE execution_id = ? AND updated_at = ?`,
		types.ExecutionStatusFailed,
		"exec-cas",
		originalUpdatedAt,
	)
	if err != nil {
		t.Fatalf("apply stale compare-and-swap write: %v", err)
	}
	staleRows, err := staleWrite.RowsAffected()
	if err != nil {
		t.Fatalf("read stale compare-and-swap row count: %v", err)
	}
	if staleRows != 0 {
		t.Fatalf("stale compare-and-swap write affected %d rows, want 0", staleRows)
	}

	var status string
	if err := db.QueryRow(`SELECT status FROM executions WHERE execution_id = ?`, "exec-cas").Scan(&status); err != nil {
		t.Fatalf("read execution after stale write: %v", err)
	}
	if status != string(types.ExecutionStatusSucceeded) {
		t.Fatalf("stale write changed terminal status to %q", status)
	}
}

func runtimeExecution(executionID, runID string) *types.Execution {
	return &types.Execution{
		ExecutionID: executionID,
		RunID:       runID,
		Status:      types.ExecutionStatusQueued,
		StartedAt:   time.Now().UTC(),
	}
}

func runtimeSQLiteDriver(t *testing.T) string {
	t.Helper()
	for _, driverName := range sql.Drivers() {
		if strings.Contains(strings.ToLower(driverName), "sqlite") {
			return driverName
		}
	}
	t.Fatalf("storage package registered no SQLite driver; available drivers: %v", sql.Drivers())
	return ""
}

func openRuntimeSQLite(t *testing.T, driverName, databasePath string) *sql.DB {
	t.Helper()
	db, err := sql.Open(driverName, databasePath)
	if err != nil {
		t.Fatalf("open disposable SQLite database with %s: %v", driverName, err)
	}
	db.SetMaxOpenConns(4)
	if err := db.Ping(); err != nil {
		db.Close()
		t.Fatalf("ping disposable SQLite database with %s: %v", driverName, err)
	}
	return db
}

func runtimeLocalStorage(t *testing.T, db *sql.DB) *LocalStorage {
	t.Helper()
	store := &LocalStorage{}
	storeValue := reflect.ValueOf(store).Elem()
	databaseType := reflect.TypeOf(db)
	for index := 0; index < storeValue.NumField(); index++ {
		field := storeValue.Field(index)
		if field.Type() != databaseType || !field.CanAddr() {
			continue
		}
		writable := reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem()
		writable.Set(reflect.ValueOf(db))
		return store
	}
	t.Fatalf("LocalStorage has no *sql.DB field for isolated runtime injection")
	return nil
}

func createRuntimeIdempotencySchema(t *testing.T, db *sql.DB) {
	t.Helper()
	statements := []string{
		`CREATE TABLE executions (
			execution_id TEXT PRIMARY KEY,
			run_id TEXT NOT NULL,
			parent_execution_id TEXT,
			agent_node_id TEXT,
			instance_id TEXT,
			reasoner_id TEXT,
			node_id TEXT,
			status TEXT,
			status_reason TEXT,
			input_payload BLOB,
			result_payload BLOB,
			error_message TEXT,
			input_uri TEXT,
			result_uri TEXT,
			session_id TEXT,
			actor_id TEXT,
			started_at TIMESTAMP,
			completed_at TIMESTAMP,
			duration_ms INTEGER,
			notes BLOB,
			created_at TIMESTAMP,
			updated_at TIMESTAMP
		)`,
		`CREATE TABLE execution_request_idempotency (
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
		)`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("create disposable runtime schema: %v", err)
		}
	}
}
