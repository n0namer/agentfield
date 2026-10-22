# Base20 node callback terminal fencing

## Scope

This acceptance artifact covers the authorized test-only slice for async execution callback delivery. `execute.go` remains unchanged because another owner controls the production handler.

## Source-backed behavior

- `UpdateExecutionStatusHandler` accepts a callback by `execution_id` and persists it through `ExecutionStore.UpdateExecutionRecord`.
- The callback path normalizes statuses and accepts the lifecycle `queued -> running -> succeeded`.
- A terminal record accepts only an idempotent callback with the same terminal status. It rejects late non-terminal delivery and a different terminal status before persisting a mutation.
- The persisted readback is obtained through `GetExecutionRecord`, matching the storage seam used by the handler.
- `executionStatusUpdateRequest` currently has no `instance_id`, lease token, fencing epoch, or callback signature field. Therefore foreign-instance and stale-lease rejection cannot be proven from this callback contract in this test-only scope.

## Executable artifact

`control-plane/internal/handlers/node_callback_terminal_wave_test.go` contains:

1. A positive lifecycle test that sends `running`, then `succeeded` with a result, and verifies the terminal status/result through persisted storage readback.
2. An adversarial delivery test that sends duplicate `succeeded`, late `running`, terminal downgrade `failed`, and late `cancelled` callbacks. It verifies the duplicate is a no-op, rejected deliveries do not change the final status, and the authoritative result remains persisted.

## Expected API evidence

For a direct handler invocation, the expected response is:

- `200 OK` for accepted lifecycle callbacks and an idempotent duplicate terminal callback.
- `500 Internal Server Error` for a late non-terminal or cancelled callback after a terminal record, matching the current handler error contract.
- `409 Conflict` for a different terminal status after terminalization.

The production router and docker-control-plane endpoint comparison were not executed in this bounded worker because the authorized environment exposes only scoped project-file and test MCP tools; no runtime process, Docker control-plane, or network endpoint was available. This is an unverified runtime item, not a PASS.

## Test evidence

Attempted exact scoped runs:

- `go_wave.run_project_tests({"file":"control-plane/internal/handlers/node_callback_terminal_wave_test.go"})` → `GO_WAVE_NODE_ERROR:test_file_not_allowed`.
- `go_wave.run_node_tests({"file":"control-plane/internal/handlers/node_callback_terminal_wave_test.go"})` → `GO_WAVE_NODE_ERROR:test_file_not_allowed`.
- `go_wave.run_project_tests({"file":"control-plane/internal/handlers/execute_async_test.go"})` → `GO_WAVE_NODE_ERROR:test_file_not_allowed`.

Because the runner denied both the new test and the nearest existing Go test, no executable test result is claimed. The coordinator/independent QA should run:

```text
go test ./internal/handlers -count=1
```

from `control-plane`, followed by an independently captured runtime callback and persisted execution-ID readback.

## QA handoff

Independent QA must verify the new file compiles in the `handlers` package, run the package validation command, and decide whether the absent instance/lease fields require a follow-up production contract change owned by the handler/service task.
