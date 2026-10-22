# Base20 Async Idempotency Race/Restart Independent QA — 2026-10-13

## Authority and status

- Task: `agentfield:v2-async-idempotency-independent-qa-sql-fencing-20261013-r3`
- Attempt: `attempt-f0b19c1d-72b9-4137-94e9-73163fa3de5d`
- Session: `manual-20261013-five-modules-targeted-repair-qa-485616df1a867054`
- Scope: independent QA artifacts only
- Verdict: `BLOCKED` / `UNACCEPTED`

No production Go source, migration, live database, server, Ruflo Native state, task claim, lease, deployment, or process was changed.

## Named defect and fail-closed classification

The reported production symptom is two distinct `execution_id` values for the same `Idempotency-Key` and the same request payload. That symptom is a P0 violation of exactly-once request admission: at least one deployed request path did not resolve the durable owner/key/payload identity to one authoritative execution.

This worker could not honestly classify the checked-in candidate as fixed or still defective at runtime. The authorized Go runner failed before package compilation because it ran from repository root `/workspace`, while the Go module is nested under `control-plane`. No mock or static inspection is labeled as a runtime PASS.

## Independent QA artifact

`control-plane/internal/storage/execution_idempotency_independent_qa_test.go` is a new, separate QA file. It does not overwrite the existing async test suite. Its executable assertions bind the candidate implementation and migration 036 to these required properties:

1. Durable request identity includes idempotency key, request/payload digest, execution ID, and an owner/scope discriminator.
2. Same owner + key + payload uses an atomic claim/lookup path and replays the existing execution identity.
3. Same owner + key + different payload has an explicit mismatch/conflict branch.
4. Restart/replay evidence is stored in a durable table and read/written through persisted claim paths.
5. Completion is conditional, state guarded, owner scoped, and detects a stale zero-row write or equivalent stale result.
6. Migration 036 has a unique owner-scoped idempotency key.
7. Migration 036 rejects a bounded set of SQLite-only and PostgreSQL-only SQL constructs.
8. Negative mutation subtests prove the QA checks reject missing uniqueness, missing ownership, PostgreSQL-only `JSONB`, and SQLite-only `AUTOINCREMENT`.

These checks are source- and SQL-backed, but they are not a substitute for executing concurrent requests against SQLite and PostgreSQL. Exactly-one external effect, process crash/restart, and live request admission remain unverified until the Go package test and a disposable database fixture run.

## Validation receipt

Authorized call:

```text
go_wave.run_project_tests({"file":"control-plane/internal/storage/execution_idempotency_independent_qa_test.go"})
```

Result:

```text
status: FAIL
engine: docker:go-test
go_image: golang:1.24.0-alpine
source_identity_sha256: 14688117b5c364bcbf733d528124e1ba588a4d21cf208ec92fa934b24f84b9ff
exit_code: 1
stderr:
go: cannot find main module, but found .git/config in /workspace
    to create a module there, run:
    go mod init
```

The failure occurs before compilation or test discovery. It is classified as `INFRA BLOCKED`, not a test failure and not a PASS. The identical runner call was not retried after the root-cause receipt.

## Source and producer consistency

The runner bound the attempted validation to repository source identity `14688117b5c364bcbf733d528124e1ba588a4d21cf208ec92fa934b24f84b9ff`.

No individual SHA-256 values for `execute.go`, `execution_idempotency.go`, or `036_execution_request_idempotency.sql` were present in the permitted evidence. Therefore comparison to the original producer's exact source SHAs is unavailable and acceptance fails closed. Existing independent control-plane QA remains `UNACCEPTED` and explicitly records async request idempotency as an unproved P0.

## Required independent execution

A Go-capable verifier must run from the nested module root:

```text
cd control-plane
go test ./internal/storage -run 'TestExecutionIdempotencyIndependentQA' -count=1
go test ./internal/storage -run 'Idempotency' -race -count=20
```

The verifier must then use disposable SQLite and PostgreSQL fixtures to issue concurrent same-owner/same-key/same-payload claims and assert:

- one durable identity row;
- one authoritative `execution_id`;
- same-key/different-body conflict;
- foreign owner/key isolation;
- replay after close/reopen or process restart;
- stale completion cannot replace the authoritative terminal result;
- exactly one externally observable dispatch/effect.

## Original-producer repair boundary

If the focused test, race loop, or disposable database fixtures reveal duplicate execution identity or a stale completion overwrite, the repair belongs to the original async idempotency producer. The minimum production repair must be inside the atomic durable claim/completion path and migration compatibility surface; this QA worker is not authorized to edit those files.

The producer must return:

- exact SHAs for `execute.go`, `execution_idempotency.go`, and migration 036;
- SQLite and PostgreSQL test commands with exit codes and untruncated output;
- a same-key/same-payload concurrency result proving one execution ID and one effect;
- a same-key/different-body conflict result;
- owner isolation and stale completion fencing evidence.

## Handoff

- Worker result: `BLOCKED`, not `CANDIDATE`, because focused executable evidence did not reach compilation.
- Product acceptance: `UNACCEPTED`.
- Next action: repair the scoped Go runner to execute from `control-plane`, then run the listed focused, race, restart, and disposable-database checks. Independent reviewer/reducer and CrewSync exact terminal readback remain mandatory.
