# Northstar Callback Fence C4 Independent QA — 2026-10-13

## Authority

- Task: `agentfield:v2-c4-native-acceptance-callback-fence-20261013-r1`
- Attempt: `attempt-e6e90a59-593d-45aa-8603-1323eeb47bde`
- Session: `manual-20261013-c4-acceptance-provider-retryfro-44a671bf85def856`
- Git identity supplied at launch: `44167dc1050f5bff79abc32fab19c0bb50643093`
- Acceptance level: `UNACCEPTED`
- Production source is read-only for this task. No live task, lease, service, database, deployment, or CrewSync state was modified.

## Current C4 gate

- **FACT:** The active BMad source of truth is `docs/acceptance/c4-releasegate-reducer-frontier-20261013.md` at SHA-256 `2bab9006af2f8a4f16713245176bba862b82a1a9b2374fee8bb02cdbec7a4e03`.
- **FACT:** That reducer requires source-bound C4 interface evidence plus native positive, failure, and restart receipts before independent acceptance.
- **FACT:** This task adds the scoped native handler QA test and this evidence report. It does not replace the reducer or claim `ACCEPTED_GREEN`.

## Function and DTO boundary

- Native entry point: `UpdateExecutionStatusHandler(store, payloads, webhooks, timeout)`.
- Native implementation path: `UpdateExecutionStatusHandler` constructs `executionController`, then routes to `executionController.handleStatusUpdate`.
- Request DTO: `executionStatusUpdateRequest` in `control-plane/internal/handlers/execute.go`.
- Stored ownership/session fields: `types.Execution.AgentNodeID` and `types.Execution.InstanceID`, exposed by `ExecutionStatusResponse` as `agent_node_id` and `instance_id`.
- Production source SHA-256: `control-plane/internal/handlers/execute.go` = `be09150579e29e16ce9d96775ef1b161898d829570b9bdd5885c9db5327a25c1`.
- Test source SHA-256: `control-plane/internal/handlers/northstar_callback_fencing_20261013_test.go` = `ba2dab598b9bf2d6d03e42b14b0691fe50ca8fda387aabf2a4d798c920232237`.

## Source-backed root cause

- **FACT:** `executionStatusUpdateRequest` contains status/result/error/timing/progress/usage fields but no owner or session identity fields.
- **FACT:** `handleStatusUpdate` binds the request, normalizes status, and calls `UpdateExecutionRecord`, but it does not compare callback owner/session identity with `current.AgentNodeID` and `current.InstanceID` before mutation.
- **FACT:** The existing terminal-state branch accepts a duplicate delivery of the same terminal status as a no-op and rejects a different terminal status with HTTP `409 Conflict`.
- **HYPOTHESIS TO EXECUTE:** Because Gin JSON binding ignores unrecognized fields by default and the handler contains no owner/session check, stale-owner and mismatched-session callbacks reach the record mutation path. The focused tests encode the required rejection and unchanged-record invariant, but the runner failed before compilation, so this hypothesis has no runtime result in this attempt.

## Scoped QA scenarios

The new test invokes the actual Gin route backed by `UpdateExecutionStatusHandler`; it uses the repository's existing execution-store test double and does not label the route as a live service.

1. **Positive native path:** matching `agent_node_id` and `instance_id`, terminal success, result receipt, duration, response identity, and stored terminal record.
2. **Stale owner negative:** foreign `agent_node_id` must return HTTP `409` and leave status, result, and completion timestamp unchanged.
3. **Mismatched session negative:** stale `instance_id` from before restart must return HTTP `409` and leave the record unchanged.
4. **Restart/retry negative:** reconstruct the handler over retained execution state, redeliver the same terminal callback, then send a conflicting terminal callback. The first result and completion timestamp must remain authoritative and the conflict must return HTTP `409`.

The restart scenario is a controller reconstruction over retained test-store state. It checks handler/statestore invariants but is not evidence of a real database-backed process crash or live-service restart.

## Test receipt

Authorized invocation:

`go_wave.run_project_tests({"file":"control-plane/internal/handlers/northstar_callback_fencing_20261013_test.go"})`

Runner-reported engine/package:

`docker:go-test` / `./control-plane/internal/handlers`

Result:

- Status: `FAIL`
- Exit code: `1`
- Source identity SHA-256: `ba2dab598b9bf2d6d03e42b14b0691fe50ca8fda387aabf2a4d798c920232237`
- Compilation: not reached
- Exact stderr: `go: cannot find main module, but found .git/config in /workspace` followed by `to create a module there, run: go mod init`

Classification: **INFRA BLOCKED**. The runner started at `/workspace` instead of the Go module root under `control-plane`. This receipt is not a product test failure and is not a PASS.

Affected regression: **BLOCKED/SKIP**. No second Go test was launched because it would repeat the same pre-compilation module-root failure. No build, lint, database, live service, or deployment check is represented as passing.

Runnable module-root command for the repaired runner/operator:

`go test ./internal/handlers -run '^TestNorthstarCallbackFence_' -count=1`

Required working directory: `control-plane`.

Affected regression command after the focused target executes:

`go test ./internal/handlers -count=1`

Required working directory: `control-plane`.

## Edge-case review

Handled by the scoped test:

- matching current owner/session positive callback;
- stale owner with current session;
- current owner with stale session;
- duplicate same-terminal delivery after handler reconstruction;
- conflicting terminal delivery after handler reconstruction;
- preservation of the first result and completion timestamp.

Not proved in this attempt:

- missing or empty owner/session identity;
- atomic owner/session validation in a production storage transaction;
- database-backed process crash and recovery;
- concurrent callbacks racing before either becomes terminal;
- compatibility with the current agent SDK callback producer.

These gaps must not be converted to PASS without source-bound execution evidence.

## Root-cause-only repair route

The production repair belongs at the C4 callback contract boundary in `control-plane/internal/handlers/execute.go` and any corresponding producer DTO owned by the original C4 producer task. The minimal compatible repair must:

1. define authoritative callback owner/session identity in the request or authenticated route context;
2. compare that identity against the current execution's `AgentNodeID` and `InstanceID` before any mutation or side effect;
3. reject stale/foreign callbacks deterministically without rewriting the record;
4. preserve the existing same-terminal idempotent no-op and conflicting-terminal `409` behavior;
5. update the SDK/producer contract and compatibility evidence if the callback DTO changes.

The exact original C4 producer task ID is absent from the authorized read scope. The coordinator must resume that existing owner and must not create a duplicate producer task. This QA task has no authority to edit `execute.go`, the SDK producer, live CrewSync tasks, or terminal state.

## BMad and Superpowers method receipt

- BMad Help resolved `agentfield` as the target project and the C4 reducer as the current gate.
- BMad Quick Dev was used for the bounded test/report implementation. Its full interactive planning flow could not be executed because task authority forbids a competing plan and permits only the two scoped output files.
- Systematic Debugging traced the missing fence from callback DTO through `handleStatusUpdate` to record mutation before proposing the repair boundary.
- Edge Case Hunter drove stale owner, stale session, duplicate terminal, conflicting terminal, and reconstruction/retry cases.
- Verification Before Completion was applied: the actual runner output is recorded as infrastructure-blocked, and no unrun check is called PASS.

## Changed files

- `control-plane/internal/handlers/northstar_callback_fencing_20261013_test.go`
- `docs/acceptance/northstar-callback-fence-c4-qa-20261013.md`

## QA verdict

- Scoped evidence artifact: `CANDIDATE`
- Product behavior: `UNACCEPTED`
- Focused runtime verification: `INFRA BLOCKED` before compilation
- Source review: owner/session callback fencing is absent from the authorized production source
- Independent reviewer: required before any GREEN decision
