package storage

import (
	"testing"
	"time"

	"github.com/Agent-Field/agentfield/control-plane/pkg/types"
	"github.com/stretchr/testify/require"
)

// A working execution must survive the reaper when an agent heartbeat updates
// its own updated_at, yet genuinely abandoned work must still be reaped.
func TestExecutionActivityHeartbeatKeepsLiveReasoner(t *testing.T) {
	ls, ctx := setupTestLocalStorage(t)
	id := "exec-progress-heartbeat"
	start := time.Now().UTC().Add(-2 * time.Hour)
	require.NoError(t, ls.CreateExecutionRecord(ctx, &types.Execution{
		ExecutionID: id, RunID: "run-progress", AgentNodeID: "swe-planner",
		InstanceID: "current-generation", Status: "running", StartedAt: start,
	}))
	backdateExecutionUpdatedAt(t, ls, "executions", id, time.Now().UTC().Add(-time.Hour))
	updated, err := ls.UpdateExecutionRecord(ctx, id, func(rec *types.Execution) (*types.Execution, error) {
		require.Equal(t, "current-generation", rec.InstanceID)
		require.Equal(t, "running", rec.Status)
		return rec, nil
	})
	require.NoError(t, err)
	require.WithinDuration(t, time.Now().UTC(), updated.UpdatedAt, 5*time.Second)
	count, err := ls.MarkStaleExecutions(ctx, 30*time.Minute, 100)
	require.NoError(t, err)
	require.Equal(t, 0, count)
	record, err := ls.GetExecutionRecord(ctx, id)
	require.NoError(t, err)
	require.Equal(t, "running", record.Status)

	// A worker that died with no more heartbeats is still collected.
	backdateExecutionUpdatedAt(t, ls, "executions", id, time.Now().UTC().Add(-time.Hour))
	count, err = ls.MarkStaleExecutions(ctx, 30*time.Minute, 100)
	require.NoError(t, err)
	require.Equal(t, 1, count)
	record, err = ls.GetExecutionRecord(ctx, id)
	require.NoError(t, err)
	require.Equal(t, "timeout", record.Status)
}
