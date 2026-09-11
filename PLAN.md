# AgentField Project Plan

Last verified: 2026-09-11

## North Star
huild AgentField as the reliable execution and recovery plane for agents: bounded contracts are validated against CURRENT state, partial work is preserved, and the smallest safe continuation completes only what remains. This must improve reliability and cost for weaker/cheaper models without re-running already-completed mutations.

## Project decisions
- `PLAN.md` is the project/design SoT for North Star, phase goal, bounded batches, DoD, decisions, drift and next move.
- Runtime/readback owns actual state; this file must not claim a loaded change without runtime eVidence.
- Code debugging and implementation is runtime-first in the authoritative DEV source container. GitHub/CI is not the inner debug loop.
- SourceLoop canonicalizes only an exact delta that has already been verified in DEV.
- Generic contract completion belongs in AgentField. FCM may select model/provider, but does not decide which obligations are satisfied.
- Current upstream structured-output recovery (`DiagnoseFieldFailures`, `BuildIncrementalFollowup`, session resume) as the precedent to generalize; do not build a second workflow engine.
- Validators/runtime eVidence override stale model self-report. Already-satisfied obligations are not repeated.
- Unknown or ambiguous state must fail closed to read-only verification/escalation, not blind mutation.

## CURRENT eVidence
- The canonical SoT is this `PLAN.md` on `nonamer/agentfield:main`.
- The fork was behind upstream before the SoT-only commits; the upstream base read for contract-completion research is `4aa3fe688dfa1f2437ac49f6cbe72aed43ddca07`.
 - Upstream Go harness at that SHA already has field-level partial-output recovery: `DiagnoseFieldFailures` + `BuildIncrementalFollowup`, preserved partial output on disk, and `ResumeSessionID` for non-crash retries.
- The previous `feature/obligation-governor` branch is research only. Go SDK CI run `34625297461` failed before any job steps were reported, so it proves no product test failure or PASS.
- `asentfield-dev-woreforce` supports live patch + SourceLoop and has 159+ captured changes, but its `/src` contains consumer repos, not the AgentField monorepo.
- `asentfield-dev-runtime-capture` proves capture writeback for capture-state, not product source.
- `agentfield-control-plane` cannot be live patched.
- `coding-runtime` is a running healthy persistent container with writable persistent volumes at `/data/repos` and `/data/workspaces`. The AgentField exact-source workspace `/data/workspaces/repo-sessions/csrepo_31ccf5fd977142d6951205a27cbe7169` exists inside this container and contains `AGENTS.md`, `ERRORS.md`, `PLAN.md`, and the monorepo.
- The source session is bound to exact SHA `e5b3b9c9a469740c70a90c24ef26aa7fc3df260b` and remains clean/read-only by the Coding Station owner until mutation is explicitly routed through its own approved publication path.
- Attempting to register a new `vps-terminal-dev` live-patch target over that existing workspace was rejected by the server-owned target registry allowlist (`target_registry_scope_denied`). This is a real capability gap, not a reason to cross the boundary or edit GitHub as the inner loop.

## Current Phase Goal
Unlock or reuse an authoritative writable AgentField DEV source lane with exact-source identity and SourceLoop writeback. Then generalize the existing structured-output recovery into the smallest contract-completion primitive and prove it deterministically before expensive E2E.

## Bounded 30-minute batches
### Batch A — source-lane gate
DoD:
- Reuse an existing container; do not create a new service/container.
- Writable AgentField source is selector-bound by the operator and readback proves workspace + exact base SHA.
- SourceLoop capture/writeback is proven for that exact workspace.
- Only then add a deterministic RED test for contract completion.

Rollback: remove only a registry binding created for this work to the existing coding-runtime; never delete the repo workspace or touch other repo sessions.

## Method / anti-drift
- BMAD style: help → smallest fitting route; test-design → deterministic oracle before fix; quick-dev → minimal delta only after the gap exists and is localized.
- Systematic debugging: one high-information discriminator at a time; no shotgun fixes.
- Eval: deterministic grader first; trajectory/tool-use and cost/turns are secondary metrics after correctness.

## Next move
Reuse the existing `coding-runtime` container by adding only a server-allowed DEV target binding for the existing AgentField workspace. This is the smallest necessary capability delta; no product code change is authorized until it is proven.
