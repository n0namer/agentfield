# MVP R9 SQL Atomic Replay Native Producer

## Scope

- Task: `agentfield:v2-hyperwave-sql-atomic-replay-native-20261014-r9`
- Goal: exercise the existing SQL-backed async execution idempotency path in isolated SQLite without changing the active r8 handler or live Docker state.
- Source of truth: `docs/acceptance/c4-releasegate-reducer-frontier-20261013.md`.
- Authorized implementation files: `control-plane/internal/storage/execution_idempotency.go`, `control-plane/internal/storage/northstar_mvp_r9_atomic_replay_test.go`, and this artifact.

## Source finding

`control-plane/internal/storage/execution_idempotency.go` already provides the bounded native mechanism:

- `CreateAsyncExecutionIdempotent` starts a SQL transaction and claims `(scope_key, request_key)` with `ON CONFLICT ... DO NOTHING`.
- The winning claim inserts the execution in the same transaction before commit.
- A losing claim commits, reloads the persisted binding, and returns the canonical execution.
- `LookupAsyncExecutionIdempotency` returns `Conflict` when target or body digest differs.
- The existing path rejects a nil execution payload and propagates canceled transaction errors.

No production-source repair was justified by the inspected evidence; the missing bounded deliverable was executable MVP R9 coverage.

## Added focused coverage

`control-plane/internal/storage/northstar_mvp_r9_atomic_replay_test.go` adds `TestMVPR9_SQLAtomicReplayNative` using an isolated temporary SQLite database and Bolt path. It checks:

1. migration 036 bootstraps `execution_request_idempotency`;
2. a real native create returns a typed persisted execution;
3. the same exact key and digest replay the canonical execution;
4. wrong digest and wrong target return `Conflict` without a second execution;
5. a canceled request leaves no durable binding;
6. concurrent exact-key requests produce one winner and one canonical execution;
7. close/reopen replay preserves the canonical execution and avoids duplication.

## Validation receipt

- Command requested: `go test ./internal/storage -run TestMVPR9 -count=1`.
- Authorized runner invocation: `go_wave.run_project_tests({"file":"control-plane/internal/storage/northstar_mvp_r9_atomic_replay_test.go"})`.
- Result: `FAIL`, exit code `1`, engine `docker:go-test`.
- Exact runner error: `go: cannot find main module, but found .git/config in /workspace` followed by `to create a module there, run: go mod init`.
- Classification: `INFRA BLOCKED` before compilation; no product PASS is claimed and the denied runner was not retried identically.
- Runner source identity: `55ec38d4039e139de397fb4e8a6f5809d57ca8e935aeccb840c9e7dd0e642e69` before the final scoped test cleanup. The final test file was subsequently self-reviewed and changed to SHA-256 `8fe7348bfaabe84c41f3a781ab785cbd5e816f0d9f2069b172e2f24ff6d59758` before this artifact was written.

## Self-review and remaining gate

- The active r8 handler and live Docker state were not modified.
- The focused test file is within the exact write scope and uses only existing storage APIs plus isolated temporary paths.
- Independent native test execution from the repository's Go module root remains required.
- Independent QA/reducer review and coordinator terminal readback remain required; this producer artifact is `UNACCEPTED`.
