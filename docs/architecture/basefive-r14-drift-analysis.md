# Basefive R14 — source-bound drift analysis and ADR

Date: 2026-10-14. Status: CANDIDATE / UNACCEPTED; independent acceptance required.

## Current gate and bounded Definition of Done

FACT: The existing BMad source of truth is `docs/acceptance/c4-releasegate-reducer-frontier-20261013.md`. It records a blocked C4 ReleaseGate, absent source/interface bindings, absent native failure/restart E2E and a previous INVALID_REPORT publication. This task does not repair that other attempt or change its gate.

This slice delivers three C4 levels grounded in two authorized Go files, machine-readable source/API/DTO bindings, explicit Graphify/ArchSteer limitations, deterministic Node positive and mutation-negative checks, and one typed UNACCEPTED publication. It neither changes Go behavior nor claims live service validation. The six target files were not already implemented: the authorized architecture directory lookup returned ENOENT before creation.

## Exact evidence

| Input | SHA-256 / availability | Meaning |
|---|---|---|
| `control-plane/internal/handlers/execute.go` | `be09150579e29e16ce9d96775ef1b161898d829570b9bdd5885c9db5327a25c1` | Handler entry signatures, request/response DTO declarations, raw-body idempotency and queue admission references. |
| `control-plane/internal/storage/execution_idempotency.go` | `7ff044cdce4e1ec772e3e79a6d4e074e9fece17c2a6857bdf0f4552a1db9cfe2` | Optional store interface, lookup/conflict behavior, binding + execution transaction. |
| `graphify-out/graph.json` | `1f00b4b243e4f4ec7e9fb35a8373c2054f95615fe4a454ab8e02addee1cc8f5d` | Graph artifact identity, not proof of matching source fingerprints or edge completeness. |
| `.archsteer/model.json` | `0350f5afc09c996266c2745fb27e2fd2d29e3245e15f1568f8f6681206f17f04` | Initial full read returned `GO_WAVE_NODE_ERROR:file_missing_or_oversized`; Node stat proved PRESENT (2,512,613 bytes), then a bounded range read succeeded and Node parsed the authorized artifact. Model schema 1.0.0, commit field `44167dc1`. |
| `.archsteer/architecture.md` | `af90dfa9b7d59211a5d90d0fe057d569d841ef34d7c1c177f446e957584d4c5a` | Initial full read returned the same size-limit receipt; Node stat proved PRESENT (157,647 bytes), then a bounded range read succeeded. Generated artifact preserved read-only. |
| `AGENTS.md` | `591371e0370fe3d95ec809f7e47a3c80ca557b0c726c3f01354732bf422ca75e` | Independent release authority; minimal scoped change; no secrets or live coordination mutation. |

The launcher Git identity is `44167dc1050f5bff79abc32fab19c0bb50643093`; it is not asserted to identify a clean worktree. Node verifies source bytes rather than relying on HEAD. No Git or shell tool was run.

## Drift findings

1. FACT — The accessible execution control-plane implementation is Go/Gin, whereas AGENTS.md's overview advertises TypeScript/Node.js. The overview is insufficient to describe this bounded subsystem; no repository-wide tech-stack conclusion follows.
2. FACT — `ExecuteRequest.RunMetadata` is commented as excluded from replay dedupe. Opt-in async `Idempotency-Key` uses SHA-256 of all raw request body bytes, so whitespace and run_metadata changes affect this distinct durable binding. Do not conflate result replay and request idempotency.
3. FACT — Durable uniqueness in the visible SQL insert is `(scope_key, request_key)`, not `(target, request_key)`. Scope combines verified caller DID, NUL and target DID; stored target/body digest mismatch produces conflict. Actual migration/index enforcement is UNVERIFIED without migration source/runtime evidence.
4. FACT — An unsupported optional durable capability rejects opt-in idempotency with a 503 precondition; no process-local deduplication fallback is present in the visible handler branch. Missing verified caller is a 401 precondition; malformed key is 400; conflict is 409; replay/new async acceptance is 202. Error-renderer implementation and actual HTTP responses remain unexecuted.
5. FACT — Idempotency replay returns `workflow_id = RunID`, `enqueued_at = CreatedAt`, and an empty `type` string because Type is not populated and is not omitempty. It can report the persisted execution status, not necessarily queued. These differ from fresh queue-response population; changing them would require a separate authorized product decision.
6. FACT — Queue and reservations are process-local channels. Transactional request binding does not prove durable dispatch, restart recovery or exactly-once external execution. A visible stopped-submission branch references detached shutdown termination, but the worker/recovery implementation is outside scope.
7. FACT — DTO JSON names, Go field types, omitempty and binding are captured exactly, including synchronous `error_message` versus status/callback `error`, and callback `usage`. Omitempty is serialization behavior, not a business-requiredness declaration. Imported RunMetadataInput, types.Execution and webhook event schemas remain UNVERIFIED. No explicit product wire-version declaration is visible; the new JSON artifact's schema version is documentation only.
8. FACT — Handler factories are visible, but route registration, URL/method, middleware installation, deployed topology, agent call implementation and database engine are outside scope. Diagrams explicitly mark external references and logical boundaries, not verified physical deployment.

## Graphify / ArchSteer coverage

The Node runner computes Go-node count, distinct `.go` source paths, target-file node inventory and package-directory path counts from Graphify's `nodes[].source_file`, with language/kind samples. This is a bounded inventory, not an exhaustive repository audit. It does not execute graph edges, verify an AST index generation, infer source absence from graph absence, or inspect unauthorized Go package files. Fingerprint/semantic/call-edge coverage remains UNVERIFIED even if the two files appear as nodes.

FACT — Focused Node inventory (same exact input hashes):

| Metric | Graphify | ArchSteer |
|---|---|---|
| Entire artifact inventory | 31,049 nodes; 14,586 nodes whose source path ends `.go`; 1,102 distinct Go paths | 1,210 components; languages: Python 447, JavaScript 15, TSX 354, TypeScript 394; zero Go components by `.go` path or `language = go` |
| `handlers/execute.go` | 43 nodes; 235 distinct Go paths under handlers directory | No component; zero handlers-directory components; no exact file-path mention in generated architecture.md |
| `storage/execution_idempotency.go` | Zero nodes; 75 distinct Go paths under storage directory | No component; zero storage-directory components; no exact file-path mention in generated architecture.md |
| Source freshness | Artifact SHA bound; target nodes have no metadata.language/kind in the inspected fields; semantic freshness UNVERIFIED | No component-level key matching sha/hash/fingerprint; short commit field alone does not bind dirty source bytes; semantic freshness UNVERIFIED |

FACT — Graphify's omission of the authorized storage file is an observed coverage gap, not proof the implementation is missing. ArchSteer has a complete Go blind spot in this model's component inventory; that statement is bounded to the inspected JSON, not a claim about tool support generally. The model's reported schema 1.0.0 is not the execution API version. Package-directory counts are artifact path counts, not filesystem completeness or verified Go packages. Exact measurements are captured in `coverage.inventory` and compared by the test; false storage presence/Go coverage mutations must fail.

The initial ambiguous read errors were diagnosed rather than upgraded to absence or treated as a persistent infrastructure block. Bounded source reads and authorized Node parsing recovered the comparison. No identical denied call was retried.

Blind spots retained in the machine contract: router registration; imported DTO versions; migrations/database engine; worker restart dispatch; Go graph AST/call-edge completeness; native ReleaseGate task/result E2E and independent QA. No index is regenerated and no competing architecture model is introduced.

## ADR-R14-01 — bounded source contract, fail-closed release evidence

Status: proposed documentation decision / UNACCEPTED, not an approved production ADR.

Context: The prior reducer lacks source/version and native E2E receipts. Graph presence or a generic framework overview cannot safely substitute for actual Go DTO/interface evidence.

Decision: Model only the source-visible execution boundary using Mermaid flowcharts with C4 level semantics. Keep a JSON manifest of exact source SHA-256 values, field-level DTO tuples, handler/storage signatures and source anchors for every diagram node/edge. A strict verifier for the generated Mermaid subset checks inventory, labels, endpoints and anchors; mutation negatives prevent stale documentation, invented version claims and coverage/acceptance escalation. It uses Node built-ins, adding no dependency.

Alternatives rejected: declaring Graphify semantically complete without Go AST evidence; manufacturing an ArchSteer comparison; inventing routes or API v1; treating durable idempotency as durable dispatch; modifying live task state or foreign acceptance artifacts.

Consequences: Source drift intentionally fails the check until reviewed and rebound. Source anchors prove references, not full behavior or causality. Full Mermaid rendering/parser integration is UNVERIFIED: no authorized renderer or package installation is exposed. This limited ASCII flowchart subset is checked structurally, not represented as a complete Mermaid parser. Go runtime, races, database constraints, process restart and native task-result publication remain separate executable gates.

## Validation and handoff

Command: `node --test docs/architecture/basefive-r14-contract.test.mjs` through the authorized `go_wave.run_project_tests` runner.

Focused evidence: the initial and recovered-inventory runs each passed 32/32 Node tests, exit 0, no stderr. Final self-review extends the suite to six handler factories and persisted inventory comparisons, adds four coverage/scope/queue mutation negatives and an artifact-integrity check; its fresh exact receipt is emitted by the final authorized runner and carried in the typed result. Tests cover exact bytes, complete DTO fields for seven local structs, six handler factories, storage method signatures, protocol source anchors, all three diagram inventories, and mutation negatives for hashes, paths, fields, signatures, limits, statuses, coverage/version escalation and invented diagram nodes/edges. Raw-body digest distinction is a Node illustration bound to the Go source anchor, not an execution of the Go handler.

No Go runtime, build, router HTTP smoke, SQL race/restart, native ReleaseGate E2E, independent QA or release check has run. The earlier Go module-root failure is not repeated or labeled PASS. BMad resolver/config discovery, extra spec/sprint writes and interactive Quick Dev checkpoints are not executed because they exceed the trusted read/write/command scope. Graph/Graft MCP tools are not exposed; exact scoped reads and the bounded graph inventory are the fallback.

Next gate: independent reviewer validates these six artifacts against this exact source snapshot; coordinator retains the existing gate until original native producer/QA receipts, negative/restart E2E and authoritative terminal readback exist. Useful-work estimate: 20–25 minutes (investigation 5, implementation 8, tests/repair 5, review/evidence 4; external coordination excluded). Finish promptly once the bounded evidence is proven; do not pad duration.
