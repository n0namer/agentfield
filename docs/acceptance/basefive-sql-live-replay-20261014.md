# BaseFive Native SQL Live Replay — 2026-10-14

## Authority and bounded scope

- Task: `agentfield:v2-basefive-sql-live-replay-20261014-r4`
- Attempt: `attempt-e0db55ba-5caa-460b-9571-1109ee7d9630`
- Session: `manual-20261014-basefive-northstar-hyper-r4-age-dfe715394858f56e`
- Project: `agentfield`
- Launch Git identity: `44167dc1050f5bff79abc32fab19c0bb50643093`
- Active BMad source of truth: `docs/acceptance/c4-releasegate-reducer-frontier-20261013.md`
- Authorized write scope: `control-plane/internal/storage/execution_idempotency.go`, `control-plane/internal/storage/northstar_basefive_sql_live_replay_20261014_test.go`, this document
- Worker state: `CANDIDATE` / `UNACCEPTED`

This artifact records source-bound implementation and test evidence only. It does not change native task state, CrewSync claims, leases, release state, or acceptance.

## D1 — C4 contract and source identity

The existing production implementation already provides the requested durable SQL idempotency boundary; no production repair was necessary:

```go
type AsyncExecutionIdempotencyResult struct {
    Execution *types.Execution
    Created   bool
    Conflict  bool
}

type AsyncExecutionIdempotencyStore interface {
    LookupAsyncExecutionIdempotency(ctx context.Context, scopeKey, target, requestKey, bodyDigest string) (*AsyncExecutionIdempotencyResult, error)
    CreateAsyncExecutionIdempotent(ctx context.Context, scopeKey, target, requestKey, bodyDigest string, execution *types.Execution) (*AsyncExecutionIdempotencyResult, error)
}
```

- Source: `control-plane/internal/storage/execution_idempotency.go`
- Source SHA-256: `7ff044cdce4e1ec772e3e79a6d4e074e9fece17c2a6857bdf0f4552a1db9cfe2`
- Git/source identity: `44167dc1050f5bff79abc32fab19c0bb50643093`
- Durable fence: one transaction inserts the binding with `ON CONFLICT (scope_key, request_key) DO NOTHING`; a losing caller commits and re-reads the canonical binding.
- Boundary behavior: matching target and digest replay the stored execution; foreign target or digest returns `Conflict` without creating another execution.

## D2 — Native focused test added

Authorized test source:

- File: `control-plane/internal/storage/northstar_basefive_sql_live_replay_20261014_test.go`
- Test: `TestBaseFiveSQLLiveReplay`
- Test SHA-256 after migration-fixture repair: `c759e7870268acc7724408f51f83f3c6797ccdf94b289abc73457a107e630d27`
- Fixture repair: applies the exact `036_execution_request_idempotency.sql` table and index DDL idempotently after each normal `LocalStorage.Initialize` call; production code remains unchanged.

The test exercises the real LocalStorage SQLite schema and asserts:

1. **Positive create:** the first request creates one canonical execution.
2. **Concurrent same-key replay:** two synchronized native SQL calls produce exactly one creator and resolve to the same execution.
3. **Negative foreign digest:** the same scope, target, and request key with a different digest returns `Conflict` and no execution.
4. **Restart recovery:** close and reinitialize the SQLite-backed storage, then replay the original request and recover the original execution ID.

## D3 — Focused native runner receipt

Required validation command:

```text
go test ./internal/storage -run TestBaseFiveSQLLiveReplay -count=1
```

Authorized runner invocation:

```text
go_wave.run_project_tests({"file":"control-plane/internal/storage/northstar_basefive_sql_live_replay_20261014_test.go"})
```

Receipt:

- Status: `FAIL`
- Exit code: `1`
- Engine: `docker:go-test`
- Runner source identity SHA-256: `c759e7870268acc7724408f51f83f3c6797ccdf94b289abc73457a107e630d27`
- Duration: `5216ms`
- Go image: `golang:1.24.0-alpine` (the independent Docker Go 1.26.5 result is also recorded by QA)
- Classification: `BLOCKED_INFRA`, before compilation
- Exact stderr: `go: cannot find main module, but found .git/config in /workspace` followed by `to create a module there, run: go mod init`

The repaired fixture has not reached compilation because the authorized runner still starts in `/workspace` instead of the repository's `control-plane` Go module root. The identical module-root failure was not retried again because it cannot be repaired by changing the test.

## D4 — Regression and limitations

- Affected native regression was not run: the repaired focused runner was blocked before compilation by the module-root error.
- No runtime PASS is claimed for the repaired test.
- No production source file was changed because source inspection found the transaction fence and replay/conflict behavior already implemented; the defect was isolated to fixture migration setup.
- No C4 ReleaseGate producer, independent reviewer receipt, or authoritative terminal readback is available in this worker attempt.

## D5 — Self-review and handoff

- Changed paths remain inside the exact manifest write scope.
- The test uses deterministic IDs, a synchronized concurrent native SQL path, a foreign-digest negative assertion, and a close/reinitialize restart assertion.
- The source contract and implementation SHA are bound to the launch Git identity.
- The native test evidence is explicitly `BLOCKED_INFRA`, not PASS.
- Independent reviewer/reducer must rerun from the correct Go module root, execute affected regression, and perform authoritative terminal readback before any acceptance decision.
