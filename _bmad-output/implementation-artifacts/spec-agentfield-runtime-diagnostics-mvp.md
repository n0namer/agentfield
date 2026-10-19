---
title: AgentField runtime diagnostics MVP
status: in-review
baseline_commit: 85fc69ad3b6ae4e2838b78f93a629891b7ef40d9
crew_sync_key: agentfield:runtime-diagnostics-local-watch-20261019-r1
crew_sync_writeback: blocked
---
# Goal
Read-only anomaly detection on A55 via existing Native AgentField REST APIs. No second control plane, paid API, new dashboard, secrets, or outbound delivery.

## Existing surfaces
GET /api/ui/v2/workflow-runs, /api/v1/executions/:id, /api/ui/v1/nodes/summary, /api/ui/v1/nodes/:id/logs. Existing observability_forwarder is disabled and remains unchanged. Existing React UI has DAG, logs and application-success indicator.

## Tasks & Acceptance
- [x] Implement scripts/agentfield_runtime_diagnostics.py: localhost-only bounded polling, status/app-success contrast, stale running, failed/timeout and classified node connection/retry/429 diagnostics.
- [x] Persist sanitized JSONL and atomic cursor state; suppress duplicate alerts on repeated runs and after restart, never persist raw payload or log line.
- [x] Add deterministic stdlib tests for positive, false-green, timeout, no result, node log, replay, malformed input, failure and privacy.
- [x] Live A55 smoke twice on known Fast/Full: false-green must surface, repeated run must report 0 duplicate alerts.
- [x] Review adversarial and edge cases, record evidence, commit only scoped changes.

## Scope / DoD
Given the native API with historical Fast and Full records, when the collector reads existing data, it emits stable local alert keys with run/execution/node IDs, type, severity and canonical UI link. Given replay, no duplicates occur. All data is read-only; legacy routes/services, SWE/Codex and their databases are not modified. External alerts, broader Codex-session token metrics and advanced dashboard are deferred until recipient and security contract are available.

## Source of truth
This repository has no existing BMad output or sprint-status.yaml. This is the feature-specific implementation artifact; no PLAN.md. CrewSync task creation was blocked by tool safety; do not assert CrewSync acceptance.

## Verified evidence (A55, 2026-10-19)
- `python -m unittest discover -s tests -p test_agentfield_runtime_diagnostics.py -q`: 15/15 PASS (0.095s).
- `python -m py_compile scripts/agentfield_runtime_diagnostics.py`: PASS.
- `python scripts/agentfield_runtime_diagnostics.py --state-dir D:\\_cache\\agentfield-ui-runtime\\diagnostics-smoke`: 13 anomalies; repeated polling: 0 new alerts.
- Confirmed categories on real data: 3 execution_failed, 2 application_failed, 8 node_runtime_error. Existing Full `exec_20261019_174256_v7bjqcip` classified application_failed despite native transport succeeded.
- Windows Task Scheduler `AgentField Runtime Diagnostics`: registered every 5 minutes, launched; LastTaskResult=0; persistent journal `C:\\Users\\NIKITA\\.agentfield\\diagnostics\\alerts.jsonl` contains 13 sanitized entries.
- Native AgentField health healthy; 18080 bound to 127.0.0.1. No SWE/Codex restarts, backend schema mutations, webhook configuration or external sends.
- CrewSync readback found older overlapping but different forensic task `agentfield:v2-base20-runtime-log-forensics-20261013-r1` with a restricted write scope. New task creation refused by tool safety. Its status must not be represented as claimed, created or DONE.

## Adversarial review — triaged findings
1. API outage is printed as an error, but a failed poll is not stored as a journal alert (defer).
2. Node-local log seq may reset after node restart, so old keys can suppress new same-seq errors (defer).
3. The task is interactive-user Task Scheduler: execution while Windows user is logged off is not guaranteed (defer).
4. No external recipient is configured: no Telegram/email/webhook delivery occurred (external prerequisite).
5. Per-child stalled step detection is absent; root latest_activity may mask a stuck child (defer).
6. The classifier cannot associate every stdout/stderr line with an execution_id (defer).
7. Only three pages of 100 recent runs are scanned per cycle (bounded MVP).
8. For unknown reasoners with missing result.success, outcome is unknown rather than failed (intentional fail-safe).
9. Log levels and broad rate-limit regex can produce false positives; no per-rule precision metric (defer).
10. The 16 MB journal ceiling intentionally fails closed and lacks automatic rotation (defer).
11. Malformed NDJSON rows are skipped; their details are not collected as a distinct diagnostic (defer).
12. No Codex JSONL token/use attribution was added; only already-authorized AgentField summaries are read (separate phase).

## Edge-case review — actions
- Fixed: duplicate keys in one response, crash after append/before state save, concurrent state access, malformed journal/state fail-closed, missing node seq not hashed from private text, long-running execution with recent activity not treated as stalled.
- Verified: malformed timestamps, missing application result, network error classification, invalid origin, unsafe node IDs, initial and replay execution IDs.
- Unresolved: node generation/seq collision after restart, internal scheduler outage when logged off, delivery to recipient. These do not block the bounded local read-only collector but block full alert-delivery DoD.

## Gate
Local diagnostic collector operational GREEN and persisted schedule readback GREEN. `in-review`: independent QA not yet signed and CrewSync acceptance not available. Do not call this CrewSync SERVER_DONE. Next: independent QA and existing observability-forwarder notification destination, then node-generation correlation and Codex metadata-only token sourcing.
