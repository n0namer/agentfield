# Northstar SQL Idempotency Race C4 QA — 2026-10-13

## Authority and state

- Task: `agentfield:v2-c4-native-acceptance-sql-race-20261013-r1`
- Attempt: `attempt-8b66a55c-d5a4-43f2-8338-01de03079c58`
- Session: `manual-20261013-c4-native-acceptance-retry-cana-a57ced03563ef40b`
- Git identity supplied at launch: `44167dc1050f5bff79abc32fab19c0bb50643093`
- Acceptance authority: independent reviewer/coordinator only.
- Worker verdict: `BLOCKED` / `UNACCEPTED`; the scoped Go runner failed before compilation.

This artifact records a source-backed native SQL test candidate and exact runner evidence. It does not mutate production source, native task state, CrewSync claims, leases, or acceptance state.

## C4 storage boundary

The read-only production boundary is `control-plane/internal/storage/execution_idempotency.go`, SHA-256 `c118186dd8c5b01bf527afbdc75f39dcea2b4d2a672893b4bfd4eebce5da95da`.

`AsyncExecutionIdempotencyStore` exposes:

- `LookupAsyncExecutionIdempotency(ctx context.Context, scopeKey, target, requestKey, bodyDigest string) (*AsyncExecutionIdempotencyResult, error)`
- `CreateAsyncExecutionIdempotent(ctx context.Context, scopeKey, target, requestKey, bodyDigest string, execution *types.Execution) (*AsyncExecutionIdempotencyResult, error)`

The DTO is `AsyncExecutionIdempotencyResult` with `Execution *types.Execution`, `Created bool`, and `Conflict bool`. The expected contract is one durable `(scope_key, request_key)` winner, canonical replay for the same target/digest, `Conflict` for target or digest mismatch, and no cross-scope leakage.

## Added native SQL scenario

`control-plane/internal/storage/northstar_sql_idempotency_20261013_test.go`, SHA-256 `7dbc0bd6007df77070c38b5ed1862941577e1cb24831fcfd0f5835ce897a0300`, adds `TestNorthstarSQLIdempotencyRaceRestartAndConflicts` using the registered local SQLite driver, real `database/sql` connections, real transactions, and real tables. It reuses the established runtime SQL harness in `execution_idempotency_runtime_counterexample_test.go`, SHA-256 `42a4732c9ed2d8b464521112b55efa42796ae6a799b0ab99db5d3b4bc918cd40`.

The test asserts:

1. Two separate SQL connections concurrently call `CreateAsyncExecutionIdempotent` with the same scope, target, request key, and digest.
2. Both calls return successfully, exactly one reports `Created`, and both return the same canonical execution ID.
3. After both original connections close and storage is reopened, a new create attempt replays the canonical execution rather than creating another record.
4. A different digest for the same scoped key returns `Conflict` with no execution.
5. The same request key in a foreign scope returns no binding and no conflict.
6. SQL contains exactly one canonical binding and one execution record after the race and restart.

No mock is described as live or native SQL evidence.

## Source-backed race hypothesis

The production implementation starts a transaction, reads the binding, and inserts when the read returns `sql.ErrNoRows`. Two deferred SQLite transactions can both observe no row. The losing insert or lock error is returned directly as `insert async idempotency binding`; the function has no duplicate-key/lock recovery path that re-reads and returns the committed canonical winner. Therefore the new test is expected to expose either a uniqueness or database-lock error instead of two successful canonical results. This is a source-backed hypothesis, not a runtime-confirmed defect in this attempt, because the runner never reached compilation.

The smallest production repair, if the test reproduces, belongs to the existing original producer owner for `control-plane/internal/storage/execution_idempotency.go`: classify retryable lock/duplicate-key failure, end the losing transaction, re-read the durable binding, then return canonical replay or conflict. The authorized evidence does not contain that producer task's exact CrewSync ID, so this worker does not invent one or dispatch a replacement. The coordinator must route the repair to the existing original owner.

## Test receipt

Focused invocation:

`go_wave.run_project_tests({"file":"control-plane/internal/storage/northstar_sql_idempotency_20261013_test.go"})`

Result:

- Status: `FAIL` before compilation; classified `INFRA BLOCKED`, not product failure.
- Engine: `docker:go-test`
- Package selected by runner: `./control-plane/internal/storage`
- Go image: `golang:1.24.0-alpine`
- Exit code: `1`
- Source identity SHA-256: `7dbc0bd6007df77070c38b5ed1862941577e1cb24831fcfd0f5835ce897a0300`
- Exact stderr: `go: cannot find main module, but found .git/config in /workspace` followed by `to create a module there, run: go mod init`.

The identical denied/mis-rooted runner call was not retried. Affected regression was not run because it would use the same module-root path and repeat the already classified infrastructure failure.

Runnable commands for an operator or corrected scoped runner, from the `control-plane` module root:

- Focused: `go test ./internal/storage -run '^TestNorthstarSQLIdempotencyRaceRestartAndConflicts$' -count=1`
- Affected regression: `go test ./internal/storage -run 'Test(CreateAsyncExecutionIdempotent|IdempotencyRuntimeCounterexample|IdempotencyConcurrentSameKeyCreatesOneCanonicalExecution|NorthstarSQLIdempotency)' -count=1`

These commands are prescribed recovery steps and are not reported as run or passing.

## Acceptance mapping

- Actual native positive path and invariant assertion: implemented in the new real-SQL test; runtime result is blocked before compilation.
- At least two adversarial negatives including stale/restart: restart replay, digest conflict, and foreign-scope isolation are asserted; runtime result is blocked before compilation. Stale storage state is represented by closing all original connections before reopening the database.
- Exact source SHA, runnable command, and exit code: recorded above; no unrun check is labeled PASS.
- C4 function/DTO boundary and affected-only repair route: recorded above; production repair remains coordinator-routed to the original producer owner.
- Independent GREEN: not claimed; independent reviewer remains required.

## Self-review

- Changes are limited to the two authorized write-scope files.
- The test uses the actual `LocalStorage` methods and SQL database state, not a behavioral mock.
- The SQL invariant is checked after restart, so transient in-memory agreement cannot satisfy the test.
- Race errors are sorted before failure to keep evidence deterministic.
- No production refactor, migration, task administration, commit, push, deploy, or acceptance mutation occurred.
- Remaining blocker is the runner working directory/module-root defect; product behavior remains unverified in this attempt.
