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

const (
	northstarExecutionID = "northstar-c4-callback"
	northstarOwnerID     = "node-current-owner"
	northstarSessionID   = "session-current"
)

func TestNorthstarCallbackFence_NativePositivePathPersistsSingleTerminalReceipt(t *testing.T) {
	store := newNorthstarCallbackStore(t)
	router := newNorthstarCallbackRouter(t, store)

	response := sendNorthstarCallback(t, router, `{
		"agent_node_id":"node-current-owner",
		"instance_id":"session-current",
		"status":"succeeded",
		"result":{"receipt":"durable-authoritative"},
		"duration_ms":17
	}`)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())

	var callbackResult ExecutionStatusResponse
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &callbackResult))
	require.Equal(t, northstarExecutionID, callbackResult.ExecutionID)
	require.Equal(t, northstarOwnerID, callbackResult.AgentNodeID)
	require.Equal(t, northstarSessionID, callbackResult.InstanceID)
	require.Equal(t, types.ExecutionStatusSucceeded, callbackResult.Status)

	persisted, err := store.GetExecutionRecord(context.Background(), northstarExecutionID)
	require.NoError(t, err)
	require.Equal(t, types.ExecutionStatusSucceeded, persisted.Status)
	require.JSONEq(t, `{"receipt":"durable-authoritative"}`, string(persisted.ResultPayload))
	require.NotNil(t, persisted.CompletedAt)
	require.NotNil(t, persisted.DurationMS)
	require.Equal(t, int64(17), *persisted.DurationMS)
}

func TestNorthstarCallbackFence_RejectsStaleOwnerWithoutMutatingReceipt(t *testing.T) {
	store := newNorthstarCallbackStore(t)
	router := newNorthstarCallbackRouter(t, store)

	response := sendNorthstarCallback(t, router, `{
		"agent_node_id":"node-stale-owner",
		"instance_id":"session-current",
		"status":"succeeded",
		"result":{"receipt":"must-not-persist"}
	}`)
	require.Equal(t, http.StatusConflict, response.Code, response.Body.String())

	assertNorthstarExecutionStillRunning(t, store)
}

func TestNorthstarCallbackFence_RejectsMismatchedSessionWithoutMutatingReceipt(t *testing.T) {
	store := newNorthstarCallbackStore(t)
	router := newNorthstarCallbackRouter(t, store)

	response := sendNorthstarCallback(t, router, `{
		"agent_node_id":"node-current-owner",
		"instance_id":"session-from-before-restart",
		"status":"succeeded",
		"result":{"receipt":"must-not-persist"}
	}`)
	require.Equal(t, http.StatusConflict, response.Code, response.Body.String())

	assertNorthstarExecutionStillRunning(t, store)
}

func TestNorthstarCallbackFence_RestartRetryPreservesFirstTerminalReceipt(t *testing.T) {
	store := newNorthstarCallbackStore(t)
	firstRouter := newNorthstarCallbackRouter(t, store)

	firstResponse := sendNorthstarCallback(t, firstRouter, `{
		"agent_node_id":"node-current-owner",
		"instance_id":"session-current",
		"status":"succeeded",
		"result":{"receipt":"first-and-only"}
	}`)
	require.Equal(t, http.StatusOK, firstResponse.Code, firstResponse.Body.String())

	beforeRestart, err := store.GetExecutionRecord(context.Background(), northstarExecutionID)
	require.NoError(t, err)
	require.NotNil(t, beforeRestart.CompletedAt)
	firstCompletedAt := *beforeRestart.CompletedAt

	// Rebuilding the handler while retaining the durable store models a control-plane restart.
	restartedRouter := newNorthstarCallbackRouter(t, store)
	duplicateResponse := sendNorthstarCallback(t, restartedRouter, `{
		"agent_node_id":"node-current-owner",
		"instance_id":"session-current",
		"status":"succeeded",
		"result":{"receipt":"duplicate-must-not-rewrite"}
	}`)
	require.Equal(t, http.StatusOK, duplicateResponse.Code, duplicateResponse.Body.String())

	conflictingResponse := sendNorthstarCallback(t, restartedRouter, `{
		"agent_node_id":"node-current-owner",
		"instance_id":"session-current",
		"status":"failed",
		"error":"late-conflict"
	}`)
	require.Equal(t, http.StatusConflict, conflictingResponse.Code, conflictingResponse.Body.String())

	afterRestart, err := store.GetExecutionRecord(context.Background(), northstarExecutionID)
	require.NoError(t, err)
	require.Equal(t, types.ExecutionStatusSucceeded, afterRestart.Status)
	require.JSONEq(t, `{"receipt":"first-and-only"}`, string(afterRestart.ResultPayload))
	require.Nil(t, afterRestart.ErrorMessage)
	require.NotNil(t, afterRestart.CompletedAt)
	require.True(t, firstCompletedAt.Equal(*afterRestart.CompletedAt))
}

func newNorthstarCallbackStore(t *testing.T) *testExecutionStorage {
	t.Helper()
	store := newTestExecutionStorage(nil)
	now := time.Now().UTC()
	require.NoError(t, store.CreateExecutionRecord(context.Background(), &types.Execution{
		ExecutionID: northstarExecutionID,
		RunID:       "northstar-run",
		AgentNodeID: northstarOwnerID,
		InstanceID:  northstarSessionID,
		Status:      types.ExecutionStatusRunning,
		StartedAt:   now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}))
	return store
}

func newNorthstarCallbackRouter(t *testing.T, store ExecutionStore) http.Handler {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.PUT(
		"/api/v1/executions/:execution_id/status",
		UpdateExecutionStatusHandler(store, services.NewFilePayloadStore(t.TempDir()), nil, time.Second),
	)
	return router
}

func sendNorthstarCallback(t *testing.T, router http.Handler, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(
		http.MethodPut,
		"/api/v1/executions/"+northstarExecutionID+"/status",
		strings.NewReader(body),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func assertNorthstarExecutionStillRunning(t *testing.T, store ExecutionStore) {
	t.Helper()
	persisted, err := store.GetExecutionRecord(context.Background(), northstarExecutionID)
	require.NoError(t, err)
	require.Equal(t, types.ExecutionStatusRunning, persisted.Status)
	require.Empty(t, persisted.ResultPayload)
	require.Nil(t, persisted.CompletedAt)
}
