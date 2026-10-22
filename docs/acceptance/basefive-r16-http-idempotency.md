# Basefive R16 HTTP idempotency — resumed R19 evidence

## State and authority

- Date: 2026-10-14.
- Task: `agentfield:v2-r19-http-idempotency-20261014-r1`.
- Attempt: `attempt-fae05e7f-4035-40dd-81ae-6885e8ba9cbb`.
- Session: `manual-20261014-basefive-r19-affected-native-ag-d962fbb28891b24a`.
- Target: `D:/Users/NIKITA/Documents/DEV/universal-solver-site/agentfield`, not canonical BMAD-MNNZ.
- Git at launch: `44167dc1050f5bff79abc32fab19c0bb50643093` (launcher identity; live Git status/branch unavailable through the authorized interfaces).
- Outcome: `BLOCKED` / `UNACCEPTED`; no product green or independent acceptance is established.
- Stage: resumed implementation verification. Governing existing gate: `docs/acceptance/c4-releasegate-reducer-frontier-20261013.md`; this bounded HTTP evidence does not replace or satisfy the native C4 ReleaseGate.

BMad Help was loaded first. Verification Before Completion was used to classify the fresh execution receipt. Config resolution, graph navigation, Git inspection and shell workflows are unavailable under the scoped tool policy; no claim that those method steps ran is made. No competing planning artifact, production edit, task claim, external send or acceptance mutation was made.

## Bounded Definition of Done

Preserve the existing R16 candidate, bind current handler/test bytes and visible API, reproduce the reported failure through the authorized runner, make only evidence-backed scoped repairs, prove positive/two-negative/crash behavior and affected regression with native Go outputs, then publish one original-task UNACCEPTED report for independent QA. If compilation is infrastructure-blocked, preserve the exact receipt and outstanding behavioral gates rather than weakening fixtures or reporting PASS.

Initial useful-work estimate: investigation 4 minutes, implementation 3, tests 5, conditional repair 4, review 2, evidence 2 (20 minutes excluding external coordination). The pre-compilation failure reduced useful authorized work to source review and evidence preservation. Actual elapsed work was not instrumented; no elapsed-time claim is made.

## Source-bound inspection

| File | Exact SHA-256 | Finding |
|---|---|---|
| `control-plane/internal/handlers/execute.go` | `be09150579e29e16ce9d96775ef1b161898d829570b9bdd5885c9db5327a25c1` | Scoped range-read receipt; matches the existing test pin and the R14 architecture contract's handler identity. |
| `control-plane/internal/handlers/northstar_r16_http_canary_test.go` | `d0801c513c08ec16426edd22dea3bb9f4b8df24bc2ca31808944e210dd53d5ba` | Scoped range-read receipt; identical to the runner's reported source identity. Preserved without edits. |

FACT: The mechanism already exists in the authorized handler. `ExecuteAsyncHandler(ExecutionStore, services.PayloadStore, services.WebhookDispatcher, time.Duration, string)` returns `gin.HandlerFunc`. `ExecuteRequest` has input, context, webhook and run_metadata fields; `AsyncExecuteResponse` has execution_id, run_id, workflow_id, status, target, type, created_at, enqueued_at, webhook_registered and webhook_error fields. Existing reflection assertions pin those field counts/tags, but did not execute in this attempt.

FACT: Source-visible idempotency validates the key before accessing storage, reads the verified caller through `middleware.GetVerifiedCallerDID`, scopes by caller DID plus NUL plus target DID, and hashes raw request bytes. The async handler checks durable-store availability, lookup errors, request conflict and replay before worker admission. The adapter delegates atomic creation to the existing storage interface and distinguishes conflict from replay. Storage implementation and middleware internals are outside this attempt's read scope; their semantics are not independently verified here.

FACT: `docs/architecture/basefive-r14-contracts.json` labels wire version unversioned and router unverified, and explicitly warns that a process-local queue does not prove restart dispatch. The C4 reducer artifact concerns a separate native ReleaseGate and remains blocked for missing producer/QA/interface/restart evidence. An HTTP replay receipt cannot stand in for a native task-to-result C4 E2E receipt.

## Existing candidate coverage and gaps

- `TestR16HTTPSourceAndInterfacePin`: handler bytes, exported function assignability and DTO wire field counts/tags.
- `TestR16HTTPRejectBeforeStoreOrDispatch`: real local HTTP requests for unverified caller, blank key, embedded space and oversized key; nil controller store/pool makes early rejection material. Checks status, JSON and absence of execution/replay receipts.
- `TestR16HTTPCancelledUnverifiedRequest`: cancelled request remains unauthenticated and gets no execution receipt. Cancellation is not a process crash or restart.
- `TestR16HTTPReplayReceiptFormatting`: invokes the response helper twice with one fixture and asserts 202, replay headers and exact JSON. This is formatting coverage, not a full handler/storage/dispatch positive path or proof of effect deduplication.

Remaining behavioral gaps: authenticated durable handler replay, two independent conflict/trust-isolation negatives, dispatch-effect counting, real crash/restart recovery and affected package regression. The existing tests must not be renamed or reinterpreted to fabricate these outcomes. No product defect is established by this attempt's runner output, and no speculative unexecuted test rewrite was made after infrastructure failure.

## Exact fresh runner receipt

Invocation:

```json
{"tool":"go_wave.run_project_tests","arguments":{"file":"control-plane/internal/handlers/northstar_r16_http_canary_test.go"}}
```

Returned fields:

```json
{"status":"FAIL","code":null,"test":"control-plane/internal/handlers/northstar_r16_http_canary_test.go","engine":"docker:go-test","source_identity_sha256":"d0801c513c08ec16426edd22dea3bb9f4b8df24bc2ca31808944e210dd53d5ba","exit_code":1,"signal":null,"error":null,"duration_limit_ms":120000,"output_truncated":false,"stdout":"","package":"./control-plane/internal/handlers","go_image":"golang:1.24.0-alpine"}
```

Exact stderr:

```text
go: cannot find main module, but found .git/config in /workspace
	to create a module there, run:
	go mod init
```

Classification: **INFRA BLOCKED before compilation**, not a failing product assertion. No test count, compile success, runtime PASS or affected regression result exists. The runner did not return its complete internal command; `package` above is metadata, not an invented command receipt. This reproduces the module-root failure recorded in the governing C4 evidence. The identical call was not retried and no foreign module or launcher file was edited. The Go diagnostic's `go mod init` suggestion is not an authorized or appropriate repair for this nested-module task.

Required focused command from the task (NOT successfully executed here):

```text
Working directory: D:/Users/NIKITA/Documents/DEV/universal-solver-site/agentfield/control-plane
go test ./internal/handlers -run TestR16HTTP -count=1
```

After authorized runner-root repair, the operator should execute that exact focused command through the scoped native runner and retain its actual command, exit, output and immutable source identity. Then prove the missing behavioral scenarios within the existing original task and run affected regression `go test ./internal/handlers -count=1` from the same module root when authorized. Do not add a JS surrogate and represent it as native Go evidence.

## Self-review and handoff

- Only this previously absent authorized Markdown evidence artifact was created. Existing R16 Go candidate bytes and all production files remain unchanged by this attempt.
- No fixtures, expectations or source pin were weakened. The prior R16 candidate remains unverified, not superseded by an alleged repair.
- Exact source/test SHAs and the failed native runner receipt are retained; architecture source facts are distinguished from unexecuted assertions and broader C4 evidence.
- The infrastructure repair is outside the exclusive write scope: coordinator/operator must correct the scoped native runner's nested-module working directory, preserving original task/thread continuity.
- Required next gate: same-thread resumption after runner repair, native focused positive/negative/restart evidence, affected regression, source-bound independent QA/reducer and coordinator terminal readback. Worker publication is transport evidence only, never acceptance.
- The exact publication response/readback will be retained in the worker's final handoff; it is not predicted in this pre-publication artifact.
