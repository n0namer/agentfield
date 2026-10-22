# Base20 Runtime Log Forensics — 2026-10-13

## Scope and evidence

- **Target:** `agentfield`, control-plane runtime and sanitized demo Go-agent logs.
- **Snapshot:** `docs/acceptance/_evidence/agentfield-runtime-log-snapshot-20261013.json`, captured `2026-10-13T12:04:22.976Z`; no raw customer payloads or credentials are used.
- **Source cross-check:** `control-plane/internal/handlers/execute.go` and `control-plane/internal/storage/execution_records.go` at the read-only source revisions supplied to this lane.
- **Regression artifact:** `control-plane/internal/handlers/runtime_log_observability_wave_test.go`.

## Reconstructed state

- The snapshot contains two structured async acknowledgement lines for `exec_20261013_110654_nynu85bf` at `2026-10-13T11:06:54Z`: `agent acknowledged async execution` followed by `agent accepted execution for async processing`.
- The deterministic fixture correlates both lines to one durable readback: `run_20261013_demo_01`, node `demo-go-agent`, status `queued`. This is a duplicate observation, not evidence of two durable executions.
- The control-plane snapshot repeatedly reports `prepare stale execution update` and `prepare stale workflow execution update` with PostgreSQL `SQLSTATE 42601` at five-minute intervals from `10:38:28Z` through `12:03:28Z`.
- The same window contains canceled heartbeat/health/lifecycle transactions and a `500` status callback for `/api/v1/nodes/demo-go-agent/status` with approximately five-second latency.
- Permission middleware warnings show caller identity headers present without a verified DID; source behavior intentionally treats that caller as anonymous.
- Demo-agent stdout/stderr noise is high-volume and unstructured (`unknown` level), so it is not a reliable lifecycle signal without a correlation ID.

## Source correlation and classification

1. **Idempotency / duplicate handling — product path is present.** `execute.go` reads the `Idempotency-Key`, hashes the request body, performs durable lookup before admission, returns a replay response for an existing matching record, and returns `409` for a body conflict. The handler persists `execution_id` and `run_id` through the execution plan. The sanitized runtime snapshot does not expose the raw header value; the duplicate is therefore represented by repeated acknowledgement lines and checked against one durable row.
2. **Stale timeout SQL — likely environment/image drift, with a product-impacting symptom.** Current `execution_records.go` builds dialect-aware timestamp expressions (`julianday(...)` for SQLite and plain timestamps for Postgres) and prepares valid stale execution/workflow updates. The runtime error is a PostgreSQL syntax error at statement preparation, which is inconsistent with the checked-in source path. The smallest justified repair is to verify the running control-plane image/build and database dialect/config first; only then patch SQL if the deployed binary reproduces the checked-in statement failure.
3. **Canceled heartbeat/lifecycle writes — environmental or shutdown/lease race until reproduced.** `context canceled` plus a five-second `PATCH` strongly indicates request or transaction cancellation during lease/status handling. The evidence does not prove a durable product defect by itself. Reproduce with a controlled node status update while the database/container is healthy before changing retry or timeout behavior.
4. **Anonymous caller warning — expected security behavior, not a failure.** Header presence without verified DID is explicitly ignored. Treat the warning as an authentication/fixture setup signal; do not weaken the middleware to make the demo appear authenticated.
5. **Demo-agent noise — observability quality issue.** Unstructured logs are privacy-safe but not state-correlatable. Add structured execution/run identifiers at the agent logging source in a separate authorized change; this lane intentionally does not add a monitoring daemon or modify the owned handler.

## Ranked minimal repair proposals

1. Verify the deployed control-plane image digest, source revision, SQL dialect, and migration state against the checked-in stale-reaper implementation; rerun the stale reaper with one sanitized row.
2. Reproduce the canceled status transaction with a bounded request/DB timeout matrix and persist the exact `execution_id`/`run_id` before deciding whether the fix belongs in lease handling or infrastructure.
3. Preserve the current idempotency contract and add an integration check that two identical requests yield one durable execution row and a replay response; a same-key/different-body request must remain `409`.
4. Improve demo-agent structured logging in its owning lane, retaining redaction and correlation IDs.

## Regression checks

- Positive: decode nested structured log events, correlate both duplicate acknowledgements to one durable execution/run record, and classify stale execution, stale workflow, lifecycle cancellation, and anonymous-caller events.
- Negative: reject an acknowledgement with no durable execution readback, reject malformed structured JSON, and refuse fixtures containing authorization/token/secret markers.
- No production handler, storage, idempotency, native-worker, or lease code is modified by this forensic lane.

## Gate status

- **Candidate evidence:** the regression test and this diagnostic are ready for independent QA/reducer review.
- **Acceptance:** `UNACCEPTED`; independent verification remains required.
- **Unverified here:** live runtime replay, deployed image digest, database readback from the captured environment, and the requested package-level `go test ./internal/handlers -count=1` command. The scoped runner returned `GO_WAVE_NODE_ERROR:test_file_not_allowed` for both the new regression file and the existing handler test file, so no Go PASS is claimed.
