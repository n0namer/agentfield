# AgentField Project Plan

Last verified: 2026-09-15

## North Star
Build AgentField as the reliable execution and recovery plane for agents: bounded contracts are validated against CURRENT state, partial work is preserved, and the smallest safe continuation completes only what remains. This must improve reliability and cost for weaker/cheaper models without re-running already-completed mutations.

## Project decisions
- `PLAN.md` is the project/design SoT for North Star, phase goal, bounded batches, DoD, decisions, drift and next move.
- Runtime/readback owns actual state; this file must not claim a loaded change without runtime evidence.
- Code debugging, implementation, and validation are direct-target/container-first. Do not place Coding Station or any other helper/control-plane proxy in the critical path. GitHub/CI/deploy is publication/release boundary, not the inner coding loop.
- SourceLoop is a project-agnostic canonicalization/replay layer, not an SWE-specific feature. SWE-AF is the first reference implementation; the same contract must later onboard VPS Terminal, AgentField runtime, and other existing projects without redesigning the mechanism.
- SourceLoop canonicalizes only an exact delta already verified in DEV. Its durable identity is a logical patch record with provenance (`project`, canonical repo, runtime source root, exact base SHA, capture/change IDs, validation evidence, ordered patch position, canonical commit generation), not a Git branch and not a single immutable Git SHA.
- Branch model is intentionally small: one long-lived `dev` integration line per project, one accepted `main` line, and PROD as an immutable exact SHA/tag. `sourceloop/<patch>` branches are temporary migration/recovery evidence only and must not be the steady-state patch registry. A temporary replay/conflict branch/workspace may exist only while an upstream replay is unresolved, then is deleted/retired after `dev` advances.
- DEV is a moving integration line; PROD is an immutable tested snapshot at an exact commit. Canonicalization (`VERIFIED -> CANONICAL_ON_DEV`) is separate from release promotion (`dev -> main -> RELEASED`); no live runtime edit may auto-promote to production. PR, when used, is a promotion/review boundary for a tested dev batch, not one PR per logical patch.
- Upstream sync is replay/rebase of the ordered ACTIVE logical patch stack onto a fresh upstream base in a temporary workspace/branch. Each patch independently becomes CLEAN/REPLAYED, SUPERSEDED_BY_UPSTREAM, or CONFLICT requiring bounded repair + its regression test. After all active patches pass, advance the single `dev` line atomically to the replayed stack and retire the temporary replay surface; full-fork blind merge is not the default.
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
- Fresh 2026-09-15 runtime readback confirms the same workforce container `8431ce530b15f006c1422ff0b10724f70fabc420b8279a37352870287412e1a2` is running/healthy; no replacement container is required. `swe-planner` registered at 11:06:39Z, but its immediate status update and every observed two-minute lease refresh still fail `409 stale_agent_instance`.
- Fresh control-plane logs independently prove the current mismatch: `current_instance_id="4d4ac40123884c6a8aa69f0fc13678c4"` while `incoming_instance_id=""`. Therefore the stale-instance gate is still OPEN; storage and the stale guard remain unchanged while diagnosis stays on the installed SWE SDK/build provenance.
- Fresh source comparison identifies DESIGN_RUNTIME_DRIFT in the Go installer. Fork `main` and upstream `Agent-Field/agentfield:main` both use blob `f93c8cf459890c79de994e1532828b91e72c2d75` for `control-plane/internal/packages/gointerp.go`; that canonical file already supports explicit `AGENTFIELD_GO_REPLACE`. Live runtime file SHA-256 `6187243a10c405f3ae5ea89de8d6c2fa38d28d4d4252d4a9130ef85d3de7f758` additionally auto-translates the inherited `go.work` via `applyGoWorkspaceOverrides(...)` and forces internal build `GOWORK=off`. Treat the automatic workspace translation as a bootstrap experiment, not accepted architecture, until an upstream-native path is disproven.
- BMAD anti-drift refresh: current BMAD renamed `bmad-quick-dev` to `bmad-build` (old IDs are compatibility shims) and defines Build as the official implementation loop. Use `bmad-build` semantics for this phase: evidence-first, smallest scoped implementation, then focused review/verification. External debugging/TDD/verification skills are method input only; do not install another framework without a proven capability gap.
- Batch E stale-instance/bootstrap gate is now CLOSED by same-container evidence. Before reinstall, `/afhome/packages/swe-planner/bin/swe-planner` embedded only SDK pseudo-version `v0.0.0-20260723130821-20955b2637b4` and no local SDK path. The existing package was reinstalled in the same workforce with canonical `AGENTFIELD_GO_REPLACE=github.com/Agent-Field/agentfield/sdk/go=/core-src/agentfield-runtime/sdk/go` and invocation-boundary `GOWORK=off`; resulting installed `go.mod` contains that single explicit replace and binary build metadata records `=> /core-src/agentfield-runtime/sdk/go (devel)`. Binary SHA-256 changed from `0e058732e0664debd1ae0fe68aea6fa9863e8df4f08bce705100e72ca5b9e5c1` to `7d8824a441ccff29c4f07113135eb3c8122ce7b049a6d8d54da28c30d53c8b0d`.
- The newly started same-container `swe-planner` registered at 11:46:43Z without the previous immediate `409 stale_agent_instance`. Native node readback now reports `state=active`, `lifecycle_status=ready`, `health_status=active`, authoritative `instance_id=433d42e1a1e540f1aa47525af2dbce22`; native discovery at 11:48:09Z sees one active `swe-planner` with 31 reasoners, including public `implement_issue`. This proves the SDK provenance fix and closes the stale-instance gate without weakening storage/stale guards or canonicalizing automatic `go.work` translation.
- SourceLoop product-source bootstrap is now functionally proven for future live patches. `agentfield-dev-runtime-capture` registry revision 5 maps `/core-src/agentfield-runtime` to `n0namer/agentfield:main` with `configured_sha`; runtime `.source-commit` is pinned to the exact live base `c0923acdfca043c2c07e3d34daaa09e2a7e41d38`. A bounded canary exact patch produced `vtchg_94d2145eb9144827aba5716afccda3a2` with immutable base, repo path, `vtcap_94d2145eb9144827aba5716afccda3a2` artifact and verified artifact readback. The canary was explicitly REJECTED and deleted. Therefore new live patches can now carry provenance/capture; the 17 older PENDING journal events predate this binding and must not be mistaken for captured publication artifacts.
- Batch E native non-mutating smoke gate is CLOSED. After isolating full-plan LLM/tool-use noise, `swe-planner.run_product_manager` was invoked natively through the control plane with `max_turns=1`, `permission_mode=plan`, repo `/afhome/batch-e-smoke-repo`, and artifacts outside product source. Execution `exec_20260915_122235_69og7el9` persisted terminal `succeeded` in 19.531s with a structured PRD result. This proves discovery/execution/result persistence without product-source mutation; the earlier broad `plan` attempts remain diagnostic evidence only, not the smoke gate.
- SWE-AF itself is now SourceLoop-bound separately: existing target `agentfield-dev-workforce` revision 30 maps `/src/swe-af` to `n0namer/swe-af:main` with `.source-commit=6f5b4382e6231721f60be7045b9d91fd85e34fb5`. New live test/fix patches therefore carry repository/base provenance instead of being anonymous runtime edits.
- SourceLoop canonicalization lane proof is PARTIAL but materially advanced for the `resume_build_id` recovery patch. Fresh journal readback shows the durable edits on `go/internal/issue/build.go`, `go/internal/issue/build_test.go`, and `go/internal/node/register.go` captured with repository `n0namer/swe-af`, exact base `6f5b4382e6231721f60be7045b9d91fd85e34fb5`, and per-change `vtcap_*` artifacts; unrelated dirty `PLAN.md`, planning files, runtime artifacts, and `.source-commit` were explicitly excluded.
- A canonical no-PR Git representation now exists on branch `sourceloop/resume-build-id-6f5b438`, rooted exactly at `6f5b4382e6231721f60be7045b9d91fd85e34fb5`, with head `8969c50a92ff4df737d575fd27d7b07995441b4f`. GitHub compare reread shows exactly the same three files and same hunks as the live runtime diff; no bootstrap/diagnostic experiment was included. Fresh runtime validation passed `TestResumeBuildIDReusesExistingIssueWorktree` (`ok`, 0.213s) and full `go test ./internal/issue ./internal/node -count=1` (`ok`, 12.382s / 0.087s).
- The current SourceLoop reference proof is SWE-only (`/src/swe-af` → `n0namer/swe-af` via `agentfield-dev-workforce`), but the mechanism is project-agnostic. Do not turn unrelated VPS Terminal deployment/runtime health into the objective; only SourceLoop-owned code/bindings that directly block the universal canonicalization contract are in scope. No PR/merge/deploy/redeploy is allowed in this lane.
- The first durable replay proof remains `resume_build_id`: exact runtime base `6f5b4382e6231721f60be7045b9d91fd85e34fb5`, fresh targeted/full tests PASS, captures exist for the three KEEP files, and canonical branch `sourceloop/resume-build-id-6f5b438` rereads as exactly those runtime hunks. The typed SourceLoop journal still reports those events `PENDING`; that status is not success by itself.
- A second independent durable patch is now also verified and canonicalized from the same exact base: OpenCode Architect now uses `SchemaMode=incremental` plus runtime-owned `OPENCODE_CONFIG_CONTENT`, with regression `TestArchitectOpenCodeUsesIncrementalSchemaContract`. Fresh runtime targeted test PASS (`ok`, 0.009s) and full `./internal/roles/planning` PASS (`ok`, 0.027s). SourceLoop captures `vtchg_ba59190401384e4bb13d15ac01497726` (`planning.go`) and `vtchg_a29f5b3108e94374b6a4ca9ce10b50a3` (`planning_test.go`) both bind to `n0namer/swe-af`, exact base `6f5b4382e6231721f60be7045b9d91fd85e34fb5`. Canonical branch `sourceloop/opencode-architect-schema-6f5b438`, head `404082d0ec93f57d73bc2c118827959a1756ab44`, rereads as exactly two files / 37 added lines matching the live runtime diff. Upstream search found no existing `TestArchitectOpenCodeUsesIncrementalSchemaContract`. This proves the replayable canonicalization pattern twice; what remains unproven is automatic journal status linkage from capture IDs to canonical commit without PR.
- Fresh CURRENT linkage proof on the second SWE patch failed closed before mutation: `sourceLoopAction(candidate)` with change `vtchg_ba59190401384e4bb13d15ac01497726`, exact canonical branch `sourceloop/opencode-architect-schema-6f5b438`, and commit `404082d0ec93f57d73bc2c118827959a1756ab44` returned `writeback_candidate_evidence_required: CANDIDATE requires branch, commit_sha and pr_number`. Therefore no-PR capture→commit linkage is a proven CURRENT capability gap, not missing SWE evidence. Do not create a PR merely to satisfy this transport contract.
- Container-first implementation discovery found the existing SourceLoop owner runtime target `vps-terminal-dev-gateway` with direct readable source at `/app/gateway/live-aci.mjs`; current implementation still enforces `branch + commit_sha + pr_number`. No root/nearest `AGENTS.md` or `ERRORS.md` exists in that runtime source. The repository test file already covers the writeback state machine, but a full run currently has unrelated environment failures, so the correct regression gate is a new isolated unit test for `PENDING + exact PASS validation -> CANDIDATE(branch, commit_sha, no PR)`.
- SourceLoop owner-path diagnosis is now specific. `vps-terminal-dev-gateway` remains image/read-only (`/app/gateway` writes fail `EROFS`), so no redeploy was used to make it writable. Its known helper-runtime defect from `vps-terminal/ERRORS.md` recurred because `live_patch_runtime` was omitted during registry normalization; authoritative target revision 11→12 restored `live_patch_runtime=node` with readback.
- The existing DEV workbench lane was recovered without a new container. The active DEV workbench was distinguished from the separate production workbench by exact Compose project/image selector. Exact source under `/workspace/vps-terminal-complete` is root-owned/read-only; existing writable mirror `/workspace/vps-terminal` is node-owned. Before coding, only `gateway/live-aci.mjs` and `gateway/test/live-aci.test.mjs` were reconciled byte-for-byte from the exact tree into the writable mirror; SHA readback matched exactly. The runtime Node helper `gateway/live-aci-node-helper.mjs` in the writable mirror matches loaded gateway helper SHA-256 `c8e8ac8484e0b37e9bbee4ded605e4a179fe056dc525f55791effd6fe8968c0c`.
- Workbench Target Registry is now revision 16, scoped to the active DEV workbench, `workspace_root=/workspace/vps-terminal`, `live_patch_roots=[/workspace/vps-terminal/gateway,/workspace/vps-terminal]`, and `live_patch_runtime=node`. Typed File ACI still returns `live_patch_invalid_output` even after exact helper/root/selector recovery, so typed mutation remains unavailable; direct opaque mutation remains mediation-blocked and is not bypassed.
- The isolated container-first regression `gateway/test/source-loop-no-pr.test.mjs` now has a valid TDD cycle on the writable DEV mirror. A temporary test-only Makefile removed inherited broken `NODE_OPTIONS`; RED then failed specifically with `writeback_candidate_evidence_required: CANDIDATE requires branch, commit_sha and pr_number`. Only after that intended RED, one stale-safe exact-text patch changed the single CANDIDATE guard to require only `branch + commit_sha`; File ACI readback SHA is `75c0e14992fa8e3d54743c70c3ecaecda57f6bfe022cb3eb66fa0f0484a607b1`, SourceLoop journal change `vtchg_05d3494e0fde4759ba4e541a42a0522d`.
- GREEN evidence is container-native and hermetic: `make check` with only test-boundary `NODE_OPTIONS` removal passed `node --check gateway/live-aci.mjs`, isolated no-PR regression `1/1 PASS`, and repository-native `gateway/test/live-aci.test.mjs` `6/6 PASS` (exit 0). The generic typed `node_check` still fails before checking source because of the same missing injected `memory-emitter.mjs`; that FAIL is environment contamination, not product regression. The temporary Makefile was deleted and absence reread was verified.
- The writable DEV mirror File ACI was recovered without a new container by switching workbench helper runtime to existing `/usr/bin/python3`; authoritative workbench registry is revision 17 with exact DEV selector, workspace `/workspace/vps-terminal`, roots `[/workspace/vps-terminal/gateway,/workspace/vps-terminal]`, `live_patch_runtime=python`. Gateway remains revision 12 with `live_patch_runtime=node`. Registry state is preserved in `n0namer/vps-terminal:sourceloop/registry-bindings-20260915`, current head `40b322ef3029edeaa3deda3047cc071dc8a1ccf2`.
- Exact-base provenance for the byte-tested SourceLoop owner generation is now proven, not inferred. The pre-patch `/workspace/vps-terminal-complete/gateway/live-aci.mjs` Git blob was computed hermetically as `17e03156a189b37605cb49d2ebeadd4f3984e79a`, exactly matching GitHub blob `gateway/live-aci.mjs` at commit `97a1a675d5d67e6045d140f79664819c2736c8ff`. Therefore logical patch generation 1 is canonicalized on `n0namer/vps-terminal:sourceloop/no-pr-linkage-97a1a675`, exact base `97a1a675d5d67e6045d140f79664819c2736c8ff`, head `f3b98dc2c4b84700b45ee3f40047a0bded313554`. GitHub compare reread shows exactly two files: the one-line CANDIDATE guard change and `gateway/test/source-loop-no-pr.test.mjs`; no unrelated delta is present.
- The older/newer safety branch `n0namer/vps-terminal:sourceloop/no-pr-candidate` remains a logical replay candidate on a newer repository generation, not the exact tested generation. Treat `no-pr-linkage-97a1a675` as generation-1 evidence and validate any newer replay with the same isolated regression + repository-native SourceLoop tests before accepting that generation. This is the intended SourceLoop logical-patch model: patch identity persists while base/commit generation changes.
- CURRENT runtime linkage remains intentionally unchanged because no deploy/reload occurred: immediately after source GREEN, `sourceLoopAction(candidate)` for SWE change `vtchg_ba59190401384e4bb13d15ac01497726` still failed closed with `CANDIDATE requires branch, commit_sha and pr_number`. Source-level fix is VERIFIED; first-class runtime linkage is still PARTIAL until a later allowed release boundary loads the verified owner change.
- Branch-migration rule: stop creating one long-lived `sourceloop/<patch>` branch per new durable patch. New VERIFIED changes must enter the logical patch registry and then the single project `dev` integration line after replay/tests; a temporary patch/replay branch is permitted only while first-class linkage or a replay conflict is unresolved. Existing `sourceloop/resume-build-id-6f5b438`, `sourceloop/opencode-architect-schema-6f5b438`, `sourceloop/no-pr-linkage-97a1a675`, and `sourceloop/no-pr-candidate` remain legacy evidence until their logical identities/generations are represented durably elsewhere; do not delete them before that migration is verified.
- SWE reference migration is now materially executed: `n0namer/swe-af:dev` advanced from `dc39e0ffb990c766f1bba43263a2ec9ffe184443` to `cc48bbe7143c519327d9c8b6f7d49265c69df63d` without PR/deploy. Logical patch `resume_build_id` was semantically replayed into the single `dev` line; targeted regression and `go test ./internal/issue ./internal/node -count=1` PASS. Published product blobs exactly match the tested replay bytes: `build.go=31ae28948245a48ec71db447e0391581f58e9421`, `build_test.go=57604e59e959060b12a90d4ddb164c28826836a5`, `register.go=843cf61460e96b570e39f4a8c925158adcf8dd17`. The old Architect/OpenCode logical patch is `PARTIALLY_SUPERSEDED / HOLD`: current `dev` already centralizes OpenCode incremental schema mode; its removed overlay API is not resurrected without fresh evidence.
- SWE upstream cutover is now source-level DONE. Fresh upstream `Agent-Field/SWE-AF:main@311f376a2f12df01134acd80384d599dd8039178` was reconciled through a temporary replay surface, not blind-rebased across the 336-vs-4 divergent history. Only one textual upstream conflict existed (`go/internal/node/register.go`); the merge kept upstream Pro/Furrow registration semantics and downstream `resume_build_id`. The parallel coding cutoff was replayed semantically as one logical patch (nested-module coder CWD + regression + prompt guardrails), explicitly excluding superseded `coderEnv`/venv/OpenCode-overlay baggage. Final `n0namer/swe-af:dev` product cutover commit is `0944bdfbfdb727caad12455085a561940306e29b`; upstream `311f376...` is its ancestor and the promoted tree `1642073e66812f25e7e15db25fdd83505ae1da9d` exactly matched the independently tested temporary tree. Fresh tests on exact published `dev@0944bdfb...` PASS: `go test ./internal/issue ./internal/node ./internal/roles/coding ./internal/prompts/coding -count=1`; `git diff --check` PASS.
- SWE runtime cutover is now DONE. Live `/src/swe-af` was switched to exact `dev@5b54a1dd4c579fd41ddf141e20dd46259519322e` after freezing the old-generation dirty diff (`/tmp/swe-oldgen-cutover-20260915.patch`, SHA-256 `cdb5f3988f9ae606b304404b33837ac42ea52309355191913e7de6acfd32511a`) and classifying every tracked old-generation delta. Live affected validation PASS: `go test ./internal/issue ./internal/node ./internal/roles/coding ./internal/prompts/coding -count=1`. Durable `agentfield-dev-workforce` registry advanced rev30→31 changing only SourceLoop writeback branch `main→dev`; `.source-commit` equals the live HEAD. Ephemeral canary capture `vtchg_01e9dbec383d4df99e0207777b5c3f4b` independently proves new journal provenance `base_branch=dev`, `base_commit=5b54a1dd...`; the canary was deleted. Historical captures remain on their original old base and are not rewritten.

## Universal upstream synchronization architecture

This is the default synchronization architecture for every project that carries local/downstream changes while consuming an external or canonical upstream. Project-specific deviations require an explicit SoT decision; they are not inferred from repository history.

### Goal and invariants
The goal is to update to a fresh upstream without losing verified downstream intent, without turning Git branches into the patch database, and without confusing source publication with runtime loading.

1. **One moving integration line.** Each project has one long-lived downstream `dev` integration line. `main` is the accepted/release line or clean upstream mirror according to that project's SoT; PROD is always an exact accepted SHA/tag, never a moving branch name.
2. **Logical patch identity outlives Git SHA.** Durable downstream intent is represented as a logical patch record. Git commits are materialized generations of that patch on a particular base. A rebase/replay changing the commit SHA does not create a new logical patch.
3. **Exact provenance before mutation.** Every captured change records project/repository, runtime source root, exact base SHA, base branch, capture/change IDs, affected paths, and validation evidence. Unknown base or ambiguous owner fails closed.
4. **Replay intent, not history.** Upstream synchronization replays the ordered ACTIVE logical patch stack onto a fresh upstream-derived base. It does not blindly rebase/merge an arbitrarily divergent fork history.
5. **Semantic conflict resolution.** Textual conflict resolution must preserve current intent and current upstream contracts; stale implementation baggage is not resurrected merely because it existed in an older patch generation.
6. **Differential verification.** A failure seen after replay is attributed to the patch only when it does not reproduce on the corresponding clean baseline. Baseline/environment failures are recorded separately and do not silently block or falsely fail the patch.
7. **Source and runtime are separate states.** Publishing a new `dev` generation does not imply the DEV runtime loaded it. Runtime cutover requires exact source identity readback plus post-cutover validation before SourceLoop provenance advances.
8. **Historical evidence is immutable.** Old captures remain bound to the base on which they were observed. Never rewrite old provenance to make history look cleaner.
9. **No branch explosion.** Permanent `sourceloop/<patch>` branches are not the patch registry. Temporary replay/conflict branches or worktrees exist only for unresolved integration work and are retired after accepted integration.
10. **Release is a separate promotion.** `VERIFIED -> CANONICAL_ON_DEV` and `CANONICAL_ON_DEV -> RELEASED` are distinct gates. PR/CI/deploy may be used at the release boundary but are not prerequisites for container-first capture/replay.

### Minimal logical patch record
The first implementation should keep this deliberately small; do not build a new workflow engine.

- `logical_patch_id`: stable identity.
- `project`, `repository`, `runtime_root`.
- `intent`: one-sentence behavioral invariant.
- `state`: `ACTIVE | SUPERSEDED | HOLD | RETIRED`.
- `order`: position in the active replay stack when ordering matters.
- `generation[]`: `{base_sha, materialized_commit_sha, capture_ids, affected_paths, validation_evidence, result}`.
- generation `result`: `REPLAYED | SUPERSEDED_BY_UPSTREAM | CONFLICT_REPAIRED | HOLD`.
- `regressions`: deterministic tests/checks that prove the behavior.
- optional `depends_on[]`: only when replay order is semantically required; avoid speculative dependency graphs.

A branch name, PR number, or commit SHA alone is never sufficient logical-patch identity.

### Synchronization state machine

`OBSERVE -> FREEZE_GENERATION -> BUILD_FRESH_BASE -> REPLAY -> VERIFY -> ADVANCE_DEV -> RUNTIME_CUTOVER -> PROVE_NEW_PROVENANCE -> RESUME`

**OBSERVE**
- Read project SoT/`AGENTS.md`, exact `dev`, exact upstream, runtime `HEAD`, `.source-commit`/equivalent, dirty state, active mutators, and current SourceLoop binding.
- If source owner, runtime identity, or base is ambiguous: read-only diagnosis only.

**FREEZE_GENERATION**
- Define a cutoff for the current runtime generation.
- Finish/capture only already-started atomic work; new work after cutoff belongs to the next generation queue.
- Preserve exact dirty delta before destructive source switching. Classify every changed path `KEEP | DROP | HOLD/UNKNOWN`.

**BUILD_FRESH_BASE**
- Fetch/read the new upstream exact SHA.
- Use a temporary worktree/branch from the project's accepted integration base; never mutate the dirty live checkout just to discover conflicts.
- For legacy repositories without a mature patch registry, run merge/replay analysis to establish a one-time reconciled baseline, then switch to logical-patch replay for subsequent syncs.

**REPLAY**
- Apply ACTIVE patches in order.
- For each patch: clean apply -> `REPLAYED`; equivalent behavior already upstream -> `SUPERSEDED_BY_UPSTREAM`; conflict -> semantic repair preserving current intent; no current requirement/evidence -> `HOLD` rather than resurrection.
- Run the patch's regression immediately after its replay before layering more patches where practical.

**VERIFY**
- Run affected deterministic regressions and the smallest canonical package/suite checks.
- Run `diff --check`/format/static checks appropriate to the project.
- When a broader suite fails, compare the same failing test on the clean fresh baseline before attributing it to replay. Record flaky/environment/baseline debt separately.
- Before publication, tested source bytes/tree identity must equal the source being promoted.

**ADVANCE_DEV**
- Fresh-read remote `dev`; abort/replan on concurrent movement.
- Advance only the single `dev` integration line to the verified replay result. Temporary replay surfaces are retired after readback.
- Parallel post-cutoff work remains queued and is replayed as the next generation; it is never silently folded into the previous cutoff.

**RUNTIME_CUTOVER**
- Prove no active mutator will be orphaned.
- Preserve any remaining old-generation dirty delta, then switch the existing runtime source to exact accepted `dev` using the authoritative owner route; do not create replacement infrastructure merely for sync.
- Read back runtime `HEAD` and source-identity marker; they must exactly equal the accepted `dev` SHA.
- Run the affected runtime validation again on the actual live source.

**PROVE_NEW_PROVENANCE**
- Only after runtime readback passes, update SourceLoop binding/base branch to the new generation.
- Produce one bounded ephemeral capture/probe proving new journal provenance resolves to the new branch + exact base SHA, then remove the probe.
- Never relabel historical captures.

**RESUME**
- Normal container-first programming resumes on the new generation. The next durable change is a new logical patch/capture against the new `dev` base.

### Parallel-development generation rule
A sync must not require stopping all development for a long interval. Use a generation cutoff:

- generation `N`: all verified/captured work at or before cutoff; replayed onto fresh upstream now.
- generation `N+1`: work created after cutoff; remains queued while `N` is replayed and is applied only after the new `dev` baseline is accepted.
- never let a moving dirty runtime continuously expand the in-flight replay scope.

This converts parallel development from an unbounded merge race into two bounded queues.

### Conflict policy
Prefer, in order:
1. current upstream contract + current downstream behavioral invariant;
2. existing project-native replacement API/implementation;
3. minimal semantic adaptation plus regression;
4. `HOLD` when current intent cannot be proven.

Do not choose `ours`/`theirs` mechanically when the implementation architecture changed. A semantic replay may intentionally drop historical implementation details while preserving the tested behavior.

### Legacy bootstrap rule
For a project first adopting this architecture:
- do not attempt to reconstruct every historical commit as a logical patch;
- identify the current accepted downstream tree, fresh upstream, known durable local behaviors, and UNKNOWN drift;
- perform one bounded legacy reconciliation to create a tested `dev` baseline;
- establish logical patch identities only for durable behavior that must survive future upstream updates;
- from that baseline onward, all new durable changes follow normal capture -> logical patch -> replay generations.

### Release and rollback
- `dev` is integration, not production identity.
- release selects one exact tested source SHA/tag/artifact and records provenance.
- rollback moves the release/runtime pointer to a previously accepted exact SHA; it does not reverse or rewrite SourceLoop history.
- build/release provenance should retain exact source/material identities so the deployed artifact can be traced to the accepted source generation.

### 80/20 implementation priority
Do these first because they eliminate most operational risk with little new machinery:
1. enforce exact `{repo, runtime_root, base_branch, base_sha}` binding + immutable capture provenance;
2. add first-class `logical_patch_id` with generations and regression evidence;
3. standardize the temporary replay + differential baseline test + single-`dev` advancement procedure;
4. standardize runtime cutover readback + one ephemeral provenance probe.

Defer until evidence demands them: complex patch-dependency solvers, automatic conflict synthesis, permanent patch branches, per-patch PRs, new synchronization services, or organization-wide CI orchestration.

### Evidence basis / design rationale
- Empirical CI research supports frequent integration and short-lived divergence; merge conflicts are costly and error-prone, so keeping one integration line plus temporary isolated replay surfaces reduces coordination state rather than multiplying permanent branches.
- Change-propagation research shows that changes have ripple effects and long-term propagation channels; replay therefore verifies behavioral intent and affected regressions rather than treating textual patch application as correctness.
- Empirical work on unrelated CI failures supports differential baseline reproduction before attributing a failing check to the current patch.
- SLSA provenance principles support recording exact source/material identities and separating source revision/provenance from later build/release artifacts.
- BMAD current method is used as process shape, not infrastructure: `bmad-architecture` for explicit architecture/validation, `bmad-build` for observe -> smallest implementation -> verify, and evidence-backed review before declaring a gate closed. No separate BMAD runtime is required.
- External skill review reinforced only reusable primitives: native Git worktrees/base detection/recovery, explicit integration-state inspection, atomic changes, and verification-before-completion. Project SoT overrides generic Gitflow/PR conventions where they conflict with the single-`dev`, container-first model.

## AgentField upstream cutover — 2026-09-16

**Status: DONE — upstream replay, single-`dev` publication, runtime cutover, deployment binding, and new-generation SourceLoop provenance are all verified. `main` remains the separate accepted/release line and was not rewritten.**

AgentField is now the second concrete proving ground for the universal synchronization architecture, after SWE-AF. The same architecture applies, but current runtime acceptance remains authoritative: do not sacrifice the active native SWE runtime gate merely to make Git history look current.

### Fresh observed state

- Canonical fork: `n0namer/agentfield`.
- Current downstream integration candidate: `dev@c0923acdfca043c2c07e3d34daaa09e2a7e41d38`.
- Current upstream: `Agent-Field/agentfield:main@2180e30c7f1619a652635cf8e5038c36f4d0dc3e`.
- User-observed divergence is 5 downstream commits and 91 upstream commits from the old common line; treat this as divergent history, not a fast-forward distance.
- Exact five committed downstream patches are:
  1. `948d20f41909e44cf3a1480a83377f10caaa3e2e` — repair persisted agent DID derivation paths on restart.
  2. `e07dfcae940060d4e4a123f5aea7f2fbffe1a024` — restart-regression JSON assertions.
  3. `4d337c1ae5104418311fcba414a1c2f85c2abb89` — preserve agent signing key across restart.
  4. `b160245833ec51f5296905c32e49260a62c76e26` — Go SDK agent process identity / instance-id propagation plus in-flight cancellation on shutdown.
  5. `c0923acdfca043c2c07e3d34daaa09e2a7e41d38` — restart generation and stale-state recovery hardening.
- Existing fresh replay base already exists at `tmp/agentfield-replay-2180e30c@2180e30c7f1619a652635cf8e5038c36f4d0dc3e`; do not create another permanent branch for the same purpose.
- Existing AgentField DEV stack is compose project `edshqtkwskg3lrczekhcmd71`; relevant existing containers include `control-plane`, `workforce`, `runtime-capture`, and exited reusable `control-plane-build`. No new container/service is required for synchronization.
- Authoritative runtime source checkout is `/core-src/agentfield-runtime` in the existing AgentField source volume. Readback proves `.git/HEAD = c0923acdfca043c2c07e3d34daaa09e2a7e41d38`.
- Runtime-capture chain `runtime-capture/agentfield/c0923acdfca0` preserves post-commit live SDK work from exact base `c0923acd...`. Latest observed capture head is `3f433227fb27c04bf2fa817fc57b376da60478b5`; captures include Go harness durable-schema completion behavior (`schemaOutputDir`, `watchStableSchemaOutput`) and its regression surface.
- Upstream exact-symbol search does not currently contain `InstanceID`, `repairLoadedAgentDerivationPaths`, or `watchStableSchemaOutput`; therefore those behaviors are **not proven superseded** and must enter replay classification rather than being dropped automatically.
- Current old-generation SDK baseline is GREEN on exact runtime source through the existing `agentfield-dev-workforce` Go toolchain: `go test ./agent ./harness ./types -count=1` PASS from `/core-src/agentfield-runtime/sdk/go`.
- A broad control-plane `./internal/services` filtered test invocation exceeded the 120 s execution bound without a test verdict; classify as `VALIDATION_BLOCKER/TIMEOUT`, not product failure. Subsequent validation must use narrower named restart/DID/node-status regressions first.
- Direct generic Git workspace creation inside `agentfield-dev-runtime-capture` is blocked by enforced operator mediation (`REVIEW_REQUIRED: opaque_or_unknown_mutation`). Do not bypass this with Coding Station or a new helper container. Use an existing typed/allowed workspace route, or form the replay candidate source-side and validate it through the existing shared `/core-src` volume before any runtime cutover.

### AgentField logical patch stack for replay

Treat the five committed changes and the runtime-capture generation as behavioral patches, not as six permanent branches:

- `AF-P1 restart-did-derivation-repair` — ACTIVE until fresh upstream proves an equivalent cryptographic restart invariant.
- `AF-P2 restart-signing-key-regression` — ACTIVE as regression/evidence; may become test-only or SUPERSEDED if upstream already guarantees the same invariant.
- `AF-P3 agent-process-instance-identity` — ACTIVE and high-priority because current native SWE acceptance blocker is `409 stale_agent_instance`; replay must validate registration, status update, lease renewal, and same-node/same-version process replacement semantics together.
- `AF-P4 restart-generation-stale-state-recovery` — ACTIVE; semantic replay must prefer fresh upstream lifecycle APIs over restoring historical implementation details.
- `AF-P5 harness-durable-schema-completion` — ACTIVE runtime-capture generation from `c0923acd...`; include `schemaOutputDir`/stable output completion behavior and its regression. Do not infer completeness merely from the latest capture commit; the full ordered runtime-capture chain is the evidence source.

The exact mapping of `e07dfca` and `4d337c1` into AF-P1/AF-P2 is behavioral rather than one-commit-one-patch: tests may belong to the same logical invariant as the implementation they prove.

### Required replay sequence

1. **Freeze AgentField generation N.** Use `c0923acd...` plus the ordered `runtime-capture/agentfield/c0923acdfca0` chain as immutable old-generation evidence. New edits after the cutoff belong to generation N+1.
2. **Fresh base.** Reuse exact upstream replay base `2180e30c...`; do not rebase the live runtime checkout.
3. **Classify/replay AF-P1..AF-P5 in semantic order.** For each: `REPLAYED`, `SUPERSEDED_BY_UPSTREAM`, `CONFLICT_REPAIRED`, or `HOLD`. Exact-symbol absence alone is not sufficient to prove necessity; regression behavior decides.
4. **Targeted validation first.** At minimum: Go SDK agent lifecycle/instance-ID tests, harness durable-schema-output regression, DID restart/signing-key regressions, and control-plane node registration/status/lease stale-instance behavior. Broader suites follow only after targeted GREEN.
5. **Differential baseline.** Any failure after replay must be rerun on clean upstream `2180e30c...` before attribution to a downstream patch.
6. **Advance one `dev` line only after source GREEN.** Fresh-read remote `dev` first; abort/replan on concurrent movement. No permanent per-patch branches.
7. **Runtime cutover remains a separate gate.** Only after accepted source candidate exists and current native-runtime work is safely quiesced may `/core-src/agentfield-runtime` move to the accepted `dev` SHA. Preserve any remaining old-generation delta first.
8. **Post-cutover proof.** Read back exact runtime source identity; run the same targeted validations on live source; only then advance AgentField SourceLoop/writeback provenance from its current old branch/base to the new `dev` generation and prove it with one ephemeral capture.
9. **Do not deploy/redeploy or use release CI** merely to accomplish source synchronization. Existing container-first runtime remains the validation surface.

### Replay progress / current blocker

- Existing `agentfield-dev-workforce` + persistent `/workspaces` was proven as the mediated writable replay surface; no new service/container was created. Temporary checkout `/workspaces/agentfield-upstream-replay` is rooted at exact upstream `2180e30c...` and owns this replay batch.
- AF-P1/AF-P2 restart-DID/signing-key history replayed cleanly as local temporary commits `8e9c4765`, `5da450fe`, and `e86b7e23`.
- AF-P3 `agent-process-instance-identity` conflicted only in `sdk/go/agent/agent_lifecycle.go`; semantic repair preserved upstream `stopLeaseOnce` shutdown semantics and downstream in-flight execution/pause cancellation. Resulting temporary commit is `9d0710f9`. Focused regressions `TestAgentInstanceIDPropagatesAndChangesPerProcess|TestShutdownCancelsInFlightReasoners` PASS on the fresh-upstream replay tree.
- The monolithic old `c0923acd...` commit was intentionally not replayed wholesale after conflicts across lifecycle/storage/handler areas. Its behavioral tests were decomposed. Fresh upstream already contains much of execution-instance persistence/orphan-reaping, queue rejection persistence, stale-workflow activity protection, and accepted-execution shutdown behavior; those portions are treated as superseded unless a missing regression proves otherwise.
- The still-missing AF-P4 stale-instance ownership slice was isolated to `nodes_rest.go`, `nodes_heartbeat.go` and their regressions. Its four-file patch (`/tmp/af-p4-stale-instance.patch`, 11,608 bytes) applied cleanly with `git apply --3way` on the fresh-upstream replay tree. This is the narrow contract needed to prevent stale/missing process instances from renewing status/heartbeat ownership while preserving legacy empty-instance compatibility.
- AF-P4 validation currently has **no product verdict**. Two bounded handler test attempts first spent the execution window downloading/compiling control-plane dependencies; a subsequent single-test diagnostic could not start because Docker/runc returned `no space left on device`.
- The disk-capacity blocker is RESOLVED. Fresh Hostinger metrics after cleanup show host disk usage reduced from 100% to roughly 57%; Coolify, Coolify Redis, AgentField control-plane, runtime-capture, diagnostics and Postgres are running again. `agentfield-dev-runtime-capture` target stats succeed after recovery, so SourceLoop/source-volume observation is available again.
- Recovery changed runtime topology: the previous `agentfield-dev-workforce` container/service is not present in current Docker inventory, so its registered target resolves to `target_container_unavailable`. This is now the active validation blocker: `VALIDATION_BLOCKER: WORKFORCE_SERVICE_MISSING_AFTER_RECOVERY`. Do not misclassify it as AgentField product failure or as remaining ENOSPC.
- The authoritative AgentField runtime source remains preserved on the shared volume through `runtime-capture`; `/core-src/agentfield-runtime` is still the current source owner. The old runtime generation remains `c0923acd...` until replay source is GREEN and an explicit runtime cutover is proven.
- Do not advance `dev`, modify `/core-src/agentfield-runtime`, update `.source-commit`, change SourceLoop writeback provenance, or replay AF-P5 through an unverified alternate toolchain while `agentfield-dev-workforce` is unavailable. Replay evidence already recorded in `/workspaces/agentfield-upstream-replay` remains the source-reconciliation state owner; if that workspace was container-local and is no longer present, reconstruct only from its recorded exact upstream base + logical patch evidence rather than from ad-hoc history.

### Completion evidence — 2026-09-16

- Fresh upstream base was `Agent-Field/agentfield:main@2180e30c7f1619a652635cf8e5038c36f4d0dc3e`; ACTIVE AF-P1..AF-P5 behaviors were semantically replayed in `/workspaces/agentfield-upstream-replay`, not by blindly rebasing divergent fork history.
- Source verification is GREEN. AF-P4 six stale/missing/current/legacy ownership regressions PASS; full `control-plane/internal/handlers` PASS; AF-P5 durable-schema regression and full harness PASS; AF-P3 shutdown/instance regressions and full `./agent` PASS; full Go SDK `go test ./... -count=1` PASS; `git diff --check` PASS.
- Tested local candidate tree was `3a39017d325198db6fdc18484b0d7532b83d3b9e`. GitHub replay commit `26841718c42d2a7fb9008014f4a1d552f945bac6` has exactly that same tree, proving published bytes equal tested bytes.
- The single integration line `n0namer/agentfield:dev` was advanced from old `c0923acd...` to exact tested `26841718...`. This one-time legacy reconciliation was non-fast-forward; rollback refs retaining `c0923acd...` remain. `main` was not rewritten and remains the accepted/release line.
- Old-generation post-cutoff work was preserved before runtime reset both as binary patch `/core-src/workspaces/agentfield-oldgen-cutover-20260916.patch` (SHA-256 `6728ac1fc37f50abd0ba60d33e96845f4fc1ff9a9c4e2b090a62c9fe66e5a419`) and durable branch `runtime-capture/agentfield/c0923acdfca0@3f433227...`; all ten N+1 dirty-file blobs were verified equal to that capture branch.
- Runtime source cutover is verified: `/core-src/agentfield-runtime` `HEAD` and `.source-commit` both equal `26841718...`; tracked source is clean. Live post-cutover full Go SDK PASS, full `control-plane/internal/handlers` PASS, and DID restart regression PASS.
- Deployment drift was closed in canonical `n0namer/universal-solver:archops/swe-native-fb0-stack@22470b55543a71187c9a2f973583529a392fa488`: all active AgentField source/build/workforce/runtime-capture pins now target `26841718...`. New `control-plane-build` used that source, full `internal/services` PASS, and exited 0; new control-plane and workforce are healthy and runtime-capture is running.
- Workforce runtime provenance `/afhome/us-e2e-provenance.txt` reports `agentfield=26841718c42d2a7fb9008014f4a1d552f945bac6`; `swe-planner` registered successfully on the rebuilt workforce. Native AgentField health reports `healthy` / gateway `ok`.
- Runtime capture on the new generation is independently proven in Git: `runtime-capture/agentfield/26841718c42d@c0b02cd135e5a095ff66c99a89beb2400ef4d9b5` has parent exactly `dev@26841718...`; PR #9 uses that capture branch as head and `dev@26841718...` as base. This branch is recovery/capture staging, not canonical logical-patch identity and must not be merged merely because it exists.
- SourceLoop binding `agentfield-dev-runtime-capture` advanced revision 5→6 changing writeback branch only `main→dev`. Ephemeral canary `vtchg_ca69503f157840f895b39c4f60481cd9` independently proves journal provenance `repository=n0namer/agentfield`, `base_branch=dev`, `base_commit=26841718...`; canary file was deleted. Historical old-generation captures remain immutable on `main@c0923acd...` provenance.

### Current bounded next move

Restore the **existing** AgentField workforce service/target through its authoritative Coolify owner route; do not create a replacement container or Coding Station path. After fresh target identity/readback succeeds, resume exactly at AF-P4 targeted stale-instance regressions on the fresh-upstream replay source, then complete only the still-non-superseded behavioral slices and AF-P5 harness replay. Run differential baseline checks, and only after source GREEN consider advancing the single `dev` line. No deploy/redeploy of AgentField product code, PR, release CI, runtime cutover, or provenance rebinding before source GREEN.

## Universal SourceLoop onboarding contract
Apply this same contract to every existing project; do not redesign SourceLoop per repository.
1. Identify one canonical repository/branch and, where applicable, one upstream repository/branch. Ambiguous source ownership is `SOURCE_OWNER_UNKNOWN` and blocks canonicalization.
2. Identify the exact runtime source root actually executed by DEV and prove its immutable base SHA. Missing/ambiguous base is `SOURCELOOP_GAP`; unknown drift must not be canonicalized.
3. Reuse an existing runtime target/container first. Register/bind that target to `{project, runtime_root, repository, branch, base_resolver}` only when no valid binding exists; do not create new services/containers merely for SourceLoop.
4. Define the project verification contract (targeted regression + smallest canonical package/suite checks). Product code remains container-first; GitHub/CI/deploy are not the inner coding loop.
5. Bootstrap with one small durable RED→GREEN patch, classify runtime delta as KEEP/DROP/UNKNOWN, and canonicalize only KEEP from the exact base. Require runtime KEEP diff == canonical Git diff on reread.
6. Record first-class linkage `{logical_patch_id, capture/change IDs, original base, canonical commit generation, tests/evidence}`. A Git branch/commit existing without this linkage is replayable evidence but SourceLoop onboarding remains PARTIAL.
7. Repeat on a second independent durable patch. Two successful captures/replays prove the project binding and replay pattern; WIP limit is at most two VERIFIED-but-not-CANONICAL durable patches.
8. For an already-dirty legacy project, baseline first: separate known canonical state, reconstructable durable patches, and UNKNOWN drift. Quarantine UNKNOWN; never bless the whole dirty tree as the initial SourceLoop state.
9. Upstream update procedure: fetch/read new upstream base → replay ordered ACTIVE logical patches one by one → for each classify CLEAN/REPLAYED, SUPERSEDED_BY_UPSTREAM, or CONFLICT → run that patch's regression/canonical tests → create a new patch generation linked to the same logical patch identity. Do not blindly merge the whole fork.
10. DEV may continue ahead after canonicalization. PROD promotion selects one exact tested canonical commit/release snapshot; promotion never implies that newer DEV patches auto-enter PROD. Rollback is movement of the PROD pointer to the previous accepted exact commit.

Universal onboarding DoD: canonical owner known; runtime root known; exact base known; binding/captures proven; project validators defined; two durable patches pass capture→exact-base→canonical reread; formal capture→commit linkage exists; active ordered patch stack is reconstructable; upstream owner is known; release snapshot is expressible as an exact commit. Missing any item means PARTIAL/BLOCKED, never GREEN.

## Current Phase Goal
Initial SWE Quality Acceptance is CLOSED on the existing AgentField container stack. Continue container-first operational hardening on the accepted classic `swe-planner` path: keep `swe-planner` active/ready with canonical SDK provenance, preserve the accepted `broker/fast-coding` + OpenCode runtime contract, enforce scope/junk hygiene, and reduce avoidable coder/reviewer/verifier latency without weakening verification. No PR/deploy/redeploy/CI is part of this phase.

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
- BMAD current contract: `bmad-build` is the implementation loop (`bmad-quick-dev` is a compatibility shim). For this debugging/implementation phase use its lean shape only: observe/reproduce → isolate root cause → smallest scoped change → canonical validation → evidence-backed review. Do not add ceremony that does not improve a gate.
- Systematic debugging skill: reproduce/observe before fixing; test one high-information hypothesis at a time; don't stack speculative fixes.
- TDD skill: no product implementation before an observed failing test for the target behavior; RED must fail for the intended reason, then minimal GREEN.
- Verification-before-completion: functional readback/test evidence, not tool acknowledgement or health alone, closes each guarantee.
- Eval: deterministic correctness first; trajectory/tool-use, repair count, cost and latency secondary.
- External skills are method input only; no framework installation without a proven gap.
- Anti-drift invariant: objective is proven generic contract completion on exact AgentField source: runtime truth, preserved partial work, no duplicate mutation, smallest continuation, bounded termination/cost, then E2E + golden regression. Git/fork reconciliation and SourceLoop publication are release mechanics only and must not become the objective or run before those behavioral gates PASS. SWE/FCM/OpenCode remains out of scope unless it directly blocks this objective.

### Batch C — relevant E2E proof — DONE (2026-09-14)
DoD:
- Run the repository's existing end-to-end resilience harness on the exact container workspace before any publication/fork reconciliation.
- Add or reuse the thinnest end-to-end scenario that exercises schema/contract continuation through the public Runner path, not just helper functions.
- Prove a durable partial/completed effect is observed after an ambiguous provider/transport result and is not executed twice.
- Prove unresolved fields continue with the smallest repair while satisfied fields remain preserved.
- Capture executable evidence: command, exit status, assertions, and relevant artifact/readback.

### Batch D — golden regression — DONE (2026-09-14)
DoD:
- Freeze representative contract-completion inputs/outputs as deterministic golden fixtures or equivalent repository-native snapshots.
- Include at minimum: satisfied+missing mix, invalid field, unknown/fail-closed state, ambiguous-result-with-observed-effect, and successful incremental repair.
- Golden comparison must fail on duplicate mutation, loss of preserved partial output, widened continuation, or changed fail-closed behavior.
- Re-run affected harness tests + full Go SDK after golden coverage is added.

### Batch E — native SWE runtime acceptance — DONE (2026-09-15)
DoD:
- Work only inside existing AgentField containers/terminals; no PR, CI/CD, deploy or compose recreate while the runtime is not functionally green. — PASS.
- Preserve authoritative `/src/swe-af` runtime state and prove the bootstrap path does not discard accepted dirty work. — PASS on `dev@5b54a1dd4c579fd41ddf141e20dd46259519322e`.
- Bring the штатный `swe-planner` to active/ready through the existing AgentField control plane. — PASS.
- Pass one non-mutating native reasoner smoke through AgentField discovery/execution. — PASS (`exec_20260915_122235_69og7el9`).
- Pass one bounded `implement_issue` against the accepted runtime source and its canonical tests. — PASS on nested-Go canary build `a1e63951`; resumed execution `exec_20260915_193318_opt87s9w` ended `succeeded`, outcome `completed`, verification `passed=true`, commit `7f3605cfed7bcd41e2ba2d6a087294d75c8c5253`, exactly one changed file (`go/internal/calc/calc.go`), independent `go test ./...` PASS.
- Pass a recovery canary proving no duplicate mutation/lost partial state after an interrupted or ambiguous execution. — PASS: observable mutation was preserved across controlled interruption; explicit resume reused the same build `a1e63951`, same issue worktree/branch, produced no duplicate worktree, and completed successfully with one commit.

### Batch F — initial SWE quality acceptance — DONE (2026-09-15)
DoD:
- Use the accepted live DEV path only: classic `swe-planner`, `open_code`, `broker/fast-coding`, no PR/deploy/redeploy/CI. — PASS.
- Reference bounded bug-fix task remains clean PASS from Batch E (`a1e63951`, one expected source file, independent tests PASS).
- Recovery task remains mandatory PASS: controlled interruption + same-build resume with no duplicate worktree/mutation. — PASS.
- Test-gap task `exec_20260915_201331_39lhjedj`: terminal `succeeded`, `success=true`, one iteration, exactly one added test file, verifier PASS. — PASS.
- Cross-file task `exec_20260915_201337_4er7dpxh`: terminal `succeeded`, `success=true`, one iteration, exactly two expected production files, verifier PASS. — PASS.
- Ambiguous task did not count as clean PASS: initial run `exec_20260915_201345_5pgcia09` ended `failed_unrecoverable` after coder timeout and exposed tracked `.agentfield-out-*`; resumed run `exec_20260915_202821_u3u0hsgx` reached completed/reviewer-approved code but final verification was unavailable while planner registration was stale, and the branch widened scope into `port_test.go`. — FAIL / diagnostic evidence.
- Initial quality threshold `>=4/5 clean tasks with mandatory recovery PASS` is therefore MET exactly at 4/5. Do not round this to 5/5.
- Quality-run defect `AgentField structured-output artifacts can be committed` was reproduced RED then fixed GREEN by extending `junkPathspecs`; targeted regression and full `internal/issue` suite PASS. Canonical `swe-af:dev` and live source now equal `0a3f7ef2ecfdc71dd218fad41bbfe28abe1fb3ec`.
- Planner stale-instance regression caused by a temporary `go.work` reinstall was re-closed using canonical `GOWORK=off + AGENTFIELD_GO_REPLACE`; current `instance_id=28203be8f1884c449c5425862586283e`, health 100, lifecycle `ready`, and the 20:39 heartbeat is active with no fresh 409 after the 20:37 registration.

## Next move
Technical Acceptance and initial Quality Acceptance are CLOSED. Current canonical/live SWE generation is `n0namer/swe-af:dev@0a3f7ef2ecfdc71dd218fad41bbfe28abe1fb3ec`; live `HEAD` and `.source-commit` match it, `go test ./internal/issue ./internal/node -count=1` PASS, and `swe-planner` is active/ready with health 100 and non-empty instance `28203be8f1884c449c5425862586283e`. The accepted runtime path is classic engine (`SWE_PRO_ENGINE=0`) + `open_code` + `broker/fast-coding`; OpenCode wrapper selects the latest installed runtime and exposes the existing Go toolchain.

Canonical cold-bootstrap source is updated without deploy/redeploy: `n0namer/universal-solver:archops/swe-native-fb0-stack` pins SWE to `0a3f7ef...`, recreates the proven OpenCode wrapper, installs SWE with `GOWORK=off + AGENTFIELD_GO_REPLACE=<live SDK>`, and explicitly pins `SWE_PRO_ENGINE=0`. These source changes are not claimed as runtime-recreated until a later allowed cold start.

Next 30-minute Batch G is operational quality hardening, not CI/CD: (1) enforce declared-file scope for `implement_issue` so out-of-scope mutations cannot silently become a clean success; add a deterministic RED→GREEN regression first; (2) rerun one bounded task on the accepted OpenCode runtime and record coder/reviewer/verifier wall-clock latency; change latency behavior only if evidence identifies a deterministic local cause. Preserve verification strength. No PR/deploy/redeploy/CI and no Coding Station.
