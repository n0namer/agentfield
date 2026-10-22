package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Agent-Field/agentfield/control-plane/internal/services"
	"github.com/Agent-Field/agentfield/control-plane/internal/storage"
	"github.com/Agent-Field/agentfield/control-plane/pkg/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestBase20IndependentExecutionQA_CallbackRecoveryAndAdversarialPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	tempDir := t.TempDir()
	cfg := storage.StorageConfig{
		Mode: "local",
		Local: storage.LocalStorageConfig{
			DatabasePath: filepath.Join(tempDir, "agentfield.db"),
			KVStorePath:  filepath.Join(tempDir, "agentfield.bolt"),
		},
	}

	first := storage.NewLocalStorage(storage.LocalStorageConfig{})
	err := first.Initialize(ctx, cfg)
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "fts5") {
		t.Skip("sqlite3 compiled without FTS5; skipping independent execution QA")
	}
	require.NoError(t, err)

	primaryID := "base20-independent-primary"
	primaryRunID := "base20-independent-run"
	require.NoError(t, first.CreateExecutionRecord(ctx, &types.Execution{
		ExecutionID: primaryID,
		RunID:       primaryRunID,
		SessionID:   "base20-independent-session",
		AgentNodeID: "node-a",
		InstanceID:  "instance-a",
		Status:      types.ExecutionStatusQueued,
		StartedAt:   time.Now().UTC(),
	}))

	router := gin.New()
	router.POST("/executions/:execution_id/status", UpdateExecutionStatusHandler(
		first,
		services.NewFilePayloadStore(t.TempDir()),
		nil,
		time.Second,
	))
	post := func(executionID, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/executions/"+executionID+"/status", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		return resp
	}

	require.Equal(t, http.StatusOK, post(primaryID, `{"status":"running"}`).Code)
	require.Equal(t, http.StatusOK, post(primaryID, `{"status":"succeeded","result":{"answer":"authoritative"}}`).Code)
	require.Equal(t, http.StatusOK, post(primaryID, `{"status":"succeeded","result":{"answer":"replayed"}}`).Code)

	primary, err := first.GetExecutionRecord(ctx, primaryID)
	require.NoError(t, err)
	require.Equal(t, types.ExecutionStatusSucceeded, primary.Status)
	require.JSONEq(t, `{"answer":"authoritative"}`, string(primary.ResultPayload))

	lateRunning := post(primaryID, `{"status":"running","result":{"answer":"late"}}`)
	require.NotEqual(t, http.StatusOK, lateRunning.Code)
	lateTerminal := post(primaryID, `{"status":"failed","error":"late-terminal"}`)
	require.NotEqual(t, http.StatusOK, lateTerminal.Code)

	unrelatedID := "base20-independent-unrelated"
	mismatched := post(unrelatedID, `{"status":"succeeded","result":{"answer":"wrong-target"}}`)
	require.NotEqual(t, http.StatusOK, mismatched.Code)
	primary, err = first.GetExecutionRecord(ctx, primaryID)
	require.NoError(t, err)
	require.Equal(t, types.ExecutionStatusSucceeded, primary.Status)
	require.JSONEq(t, `{"answer":"authoritative"}`, string(primary.ResultPayload))

	delayedID := "base20-independent-delayed"
	delayedCallback := post(delayedID, `{"status":"succeeded","result":{"answer":"too-early"}}`)
	require.NotEqual(t, http.StatusOK, delayedCallback.Code)
	require.NoError(t, first.CreateExecutionRecord(ctx, &types.Execution{
		ExecutionID: delayedID,
		RunID:       "base20-independent-delayed-run",
		SessionID:   "base20-independent-delayed-session",
		AgentNodeID: "node-a",
		InstanceID:  "instance-a",
		Status:      types.ExecutionStatusQueued,
		StartedAt:   time.Now().UTC(),
	}))
	delayed, err := first.GetExecutionRecord(ctx, delayedID)
	require.NoError(t, err)
	require.Equal(t, types.ExecutionStatusQueued, delayed.Status)
	require.Empty(t, delayed.ResultPayload)
	require.Equal(t, http.StatusOK, post(delayedID, `{"status":"running"}`).Code)
	require.Equal(t, http.StatusOK, post(delayedID, `{"status":"succeeded","result":{"answer":"persisted-after-write"}}`).Code)

	missingBindingID := "base20-independent-missing-bindings"
	require.NoError(t, first.CreateExecutionRecord(ctx, &types.Execution{
		ExecutionID: missingBindingID,
		AgentNodeID: "node-a",
		InstanceID:  "instance-a",
		Status:      types.ExecutionStatusQueued,
		StartedAt:   time.Now().UTC(),
	}))
	require.Equal(t, http.StatusOK, post(missingBindingID, `{"status":"succeeded","result":{"answer":"unbound"}}`).Code)
	missingBinding, err := first.GetExecutionRecord(ctx, missingBindingID)
	require.NoError(t, err)
	require.Empty(t, missingBinding.RunID, "current callback contract permits records without a run binding")
	require.Empty(t, missingBinding.SessionID, "current callback contract permits records without a session binding")
	require.Equal(t, types.ExecutionStatusSucceeded, missingBinding.Status)

	require.NoError(t, first.Close(ctx))

	second := storage.NewLocalStorage(storage.LocalStorageConfig{})
	require.NoError(t, second.Initialize(ctx, cfg))
	t.Cleanup(func() { _ = second.Close(ctx) })

	recoveredPrimary, err := second.GetExecutionRecord(ctx, primaryID)
	require.NoError(t, err)
	require.Equal(t, types.ExecutionStatusSucceeded, recoveredPrimary.Status)
	require.JSONEq(t, `{"answer":"authoritative"}`, string(recoveredPrimary.ResultPayload))

	rows, err := second.QueryExecutionRecords(ctx, types.ExecutionFilter{RunID: &primaryRunID})
	require.NoError(t, err)
	require.Len(t, rows, 1, "duplicate terminal callbacks must not create another durable terminal record")
	require.Equal(t, primaryID, rows[0].ExecutionID)
}
