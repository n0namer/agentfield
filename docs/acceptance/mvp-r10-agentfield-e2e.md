# MVP R10 AgentField E2E Evidence — 2026-10-14

## Scope

This artifact records the bounded MVP R10 source and executable-test evidence for durable async idempotency over a disposable HTTP boundary. It does not claim product acceptance, release, deployment, or CrewSync terminal state.

- Project: `agentfield`
- Task: `agentfield:v2-r10-native-mvp-e2e-20261014-r1`
- Attempt: `attempt-20889932-4f44-4a18-9034-3aead1002988`
- Session: `manual-20261014-basefive-r10-active-agentfield--08e36f2ec135fdb1`
- Source identity supplied at launch: `44167dc1050f5bff79abc32fab19c0bb50643093`

## C4/source DTO receipt

The implementation source is `control-plane/internal/handlers/execute.go`, SHA-256 `be09150579e29e16ce9d96775ef1b161898d829570b9bdd5885c9db5327a25c1`.

The source-bound DTO and internal request contract used by this slice are:

- `AsyncExecuteResponse`: `execution_id`, `run_id`, `workflow_id`, `status`, `target`, `type`, `created_at`, optional `enqueued_at`, `webhook_registered`, and optional `webhook_error`.
- `asyncIdempotencyRequest`: caller/target `scopeKey`, route `target`, client `requestKey`, and SHA-256 `bodyDigest`.
- The durable adapter `asyncIdempotentExecutionStore` maps a created record to a typed replay or conflict result without a second execution write.

The authorized focused test is `control-plane/internal/handlers/northstar_mvp_r10_disposable_http_test.go`, SHA-256 `e17ed5fa49470165c81835d8d8d7f2f02184157d7c4d9d9b6a22264d030be765`.

## Executable behavior covered

`TestMVPR10` uses `httptest.NewServer` and the production `asyncIdempotentExecutionStore` adapter over a durable in-memory persistence seam:

1. Positive request returns typed execution/run identifiers; replay returns the same identifiers and the durable write count remains one.
2. Same caller/key with a different body returns conflict.
3. A different caller with the same key is isolated and creates its own durable record.
4. A cancelled context returns `context.Canceled` before persistence and does not increment the durable write count.
5. Recreating the disposable HTTP server with the same store replays the original durable result, modeling controller restart without duplicate effect.

## Validation receipt

Requested focused validation: `go test ./internal/handlers -run TestMVPR10 -count=1`.

Authorized runner invocation: `go_wave.run_project_tests({"file":"control-plane/internal/handlers/northstar_mvp_r10_disposable_http_test.go"})`.

Result: `INFRA BLOCKED` before compilation.

- Engine: `docker:go-test`
- Source identity from runner: `e17ed5fa49470165c81835d8d8d7f2f02184157d7c4d9d9b6a22264d030be765`
- Exit code: `1`
- Exact stderr: `go: cannot find main module, but found .git/config in /workspace` followed by `to create a module there, run: go mod init`
- No product test result is inferred from this failure.

The runner was not retried identically after the first deterministic module-root failure. A coordinator-owned rerun must execute from `control-plane` (or configure the authorized runner's module root) and preserve the resulting exit status and source identity.

## Self-review

- Changed files are limited to the authorized write scope.
- No production source edit was required: the existing source already contains caller-scoped keying, request-body digest conflict detection, durable create-or-replay semantics, and terminal error handling for unavailable idempotency storage.
- The test covers positive behavior, two fail-closed negative boundaries, cancellation, and restart replay.
- Acceptance remains `UNACCEPTED` pending a module-root-correct native run, affected regression, independent QA/reducer review, and coordinator terminal readback.
