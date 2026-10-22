# MVP R11 AgentField Independent QA — 2026-10-14

## Scope and authority

This is an independent C4 QA candidate for the completed R10 producer. It does not modify producer source or the original R10 test, and it does not claim release, deployment, task completion, or product acceptance.

- Project: `agentfield`
- QA task: `agentfield:v2-r11-sourcebound-native-qa-20261014-r1`
- Attempt: `attempt-aca84e02-051f-485c-9d19-b2de965e94c1`
- Session: `manual-20261014-basefive-r11-qa-agentfield-v2-r-83c4a6565af4f5dd`
- Launch Git identity: `44167dc1050f5bff79abc32fab19c0bb50643093`
- Outcome classification: `BLOCKED` by native test infrastructure; product status is not determined.

## Frozen-source receipt

The R10 producer receipt identifies these frozen inputs:

| Artifact | Expected SHA-256 | Current evidence |
|---|---|---|
| `control-plane/internal/handlers/execute.go` | `be09150579e29e16ce9d96775ef1b161898d829570b9bdd5885c9db5327a25c1` | `read_project_file` returned the same SHA-256. |
| `control-plane/internal/handlers/northstar_mvp_r10_disposable_http_test.go` | `e17ed5fa49470165c81835d8d8d7f2f02184157d7c4d9d9b6a22264d030be765` | Prior-test runner returned this exact source identity. |

Therefore the visible producer source and original producer focused-test input are source-bound to the R10 evidence, not stale under the available receipts.

## Independent R11 oracle

The new QA target is `control-plane/internal/handlers/northstar_mvp_r11_idempotency_qa_test.go`. It uses independently named R11 HTTP result types, server, request helper, IDs, and inputs. It invokes the production `asyncIdempotentExecutionStore` at the same caller/target/request-key/body-digest boundary used by the R10 disposable HTTP test; it does not edit or call the R10 test helper.

`TestMVPR11` defines these assertions:

1. **Positive:** first request returns created execution/run identifiers; same caller, target, key, and body returns a typed replay with the same identifiers and one durable create.
2. **Negative — conflict:** a changed body with the same caller, target, and key returns HTTP `409` and cannot add a durable create.
3. **Negative — cancellation:** a pre-cancelled context returns `context.Canceled` before persistence and does not increment the durable create count.
4. **Crash/replay:** after closing and recreating the disposable HTTP server against the same store, the original request is replayed and cannot create a duplicate effect.

The boundary is a controlled `httptest` transport adapter, not a deployed Gin route, database, or process-level crash. It proves only the production adapter contract when the native Go runner can compile it.

## Native validation receipts

Requested R11 focused command: `go test ./internal/handlers -run TestMVPR11 -count=1`.

Authorized invocation: `go_wave.run_project_tests({"file":"control-plane/internal/handlers/northstar_mvp_r11_idempotency_qa_test.go"})`.

- Engine: `docker:go-test`
- Source identity: `b02adc840113cf56ed3f9bc417a5a5281dc570010a636466db20a7f5528abec0`
- Exit code: `1`
- Result: `FAIL` before compilation, classified as `INFRA BLOCKED`
- Exact stderr: `go: cannot find main module, but found .git/config in /workspace` followed by `to create a module there, run: go mod init`

Requested affected producer regression: `go test ./internal/handlers -run TestMVPR10 -count=1`.

Authorized invocation: `go_wave.run_project_tests({"file":"control-plane/internal/handlers/northstar_mvp_r10_disposable_http_test.go"})`.

- Engine: `docker:go-test`
- Source identity: `e17ed5fa49470165c81835d8d8d7f2f02184157d7c4d9d9b6a22264d030be765`
- Exit code: `1`
- Result: `FAIL` before compilation, classified as `INFRA BLOCKED`
- Exact stderr: `go: cannot find main module, but found .git/config in /workspace` followed by `to create a module there, run: go mod init`

The repeated module-root failure affects distinct, authorized test files. It does not constitute product RED and it does not prove either test passes. No identical denied call was retried.

## Adversarial self-review

- The QA test depends on the package's existing `liveIdempotencyCanaryStore` test seam; it adds its own transport oracle and does not alter the R10 fixture or assertions.
- The test preserves all producer files and original tests unchanged; the only new code is the uniquely named R11 QA file.
- The tests are intentionally fail-closed on body mismatch and cancellation; no assertion converts an error into a success response.
- The restart scenario models server/controller recreation over a retained store, not an OS crash or database durability proof.
- Native evidence remains unavailable until the authorized runner executes from the `control-plane` module root or is configured with that module root.
- Independent reducer/reviewer and coordinator terminal readback remain required before any `ACCEPTED_GREEN` decision.

## Next action

A coordinator-owned runner repair or module-root-correct rerun must execute both exact commands from `control-plane`, preserve exit status and source identities, and then route the results to the independent C4 reducer. Until then, this QA candidate remains `UNACCEPTED` and infrastructure-blocked.
