# AgentField / AF Deep Research — A55 to VP49 emergency checkpoint

Date: 2026-10-22/23 (Europe/Riga). Status: **PARTIAL — code rescued, production AgentField candidate not deployed**.

## Source of truth and verified refs

GitHub: `https://github.com/n0namer/agentfield.git`. Existing VP49 `dev` checkout is `C:\Users\Рафик\Documents\DEV\universal-solver-site\agentfield`, SHA `44167dc1050f5bff79abc32fab19c0bb50643093`, left unchanged.

Rescue branches, confirmed by `git ls-remote` and a second direct VP49 `git fetch origin`:
- `rescue/a55-agentfield-main-dirty-20261022`: `a7a48c30872f5b2003182081048b4cfa877ad34d` (114 paths; control-plane, SDK, idempotency, migrations, QA).
- `rescue/a55-agentfield-upstream-dirty-20261022`: `bed7baaa9fd7feb1484f7dec8ab96dc0885b3ef6` (430 paths, whitespace-check RED).
- `rescue/a55-agentfield-b01-dirty-20261022`: `0ac79adf7763d4c6172140ae97b3d9818e9c37d0` (291 paths, whitespace-check RED).
- `rescue/a55-agentfield-afrt-integration-dirty-20261022`: `eab6cee8f2d9b3a069b979b97da300ca7699466e` (25 paths).
- `rescue/a55-agentfield-afrt-core-go-dirty-20261022`: `319dfa3e9defc329b5ebebb04e554e1b4feebb9d` (8 paths).
- `rescue/a55-agentfield-afrt-core-py-dirty-20261022`: `5e92c0f4e52f88d9b4e38138cd3446d7cd02b9ca` (16 paths).
- `rescue/a55-agentfield-runtime-clean-dirty-20261022`: `cb64faba3632552ae59e39a09b0819dfa05fefe1` (35 paths).

A55 `dev` was `e473535a3d112434b01d72cdbe6489e5e79be76b` (two commits ahead of `origin/dev`). Its dirty worktree was preserved as a separate branch. No force push, reset, clean, original worktree deletion, or deployment occurred.

## VP49 files, environment, and execution

- Canonical rescue *development candidate*: `C:\Users\Rafik\Documents\agentfield-a55-main-20261022` (branch `rescue/a55-agentfield-main-dirty-20261022`).
- Other six worktrees: `C:\Users\Rafik\Documents\agentfield-a55-<variant>-20261022`, each tracking its own remote rescue branch; do **not** combine without review.
- Local read-only A55 file snapshots and SHA-256 manifests: `C:\Users\Rafik\Documents\agentfield-a55-rescue-20261022\{primary,upstream,b01,afrt-integration,afrt-core-go,afrt-core-py,runtime-clean}\manifest.json`.
- `agentfield.mcp.json`, upstream `.env.dev`, and `.env.example` copied separately over SMB into the private recovery directory. Hash equality checked. **Secrets excluded from GitHub**; existing VP49 deployment settings were not overwritten.
- Generated/cache/state artifacts filtered from Git commits; exclusions are inspectable in manifests. No assertion that excluded generated data is restored or that all ignored files are unique.
- Running VP49 Control Plane `http://127.0.0.1:18080`: health `healthy`, `meta_deep_research` `active/ready` with 23 reasoners. Windows Scheduled Task `VP49-AgentField-control-plane` remains running. The installed `agentfield-server.exe` has **not** been rebuilt from rescue code.
- Deep Research: GitHub `n0namer/af-deep-research`, branch `vp49/afdr-mvp-20261022`, SHA `ad9170af209ea550f4cecb401c2b444fa1e04035`, local `C:\Users\Rafik\Documents\afdr-vp49-mvp-20261022`. Native AgentField execution `exec_20261022_225521_k921hobp` read back `succeeded`: 2 report sections, 4 source notes, verified coverage 1.0. This is technical E2E, not independent quality QA or CrewSync SERVER_DONE.

## QA blockers and next minimal execution

`go version` on VP49: 1.27.0, Windows. `go test ./internal/handlers ./internal/storage ./internal/packages ./internal/cli` from candidate `control-plane` **did not pass**. Observed: `CGO_ENABLED=0` with missing GCC makes go-sqlite3 tests fail; two handler tests pass string where `*string` is required; `internal/cli/uninstall_test.go` references Unix-only `syscall.Kill`. A prolonged invalid test session was terminated. Do not promote the candidate as production without compatible toolchain and a focused test repair.

Next: isolate targeted Windows/CGO repairs in a new VP49 integration worktree; run focused unit/regression tests and independent QA; verify state/backups/migrations before any Control Plane binary swap; after QA deploy existing task runner safely and demonstrate Deep Research E2E + crash/replay. No `SERVER_DONE` is claimed.
