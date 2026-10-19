---
title: AgentField collector reliability R2
status: in-progress
baseline_commit: 8030c3a6d74aff0b96836626d789b2551c2765e9
crew_sync_key: agentfield:obs-ui-mvp-r2-collector-reliability-20261019
owner: coordinator
---
# Intent
Repair independent QA RED from R1 in the existing local collector, without a second monitor or schema migration. Keep changes within scripts/agentfield_runtime_diagnostics.py and tests/test_agentfield_runtime_diagnostics.py.

## Tasks and acceptance
- [x] Reproduce three defects with RED tests (2026-10-19: 20 tests, 3 failures).
- [ ] Distinct node incident keys across restarts when seq is reused; persist no raw line or secrets; stable dedup of repeated identical NDJSON.
- [ ] Journal last valid JSON object without newline must not corrupt next append; replay must recover without duplicates after crash.
- [ ] Do not treat run.latest_activity/start time as heartbeat. Only a documented, explicit progress timestamp may yield stale; terminal failed/timeout unaffected.
- [ ] Run all focused tests, live existing Fast/Full probe twice, scheduler/health evidence, review and clean scoped commit.

## Out of scope
Per-child progress instrumentation and full logoff service-account setup require a truthful upstream heartbeat/identity contract and operational access. They cannot be falsely marked DONE by this patch; retain as separately scoped remainder if essential to full North Star.

## R2 implementation evidence — CANDIDATE
- RED: `python -m unittest discover -s tests -p test_agentfield_runtime_diagnostics.py -q` produced 20 tests, 3 reproducible failures (node seq reset, JSONL missing LF, stale with only latest_activity). Separate legacy migration RED: 21 tests, 1 fail.
- GREEN: same command yielded 21/21 PASS after changes. `python -m py_compile scripts/agentfield_runtime_diagnostics.py` PASS.
- Changes: source event-time included in node alert key with validated ISO timestamp, legacy journal migration keyed by original detection time to prevent duplicate historical alerts, correct newline separator before JSONL append, stale only with explicit last_progress_at (the current backend does not expose that as a trusted field).
- Live A55: two direct scans returned 20 findings and 0 new alerts; journal stayed at 34 lines. Windows scheduler manually re-run returned LastTaskResult=0; backend health healthy.
- Independent QA R2 not performed. Remaining DoD: prove true process restart identity beyond simulated timestamp difference, logged-off scheduler execution, and child-level correlation if available. Do not call the task DONE; exact SHA to be set after commit.
- R3 independent UI attempt in isolated `fix/af-obs-ui-r3-20261019` worktree failed at provider 429, no product changes; task released to OPEN, not DONE.
