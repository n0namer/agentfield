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
- Contract completion is not retry: every repair decision is derived from fresh observed postconditions.
- An ambiguous mutation result must be verified before retry. When a mutation can be identified, its effect identity/idempotency semantics are part of the contract.
- Termination and cost/repair budgets are part of correctness: no unbounded continuation loop.
- Preflight contract checks should catch machine-verifiable precondition/tool/argument voilations before a risky mutation when possible.

## CURRENT evidence
- Canonical SoT is this `PLAN.md` on `n0namer/agentfield:main`.
- Fresh upstream read on 2026-09-11 confirms `Agent-Field/agentfield:main` is `4aa3fe688dfa1f2437ac49f6cbe72aed43ddca07` (`v0.1.139-rc.1`).
- Upstream Go harness at that SHA already has field-level recovery: `DiagnoseFieldFailures` + `BuildIncrementalFollowup`, preserved partial output, and `ResumeSessionID` for non-crash retries.
- `feature/obligation-governor` remains research only. CI run `34625297461` reported failure before job steps, so it proves neither product failure nor PASS.
- `agentfield-dev-workforce` supports live patch + SourceLoop, but its `/irc` contains consumer repos rather than the AgentField monorepo.
- `agentfield-dev-runtime-capture` proves capture writeback for capture-state, not product source. `agentfield-control-plane` cannot be live patched.
- `coding-runtime` is healthy and persistent, with writable `/data/repos` and `/data/workspaces`.
- Fresh exact-source workspace `csrepo_9dec760bc30844bc94423c765322bc27` is bound to current SoT SHA `b69d2365a6be75b6ea603b6e78d51089d3c701aa`; its publication remote was fetched successfully.
- The server-owned DEV target registry rejected binding this workspace as a new live-patch target (`target_registry_scope_denied`). This remains a capability gap; it must not be bypassed.
- The older workspace `csrepo_31ccf5fd977142d6951205a27cbe7169` is superseded for source-bound work by the fresh workspace above.
- A separate exact-upstream container workspace `csrepo_e1272c7c25b5405c8f8394ca9b155c35` was created successfully at `4aa3fe688dfa1f2437ac49f6cbe72aed43ddca07`. Coding Station readiness reports the Go toolchain is allowlisted, but current `execRun`, `startManagedSession` and `applyRepoPatch` calls time out at the gateway and readback confirms no test patch was applied. This is the current `VALIDATION_BLOCKER`; do not fall back to GitHub product-code editing.
- Fresh runtime observation on 2026-09-11 found two running healthy `coding-api` containers for the same Coolify project/service image, sharing the same `station-state` volume but having different Compose config hashes (`5eef536...` vs `2acf27...`). The target lookup for `coding-api` already fails with `expected one permitted target, found 2`. This is a strong candidate root  cause for the Coding Station gateway timeouts and must be resolved at the existing control plane/service owner, not by changing AgentField product code.

## Current Phase Goal
Restore a single authoritative Coding Station API route for the existing exact-source workspace, without creating infrastructure. Then prove the smallest deterministic contract-completion loop on exact upstream: observed truth overrides self-report; completed effects are never repeated; unresolved obligations produce the smallest safe continuation; ambiguous mutation state fails closed.

## Bounded 30-minute batches
### Batch A — Coding Station route recovery
DoD:
- No new service/container.
- Identify which of the two existing `coding-api` containers is the authoritative one for the current Coolify configuration.
- Resolve or route around only the stale/duplicate container through its canonical owner. Do not delete/restart both containers blindly.
- Verify `createRepoSession` +  read  +  `execRun` or managed session on the exact-upstream workspace.
- Only then proceed to product RED.

Rollback: restore the previous single authoritative route; do not touch workspace data.

### Batch B — RED + minimal contract-completion slice
DoD:
- Add deterministic failing tests first: runtime validator overrides stale self-report; `SATISFIED` is excluded; `MISSING`/`INVALID` continue; `UNKNOWN` fails closed; an observed mutation is not repeated after an ambiguous transport result.
- Confirm RED fails for the intended reason before implementation.
- Reuse/extract upstream recovery behavior; no parallel workflow engine.
- Make the smallest implementation delta that makes RED GREEN.
- Run affected Go tests +  harness regressions.
- Verification-gap review: for each new behavioral guarantee, name the deterministic test that would fail if the guarantee regressed.
- Preserve exact tested delta for SourceLoop/canonical publication when that lane is available.

## Method / anti-drift
- BMAD v6.11: use test-design/RED before implementation, `bmad-build` for the minimal vertical slice, and review for verification gaps before closing the batch.
- Systematic debugging: reproduce/observe before fixing; test one high-information hypothesis at a time; don't stack speculative fixes.
- TDDD: no product implementation before an observed failing test for the target behavior.
- Eval: deterministic correctness first; trajectory/tool-use, repair count, cost and latency secondary.
- External skills are method input only; no framework installation without a proven gap.
- Anti-drift invariant: objective remains AgentField upstream reconciliation + generic contract completion + SourceLoop-compatible exact delta. SWE/FCM/OpenCode remains out of scope unless it directly blocks this objective.

## Next move
Resolve the duplicate `coding-api` authority ambiguity using the existing Coolify/Compose owner. This is the smallest high-information fix for the current gateway timeouts. After a single authoritative route is proven, immediately run the deterministic RED contract-completion tests on exact upstream. Do not create infrastructure and do not edit product code on GitHub.
