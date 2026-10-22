# BaseFive SQL Oracle Parser Repair — 2026-10-14

## Scope and binding

- Task: `agentfield:v2-basefive-storage-sql-oracle-parser-repair-20261014-r5`
- Attempt: `attempt-8b3e6f48-b998-4cbb-90cd-18c08b0b8c1e`
- Session: `manual-20261014-basefive-native-gate-repair-r5--83504aaf5859d613`
- Authorized source file: `control-plane/internal/storage/northstar_independent_sql_oracle_20261014_test.go`
- Authorized evidence artifact: this file.
- No production storage source, task state, leases, locks, or release state were changed.

## Independent QA repair

Independent QA found that the prior candidate initialized `LocalStorage` but never created the canonical `execution_request_idempotency` table, causing all three native Northstar variants to fail with `no such table: execution_request_idempotency`. This was a fixture omission, not evidence against the SQL idempotency implementation.

The repaired test reads the canonical `../../migrations/036_execution_request_idempotency.sql` fixture through the existing `readQAArtifact` helper, extracts only the Goose `Up` section, and executes it after every new `LocalStorage.Initialize`:

- `initializeNorthstarStorage` applies the migration for each independently initialized store.
- The restart test applies it after both the initial and reinitialized storage instances.
- The `Down` section is never executed.
- The migration creates table `execution_request_idempotency` with primary key `(scope_key, request_key)`, unique `execution_id`, and index `idx_execution_request_idempotency_execution`.
- Existing physical same-path SQLite concurrency and distinct Bolt paths are preserved.

## Native assertions

- Positive/concurrent: two initialized stores race on one request identity and must return one canonical execution with exactly one creator.
- Negative/conflict: a different digest is rejected with `Conflict` and no execution.
- Negative/conflict: a different target is rejected with `Conflict` and no execution.
- Restart/replay: after closing and reopening initialized storage, replay returns the durable canonical execution without creating a duplicate.

## Verification receipt

- Authorized runner: `go_wave.run_project_tests({"file":"control-plane/internal/storage/northstar_independent_sql_oracle_20261014_test.go"})`
- Result: `FAIL`, exit code `1`, before compilation.
- Exact stderr: `go: cannot find main module, but found .git/config in /workspace`, followed by `to create a module there, run: go mod init`.
- Runner image: `golang:1.24.0-alpine`; Go 1.26.5 execution is not established.
- Post-repair source identity: `2cd41b4f3671fa4264f3cd0cba70c4957b9324cc3502a74992aaad8dd22a2e1c`.

This is `BLOCKED_INFRA`, not a product PASS. No identical runner retry was made after the repeated module-root failure.

## Self-review

- Only the claimed Northstar test and this acceptance artifact changed.
- The canonical migration path, table name, primary key, unique execution index, and Up-only extraction are recorded explicitly.
- Digest/target fail-closed assertions, same-path concurrent replay, and restart coverage remain intact.
- Native Go 1.26.5 compilation/execution and affected storage regression remain unverified until the coordinator runs from the correct Go module root.
- Independent QA/reducer and coordinator terminal readback remain required before acceptance.
