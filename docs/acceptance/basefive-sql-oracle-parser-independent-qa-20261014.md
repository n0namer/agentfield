# BaseFive SQL Oracle Parser Independent QA — 2026-10-14

## Authority

- Task: `agentfield:v2-basefive-storage-sql-oracle-adversarial-qa-20261014-r5`
- Attempt: `attempt-6ebcbb16-63c2-4067-b977-803525f32029`
- Session: `manual-20261014-basefive-native-gate-repair-r5--bcf92a91722f8163`
- Project Git identity supplied at launch: `44167dc1050f5bff79abc32fab19c0bb50643093`
- Manifest SHA-256: `10a6a9d27ea50377541af4f3a35ce3aa04914f68ce9d317a61793deac028e130`
- Acceptance level: `UNACCEPTED`

This artifact records an independent QA candidate only. It does not change production source, native task state, CrewSync claims, leases, release state, or acceptance state.

## Scope and implementation

Authorized changed files are limited to:

- `control-plane/internal/storage/northstar_basefive_parser_qa_20261014_test.go`
- `docs/acceptance/basefive-sql-oracle-parser-independent-qa-20261014.md`

The new `TestBaseFiveSQLParserQA` uses the existing native `LocalStorage` API and covers:

1. Positive canonical SQL execution creation and same-key replay.
2. Negative conflicting digest rejection with `Conflict=true` and no execution.
3. Negative conflicting target rejection with `Conflict=true` and no execution.
4. Restart persistence by closing and reinitializing storage, then replaying the canonical execution ID.

No production source or unrelated test was modified.

## Root-cause evidence

Independent review first reproduced and repaired the QA fixture's invalid three-variable map range. This attempt then reproduced a second fixture defect: native initialization did not apply migration `036_execution_request_idempotency.sql`, leaving `execution_request_idempotency` absent in the fresh SQLite database. The fixture now applies the canonical migration-036 table and index through the initialized storage DB before exercising the API. The repaired test source SHA-256 is `614f6da2ac01662056b819c9969fe6e3e0bd9e855339075e29822e84b8117051`.

A separate pre-existing parser defect remains in `control-plane/internal/storage/northstar_independent_sql_oracle_20261014_test.go`, outside this task's write scope. Its source SHA-256 at inspection was `202d910ddc7a18d03ab68894384c9539d6ce475275324446c5c24a4e126684f7`; line 120, column 20 assigns three variables to a `map[string]string`. Repairing that file was not authorized and was not attempted.

## Validation receipt

Requested validation:

```text
go test ./internal/storage -run TestBaseFiveSQLParserQA -count=1
```

Executed through `go_wave.run_project_tests` for:

```text
control-plane/internal/storage/northstar_basefive_parser_qa_20261014_test.go
```

Result:

- Status: `FAIL`
- Exit code: `1`
- Runner: `docker:go-test`
- Image: `golang:1.24.0`
- Runner source identity SHA-256: `614f6da2ac01662056b819c9969fe6e3e0bd9e855339075e29822e84b8117051`
- Duration limit: `120000 ms`
- Assertions executed: none
- Exact stderr: `go: cannot find main module, but found .git/config in /workspace` followed by `to create a module there, run: go mod init`
- Standalone `gofmt -e -d` preflight: unavailable through the authorized scoped MCP interface; no shell or PowerShell fallback was used.

This is classified as `BLOCKED_INFRA`, not PASS. The available runner did not provide the requested Go 1.26.5 environment and failed before package compilation. No affected regression test was run.

## Gate status

- Positive assertion: implemented, not executed.
- Two negative assertions: implemented, not executed.
- Restart assertion: implemented, not executed.
- Exact-source parser build: blocked by the pre-existing out-of-scope syntax error.
- Focused Go validation: `BLOCKED_INFRA` before compilation.
- Affected regression: not run because the focused runner could not establish the module root.
- Product acceptance: `UNACCEPTED`.

## Recovery required

The coordinator or an authorized follow-up must first grant the exact parser test path for the one-line syntax repair (or provide a runner rooted at `control-plane` with Go 1.26.5). Then rerun the focused command and the affected storage regression, preserving fresh source identity and exit receipts. Independent QA and the coordinator retain acceptance authority.
