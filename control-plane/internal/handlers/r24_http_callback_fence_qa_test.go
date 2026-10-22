package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestR24HTTPCallbackFence(t *testing.T) {
	t.Run("unverified_caller_cannot_receive_execution_receipt", func(t *testing.T) {
		router := gin.New()
		router.POST("/execute/:target", ExecuteAsyncHandler(nil, nil, nil, time.Second, ""))
		request := httptest.NewRequest(http.MethodPost, "/execute/r24-target", bytes.NewBufferString(`{"input":{}}`))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Idempotency-Key", "r24-unverified-caller")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		if response.Code != http.StatusUnauthorized {
			t.Fatalf("unverified caller: got HTTP %d, want %d; body=%s", response.Code, http.StatusUnauthorized, response.Body.String())
		}
		if response.Header().Get("X-AgentField-Idempotency-Replay") != "" {
			t.Fatal("rejected caller received a replay receipt")
		}
		var receipt map[string]json.RawMessage
		if err := json.Unmarshal(response.Body.Bytes(), &receipt); err != nil {
			t.Fatalf("decode rejection response: %v", err)
		}
		for _, field := range []string{"execution_id", "run_id", "workflow_id"} {
			if _, exists := receipt[field]; exists {
				t.Fatalf("rejected caller received %s", field)
			}
		}
	})

	t.Run("malformed_callback_cannot_reach_execution_store", func(t *testing.T) {
		router := gin.New()
		router.POST("/executions/:execution_id/status", UpdateExecutionStatusHandler(nil, nil, nil, time.Second))
		request := httptest.NewRequest(http.MethodPost, "/executions/r24-execution/status", bytes.NewBufferString(`{"status":`))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		if response.Code != http.StatusBadRequest {
			t.Fatalf("malformed callback: got HTTP %d, want %d; body=%s", response.Code, http.StatusBadRequest, response.Body.String())
		}
		if response.Header().Get("X-AgentField-Idempotency-Replay") != "" {
			t.Fatal("malformed callback received a replay receipt")
		}
	})
}
