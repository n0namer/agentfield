# Base20 Python Harness Failure Recovery

## State

- Outcome: `CANDIDATE` / `UNACCEPTED`
- Target: `agentfield`
- Stage: implementation-stage bounded reliability exercise
- Source of truth: repository instructions, existing Python harness runner, and this acceptance receipt
- Native live control-plane proof: not established

## Implemented scope

`sdk/python/tests/test_harness_fault_cycle_wave.py` adds a disposable provider matrix covering:

- explicit, environment, and default provider-selection precedence;
- unknown-provider rejection at the provider-factory boundary;
- typed structured output with request correlation and one durable receipt;
- malformed schema output followed by same-session recovery;
- provider crash with replay recovery without session resumption;
- transient network loss with bounded retry and one effect;
- timeout fail-closed behavior without retry or effect;
- cancellation propagation without an effect.

`agentfield/harness/_defaults.py` was inspected and left unchanged: explicit provider, environment provider, and `aforge` default already implement the declared precedence. No independent source defect justified a production change.

## Evidence

- Existing runner source shows transient retry is restricted to known network/service patterns, excludes provider timeout, resumes non-crash schema retries with the last session, and replays crash recovery without `resume_session_id`.
- Existing affected test `sdk/python/tests/test_wave_full_cycle.py` already covers partial resume, transport retry, and timeout fail-closed behavior.
- New focused command requested: `python -m pytest -q sdk/python/tests/test_harness_fault_cycle_wave.py`.
- Actual scoped runner result: `GO_WAVE_NODE_ERROR:test_file_not_allowed`.
- The available `go_wave` test adapter accepts Node/Vitest/tsx test files, not Python pytest files; no identical retry was made.

## Gap and next action

The new Python tests are not runtime-verified in this worker because the authorized test tool rejected the `.py` target. Run the exact pytest command in a Python-capable authorized environment, then run the affected Python regression suite and strict negative injection/restart scenario. Independent QA/reducer must review the candidate before product acceptance.
