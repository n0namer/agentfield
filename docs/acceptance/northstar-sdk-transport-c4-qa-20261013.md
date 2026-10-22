# Northstar SDK Transport C4 QA — 2026-10-13

## Authority and state

- Task: `agentfield:v2-c4-native-acceptance-sdk-transport-20261013-r1`
- Attempt: `attempt-2f25ccfc-677a-43a7-955e-681ccc9280de`
- Session: `manual-20261013-c4-native-providerbounded-refil-05775d7d61779870`
- Git identity supplied at launch: `44167dc1050f5bff79abc32fab19c0bb50643093`
- Active BMad source of truth: `docs/acceptance/c4-releasegate-reducer-frontier-20261013.md`
- Gate: C4 native task-to-result transport evidence
- Worker state: `BLOCKED` / `UNACCEPTED`

This artifact records independent scoped QA evidence only. It does not mutate production source, native tasks, CrewSync claims, leases, release state, or acceptance state.

## Source-backed boundary

The authorized native Python source is `sdk/python/agentfield/harness/_runner.py`, SHA-256 `06c6eccd3f17a72a802baead62ddfa534b8bfbcc07c76c039b0dbbe485f9cfa8`.

The focused contract suite binds itself to that SHA and exercises these existing boundaries without changing production code:

| Boundary | Existing signature or contract | QA invariant |
|---|---|---|
| Option resolution | `_resolve_options(config, overrides) -> Dict[str, Any]` | Whitelisted non-null config fields cross the boundary; non-null overrides win; `None` does not erase configured values. |
| Metric reduction | `_accumulate_metrics(all_raws)` | Cost and token counts accumulate, `None` token values become zero, newest non-empty session wins, and the first non-empty model remains stable. |
| Result DTO projection | `_token_result_kwargs(tokens)` | Version `agentfield.harness.result-metrics.v1` has wire fields `input_tokens`, `output_tokens`, `cache_read_tokens`, `cache_creation_tokens`, `total_tokens`, and `model`; `total_tokens` is input plus output only. |
| Native execution | `HarnessRunner.run(...)` | A deterministic in-process provider traverses the real runner path and preserves result, messages, session, cost, turns, and token metrics. It is not described as a live paid-provider test. |
| Provider retry | `HarnessRunner._execute_with_retry(provider, prompt, options)` | Timeout text is intentionally non-transient; only enumerated transient transport/server failures retry. |
| Structured result | `HarnessRunner._handle_schema_with_retry(...)` | Timeout and exhausted malformed output fail closed with `parsed=None` and an error result; crash restarts from the original prompt without a stale session; non-crash correction resumes the last native session. |

## Focused scenarios

The new test file is `sdk/python/tests/test_northstar_cross_sdk_contract_20261013.py`, final SHA-256 `9a79aa0344beafaf1d73833a81a340290cd2cc5130f6a55148a264e9878d1fec`.

1. Positive native path: real `HarnessRunner.run` control flow with a deterministic local provider object, asserting result and token invariants. This is deterministic contract evidence, not a mock presented as a live provider call.
2. Option and metric boundary: config/override precedence, foreign config exclusion, nullable token normalization, session selection, cost accumulation, and the versioned DTO key set.
3. Timeout negative: a structured timeout must terminate without provider retry and return `is_error=True`, `parsed=None`, the timeout failure type, contextual output-path error text, and preserved token metrics.
4. Invalid-response negative: malformed structured responses exhaust exactly the configured schema retries and terminate as `FailureType.SCHEMA`, never as success.
5. Crash/restart negative: crash recovery reuses the original structured task but must not send the stale pre-crash `resume_session_id`.
6. Stale-session oracle: a non-crash correction resumes the latest session ID, distinguishing resumable schema correction from crash restart.

## Cross-runtime DTO compatibility oracle

The Python side is source-bound to `agentfield.harness.result-metrics.v1` and the six wire fields above. Compatibility is defined as follows:

- Go and Python must serialize the same six metric keys with integer token counts, nullable/omitted model semantics agreed by both SDKs, and `total_tokens = input_tokens + output_tokens`.
- Missing provider token fields normalize to zero before transport; they must not produce arithmetic failure or a success result with an invalid metric object.
- A later runtime may add fields only compatibly; it must not rename or reinterpret the v1 fields without a new DTO version.
- Retry state is not interchangeable: crash restart omits the stale session; non-crash schema correction may carry the last session.
- Timeout and invalid structured output are errors with no parsed DTO. Neither runtime may reduce them to a successful result.

The repository contains candidate Go harness boundaries at `sdk/go/harness/result.go`, `sdk/go/harness/runner.go`, and `sdk/go/harness/tokens.go`, but their contents were outside this attempt's exact read scope. Therefore no Go source SHA or field-tag claim is fabricated. The original Go SDK/C4 owner must bind those files and their exact SHAs to this v1 oracle, or repair the original production task if the actual Go DTO differs.

## Runner receipt

Requested focused validation:

`go_wave.run_project_tests({"file":"sdk/python/tests/test_northstar_cross_sdk_contract_20261013.py"})`

Actual receipt:

- Status: `BLOCKED`
- Code: `PYTHON_VENV_UNAVAILABLE`
- Engine: `python:pytest`
- Runner source identity: `40825ce36f666bc5fa16878fce23224cabe8d832ecfbb69d17305b1c027a8484`
- Exit code: unavailable (`null`)
- Stderr: `A regular local .venv Python executable is required.`
- Duration limit: 120000 ms

The receipt predates the final source-binding assertion; the final test SHA is recorded above. The identical runner call was not repeated after the final evidence-only assertion because the blocker is an unavailable required `.venv`, not test content. No test is reported as passing.

A runnable repository-local command after infrastructure repair is:

`D:\Users\NIKITA\Documents\DEV\universal-solver-site\agentfield\.venv\Scripts\python.exe -m pytest sdk/python/tests/test_northstar_cross_sdk_contract_20261013.py -q`

If the project intentionally keeps the Python environment under `sdk/python`, the operator must use that repository-approved interpreter instead; this worker did not guess or create an environment.

## Gate classification

### PASS

- Source inspection proved the Python C4 function boundaries and exact source SHA.
- The focused test artifact covers one deterministic native positive path and four adversarial boundary/recovery paths.
- Production source remained unchanged and all changed files stay inside the authorized write scope.

These are artifact and inspection passes, not runtime test passes.

### FAIL

- No product assertion is classified as failed because pytest never started.
- Cross-runtime Go compatibility is not proven because exact Go source reads and SHAs were outside this attempt's read scope.

### SKIP

- Affected Python regression was not run because the same missing `.venv` blocks the focused test before collection.
- Paid/live provider execution was intentionally not attempted and is not required to prove deterministic runner control flow.

### BLOCKED

- Focused and affected regression execution are infrastructure-blocked by `PYTHON_VENV_UNAVAILABLE`; exit code and test counts do not exist.
- The C4 gate remains blocked until the focused suite and affected harness regression execute under the repository's regular `.venv`.
- Full Go/Python DTO compatibility remains blocked until the original Go SDK/C4 owner supplies exact source SHAs and verifies its actual JSON tags/types against `agentfield.harness.result-metrics.v1`.

## Root-cause-only repair route

1. Infrastructure owner restores or exposes the repository's regular `.venv` Python executable; do not weaken the runner or mark an unrun test PASS.
2. Resume this exact task/attempt thread for the focused pytest and affected harness regression receipt.
3. If a Python assertion fails, repair only the new test unless evidence proves a production defect; `_runner.py` is read-only in this task.
4. If actual Go DTO fields or retry semantics differ, route the defect to the existing original Go SDK/C4 production owner through coordinator-owned CrewSync state. Do not create a duplicate task or modify live task state from this worker.
5. Independent reviewer and coordinator perform final acceptance and authoritative terminal readback.

## Self-review

- No live-provider claim is made for deterministic provider tests.
- Timeout, malformed response, crash restart, and stale-session semantics are independently asserted.
- Exact Python source and final test SHAs are recorded.
- The blocked runner receipt is preserved verbatim and is not converted to PASS.
- No production code, Go code, dependency, lockfile, task state, or acceptance state was changed.
