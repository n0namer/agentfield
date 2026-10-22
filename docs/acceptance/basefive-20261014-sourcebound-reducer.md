# BaseFive Source-Bound Release Reducer — 2026-10-14

## Authority and scope

- Task: `agentfield:v2-basefive-sourcebound-release-reducer-20261014-r4`
- Attempt: `attempt-77cd16bd-ad53-40f3-a479-30617977560f`
- Session: `manual-20261014-basefive-northstar-hyper-r4-age-18b3e913b2682dd2`
- Project: `agentfield`
- Launch Git identity: `44167dc1050f5bff79abc32fab19c0bb50643093`
- Active BMad source of truth: `docs/acceptance/c4-releasegate-reducer-frontier-20261013.md`
- Reducer write scope: this document only
- Decision: `BLOCKED` / `UNACCEPTED`

This document reduces permitted source and QA evidence. It does not change production code, migrations, native task state, CrewSync claims, leases, release state, or acceptance state.

## D1 — Source and contract map

The permitted production source exposes this durable async idempotency contract in `control-plane/internal/storage/execution_idempotency.go`:

```go
type AsyncExecutionIdempotencyStore interface {
    LookupAsyncExecutionIdempotency(ctx context.Context, scopeKey, target, requestKey, bodyDigest string) (*AsyncExecutionIdempotencyResult, error)
    CreateAsyncExecutionIdempotent(ctx context.Context, scopeKey, target, requestKey, bodyDigest string, execution *types.Execution) (*AsyncExecutionIdempotencyResult, error)
}
```

- Source-file SHA-256: `7ff044cdce4e1ec772e3e79a6d4e074e9fece17c2a6857bdf0f4552a1db9cfe2`
- Git/source identity: `44167dc1050f5bff79abc32fab19c0bb50643093`
- Result DTO: `AsyncExecutionIdempotencyResult{Execution, Created, Conflict}`.
- Source-backed behavior: lookup returns the canonical execution for matching target and digest; mismatches return `Conflict`; creation uses a transaction and `ON CONFLICT (scope_key, request_key) DO NOTHING`, then re-reads the existing binding.

This is one source-bound interface/DTO signature. It is evidence for the native SQL idempotency boundary, not proof that a separate C4 ReleaseGate DTO or producer exists. No C4-specific interface/version receipt is present in the permitted evidence.

## D2 — Positive and adversarial/restart assertions

The independent native QA artifact `docs/acceptance/northstar-independent-native-qa-agentfield-20261014.md` records these bounded assertions in test source:

1. **Positive concurrent duplicate:** two independent SQLite connections submit the same scoped request; both calls succeed, exactly one result is `Created`, both resolve to one canonical execution, and exactly one binding/execution row remains.
2. **Negative conflicting digest:** replay with a changed body digest returns `Conflict` and does not create another execution.
3. **Negative conflicting target:** replay with a changed target returns `Conflict` and does not create another execution.
4. **Restart recovery:** after database close/reopen, the original binding replays the original execution and a replacement execution does not win.

The QA artifact reports test SHA-256 `202d910ddc7a18d03ab68894384c9539d6ce475275324446c5c24a4e126684f7`, but its focused runner failed before compilation. Therefore these are source/test assertions, not runtime-proven PASS results in this reducer.

## D3 — Receipt reduction

### Independent SQL receipt

- QA task: `agentfield:v2-independent-native-longbundle-qa-20261014-r1`
- QA attempt: `attempt-923917a3-74c2-4cbd-94fa-3ffbbafe1062`
- QA session: `manual-20261014-base5-independent-native-qa-hyp-99e31d96826794a6`
- QA state: candidate / `UNACCEPTED`
- Focused command recorded: `go_wave.run_project_tests({"file":"control-plane/internal/storage/northstar_independent_sql_oracle_20261014_test.go"})`
- Recorded result: `FAIL`, exit `1`, before compilation
- Recorded stderr: `go: cannot find main module, but found .git/config in /workspace` followed by `to create a module there, run: go mod init`
- Classification: `INFRA BLOCKED`; the runner used `/workspace` rather than the `control-plane` Go module.

### Prior C4 reducer receipt

`docs/acceptance/c4-releasegate-reducer-frontier-20261013.md` records that C4 producer receipts, C4 independent QA receipts, interface/DTO versions, and native failure/restart E2E evidence were absent. Its prior async-idempotency runner also failed before compilation with the same module-root problem. That evidence remains `BLOCKED` / `UNACCEPTED` and cannot be upgraded to C4 evidence.

### Python and live-runtime comparison

The permitted SQL QA artifact mentions a Python longbundle producer artifact (`docs/acceptance/longbundle-harness-live-result-20261014.md`) with runner source SHA `06c6eccd3f17a72a802baead62ddfa534b8bfbcc07c76c039b0dbbe485f9cfa8`, but explicitly states it is not a native SQL producer receipt and its pytest run was blocked by an unavailable `.venv`. No live runtime receipt, native producer receipt, or independent C4 reviewer receipt is available in this attempt's permitted evidence.

## D4 — Focused validation and affected regression

This reducer attempted each distinct authorized test target once:

- Focused native target `control-plane/internal/storage/northstar_independent_sql_oracle_20261014_test.go`: `GO_WAVE_NODE_ERROR:test_outside_task_write_scope` for this attempt; no compilation or runtime status was obtained.
- Affected regression target `control-plane/internal/storage/northstar_sql_idempotency_20261013_test.go`: `GO_WAVE_NODE_ERROR:test_outside_task_write_scope`; no regression PASS is claimed.

The earlier independent QA artifact contains the more specific native runner receipt above, which is retained as `INFRA BLOCKED` evidence. Repeating denied calls would not repair the authorization boundary. No source repair is justified inside this Markdown-only write scope.

## D5 — Reducer verdict and handoff

### Evidence that is present

- One source-bound interface/DTO signature and exact source-file SHA are recorded.
- Positive duplicate, two conflict negatives, and restart assertions are identified from the independent QA test artifact.
- SQL concurrency semantics and canonical replay behavior are visible in the permitted production source.
- Native runner failures are preserved with command, exit/error, and infrastructure classification.
- The launch Git identity and all known task/attempt/session identities are retained without inventing missing C4 identities.

### Evidence that is missing

- Compiled native focused-test output from the correct `control-plane` module root.
- A permitted affected regression execution with exit status.
- Exact C4 ReleaseGate producer and independent reviewer task/attempt/session receipts.
- C4 DTO/interface version identifiers and exact SHAs for every producer/consumer boundary.
- Live native task-to-result success, producer-failure, and restart/recovery receipts.
- Authoritative coordinator terminal readback.

### Verdict

`BLOCKED` / `UNACCEPTED`. The available evidence supports a source-level SQL idempotency contract and test design, but not a runtime PASS or C4 ReleaseGate acceptance. The infrastructure failures do not prove a product defect; they prevent the required acceptance decision. No `ACCEPTED_GREEN` claim is made.

## Self-review

- Changed path is limited to `docs/acceptance/basefive-20261014-sourcebound-reducer.md`.
- No production, test, migration, task, lease, or release state was modified.
- No missing C4 task ID, DTO version, runtime result, or SHA was inferred.
- Positive and negative/restart behavior is reported as asserted by the QA artifact, while execution remains blocked.
- The next acceptance owner must obtain correct-module native execution, independent C4 producer/reviewer receipts, and authoritative terminal readback before changing the verdict.
