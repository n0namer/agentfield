# Native E2E Gap Closure Post-Reducer — 2026-10-13

## Authority and outcome

- Task: `agentfield:v2-native-e2e-gap-closure-postreducer-20261013-r1`
- Attempt: `attempt-e2c33244-1d50-4890-be85-6cc82cee742d`
- Session: `manual-20261013-native-e2e-gapclosure-hyperwave-730cea9f281bb359`
- Launch Git identity: `44167dc1050f5bff79abc32fab19c0bb50643093`
- Prior reducer: `agentfield:v2-c4-releasegate-independent-reducer-20261013-r6`
- Prior reducer artifact SHA-256: `2bab9006af2f8a4f16713245176bba862b82a1a9b2374fee8bb02cdbec7a4e03`
- Current focused test SHA-256: `a05eebb6944e8c3179a2c0160117349866a563432a886132c53a2c867934c558`
- Gate result: `BLOCKED` / `UNACCEPTED`

The r6 finding is reproduced as an evidence and execution gap, not classified as a proven production defect. A fresh external-process native E2E test now defines the missing positive, negative, timeout/restart, idempotency-owner, and persisted-readback checks. The authorized runner could not start pytest because its required local Python virtual environment is unavailable. No mock, skip, runner error, or document-only result is promoted to product green.

## BMad state and bounded design

- Target source of truth: this repository, not the canonical BMAD-MNNZ control-plane repository.
- Active stage: post-reducer release-gate evidence closure.
- Current gate: executable native ReleaseGate E2E plus source-bound C4 interface/DTO compatibility evidence.
- Smallest compatible change: one new pytest contract and this evidence artifact; production source remains unchanged.
- Quick Dev activation scripts and Git-diff shell inspection were unavailable under the scoped no-shell worker interface. The trusted launch Git identity and exact scoped file receipts are preserved instead.

The test launches a configured native service executable with `shell=False`, restricts service traffic to loopback HTTP, uses a persistent state directory across process replacement, and fails when native bindings are absent. It does not use a paid API, outbound messaging, deployment, or an in-process service fake.

## Focused test receipt

Authorized invocation:

`go_wave.run_project_tests({"file":"sdk/python/tests/test_releasegate_native_ownership_recovery_20261013.py"})`

Runner receipt:

| Field | Value |
|---|---|
| Status | `BLOCKED` |
| Code | `PYTHON_VENV_UNAVAILABLE` |
| Engine | `python:pytest` |
| Exit code | unavailable (`null`) |
| Signal | unavailable (`null`) |
| Duration limit | `120000 ms` |
| Source identity SHA-256 at execution | `06fab9c1e05b2d30339900af2d2b393c868f17ce355ba6d189e402e24b008464` |
| Stdout | empty |
| Stderr | `A regular local .venv Python executable is required.` |
| Provider/model identity | not exposed by the authorized runner receipt; no identity is inferred from routing text |

The test was hardened after that blocked attempt. Its final SHA-256 is `a05eebb6944e8c3179a2c0160117349866a563432a886132c53a2c867934c558`. The identical denied runner call was not retried, per infrastructure-block policy. Therefore the final file has no pytest collection or runtime receipt and remains unverified.

## Native scenarios encoded

| Scenario | Required observation | Fail-closed condition |
|---|---|---|
| Positive ownership | Two submissions with one `request_id` resolve to one `task_id`; terminal state is `succeeded`; marker is returned | Different owners, missing result, wrong marker, non-terminal timeout |
| Negative failure | Deterministic failure reaches `failed`, contains an error, and contains no success result | Failure represented as success or duplicate submission changes ownership |
| Restart recovery | A non-terminal task survives process termination, restart uses the same state directory, and readback reaches `succeeded` with `recovered_after_restart: true` | New owner after restart, lost task, duplicate result identity, missing persisted readback |
| Timeout boundary | Startup and result waits are bounded by explicit environment-configured deadlines | Hanging service or task fails the test rather than being reduced to pass |
| Safety boundary | Base URL must use loopback HTTP and an explicit port | Non-loopback/outbound service target is rejected before service launch |

## C4 interface and DTO compatibility matrix

This matrix is grounded in the consumer contract encoded by `sdk/python/tests/test_releasegate_native_ownership_recovery_20261013.py` at SHA-256 `a05eebb6944e8c3179a2c0160117349866a563432a886132c53a2c867934c558` and in the missing-evidence finding recorded by the r6 reducer. Producer source files were outside this task's authorized read scope, so producer symbol names, versions, and source SHAs are deliberately not invented.

| Boundary | Consumer contract | Compatibility rule | Producer binding | Evidence state |
|---|---|---|---|---|
| Process startup | `AGENTFIELD_NATIVE_E2E_START_ARGV` is a non-empty JSON string array; service receives base URL and persistent state directory | Native executable must start without a shell and reuse the supplied durable state path | Exact producer executable/version absent from authorized evidence | `BLOCKED` |
| Health interface | `GET /health` returns HTTP `200` and `{"status":"ok"}` | Readiness must be explicit before task submission | Producer handler and schema SHA absent | `BLOCKED` |
| Task creation DTO | `POST /v1/releasegate/tasks` accepts `request_id`, `kind`, `behavior`, and `payload.marker` | Repeated `request_id` must resolve to the same ownership identity | Producer DTO/version and idempotency implementation SHA absent | `BLOCKED` |
| Task creation response | HTTP `200`, `201`, or `202`; response includes matching `request_id` and non-empty `task_id`; optional `result_id` must remain stable | Status-code variation is compatible; identity variation is incompatible | Producer response DTO/version absent | `BLOCKED` |
| Task readback DTO | `GET /v1/releasegate/tasks/{task_id}` returns matching `task_id`, status, and terminal result/error fields | Terminal status is one of `succeeded`, `failed`, `timed_out`, `cancelled` | Producer read model/version and persistence SHA absent | `BLOCKED` |
| Success result DTO | `status=succeeded`, `result.marker=request_id` | Result must remain associated with the original owner across duplicate submission | Producer result schema/version absent | `BLOCKED` |
| Failure result DTO | `status=failed`, non-empty `error`, and `result` absent or empty | Failure must never be normalized to success | Producer error schema/version absent | `BLOCKED` |
| Recovery DTO | After process replacement, same `task_id`, `status=succeeded`, `recovered_after_restart=true`, stable optional `result_id` | Persisted ownership and terminal readback must survive restart without duplicate publication | Recovery implementation/schema/version SHA absent | `BLOCKED` |

The matrix provides a precise executable consumer contract, but it cannot establish C4 producer compatibility until the original producer owner supplies exact source symbols, DTO/version identifiers, and immutable SHAs and the test runs against that source.

## Original-owner affected-only replan

No duplicate C4 producer, QA, reducer, or integration task is created.

| Existing owner/task | Evidence state | Next affected-only action |
|---|---|---|
| `agentfield:v2-native-e2e-gap-closure-postreducer-20261013-r1` / `attempt-e2c33244-1d50-4890-be85-6cc82cee742d` / `manual-20261013-native-e2e-gapclosure-hyperwave-730cea9f281bb359` | Current worker; runner infrastructure-blocked | Preserve these artifacts and hand them to independent QA after `.venv` availability is restored |
| `agentfield:v2-c4-releasegate-independent-reducer-20261013-r6` / `attempt-ef7c82ac-fa60-4188-9907-e3ac92939292` / `manual-20261013-c4-reducer-windows-spawnfix-r6--005cb61bd5efb250` | Original reducer remains `BLOCKED` / `UNACCEPTED`; its publication was `INVALID_REPORT` due a non-ISO timestamp | Resume the same reducer owner only after native test and producer receipts exist; repair publication timestamp in that owner's sticky attempt |
| Original C4 ReleaseGate producer task | Exact task/attempt/session IDs absent from authorized evidence | Coordinator resumes the original owner and attaches executable identity plus interface/DTO/source SHAs; do not create a replacement task |
| Original C4 independent QA task | Exact task/attempt/session IDs absent from authorized evidence | Coordinator resumes the original QA owner against the producer's immutable source identity; do not duplicate implementation |
| `agentfield:v2-async-idempotency-independent-qa-sql-fencing-20261013-r3` / `attempt-f0b19c1d-72b9-4137-94e9-73163fa3de5d` / `manual-20261013-five-modules-targeted-repair-qa-485616df1a867054` | Known prior async-idempotency owner, explicitly not C4 evidence | Keep within its original SQL-fencing scope; do not reuse it as a C4 owner |

Universal Solver integration, live paid APIs, outbound messages, deployment, task administration, claims, leases, and acceptance mutation remain outside this task.

## Self-review

- The test contains three independent native scenarios and no mock service.
- Missing runtime configuration causes assertion failure, not skip.
- Process launch uses an argument array and `shell=False`.
- HTTP is restricted to loopback with an explicit port.
- Failed startup terminates the spawned process.
- Restart uses the same resolved state directory and checks stable ownership and persisted readback.
- Runner status is recorded as `BLOCKED`; no pass count or exit code is fabricated.
- The current test changed after the blocked runner attempt and is explicitly marked unexecuted.
- All changed files are inside the exact task write scope.

## Remaining gate

1. Restore a regular repository-local `.venv` Python executable for the authorized pytest runner.
2. Supply `AGENTFIELD_NATIVE_E2E_START_ARGV` for the original native C4 producer build and bind it to the recorded Git/source snapshot.
3. Run the exact focused test and capture collection count, pass/fail/skip counts, exit code, service build identity, and final test SHA.
4. Attach producer interface/DTO symbol names, explicit versions, and source SHAs to the matrix.
5. Obtain independent QA/reviewer evidence and coordinator terminal readback. Until then the only honest state is `BLOCKED` / `UNACCEPTED`.
