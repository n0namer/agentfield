# BaseFive Async Result Contract QA — 2026-10-14

## Authority and scope

- Task: `agentfield:v2-basefive-python-harness-negative-qa-20261014-r4`
- Attempt: `attempt-3101ba44-ebaa-4b27-a5a2-cbb9643899d1`
- Session: `manual-20261014-basefive-northstar-hyper-r4-age-6547d7dcc811e1dc`
- Baseline Git identity supplied at launch: `44167dc1050f5bff79abc32fab19c0bb50643093`
- Authorized write scope: `sdk/python/tests/test_basefive_async_result_contract_20261014.py`, this artifact
- Product source remains unchanged; no production edits, deploy, push, or task-state mutation performed.

## D1 — Source-bound contract map

The focused harness binds the existing Python runner rather than inventing a replacement interface:

- Runner: `sdk/python/agentfield/harness/_runner.py`
- Runner SHA-256: `06c6eccd3f17a72a802baead62ddfa534b8bfbcc07c76c039b0dbbe485f9cfa8`
- DTO signature: `agentfield.harness.result-metrics.v1`
- `_ai_schema_repair` parameters: `text, schema, options`
- `HarnessRunner._handle_schema_with_retry` parameters: `self, initial_raw, schema, cwd, start_time, provider, options`
- Focused test SHA-256 after self-review: `32e8ed8d88a050f144643d84c428b2c425d89cd74a9937bbb903b042e8d15551`

## D2 — Harness assertions

The authorized test contains four source-backed scenarios:

1. Positive async repair returns a parsed result and performs no provider replay.
2. Malformed output fails closed after exactly two bounded schema retries and carries the latest session ID.
3. Terminal timeout remains failed without repair or provider replay, even when schema retries are configured.
4. Crash restart uses the original prompt, does not resume the stale session, and remains failed when the replacement is invalid.

The test also verifies the immutable runner SHA and the C4 DTO/interface signature.

## D3 — Independent Python harness QA

- Test file: `sdk/python/tests/test_basefive_async_result_contract_20261014.py`
- Requested command: `python -m pytest -q sdk/python/tests/test_basefive_async_result_contract_20261014.py`
- Native runner result: `BLOCKED`
- Runner receipt: `PYTHON_VENV_UNAVAILABLE`
- Exact runner message: `A regular local .venv Python executable is required.`
- Exit code: unavailable (`null`)
- Source identity reported by runner: `22d1b1b732b2904fbbf191c4de00d15fbce10a2afaf3453189c68ad838baeef0`
- Classification: `BLOCKED_INFRA`; no PASS is claimed and the denied command was not retried.

## D4 — Focused validation and repair

- The first static self-review identified an invalid synchronous mock for an awaited repair coroutine in the restart scenario.
- Repaired only the authorized test fixture by replacing it with an async no-op.
- Final scoped test SHA after repair: `32e8ed8d88a050f144643d84c428b2c425d89cd74a9937bbb903b042e8d15551`.
- Focused pytest execution remains infrastructure-blocked; no affected regression command was run because the same Python venv barrier applies and the task authorizes only the focused validation command.

## D5 — Independent-review handoff

- Candidate status: `UNACCEPTED`.
- Acceptance authority remains with the independent reviewer/reducer and coordinator terminal readback.
- Remaining blocker: provide a regular local `.venv` Python executable, then rerun the exact focused pytest command.
- No known out-of-scope source change or product regression is inferred from the blocked runner.

## Self-review checklist

- Changed files are exactly within the two-file write scope.
- Assertions use the existing runner and result DTO contract.
- Positive, malformed, terminal, and restart boundaries are explicit.
- No credentials, generated artifacts, production edits, or acceptance claims are included.
- Test execution is recorded as infrastructure-blocked rather than green.
