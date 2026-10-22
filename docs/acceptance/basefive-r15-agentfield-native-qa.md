# Basefive R15 Agentfield Native QA — 2026-10-14

## Same-thread repair — attempt-5793d278-c37f-4468-8430-ec3870f72dca

**FACT: fixture repair applied; local validation INFRA BLOCKED / UNACCEPTED.** This section supersedes the original-attempt status and current-file wording below; the remaining original receipt is retained as historical evidence. No full product GREEN, independent acceptance or terminal readback is claimed.

- Original task: `agentfield:v2-r15-independent-native-mvp-qa-20261014-r1`.
- Current repair attempt: `attempt-5793d278-c37f-4468-8430-ec3870f72dca`.
- Session: `manual-20261014-basefive-r15-native-qa-agentfie-5d0a6ff6f619b999`.
- Sticky thread supplied by coordinator: `01a13b2c-7e1b-71a2-b923-88544ae0564b`; physical report binding still requires `producer.thread_id: null`.
- Exact scope remains this acceptance report and `control-plane/internal/storage/northstar_r15_independent_live_qa_test.go` only.

### Independent finding and root cause

FACT (coordinator-supplied independent review, not executed by this worker): Docker Go1.26.5 command `go test ./internal/storage -run TestR15 -count=1 -timeout 3m` failed only `TestR15AuthorizedSourceFingerprints`. With storage at `/work/internal/storage`, three parent levels resolve `/`, causing the fixture to read `/docs/architecture/basefive-r14-contracts.json` instead of the module-mounted `/work/docs/architecture/basefive-r14-contracts.json`. The review classifies this as a test fixture defect, not SQL product RED. No complete external log or source-snapshot receipt was supplied, so this summary is not upgraded to independent full-suite evidence.

The repair changes exactly one Go line: `projectRoot := filepath.Clean(filepath.Join(storageDirectory, "..", ".."))`. For `/work/internal/storage`, two parents resolve `/work`, as required by the coordinator's native mount layout. Nil-execution rejection, both independently declared API assertions, all three fingerprint targets, read-error and empty-file failures, and SHA logging are unchanged. No fallback search, skipped missing file, relaxed fingerprint check or production edit was added.

### Corrected SHA and actual retest

Scoped edit receipt:

- Previous test SHA-256: `1f5a3a95d45b5689b826d3d87ee94b032a343c17e827444ef8f84e8ea8fae449`.
- Corrected test SHA-256: `84b042ddfaf980ece26a9a8a4bff547d378793f7da0383a3ead7d98772e820d1`.
- Corrected file bytes: `2001`; atomic edit receipt returned.

Actual repair validation invocation: `go_wave.run_project_tests({"file":"control-plane/internal/storage/northstar_r15_independent_live_qa_test.go"})`.

```json
{
  "status": "FAIL",
  "code": null,
  "test": "control-plane/internal/storage/northstar_r15_independent_live_qa_test.go",
  "engine": "docker:go-test",
  "source_identity_sha256": "84b042ddfaf980ece26a9a8a4bff547d378793f7da0383a3ead7d98772e820d1",
  "exit_code": 1,
  "signal": null,
  "error": null,
  "duration_limit_ms": 120000,
  "output_truncated": false,
  "stdout": "",
  "stderr": "go: cannot find main module, but found .git/config in /workspace\n\tto create a module there, run:\n\tgo mod init\n",
  "package": "./control-plane/internal/storage",
  "go_image": "golang:1.24.0-alpine"
}
```

FACT: This scoped runner did not compile or run the corrected tests. Classification is `BLOCKED_INFRA`, represented by the physical result schema's allowed `outcome: BLOCKED`. It is not a fresh test failure in the corrected fixture. No identical retry, module initialization or runner-side workaround was attempted.

Formatting: `gofmt` execution is unavailable through the exposed scoped MCP interfaces; shell is prohibited. The single expression edit preserves existing Go formatting, but no formatter execution or formatter PASS is claimed. Affected baseline is not rerun against the unchanged pre-compilation runner blocker.

### Self-review and next bounded gate

Source-backed self-review confirms only the parent-count expression changed in Go. Runtime compatibility and fingerprint reads remain unverified by this worker. Estimated useful repair work is 3–5 minutes, with no exact elapsed measurement claimed. The original broader migration, parallel SQL, owner-isolation, restart and current HTTP/C4 gaps are not addressed or relabeled by this fixture repair.

Coordinator recovery: in the existing offline Docker Go1.26.5 `/work` module mount, with the architecture contract staged at `/work/docs/architecture/basefive-r14-contracts.json`, run `gofmt -w internal/storage/northstar_r15_independent_live_qa_test.go`, then `go test ./internal/storage -run TestR15 -count=1 -timeout 3m`. Collect verbose SHA logs with `-v` and bind the receipt to the corrected test SHA above (or any new SHA after actual formatting). The coordinator independently owns baseline validation, acceptance and terminal readback. Do not use the historical pre-repair test SHA as the corrected candidate identity.

Exactly one corrected typed result is published for the original task and this new attempt. Its transport response is preserved in the final worker handoff; this report does not predict publication success.

## Original attempt record (historical)

## State and authority

**FACT: BLOCKED / UNACCEPTED.** No product acceptance or ACCEPTED_GREEN claim is made. This is a partial independent QA handoff, not a ReleaseGate producer or terminal reducer decision.

- Task: `agentfield:v2-r15-independent-native-mvp-qa-20261014-r1`
- Attempt: `attempt-bebdb38a-7f00-4f3a-ba71-9918002396d5`
- Session: `manual-20261014-basefive-r15-native-qa-agentfie-5d0a6ff6f619b999`
- Target root: `D:/Users/NIKITA/Documents/DEV/universal-solver-site/agentfield`
- Launch Git identity: `44167dc1050f5bff79abc32fab19c0bb50643093` (supplied, not a clean-checkout attestation).
- Manifest SHA-256: `78732ac44967104d0cbaca650f2556bc262e0f12223f68e2b6d947398abb903d` (supplied).
- Existing BMad source of truth: `docs/acceptance/c4-releasegate-reducer-frontier-20261013.md`; unchanged.
- Stage/gate: implementation evidence and independent native QA before reducer acceptance. The existing C4 gate remains blocked.
- Exclusive changes: this report and `control-plane/internal/storage/northstar_r15_independent_live_qa_test.go`.

BMad Help was loaded first, followed by Quick Dev and verification-before-completion. Canonical BMAD-MNNZ is the workflow source, not the project. Configuration scripts, Git inspection, sprint-status writes, review subagents, new spec files, commits, and editor commands are unavailable or outside this manifest; those workflow steps were not executed. No complete BMad workflow claim is made.

## Local Definition of Done and useful slice

The full requested DoD requires current migration036 and HTTP/C4 source signatures, exact source/test bindings to the dirty checkout, real parallel same-key single-execution behavior, digest/target negatives, owner isolation, durable reopening/crash behavior, native focused and affected baseline receipts, and independent reducer handoff. Production SQL/source and native coordination state must remain untouched.

The existing durable mechanism is already implemented in the authorized storage source; no replacement implementation is needed. The new test file did not exist. The smallest safely grounded slice adds an independently declared compile-time storage API contract, a real nil-payload negative assertion, and authorized-file fingerprint logging. It does not substitute these checks for database or HTTP tests.

Useful-work sizing: initial full-slice estimate 20–30 minutes if database setup, migration, handler, and runner evidence are available. After inspection, the authorized partial slice is approximately 5–10 minutes; further useful integration work is externally blocked. No duration quota was padded, and no exact elapsed work measurement is claimed.

## Exact current file receipts

The following SHA-256 values were returned by scoped line-range reads of the current files:

| File | Current SHA-256 |
|---|---|
| `control-plane/internal/storage/execution_idempotency.go` | `7ff044cdce4e1ec772e3e79a6d4e074e9fece17c2a6857bdf0f4552a1db9cfe2` |
| `control-plane/internal/storage/northstar_r15_independent_live_qa_test.go` | `1f5a3a95d45b5689b826d3d87ee94b032a343c17e827444ef8f84e8ea8fae449` |
| `docs/architecture/basefive-r14-contracts.json` | `02f54cd157d76cf10ccbc07a15e9e595289e8e0f911414adde8b25e68e724185` |

These are exact per-file identities, not a complete immutable dirty-checkout snapshot. Branch, status, dependency versions, migration SHA, current handler SHA and the affected package source set were not observable through the authorized interfaces. The runner's `source_identity_sha256` equals the new test file SHA; it must not be represented as proof of whole-repository identity.

## Source-backed API and oracle review

**FACT:** The current storage interface is:

```go
LookupAsyncExecutionIdempotency(ctx context.Context, scopeKey, target, requestKey, bodyDigest string) (*AsyncExecutionIdempotencyResult, error)
CreateAsyncExecutionIdempotent(ctx context.Context, scopeKey, target, requestKey, bodyDigest string, execution *types.Execution) (*AsyncExecutionIdempotencyResult, error)
```

The result fields are `Execution *types.Execution`, `Created bool`, and `Conflict bool`. Lookup selects by `(scope_key, request_key)` and checks both target and digest before loading the canonical execution. Create rejects a nil execution before touching storage, inserts the binding with `ON CONFLICT (scope_key, request_key) DO NOTHING`, inserts the execution in the same transaction only for a winning binding, then commits. These are inspected implementation facts, not runtime database proof.

**FACT:** A changed owner scope is not rejected by this storage API: a different `scope_key` is a distinct namespace and a missing binding returns an empty result. Therefore the proper independent owner oracle is no disclosure/replay of another owner's canonical execution, not unconditional rejection of every new owner. Authentication/rejection must be checked at the current HTTP boundary. No test expectation was changed to conceal this distinction.

The R14 architecture contract records `ExecuteAsyncHandler(store ExecutionStore, payloads services.PayloadStore, webhooks services.WebhookDispatcher, timeout time.Duration, internalToken string) gin.HandlerFunc`, an HTTP adapter to this optional durable interface, verified-caller/target scope construction, and a process-local queue. Its handler SHA is historical evidence only. This attempt cannot independently confirm the current HTTP handler, router registration, C4 ReleaseGate API, migration schema, DTO version or restart dispatch. Durable binding persistence alone would not prove post-crash queue dispatch or exactly-once agent execution.

## New independent checks

- The test-local interface independently declares the expected API and requires both `LocalStorage` and `AsyncExecutionIdempotencyStore` to satisfy it at compilation.
- `TestR15NilExecutionRejectedBeforeDatabaseAccess` calls the actual production method with a nil receiver and nil execution. It requires a non-nil error and nil binding, so premature database access causes a test failure rather than a mocked success. No database is opened.
- `TestR15AuthorizedSourceFingerprints` logs current SHA-256 values for the authorized storage source, the test itself and the architecture contract. It fails on unreadable/empty files. It is an evidence helper, not a behavioral positive scenario or historical-hash equality assertion.
- The tests use only the Go standard library and the existing `pkg/types` import already observed in production. No dependencies, production source, migrations, network clients, live SQL databases, native task state or claims were changed.

## Actual native runner receipt

Authorized invocation:

```text
go_wave.run_project_tests({"file":"control-plane/internal/storage/northstar_r15_independent_live_qa_test.go"})
```

Exact returned fields:

```json
{
  "status": "FAIL",
  "code": null,
  "test": "control-plane/internal/storage/northstar_r15_independent_live_qa_test.go",
  "engine": "docker:go-test",
  "source_identity_sha256": "1f5a3a95d45b5689b826d3d87ee94b032a343c17e827444ef8f84e8ea8fae449",
  "exit_code": 1,
  "signal": null,
  "error": null,
  "duration_limit_ms": 120000,
  "output_truncated": false,
  "stdout": "",
  "stderr": "go: cannot find main module, but found .git/config in /workspace\n\tto create a module there, run:\n\tgo mod init\n",
  "package": "./control-plane/internal/storage",
  "go_image": "golang:1.24.0-alpine"
}
```

Classification: **INFRA BLOCKED**, before compilation. No test ran, no test count or PASS is inferred, and compile compatibility remains unverified. The suggested `go mod init` was not executed: changing project module structure would be an unauthorized and incorrect worker-side workaround. No identical runner retry was made. Affected baseline regression was not attempted against the same broken module-root configuration.

Coordinator/runner-owner recovery: execute from the existing `control-plane` module root, preserving this sticky attempt and test file. Required focused command is `go test ./internal/storage -run TestR15 -count=1`; add `-v` when collecting fingerprint log output. Affected baseline after runner recovery is `go test ./internal/storage -count=1`, subject to coordinator authorization. These are next commands, not executed receipts.

## Acceptance matrix and blockers

| Gate | Status | Evidence / missing requirement |
|---|---|---|
| D1 current storage API and per-file SHAs | Partial source evidence | Exact API and three current file SHAs above; complete dirty snapshot missing |
| D1 migration036 and current HTTP/C4 API | BLOCKED | Files and current producer receipts absent from read scope; R14 contract is not current source verification |
| D2 actual positive, digest/target negatives | NOT RUN | Live SQL initialization/schema not exposed; no parallel/replay scenario constructed |
| D2 foreign owner / restart / crash | NOT RUN | Owner semantics reviewed; no persistence/reopen or process-crash assertion executed |
| D3 independent oracle in own file | Partial, UNVERIFIED | Independent API signature and nil-input oracle authored; no live database oracle |
| D4 focused native execution | INFRA BLOCKED | Exact runner exit 1 before compilation |
| D4 baseline affected regression | NOT RUN | Same module-root blocker; baseline not authorized through an exact test target |
| D5 worker publication / independent reduction | Pending publication / UNACCEPTED | Typed BLOCKED report required; terminal readback belongs to coordinator |

Additional read authorization is required for `control-plane/internal/handlers/execute.go`, the actual `036_execution_request_idempotency.sql` migration location, and the current storage initialization/migration setup plus adjacent SQL QA tests. Their exact setup paths cannot be inferred from the one authorized source file; the coordinator should supply them without broadening worker writes. The native ReleaseGate producer identity/API remains absent from the active reducer artifact and must not be conflated with async execution idempotency.

## Self-review and handoff

- Scope: exactly two new authorized files; no production source edits, external sends, live SQL mutations or coordination administration.
- Compatibility: test package and production import pattern preserved; independent method signatures match inspected source. Build/test compatibility is still unverified because the runner never compiled.
- Coverage honesty: nil rejection is the only authored runtime behavior assertion; no SQL/migration/parallel/restart/HTTP acceptance is claimed.
- Evidence honesty: launch Git identity and historical handler SHA are not presented as a current immutable checkout binding.
- Existing source of truth is unchanged. This report is the explicitly authorized QA evidence artifact, not a competing plan.
- Next action: repair the native runner's module root and provide narrowly authorized database/handler/migration setup evidence; resume this same owner to add and execute the full positive/negative/restart oracle, then run affected regression and independent reducer review.

The immutable publication response will be returned in the worker's final handoff. This file is written before the one permitted publication call and does not predict transport success or acceptance.

===== BEGIN PHYSICAL FOOTER =====
# CrewSync Result Contract — GO Core Physical Footer v1

This project-agnostic footer is authoritative for one bounded worker attempt. The coordinator supplies the trusted binding; the worker must copy it exactly and must not invent, substitute, or broaden any identifier.

The only worker result shape is `go-core.worker-result.v1` with exactly these top-level fields:

`schema, task_key, project, attempt_id, session_id, producer, outcome, summary, functional_delta, checks, changed_files, evidence_refs, remaining, questions, next_action, acceptance, recorded_at`.

`producer` is exactly `{thread_id, role}`. `outcome` is one of `CANDIDATE, BLOCKED, FAILED, NEEDS_CLARIFICATION, REPAIR_REQUIRED`. `checks` contains `{name, status, evidence_ref}`. `acceptance` is exactly `{level:"UNACCEPTED"}`.

The report is evidence, never task administration. Do not claim DONE, ACCEPTED, or ACCEPTED_GREEN; do not self-accept; do not call task completion; do not include credentials or secrets. The coordinator and independent reviewer own acceptance.

Before the final answer, CALL ONLY `go_wave.publish_worker_result` exactly once with the complete report. Include the returned result status and readback in the final evidence. Never call raw CrewSync administration, mutate claims or locks, or create/complete tasks. If publication is unavailable, report the blocker and preserve the typed repair evidence.

The result must retain the exact trusted task key, project, attempt ID, and session ID. The initial producer thread_id may be null while the provider thread is pending; a later coordinator readback verifies the actual thread. Changed files must stay inside the manifest write scope.
===== END PHYSICAL FOOTER =====
