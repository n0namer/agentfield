# Async idempotency live-SQL counterexample QA — 2026-10-13

## Scope and safety

- Task: `agentfield:v2-async-idempotency-runtime-counterexample-qa-20261013-r4`
- Attempt: `attempt-b843ad9d-d13d-4c64-8378-6c41b9d70cd9`
- The QA artifact uses only disposable SQLite files under `testing.T.TempDir()`.
- No production migration, ingress, service restart, external effect, lease, task, or deployment was touched.
- Production source was read only. The only code artifact is an independent test in the authorized write scope.

## Source-backed classification

The three reported native-Go failures are not, by themselves, evidence of a product defect. They assert identifier spellings that differ from the implementation:

1. **Missing `idempotency_key` literal** — the handler reads the `Idempotency-Key` header, trims and validates it, then passes it as `requestKey`; migration 036 and storage persist the value as `request_key`.
2. **Missing `RequestHash` / `PayloadHash` comparison literal** — the handler hashes the exact request body with SHA-256 and passes the hexadecimal value as `bodyDigest`; storage compares `storedDigest != bodyDigest` and reports `Conflict` on mismatch.
3. **Missing `RowsAffected` / stale literal** — absence of those words in the previously locked QA target is only a static-fixture mismatch. The new runtime counterexample directly checks SQLite compare-and-swap row counts. The authorized read scope did not include the general execution update implementation, so this attempt does not claim that its internal CAS is proven by source inspection.

**Classification:** the three original failures are false-positive static assertions unless independent runtime execution demonstrates a semantic failure. The concurrent read-then-insert path remains an explicit product-risk hypothesis: two absent-row transactions may surface a SQLite lock or a PostgreSQL unique violation instead of returning the canonical replay. The new test is designed to settle that question against the real storage methods.

## Independent behavior test

`control-plane/internal/storage/execution_idempotency_runtime_counterexample_test.go` SHA-256 after the final edit: `42a4732c9ed2d8b464521112b55efa42796ae6a799b0ab99db5d3b4bc918cd40`.

The test calls the real `LocalStorage` methods against an isolated SQLite database and covers:

- initial durable binding and execution creation;
- same-key/same-body replay after closing and reopening the database;
- payload-digest mismatch conflict;
- target mismatch conflict;
- owner/target scope isolation for the same request key;
- two concurrent same-key creators returning one canonical execution and producing one binding plus one execution row;
- SQLite compare-and-swap semantics where the first token-matched update affects one row and a stale update affects zero rows without changing terminal status.

The test deliberately discovers the SQLite driver already registered by the storage package and does not add a dependency. It injects only the disposable `*sql.DB` into a zero-value `LocalStorage`; all schema and data are test-local.

## Validation receipts

### Focused Go test

Requested validation: `go test ./internal/storage -run Idempotency -count=1` from `control-plane`, with the user-installed Go 1.26.5 and D cache.

The authorized runner call for the exact test file returned **FAIL before compilation**:

- engine: `docker:go-test`
- image: `golang:1.24.0-alpine`
- source identity: `d96f6c47bd5a037644869781a97e2c91bdfb47fbe0e773f627a29fc6fa33fec1`
- exit code: `1`
- stderr: `go: cannot find main module, but found .git/config in /workspace`

This is an infrastructure/cwd/toolchain mismatch, not a test result. The runner used `/workspace` instead of the `control-plane` module and Go 1.24.0 instead of the required Go 1.26.5. The denied/misconfigured call was not repeated.

### Worker-result repair-loop test

The authorized Node runner call for `test/worker-result-repair-loop.test.mjs` returned `test_outside_task_write_scope`. It did not execute. The denied call was not repeated.

## Result and remaining verification

- No production defect was proven or repaired in this attempt.
- The original three failures are classified as overstrict static QA assertions from source evidence.
- Runtime confirmation is **infrastructure-blocked** because the only authorized Go runner did not enter the module or use the required toolchain.
- The concurrency hypothesis is intentionally not resolved by assertion. If the focused test fails at `TestIdempotencyConcurrentSameKeyCreatesOneCanonicalExecution`, the same original producer should minimally repair `CreateAsyncExecutionIdempotent` to recover the unique/lock race by loading and comparing the winning binding, then rerun this exact test.
- Independent QA/reducer must run the focused command from `control-plane` with Go 1.26.5 and preserve the full output before acceptance.

Exact next command in an authorized correctly configured environment:

```text
cd control-plane
go test ./internal/storage -run Idempotency -count=1
```
