# Release-gate live cross-SDK E2E — 2026-10-13

## Authority and result

- Task: `agentfield:v2-releasegate-live-cross-sdk-e2e-20261013-r1`
- Attempt: `attempt-19e9b2a8-e7b5-4bfa-8197-7994af097743`
- Crew session: `manual-20261013-native-releasegate-hyperwave-r1-f3a7e7ac52dd1f26`
- Authorized additions only:
  - `sdk/python/tests/test_releasegate_live_cross_sdk_20261013.py`
  - this acceptance record
- Worker result: **BLOCKED / UNACCEPTED** because the authorized pytest runner stopped before collection.

No production handler, SDK implementation, migration, database, deployment, external service, task, claim, lease, lock, or Ruflo Native state was modified. The test is designed to create exactly one harmless localhost async execution when an operator supplies the controlled fixture. This attempt did not execute that request because the test engine was unavailable.

## Source-backed gap and bounded design

The existing live native-control-plane test uses raw HTTP against a pre-existing node. It does not require a Python SDK submission object, an explicitly identified Go SDK registry target, two valid owner credentials, or terminal timeout denial. The new test is separate and does not overwrite that producer/QA file.

The bounded test performs this sequence:

1. Require `AGENTFIELD_API_KEY` and a distinct valid `AGENTFIELD_RELEASEGATE_FOREIGN_API_KEY`; absence is an explicit fixture failure rather than simulated owner isolation.
2. Probe only the configured localhost control plane and find the configured target in a registry response.
3. Require explicit registry metadata identifying the target runtime/SDK as Go.
4. Submit exactly one harmless async request through `AgentFieldClient` with caller-supplied `run_id`, actor/caller owner IDs, and an idempotency key.
5. Capture the Python SDK submission `execution_id` and `run_id`, wait for the Go target to complete, and require the persisted response to retain both identifiers and the request correlation.
6. Read the durable execution repeatedly and require stable structure.
7. Construct a fresh Python SDK client and read the same execution again. This is a **client restart/readback** check; it does not claim a control-plane process or database restart.
8. Read the execution with the second valid owner credential and require `401`, `403`, or `404` without correlation leakage.
9. Discover POST cancel and timeout routes from the live OpenAPI document, invoke each only after terminal success, require a strict `4xx` denial, and require the persisted terminal record to remain unchanged.

The request asks the Go reasoner only for a deterministic acknowledgement and explicitly forbids file changes or external calls. All HTTP destinations derive from the localhost fixture URL; the test does not install dependencies, launch processes, migrate storage, or contact a provider.

## Controlled fixture

Required environment:

```text
AGENTFIELD_RELEASEGATE_URL=http://127.0.0.1:8080
AGENTFIELD_API_KEY=<valid owner credential>
AGENTFIELD_RELEASEGATE_FOREIGN_API_KEY=<different valid owner credential>
AGENTFIELD_RELEASEGATE_GO_NODE=harness-duo-go
AGENTFIELD_RELEASEGATE_GO_REASONER=compare
```

Optional bounds:

```text
AGENTFIELD_RELEASEGATE_REQUEST_TIMEOUT=10
AGENTFIELD_RELEASEGATE_POLL_TIMEOUT=30
```

The foreign credential must represent a valid different owner. An arbitrary invalid string is insufficient evidence for owner isolation and is intentionally not synthesized by the test.

## Exact source evidence

- New live test SHA-256: `764cbaf681037405ffd15ddf82a6693ac0820e5b65a0e1ba1eb065478f5bed49`
- Python SDK client inspected SHA-256: `9b0d01e7a8a1262caeaf89fd8d82bd94401385d1dba0e4dddb18aec231ba5511`
  - `AgentFieldClient._submit_execution_sync` returns an internal submission only after both `execution_id` and `run_id` are present.
  - `AgentFieldClient._await_execution_sync` reads `GET /api/v1/executions/{execution_id}` until a terminal status or configured timeout.
- Go SDK agent inspected SHA-256: `45c90416e5cd7737bcffe2125b8417622ffb4ad729c618d56b739a7d5f9ffbc9`
  - the async reasoner response includes `execution_id` and `run_id` from the Go execution context;
  - the Go SDK exposes cancel handling keyed by `execution_id`.
- Existing raw-HTTP live test: content inspected, but no exact SHA-256 receipt was captured in this attempt; no revision claim is made for it.

## Validation receipt

Authorized call:

```text
go_wave.run_project_tests({"file":"sdk/python/tests/test_releasegate_live_cross_sdk_20261013.py"})
```

Exact result:

```text
status: BLOCKED
code: PYTHON_VENV_UNAVAILABLE
engine: python:pytest
source_identity_sha256: 764cbaf681037405ffd15ddf82a6693ac0820e5b65a0e1ba1eb065478f5bed49
exit_code: null
stderr: A regular local .venv Python executable is required.
```

The runner stopped before pytest collection, import, localhost access, or test execution. This is an infrastructure blocker, not PASS and not a product failure. The identical denied call was not repeated.

## Adversarial self-review

- The test does not accept a node name containing `go` as proof by itself; explicit registry SDK/language/runtime metadata is required.
- The test does not replace the second valid owner with a fabricated invalid key.
- Exact readback equality is intentionally strict and may reveal volatile server fields; weakening it would reduce the requested repeat-read stability proof.
- Cancel and timeout are invoked only after terminal success, preventing cancellation of useful in-flight work.
- Missing OpenAPI mutation routes fail closed rather than silently skipping a required negative case.
- A fresh SDK client is not mislabeled as a service restart.
- No claim is made that a live request, Go response, persistence restart, or denial path passed in this attempt.
- Private Python client helpers are used to retain the immediate submission identifiers; this is a QA coupling risk and should be revisited if the SDK exposes a public submission API.
- The test requires the configured reasoner to preserve the harmless `request_id` in persisted output; a reasoner that intentionally omits correlation will fail and requires a fixture adjustment, not a weakened assertion.
- The live gate depends on a regular repository `.venv`; the authorized runner provides no alternative interpreter path.

## Independent verifier handoff

1. Provision a regular local `.venv` containing the project test dependencies.
2. Provision two valid, distinct owner credentials and the controlled localhost Go SDK node.
3. Run the exact test file with pytest through the authorized runner and retain untruncated output.
4. If collection or fixture setup fails, classify the first error before changing assertions.
5. If the foreign credential can read the owner execution, or terminal cancel/timeout returns success or mutates persistence, route the concrete product defect to the owning control-plane task; this QA task is not authorized to modify production files.
6. Re-read the two scoped artifact hashes before independent reduction and CrewSync terminal acceptance.

Product acceptance remains **UNACCEPTED**. The next action is infrastructure repair and one focused live pytest run; no second implementation worker is needed.
