# Releasegate Independent C4 Fan-In QA — 2026-10-13

## Authority and verdict

- Independent task: `agentfield:v2-releasegate-c4-independent-fanin-20261013-r1`
- Attempt: `attempt-1864f66f-9f84-43aa-88f4-c083344f733c`
- Crew session: `manual-20261013-releasegate-c4-fanin-independen-207e6abdf98d56b7`
- Upstream hyperwave: `20261013-native-releasegate-hyperwave-r1-agentfield`
- Reviewed producer task: `agentfield:v2-releasegate-live-cross-sdk-e2e-20261013-r1`
- Reviewed producer attempt: `attempt-19e9b2a8-e7b5-4bfa-8197-7994af097743`
- Reviewed producer session: `manual-20261013-native-releasegate-hyperwave-r1-f3a7e7ac52dd1f26`
- QA verdict: **RED / BLOCKED / UNACCEPTED**.

`RED` applies to contract mismatches visible in checked-in test/runtime evidence. `BLOCKED` applies to execution: the authorized pytest runner stopped before collection because a regular repository `.venv` Python executable was unavailable. This report does not mutate CrewSync or Ruflo Native state and does not accept the upstream task.

## Source and receipt bindings

| Evidence | SHA-256 | Independent observation |
| --- | --- | --- |
| `AGENTS.md` | `591371e0370fe3d95ec809f7e47a3c80ca557b0c726c3f01354732bf422ca75e` | Requires exact source-state receipts and independent release authority. |
| `sdk/python/tests/test_releasegate_live_cross_sdk_20261013.py` | `764cbaf681037405ffd15ddf82a6693ac0820e5b65a0e1ba1eb065478f5bed49` | Matches the producer-recorded test hash. |
| `control-plane/internal/handlers/execute.go` | `be09150579e29e16ce9d96775ef1b161898d829570b9bdd5885c9db5327a25c1` | Current execution DTO and handler entrypoint source inspected. |
| `docs/acceptance/releasegate-live-cross-sdk-20261013.md` | `7789cf969509272b43b171327316bf5f44987d6ae3b6bb6386804e5e96c0ab69` | Producer explicitly records `BLOCKED / UNACCEPTED`; no live request ran. |
| `docs/acceptance/base20-independent-controlplane-qa-20261013.md` | `f7f5751d1d2006c562872700a2850c8a34ea0cbcc5dd5d3a654b60301aede25d` | Independent callback QA records missing run/session/instance binding and no live E2E proof. |
| `docs/acceptance/_evidence/agentfield-runtime-log-snapshot-20261013.json` | `7dbc7d1c661853807373ac44e45281bb1241ed19ed9527fbc5cd389de939fceb` | Physical container-log snapshot inspected; it is operational evidence, not source/build identity. |

The producer receipt also names Python client SHA `9b0d01e7a8a1262caeaf89fd8d82bd94401385d1dba0e4dddb18aec231ba5511` and Go SDK agent SHA `45c90416e5cd7737bcffe2125b8417622ffb4ad729c618d56b739a7d5f9ffbc9`. Those files were outside this verifier's read scope, so these are **receipt-only bindings**, not independently rehashed source claims.

No physical provider completion receipt proves a provider call, model execution, or live cross-SDK completion. The trusted worker footer has `thread_state: THREAD_PENDING` and `producer.thread_id: null`; this report does not infer a provider identity from that state.

## C4 context ledger

| Element | Actual role | Trust boundary / evidence |
| --- | --- | --- |
| Owner client | Python `AgentFieldClient` submits and polls one async execution. | Uses `AGENTFIELD_API_KEY`; test calls private `_submit_execution_sync`, `_await_execution_sync`, and `_prepare_execution_headers`. |
| Foreign owner client | A second valid credential attempts to read the owner's execution. | Expected denial is `401`, `403`, or `404` with no request-correlation leakage. No live result exists. |
| AgentField control plane | Accepts async work, stores execution state, exposes status readback, and receives SDK status callbacks. | Public client/control-plane boundary plus internal agent callback boundary. Handler source proves DTOs and terminal-state fencing, not route-level owner authorization. |
| Registered Go SDK node | Intended target `harness-duo-go.compare` executes a harmless deterministic request. | Test requires registry metadata containing Go in `sdk`, `language`, `runtime`, `framework`, or `implementation`. Runtime snapshot instead identifies `demo-go-agent` traffic. |
| Durable execution store | Persists execution identity, status, result, timestamps, webhook state, and workflow linkage. | Accessed through `ExecutionStore`; restart durability is not exercised by the releasegate test. |
| OpenAPI surface | Intended discovery source for terminal cancel/timeout POST routes. | Runtime snapshot records `GET /openapi.json` returning `404`; the negative checks cannot start against that observed runtime. |
| CrewSync/reducer | Owns final acceptance and terminal state. | This QA artifact is only `CANDIDATE/UNACCEPTED` evidence and performs no administration. |

## C4 container ledger

1. **Python SDK test process**
   - Reads controlled environment configuration.
   - Constructs owner and foreign-owner HTTP clients.
   - Generates `run_id`, `owner_id`, `request_id`, and target `node.reasoner`.
   - Submits exactly one execution only after fixture assertions pass.

2. **AgentField control-plane HTTP service**
   - Expected test entrypoints: `/health`, a registry endpoint, async execution submission through the SDK, `GET /api/v1/executions/{execution_id}`, `/openapi.json`, and discovered terminal mutation routes.
   - Source entrypoints in `execute.go`: `ExecuteAsyncHandler`, `GetExecutionStatusHandler`, `BatchExecutionStatusHandler`, and `UpdateExecutionStatusHandler`.
   - Emits `X-Execution-ID` and `X-Run-ID` on accepted async submission.

3. **Execution persistence and workflow services**
   - `ExecutionStore` provides create/read/batch/update operations, webhook registration, workflow execution persistence, and event-bus access.
   - `handleStatusUpdate` persists terminal payload/state and triggers workflow, webhook, usage, and lifecycle side effects only after a non-no-op update.

4. **Go SDK agent node**
   - Receives the forwarded target invocation and later reports status/result to the callback entrypoint.
   - The current callback body has no protocol version, run ID, session ID, instance ID, lease token, fencing epoch, or callback signature.

5. **Independent release reducer**
   - Must correlate source snapshot, service/build identity, execution IDs, test output, and terminal CrewSync readback before any full release verdict.

## Component and interface ledger

### Python releasegate test components

- `_headers(api_key, **extra) -> dict[str, str]`: emits `accept`, optional `X-API-Key`, and request-specific headers.
- `_request_json(method, path, api_key, body) -> (status, payload)`: raw HTTP helper bounded by configured localhost URL and timeout.
- `_registry_go_node() -> (path, node)`: probes `/api/v1/nodes`, `/api/v1/agents`, `/api/v1/registry`, and `/api/v1/discover`; requires explicit Go metadata.
- `_execution_identity(payload) -> (execution_id, run_id, status)`: requires all three fields.
- `_terminal_mutation_paths(openapi, token) -> list[path]`: discovers POST execution routes containing `cancel` or `timeout` and an execution-ID parameter.
- `_assert_terminal_mutation_denied(...)`: requires `4xx`, then exact persisted-record equality.

### Control-plane DTOs

- `ExecuteRequest`: `input`, optional `context`, optional `webhook`, optional `run_metadata`.
- `AsyncExecuteResponse`: `execution_id`, `run_id`, `workflow_id`, `status`, `target`, `type`, `created_at`, optional `enqueued_at`, `webhook_registered`, optional `webhook_error`.
- `ExecutionStatusResponse`: `execution_id`, `run_id`, optional `agent_node_id`, optional `instance_id`, `status`, optional reason/result/error/details, timestamps/duration, webhook information, and optional approval information.
- `executionStatusUpdateRequest`: required `status`; optional `status_reason`, `result`, `error`, `duration_ms`, `completed_at`, `progress`, `error_status_code`, and `usage`.

### Control-plane behavior contracts

- Async acceptance returns HTTP `202`, `X-Execution-ID`, `X-Run-ID`, and queued or replayed execution metadata.
- Status read requires an `execution_id`, loads exactly one store record, returns `404` if absent, otherwise renders `ExecutionStatusResponse`.
- Callback status is normalized and rejected if unsupported.
- Waiting executions accept only running, cancelled, or failed transitions.
- A repeated identical terminal callback is HTTP `200` no-op.
- A different terminal callback returns HTTP `409`; a late non-terminal callback is rejected without rewriting the record.

## Versioned handshake finding

The only visible version marker is the HTTP namespace `/api/v1`. The test does not assert:

- Python SDK version;
- Go SDK version;
- control-plane build or API schema digest;
- node protocol/capability version;
- callback DTO version;
- compatibility negotiation between Python client, control plane, and Go node.

Registry metadata proving merely that a node is implemented in Go is not a versioned handshake. A full cross-SDK release claim therefore lacks source-bound compatibility evidence even if the live behavior later passes.

## Positive and adversarial checks

The test encodes one positive chain:

1. owner credentials present and distinct;
2. control-plane health is `200`;
3. a registry-confirmed Go target exists;
4. Python SDK submission preserves caller `run_id` and returns `execution_id`;
5. terminal result is successful and retains request correlation;
6. repeated and fresh-client reads are identical.

It also encodes adversarial checks:

1. foreign-owner read returns `401`, `403`, or `404` and does not leak `request_id`;
2. post-terminal cancel is denied and persistence is unchanged;
3. post-terminal timeout is denied and persistence is unchanged;
4. final `execution_id` and `run_id` remain stable.

Authorized execution receipt:

```text
go_wave.run_project_tests({"file":"sdk/python/tests/test_releasegate_live_cross_sdk_20261013.py"})
status: BLOCKED
code: PYTHON_VENV_UNAVAILABLE
engine: python:pytest
source_identity_sha256: 764cbaf681037405ffd15ddf82a6693ac0820e5b65a0e1ba1eb065478f5bed49
exit_code: null
stdout: empty
stderr: A regular local .venv Python executable is required.
```

The runner stopped before collection, imports, fixture validation, localhost access, positive execution, or negative denial checks. There are **zero PASS results** from this attempt. The identical denied call was not repeated.

## Adversarial findings

1. **RED — discovery contract mismatch.** The test probes four registry paths, but the inspected runtime snapshot records `/api/v1/registry`, `/api/v1/agents`, and `/api/v1/discover` returning `404`, while successful discovery is logged at `/api/v1/discovery/capabilities`. The test does not probe that observed endpoint.
2. **RED — OpenAPI dependency mismatch.** The runtime snapshot records `/openapi.json` returning `404`. The cancel/timeout negative checks require that document and will fail before testing terminal denial unless the fixture differs or the correct schema endpoint is used.
3. **RED — fixture target is not bound to runtime evidence.** The test defaults to `harness-duo-go.compare`; the inspected runtime evidence identifies `demo-go-agent` and its reasoners. No receipt binds `harness-duo-go` to the captured container/build.
4. **RED — no versioned cross-SDK handshake.** Language/runtime metadata is not compatibility negotiation and cannot bind the Python SDK, Go SDK, control-plane API, and callback schema versions.
5. **RED — callback identity/fencing contract is incomplete.** The callback DTO lacks run/session/instance/lease/fencing/signature fields; independent control-plane QA already records that missing binding as RED.
6. **RED — owner isolation is asserted but not source-grounded in the inspected handler.** `handleStatus` loads by `execution_id` without owner checks. Route middleware may enforce authorization, but that middleware and route registration were outside this read scope and the live denial never ran.
7. **RED — caller identity headers can be ignored.** Runtime logs state that caller identity headers are ignored without a verified DID and the caller is treated as anonymous. The test depends on API-key authentication establishing the correct owner boundary, but no live receipt proves that mapping.
8. **BLOCKED — producer status was never green.** The producer receipt explicitly says `BLOCKED / UNACCEPTED`; this independent rerun reproduced the same pre-collection infrastructure blocker.
9. **BLOCKED — no physical native E2E receipt.** There is no correlated live execution showing Python submission, Go execution, callback, durable readback, foreign denial, and terminal mutation denial from one source/build snapshot.
10. **LIMITATION — fresh client is not service restart.** Reconstructing `AgentFieldClient` proves client-instance readback only; it does not prove control-plane or database restart durability.
11. **LIMITATION — exact object equality may conflate durability with representation stability.** It is intentionally strict, but any server-added volatile field can fail the test without an identity or persistence defect.
12. **LIMITATION — correlation depends on reasoner echo behavior.** Requiring `request_id` in the terminal JSON proves the configured reasoner returned it, not a generic execution-platform guarantee.
13. **LIMITATION — private Python methods create brittle QA coupling.** The test depends on underscored SDK methods and may break on internal refactoring without a public-contract change.
14. **LIMITATION — broad `4xx` acceptance does not distinguish authorization from route or validation failure.** A `404` for an absent route satisfies the terminal mutation assertion even though no real cancel/timeout authorization guard was exercised; OpenAPI discovery partly mitigates this only when OpenAPI itself is available and current.

## Scoped verdict

**Not `ACCEPT_SCOPED`.** The test design contains useful positive and negative intent, and its current hash matches the producer receipt, but the runnable evidence is absent and the observed runtime contradicts its discovery and OpenAPI prerequisites.

**RED:** contract repair is required before this test can be treated as the intended release gate.

**BLOCKED:** after repair, an authorized Python environment and controlled live fixture are still required. No full GREEN is permitted without separately proven native E2E plus independent terminal reducer/CrewSync readback.

## Sticky-owner repair and affected-only replan

1. **Original releasegate test owner:** update registry discovery to the actual supported discovery contract, preferably one canonical endpoint and typed DTO; do not add more speculative fallback paths. Bind the expected Go node and reasoner to fixture evidence.
2. **Original releasegate test/API documentation owner:** use the control plane's real schema endpoint or a source-owned route contract for cancel/timeout. Require evidence that the discovered route exists in the running build.
3. **Original releasegate test owner:** tighten terminal-denial expectations so a missing route is not accepted as authorization/state-machine proof. Record the expected denial class per route.
4. **Control-plane callback contract owner:** add and verify authenticated/fenced callback identity bound to the durable execution's run, session, active instance/lease, and compatible protocol version. This is production work outside this QA scope.
5. **SDK/control-plane compatibility owner:** expose a minimal versioned handshake or capability/schema digest and assert it before submission.
6. **Infrastructure owner:** provision a regular repository `.venv` with test dependencies and the controlled localhost services, two valid distinct owners, and the intended Go SDK node.
7. **Independent verifier:** rerun only `sdk/python/tests/test_releasegate_live_cross_sdk_20261013.py`, retain full output, source/build identities, execution/run IDs, and denial responses. Then hand the evidence to the independent reducer; do not self-accept.

## Final QA state

- Production source writes: none.
- QA artifact written: `docs/acceptance/releasegate-independent-c4-fanin-20261013.md`.
- Focused test passes: zero.
- Focused test failures: zero product assertions reached.
- Infrastructure blockers: one (`PYTHON_VENV_UNAVAILABLE`).
- Release recommendation: **RED / BLOCKED / UNACCEPTED**.
