package handlers

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type runtimeLogFixtureEvent struct {
	Container string
	Time      string
	Level     string
	Message   string
}

type runtimeLogPayload struct {
	Level       string `json:"level"`
	Error       string `json:"error"`
	Time        string `json:"time"`
	Message     string `json:"message"`
	ExecutionID string `json:"execution_id"`
	Agent       string `json:"agent"`
	Reasoner    string `json:"reasoner"`
	NodeID      string `json:"node_id"`
}

type durableExecutionReadback struct {
	ExecutionID string
	RunID       string
	NodeID      string
	Status      string
}

type runtimeForensicSummary struct {
	AcceptedExecutionIDs      map[string]int
	DurableCorrelations       int
	StaleExecutionFailures    int
	StaleWorkflowFailures     int
	CanceledLifecycleFailures int
	AnonymousCallerWarnings   int
}

func decodeRuntimeLogPayload(event runtimeLogFixtureEvent) (runtimeLogPayload, error) {
	start := strings.Index(event.Message, "{")
	if start < 0 {
		return runtimeLogPayload{}, fmt.Errorf("runtime log has no structured payload")
	}
	var payload runtimeLogPayload
	structured := event.Message[start:]
	structured = strings.ReplaceAll(structured, `\"`, `"`)
	if err := json.Unmarshal([]byte(structured), &payload); err != nil {
		return runtimeLogPayload{}, fmt.Errorf("decode structured runtime log: %w", err)
	}
	return payload, nil
}

func correlateRuntimeLogs(events []runtimeLogFixtureEvent, durable map[string]durableExecutionReadback) (runtimeForensicSummary, error) {
	summary := runtimeForensicSummary{AcceptedExecutionIDs: make(map[string]int)}
	for _, event := range events {
		lowerMessage := strings.ToLower(event.Message)
		for _, sensitiveMarker := range []string{"authorization", "bearer ", "secret", "api_key", "access_token"} {
			if strings.Contains(lowerMessage, sensitiveMarker) {
				return runtimeForensicSummary{}, fmt.Errorf("privacy violation: runtime fixture contains %q", sensitiveMarker)
			}
		}

		payload, err := decodeRuntimeLogPayload(event)
		if err != nil {
			return runtimeForensicSummary{}, err
		}
		switch {
		case strings.Contains(payload.Message, "failed to mark stale executions as timed out"):
			summary.StaleExecutionFailures++
		case strings.Contains(payload.Message, "failed to mark stale workflow executions as timed out"):
			summary.StaleWorkflowFailures++
		case strings.Contains(payload.Message, "failed to update agent status from lease handler"):
			summary.CanceledLifecycleFailures++
		case strings.Contains(payload.Message, "Permission middleware: caller identity headers ignored"):
			summary.AnonymousCallerWarnings++
		case strings.Contains(payload.Message, "agent acknowledged async execution") || strings.Contains(payload.Message, "agent accepted execution for async processing"):
			if payload.ExecutionID == "" {
				return runtimeForensicSummary{}, fmt.Errorf("async acknowledgement has no execution_id")
			}
			readback, ok := durable[payload.ExecutionID]
			if !ok {
				return runtimeForensicSummary{}, fmt.Errorf("execution_id %q has no durable readback", payload.ExecutionID)
			}
			if payload.Agent != "" && readback.NodeID != payload.Agent {
				return runtimeForensicSummary{}, fmt.Errorf("execution %q node mismatch: log=%q durable=%q", payload.ExecutionID, payload.Agent, readback.NodeID)
			}
			summary.AcceptedExecutionIDs[payload.ExecutionID]++
			summary.DurableCorrelations++
		}
	}
	return summary, nil
}

func TestRuntimeLogForensics_CorrelatesDuplicateAcknowledgementToOneDurableRun(t *testing.T) {
	executionID := "exec_20261013_110654_nynu85bf"
	durable := map[string]durableExecutionReadback{
		executionID: {
			ExecutionID: executionID,
			RunID:       "run_20261013_demo_01",
			NodeID:      "demo-go-agent",
			Status:      "queued",
		},
	}
	events := []runtimeLogFixtureEvent{
		{Container: "docker-control-plane-1", Time: "2026-10-13T11:06:54.664513988Z", Level: "info", Message: `2026-10-13T11:06:54.664513988Z {"level":"info","execution_id":"exec_20261013_110654_nynu85bf","agent":"demo-go-agent","reasoner":"demo_echo","time":"2026-10-13T11:06:54Z","message":"agent acknowledged async execution"}`},
		{Container: "docker-control-plane-1", Time: "2026-10-13T11:06:54.664559055Z", Level: "info", Message: `2026-10-13T11:06:54.664559055Z {"level":"info","execution_id":"exec_20261013_110654_nynu85bf","time":"2026-10-13T11:06:54Z","message":"agent accepted execution for async processing"}`},
		{Container: "docker-control-plane-1", Time: "2026-10-13T10:58:28.824297831Z", Level: "error", Message: `2026-10-13T10:58:28.824297831Z {"level":"error","error":"prepare stale execution update: ERROR: syntax error at or near \"\" (SQLSTATE 42601)","time":"2026-10-13T10:58:28Z","message":"failed to mark stale executions as timed out"}`},
		{Container: "docker-control-plane-1", Time: "2026-10-13T10:58:28.826238788Z", Level: "error", Message: `2026-10-13T10:58:28.826238788Z {"level":"error","error":"prepare stale workflow execution update: ERROR: syntax error at or near \"\" (SQLSTATE 42601)","time":"2026-10-13T10:58:28Z","message":"failed to mark stale workflow executions as timed out"}`},
		{Container: "docker-control-plane-1", Time: "2026-10-13T10:58:36.166326925Z", Level: "error", Message: `2026-10-13T10:58:36.166326925Z {"level":"error","error":"failed to persist status: failed to update health status: failed to commit agent health status transaction: context canceled","node_id":"demo-go-agent","time":"2026-10-13T10:58:36Z","message":"failed to update agent status from lease handler"}`},
		{Container: "docker-control-plane-1", Time: "2026-10-13T10:57:16.313581268Z", Level: "warn", Message: `2026-10-13T10:57:16.313581268Z {"level":"warn","x_caller_agent_id_present":true,"x_agent_node_id_present":false,"time":"2026-10-13T10:57:16Z","message":"Permission middleware: caller identity headers ignored without a verified DID, treating caller as anonymous"}`},
	}

	summary, err := correlateRuntimeLogs(events, durable)
	require.NoError(t, err)
	require.Equal(t, 2, summary.AcceptedExecutionIDs[executionID], "two log acknowledgements must reconcile to one durable execution row")
	require.Equal(t, 2, summary.DurableCorrelations)
	require.Equal(t, 1, summary.StaleExecutionFailures)
	require.Equal(t, 1, summary.StaleWorkflowFailures)
	require.Equal(t, 1, summary.CanceledLifecycleFailures)
	require.Equal(t, 1, summary.AnonymousCallerWarnings)
	require.Len(t, summary.AcceptedExecutionIDs, 1)
	require.Equal(t, "run_20261013_demo_01", durable[executionID].RunID)
}

func TestRuntimeLogForensics_RejectsUncorrelatedOrSensitiveEvents(t *testing.T) {
	t.Run("missing durable execution readback", func(t *testing.T) {
		events := []runtimeLogFixtureEvent{{
			Container: "docker-control-plane-1",
			Message:   `2026-10-13T11:06:54Z {"level":"info","execution_id":"exec_missing","agent":"demo-go-agent","message":"agent acknowledged async execution"}`,
		}}
		_, err := correlateRuntimeLogs(events, map[string]durableExecutionReadback{})
		require.EqualError(t, err, `execution_id "exec_missing" has no durable readback`)
	})

	t.Run("durable node mismatch is refused", func(t *testing.T) {
		events := []runtimeLogFixtureEvent{{
			Container: "docker-control-plane-1",
			Message:   `2026-10-13T11:06:54Z {"level":"info","execution_id":"exec_mismatch","agent":"unexpected-agent","message":"agent acknowledged async execution"}`,
		}}
		_, err := correlateRuntimeLogs(events, map[string]durableExecutionReadback{
			"exec_mismatch": {ExecutionID: "exec_mismatch", RunID: "run_mismatch", NodeID: "demo-go-agent"},
		})
		require.EqualError(t, err, `execution "exec_mismatch" node mismatch: log="unexpected-agent" durable="demo-go-agent"`)
	})

	t.Run("sensitive payload is refused", func(t *testing.T) {
		events := []runtimeLogFixtureEvent{{
			Container: "docker-control-plane-1",
			Message:   `2026-10-13T11:06:54Z {"level":"error","error":"authorization bearer test-token","message":"agent call failed"}`,
		}}
		_, err := correlateRuntimeLogs(events, map[string]durableExecutionReadback{})
		require.Error(t, err)
		require.Contains(t, err.Error(), "privacy violation")
	})

	t.Run("malformed structured payload is refused", func(t *testing.T) {
		events := []runtimeLogFixtureEvent{{
			Container: "docker-control-plane-1",
			Message:   `2026-10-13T11:06:54Z {"level":"info","execution_id":"exec_broken"`,
		}}
		_, err := correlateRuntimeLogs(events, map[string]durableExecutionReadback{})
		require.Error(t, err)
		require.Contains(t, err.Error(), "decode structured runtime log")
	})
}
