package storage

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Agent-Field/agentfield/control-plane/pkg/types"
	"github.com/stretchr/testify/require"
)

func TestWaveExecutionRecordRecoveryAcrossRestartAndReplay(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	cfg := StorageConfig{
		Mode: "local",
		Local: LocalStorageConfig{
			DatabasePath: filepath.Join(tempDir, "agentfield.db"),
			KVStorePath:  filepath.Join(tempDir, "agentfield.bolt"),
		},
	}

	first := NewLocalStorage(LocalStorageConfig{})
	err := first.Initialize(ctx, cfg)
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "fts5") {
		t.Skip("sqlite3 compiled without FTS5; skipping execution recovery test")
	}
	require.NoError(t, err)

	executionID := "wave-execution-1"
	runID := "wave-run-1"
	instanceID := "worker-instance-a"
	exec := &types.Execution{
		ExecutionID:  executionID,
		RunID:        runID,
		AgentNodeID: "recovery-agent",
		InstanceID:  instanceID,
		ReasonerID:  "recovery.reasoner",
		NodeID:      "node-a",
		Status:      string(types.ExecutionStatusPending),
		InputPayload: json.RawMessage("null"),
	}
	require.NoError(t, first.CreateExecutionRecord(ctx, exec))

	_, err = first.UpdateExecutionRecord(ctx, executionID, func(current *types.Execution) (*types.Execution, error) {
		current.Status = string(types.ExecutionStatusRunning)
		return current, nil
	})
	require.NoError(t, err)

	completedAt := time.Now().UTC()
	resultPayload := json.RawMessage("true")
	_, err = first.UpdateExecutionRecord(ctx, executionID, func(current *types.Execution) (*types.Execution, error) {
		current.Status = string(types.ExecutionStatusSucceeded)
		current.ResultPayload = resultPayload
		current.CompletedAt = &completedAt
		return current, nil
	})
	require.NoError(t, err)
	require.NoError(t, first.Close(ctx))

	second := NewLocalStorage(LocalStorageConfig{})
	require.NoError(t, second.Initialize(ctx, cfg))
	t.Cleanup(func() { _ = second.Close(ctx) })

	replayed, err := second.GetExecutionRecord(ctx, executionID)
	require.NoError(t, err)
	require.NotNil(t, replayed)
	require.Equal(t, executionID, replayed.ExecutionID)
	require.Equal(t, runID, replayed.RunID)
	require.Equal(t, instanceID, replayed.InstanceID)
	require.Equal(t, string(types.ExecutionStatusSucceeded), replayed.Status)
	require.JSONEq(t, string(resultPayload), string(replayed.ResultPayload))

	_, err = second.UpdateExecutionRecord(ctx, executionID, func(current *types.Execution) (*types.Execution, error) {
		current.Status = string(types.ExecutionStatusSucceeded)
		current.ResultPayload = resultPayload
		return current, nil
	})
	require.NoError(t, err, "replaying the same terminal delivery must be idempotent")

	_, err = second.UpdateExecutionRecord(ctx, executionID, func(current *types.Execution) (*types.Execution, error) {
		current.Status = string(types.ExecutionStatusRunning)
		return current, nil
	})
	require.Error(t, err, "a stale delivery must not reopen a terminal execution")
	var transitionErr *InvalidExecutionStateTransitionError
	require.ErrorAs(t, err, &transitionErr)
	require.Equal(t, executionID, transitionErr.ExecutionID)

	duplicate := *replayed
	require.Error(t, second.CreateExecutionRecord(ctx, &duplicate), "repeated delivery must not create a second success record")
	rows, err := second.QueryExecutionRecords(ctx, types.ExecutionFilter{RunID: &runID})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, string(types.ExecutionStatusSucceeded), rows[0].Status)
}
