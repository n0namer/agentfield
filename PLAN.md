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
- Exact-upstream workspace `csrepo_e1272c7c25b5405c8f8394ca9b155c35` remains open at `4aa3fe688dfa1f2437ac49f6cbe72aed43ddca07`.
- On 2026-09-11 the existing Coolify owner reconciled the earlier duplicate Coding Station deployment after one queued application restart: fresh readback now shows exactly one healthy `coding-api` and one healthy `coding-runtime`; `stationHealth` returns `ok`. The duplicate-route gateway blocker is therefore RESOLVED.
- Fresh `readRepoFile` on the exact-upstream workspace succeeds, including `sdk/go/harness/parity_test.go`; the previous gateway timeout is no longer the blocker.
- Fresh `execRun(["go","version"])` reaches the runtime deterministically but exits with `FileNotFoundError`; execution evidence reports `toolchains: []`. Exact deployed Coding Station source `36fe4e2480184f1461f557579ed88d18782afb7c` explains this: bare commands do not auto-provision languages. Dynamic toolchains are an explicit contract `station-toolchain exec TOOL@VERSION -- COMMAND`, backed by `mise`; Compose enables this lane and allowlists `go`. Therefore the correct RED prerequisite is a pinned toolchain invocation, not installing Go into the image.
- A pinned probe `station-toolchain exec go@1.25.1 -- go version` was issued through the authoritative Coding Station API but returned a gateway timeout. Post-timeout health/readiness calls also timed out while container readback still showed exactly one healthy API and one healthy runtime. This is an ambiguous transport/runtime result: verify current state before any retry; do not assume toolchain install failed and do not repeat mutation blindly.

## Current Phase Goal
Prove the smallest deterministic contract-completion loop on exact upstream: observed truth overrides self-report; completed effects are never repeated; unresolved obligations produce the smallest safe continuation; ambiguous mutation state fails closed. Keep the exact-source delta container-first and publish only after deterministic verification.

## Bounded 30-minute batches
### Batch A — exact-source RED readiness
DoD:
- No new service/container and no ad-hoc toolchain installation.
- Preserve the now-single authoritative Coding Station route and exact-upstream workspace.
- Use Coding Station's existing pinned dynamic-toolchain contract; do not require bare `go` to exist in the image and do not install it ad hoc.
- After the ambiguous first pinned-toolchain timeout, verify post-state before retry. Then verify `station-toolchain exec go@<pinned> -- go version` and one focused existing harness test execute on exact upstream.
- Only then mutate product source for RED.

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
- BMAD v6.12: research/observe first; use the smallest Build ceremony justified by risk, then one evidence-backed review/verdict pass instead of duplicating equivalent review layers.
- Systematic debugging skill: reproduce/observe before fixing; test one high-information hypothesis at a time; don't stack speculative fixes.
- TDD skill: no product implementation before an observed failing test for the target behavior; RED must fail for the intended reason, then minimal GREEN.
- Verification-before-completion: functional readback/test evidence, not tool acknowledgement or health alone, closes each guarantee.
- Eval: deterministic correctness first; trajectory/tool-use, repair count, cost and latency secondary.
- External skills are method input only; no framework installation without a proven gap.
- Anti-drift invariant: objective remains AgentField upstream reconciliation + generic contract completion + SourceLoop-compatible exact delta. SWE/FCM/OpenCode remains out of scope unless it directly blocks this objective.

## Next move
Resolve the Coding Station Go toolchain provisioning/advertisement mismatch without new infrastructure or ad-hoc installation. Then run one existing harness test as execution proof and immediately add the deterministic RED contract-completion tests on exact upstream. Do not edit AgentField product code on GitHub.
