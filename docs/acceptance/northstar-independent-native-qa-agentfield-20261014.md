# Northstar Independent Native SQL QA — 2026-10-14

## Authority

- Task: `agentfield:v2-independent-native-longbundle-qa-20261014-r1`
- Attempt: `attempt-923917a3-74c2-4cbd-94fa-3ffbbafe1062`
- Session: `manual-20261014-base5-independent-native-qa-hyp-99e31d96826794a6`
- Project source identity supplied at launch: `44167dc1050f5bff79abc32fab19c0bb50643093`
- Active BMad source of truth: `docs/acceptance/c4-releasegate-reducer-frontier-20261013.md`
- Worker state: candidate / `UNACCEPTED`

This artifact is independent QA evidence only. It does not change production source, migrations, native task state, claims, leases, release state, or acceptance.

## Definition of Done

- One positive concurrent duplicate case proves one SQL winner and one canonical execution.
- Two fail-closed negative/recovery cases reject conflicting identity and prove replay after database restart.
- The test binds to the exact native SQL storage method names and immutable source SHA.
- Focused and affected regression commands are attempted through the authorized runner; failures are classified without fabricated PASS.
- Producer receipt comparison and remaining verifier work are explicit.

## Source-bound inspection

- Inspected source: `control-plane/internal/storage/execution_idempotency.go`
- Immutable source SHA-256: `7ff044cdce4e1ec772e3e79a6d4e074e9fece17c2a6857bdf0f4552a1db9cfe2`
- Verified signatures: `LookupAsyncExecutionIdempotency(ctx, scopeKey, target, requestKey, bodyDigest)` and `CreateAsyncExecutionIdempotent(ctx, scopeKey, target, requestKey, bodyDigest, execution)`.
- The implementation uses a database transaction, `ON CONFLICT (scope_key, request_key) DO NOTHING`, `RowsAffected`, canonical replay lookup, digest/target conflict detection, and durable execution insertion.
- No production source or migration was edited.

## Scoped independent test

Added `control-plane/internal/storage/northstar_independent_sql_oracle_20261014_test.go`.

- Test SHA-256: `202d910ddc7a18d03ab68894384c9539d6ce475275324446c5c24a4e126684f7`
- `TestNorthstarIndependentSQLOracleConcurrentDuplicate` starts two independent SQLite connections against one disposable database, asserts two successful results, exactly one `Created` result, one canonical execution ID, one binding, and one execution row.
- `TestNorthstarIndependentSQLOracleRejectsConflictingDigestAndTarget` asserts both a changed body digest and changed target return `Conflict` without an execution.
- `TestNorthstarIndependentSQLOraclePersistsAcrossRestart` closes and reopens the database, then asserts the original execution is replayed and a replacement execution does not win.
- All databases are disposable `testing.T.TempDir()` files; no external effects or concurrent production writes occur.
- Existing runtime fixtures are reused; no duplicate helper or production implementation was added.

## Validation receipts

### Focused native test

Requested scoped runner call:

`go_wave.run_project_tests({"file":"control-plane/internal/storage/northstar_independent_sql_oracle_20261014_test.go"})`

Receipt:

- Status: `FAIL` before compilation
- Engine: `docker:go-test`
- Image: `golang:1.24.0-alpine`
- Runner source identity: `202d910ddc7a18d03ab68894384c9539d6ce475275324446c5c24a4e126684f7`
- Exit code: `1`
- Exact stderr: `go: cannot find main module, but found .git/config in /workspace` followed by `to create a module there, run: go mod init`
- Classification: `INFRA BLOCKED`; the runner used `/workspace` rather than the `control-plane` Go module and did not reach compilation.

The identical focused call was not retried.

### Affected regression

Attempted scoped runner call:

`go_wave.run_project_tests({"file":"control-plane/internal/storage/northstar_sql_idempotency_20261013_test.go"})`

Receipt:

- Status: not executed
- Exact runner response: `GO_WAVE_NODE_ERROR:test_outside_task_write_scope`
- Classification: `INFRA / AUTHORIZATION BLOCKED`; the existing regression file is outside this attempt's exact write/test scope.

No regression PASS is claimed.

## Producer receipt comparison

The available 2026-10-14 longbundle producer artifact is `docs/acceptance/longbundle-harness-live-result-20261014.md`:

- Task: `agentfield:v2-longbundle-harness-live-result-20261014-r1`
- Attempt: `attempt-1c4befdc-0f7b-42b8-8b11-ae35269f4707`
- Session: `manual-20261014-base5-longbundles-northstar-r1--4fd3016b881bfe68`
- Producer source inspected there: Python `sdk/python/agentfield/harness/_runner.py`, SHA-256 `06c6eccd3f17a72a802baead62ddfa534b8bfbcc07c76c039b0dbbe485f9cfa8`
- Its pytest runner was also blocked by unavailable `.venv`; it is not a native SQL producer receipt and does not prove this Go storage behavior.

No exact native SQL producer receipt for the current task was available in the permitted evidence. The test and report therefore remain independent candidate evidence, not acceptance.

## Self-review and handoff

- Changed files are exactly the two manifest-authorized paths.
- Positive, conflict, and restart boundaries are encoded without production writes.
- The runner never compiled the test, so runtime behavior remains unverified and no GREEN status is claimed.
- Independent verifier/reviewer must run the focused test from `control-plane` with the required Go 1.26.5 and writable cache, then select an authorized affected regression.
- If runtime execution exposes a concurrency failure, repair must be made by the original producer owner; this QA attempt must not edit `execution_idempotency.go`.
- Acceptance remains `UNACCEPTED` pending separate reducer and coordinator authority.
