# R24 HTTP idempotency and callback fence QA — 2026-10-15

## Authority and current gate

- Task: `agentfield:v2-r24-http-callback-fence-20261015-r2`.
- Attempt: `attempt-2589d5fc-17ea-4249-a5d8-e7250a830935`.
- Session: `manual-20261015-basefive-r24b-20owners-agentfie-dd05c198d5438e10`.
- Target root supplied by launcher: `D:/Users/NIKITA/Documents/DEV/universal-solver-site/agentfield`.
- Git identity supplied by launcher: `44167dc1050f5bff79abc32fab19c0bb50643093`. Live Git status, branch, dirty state and current source fingerprints were not available through the authorized tools; this is not an independently verified current checkout identity.
- Outcome: **BLOCKED / UNACCEPTED**, specifically **INFRA BLOCKED before compilation**, not PRODUCT_RED or CANDIDATE behavioral evidence.
- BMad source of truth remains `docs/acceptance/c4-releasegate-reducer-frontier-20261013.md`. This report does not replace its blocked independent acceptance gate.

FACT: Both exact authorized output paths were absent at inspection. Absence did not prevent creation. Only the two manifest-owned files were written. No production source, existing sibling test, live task, lease, external service or acceptance state was changed.

## Bounded Definition of Done and work sizing

Deliver source-bound actual HTTP replay and callback ownership evidence, positive execution, two trust/conflict negatives, real process crash/restart with durable state, affected native regression, exact execution receipts, self-review and one typed original-task publication. No helper-only response test, controller reconstruction, cancellation or JS substitute counts as durable process-restart proof.

Initial useful-work estimate: investigation 5 minutes, implementation 4, positive/negative/restart testing 6, conditional repair 3, compatibility review 2, evidence/publication 2: 22 minutes, excluding unknown coordination overhead. The actual attempt reached an external runner blocker before compilation; additional identical attempts would add no useful evidence. Actual total elapsed work was not instrumented. One instructions read consumed a reported 300.0341 seconds; the test tool reported 14.3960 seconds. These are tool durations, not total useful engineering time.

## Existing mechanism and call-path evidence

The following are document-backed observations, not fresh source inspection or runtime proof:

| Boundary / owner | Authorized evidence and observation |
| --- | --- |
| HTTP admission: `control-plane/internal/handlers/execute.go` | R14 contract lists `ExecuteAsyncHandler` -> `executionController.handleAsync` -> `readAsyncIdempotencyRequest`; it records verified caller DID, target DID and raw-body digest as the idempotency boundary. |
| Durable binding: `control-plane/internal/storage/execution_idempotency.go` | R14 contract lists `asyncIdempotentExecutionStore` -> `CreateAsyncExecutionIdempotent` -> atomic binding/execution insert. Durable binding already exists according to that contract; it must not be reimplemented by this QA task. |
| Dispatch: handler process-local worker pool | R14 explicitly warns that durable binding does not prove restart dispatch. No persisted queue recovery is inferred. |
| Callback: `UpdateExecutionStatusHandler` -> `executionController.handleStatusUpdate` | `docs/acceptance/northstar-callback-fence-c4-qa-20261013.md` reports no owner/session fields or comparison before mutation, and existing same-terminal no-op / conflicting-terminal 409 behavior. This historical finding has not been revalidated against current source or runtime here. |
| Original production repair owner | Exact original C4 producer task identity is absent from the evidence read for this attempt. Coordinator must identify the existing owner; no replacement ID is invented. Production repair is outside this write scope. |

R14 recorded source fingerprints:

- Handler: `be09150579e29e16ce9d96775ef1b161898d829570b9bdd5885c9db5327a25c1`.
- Storage: `7ff044cdce4e1ec772e3e79a6d4e074e9fece17c2a6857bdf0f4552a1db9cfe2`.

FACT: These are contract-recorded hashes, not current independent SHA receipts. The explicit read scope exposes architecture and acceptance artifacts but not `execute.go`, the storage source, test doubles, callback DTO or Go module files. The exact owned new test is readable/writable; that does not authorize reading sibling production files. No out-of-scope read or test-based source extraction was used to bypass that limitation.

The R14 contract labels router registration unverified and the wire version unversioned in authorized source. The local Gin routes in this test are fixture routes, not assertions about production URLs or methods. Graphify and ArchSteer are not semantic execution evidence: R14 records partial/unverified Graphify coverage and zero Go components in ArchSteer. No current graph freshness, Structurizr execution or ArchSteer Go coverage is claimed.

## Scoped implementation and self-review

Created `control-plane/internal/handlers/r24_http_callback_fence_qa_test.go` with `TestR24HTTPCallbackFence` and two limited negative subtests using actual exported Gin handlers:

1. A valid idempotency key without verified caller context must return HTTP 401 and no execution/run/workflow identifier or replay header.
2. A syntactically incomplete callback JSON body must return HTTP 400 without a replay header.

Nil stores intentionally make an unexpected store dereference fail the test rather than silently using a permissive mock. They do not measure external dispatch effects, durable deduplication or callback owner/session fencing. No `gin.Recovery` middleware converts such a panic into an alleged successful rejection. No global Gin mode mutation or parallel execution was introduced.

HYPOTHESIS: The documented entry-point signatures and early rejection ordering remain compatible with the current source. The runner did not compile the test, so even compile compatibility is unverified. These two input-validation negatives do **not** satisfy the required foreign-owner and stale-session negative scenarios. No positive replay, real crash/restart or retained-state callback test is claimed or simulated.

The native compiler, formatting, lint and complete package regression remain unverified. Existing tests and production behavior were not modified. No fixture or expectation was weakened to conceal RED.

## Exact fresh execution receipt

Authorized invocation, performed once:

```json
{"tool":"go_wave.run_project_tests","arguments":{"file":"control-plane/internal/handlers/r24_http_callback_fence_qa_test.go"}}
```

Returned receipt:

```json
{"status":"FAIL","code":null,"test":"control-plane/internal/handlers/r24_http_callback_fence_qa_test.go","engine":"docker:go-test","source_identity_sha256":"9124a24a8c10c10c4f18691be9aca4233a682f7331f5c5f9e3347994d5a436cb","exit_code":1,"signal":null,"error":null,"duration_limit_ms":120000,"output_truncated":false,"stdout":"","package":"./control-plane/internal/handlers","go_image":"golang:1.24.0-alpine"}
```

Exact stderr:

```text
go: cannot find main module, but found .git/config in /workspace
	to create a module there, run:
	go mod init
```

Classification: **INFRA BLOCKED**. The command failed before compilation because execution was rooted above the nested Go module. No product assertion ran. The engine did not expose its complete internal command; package metadata is not a command receipt. The source identity above is the runner-returned identity, not a separately calculated file hash. Do not run the diagnostic's `go mod init` suggestion: creating a root module is outside scope and is not the requested fix.

The required command is still NOT successfully executed:

```text
cd control-plane && go test ./internal/handlers -run TestR24HTTPCallbackFence -count=1
```

Affected native regression, also NOT executed:

```text
cd control-plane && go test ./internal/handlers -count=1
```

No identical retry or broader package run was attempted after the pre-compilation failure. Both would use the same broken runner; runner configuration is not owned by this task.

Additional tooling receipt: `go_wave.read_project_file({"file":"AGENTS.md","start_line":1,"line_count":220})` returned `tool call error: tool call failed for go_wave/read_project_file`, caused by `timed out awaiting tools/call after 299.9999998s`. It was not retried. Project instructions supplied in the task context were used; a fresh instructions-file read is not claimed.

## Acceptance matrix

| Required condition | Evidence / status |
| --- | --- |
| Actual source SHA, C4 path and owner | PARTIAL: launcher Git and R14 source hashes/call paths cited; current source identity and original producer owner are not independently resolved. |
| Native positive with exact command and exit | BLOCKED: one exact tool invocation and exit 1 recorded; no native positive ran and complete internal command was not returned. |
| Two negatives and crash/restart, no repeated effect / foreign owner | NOT PROVED: only two narrower rejection fixtures were written; neither compiled. No foreign-owner, stale-session, effect-count or process-restart proof exists. |
| Preserve RED without weakening fixtures | Preserved exact infrastructure failure; no product RED or product PASS inferred. |
| Affected native regression | BLOCKED by the same module-root infrastructure failure; no repeated denied/broken invocation. |
| Original-task typed publication and independent QA | Publication attempted once after this report; its exact response/readback belongs to the final handoff. Independent reviewer/reducer remains required. |

## Method exceptions and recovery

BMad Help was loaded first and resolved the launcher target, acceptance-stage C4 gate and existing architecture contract. Quick Dev and its clarify/route step were read for the bounded QA slice. Full resolver scripts, project BMad config inspection, a new spec, interactive workflow, Git commands and subagents were not executed because they are outside this task's allowed interfaces/scope. Their completion is not claimed. Verification Before Completion was loaded and applied to the fresh failing execution receipt. No product bug was reproduced, so no source-level debugging success is asserted.

Next bounded action: coordinator/operator repairs the existing scoped native runner to execute from `control-plane`, grants read-only access to the actual handler/storage/DTO/test-double/module sources and applicable nested instructions, and resumes this original task/thread. Preserve these artifacts, compile and repair only the owned test as needed, then implement and run authenticated replay plus foreign-owner/stale-session and actual durable process-restart scenarios using the existing interfaces. Any necessary production fix belongs to its existing authorized owner, not this QA task. Only after meaningful focused results and affected regression should independent reviewer/reducer QA assess acceptance. Do not accept this partial fixture or a VERIFIED transport response as product GREEN.
