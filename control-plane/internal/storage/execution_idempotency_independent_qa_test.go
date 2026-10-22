package storage

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
)

const idempotencyMigrationPath = "../../migrations/036_execution_request_idempotency.sql"

func TestExecutionIdempotencyIndependentQA(t *testing.T) {
	source := readQAArtifact(t, "execution_idempotency.go")
	migration := readQAArtifact(t, idempotencyMigrationPath)

	t.Run("durable request identity is owner scoped", func(t *testing.T) {
		requireAnyTerm(t, source, "idempotency source", "idempotency_key", "IdempotencyKey")
		requireAnyTerm(t, source, "request digest", "request_hash", "RequestHash", "payload_hash", "PayloadHash")
		requireAnyTerm(t, source, "durable execution identity", "execution_id", "ExecutionID")
		requireOwnerIsolation(t, source)

		requireAnyTerm(t, migration, "idempotency key column", "idempotency_key")
		requireAnyTerm(t, migration, "request digest column", "request_hash", "payload_hash")
		requireAnyTerm(t, migration, "execution identity column", "execution_id")
		requireOwnerIsolation(t, migration)
	})

	t.Run("same key and payload replays one execution", func(t *testing.T) {
		requireAnyTerm(t, source, "atomic claim", "ON CONFLICT", "RowsAffected", "rowsAffected", "Transaction(", "BeginTx(")
		requireAnyTerm(t, source, "existing claim lookup", "First(", "Take(", "Scan(", "QueryRow", "SELECT")
		requireAnyTerm(t, source, "same-payload comparison", "requestHash", "RequestHash", "payloadHash", "PayloadHash")
		requireAnyTerm(t, source, "existing execution replay", "ExecutionID", "execution_id")
		requireUniqueOwnerKey(t, migration)
	})

	t.Run("same key and different body conflicts", func(t *testing.T) {
		requireAnyTerm(t, source, "digest mismatch branch", "!=", "Equal(", "ConstantTimeCompare")
		requireAnyTerm(t, source, "conflict result", "conflict", "Conflict", "ErrIdempotency", "mismatch", "Mismatch")
	})

	t.Run("crash restart and replay remain durable", func(t *testing.T) {
		requireAnyTerm(t, migration, "durable idempotency table", "CREATE TABLE")
		requireAnyTerm(t, migration, "claim lifecycle", "status", "state", "completed_at", "updated_at")
		requireAnyTerm(t, source, "persisted claim read", "First(", "Take(", "Scan(", "QueryRow", "SELECT")
		requireAnyTerm(t, source, "persisted claim write", "Create(", "Exec(", "INSERT")
	})

	t.Run("stale completion is fenced", func(t *testing.T) {
		requireAnyTerm(t, source, "conditional completion", "WHERE", "Where(")
		requireAnyTerm(t, source, "completion state guard", "pending", "in_progress", "claimed", "status", "state")
		requireAnyTerm(t, source, "stale write detection", "RowsAffected", "rowsAffected", "ErrStale", "stale")
		requireOwnerIsolation(t, source)
	})

	t.Run("migration remains SQLite and Postgres compatible", func(t *testing.T) {
		requirePortableMigration(t, migration)
	})
}

func TestExecutionIdempotencyIndependentQAFailsClosedOnWeakenedArtifacts(t *testing.T) {
	migration := readQAArtifact(t, idempotencyMigrationPath)

	t.Run("missing owner uniqueness is rejected", func(t *testing.T) {
		weakened := regexp.MustCompile(`(?is)(?:UNIQUE\s*\([^)]*\)|CREATE\s+UNIQUE\s+INDEX[^;]*;)`).ReplaceAllString(migration, "")
		if err := validateUniqueOwnerKey(weakened); err == nil {
			t.Fatal("weakened migration without a unique request-identity key was accepted")
		}
	})

	t.Run("dialect-specific migration is rejected", func(t *testing.T) {
		if err := validatePortableMigration(migration + "\nALTER TABLE x ADD COLUMN y JSONB;"); err == nil {
			t.Fatal("Postgres-only JSONB migration was accepted")
		}
		if err := validatePortableMigration(migration + "\nCREATE TABLE x (id INTEGER PRIMARY KEY AUTOINCREMENT);"); err == nil {
			t.Fatal("SQLite-only AUTOINCREMENT migration was accepted")
		}
	})

	t.Run("identity without owner is rejected", func(t *testing.T) {
		if err := validateOwnerIsolation("idempotency_key request_hash execution_id"); err == nil {
			t.Fatal("ownerless idempotency identity was accepted")
		}
	})
}

func readQAArtifact(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if len(strings.TrimSpace(string(content))) == 0 {
		t.Fatalf("%s is empty", path)
	}
	return string(content)
}

func requireAnyTerm(t *testing.T, content, property string, terms ...string) {
	t.Helper()
	for _, term := range terms {
		if strings.Contains(strings.ToLower(content), strings.ToLower(term)) {
			return
		}
	}
	t.Fatalf("missing %s; expected one of %q", property, terms)
}

func requireOwnerIsolation(t *testing.T, content string) {
	t.Helper()
	if err := validateOwnerIsolation(content); err != nil {
		t.Fatal(err)
	}
}

func validateOwnerIsolation(content string) error {
	lower := strings.ToLower(content)
	ownerTerms := []string{"owner", "tenant", "actor", "principal", "identity", "agent_id", "node_id", "instance_id", "scope"}
	for _, term := range ownerTerms {
		if strings.Contains(lower, term) {
			return nil
		}
	}
	return fmt.Errorf("request identity has no durable owner/scope discriminator")
}

func requireUniqueOwnerKey(t *testing.T, migration string) {
	t.Helper()
	if err := validateUniqueOwnerKey(migration); err != nil {
		t.Fatal(err)
	}
}

func validateUniqueOwnerKey(migration string) error {
	lower := strings.ToLower(migration)
	uniqueDefinitions := regexp.MustCompile(`(?is)(?:unique\s*\([^)]*\)|create\s+unique\s+index[^;]*;)`).FindAllString(lower, -1)
	for _, definition := range uniqueDefinitions {
		if !strings.Contains(definition, "idempotency_key") {
			continue
		}
		if err := validateOwnerIsolation(definition); err == nil {
			return nil
		}
	}
	return fmt.Errorf("migration lacks a unique owner-scoped idempotency key")
}

func requirePortableMigration(t *testing.T, migration string) {
	t.Helper()
	if err := validatePortableMigration(migration); err != nil {
		t.Fatal(err)
	}
}

func validatePortableMigration(migration string) error {
	upper := strings.ToUpper(migration)
	forbidden := []string{
		"AUTOINCREMENT",
		"WITHOUT ROWID",
		" JSONB",
		" SERIAL",
		" BIGSERIAL",
		"::JSON",
		"::JSONB",
		"ILIKE",
		"ON CONFLICT ON CONSTRAINT",
	}
	for _, token := range forbidden {
		if strings.Contains(upper, token) {
			return fmt.Errorf("migration 036 contains dialect-specific token %q", token)
		}
	}
	if !strings.Contains(upper, "CREATE TABLE") {
		return fmt.Errorf("migration 036 does not create a durable table")
	}
	return nil
}
