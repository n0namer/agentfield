# AgentField Project Plan

Last verified: 2026-09-11

## North Star
Build AgentField as the reliable execution and recovery plane for agents: bounded contracts are validated against CURRENT state, partial work is preserved, and the smallest safe continuation completes only what remains. This must improve reliability and cost for weaker/cheaper models without re-running already-completed mutations.

## Project decisions
- `PLAN.md` is the project/design SoT for North Star, phase goal, bounded batches, DoD, decisions, drift and next move.
- Runtime/readback owns actual state; this file must not claim a loaded change without runtime evidence.
- Code debugging and implementation is container-first. GitHub/CI/deploy is publication/release boundary, not the inner coding loop.
- SourceLoop canonicalizes only an exact delta already verified in DEV.
- Generic contract completion belongs in AgentField. FCM may select model/provider, but does not decide which obligations are satisfied.
- Generalize current upstream structured-output recovery (`DiagnoseFieldFailures`, `BuildIncrementalFollowup`, session resume); do not build a second workflow engine.
- Validators/runtime evidence override stale model self-report. Already-satisfied obligations are not repeated.
- Unknown or ambiguous state fails closed to read-only verification/escalation, not blind mutation.

## CURRENT evidence
- Canonical SoT is this `PLAN.md` on `n0namer/agentfield:main`.
- Fresh upstream read on 2026-09-11 confirms `Agent-Field/agentfield:main` is still `4aa3fe688dfa1f2437ac49f6cbe72aed43ddca07` (`v0.1.139-rc.1`).
- Upstream Go harness at that SHA already has field-level recovery: `DiagnoseFieldFailures` + `BuildIncrementalFollowup`, preserved partial output, and `ResumeSessionID` for non-crash retries.
- `feature/obligation-governor` remains research only. CI run `34625297461` reported failure before job steps, so it proves neither product failure nor PASS.
- `agentfield-dev-workforce` supports live patch + SourceLoop, but its `/src` contains consumer repos rather than the AgentField monorepo.
- `agentfield-dev-runtime-capture` proves capture writeback for capture-state, not product source. `agentfield-control-plane` cannot be live patched.
- `coding-runtime` is healthy and persistent, with writable `/data/repos` and `/data/workspaces`.
- Fresh exact-source workspace `csrepo_9dec760bc30844bc94423c765322bc27` is bound to current SoT SHA `b69d2365a6be75b6ea603b6e78d51089d3c701aa`; its publication remote was fetched successfully.
- The server-owned DEV target registry rejected binding this workspace as a new live-patch target (`target_registry_scope_denied`). This remains a capability gap; it must not be bypassed.
- The older workspace `csrepo_31ccf5fd977142d6951205a27cbe7169` is superseded for source-bound work by the fresh workspace above.

## Current Phase Goal
Use the existing containerized exact-source lane without creating infrastructure, reconcile it with exact upstream, then generalize the existing recovery behavior into the smallest contract-completion primitive and prove it deterministically before expensive E2E. Keep the missing live-patch/SourceLoop registration explicit rather than blocking all source-bound work.

## Bounded 30-minute batches
### Batch A — source reconciliation + RED design
DoD:
- Reuse existing `coding-runtime`; no new service/container.
- Exact upstream remains pinned and verified before source reconciliation.
- Preserve downstream `PLAN.md`, `AGENTS.md`, `ERRORS.md`, and intentional extensions while bringing the implementation base to current upstream.
- Add a deterministic failing test before product implementation: validator overrides stale self-report; `SATISFIED` is excluded; only `MISSING`/`INVALID` continue; `UNKNOWN` fails closed; observed mutation effects are not blindly repeated.
- No product code is edited through GitHub.

Rollback: discard only the bounded source workspace/session if reconciliation is wrong; do not rewrite protected history or touch unrelated sessions/runtime.

### Batch B — minimal contract-completion slice
DoD:
- Reuse/extract upstream recovery behavior instead of introducing a parallel workflow engine.
- RED test becomes GREEN with the smallest implementation delta.
- Run affected Go tests plus related harness regression tests on the exact source.
- Record missing runner/dependency as `VALIDATION_BLOCKER`; do not install ad hoc.
- Preserve exact tested delta for SourceLoop/canonical publication when that lane is available.

## Method / anti-drift
- BMAD v6: help → smallest fitting route; test-design/RED → deterministic oracle before fix; `bmad-build` → minimal vertical slice; review/verification closes the batch.
- Systematic debugging: reproduce/observe first, then one high-information discriminator at a time; no shotgun fixes.
- TDD: failing test first for deterministic behavior, then minimum code to pass, then regression.
- Eval: deterministic grader first; trajectory/tool-use and cost/turns are secondary after correctness.
- External skill research is method input only; do not install another framework merely because a skill exists.
- Anti-drift invariant: objective is AgentField upstream reconciliation + generic contract completion + SourceLoop-compatible exact delta. SWE/FCM/OpenCode is out of scope unless it directly blocks this objective.

## Next move
Inside the fresh `coding-runtime` exact-source workspace, reconcile the implementation base with upstream `4aa3fe688dfa1f2437ac49f6cbe72aed43ddca07` while preserving downstream SoT/guardrails. Then add the deterministic RED contract-completion test. Do not create infrastructure and do not edit product code on GitHub.
