# Base20 Go Harness Runner Lifecycle Acceptance

- Task: `agentfield:v2-base20-go-harness-runner-lifecycle-20261013-r1`
- Project: `agentfield`
- Attempt: `attempt-d455c6f7-4e97-437a-a9ef-7d6af1196d5d`
- Session: `manual-20261013-base20-agentfield-core-r1-v2-ba-74bd4c5ce9ad5cf3`
- Status: `CANDIDATE` / `UNACCEPTED`

## Scope and evidence

The runner already provided provider precedence, schema parsing/validation, retry and session recovery, output isolation, cleanup, cancellation-aware backoff, and metric aggregation. The bounded lifecycle test covers:

- successful provider selection and deterministic rejection of an unknown provider;
- structured output recovery after a crashed agent session;
- no reuse of a crashed session ID;
- model, session, turn, cost, and token accounting across attempts;
- cancellation before provider dispatch and cancellation after a late provider result;
- bounded handling of a nil provider result;
- recovery artifact creation inside the supplied temporary project directory.

The runner repair is limited to `executeWithRetry`: context cancellation wins before dispatch and after provider return, and `(nil, nil)` provider results return a deterministic error instead of dereferencing nil.

## Validation

- `sdk/go/harness/runner_goal_lifecycle_wave_test.go`: authored but not executable through the available scoped MCP runner; `run_project_tests` returned `GO_WAVE_NODE_ERROR:test_file_not_allowed`.
- `sdk/go/harness/runner_invariant_test.go`: attempted once; the same runner returned `GO_WAVE_NODE_ERROR:test_file_not_allowed`.
- Required `from sdk/go: go test ./harness -count=1`: not run because no authorized Go command runner is available in this worker scope. This is an infrastructure block, not a passing result.

## Changed files

- `sdk/go/harness/runner_goal_lifecycle_wave_test.go`
- `sdk/go/harness/runner.go`
- `docs/acceptance/base20-go-harness-runner-20261013.md`

Independent QA must run the focused test and affected package regression with the offline Go toolchain, inspect the final diff, and decide acceptance. No live paid provider or external relay was used.
