# Base20 Independent Native Control-Plane QA — 2026-10-13

## Scope and authority

This is an independent, test-only QA/reducer artifact for the native control-plane lifecycle:

```text
initial request -> durable execution record -> callback -> final persisted readback
```

Authorized additions only:

- `control-plane/internal/handlers/base20_independent_execution_qa_test.go`
- this report

No production handler, storage, async idempotency, lease, deployment, source-owner lock, or Ruflo Native task state was changed. This report is a candidate/reducer recommendation, not acceptance of any original implementation.

## Candidate revisions and producer trace

The three supplied candidate artifacts were inspected at these exact SHA-256 revisions:

| Candidate | Artifact | SHA-256 | Claimed slice |
| --- | --- | --- | --- |
| durable storage recovery | `control-plane/internal/storage/execution_record_recovery_wave_test.go` | `2cbe6e43c2d142f418d6d0c520d1ecab36dc1a4cfc77a6f3b1b11ec5f4f4b3f3` | local SQLite restart, replay, terminal state fencing, unique record |
| callback terminal fencing | `control-plane/internal/handlers/node_callback_terminal_wave_test.go` | `feb5aabd0c5258789ad8d3f6879e38e6194a50c108cffc442d9712baf631f19e` | queued/running/succeeded callback lifecycle and terminal duplicate handling |
| runtime log correlation | `control-plane/internal/handlers/runtime_log_observability_wave_test.go` | `f26d51f0ec2f3df47455ff2564f327e5ad2352833e6d81a13f68c98f5a249339` | sanitized duplicate acknowledgement correlation to one durable execution |

The persistence owner was read at `control-plane/internal/storage/execution_records.go` SHA-256 `099014f324b4204cf02a88c9b372db93a5baf51cecdd2f8cde1c7f55d9223ec0`. It creates rows by `execution_id`, reads them back by the same ID, and applies state-transition validation in its update path. The supplied storage acceptance artifact identifies existing transition validation and database uniqueness as the intended durable fencing mechanism.

The BMad source-of-truth gate is an independent code review/QA gate. `bmad-code-review` step 1 required unavailable shell/git diff construction and an interactive confirmation; that method step is BLOCKED by this scoped MCP-only worker, so no completion of that external workflow is claimed. BMad Help, edge-case review, adversarial review, and verification-before-completion were loaded as advisory methods. Source evidence and executable test coverage remain the governing evidence.

## Independent executable adversarial regression

`control-plane/internal/handlers/base20_independent_execution_qa_test.go` creates an on-disk local storage instance, uses the real callback handler with a temporary payload store, then closes and reopens the storage instance. It covers:

1. Positive callback lifecycle: queued -> running -> succeeded with durable result readback.
2. Duplicate succeeded callback: the first terminal result remains authoritative; no second durable row appears for the run.
3. Late/stale callback: late `running` and conflicting `failed` after success are non-OK and leave the stored terminal result unchanged.
4. Mismatched execution ID: a callback addressed to an unrelated ID is non-OK and cannot alter the original terminal execution.
5. Delayed storage write: a callback before the durable row exists is non-OK; later creation starts queued without the premature result, then a subsequent valid callback terminalizes it.
6. Restart readback: after `Close` and fresh `Initialize` with the same paths, the terminal execution and authoritative payload remain readable.
7. Missing run/session binding: the current callback contract accepts a record whose `RunID` and `SessionID` are empty. This is deliberately made visible rather than mislabeled as isolation/fencing.

## Findings and reducer recommendation

### ACCEPT_SCOPED, contingent on focused Go evidence

The three candidates form a coherent in-process lifecycle evidence chain:

- Storage recovery proves persisted terminal readback, idempotent same-terminal replay, rejected reopening, and uniqueness.
- Callback tests prove direct handler delivery preserves the first terminal result under duplicate and late deliveries.
- Log-forensics tests prove two sanitized acknowledgement lines can correlate to one durable execution row, while refusing absent correlation, node mismatch, malformed payload, and sensitive values.
- The independent test joins those seams using the real local store and handler, adding delayed-write, unrelated-ID, restart, and missing-binding observations.

This remains **UNACCEPTED** until an independently captured focused Go test result is available and source SHA consistency is checked again by the verifier.

### RED: callback identity is not bound to run/session/instance

The supplied callback acceptance artifact states that `executionStatusUpdateRequest` has no `instance_id`, lease token, fencing epoch, or callback signature. This independent test additionally shows that a preexisting record with empty `RunID` and `SessionID` reaches terminal success through the callback path. Therefore the native callback contract does not prove run/session binding or foreign-instance rejection.

**Reducer action:** keep this as a same-owner repair request for the callback/handler contract owner. Require a dedicated source change and regression proving that a callback is authenticated/fenced to the durable execution's run, session, and active instance/lease before terminal mutation. Do not weaken the test to claim that path is protected.

### RED: stale-reaper production symptom remains uncorrelated to source revision

The runtime-log candidate reports PostgreSQL `SQLSTATE 42601` stale-timeout preparation failures, while the inspected storage source documents dialect-aware timestamp handling. Without a live image digest, running source revision, database dialect/config, and sanitized statement reproduction, this is a product-impacting symptom but not an approved source repair.

**Reducer action:** verify deployed image/source/dialect first, then assign the smallest repair to the actual owner only if the checked-in SQL path reproduces.

### BLOCKED: live localhost health is not native E2E evidence

This worker has no authorized HTTP, process, Docker, or loopback health probe tool. No request to localhost was issued. Consequently:

- **Service health:** BLOCKED/UNVERIFIED. A future authorized operator may safely query the documented loopback health endpoint and capture process/container identity.
- **One-task native E2E acceptance:** BLOCKED/UNVERIFIED. A healthy endpoint alone cannot establish a persisted request-to-callback-to-restart lifecycle; it needs an independently correlated execution ID, run ID, callback outcome, and durable readback from the same build and source snapshot.

### Explicitly unaccepted P0

The in-flight async-idempotency P0 remains **UNACCEPTED**. Repeated acknowledgement log lines and one durable row are useful observations, but they do not independently prove request admission idempotency, same-key/different-body conflict behavior, callback authentication, or cross-instance fencing.

## Validation evidence

Requested commands from `control-plane`:

```text
go test ./internal/handlers -run Base20Independent -count=1
go test ./internal/storage -run Execution -count=1
```

At 2026-10-13, this worker attempted both nearest scoped validations through the only authorized test interface:

```text
go_wave.run_project_tests({"file":"control-plane/internal/handlers/base20_independent_execution_qa_test.go"}) -> GO_WAVE_NODE_ERROR:test_file_not_allowed
go_wave.run_project_tests({"file":"control-plane/internal/storage/execution_record_recovery_wave_test.go"}) -> GO_WAVE_NODE_ERROR:test_file_not_allowed
```

Both runner denials are **INFRA BLOCKED**, not PASS. No identical retry was performed. Independent QA must run both requested commands with a Go-capable executor from `control-plane`.

## Verifier checklist

1. Re-read the four listed source hashes before treating this report as current; fail closed on any mismatch.
2. Run both requested Go commands from `control-plane` with a Go-capable runner and retain untruncated output.
3. Confirm the new test's callback-before-record behavior is non-OK and the subsequent record is still queued.
4. Confirm duplicate terminal result preservation and exactly one primary run row after restart.
5. Treat empty `RunID`/`SessionID` callback acceptance as a RED contract finding, not a passing security property.
6. Obtain an authorized live-service evidence bundle separately; do not substitute a health response for native E2E acceptance.

## Status

- Independent QA outcome: `CANDIDATE` only after focused runner evidence is recorded.
- Product acceptance: `UNACCEPTED`.
- Next action: independent verifier runs the Go commands, rechecks source SHA, and routes the callback identity/fencing RED finding to the original callback contract owner without accepting async-idempotency P0.
