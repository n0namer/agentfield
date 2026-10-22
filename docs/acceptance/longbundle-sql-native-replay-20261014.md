# Longbundle SQL Native Replay — 2026-10-14

## Authority

- Task: `agentfield:v2-longbundle-sql-native-replay-20261014-r1`
- Attempt: `attempt-d360ace1-0d17-4ce1-8826-4fbc8fa0230c`
- Session: `manual-20261014-base5-longbundles-northstar-r1--ea0b58f17f9572ab`
- Project git identity supplied at launch: `44167dc1050f5bff79abc32fab19c0bb50643093`
- Acceptance state: `UNACCEPTED`; independent review and reducer remain required

## Definition of Done

- Preserve the existing C4 idempotency DTO and function signatures.
- Make concurrent same-key SQL requests resolve to one canonical execution without surfacing a unique-key race error.
- Prove canonical replay, digest conflict, target conflict, mixed-digest concurrency, and reopen persistence against LocalStorage.
- Record exact source hashes, test evidence, and any infrastructure blocker without upgrading status.

## B1 — Inspect and map

- Target root: `D:/Users/NIKITA/Documents/DEV/universal-solver-site/agentfield`.
- Existing source contract: `AsyncExecutionIdempotencyResult`, `AsyncExecutionIdempotencyStore`, `LookupAsyncExecutionIdempotency`, and `CreateAsyncExecutionIdempotent` in `control-plane/internal/storage/execution_idempotency.go`.
- Existing C4 evidence: `docs/acceptance/c4-releasegate-reducer-frontier-20261013.md` records missing native source-bound evidence and remains unchanged.
- Existing package tests already establish replay/conflict and race expectations; the authorized test adds the longbundle native/restart slice.

## B2 — Reproduction and root cause

- The pre-change implementation performed a transactional existence read followed by a plain binding insert.
- Concurrent contenders could both observe no row; the database uniqueness fence then rejected the loser at insert instead of returning the winner's canonical execution.
- An attempted pre-change focused runner was not permitted because the existing counterexample test was outside this task's write/test scope; no product conclusion was drawn from that policy denial.

## B3 — Implementation

`CreateAsyncExecutionIdempotent` now inserts the request binding with `ON CONFLICT (scope_key, request_key) DO NOTHING`. A zero affected-row result commits the no-op and performs the existing durable lookup, returning either the canonical replay or the target/digest conflict. A successful insert continues to create the execution row in the same transaction.

Changed source hash:

- `control-plane/internal/storage/execution_idempotency.go` — `7ff044cdce4e1ec772e3e79a6d4e074e9fece17c2a6857bdf0f4552a1db9cfe2`

## B4 — Focused validation

Authorized command requested by the task:

`go test ./internal/storage -run TestNorthstarLongBundle -count=1`

Authorized runner invocation:

`go_wave.run_project_tests({"file":"control-plane/internal/storage/northstar_longbundle_sql_replay_20261014_test.go"})`

Result: `INFRA BLOCKED`, exit `1`, before Go compilation.

Exact runner evidence:

- Engine: `docker:go-test`
- Go image: `golang:1.24.0-alpine`
- Runner source identity: `b74c16b2e0d8e3f7794794cb01d1506ef401c0cefbc6cd800b8e210b9adf479f`
- Error: `go: cannot find main module, but found .git/config in /workspace` followed by `to create a module there, run: go mod init`
- No identical retry was made; this is a runner working-directory/module-root failure, not a passing or failing product test.

Authorized focused test file final hash:

- `control-plane/internal/storage/northstar_longbundle_sql_replay_20261014_test.go` — `2617b87b36db43e6181eee450cbbd0815e49e824eaab999a7111bf698675dcc2`

The final test source contains assertions for one positive replay, digest and target conflicts, concurrent same-key mixed digests with exactly one creator and one conflict, and durable replay/conflict after LocalStorage close/reinitialize.

## B5 — Self-review and handoff

- Write scope is limited to the authorized source file, focused test file, and this acceptance artifact.
- Public C4 DTO and method signatures remain unchanged.
- No mocks replace LocalStorage; the test initializes the real storage path and reopens it for persistence checks.
- No acceptance, green status, task completion, lease, lock, deployment, commit, or release claim is made.
- Independent verifier/reviewer must rerun from the `control-plane` Go module root, inspect the final source hashes, and perform the affected regression before any reducer decision.

## Remaining

- Runtime proof is unavailable in this attempt because the scoped runner launches from `/workspace` instead of the repository's `control-plane` module root.
- Independent QA and reducer evidence remain required before `ACCEPTED_GREEN`.
