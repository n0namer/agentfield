package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Agent-Field/agentfield/control-plane/internal/services"
	"github.com/Agent-Field/agentfield/control-plane/pkg/types"
	"github.com/gin-gonic/gin"
)

func TestR16HTTPSourceAndInterfacePin(t *testing.T) {
	source, err := os.ReadFile("execute.go")
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(source)
	const expected = "be09150579e29e16ce9d96775ef1b161898d829570b9bdd5885c9db5327a25c1"
	if actual := hex.EncodeToString(digest[:]); actual != expected {
		t.Fatalf("execute.go source drift: got %s, want %s; re-review the C4 binding", actual, expected)
	}
	var handler func(ExecutionStore, services.PayloadStore, services.WebhookDispatcher, time.Duration, string) gin.HandlerFunc = ExecuteAsyncHandler
	if handler == nil {
		t.Fatal("async execution handler is unavailable")
	}
	requestType := reflect.TypeOf(ExecuteRequest{})
	requestTags := map[string]string{
		"Input":       "input",
		"Context":     "context,omitempty",
		"Webhook":     "webhook,omitempty",
		"RunMetadata": "run_metadata,omitempty",
	}
	if requestType.NumField() != len(requestTags) {
		t.Fatalf("ExecuteRequest field count changed: %d", requestType.NumField())
	}
	for name, tag := range requestTags {
		field, found := requestType.FieldByName(name)
		if !found || field.Tag.Get("json") != tag {
			t.Fatalf("ExecuteRequest.%s wire tag changed: %v", name, field)
		}
	}
	responseType := reflect.TypeOf(AsyncExecuteResponse{})
	responseTags := map[string]string{
		"ExecutionID":       "execution_id",
		"RunID":             "run_id",
		"WorkflowID":        "workflow_id",
		"Status":            "status",
		"Target":            "target",
		"Type":              "type",
		"CreatedAt":         "created_at",
		"EnqueuedAt":        "enqueued_at,omitempty",
		"WebhookRegistered": "webhook_registered",
		"WebhookError":      "webhook_error,omitempty",
	}
	if responseType.NumField() != len(responseTags) {
		t.Fatalf("AsyncExecuteResponse field count changed: %d", responseType.NumField())
	}
	for name, tag := range responseTags {
		field, found := responseType.FieldByName(name)
		if !found || field.Tag.Get("json") != tag {
			t.Fatalf("AsyncExecuteResponse.%s wire tag changed: %v", name, field)
		}
	}
}

func TestR16HTTPRejectBeforeStoreOrDispatch(t *testing.T) {
	controller := &executionController{}
	router := gin.New()
	router.POST("/r16/execute/:target", controller.handleAsync)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	client := server.Client()
	client.Timeout = 5 * time.Second
	cases := []struct {
		name string
		key  string
		code int
	}{
		{name: "unverified_caller", key: "r16-key", code: http.StatusUnauthorized},
		{name: "blank_key", key: " ", code: http.StatusBadRequest},
		{name: "embedded_space", key: "r16 key", code: http.StatusBadRequest},
		{name: "oversized_key", key: strings.Repeat("x", maxAsyncIdempotencyKeyLength+1), code: http.StatusBadRequest},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			request, err := http.NewRequest(http.MethodPost, server.URL+"/r16/execute/disposable.reasoner", strings.NewReader(`{"input":{"value":1}}`))
			if err != nil {
				t.Fatal(err)
			}
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Idempotency-Key", testCase.key)
			request.Header.Set("X-Caller-DID", "did:example:untrusted")
			response, err := client.Do(request)
			if err != nil {
				t.Fatalf("request must be rejected before touching the nil store or worker pool: %v", err)
			}
			defer response.Body.Close()
			body, err := io.ReadAll(response.Body)
			if err != nil {
				t.Fatal(err)
			}
			if response.StatusCode != testCase.code {
				t.Fatalf("got HTTP %d, want %d: %s", response.StatusCode, testCase.code, body)
			}
			if !json.Valid(body) {
				t.Fatalf("rejection is not JSON: %s", body)
			}
			if response.Header.Get("X-Execution-ID") != "" || response.Header.Get("X-AgentField-Idempotency-Replay") != "" {
				t.Fatal("rejected request received an execution or replay receipt")
			}
		})
	}
}

func TestR16HTTPCancelledUnverifiedRequest(t *testing.T) {
	requestContext, cancel := context.WithCancel(context.Background())
	cancel()
	request := httptest.NewRequest(http.MethodPost, "/r16/execute/disposable.reasoner", strings.NewReader(`{"input":{}}`)).WithContext(requestContext)
	request.Header.Set("Idempotency-Key", "r16-cancelled")
	router := gin.New()
	controller := &executionController{}
	router.POST("/r16/execute/:target", controller.handleAsync)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("cancelled unverified request got %d, want 401: %s", recorder.Code, recorder.Body.String())
	}
	if recorder.Header().Get("X-Execution-ID") != "" {
		t.Fatal("cancelled unverified request received an execution receipt")
	}
}

func TestR16HTTPReplayReceiptFormatting(t *testing.T) {
	createdAt := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)
	execution := &types.Execution{
		ExecutionID: "r16-fixture-execution",
		RunID:       "r16-fixture-run",
		Status:      types.ExecutionStatusSucceeded,
		CreatedAt:   createdAt,
	}
	router := gin.New()
	router.POST("/r16/receipt", func(requestContext *gin.Context) {
		writeAsyncIdempotencyResponse(requestContext, execution, "disposable.reasoner")
	})
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	client := server.Client()
	client.Timeout = 5 * time.Second
	for attempt := 0; attempt < 2; attempt++ {
		response, err := client.Post(server.URL+"/r16/receipt", "application/json", strings.NewReader(`{"input":{}}`))
		if err != nil {
			t.Fatal(err)
		}
		var receipt map[string]interface{}
		err = json.NewDecoder(response.Body).Decode(&receipt)
		response.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != http.StatusAccepted {
			t.Fatalf("got %d, want 202", response.StatusCode)
		}
		if response.Header.Get("X-AgentField-Idempotency-Replay") != "true" || response.Header.Get("X-Execution-ID") != execution.ExecutionID || response.Header.Get("X-Run-ID") != execution.RunID {
			t.Fatalf("unexpected replay headers: %v", response.Header)
		}
		expected := map[string]interface{}{
			"execution_id":       execution.ExecutionID,
			"run_id":             execution.RunID,
			"workflow_id":        execution.RunID,
			"status":             string(execution.Status),
			"target":             "disposable.reasoner",
			"type":               "",
			"created_at":         createdAt.Format(time.RFC3339),
			"enqueued_at":        createdAt.Format(time.RFC3339),
			"webhook_registered": false,
		}
		if !reflect.DeepEqual(receipt, expected) {
			t.Fatalf("replay wire contract mismatch: got %v, want %v", receipt, expected)
		}
	}
}
