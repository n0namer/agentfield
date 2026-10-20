package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/Agent-Field/agentfield/control-plane/pkg/types"
	"github.com/gin-gonic/gin"
)

var (
	errExecutionHeartbeatOwner   = errors.New("execution heartbeat owner mismatch")
	errExecutionHeartbeatState   = errors.New("execution is not running")
	errExecutionHeartbeatMissing = errors.New("execution not found")
)

type executionHeartbeatRequest struct {
	NodeID     string `json:"node_id"`
	InstanceID string `json:"instance_id"`
}

// ExecutionHeartbeatHandler refreshes a live execution's activity timestamp.
//
// This is intentionally narrower than the existing /status callback. It cannot
// change the lifecycle, result, usage, or workflow DAG, nor resurrect a terminal
// execution. The update remains atomic with the stale reaper and other status
// writers by reusing the existing storage transaction.
func ExecutionHeartbeatHandler(store ExecutionStore) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		executionID := ctx.Param("execution_id")
		if executionID == "" || len(executionID) > 180 {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid execution id"})
			return
		}
		ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, 1024)
		var req executionHeartbeatRequest
		if err := json.NewDecoder(ctx.Request.Body).Decode(&req); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid heartbeat body"})
			return
		}
		if strings.TrimSpace(req.NodeID) == "" || len(req.NodeID) > 180 ||
			strings.TrimSpace(req.InstanceID) == "" || len(req.InstanceID) > 180 {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "node_id and instance_id required"})
			return
		}

		// The callback is allowed to touch only the same process instance that
		// originally admitted this execution. Do not accept an unversioned
		// instance, because that lets a new worker keep an old orphan alive.
		_, err := store.UpdateExecutionRecord(ctx.Request.Context(), executionID, func(current *types.Execution) (*types.Execution, error) {
			if current == nil {
				return nil, errExecutionHeartbeatMissing
			}
			if current.AgentNodeID != req.NodeID || current.InstanceID == "" ||
				current.InstanceID != req.InstanceID {
				return nil, errExecutionHeartbeatOwner
			}
			if current.Status != string(types.ExecutionStatusRunning) {
				return nil, errExecutionHeartbeatState
			}
			return current, nil
		})
		switch {
		case errors.Is(err, errExecutionHeartbeatMissing):
			ctx.JSON(http.StatusNotFound, gin.H{"error": "execution not found"})
		case errors.Is(err, errExecutionHeartbeatOwner):
			ctx.JSON(http.StatusForbidden, gin.H{"error": "heartbeat instance mismatch"})
		case errors.Is(err, errExecutionHeartbeatState):
			ctx.JSON(http.StatusConflict, gin.H{"error": "execution is not running"})
		case err != nil:
			// Some stores report an unknown ID before invoking the updater.
			// Probe only on the error path; do not disclose storage diagnostics.
			current, lookupErr := store.GetExecutionRecord(ctx.Request.Context(), executionID)
			if lookupErr == nil && current == nil {
				ctx.JSON(http.StatusNotFound, gin.H{"error": "execution not found"})
			} else {
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": "heartbeat update failed"})
			}
		default:
			ctx.Status(http.StatusNoContent)
		}
	}
}
