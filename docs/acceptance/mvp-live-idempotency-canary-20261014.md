# MVP Live Async Idempotency Canary — 2026-10-14

## Scope and source of truth

- Task: `agentfield:v2-mvp-live-async-idempotency-canary-20261014-r8`
- Attempt: `attempt-5957e362-efe4-4988-9b69-16374183aa54`
- Session: `manual-20261014-basefive-mvp-10owners-r8-agentf-f7774c5232400340`
- Launch Git identity: `44167dc1050f5bff79abc32fab19c0bb50643093`
- Authorized write scope: `control-plane/internal/handlers/execute.go`, `control-plane/internal/handlers/northstar_mvp_live_idempotency_20261014_test.go`, this document
- Existing C4 artifact: `docs/acceptance/c4-releasegate-reducer-frontier-20261013.md`

## Implementation finding

`control-plane/internal/handlers/execute.go` already contains the scoped durable async idempotency mechanism:

- authenticated caller and target-derived scope;
- bounded printable `Idempotency-Key` validation;
- request-body SHA-256 binding;
- durable lookup before admission;
- atomic create/replay/conflict adapter;
- replay response using the canonical execution and run IDs.

No production change was made because the mechanism is already present at the current source SHA. The authorized canary test exercises that existing adapter contract without introducing a competing implementation.

## Focused canary coverage

`control-plane/internal/handlers/northstar_mvp_live_idempotency_20261014_test.go` covers:

1. Positive durable create followed by replay returns the canonical execution and records one durable write.
2. Wrong target identity and wrong body digest fail closed as idempotency conflicts.
3. A cancelled context performs no durable write.
4. A fresh adapter over the same durable store replays the canonical execution after restart.

## Validation receipts

| Check | Status | Evidence |
|---|---|---|
| `go test ./internal/handlers -run TestMVPLiveIdempotency -count=1` equivalent scoped runner | `BLOCKED_INFRA` | `go_wave.run_project_tests({"file":"control-plane/internal/handlers/northstar_mvp_live_idempotency_20261014_test.go"})`; exit `1`; source SHA `de22924928047588c84f64aafe09a374673e013dba09bfebb791aa48415b60d8`; stderr: `go: cannot find main module, but found .git/config in /workspace` and `to create a module there, run: go mod init` |
| Affected storage baseline `control-plane/internal/storage/northstar_basefive_sql_live_replay_20261014_test.go` | `BLOCKED_INFRA` | `go_wave.run_project_tests`; exit `1`; source SHA `c759e7870268acc7724408f51f83f3c6797ccdf94b289abc73457a107e630d27`; same pre-compilation `/workspace` module-root error |
| Production source self-review | `PASS_REVIEW_ONLY` | `execute.go` SHA `be09150579e29e16ce9d96775ef1b161898d829570b9bdd5885c9db5327a25c1`; existing adapter path reviewed; no unrelated files changed |

The focused and affected Go checks were not executed through compilation. They are not treated as PASS. The runner must execute from the repository's Go module root before these checks can establish product evidence.

## Remaining gate

Independent verification must rerun the focused handler test and the affected storage baseline from the correct module root, then perform the coordinator-owned typed result readback. Product acceptance remains `UNACCEPTED`; this artifact makes no `ACCEPTED_GREEN` claim.
