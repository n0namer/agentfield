# Base20 Execution Record Recovery

## Scope

Prove that persisted execution records survive a local-storage restart and repeated delivery without losing logical identity or reopening a terminal effect.

## Source of truth

- `control-plane/internal/storage/execution_records.go` owns execution-record persistence and atomic read-modify-write updates.
- The existing execution state validator owns legal status transitions.
- The existing database uniqueness constraint on `execution_id` prevents a second record for the same logical execution.

## Definition of done

- Create a pending record with exact `execution_id`, `run_id`, and `instance_id`.
- Advance it to running, then succeeded with a result payload.
- Close and reopen local SQLite storage using the same database path.
- Read back the terminal record and verify identity, status, and result payload.
- Replay the same succeeded delivery idempotently.
- Reject a stale delivery that attempts to reopen the terminal record.
- Reject a duplicate insert, leaving exactly one succeeded record.

## Implementation

`UpdateExecutionRecord` now invokes the existing `validateExecutionStateTransition` before persisting an updated record and attaches the execution ID to transition errors. No new store, queue, scheduler, migration, or idempotency mechanism was introduced.

## Evidence

Focused command from `control-plane`:

```text
go test ./internal/storage -run TestWaveExecutionRecordRecoveryAcrossRestartAndReplay -count=1
```

Affected package regression:

```text
go test ./internal/storage -count=1
```

Worker validation status: BLOCKED by the scoped runner returning `test_file_not_allowed` for both the focused Go test and the existing affected Go test file. No green test result is claimed.

Independent QA/reducer acceptance remains required. This artifact records the worker candidate evidence only; it is not a product acceptance claim.
