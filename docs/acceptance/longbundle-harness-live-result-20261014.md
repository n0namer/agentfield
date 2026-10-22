# Northstar Longbundle Harness Live Result — 2026-10-14

## Authority

- Task: `agentfield:v2-longbundle-harness-live-result-20261014-r1`
- Attempt: `attempt-1c4befdc-0f7b-42b8-8b11-ae35269f4707`
- Session: `manual-20261014-base5-longbundles-northstar-r1--4fd3016b881bfe68`
- Project source identity supplied at launch: `44167dc1050f5bff79abc32fab19c0bb50643093`
- Active BMad source of truth: `docs/acceptance/c4-releasegate-reducer-frontier-20261013.md`
- Worker result: candidate / `UNACCEPTED`

This artifact records bounded implementation evidence only. It does not change native task state, claims, leases, release state, or acceptance.

## Definition of Done

- Bind the live-result test to the C4 runner function signatures, DTO signature, and immutable runner SHA.
- Exercise one successful awaitable repair path without provider replay.
- Exercise malformed output with bounded retries and persistent resume-session propagation.
- Exercise crash/restart with an invalid replacement and fail closed without stale-session replay.
- Run the focused pytest command or preserve an exact infrastructure-blocked receipt.
- Leave independent review and reducer acceptance to their owners.

## B1 — Inspect and map

- The inspected runner is `sdk/python/agentfield/harness/_runner.py`, SHA-256 `06c6eccd3f17a72a802baead62ddfa534b8bfbcc07c76c039b0dbbe485f9cfa8`.
- `_ai_schema_repair(text, schema, options)` is awaitable and returns an optional parsed value.
- `HarnessRunner._handle_schema_with_retry(initial_raw, schema, cwd, start_time, provider, options)` performs stdout/file validation, one focused repair, bounded retries, resume-session propagation, and terminal schema failure.
- The prior C4 reducer artifact is evidence-only and records the C4 gate as blocked; this task adds the missing Python harness evidence without editing the runner.

## B2 — Reproduction and boundary cases

The new test module encodes a disposable provider fixture with explicit call history and finite responses. Its cases cover:

1. Valid async repair from malformed text, proving no provider replay.
2. Persistent malformed output, proving bounded retries, fail-closed schema status, and resume-session propagation.
3. Provider crash followed by invalid replacement, proving restart remains failed and does not reuse the stale session.

Runtime reproduction is not claimed because the authorized pytest runner was unavailable in this environment.

## B3 — Scoped implementation

Added only the authorized test and acceptance files:

- `sdk/python/tests/test_northstar_longbundle_harness_live_result_20261014.py`
  - SHA-256 `40238b8cd7a0a5c58694374c75c23ee43852e0873d7ed7200ae50f09f8bd5ed4`
  - Four focused tests: immutable signatures/SHA, positive repair, bounded resume failure, and crash/restart failure.
- `docs/acceptance/longbundle-harness-live-result-20261014.md`
  - This source-bound receipt.

No production source, prior dirty file, task state, lease, or lock was modified.

## B4 — Validation receipt

Requested command:

`python -m pytest -q -p no:cacheprovider sdk/python/tests/test_northstar_longbundle_harness_live_result_20261014.py`

Scoped runner receipt:

- Status: `BLOCKED`
- Code: `PYTHON_VENV_UNAVAILABLE`
- Engine: `python:pytest`
- Exit code: unavailable (`null`)
- Duration limit: `120000 ms`
- Exact runner error: `A regular local .venv Python executable is required.`
- Runner source identity: `40238b8cd7a0a5c58694374c75c23ee43852e0873d7ed7200ae50f09f8bd5ed4`

The check was not retried because the same environment denial would not provide new evidence. Focused tests and affected regression remain unexecuted; no PASS is claimed.

## B5 — Self-review and handoff

- Changed-file scope is exactly the two manifest-authorized paths.
- The test binds the runner's immutable SHA and exact callable parameter names rather than asserting only behavior through mocks.
- The disposable provider is awaitable, finite, records options, and fails if retries exceed the declared fixture boundary.
- Positive and negative assertions preserve `HarnessResult.is_error`, `parsed`, `failure_type`, session identity, retry count, and replay behavior.
- Independent reviewer/QA must run the focused pytest command from a workspace with the required local `.venv`, then run the affected regression selected by the coordinator.
- The reducer must not upgrade this candidate until fresh runtime evidence and independent review exist.

## Acceptance state

- C4 function/DTO signature and immutable source SHA: implemented in the scoped test; runtime verification pending infrastructure recovery.
- Positive plus two negative/crash-restart assertions: encoded in the scoped test; runtime verification pending infrastructure recovery.
- Focused command exit 0: not established; infrastructure-blocked by unavailable `.venv`.
- B1–B5 evidence: recorded here and in the worker result.
- Independent review and reducer: pending; worker remains `UNACCEPTED`.
