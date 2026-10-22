# Luna High Harness Request Replay Boundary — 2026-10-13

## Gate status

- Outcome: `CANDIDATE` / `UNACCEPTED`
- Scope: deterministic Python AgentField harness regression only
- Production source changed: none
- Live remote E2E claim: none
- Acceptance owner: independent QA/reviewer and coordinator

## Source-of-truth evidence

- Project: `agentfield`
- Git identity supplied at launch: `44167dc1050f5bff79abc32fab19c0bb50643093`
- Runner source: `sdk/python/agentfield/harness/_runner.py`
- Runner source SHA-256: `06c6eccd3f17a72a802baead62ddfa534b8bfbcc07c76c039b0dbbe485f9cfa8`
- New test: `sdk/python/tests/test_luna_high_request_replay_boundary_20261013.py`
- New test SHA-256: `697d029a3058e19472f8c0840e636e1efbecd141f4d73780cf707c452513f5ed`
- Existing adjacent contract test: `sdk/python/tests/test_northstar_cross_sdk_contract_20261013.py`
- Existing contract test source SHA-256 at inspection: `9a79aa0344beafaf1d73833a81a340290cd2cc5130f6a55148a264e9878d1fec`

## C4 boundary and dependency trace

The regression binds the actual runner boundary rather than a replacement implementation:

1. `HarnessRunner.run(prompt, ..., **kwargs)` resolves caller/config options through `_resolve_options`, resolves the provider, builds a provider, and returns `HarnessResult` for an unstructured request.
2. `_resolve_options(config, overrides)` copies only the declared harness configuration fields, then applies non-`None` explicit overrides. The test verifies an explicit replay session override survives while unrelated config does not cross the boundary.
3. `_execute_with_retry(provider, prompt, options)` retries only errors matching `_is_transient`; terminal timeout behavior is not treated as transient by the source contract.
4. `_handle_schema_with_retry(initial_raw, schema, cwd, start_time, provider, options)` validates output, attempts bounded schema repair/retry, preserves the original prompt, resumes non-crash retries from the latest session, and discards stale sessions after a crash.
5. `_token_result_kwargs(tokens)` produces the accepted result DTO metrics wire fields: `input_tokens`, `output_tokens`, `cache_read_tokens`, `cache_creation_tokens`, `total_tokens`, and `model`. The version binding used by the adjacent focused contract is `agentfield.harness.result-metrics.v1`.

Relevant source dependencies are imported directly by `_runner.py`: provider resolution/factory, `RawResult`/`HarnessResult`/`FailureType`, and schema output parsing/diagnostics. The new test mocks only the provider and schema parser edges; it executes real option resolution, runner orchestration, retry classification, session propagation, and DTO construction.

## Regression coverage

| Case | Expected invariant | Evidence in test |
|---|---|---|
| Positive request → result | Deterministic provider result remains non-error and metrics are preserved in `HarnessResult`. | `test_request_to_result_positive_binds_source_and_dto_boundary` |
| Option boundary | Explicit replay override wins; unrelated config and `None` override do not leak. | `test_option_resolution_rejects_config_leak_but_preserves_explicit_replay_override` |
| Malformed result | Invalid structured output exhausts bounded schema retries and returns `FailureType.SCHEMA` with no parsed result. | `test_malformed_result_exhausts_schema_retries_and_fails_closed` |
| Terminal timeout | Timeout fails closed and does not invoke a replay provider. | `test_terminal_timeout_is_not_replayed_or_retried` |
| Crash restart | Crash retry uses the original request and does not reuse the stale session. | `test_restart_discards_stale_session_and_reuses_original_request` |
| Non-crash resume | One retry resumes the latest native session exactly once. | `test_non_crash_replay_resumes_latest_session_once` |

These are deterministic local tests. They do not claim provider credentials, network reachability, remote task creation, durable external publication, or live native E2E behavior.

## Validation receipts

### New focused test

Command requested through scoped runner:

`go_wave.run_project_tests({"file":"sdk/python/tests/test_luna_high_request_replay_boundary_20261013.py"})`

Result: `BLOCKED` / `PYTHON_VENV_UNAVAILABLE` / `exit_code=null`.

Exact runner stderr: `A regular local .venv Python executable is required.`

The runner reported source identity SHA-256 `697d029a3058e19472f8c0840e636e1efbecd141f4d73780cf707c452513f5ed` for the new test. No PASS/FAIL claim is made.

### Existing adjacent contract test

Attempted scoped validation of `sdk/python/tests/test_northstar_cross_sdk_contract_20261013.py` returned `GO_WAVE_NODE_ERROR:test_outside_task_write_scope`. This is an authorization/test-scope denial, not a product result, and was not retried.

### Host rerun recovery command

When the AgentField SDK virtual environment is available, run from the repository root:

`python -m pytest sdk/python/tests/test_luna_high_request_replay_boundary_20261013.py`

Then, if authorized by the coordinator’s test scope, run:

`python -m pytest sdk/python/tests/test_northstar_cross_sdk_contract_20261013.py`

Those host commands remain unexecuted in this attempt.

## Self-review

- Changed files are limited to the two authorized write-scope paths.
- No production source, existing worker test, task state, lease, lock, or remote effect was modified.
- The test uses deterministic provider doubles and does not invent a live E2E result.
- At least three fail-closed/recovery boundaries are covered: malformed output, timeout, crash restart, and non-crash resume.
- Focused execution is infrastructure-blocked; the candidate remains `UNACCEPTED` pending an available scoped Python venv and independent QA.
