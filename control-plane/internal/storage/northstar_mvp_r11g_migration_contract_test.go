package storage

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestMVPR11GMigrationContract(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	storage := NewLocalStorage(LocalStorageConfig{})
	config := StorageConfig{Mode: "local", Local: LocalStorageConfig{
		DatabasePath: filepath.Join(root, "migration-contract.db"),
		KVStorePath:  filepath.Join(root, "migration-contract.bolt"),
	}}
	if err := storage.Initialize(ctx, config); err != nil {
		if strings.Contains(err.Error(), "no such module: fts5") {
			t.Skip("sqlite3 compiled without FTS5")
		}
		t.Fatalf("initialize disposable SQLite storage: %v", err)
	}
	t.Cleanup(func() { _ = storage.Close(ctx) })

	t.Run("LocalStorage initialization does not substitute for goose migration lifecycle", func(t *testing.T) {
		var count int
		if err := storage.requireSQLDB().QueryRowContext(ctx, `
			SELECT COUNT(*) FROM sqlite_master
			WHERE type = 'table' AND name = 'execution_request_idempotency'`).Scan(&count); err != nil {
			t.Fatalf("inspect pre-migration schema: %v", err)
		}
		if count != 0 {
			t.Fatalf("expected disposable storage to require canonical migration 036, found %d request tables", count)
		}
	})

	applyMVPR11GCanonicalMigration(t, storage)

	t.Run("canonical migration creates durable table and index", func(t *testing.T) {
		var tableSQL string
		if err := storage.requireSQLDB().QueryRowContext(ctx, `
			SELECT sql FROM sqlite_master
			WHERE type = 'table' AND name = 'execution_request_idempotency'`).Scan(&tableSQL); err != nil {
			t.Fatalf("canonical migration did not create durable request table: %v", err)
		}
		for _, fragment := range []string{
			"PRIMARY KEY (scope_key, request_key)",
			"UNIQUE (execution_id)",
		} {
			if !strings.Contains(tableSQL, fragment) {
				t.Fatalf("durable request table is missing contract %q: %s", fragment, tableSQL)
			}
		}

		var indexName string
		if err := storage.requireSQLDB().QueryRowContext(ctx, `
			SELECT name FROM sqlite_master
			WHERE type = 'index' AND name = 'idx_execution_request_idempotency_execution'`).Scan(&indexName); err != nil {
			t.Fatalf("canonical migration did not create execution lookup index: %v", err)
		}
		if indexName != "idx_execution_request_idempotency_execution" {
			t.Fatalf("unexpected execution lookup index %q", indexName)
		}
	})
}
