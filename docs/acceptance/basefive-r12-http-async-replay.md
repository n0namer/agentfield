# BaseFive R12 HTTP Async Replay Evidence — 2026-10-14

## Scope and authority

- Task: `agentfield:v2-r12-http-async-replay-20261014-r1`
- Attempt: `attempt-7c06feba-0d98-4390-a705-7be5d0e508e4`
- Session: `manual-20261014-basefive-r12-producers-agentfie-a160c7810d564f71`
- Launch Git identity: `44167dc1050f5bff79abc32fab19c0bb50643093`
- Active C4 gate: `docs/acceptance/c4-releasegate-reducer-frontier-20261013.md`
- C4 artifact SHA supplied at launch: `2bab9006af2f8a4f16713245176bba862b82a1a9b2374fee8bb02cdbec7a4e03`
- Acceptance remains `UNACCEPTED`; independent QA and the reducer retain authority.

## Existing mechanism evidence

The reusable disposable HTTP fixture in `control-plane/internal/handlers/northstar_mvp_r10_disposable_http_test.go` constructs its idempotency scope from `X-Verified-Caller` and `X-Target`, computes a SHA-256 body digest, and forwards the request to `asyncIdempotentExecutionStore`.

The HTTP DTO is `mvpR10HTTPResult`:

| Field | Meaning |
|---|---|
| `execution_id` | Durable execution identity returned on creation and replay |
| `run_id` | Durable run identity returned on creation and replay |
| `status` | `created` for the first request or `replay` for a matching retry |
| `error` | Conflict or internal failure description |

The fixture maps a replay error to `202 Accepted`, a digest conflict to `409 Conflict`, and a new request to `202 Accepted`. This is test-fixture evidence only; it does not promote the blocked C4 gate.

## R12 coverage

`control-plane/internal/handlers/northstar_mvp_r12_http_replay_test.go` adds a focused HTTP contract around the existing fixture:

1. **Positive replay:** a matching caller, target, key, and payload returns the original execution/run with `replay`; the store observes exactly one durable effect.
2. **Foreign-owner isolation:** a different verified caller using the same target, key, and payload receives a distinct creation rather than the first caller's execution. The original caller still replays its own result, proving no foreign-owner drift or response disclosure.
3. **Digest rejection:** the same caller, target, and key with a changed body receives `409 Conflict`; the store remains at one durable effect.
4. **Restart recovery:** recreating the HTTP server over the same store returns the original replay result and does not add a durable effect.

The foreign-owner case intentionally checks caller-scoped isolation rather than a `409` response: the existing fixture includes the verified caller in the idempotency scope, so a foreign caller is a distinct request owner and must not obtain the original owner’s execution.

## Focused validation receipt

Requested command:

```text
go test ./internal/handlers -run TestMVPR12 -count=1
```

Authorized runner invocation:

```text
go_wave.run_project_tests({"file":"control-plane/internal/handlers/northstar_mvp_r12_http_replay_test.go"})
```

Result: `FAIL`, exit `1`, before Go compilation.

- Runner engine: `docker:go-test` (`golang:1.24.0-alpine`)
- Runner source identity SHA-256: `199c6ace7fdc489650610d4df6e2a655729029c669d6d1c45c9ef1c6c6bdd20d`
- Package selected by the runner: `./control-plane/internal/handlers`
- Exact stderr: `go: cannot find main module, but found .git/config in /workspace` followed by `to create a module there, run: go mod init`.

This is an infrastructure-blocked receipt, not a passing test claim. The same runner was not retried because the failure occurs before compilation and the task does not authorize runner or module-root repair.

## Self-review and next action

- The new test reuses the existing HTTP fixture and DTO rather than introducing a second handler or contract.
- Changed code remains limited to the authorized R12 test target; the pre-existing R10 fixture remains unmodified.
- No deployment, task-state mutation, acceptance claim, or external side effect was performed.
- Independent QA must run the requested command from the `control-plane` module root (or repair the authorized runner's working directory) before any replay behavior is represented as passing.
