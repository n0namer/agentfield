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

## Project-specific migration inventory (2026-10-20, second pass)
- **PRAF Deep Research (af-deep-research)**: Windows process on 8802 -> native 18080, healthy active, native classify-query E2E succeeded. Legacy 8801 -> 8080 remains parallel. Existing consumer cutover still requires identification; canary must not be represented as the end of old traffic.
- **PR-AF (independent pr-af repository, not Deep Research)**: source includes Go PR review node and Python PR review node. Existing defaults point at 8080, but no local active PR-AF listener/process was found; 8004 is SWE Fast, not PR-AF. Built source `go build -o D:\_cache\agentfield-ui-runtime\pr-af-18080-canary.exe ./cmd/pr-af` PASS. Started canary with `NODE_ID=pr-af-migration-canary PORT=8803 AGENTFIELD_SERVER=http://127.0.0.1:18080 AGENT_CALLBACK_URL=http://127.0.0.1:8803`: health OK, 18080 native registration active. No OPENROUTER_API_KEY/GH_TOKEN/AGENTFIELD_API_KEY present in observed launch environment: real pr-af.review integration cannot be accepted. The temporary Go process was stopped deliberately after smoke; port 8803 closed and node subsequently offline. Do not auto-start this credential-less/insecure canary.
- **CodeAF (separate CodeAF repository)**: existing native 18080 nodes `swe-codeaf-mvp` (127.0.0.1:8006) and `swe-codeaf-validation` (127.0.0.1:8005) are healthy. This proves SWE wrappers use CodeAF; it does not prove every standalone CodeAF workflow registers as an AgentField node. Dirty CodeAF source untouched.
- **SWE-AF**: native `swe-fast`, `swe-planner`, `swe-factory`, `swe-factory-30` on 18080, healthy. No migration required for these confirmed workers.
- **Universal Solver (separate universal-solver repository)**: code-first orchestrator with multiple isolated AgentField compose testbeds, some intentionally using `control-plane:8080` inside their container network; they are not necessarily using the host's legacy 8080. Do not replace Docker-internal URLs with host `127.0.0.1:18080`. No distinct active universal-solver AgentField node confirmed in this inventory. Repository has foreign dirty changes; untouched.
- **Skill Intelligence (separate af-skill-intelligence repository)**: Go libraries `acquisition`, `authoring`, `skillindex`, with `authoring.AgentFieldAuthor.BaseURL` as a client endpoint. Historical `skill-intelligence` node on legacy 8080 offline since Oct 5; no live standalone process found. Need resolve active caller + target reasoner before switching actual BaseURL. Dirty source untouched.
- **Legacy 8080**: still has healthy `meta_deep_research` and `demo-go-agent`, plus 94 historical runs. Do not shut it down or merge storage. Docker/remote runtime checks remain unresolved.

## Exact follow-up gates
1. Find actual production PR-AF deployment credentials, webhook routes and active listeners through authorized runtime; launch configured PR-AF pointing to 18080, then verify a read-only/authorized review E2E and health before calling it migrated.
2. For Universal Solver and Skill Intelligence, inventory live callers and isolate true node registrations from agent-client integrations; redirect only the real client/control-plane route. Preserve self-contained Docker testbeds.
3. Verify source URL/route, identity and real app result for every new project, run independent QA. Keep legacy 8080 until callbacks, historical storage backup, and external consumers are accounted for.

Gate for project-wide migration: OPEN. PR-AF compile/registration CANARY PASS only, NOT production GREEN. Avoid claiming CrewSync SERVER_DONE.
