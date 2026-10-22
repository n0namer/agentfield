package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Agent-Field/agentfield/control-plane/internal/services"
	"github.com/Agent-Field/agentfield/control-plane/pkg/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestNodeCallbackTerminalWave_LifecyclePersistsFinalReadback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := newTestExecutionStorage(testRestartAgent("http://agent.example"))
	executionID := "callback-wave-lifecycle"
	require.NoError(t, store.CreateExecutionRecord(context.Background(), &types.Execution{
		ExecutionID: executionID,
		RunID:       "run-callback-wave-lifecycle",
		AgentNodeID:  "node-a",
		InstanceID:   "instance-a",
		Status:       types.ExecutionStatusQueued,
		StartedAt:    time.Now().UTC(),
	}))

	router := gin.New()
	router.POST("/executions/:execution_id/status", UpdateExecutionStatusHandler(
		store,
		services.NewFilePayloadStore(t.TempDir()),
		nil,
		time.Second,
	))

	callbackStatus(t, router, executionID, `{"status":"running"}`, http.StatusOK)
	callbackStatus(t, router, executionID, `{"status":"succeeded","result":{"answer":"ok"}}`, http.StatusOK)

	persisted, err := store.GetExecutionRecord(context.Background(), executionID)
	require.NoError(t, err)
	require.NotNil(t, persisted)
	require.Equal(t, types.ExecutionStatusSucceeded, persisted.Status)
	require.JSONEq(t, `{"answer":"ok"}`, string(persisted.ResultPayload))
	require.NotNil(t, persisted.CompletedAt)
}

func TestNodeCallbackTerminalWave_DuplicateAndLateDeliveryCannotCorruptTerminalResult(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := newTestExecutionStorage(testRestartAgent("http://agent.example"))
	executionID := "callback-wave-fenced"
	require.NoError(t, store.CreateExecutionRecord(context.Background(), &types.Execution{
		ExecutionID: executionID,
		RunID:       "run-callback-wave-fenced",
		AgentNodeID:  "node-a",
		InstanceID:   "instance-a",
		Status:       types.ExecutionStatusQueued,
		StartedAt:    time.Now().UTC(),
	}))

	router := gin.New()
	router.POST("/executions/:execution_id/status", UpdateExecutionStatusHandler(
		store,
		services.NewFilePayloadStore(t.TempDir()),
		nil,
		time.Second,
	))

	callbackStatus(t, router, executionID, `{"status":"running"}`, http.StatusOK)
	callbackStatus(t, router, executionID, `{"status":"succeeded","result":{"answer":"authoritative"}}`, http.StatusOK)

	callbackStatus(t, router, executionID, `{"status":"succeeded","result":{"answer":"replayed"}}`, http.StatusOK)
	persisted, err := store.GetExecutionRecord(context.Background(), executionID)
	require.NoError(t, err)
	require.JSONEq(t, `{"answer":"authoritative"}`, string(persisted.ResultPayload))

	callbackStatus(t, router, executionID, `{"status":"running","result":{"answer":"late-running"}}`, http.StatusInternalServerError)
	callbackStatus(t, router, executionID, `{"status":"failed","error":"late-downgrade"}`, http.StatusConflict)
	callbackStatus(t, router, executionID, `{"status":"cancelled","status_reason":"cancelled-by-worker"}`, http.StatusInternalServerError)

	persisted, err = store.GetExecutionRecord(context.Background(), executionID)
	require.NoError(t, err)
	require.Equal(t, types.ExecutionStatusSucceeded, persisted.Status)
	require.JSONEq(t, `{"answer":"authoritative"}`, string(persisted.ResultPayload))
	require.Nil(t, persisted.ErrorMessage)
}

func callbackStatus(t *testing.T, router http.Handler, executionID string, body string, expectedStatus int) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/executions/"+executionID+"/status", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	require.Equal(t, expectedStatus, resp.Code, resp.Body.String())

	if expectedStatus == http.StatusOK {
		var status ExecutionStatusResponse
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &status))
		require.Equal(t, executionID, status.ExecutionID)
	}
}
