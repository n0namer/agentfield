# Native Future Oracle Runtime Independent QA — 2026-10-13

## Authority and independent scope

- Current task: `agentfield:v2-native-future-oracle-runtime-independent-qa-20261013-r1`
- Current attempt: `attempt-dd30510a-a36f-4632-940b-8624619919b8`
- Current session: `manual-20261013-future-oracle-runtime-qa-hyperw-21f201374154f176`
- Launch Git identity: `44167dc1050f5bff79abc32fab19c0bb50643093`
- Launch manifest SHA-256: `a129821f7572d3be567891281b708b730564b6400f41a14732565c627f52f095`
- Governing previous worker task: `agentfield:v2-native-future-independent-oracle-20261013-r1`
- Governing previous worker attempt: `attempt-ea78c0f6-797c-43f4-bf33-d469f5d69158`
- Governing previous worker session: `manual-20261013-native-future-independent-qa-wa-d19ab0dbb9b51e53`
- Governing artifact SHA-256 from the trusted launch context: `b1481a2140452ade2dc7698c9a7730028db2920304642c2a743f005b76bfc2e7`
- Oracle source: `sdk/python/tests/test_releasegate_future_idempotency_qa_20261013.py`
- Oracle source SHA-256 reported by the authorized runner: `c63207f2d74541bf4bedea5ec3ee3f36ef999076e641e2380242681e539fb851`

This QA pass is independent of the oracle authoring task. It reads the preexisting oracle and governing evidence, does not overwrite the foreign test, does not mutate native tasks, claims, leases, persistence, release state, or acceptance state, and cannot self-accept.

## BMad gate and source of truth

BMad Help resolves the target project as `agentfield`, not the canonical BMAD-MNNZ repository. The active source of truth is `docs/acceptance/native-future-independent-oracle-20261013.md`. Its current gate is `BLOCKED` / `UNACCEPTED`: the oracle contract exists, but the Python runner lacks a regular local `.venv`, no live bundle is supplied, and the permitted evidence does not expose real producer DTO signatures or an executable native positive/failure/restart receipt.

The appropriate action is verification and affected-only recovery, not Correct Course or a replacement implementation. No competing plan or replacement test was created.

## Original identity recheck

The current launch binds Git identity `44167dc1050f5bff79abc32fab19c0bb50643093`, matching the launch Git identity recorded by the previous worker. The current manifest SHA differs from the previous worker's recorded manifest SHA `43f17d6ad8328f95d30a5829423ea17dd9308ab3887839878fae1be828a345e9`, as expected for a separate independent-QA task and attempt. The previous worker task, attempt, session, governing artifact hash, oracle path, and oracle source SHA are recorded above without substitution.

The authorized runner independently returned the same oracle source SHA-256 recorded by the previous worker: `c63207f2d74541bf4bedea5ec3ee3f36ef999076e641e2380242681e539fb851`. This proves the focused runner addressed the same test source identity; it does not prove that the test executed.

## Exact oracle interface and C4 compatibility boundary

The real test source exposes this validation interface:

```text
validate_native_bundle(bundle: dict[str, Any], *, require_live: bool) -> dict[str, str]
```

The source-backed evidence schema is `native-future-independent-evidence.v1`. Its C4 boundary requires:

- a non-empty `dto_boundary.version`;
- qualified `request_signature`, `receipt_signature`, and `persistence_signature` strings;
- `dto_boundary.source_paths` fully contained in the source file set;
- an exact lowercase SHA-256 for the repository source identity and every implementation file;
- three distinct phases: `initial`, `failure`, and `restart_retry`;
- terminal persisted statuses, with initial and restart success but an explicit failed failure path;
- stable request, operation, receipt, and payload identities across restart;
- a changed process instance and `recovered_from_restart: true`;
- exactly one publication for success, failure, and restart convergence;
- durable persisted receipt evidence and an exact record SHA-256;
- an exact command argument vector, zero exit code, and non-negative duration.

The synthetic fixture declares DTO version `c4-native.v1` and illustrative qualified signatures, but the source explicitly marks it `evidence_mode: synthetic`. `require_live=True` rejects that fixture. Therefore the fixture proves only the oracle's intended contract shape; it is not compatible-production evidence and cannot establish a real C4 producer version or signature.

No real C4 producer DTO/version signature is available in the authorized evidence. Compatibility remains pending until the original producer supplies immutable implementation paths and hashes plus its actual qualified request, receipt, and persistence signatures. No signature is inferred from fixture names.

## Focused runtime attempt

The single authorized invocation was:

```text
go_wave.run_project_tests({"file":"sdk/python/tests/test_releasegate_future_idempotency_qa_20261013.py"})
```

Exact receipt:

- Status: `BLOCKED`
- Code: `PYTHON_VENV_UNAVAILABLE`
- Engine: `python:pytest`
- Exit code: `null`
- Signal: `null`
- Duration limit: `120000` ms
- Output truncated: `false`
- Standard output: empty
- Standard error: `A regular local .venv Python executable is required.`
- Source identity SHA-256: `c63207f2d74541bf4bedea5ec3ee3f36ef999076e641e2380242681e539fb851`

The runner stopped before collection or execution. Test count, pass count, skip count, failure count, and positive/negative/restart runtime outcomes are therefore unknown. This is an infrastructure failure, not a product bug, test failure, PASS, or GREEN. The same denied call was not retried.

## Static positive, negative, and restart review

The source contains five `unittest` cases:

1. a synthetic positive/failure/restart contract case;
2. rejection of a failure attempt mislabeled `SUCCEEDED`;
3. rejection of restart publication count `2`;
4. rejection of synthetic evidence by the live gate;
5. an environment-fed live bundle check that skips when `NATIVE_FUTURE_ORACLE_BUNDLE` is absent.

Static review confirms explicit guards for the major contract conditions, but no case has runtime PASS evidence in this attempt. The live case cannot demonstrate native behavior without both a runnable Python environment and a source-bound live evidence bundle.

Adversarial limitations retained as open evidence requirements:

- no real producer DTO signature or version is exposed;
- no immutable live bundle is exposed;
- no executed collection/test counts exist;
- no command duration below the runner limit exists;
- no real positive receipt exists;
- no real failed receipt exists;
- no real restart process transition exists;
- no real persisted-record digest exists;
- no real duplicate-publication rejection receipt exists;
- no coordinator terminal acceptance readback exists.

## Affected-only repair routing

No new task, claim, lease, queue, or duplicate implementation was created.

| Existing task / owner evidence | Required affected-only action | Duplicate prevention |
|---|---|---|
| Current QA task `agentfield:v2-native-future-oracle-runtime-independent-qa-20261013-r1` | Resume this same task/thread after infrastructure repair and rerun the one focused oracle. | Do not create a second runtime-QA task for the same source. |
| Previous oracle task `agentfield:v2-native-future-independent-oracle-20261013-r1` | Preserve its test and artifact unchanged; use it only as governing evidence. | Do not reimplement or overwrite its foreign test. |
| Prior reducer `agentfield:v2-c4-releasegate-independent-reducer-20261013-r6` | Re-run reduction only after executable live oracle evidence and authoritative source identities exist. | Do not reduce synthetic or blocked evidence. |
| Original native/C4 producer owner | Resume the original producer claim and export immutable source hashes, actual DTO signatures/version, and the live positive/failure/restart bundle. | Its exact task/attempt IDs are absent from authorized evidence; do not invent or duplicate them. |
| Original native/C4 independent QA owner | Resume the original independent QA claim to execute the live bundle against the immutable producer snapshot. | Its exact task/attempt IDs are absent from authorized evidence; do not substitute this worker. |
| Unrelated task `agentfield:v2-async-idempotency-independent-qa-sql-fencing-20261013-r3` | Keep it outside this native/C4 gate. | Do not reuse unrelated async evidence as native evidence. |

## Safe recovery command

After a regular repository-local `.venv` Python executable is available, the coordinator should resume the same task/attempt lineage and invoke exactly:

```text
go_wave.run_project_tests({"file":"sdk/python/tests/test_releasegate_future_idempotency_qa_20261013.py"})
```

For the live gate, the coordinator must first provide `NATIVE_FUTURE_ORACLE_BUNDLE` pointing to the immutable producer bundle in the runner environment, then use the same focused invocation. The resulting receipt must preserve exact collection counts, pass/fail/skip counts, exit code, duration, stdout, stderr, source SHA, and bundle digest. Until then, no runtime or product GREEN is supportable.

## Independent QA conclusion

- `FACT`: The oracle source identity matches the previous worker's recorded SHA-256.
- `FACT`: The oracle's static contract covers synthetic positive, negative, duplicate-publication, and live-gate boundaries.
- `FACT`: The authorized runner was blocked before execution by missing regular local `.venv` Python.
- `FACT`: No test result, live product behavior, or C4 producer compatibility is proven.
- `FACT`: Acceptance remains `UNACCEPTED` and must stay with the independent reviewer, reducer, and coordinator.
