# Native Async Schema Retry Contract — 2026-10-13

## Scope and authority

- Task: `agentfield:v2-luna-high-async-schema-retry-repair-20261013-r1`
- Attempt: `attempt-efb82a98-22a0-4417-b3dc-dae2f469d459`
- Session: `manual-20261013-luna-high-northstar-rootcause-r-98844a014db2cc5f`
- Project source identity supplied at launch: `44167dc1050f5bff79abc32fab19c0bb50643093`
- Write scope used: this artifact and `sdk/python/tests/test_native_async_schema_retry_contract_20261013.py`
- Original production runner and claimed regression test were read-only and unchanged.

## Definition of done

- Reproduce or preserve evidence for the original malformed-schema `TypeError`.
- Classify the failure as fixture/contract misuse versus a genuine runner async defect.
- Add a correct async provider fixture with one positive and at least two adversarial checks.
- Bind the test contract to the runner source SHA and C4 DTO signature.
- Run focused and affected tests when the authorized runner permits them; preserve exact blocked evidence otherwise.

## Root-cause finding

**FACT:** `HarnessRunner._handle_schema_with_retry` awaits `_ai_schema_repair(...)` in `sdk/python/agentfield/harness/_runner.py`.

**FACT:** The read-only regression fixture in `sdk/python/tests/test_luna_high_request_replay_boundary_20261013.py` monkeypatches `_ai_schema_repair` with a synchronous lambda returning `None`. Awaiting that return value raises `TypeError` before the intended retry assertions execute.

**CLASSIFICATION:** Fixture/contract defect, not evidence of a production async schema-retry defect. The production helper is declared `async def` and returns through the awaited path.

**INFRASTRUCTURE EVIDENCE:** The authorized runner rejected the read-only affected regression with `GO_WAVE_NODE_ERROR:test_outside_task_write_scope`; it was not retried because the same test scope remains unauthorized.

## New contract coverage

`sdk/python/tests/test_native_async_schema_retry_contract_20261013.py` contains four focused pytest cases using an actual `async def` repair fixture:

1. **Positive repair:** valid repaired data returns success and does not invoke harness replay.
2. **Malformed output:** async repair returns `None`; schema retries remain bounded and terminate with `FailureType.SCHEMA`.
3. **Terminal timeout:** timeout remains an error, never invokes repair, and cannot become false success.
4. **Crash/restart:** replacement output that remains invalid terminates as schema failure rather than false success.

The test binds the implementation to:

- Runner source SHA-256: `06c6eccd3f17a72a802baead62ddfa534b8bfbcc07c76c039b0dbbe485f9cfa8`
- C4 result DTO signature: `agentfield.harness.result-metrics.v1`

## Validation receipts

| Check | Status | Evidence |
|---|---|---|
| Original affected regression | BLOCKED | `go_wave.run_project_tests({"file":"sdk/python/tests/test_luna_high_request_replay_boundary_20261013.py"})` returned `GO_WAVE_NODE_ERROR:test_outside_task_write_scope`. |
| New focused pytest contract | BLOCKED | `go_wave.run_project_tests({"file":"sdk/python/tests/test_native_async_schema_retry_contract_20261013.py"})` returned `PYTHON_VENV_UNAVAILABLE`; engine `python:pytest`; exact message: `A regular local .venv Python executable is required.`; source identity `eef47ff5c274eaa6cf1bc4fccbf379cdf408407e11b0f94c048f5028f84b7e7a`; exit code `null`. |
| Production source mutation | PASS | No production source file is in write scope; runner SHA is recorded above and the runner was not changed. |
| Scope self-review | PASS | Only the two manifest-authorized files were written. |

## Remaining gate

The implementation candidate is source-backed but not test-green because the authorized Python runner lacks the required local `.venv`. The coordinator or independent verifier must rerun the focused test and the affected regression after the SDK Python virtual environment is restored, then perform independent acceptance. No `ACCEPTED_GREEN` claim is made here.
