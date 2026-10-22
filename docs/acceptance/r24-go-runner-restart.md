# R24 Go runner restart evidence — 2026-10-15

## Identity and gate

- Task: `agentfield:v2-r24-go-runner-restart-20261015-r2`
- Attempt: `attempt-4e40256b-ba75-40fd-b7d3-e145cdfcee81`
- Session: `manual-20261015-basefive-r24b-20owners-agentfie-4d581fdbc4d2604a`
- Target: `D:/Users/NIKITA/Documents/DEV/universal-solver-site/agentfield`, not BMAD-MNNZ.
- Launch Git identity: `44167dc1050f5bff79abc32fab19c0bb50643093`; current Git state was not independently inspected because no authorized Git interface is exposed.
- State: **BLOCKED / INFRA BLOCKED / UNACCEPTED**. No product PASS or PRODUCT_RED is established.
- Existing BMad SoT: `docs/acceptance/c4-releasegate-reducer-frontier-20261013.md`. Its independent C4 gate remains blocked. This report supplements, not replaces, it.

## Local Definition of Done and scope

The requested outcome is executable Go success, two negative cases and real crash/restart evidence proving no duplicate effect, no foreign terminal owner and no false success, bound to current source and independently reviewed. Only the new test and this report may change. Completion requires actual native focused and affected-regression results; a test file, successful write or transport receipt is insufficient.

The bounded move is an offline boundary QA test reusing `NewRunner` and `Runner.Run`, not a new runner or a simulated durable owner ledger. Production changes, live providers, task/lease changes and external sends are excluded. The initial useful-work estimate was approximately 15–20 minutes for inspection, tests, review and evidence, contingent on executable native tooling; the pre-compilation failure truncated that cycle. Actual elapsed useful work was not instrumented.

## Existing implementation and source evidence

FACT: The target test and report were absent when first read (both returned ENOENT); a missing authorized target was not treated as a blocker.

FACT: `docs/acceptance/base20-go-harness-runner-20261013.md` already describes lifecycle tests and a cancellation/nil-result repair. It explicitly records `test_file_not_allowed`, not runtime PASS. This attempt does not reimplement that repair.

The source read receipt for `sdk/go/harness/runner.go`, lines 1–240, reports SHA-256 `63312956a23baf05c5d14682af14499d2739c7f580e2e7ff771161e85634b7fb` and 757 total lines. Observed anchors:

- `Runner` owns per-instance defaults and logger; `NewRunner` constructs it without resolving a provider.
- `Runner.Run` resolves merged options and calls `buildProvider` before creating a schema-output directory.
- `buildProvider` delegates provider construction to `BuildProvider`.
- After provider construction, `Runner.Run` delegates dispatch to `executeWithRetry`, then translates raw output into `Result` or delegates to `handleSchemaWithRetry`.

LIMITATION: This source file is not enumerated in the trusted read_scope, although the scoped read tool returned it. No further production-source exploration was performed. The remainder of the runner, provider interfaces, terminal-owner implementation and relevant Go module/instructions are not independently inspected here. Future source investigation needs explicit read authorization; tool access is not a scope extension.

FACT: `docs/architecture/basefive-r14-contracts.json` is `UNACCEPTED` and scopes itself to control-plane execution and optional async idempotency, not Go harness terminal ownership. Its observed component chain is `executionController.handleAsync` → `readAsyncIdempotencyRequest` / `asyncIdempotentExecutionStore` → `LocalStorage` / SQL, followed by submission to a process-local `asyncWorkerPool`. It expressly does not prove restart dispatch. Those are contract-reported anchors, not independently observed harness calls.

FACT: The same contract records zero ArchSteer Go components, unverified Graphify call edges and worker/restart/native-E2E blind spots. Neither model is used as proof of Go owner fencing. No terminal-owner identity, fencing epoch or durable completion interface can honestly be derived from these artifacts.

## Scoped test coverage and limitations

Added `sdk/go/harness/r24_go_runner_restart_test.go` with these offline checks:

1. Construction smoke: runner, logger and selected default provider exist. This is NOT a successful provider execution.
2. Negative: an unknown provider must return an error and nil result before creating schema output.
3. Negative: a pre-canceled context must return `context.Canceled` and nil result; its configured binary is a nonexistent temporary path, not an external provider.
4. Process-boundary fixture: start the native test executable, wait for a one-byte pre-dispatch readiness signal, forcibly terminate it, require non-success exit, then start a fresh executable and recheck the unknown-provider boundary. A ten-second context bounds the fixture. No shell or paid CLI is used.

LIMITATION: The process fixture interrupts BEFORE dispatch or effects. It is not recovery after a committed effect and does not test durable replay or terminal-owner fencing. No mock owner store or invented public API is used to make that missing requirement appear green. Full native positive execution, duplicate-effect prevention and foreign-owner rejection remain missing acceptance evidence even if these boundary tests later pass.

Self-review: subprocess selection prevents the helper from blocking ordinary package runs; the parent waits for readiness before killing; the child sleeps on a timer rather than deadlocking on an empty select; child exit and restarted-process output are checked; temporary artifacts remain test-owned. Compilation and these behaviors remain unverified due to the runner failure. Existing source and sibling tests were not changed.

## Exact native execution receipt

Requested validation:

```text
cd sdk/go && go test ./harness -run TestR24GoRunnerRestart -count=1
```

Authorized invocation actually made:

```text
go_wave.run_project_tests({"file":"sdk/go/harness/r24_go_runner_restart_test.go"})
```

Returned receipt:

```json
{
  "status": "FAIL",
  "code": null,
  "test": "sdk/go/harness/r24_go_runner_restart_test.go",
  "engine": "docker:go-test",
  "source_identity_sha256": "0d23e9d1f6bb373ba1513eaad5d821f7826a63f3ba66ed1dbebfaeb357d90d20",
  "exit_code": 1,
  "signal": null,
  "error": null,
  "duration_limit_ms": 120000,
  "output_truncated": false,
  "stdout": "",
  "stderr": "go: cannot find main module, but found .git/config in /workspace\n\tto create a module there, run:\n\tgo mod init\n",
  "package": "./sdk/go/harness",
  "go_image": "golang:1.24.0-alpine"
}
```

Classification: INFRA BLOCKED before compilation, not a test assertion failure. The receipt reports a package, not the exact underlying command/argv; no command equivalence or successful execution is inferred. No `go mod init` was run: modifying the repository root to mask an incorrect runner working directory is not an authorized repair. The same denied route was not retried. No test counts or runtime duration are inferred from the duration limit.

Affected package regression, `cd sdk/go && go test ./harness -count=1`, was not run: the native module-root failure is already established, and there is no authorized runner repair or shell fallback. No lint, formatting, build or runtime PASS is claimed.

## Acceptance matrix and recovery

| Requirement | Evidence / current status |
| --- | --- |
| Source/C4 call paths and owners | Partial source read and contract anchors recorded; durable terminal-owner contract absent. |
| Native successful execution with exact command and exit | NOT MET: construction-only smoke; runner exits 1 before compilation. |
| Two negative cases plus crash/restart | Encoded boundary checks, UNEXECUTED; no post-effect restart or foreign-owner proof. |
| Preserve RED without weakening fixtures | Exact infrastructure failure retained; no fixture weakened and no root module fabricated. |
| Typed original-task result, independent QA | Publication performed separately; independent reviewer/reducer required, acceptance remains UNACCEPTED. |

Next action: coordinator repairs native runner module selection to execute inside `sdk/go`, and authorizes read-only access to actual harness/provider/terminal-owner source, module metadata and applicable instructions. Resume this same task/owner for compilation, focused failures and bounded owned-test repairs. Add genuine provider-success and post-effect/foreign-owner tests only against the verified actual interface; production gaps outside the two-file write scope require the existing source owner. Then run the focused command and affected package regression, and request independent reviewer/reducer QA. Do not accept this boundary test as the missing durable terminal-owner gate.

## Workflow and publication boundaries

BMad Help, Quick Dev and verification-before-completion were loaded. Quick Dev clarification/planning references were read. Script/config resolution, duplicate-spec creation and interactive approval steps were not executed: unavailable tools and the immutable two-file autonomous task take precedence. No full BMad workflow completion is claimed. No reproduced product defect justified a systematic-debugging session.

Publication response is preserved in the final worker handoff, not predicted in this pre-publication artifact. Only an independently authorized owner can accept or perform terminal readback.
