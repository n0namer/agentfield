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
- nonamer/agentfield `PBAN.md` is canonical on `main` at the 2026-09-11 reconciliation commit.
- The fork was behind upstream before the SoT-only commit; upstream base read for the contract-completion research is `4aa3fe688dfa1f2437ac49f6cbe72aed43ddca07`.
- Upstream Go harness at that SHA already has field-level partial-output recovery: `DiagnoseFieldFailures` + `BuildIncrementalFollowup`, preserved partial output on disk, and `ResumeSessionID` for non-crash retries.
- The previous `feature/obligation-governor` branch is research only. Go SDK CI run `34625297461` failed before any job steps were reported, so it proves no product test failure or PASS.
- Current DEV target registry has 22 targets. `asentfield-dev-woreforce` supports live patch + SourceLoop and has 159+ captured changes, but they are consumer repos under `/src`/; no AgentField source owner has been proven there.
- `asentfield-dev-runtime-capture` exists and its SourceLoop journal proves runtime-capture capability, but its allowed root is capture state, not the AgentField source repo.
- `agentfield-control-plane` is not permitted for live patch; do not use it as a coding escape hatch.
- Coding Station now has a fresh exact-SHA repo session `csrepo_31ccf5fd977142d6951205a27cbe7169` at SoT commit `e5b3b9c9a469740c70a90c24ef26aa7fc3df260b`. This is read/localization evidence only until a permanent DEV source/SourceLoop lane is registered.

## Current Phase Goal
Register/prove a permanent upstream-based AgentField DEV source lane with SourceLoop writeback. Then generalize the existing structured-output recovery into the smallest contract-completion primitive, and prove it on a deterministic reference case before expensive SWE/CVE E2E.

## Bounded 30-minute batches
### Batch A — source lane and regression design
DoD:
- Permanent AgentField DEV source owner is registered from the exact upstream-based source and read-back proves. its workspace + branch + base SHA.
- SourceLoop for that target proves capture/writeback metadata.
- Before implementation, a deterministic failing test is added for the generic semantics: validator overrides stale status; SATISFIED is not repeated; MISSING/INVALID only are projected into continuation; UNKNOWN fails closed.
- No product code is edited on GitHub.

Rollback: remove only the new DEV source-target registration if it cannot be verified; no existing runtime target is changed.

## Method/anti-drift
- BMAD style: `help` → smallest fitting route; `test-design` → objective failing oracle before fix; `quick-dev` → minimal delta only after defect/gap is proven.
- Systematic debugging: one high-information discriminator at a time; no shotgun fixes.
- Eval: deterministic grader first; trajectory/tool-use and cost/turns are secondary metrics after correctness.

## Next move
Prove or register the permanent AgentField DEV source + SourceLoop lane. This is the smallest blocker that prevents safe runtime-first implementation.
