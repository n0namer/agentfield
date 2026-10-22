# MVP R11 SQL DB Wrapper Repair — 2026-10-14

## Authority

- Task: `agentfield:v2-r11-qa-sql-db-wrapper-repair-20261014-r1`
- Attempt: `attempt-2d1a2a16-9dee-4b1e-9cb5-a4cd51cc7727`
- Session: `manual-20261014-basefive-r11-repair-agentfield--1acb94a7a31ce557`
- Immutable launch Git identity: `44167dc1050f5bff79abc32fab19c0bb50643093`
- Active BMad source of truth: `docs/acceptance/c4-releasegate-reducer-frontier-20261013.md`
- Scope: the independent QA fixture and this receipt only; no production source changed.

## D1 — Source and fixture evidence

- The read-only production implementation `control-plane/internal/storage/execution_idempotency.go` uses `ls.requireSQLDB()` directly for `QueryRowContext` and transaction creation.
- The independent QA fixture uses `assertR9Counts` with a concrete `*sql.DB`, so it must obtain the exported database handle from the SQL wrapper.
- Before repair, fixture SHA-256: `729173899b4288b71955bfc73e5489f4d2cb6ece495e1b366548acfdd1619923`.
- After repair, fixture SHA-256: `0b356c533f7d5e7dd625b1862805dbddb5eaccfd9170fe2ea92c58286857f5b3`.

## D2–D3 — Reproduction and smallest correction

The authorized focused runner invocation was:

```text
go_wave.run_project_tests({"file":"control-plane/internal/storage/northstar_mvp_r9_sql_adversarial_qa_test.go"})
```

It returned `FAIL`, exit code `1`, before Go compilation. The native Docker runner reported:

```text
go: cannot find main module, but found .git/config in /workspace
to create a module there, run: go mod init
```

This is an infrastructure failure in the runner working directory, not evidence that the corrected source compiles or that the test passes. Repeating the unchanged denied/misconfigured runner would provide no new evidence.

The fixture previously passed the unexported wrapper field, `requireSQLDB().db`, to `assertR9Counts`. The wrapper exposes `DB`; the repair changes only the four count-check call sites to `requireSQLDB().DB`:

1. nil payload rejection;
2. canceled request rejection;
3. conflicting digest replay;
4. replay after storage restart.

The fixture continues to exercise native SQL creation and `SELECT COUNT(*)` checks, rather than stubbing or bypassing the database.

## D4 — Existing adversarial and restart coverage

The repaired fixture retains these assertions:

- canonical SQL winner creates exactly one binding and execution;
- nil execution payload fails closed without durable mutation;
- canceled context fails without durable mutation;
- a foreign scope cannot read the canonical binding;
- a different digest produces a conflict without replacing the canonical execution;
- reopening the same database replays the canonical execution without creating a duplicate.

No success or compile claim is made because the only authorized test execution stopped before compilation.

## Self-review and handoff

- Changed paths are restricted to the exact write scope.
- No SQL query, migration, production implementation, or security/fencing invariant was changed.
- The test-specific compile defect is repaired at the field-access site.
- Independent QA/reducer and coordinator terminal readback remain required before any acceptance claim.

## Required recovery

Run the requested command from the Go module root (`control-plane`) with a runner that preserves that working directory:

```text
go test ./internal/storage -run TestMVPR9 -count=1
```

The next owner must record the actual exit code and output. If compilation or behavior then fails, diagnose that new product-level failure separately; do not treat this infrastructure receipt as a passing test.
