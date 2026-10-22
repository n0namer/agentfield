# Basefive R14 — independent source-bound C4 QA

Date: 2026-10-14. Worker evidence: CANDIDATE / UNACCEPTED, subject to native checks below. Product ReleaseGate remains BLOCKED / UNACCEPTED. This review does not grant acceptance or change live coordination state.

## Authority, target and gate

- Task: `agentfield:v2-r14b-independent-c4-qa-20261014-r1`.
- Attempt: `attempt-fa2e3714-5e16-456f-822d-767bd94ea109`.
- Session: `manual-20261014-basefive-architecture-review-r1-03176d6cd68a5abd`.
- Target root: `D:/Users/NIKITA/Documents/DEV/universal-solver-site/agentfield`; launch Git identity `44167dc1050f5bff79abc32fab19c0bb50643093`. Clean worktree/branch state is UNVERIFIED; no Git command is authorized.
- BMad source of truth: `docs/acceptance/c4-releasegate-reducer-frontier-20261013.md`, unchanged. Current stage is independent QA of the Basefive R14 documentation producer candidate, not production release.
- Exclusive write scope: this report and `docs/architecture/basefive-r14-independent-c4.test.mjs`. No product edits, index regeneration, task administration, commit, deployment or self-acceptance.

FACT: The producer's six architecture artifacts exist. A bounded architecture-directory listing contains neither independent target before implementation; the independent mechanism was not already present there. The producer test already implements comprehensive source/DTO/inventory mutation checks. This suite imports and executes that existing Node test without copying or replacing its validator, then adds independent checks with independently fixed source identities and relationship expectations.

Value: establish whether the documentation accurately binds this exact visible Go execution boundary and rejects misleading drift. Measurable bounded DoD: rehash five inputs, recompute graph/model coverage, check seven local DTOs and all eleven diagram relationships, exercise at least two distinct negative drift cases through native Node, preserve limits and publish one typed UNACCEPTED verdict. Python SDK source and native ReleaseGate runtime are not authorized by this read scope and cannot be silently inferred.

## Source identity

These full-file identities were obtained from scoped source range-read receipts and are pinned independently of the producer JSON. The native test compares raw file bytes, not line-ending-normalized bytes, then compares the producer manifest against the same identities.

| Input | Independent SHA-256 |
|---|---|
| `control-plane/internal/handlers/execute.go` | `be09150579e29e16ce9d96775ef1b161898d829570b9bdd5885c9db5327a25c1` |
| `control-plane/internal/storage/execution_idempotency.go` | `7ff044cdce4e1ec772e3e79a6d4e074e9fece17c2a6857bdf0f4552a1db9cfe2` |
| `graphify-out/graph.json` | `1f00b4b243e4f4ec7e9fb35a8373c2054f95615fe4a454ab8e02addee1cc8f5d` |
| `.archsteer/model.json` | `0350f5afc09c996266c2745fb27e2fd2d29e3245e15f1568f8f6681206f17f04` |
| `.archsteer/architecture.md` | `af90dfa9b7d59211a5d90d0fe057d569d841ef34d7c1c177f446e957584d4c5a` |

Producer contract SHA from scoped receipt: `02f54cd157d76cf10ccbc07a15e9e595289e8e0f911414adde8b25e68e724185`. Producer test SHA: `0022e7b2789911b32b26ef212f65c1c414481e20c49598f20b7a0fec4777353e`. Native `INDEPENDENT_QA_RECEIPT` emits hashes and byte lengths for all five inputs, six producer artifacts and two independent outputs, including the report's final version after retest. A whole-worktree immutable snapshot is not asserted.

## Independent findings

### F1 — partial Go graph coverage, not semantic proof

FACT: Producer inventory records Graphify 31,049 nodes, 14,586 Go-path nodes and 1,102 distinct Go paths. The handler has 43 nodes; the storage source has zero. Relevant directory path counts are 235 and 75 respectively. Native QA recomputes these values from `nodes[].source_file`, independently compares the inventory, and retains `call_edges = UNVERIFIED`. Graph absence is not source absence; a graph SHA is not an index freshness guarantee.

FACT: ArchSteer schema 1.0.0 / short commit `44167dc1` records 1,210 components: Python 447, JavaScript 15, TSX 354 and TypeScript 394. No component is Go by extension or language; neither exact target nor directory has a component, and neither target path appears in the generated architecture document. Native QA recomputes all these values, including fingerprint-field absence, and preserves `go_semantics = UNVERIFIED`. This model inventory, not ArchSteer capability generally, has the observed Go blind spot.

### F2 — local DTO and source-visible edge contract

FACT: The local Go declarations support seven DTOs and the batch response alias. Independent parsing requires every non-comment DTO field to have a recognized JSON tag; it compares Go name/type, wire name, omitempty and binding with the manifest. Imported `RunMetadataInput`, `types.Execution`, webhook event definitions and Python SDK DTO compatibility remain UNVERIFIED. The documentation schema is not a product wire version; router registration remains UNVERIFIED.

FACT: Three diagrams have 2 context, 3 container and 6 component relationships. Independent expected tuples prevent a jointly modified diagram and manifest from redefining the topology. For the six component edges the test binds each call to its actual containing function, not merely an arbitrary occurrence anywhere in a source file. Lookup precedes queue admission in `handleAsync`; execution insertion precedes the final transaction commit in `CreateAsyncExecutionIdempotent`. These are source-layout checks, not Go AST/control-flow or runtime guarantees. External agent invocation is a referenced implementation boundary; SQL database engine/deployment is unverified.

### F3 — producer verifier hardening opportunity, not current artifact defect

FACT: The producer verifier compares diagram edges with its manifest and checks anchors anywhere in the selected source file; it fixes edge counts but not full independent edge tuples or containing call owners. A real source anchor does not by itself establish its claimed endpoints or causal relationship. The independent suite adds those constraints without editing producer-owned files. No observed mismatch in the current artifact is presumed from this validator limitation.

### F4 — acceptance and authorization limitations

FACT: The goal mentions Go/Python source, but the exact read scope contains only two Go source files and generated Python model metadata. No Python SDK source path is authorized. Python-language inventory can be verified; Python source/DTO compatibility cannot. Required next dependency is coordinator-authorized exact Python source paths if that broader gate is still required. This review is not a complete Go/Python architecture verdict.

FACT: Existing BMad frontier requires native producer/QA receipts, failure/restart E2E and coordinator terminal readback. Static Node tests do not repair the previous INVALID_REPORT publication, establish database constraints, prove restart dispatch or grant release acceptance. Source-visible durable binding must not be represented as durable queue recovery.

## Negative drift evidence design

All mutations are in memory; real source and producer artifacts remain read-only.

1. DTO drift: change async execution wire name from `execution_id` to `id`; independent comparison must throw the specific AsyncExecuteResponse DTO-drift assertion.
2. Paired edge drift: redirect durable transaction insertion from SQL to Pool in both Mermaid and JSON, retain a valid storage anchor and real endpoints; independent fixed topology must throw. This specifically avoids relying only on producer/diagram agreement.
3. Rebound source drift: mutate Go wire tag and update its candidate SHA; independent original source identity must reject the new source bytes.
4. Owner drift: remove `pool.submitReserved(job)` from its actual caller and preserve it elsewhere in a comment; independent containing-function check must reject the edge even though a file-wide anchor search succeeds.

Positive counterparts exercise unchanged DTOs, edges, source bytes and coverage. The existing producer mutation suite is also run by direct Node import in the same authorized invocation; no shell/subprocess or alternate scheduler is involved.

## Native validation

Authorized command: `node --test docs/architecture/basefive-r14-independent-c4.test.mjs`, via `go_wave.run_project_tests`.

FACT: Native runner returned `status = PASS`, `engine = node:test`, `exit_code = 0`, `signal = null`, `error = null`, empty stderr. TAP: 47 tests, 47 pass, 0 fail, 0 cancelled, 0 skipped, 0 todo; duration 3915.3671 ms. The 37 producer tests and 10 independent tests executed in the same process. All four independent negative cases passed by rejecting the intended mutations; no product failure or repair was required. Receipt timestamps: producer artifact receipt `2026-10-14T16:20:56.644Z`, independent artifact receipt `2026-10-14T16:20:57.172Z`. The MCP display truncates the middle of stdout; status, native exit, coverage inventory and final TAP summary remain explicit. Final report edits require a fresh identical authorized run; that final run's output hashes, timestamp and exit are carried in the worker result rather than recursively embedded into the report being hashed.

No Go compiler/runtime, HTTP integration, Python runtime, Mermaid renderer, SQL race/restart, lint, build, Git status or native ReleaseGate E2E is represented as executed. Parsing supports only the producer's bounded Mermaid subset and current top-level Go declaration layout, not arbitrary languages/syntax.

## Method and self-review

BMad Help was loaded first against this target and its existing acceptance artifact. Review skill selected: `bmad-code-review`; verification method: `verification-before-completion`. The review workflow's customization/config scripts, Git-diff construction, interactive checkpoint and parallel reviewer stages are BLOCKED METHOD STEPS under this launcher's no-shell, exact-scope and no-agents constraints; no full BMad code-review workflow completion is claimed. We use the explicit artifact list and source-backed independent tests instead. No extra BMad config/sprint files or competing tracker are created. Graph/Graft navigation tools are not exposed; scoped reads and native JSON inventory are the fallback.

Review checklist: only two authorized new documentation/test files; no dependencies; all reads use exact allowed paths; test mutations do not write; producer suite reused; independently pinned byte hashes; negative assertions check intended failure messages; runtime/release limits retained. Test execution hashes provide source/output readback; repository-wide cleanliness remains UNVERIFIED.

Useful-work estimate: 15–20 minutes (investigation 5, implementation 6, native checks/possible repair 4, evidence/self-review 3); no artificial runtime target. Actual runner duration and exit belong to the native receipt; wall-clock coding duration is not measured by an authorized timer.

## Verdict and handoff

FACT: Independent bounded static-contract verdict is PASS on the recorded source snapshot: byte identities and Go coverage checked, eleven relationships and seven local DTOs validated, four independent negative cases rejected, and producer regressions executed. Worker deliverable remains CANDIDATE / UNACCEPTED. Full product C4 ReleaseGate: BLOCKED / UNACCEPTED. Python source-level compatibility is NOT VERIFIED because its exact files are outside read scope. No self GREEN.

Next action: independent acceptance owner reviews these exact source-bound artifacts and the typed publication; coordinator retains the blocked gate until original native producer/QA receipts and failure/restart E2E exist, and authorizes exact Python source reads if cross-language compatibility is required. Do not duplicate the producer or edit other owners' files.

Publication transport status belongs to the single typed publisher receipt returned after final tests; transport VERIFIED, if returned, is not product acceptance.
