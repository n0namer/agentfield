# MVP R11G SQLite idempotency migration contract

## Scope

This artifact covers the isolated SQLite fixtures used by the MVP R9 atomic replay tests. It does not change production initialization, migration tooling, deployment behavior, or acceptance state.

## Findings

- `LocalStorage.Initialize` invokes `createSchema`; it does not execute Goose migrations.
- `control-plane/migrations/036_execution_request_idempotency.sql` is the authoritative lifecycle contract for `execution_request_idempotency`.
- The prior R9 fixture initialized disposable SQLite and immediately exercised `CreateAsyncExecutionIdempotent`, so the missing table was a fixture omission rather than evidence that production bootstrap should create migration 036's table.
- The R9 fixture now reads and executes only migration 036's Up section against its disposable SQLite database. This mirrors the independent SQL oracle and never targets production data.

## Coverage

`TestMVPR9_SQLAtomicReplayNative` exercises the canonical migrated fixture for:

1. successful replay returning one canonical execution;
2. wrong digest and wrong target conflicts;
3. canceled requests leaving no durable binding;
4. concurrent requests producing exactly one created execution; and
5. restart replay returning the original execution without a duplicate row.

`TestMVPR11GMigrationContract` verifies the lifecycle boundary: an initialized disposable SQLite database lacks the table before migration 036, and the canonical Up section then creates the durable table, composite primary key, unique execution fence, and execution lookup index.

## Focused runner receipt

- Requested command: `go test ./internal/storage -run 'TestMVPR9_SQLAtomicReplayNative|TestMVPR11G' -count=1`
- Scoped runner invocation: `go_wave.run_project_tests` for `control-plane/internal/storage/northstar_mvp_r9_atomic_replay_test.go`
- Result: `FAIL`, exit `1`, before Go compilation.
- Runner image: `golang:1.24.0-alpine`; this is not the requested Go `1.26.5`.
- Exact stderr: `go: cannot find main module, but found .git/config in /workspace` followed by `to create a module there, run: go mod init`.
- Runner source identity before the fixture change: `8fe7348bfaabe84c41f3a781ab785cbd5e816f0d9f2069b172e2f24ff6d59758`.
- The focused check is infrastructure-blocked because the runner executes from `/workspace` instead of the Go module root `control-plane`. No repeated invocation was attempted.

## Required follow-up

Run the requested command from `control-plane` with Go `1.26.5` against this source state. Independent QA must verify the focused result and retain terminal acceptance authority. This worker result remains `UNACCEPTED`.
