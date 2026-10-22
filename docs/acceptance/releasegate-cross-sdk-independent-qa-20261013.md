# Independent QA — release-gate live cross-SDK E2E — 2026-10-13

## Authority and gate

- Independent QA task: `agentfield:v2-releasegate-independent-qa-20261013-r1`
- Attempt: `attempt-4d4f06e4-7ee5-4663-a301-b3b77b5c94ce`
- Crew session: `manual-20261013-native-releasegate-independent--f95a68d748748622`
- Reviewed producer task: `agentfield:v2-releasegate-live-cross-sdk-e2e-20261013-r1`
- Producer attempt recorded by its artifact: `attempt-19e9b2a8-e7b5-4bfa-8197-7994af097743`
- Producer artifact status: **BLOCKED / UNACCEPTED**.
- Reviewer recommendation: **REJECT release-gate acceptance; CANDIDATE / UNACCEPTED only**.

This review did not contact the localhost service, use either credential, submit or cancel an execution, modify producer files, mutate CrewSync/Ruflo state, or perform any destructive action.

## Evidence inspected

- Producer test: `sdk/python/tests/test_releasegate_live_cross_sdk_20261013.py`
- Producer acceptance record: `docs/acceptance/releasegate-live-cross-sdk-20261013.md`
- Producer-recorded test SHA-256: `764cbaf681037405ffd15ddf82a6693ac0820e5b65a0e1ba1eb065478f5bed49`
- Independent challenge test: `sdk/python/tests/test_releasegate_cross_sdk_independent_qa_20261013.py`
- Final independent challenge test SHA-256: `8b41960d570d1cbc2bde3da1badb18d707970d9b51d04a8e60e1c031ca7d4c41`
- Authorized runner source identity before the self-review correction: `3b972135ec89ff87255d9b49e2950d4da10eb9881432b6250e22c8acae2c6602`

The producer record states that its runner stopped before collection with `PYTHON_VENV_UNAVAILABLE`, that no request executed, and that no live persistence, Go response, restart, or denial path passed. It describes only a fresh Python client readback and explicitly does not claim a control-plane process or database restart.

No permitted artifact provides a real Git HEAD, immutable dirty-tree/snapshot identity, an exact original CrewSync status readback, or the producer's typed worker-result publication response. Those required cross-checks therefore remain unavailable rather than inferred.

## Independent challenge design

The separate QA test is deliberately offline and non-destructive. It contains:

1. An immutable producer-source SHA comparison against the producer-recorded digest.
2. A fail-closed assertion that the producer remains blocked and contains no live success receipt.
3. A check for missing Git/process/publication/CrewSync attestation.
4. An adversarial foreign-owner credential check showing that nonempty and distinct strings plus an allowed `401` cannot prove the second credential is valid.
5. A fake-registry boundary showing that an arbitrary nested echo object with a matching name and `go` marker is accepted as node provenance.
6. A negative terminal-mutation check showing that accidental persisted state change is detected.
7. A route/auth ambiguity check showing that `404` is accepted as terminal-state denial.
8. A crash/retry boundary showing a transport crash propagates after exactly one mutation attempt, with no hidden retry.
9. A correlation check showing the request ID can be satisfied by echoed input anywhere in the terminal object, without proving Go-produced output.

## Adversarial findings

1. The producer live test never executed; no HTTP persistence behavior is proven.
2. The producer explicitly tests only fresh-client readback, not service, process, or database restart persistence.
3. The foreign credential is never independently authenticated as a valid second owner.
4. Accepting `401` for the foreign read allows an arbitrary invalid credential to masquerade as owner isolation.
5. Registry discovery recursively accepts a matching nested object and does not attest the serving process, build, or source.
6. A fake localhost fixture can satisfy health, registry, execution, readback, and OpenAPI responses because no process provenance is bound.
7. Terminal mutation accepts any `4xx`; `401` or `404` can reflect invalid auth or routing rather than terminal-state enforcement.
8. OpenAPI templates with additional unresolved path parameters can produce an accepted `404` without exercising the intended mutation handler.
9. Request correlation is searched anywhere in the persisted object after the same ID was supplied as input, so input echo alone passes.
10. Registry metadata proves only a string containing `go`, not that the execution response was produced by the Go SDK target.
11. No real Git HEAD or immutable working-tree snapshot binds the reviewed source to a repository revision.
12. No typed producer worker-result publication receipt or exact original CrewSync status readback is available in the permitted evidence.
13. No process-level execution outcome exists because pytest stopped before collection in both the producer and independent attempts.
14. Exact full-object equality may detect mutation, but without a live run it provides no evidence that terminal cancellation or timeout preserves state.

## Focused validation receipt

Authorized call:

```text
go_wave.run_project_tests({"file":"sdk/python/tests/test_releasegate_cross_sdk_independent_qa_20261013.py"})
```

Exact result:

```text
status: BLOCKED
code: PYTHON_VENV_UNAVAILABLE
engine: python:pytest
source_identity_sha256: 3b972135ec89ff87255d9b49e2950d4da10eb9881432b6250e22c8acae2c6602
exit_code: null
stderr: A regular local .venv Python executable is required.
```

The runner stopped before collection. This is an infrastructure blocker, not PASS and not a product failure. The identical denied call was not repeated. Self-review then corrected one offline correlation assertion that would have matched wording in the producer's error message rather than the intended structure; the final test SHA is recorded above and remains unexecuted because the same runner prerequisite is still absent. No affected live regression was run because the only authorized Python runner was unavailable and this reviewer must not invoke destructive service actions.

## Required same-producer sticky repair

The original producer, not this independent reviewer, should retain ownership of any repair:

1. Provision the repository `.venv` and run the original live test through the authorized runner with untruncated output.
2. Prove both credentials are valid principals before the isolation probe; do not count an authentication `401` as foreign-owner isolation.
3. Require the owner mutation request to reach the exact terminal-state guard and distinguish it from auth, route, or unresolved-template failure.
4. Bind the live service to process/build/source evidence and record the real Git HEAD or immutable dirty-tree snapshot.
5. Add a real controlled service/process restart or crash/recovery readback if restart persistence is part of the release claim.
6. Bind the completed execution to Go-produced output rather than a registry string and echoed request input alone.
7. Publish and preserve the typed producer worker-result response and exact original CrewSync status/attempt readback.
8. Re-run this independent challenge file after the Python runner is repaired.

## Reducer action

A separate reducer must keep the product terminal state unaccepted. It may reconsider only after fresh executable evidence closes the credential-validity, process-provenance, exact-handler, restart-persistence, immutable-source, and publication/status gaps. This reviewer cannot accept itself, promote the dependency, integrate Universal Solver, or alter foreign source files.
