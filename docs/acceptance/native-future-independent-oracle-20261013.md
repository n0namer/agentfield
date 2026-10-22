# Native Future Independent Oracle — 2026-10-13

## Authority and scope

- Task: `agentfield:v2-native-future-independent-oracle-20261013-r1`
- Attempt: `attempt-ea78c0f6-797c-43f4-bf33-d469f5d69158`
- Session: `manual-20261013-native-future-independent-qa-wa-d19ab0dbb9b51e53`
- Launch Git identity: `44167dc1050f5bff79abc32fab19c0bb50643093`
- Launch manifest SHA-256: `43f17d6ad8328f95d30a5829423ea17dd9308ab3887839878fae1be828a345e9`
- Prior C4 reducer: `agentfield:v2-c4-releasegate-independent-reducer-20261013-r6`
- Independent oracle source: `sdk/python/tests/test_releasegate_future_idempotency_qa_20261013.py`
- Oracle source SHA-256: `c63207f2d74541bf4bedea5ec3ee3f36ef999076e641e2380242681e539fb851`

This artifact defines an independent future acceptance oracle. It does not mutate native tasks, CrewSync ownership, leases, persistence, release state, or acceptance state. It is separate from the current native end-to-end goal worker and cannot self-accept.

## Current gate

The prior C4 reducer is `BLOCKED` / `UNACCEPTED` because the authorized evidence did not expose a C4 producer receipt, independent C4 QA receipt, real interface/DTO versions, source file hashes, or an executable native failure/restart receipt. This task's read scope likewise does not expose current native implementation files. Therefore no concrete implementation signature is asserted and no live product GREEN is claimed.

The new oracle closes the specification gap without fabricating implementation facts: a future live bundle must provide qualified request, receipt, and persistence signatures and bind every signature path to an exact file SHA-256. The oracle validates those supplied implementation identities before evaluating behavior.

## Reproducible commands

Focused command intended for a repository with a regular local Python environment:

```text
python sdk/python/tests/test_releasegate_future_idempotency_qa_20261013.py -v
```

Live source-bound gate:

```text
set NATIVE_FUTURE_ORACLE_BUNDLE=<absolute-path-to-native-future-independent-evidence.v1.json>
python sdk/python/tests/test_releasegate_future_idempotency_qa_20261013.py -v
```

The second command is a Windows `cmd.exe` illustration only; no shell command was run by this worker. The authorized test engine invocation was:

```text
go_wave.run_project_tests({"file":"sdk/python/tests/test_releasegate_future_idempotency_qa_20261013.py"})
```

## Measurable oracle contract

A live bundle passes only when all conditions below hold.

### Source and C4 DTO boundary

- Schema is `native-future-independent-evidence.v1` and `evidence_mode` is `live`.
- Repository source identity is a lowercase SHA-256.
- Every implementation file used by the boundary has an exact path and lowercase SHA-256.
- `dto_boundary.version` is non-empty.
- `request_signature`, `receipt_signature`, and `persistence_signature` are qualified implementation signatures.
- Every DTO boundary source path is present in the hashed implementation-file set.

The boundary is deliberately implementation-supplied rather than guessed. The current authorized evidence contains no real native signature names. Independent QA must obtain these values from the immutable producer source snapshot and attach them to the live evidence bundle.

### Positive request-to-receipt path

- The initial attempt terminates as `SUCCEEDED`.
- Request, operation, receipt, process-instance, and payload identities are present.
- The terminal receipt is persisted.
- Exactly one result publication is recorded.

### Failure path

- A distinct failure attempt terminates as `FAILED`, never `SUCCEEDED`.
- The failure receipt is persisted and has a non-empty error code.
- The failed result is published exactly once.
- The failure receipt cannot alias the successful receipt.

### Retry and restart recovery

- The retry runs under a different process instance.
- The retry explicitly records restart recovery.
- Request ID, operation ID, receipt ID, and payload SHA-256 match the original successful attempt.
- Publication count remains exactly one; duplicate publication is rejected.
- Persistence evidence proves the original receipt survived restart and supplies a record SHA-256.

### Command receipt

- The live bundle records the exact argument vector, zero exit code, and non-negative duration in milliseconds.
- The oracle prints a canonical bundle SHA-256, source SHA-256, DTO version, and receipt ID only after all live conditions pass.

## Focused test design

The Python file contains five independent checks:

1. Synthetic positive/failure/restart data exercises the full oracle contract.
2. A failure mislabeled `SUCCEEDED` is rejected.
3. A restart with publication count `2` is rejected.
4. Synthetic data is rejected by the live gate.
5. The environment-fed live bundle is skipped when absent and validated with `require_live=True` when supplied.

Synthetic fixtures test the oracle implementation only. They are never product evidence and cannot produce live acceptance because the live gate requires `evidence_mode: live`.

## Actual evidence classification

### PASS

- Target files were absent before this attempt, confirming this exact independent mechanism was not already implemented at the authorized paths.
- The oracle source was written and read back with SHA-256 `c63207f2d74541bf4bedea5ec3ee3f36ef999076e641e2380242681e539fb851`.
- Static self-review confirms explicit guards for source identity, DTO signatures, terminal persistence, failure status, restart process change, identity preservation, and duplicate publication.
- The test design prevents synthetic fixtures from passing the live gate.

These are artifact and contract checks, not executed test PASS claims.

### FAIL

- No product behavior is classified FAIL because the focused Python suite did not start.
- No live native bundle was supplied, so there is no product result to classify.

### SKIP

- Live request-to-receipt-to-persistence-to-restart execution is skipped until the original native producer supplies an immutable live evidence bundle.
- Native deployment, external provider sends, CrewSync terminal readback, and release promotion are outside this worker's authority.
- Current native implementation signatures were not readable in this task scope; none are inferred from filenames, examples, or synthetic fixtures.

### BLOCKED

The single authorized runner call returned:

- Status: `BLOCKED`
- Code: `PYTHON_VENV_UNAVAILABLE`
- Engine: `python:pytest`
- Exit code: `null`
- Duration limit: `120000` ms
- Source identity SHA-256: `c63207f2d74541bf4bedea5ec3ee3f36ef999076e641e2380242681e539fb851`
- Standard error: `A regular local .venv Python executable is required.`

The denial occurred before test execution. It is infrastructure-blocked, not PASS, FAIL, or product GREEN. The same denied runner call was not retried.

## Future-ready dependency graph

| Dependency / owner | Current evidence | Next affected-only action |
|---|---|---|
| Current task `agentfield:v2-native-future-independent-oracle-20261013-r1` | Independent oracle candidate; runner blocked; `UNACCEPTED` | Independent reviewer validates the oracle source and publication receipt. |
| Original native/C4 producer owner | Exact task, attempt, and session IDs absent from authorized evidence | Coordinator resumes the original owner; producer exports immutable source hashes, real qualified DTO signatures/version, and the positive/failure/restart evidence bundle. Do not create a duplicate producer. |
| Original native/C4 independent QA owner | Exact task, attempt, and session IDs absent from authorized evidence | Coordinator resumes the original QA owner to execute the bundle against the immutable producer snapshot. Do not substitute this oracle author as independent acceptance. |
| Prior reducer `agentfield:v2-c4-releasegate-independent-reducer-20261013-r6` | `BLOCKED` / `UNACCEPTED`; prior publication invalid because `recorded_at` was not an ISO timestamp | Re-run reduction only after live oracle evidence and original-owner IDs exist; preserve reducer independence. |
| Prior unrelated async QA `agentfield:v2-async-idempotency-independent-qa-sql-fencing-20261013-r3` | Known task ID, but explicitly not C4/native evidence | Keep within its original async-idempotency scope; do not use it to satisfy this gate. |
| Test infrastructure owner | `.venv` unavailable to scoped Python runner | Provide a regular local `.venv` Python executable, then rerun the exact focused test once. |

Launch dependencies for this task were declared `none`. The rows above are acceptance prerequisites discovered from the prior reducer and the focused runner, not new CrewSync claims or invented task IDs.

## Affected-only repair plan

1. Test infrastructure owner restores the regular local `.venv` required by `python:pytest`.
2. Same task/thread reruns only `sdk/python/tests/test_releasegate_future_idempotency_qa_20261013.py` and records exact counts, exit code, duration, and output.
3. Original producer owner supplies a live bundle with real source/DTO signatures and immutable hashes.
4. Original independent QA owner runs the live gate, verifies positive, failed, and restart paths, and records the resulting bundle digest.
5. Independent reducer consumes those receipts and authoritative coordinator readback; no worker self-accepts.

## Self-review

- Changed files are restricted to the two authorized paths.
- No current native code, test, task state, lease, claim, or release artifact was edited.
- No concrete production signature was invented; synthetic names are confined to fixtures and rejected by the live gate.
- Negative behavior covers false success, duplicate publication, missing durability, source mismatch, and synthetic/live confusion.
- Exact runner output is preserved and is not upgraded to GREEN.
- Acceptance remains `UNACCEPTED`; independent reviewer and coordinator retain terminal authority.
