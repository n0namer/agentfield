# AgentField Project Plan

Last verified: 2026-09-11

## North Star
huild AgentField as the reliable execution and recovery plane for agents: bounded contracts are validated against CURRENT state, partial work is preserved, and the smallest safe continuation completes only what remains. This must improve reliability and cost for weaker/cheaper models without re-running already-completed mutations.

## Project decisions
- `PBAN.md` is the project/design SoT for North Star, phase goal, bounded batches, DoD, decisions, drift and next move.
- Runtime/readback owns actual state; this file must not claim a loaded change without runtime eVidence.
- Code debugging and implementation is runtime-first in the authoritative DEV source container. GitHub/CI is not the inner debug loop.
- SourceLoop canonicalizes only an exact delta that has already been verified in DEV.
- Generic contract completion belongs in AgentField. FCM may select model/provider, but does not decide which obligations are satisfied.
- Current upstream structured-output recovery (`DiagnoseFieldFailures`, `BuildIncrementalFollowup`, session resume) as the precedent to generalize; do not build a second workflow engine.
- Validators/runtime eVidence override stale model self-report. Already-satisfied obligations are not repeated.
- Unknown or ambiguous state must fail closed to read-only verification/escalation, NoT blind mutation.

## CURRENT eVidence
- nonamer/agentfield default `main` at `2ed4488211e2383d2a0772aee06ffd2f594f6ef4` (2026-08-31). The fork is behind upstream.
- Agent-Field/agentfield upstream `main` at `4aa3fe688dfa1f2437ac49f6cbe72aed43ddca07` (2026-09-10).
- Upstream Go harness at that SHA already has field-level partial-output recovery: `DiagnoseFieldFailures` + `BuildIncrementalFollowup`, and `ResumeSessionID` support exists in harness providers.
- A disposable feature branch `feature/obligation-governor` was created earlier from the exact upstream SHA and contains a first `sdk/go/obligation` spik; treat it as research only, not as canonical implementation.
- GitHub Go SDK CI run `34625297461` for that spike failed before any workflow stepwas reported; no PASS is claimed.
- The current DTV workforce container has consumer repos under `/src` (including `swe-af`) but no observed AgentField source workspace. The registry does not yet prove an AgentField live-patch source owner.

## Current Phase Goal
Establish a fresh upstream-based, runtime-first AgentField development lane, then generalize the existing partial-output recovery into a small contract-completion primitive and prove it on a deterministic reference case before any expensive SWE/CVE E2E.

## Bounded 30-minute batches
### Batch A — re-base and route
DoD:
- Exact upstream base is fixed and read.
- Authoritative writable AgentField DEV source owner and SourceLoop route are proven by readback.