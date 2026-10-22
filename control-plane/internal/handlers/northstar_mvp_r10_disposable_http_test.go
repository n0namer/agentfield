package handlers

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Agent-Field/agentfield/control-plane/internal/storage"
	"github.com/Agent-Field/agentfield/control-plane/pkg/types"
)

type mvpR10HTTPResult struct {
	ExecutionID string `json:"execution_id,omitempty"`
	RunID       string `json:"run_id,omitempty"`
	Status      string `json:"status,omitempty"`
	Error       string `json:"error,omitempty"`
}

func newMVPR10DisposableHTTPServer(store *liveIdempotencyCanaryStore) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		body, err := io.ReadAll(request.Body)
		if err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(writer).Encode(mvpR10HTTPResult{Error: err.Error()})
			return
		}
		caller := request.Header.Get("X-Verified-Caller")
		target := request.Header.Get("X-Target")
		requestKey := request.Header.Get("Idempotency-Key")
		digest := sha256.Sum256(body)
		idempotencyRequest := asyncIdempotencyRequest{
			scopeKey:   caller + "\x00" + target,
			target:     target,
			requestKey: requestKey,
			bodyDigest: hex.EncodeToString(digest[:]),
		}
		adapter := &asyncIdempotentExecutionStore{store: store, request: idempotencyRequest}
		execution := &types.Execution{
			ExecutionID: "mvp-r10-execution-" + caller,
			RunID:       "mvp-r10-run-" + caller,
			Status:      types.ExecutionStatusQueued,
		}
		err = adapter.CreateExecutionRecord(request.Context(), execution)
		if err != nil {
			var replay *asyncIdempotencyReplayError
			var conflict *asyncIdempotencyConflictError
			switch {
			case errors.As(err, &replay):
				writer.WriteHeader(http.StatusAccepted)
				_ = json.NewEncoder(writer).Encode(mvpR10HTTPResult{ExecutionID: replay.execution.ExecutionID, RunID: replay.execution.RunID, Status: "replay"})
			case errors.As(err, &conflict):
				writer.WriteHeader(http.StatusConflict)
				_ = json.NewEncoder(writer).Encode(mvpR10HTTPResult{Error: conflict.Error()})
			default:
				writer.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(writer).Encode(mvpR10HTTPResult{Error: err.Error()})
			}
			return
		}
		writer.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(writer).Encode(mvpR10HTTPResult{ExecutionID: execution.ExecutionID, RunID: execution.RunID, Status: "created"})
	}))
}

func postMVPR10(t *testing.T, client *http.Client, endpoint, caller, target, key string, payload []byte) (int, mvpR10HTTPResult) {
	t.Helper()
	request, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	request.Header.Set("X-Verified-Caller", caller)
	request.Header.Set("X-Target", target)
	request.Header.Set("Idempotency-Key", key)
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer response.Body.Close()
	var result mvpR10HTTPResult
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return response.StatusCode, result
}

func TestMVPR10(t *testing.T) {
	store := newLiveIdempotencyCanaryStore()
	server := newMVPR10DisposableHTTPServer(store)
	defer server.Close()

	t.Run("positive replay is typed durable and single effect", func(t *testing.T) {
		payload := []byte(`{"input":{"value":"alpha"}}`)
		status, first := postMVPR10(t, server.Client(), server.URL, "caller-a", "agent-1.echo", "mvp-r10-key", payload)
		if status != http.StatusAccepted || first.Status != "created" || first.ExecutionID == "" || first.RunID == "" {
			t.Fatalf("unexpected first result: status=%d result=%+v", status, first)
		}
		status, replay := postMVPR10(t, server.Client(), server.URL, "caller-a", "agent-1.echo", "mvp-r10-key", payload)
		if status != http.StatusAccepted || replay.Status != "replay" || replay.ExecutionID != first.ExecutionID || replay.RunID != first.RunID {
			t.Fatalf("unexpected replay result: status=%d first=%+v replay=%+v", status, first, replay)
		}
		store.mu.Lock()
		defer store.mu.Unlock()
		if store.createCount != 1 {
			t.Fatalf("expected one durable effect, got %d", store.createCount)
		}
	})

	t.Run("wrong payload and caller fail closed", func(t *testing.T) {
		status, result := postMVPR10(t, server.Client(), server.URL, "caller-a", "agent-1.echo", "mvp-r10-key", []byte(`{"input":{"value":"different"}}`))
		if status != http.StatusConflict || result.Error == "" {
			t.Fatalf("expected payload conflict, got status=%d result=%+v", status, result)
		}
		status, result = postMVPR10(t, server.Client(), server.URL, "caller-b", "agent-1.echo", "mvp-r10-key", []byte(`{"input":{"value":"alpha"}}`))
		if status != http.StatusAccepted || result.Status != "created" || result.ExecutionID == "" {
			t.Fatalf("expected caller-isolated creation, got status=%d result=%+v", status, result)
		}
	})

	t.Run("cancelled request creates no durable effect", func(t *testing.T) {
		before := store.createCount
		cancelled, cancel := context.WithCancel(context.Background())
		cancel()
		request := asyncIdempotencyRequest{scopeKey: "caller-cancel\x00agent-1.echo", target: "agent-1.echo", requestKey: "cancel-key", bodyDigest: "cancel-digest"}
		adapter := &asyncIdempotentExecutionStore{store: store, request: request}
		err := adapter.CreateExecutionRecord(cancelled, &types.Execution{ExecutionID: "cancelled"})
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected cancellation, got %v", err)
		}
		store.mu.Lock()
		defer store.mu.Unlock()
		if store.createCount != before {
			t.Fatalf("cancelled request changed durable count: before=%d after=%d", before, store.createCount)
		}
	})

	t.Run("replay survives HTTP server restart", func(t *testing.T) {
		server.Close()
		restarted := newMVPR10DisposableHTTPServer(store)
		defer restarted.Close()
		status, result := postMVPR10(t, restarted.Client(), restarted.URL, "caller-a", "agent-1.echo", "mvp-r10-key", []byte(`{"input":{"value":"alpha"}}`))
		if status != http.StatusAccepted || result.Status != "replay" || result.ExecutionID == "" || result.RunID == "" {
			t.Fatalf("expected durable replay after restart, got status=%d result=%+v", status, result)
		}
	})
}

var _ storage.AsyncExecutionIdempotencyStore = (*liveIdempotencyCanaryStore)(nil)
