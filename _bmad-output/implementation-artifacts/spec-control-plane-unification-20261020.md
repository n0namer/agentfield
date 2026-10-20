---
title: AgentField Control Plane 8080 to 18080 Unification
status: in-progress
baseline_commit: 3fbe5d4f7749da251b9a6485bac51b2cfbd49b51
date: 2026-10-20
scope: operational-cutover
---
# Goal
Converge live AgentField nodes and new tasks on native A55 Control Plane 18080; preserve 8080 durable history and existing executions. No blind DB merge, no open control-plane binding, no restart of unrelated projects.

## Inventory (verified on A55)
- 8080: separate legacy Control Plane reachable via Docker Desktop; 7 registered nodes; 94 historical runs; live meta_deep_research and demo-go-agent, 5 offline nodes. Three stale run records display running with October 5 last_activity.
- 18080: native Windows Go server, bound to 127.0.0.1 only; originally six ready SWE nodes and 26 runs.
- meta_deep_research: old Python process on Windows port 8801, server=8080; old callback reachable via 172.26.0.1:8801.
- Skill Intelligence: historical 8080 node offline since 2026-10-05; current af-skill-intelligence repository is Go libraries including client AgentFieldAuthor (BaseURL), not an active standalone node; foreign source worktree dirty, unchanged.

## Per-stage Tasks and Acceptance
- [x] Locate both live control planes, runtime owners, registrations, current run states.
- [x] Preserve pre-cutover data: new Control Plane SQLite online backup integrity_check=ok, snapshot registration and run metadata from both APIs (8080:94, 18080:27). No raw credential or user payload export.
- [x] Start second PRAF instance on Windows 8802 with AGENTFIELD_SERVER=http://127.0.0.1:18080, callback=http://127.0.0.1:8802, independent research-run directory; no changes to dirty project source and no stop of old 8801.
- [x] Confirm 18080 node meta_deep_research active/ready, 23 advertised reasoners, node health healthy and node raw-log proxy HTTP200.
- [x] Native E2E using POST /api/v1/execute/async/meta_deep_research.classify_query_adaptive, execution exec_20261020_134104_es9x1jsm, run run_20261020_134104_48cbam04: queued 202 -> succeeded; classification result fields core_subject/key_question/query_type, DAG and execution logs HTTP200 with 4 events.
- [x] Register logon scheduled task 'AgentField Deep Research 18080' with port-8802 guard and limited current-user principal. Actual task LastTaskResult=0; no duplicate port start. Existing 8080/18080 diagnostics schedulers LastTaskResult=0.
- [ ] Identify all producers/consumers still submitting new PRAF requests to 8080 and switch them to 18080. A parallel canary being healthy is not by itself a complete traffic cutover.
- [ ] Preserve a full consistent legacy-8080 storage snapshot; only legacy API *metadata* snapshot done. No live Docker/SQLite storage modification attempted.
- [ ] Map demo-go-agent Docker service to trusted internal 18080 route before migrating. DO NOT change 18080 from loopback-only to an unprotected public listener.
- [ ] Find active runtime/caller configuration for Skill Intelligence, switch AgentFieldAuthor BaseURL when active and verify actual native readback. Do not fake-register Go libraries as an agent.
- [ ] Independent QA/replay/callback across process restart, reconcile 8080 old historical run states, confirm every new project uses 18080. Only then archive 8080.

## Evidence / locations
- Control Plane main: http://127.0.0.1:18080/ui/runs/run_20261020_134104_48cbam04
- Canary launcher: D:\Temp\af-deep-research-migration-20261020\launch_on_18080.py
- Idempotent logon wrapper: D:\_cache\agentfield-ui-runtime\afdr-18080-start.ps1
- Metadata snapshots and SQLite backup: D:\_cache\agentfield-ui-runtime\migration-backup-20261020
- Old 8080 and old PRAF 8801 both remained healthy after the canary E2E.

## Gate
PRAF parallel native migration canary: GREEN. Global project/client traffic cutover and legacy archive: OPEN. No CrewSync SERVER_DONE and no claim that 94 historical old runs were moved. No changes to foreign dirty/untracked files.
