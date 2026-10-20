---
title: AgentField execution heartbeat / premature stale timeout
status: in-progress
baseline_commit: eda6aca404808fa4702881505e83a3d554634c54
date: 2026-10-20
---
# Goal
Prevent false 10-15 minute no-activity timeouts on long-running SWE-AF reasoner calls, without suppressing genuine orphan detection or masking 2-hour/6-hour hard execution budgets. Reuse the existing AgentField execution update architecture, Python SDK watchdog, SQLite data, no new scheduler.

## Established RED
- Native 18080 serves real SWE runs; 2026-10-20 three Full builds: product_manager 29-34 min PASS -> run_architect timeout after 12-15 min, terminal DB error "execution timed out (no activity)".
- Running control-plane config: stale_execution_timeout 10m, cleanup_interval 5m; SDK max_execution_timeout 21600s (6 hours) is NOT the stale cutoff.
- Node heartbeat every few seconds proves node online, not execution activity. Python SDK _execute_async_with_callback has per-execution watchdog but no progress heartbeat during a long reasoner call.

## Scope and AC
- [ ] Extend existing control-plane execution routes with bounded POST /api/v1/executions/:id/heartbeat, using existing ExecutionStore.UpdateExecutionRecord. Only active running records with exact AgentNodeID and registered InstanceID may be refreshed. No resurrection, payload writes, callbacks, events, authorization bypass, or dependent workflow state transitions.
- [ ] Reuse existing Python SDK async watchdog to periodically heartbeat the running execution (not whole node); preserve true hard timeout and pause semantics; no unbounded retries, no private payloads.
- [ ] Go positive/adversarial tests: matching instance advances activity; terminal/waiting/foreign instance/foreign node/unknown execution never refresh; stale reaper protects genuinely active heartbeat but reaps abandoned records.
- [ ] Python tests: prolonged reasoner emits heartbeats, completion sends only one terminal result, cancelled/timed-out execution stops emitting.
- [ ] Build Go and run focused tests, Python focused tests, independent review and read-only E2E canary; only when no active executions, apply to 18080 with online DB backup and restore plan.
- [ ] Update this BMad spec and sprint-status with exact evidence and commit.

## Safety
No modification of SWE-AF dirty/untracked worktrees or Git branch deletion. Never disable stale cleanup wholesale. Preserve working 8080 legacy CP and existing SWE nodes. Do not declare production GREEN until native long-running E2E exercises the heartbeat across a reaper threshold; otherwise report scoped GREEN and next gate.

## Implementation & QA evidence — 2026-10-20
- New bounded `POST /api/v1/executions/:id/heartbeat` reuses atomic ExecutionStore.UpdateExecutionRecord; requires exact node + process-instance match and `running`, otherwise rejects with 403 or 409. No result/status/DAG/usage rewrite.
- SDK watchdog sends heartbeat approximately every 45s while non-paused coroutine is alive. It stops after terminal state; original active-time hard timeout remains. Invalid heartbeat responses never mark a bad execution succeeded.
- SDK TIMEOUT error formatting now surfaces actual server-side `error_message` (`execution timed out (no activity)`) rather than always reporting the six-hour max.
- Existing control-plane YAML stale timeout raised from 10m to 90m as compatibility guard for workers not yet restarted onto updated SDK, cleanup interval 5m retained. New heartbeat is the long-term fix; true stalls will still time out.
- Python SDK focused test: 2/2 PASS. Go heartbeat handler: matching node, foreign node, wrong generation, unfenced execution, succeeded/failed/timeout/waiting/unknown/malformed/oversize all PASS. Go storage live SQL test PASS (fresh heartbeat protects, stopped heartbeat reaped). Go server loopback test PASS; CGO build 73 MB PASS.
- Live isolated RED: 3s cleanup threshold with 5s SDK watchdog tick led to timeout at 3.56s (expected; unsupported threshold).
- Live isolated RED: reused node `heartbeat-canary` after prior registrations led to 403 on generation fence, SDK later got 409 for terminal callback. The prior execution instance and current registered instance differed. Do not waive the identity check; clear stale registration/route versions before adopting replacement workers.
- Live isolated GREEN: fresh `heartbeat-canary-r2` with matching registered and execution instance; `POST /api/v1/execute/async/heartbeat-canary-r2.wait` accepted 202, 16s coroutine completed `succeeded` with correct result even with `stale_timeout=8s` and cleanup every 1s. Run `run_20261020_154834_x11dy7i7`, execution `exec_20261020_154834_3ypnqb8c`, isolated port 18081. Main 18080 had 0 active runs during canary. Note healthy 204s are not logged by CP at warn level; actual survival beyond cutoff is evidence of touched activity.
- Live canonical `dev` AgentField branch lives at `D:\Users\NIKITA\Documents\DEV\universal-solver-site\agentfield` and has extensive unrelated dirty changes. Its working files are a separate worktree, although `.git` common dir is shared with this feature worktree. The feature must be committed, then selectively applied to untouched live paths and committed in `dev`; do not force checkout, stash, reset, or overwrite dirty files.
- Production deployment remains OPEN until live dev sync, current server backup, explicit restart and full 18080/SDK version readback. Independent QA beyond focused tests still OPEN.
