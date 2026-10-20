package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Agent-Field/agentfield/control-plane/pkg/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestExecutionHeartbeatOnlyOwnerActiveInstance(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name     string
		state    string
		owner    string
		instance string
		body     string
		id       string
		expected int
	}{
		{"running matched", "running", "swe-full", "instance-1", `{"node_id":"swe-full","instance_id":"instance-1"}`, "exec-1", http.StatusNoContent},
		{"wrong node", "running", "swe-full", "instance-1", `{"node_id":"someone-else","instance_id":"instance-1"}`, "exec-1", http.StatusForbidden},
		{"wrong generation", "running", "swe-full", "instance-1", `{"node_id":"swe-full","instance_id":"previous-worker"}`, "exec-1", http.StatusForbidden},
		{"unfenced record", "running", "swe-full", "", `{"node_id":"swe-full","instance_id":"some-worker"}`, "exec-1", http.StatusForbidden},
		{"completed", "succeeded", "swe-full", "instance-1", `{"node_id":"swe-full","instance_id":"instance-1"}`, "exec-1", http.StatusConflict},
		{"failed", "failed", "swe-full", "instance-1", `{"node_id":"swe-full","instance_id":"instance-1"}`, "exec-1", http.StatusConflict},
		{"timed out", "timeout", "swe-full", "instance-1", `{"node_id":"swe-full","instance_id":"instance-1"}`, "exec-1", http.StatusConflict},
		{"approval pending", "waiting", "swe-full", "instance-1", `{"node_id":"swe-full","instance_id":"instance-1"}`, "exec-1", http.StatusConflict},
		{"unknown execution", "running", "swe-full", "instance-1", `{"node_id":"swe-full","instance_id":"instance-1"}`, "unknown", http.StatusNotFound},
		{"missing instance", "running", "swe-full", "instance-1", `{"node_id":"swe-full"}`, "exec-1", http.StatusBadRequest},
		{"missing node", "running", "swe-full", "instance-1", `{"instance_id":"instance-1"}`, "exec-1", http.StatusBadRequest},
		{"malformed", "running", "swe-full", "instance-1", "{", "exec-1", http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newTestExecutionStorage(nil)
			now := time.Now().UTC().Add(-time.Hour)
			require.NoError(t, store.CreateExecutionRecord(context.Background(), &types.Execution{
				ExecutionID: "exec-1", RunID: "run-test", AgentNodeID: tt.owner,
				InstanceID: tt.instance, Status: tt.state, StartedAt: now, UpdatedAt: now,
			}))
			router := gin.New()
			router.POST("/api/v1/executions/:execution_id/heartbeat", ExecutionHeartbeatHandler(store))
			req := httptest.NewRequest(http.MethodPost, "/api/v1/executions/"+tt.id+"/heartbeat", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)
			require.Equal(t, tt.expected, resp.Code, resp.Body.String())
			current, err := store.GetExecutionRecord(context.Background(), "exec-1")
			require.NoError(t, err)
			require.Equal(t, tt.state, current.Status, "heartbeat must never rewrite status")
			require.Equal(t, now, current.UpdatedAt, "in-memory test store leaves activity timestamp unchanged")
			require.Nil(t, current.CompletedAt, "heartbeat must never set terminal time")
		})
	}
}

func TestExecutionHeartbeatRejectOversize(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := newTestExecutionStorage(nil)
	router := gin.New()
	router.POST("/api/v1/executions/:execution_id/heartbeat", ExecutionHeartbeatHandler(store))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/executions/exec-1/heartbeat",
		strings.NewReader(`{"node_id":"swe-full","instance_id":"`+strings.Repeat("a", 3000)+`"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	require.Equal(t, http.StatusBadRequest, resp.Code)
}
