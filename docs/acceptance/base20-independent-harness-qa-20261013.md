# Base20 Independent Cross-Runtime Harness QA

## Verdict

- Recorded date: `2026-10-13` (the scoped tool API provided no clock timestamp).
- Task: `agentfield:v2-base20-qa-harness-cross-runtime-20261013-r1`
- Attempt: `attempt-691f84fc-c448-4cfa-ba5e-bb5676ebdb6d`
- Session: `manual-20261013-base20-independent-qa-agentfiel-ecf9a0eddeb9a15e`
- Result: `RED` / `UNACCEPTED`
- Reason: the candidate receipts are still `CANDIDATE/UNACCEPTED`, the authorized runner rejected both required Python and Go tests, and a source-level Go timeout-policy mismatch requires its owner to repair and test it.

This is an independent scoped QA result, not acceptance of either producer and not a live AgentField control-plane admission.

## Frozen Candidate Evidence

The following source and receipt hashes were read before adding the QA-only test:

| Evidence | SHA-256 | Finding |
| --- | --- | --- |
| `sdk/go/harness/runner.go` | `63312956a23baf05c5d14682af14499d2739c7f580e2e7ff771161e85634b7fb` | Provider selection, retry, cancellation, and nil-result logic are present. |
| `sdk/go/harness/runner_goal_lifecycle_wave_test.go` | `ce8149eda73e7e21365e9bec14e80ee2d304722d0b2996347977421dfa012539` | Covers provider selection, crash recovery, cancellation, and nil provider output. |
| `sdk/python/agentfield/harness/_defaults.py` | `70a13beffb5e6d33b93a1150d8691adc517e0539fc0310c41549b518104e3631` | Explicit provider overrides environment, then `aforge` default. |
| `sdk/python/tests/test_harness_fault_cycle_wave.py` | `12c546f7571e873142cb44e2df4252d6e928dea998468d623f363949c0c8c84d` | Baseline Python provider/recovery/timeout/cancellation candidate test. |
| Go producer receipt | `8b3a505efd4992562e547ffee531b7974a30298510d729d5fe90fcd347028fde` | Producer stated its Go tests were not runnable in the scoped adapter. |
| Python producer receipt | `8f427f1e40eb4a6a0887d02e1d52539ec887808b8785d85cc55e75cd29190973` | Producer stated its Python test was not runnable in the scoped adapter. |

The frozen hashes must be rechecked before any later acceptance. This QA worker changed only the two authorized QA artifacts and did not mutate either candidate source file.

## Independent Regression

`sdk/python/tests/test_base20_independent_harness_qa.py` adds offline disposable-provider checks for one bounded request through provider dispatch into a typed `CorrelatedResult`:

- delayed HTTP 502 is retried once with deterministic backoff, producing one external effect and a typed result;
- malformed schema output resumes the recorded session once and produces one effect;
- provider timeout fails closed without retry or an effect;
- cancellation propagates without a retry;
- a `None` provider result must become a bounded error result;
- a valid typed output paired with a foreign request/thread trace must fail closed before success is returned.

No provider network call, native control-plane admission, production mutation, lease operation, or source-owner lock was performed.

## Actual Validation

| Command | Actual scoped result | Classification |
| --- | --- | --- |
| `python -m pytest -q sdk/python/tests/test_base20_independent_harness_qa.py` | `GO_WAVE_NODE_ERROR:test_file_not_allowed` | `INFRA BLOCKED`, not a pass |
| `go test ./harness -count=1` from `sdk/go` | The authorized adapter rejected `sdk/go/harness/runner_goal_lifecycle_wave_test.go` with `GO_WAVE_NODE_ERROR:test_file_not_allowed`; no Go command runner is available. | `INFRA BLOCKED`, not a pass |

The same denied test adapter was not retried. The focused tests are executable in a Python/Go-capable authorized runtime but have no runtime pass/fail evidence in this attempt.

## Adversarial Review and Required Repair

The Go candidate is not behaviorally equivalent to the Python candidate's fail-closed timeout contract:

- `sdk/go/harness/runner.go` includes `"timeout"` and `"timed out"` in `transientPatterns`, so `executeWithRetry` retries a provider timeout when retries remain.
- The Python producer receipt and this independent regression require a provider timeout to return one failure with no retry and no external effect.

**Foreign-owner repair request:** the owner of `sdk/go/harness/runner.go` must distinguish a provider execution timeout from retryable transport/service failures, preserve retry for 502/service failures, and add a focused Go assertion for one timeout dispatch and zero effect. Re-run `go test ./harness -count=1` from `sdk/go` after the repair. This worker did not edit foreign Go source.

The new Python foreign-trace and nil-result checks are intentionally strict. If a Python-capable runner proves either failure path returns success or raises an uncaught error, request a same-owner repair to the Python runner rather than weakening the QA expectation.

## Native Runtime Gap and Follow-up

Unproven because the permitted test adapter is Node-only:

- Python pytest execution, including the new offline QA test;
- Go package execution through the Go toolchain;
- real provider process behavior, delayed 502 transport, cancellation timing, and persistence across a restart;
- native AgentField control-plane request admission, run/thread receipt correlation, and exactly-once effect enforcement.

Next action: run the two exact commands in authorized Python and Go toolchains against the frozen hashes; if the Go timeout repair changes the source, produce a new hash and repeat the affected QA checks. An independent reviewer, not this worker, decides any acceptance.

## Adversarial Review Findings

- The new test has no runtime outcome because the scoped adapter rejects Python files; every asserted Python behavior remains unproven.
- The Go command itself was not invoked because this worker has no Go command runner; adapter rejection is infrastructure evidence only.
- The frozen hashes establish the candidate inputs read by this worker, not a whole-worktree or post-test immutable snapshot.
- The foreign-trace assertion assumes the runner exposes and validates provider messages; source inspection of the Python runner is outside this task's read scope.
- The nil-result assertion intentionally aligns with the Go repair but may reveal a Python API contract that requires a same-owner source change.
- The delayed-502 double models a returned `RawResult`; it does not cover a provider that raises a transport exception.
- The fake backoff sleep proves retry scheduling arguments but cannot prove real timer cancellation races.
- The local `effects` lists prove the doubles' behavior only and cannot demonstrate an external provider's exactly-once semantics after process loss.
- Request/run/thread values are synthetic test receipts and do not prove native control-plane correlation or durable receipt storage.
- The candidate receipts are `CANDIDATE/UNACCEPTED`, so they cannot substitute for terminal producer verification or independent acceptance.
