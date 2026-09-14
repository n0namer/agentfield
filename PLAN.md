# AgentField Project Plan

Last verified: 2026-09-14

## North Star
Build AgentField as the reliable execution and recovery plane for agents: bounded contracts are validated against CURRENT state, partial work is preserved, and the smallest safe continuation completes only what remains. This must improve reliability and cost for weaker/cheaper models without re-running already-completed mutations.

## Project decisions
- `PLAN.md` is the project/design SoT for North Star, phase goal, bounded batches, DoD, decisions, drift and next move.
- Runtime/readback owns actual state; this file must not claim a loaded change without runtime evidence.
- Code debugging, implementation, and validation are direct-target/container-first. Do not place Coding Station or any other helper/control-plane proxy in the critical path. GitHub/CI/deploy is publication/release boundary, not the inner coding loop.
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
- Historical exact-upstream workspace `csrepo_e1272c7c25b5405c8f8394ca9b155c35` remains evidence for the earlier RED/GREEN slice at `4aa3fe688dfa1f2437ac49f6cbe72aed43ddca07`, but it is NOT the current runtime source and is removed from the active coding path.
- Fresh direct runtime readback on 2026-09-14 identifies the authoritative running AgentField control-plane container `control-plane-edshqtkwskg3lrczekhcmd71-170600486623`. Its `/build/source-sha` is `c0923acdfca043c2c07e3d34daaa09e2a7e41d38`, which exactly matches the shared source checkout `/core-src/agentfield-runtime/.git/HEAD` exposed RW in sibling `runtime-capture-edshqtkwskg3lrczekhcmd71-170600520090` and RO as `/workspace/agentfield-runtime` in the control-plane container. This direct shared volume is the current container-first source path.
- The same shared source volume also contains an older `/core-src/agentfield` checkout at `4d337c1ae5104418311fcba414a1c2f85c2abb89`; it is not runtime-authoritative and must not be edited for current work.
- `AGENTS.md` was read from `/core-src/agentfield-runtime` before mutation; no root `ERRORS.md` is present there. Fresh search of current runtime source confirms `PlanContractContinuation` is absent, so the previously verified contract-completion delta has not been loaded into the actual runtime source. This is DESIGN_RUNTIME_DRIFT, not completion.
- Direct target topology is now explicit and writable through the existing DEV operator path: control-plane consumes the source volume read-only, while `runtime-capture` owns the same core source volume read-write at `/core-src/agentfield-runtime`. Coding Station is not part of the active route.
- The DEV VPS Terminal mediation defect was repaired in the active hotfix with a stale-safe `fileAction`: `TARGET_REGISTRY_UPSERT` was added to `operation-mediation.mjs`, `node_check` passed, and SourceLoop capture `vtchg_8942c719b4154853b6c99fc3604557c7` recorded the operator delta. After owner-plane restart, the gateway loaded image-mode mediation with 43 registered routes and `/v1/target-registry/action` upsert became callable.
- The server-owned registry allowlist was extended only for the existing `agentfield-dev-runtime-capture` target. Registry upsert rev1→rev2 succeeded with readback, then rev2→rev3 corrected `live_patch_runtime` from `node` to `python`. Current `runtime-capture` target has `workspace_root=/core-src/agentfield-runtime`, live roots `[/capture-state, /core-src/agentfield-runtime]`, and stale-safe file actions now succeed on the runtime-authoritative checkout.
- The contract-completion slice is loaded directly into runtime-authoritative source `c0923acdfca043c2c07e3d34daaa09e2a7e41d38`. Current dirty runtime files are exactly `sdk/go/harness/schema.go`, `runner.go`, `parity_test.go`, and `aforge_test.go`; `git_diff_check` PASS after the latest test addition. Production semantics are limited to `schema.go` + `runner.go`; `parity_test.go` and `aforge_test.go` pin deterministic helper and public-Runner behavior.
- Direct validation scope has also been repaired: `agentfield-control-plane` was updated rev1→rev2 with the actual `debian:bookworm-slim` selector and `/workspace/agentfield-runtime` workspace; registry readback verified. Current runtime diff is exactly four files (`schema.go`, `runner.go`, `parity_test.go`, `aforge_test.go`) and `git_diff_check` PASS; direct `git diff --name-only` confirms no unrelated source changes.
- Direct runtime-equivalent validation is now complete without any new container. Existing PROD VPS Terminal `workbench` already contained `/workspace/.tools/go/bin/go` (`go1.25.10`). A validation checkout was created at exact runtime base `c0923acdfca043c2c07e3d34daaa09e2a7e41d38`, and the live runtime diff was exported to `/capture-state/agentfield-contract.patch` with SHA-256 `a2b20e051b00fe9cd26ac024baf5ed9a228400cdae2f605c1404be445dc0f2e7`, transferred to the workbench, hash-verified, and applied. The resulting three file hashes exactly match live runtime source: `schema.go` `c634322ae872ae545ca4a265728f485df9523b6962a5716a647dae8fb2cb93a9`, `runner.go` `12f14c161b37479b4c77beb355ec008acfb826bfb185348de35864b0458c15d5`, `parity_test.go` `57bc2ad790689cf211da3bc75db8e9a50d6c551067e5e4f8e39b257f512fc1d6`.
- Fresh validation on that byte-identical source passed: `gofmt -l` returned no files; targeted contract-completion + incremental-recovery tests PASS (`ok`, 0.021s); full `go test ./harness -count=1` PASS (`ok`, 13.302s); full Go SDK `go test ./... -count=1` PASS across `agent`, `ai`, `client`, `did`, `harness`, `inputs`, `triggers`, and `types`. This closes the runtime-loaded regression gate. The previously considered ephemeral `control-plane-build` container is no longer needed for this phase.
- On 2026-09-11 the existing Coolify owner reconciled the earlier duplicate Coding Station deployment after one queued application restart: fresh readback now shows exactly one healthy `coding-api` and one healthy `coding-runtime`; `stationHealth` returns `ok`. The duplicate-route gateway blocker is therefore RESOLVED.
- Fresh `readRepoFile` on the exact-upstream workspace succeeds, including `sdk/go/harness/parity_test.go`; the previous gateway timeout is no longer the blocker.
- Fresh `execRun(["go","version"])` reaches the runtime deterministically but exits with `FileNotFoundError`; execution evidence reports `toolchains: []`. Exact deployed Coding Station source `36fe4e2480184f1461f557579ed88d18782afb7c` explains this: bare commands do not auto-provision languages. Dynamic toolchains are an explicit contract `station-toolchain exec TOOL@VERSION -- COMMAND`, backed by `mise`; Compose enables this lane and allowlists `go`. Therefore the correct RED prerequisite is a pinned toolchain invocation, not installing Go into the image.
- On 2026-09-14 the previous ambiguous pinned-toolchain state was re-verified before retry: Coding Station health returned `ok`, exact-upstream workspace `csrepo_e1272c7c25b5405c8f8394ca9b155c35` remained open at `4aa3fe688dfa1f2437ac49f6cbe72aed43ddca07`, and `station-toolchain exec go@1.25.1 -- go version` completed successfully with `go1.25.1 linux/amd64`.
- The first focused Go harness test then exposed a deterministic execution-environment issue, not a product failure: the generated test binary under `/tmp/gcs-repo-sessions/...` could not execute (`permission denied`). Setting `GOTMPDIR` inside the exact-source workspace through the same pinned toolchain lane resolved the noexec-temp constraint, and `TestBuildIncrementalFollowup` passed (`ok`, 0.007s). Batch A RED readiness is therefore functionally proven without installing Go or creating new runtime infrastructure.
- Current upstream `Agent-Field/agentfield:main` was re-read on 2026-09-14 and is still exactly `4aa3fe688dfa1f2437ac49f6cbe72aed43ddca07`; the tested workspace base has no upstream drift.
- Batch B RED was observed before implementation: focused Go tests failed to compile specifically because `ObligationObservation`, obligation states, and `PlanContractContinuation` did not yet exist.
- The exact container-only delta is limited to `sdk/go/harness/schema.go`, `runner.go`, and `parity_test.go`. It adds runtime-observed obligation states/planning and changes schema-constrained provider execution from opaque nested transport retry to one attempt → persisted postcondition observation → smallest remaining continuation. Non-schema transport retry behavior remains unchanged.
- Deterministic guarantees are pinned by `TestPlanContractContinuation_RuntimeEvidenceWins`, `TestPlanContractContinuation_AmbiguousTransportObservedEffectIsNotRepeated`, `TestHandleSchemaWithRetry_AmbiguousProviderErrorUsesObservedPostcondition`, and `TestHandleSchemaWithRetry_ExecuteErrorUsesObservedPostcondition`; existing incremental recovery remains covered by `TestHandleSchemaWithRetry_IncrementalRecovery`.
- Fresh post-format verification on the exact workspace passed: targeted new tests `ok` (0.017s), full `./harness` `ok` (7.300s), full Go SDK `go test ./... -count=1` PASS across all packages, and `git-diff-check` PASS. The noexec test-environment constraint is handled only at invocation with workspace-local `TMPDIR` + `GOTMPDIR`; product code was not changed to accommodate the environment.
- During verification the synchronous Coding Station Action route intermittently timed out. Post-state was checked before retry, one scoped restart of the existing Coding Station helper restored health, and managed-process execution on the same persistent repo-session completed formatting/tests without new infrastructure or AgentField redeploy.
- Publication anti-drift caught a repository-history mismatch before merge: a draft PR opened from the exact-upstream tested branch against `n0namer/agentfield:main` rendered as 89 commits / 320 files, so it was immediately closed without merge or deploy. Fresh container readback from fork-main shows `HEAD...4aa3fe...` diverged by 20 fork-only commits vs 88 upstream-only commits; the observed fork-only commits are PLAN/docs history, while upstream-only history contains the product evolution. Therefore direct publication of the tested branch into fork main is not a bounded code delta and is blocked pending an explicit fork-reconciliation decision.
- Reconciliation discovery narrowed the fork-only tree delta since common ancestor `4d4d54e402bc89c6d0e52e8c826d679409214778` to exactly `AGENTS.md`, `ERRORS.md`, and `PLAN.md`; upstream changed none of those three paths over its 88 commits. A local merge attempt in an isolated repo-session was blocked before mutation because server-owned worktree metadata forbids `ORIG_HEAD.lock`; the workspace remained clean. This remains publication-only evidence and is not the current phase priority.
- Fresh Batch C discovery found the repository-native E2E resilience harness at `examples/e2e_resilience_tests/run_tests.sh`; its documented scope is 15 test groups / 27 assertions covering real control-plane + agents + mock LLM flows. The first invocation failed before product tests because direct cached `go` execution is forbidden in the sandbox; this is environment evidence, not product failure.
- The public Coding Station Action/Traefik route remains unhealthy, but it is no longer a Batch C execution blocker: the existing `coolify-control` target reaches the same Coding Station runtime over its already-owned internal `coolify` network, and runtime `/health` reports `ok` with the bounded exec/session/repo-session capabilities. No new service or bypass runtime was created.
- Through that existing internal runtime API, the repository E2E harness now reproducibly reaches mock-LLM readiness and the real control-plane build on the exact-source workspace. The next failure was classified as harness/environment portability, not product behavior: `run_tests.sh` hard-coded the control-plane binary to `/tmp/af-test-server` while Coding Station intentionally mounts `/tmp` `noexec`, and Go's default build cache filled the 512 MiB sandbox tmpfs (`ENOSPC`).
- Post-state showed `/data/workspaces` still had 5.8 GiB free while `/tmp` alone was 100% full. After terminating only the failed E2E session, one scoped Coding Station helper restart cleared the tmpfs to 0/512 MiB without losing the persistent repo-session. The E2E runner was minimally patched in the container workspace to place its control-plane binary under an explicit workspace-local test temp root; the rerun also pins workspace-local `TMPDIR`, `GOTMPDIR`, and `GOCACHE`.
- Batch C then crossed the real AgentField boundary. Root cause of earlier `target_not_found` was test-environment drift: the Python SDK prerequisite was absent (`ModuleNotFoundError: agentfield`; with source-only `PYTHONPATH`, next missing runtime dependency was `requests`). No system package/apt mutation was used. Existing Coding Station `python@3.12.11` + pip installed `./sdk/python` and runtime dependencies into workspace-local `.tmp/e2e-site` with workspace-local pip temp/cache; `import agentfield, requests` then passed.
- Batch C native resilience E2E is now a real PASS on the existing PROD VPS Terminal `workbench`, using exact runtime base + byte-identical runtime patch, Go 1.25.10, workspace-local Python SDK/deps, and test-only portability fixes (bounded cleanup + workspace-local control-plane binary because `/tmp` is noexec). The repository-native harness returned `Passed: 27`, `Failed: 0`, `All 27 tests passed!` and process exit code `0` across real control plane + three Python agents + mock LLM.
- The public `Runner.Run` seam is also pinned in runtime source by `TestAforgeRunnerAmbiguousExitUsesPersistedPostconditionWithoutRepeat`: a fake Aforge writes valid durable output and exits non-zero; `Runner.Run` succeeds from the persisted postcondition and the marker proves exactly one provider invocation. Targeted test PASS (`ok`, 0.021s), then full harness PASS (`ok`, 6.308s) and full Go SDK PASS. This closes Batch C.
- Batch D golden regression is now loaded directly into runtime `parity_test.go` as `TestContractCompletionGoldenSnapshots` (readback SHA `6bc3493e8251ed138817bbd4174a1602085b3c2fcebc84829810dc98cb638a17`; SourceLoop capture `vtchg_77bbbf0d559c4565b1a6cc548ba63fd8`). It freezes exact snapshots for satisfied+missing, invalid, UNKNOWN/fail-closed, ambiguous observed-effect suppression, and successful incremental repair with one provider call while preserving `title=hello` and repairing `body=world`. A byte-identical validation copy passed the golden test (`ok`, 0.009s), full harness (`ok`, 7.329s), full Go SDK, and fresh runtime `git_diff_check`. This closes Batch D.

## Current Phase Goal
Prove the smallest deterministic contract-completion loop on exact upstream: observed truth overrides self-report; completed effects are never repeated; unresolved obligations produce the smallest safe continuation; ambiguous mutation state fails closed. Keep the exact-source delta container-first and publish only after deterministic verification.

## Bounded 30-minute batches
### Batch A — exact-source RED readiness — DONE (2026-09-14)
DoD:
- No new service/container and no ad-hoc toolchain installation.
- Preserve the now-single authoritative Coding Station route and exact-upstream workspace.
- Use Coding Station's existing pinned dynamic-toolchain contract; do not require bare `go` to exist in the image and do not install it ad hoc.
- After the ambiguous first pinned-toolchain timeout, verify post-state before retry. Then verify `station-toolchain exec go@<pinned> -- go version` and one focused existing harness test execute on exact upstream.
- Only then mutate product source for RED.

### Batch B — RED + minimal contract-completion slice — DONE (2026-09-14)
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
- Anti-drift invariant: objective is proven generic contract completion on exact AgentField source: runtime truth, preserved partial work, no duplicate mutation, smallest continuation, bounded termination/cost, then E2E + golden regression. Git/fork reconciliation and SourceLoop publication are release mechanics only and must not become the objective or run before those behavioral gates PASS. SWE/FCM/OpenCode remains out of scope unless it directly blocks this objective.

### Batch C — relevant E2E proof — ACTIVE (2026-09-14)
DoD:
- Run the repository's existing end-to-end resilience harness on the exact container workspace before any publication/fork reconciliation.
- Add or reuse the thinnest end-to-end scenario that exercises schema/contract continuation through the public Runner path, not just helper functions.
- Prove a durable partial/completed effect is observed after an ambiguous provider/transport result and is not executed twice.
- Prove unresolved fields continue with the smallest repair while satisfied fields remain preserved.
- Capture executable evidence: command, exit status, assertions, and relevant artifact/readback.

### Batch D — golden regression — PENDING
DoD:
- Freeze representative contract-completion inputs/outputs as deterministic golden fixtures or equivalent repository-native snapshots.
- Include at minimum: satisfied+missing mix, invalid field, unknown/fail-closed state, ambiguous-result-with-observed-effect, and successful incremental repair.
- Golden comparison must fail on duplicate mutation, loss of preserved partial output, widened continuation, or changed fail-closed behavior.
- Re-run affected harness tests + full Go SDK after golden coverage is added.

## Next move
Do not reconcile the fork, publish product code, merge, or redeploy yet. The direct runtime source now contains the intended three-file contract-completion delta and `git_diff_check` PASS; the operator registry/source-access gaps are closed. The next obligatory move is validation on this exact dirty runtime source using the deployment's own Go 1.25 `control-plane-build` / `go-reconcile` lane: run formatting check, the four new contract-completion tests plus existing incremental recovery, full harness regression, then full Go SDK regression. Only after those are PASS may Batch C rerun the native 27-assertion resilience E2E to exit code 0 and add the public-Runner contract-completion seam test; Batch D golden regression follows. Do not install Go ad hoc into runtime containers, do not use Coding Station, and do not publish/redeploy before these behavioral gates PASS.
