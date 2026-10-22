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

	"github.com/Agent-Field/agentfield/control-plane/pkg/types"
)

type mvpR11QAHTTPResult struct {
	ExecutionID string `json:"execution_id,omitempty"`
	RunID       string `json:"run_id,omitempty"`
	Status      string `json:"status,omitempty"`
	Error       string `json:"error,omitempty"`
}

func newMVPR11QAHTTPServer(store *liveIdempotencyCanaryStore) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		body, err := io.ReadAll(request.Body)
		if err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(writer).Encode(mvpR11QAHTTPResult{Error: err.Error()})
			return
		}

		caller := request.Header.Get("X-Verified-Caller")
		target := request.Header.Get("X-Target")
		requestKey := request.Header.Get("Idempotency-Key")
		digest := sha256.Sum256(body)
		adapter := &asyncIdempotentExecutionStore{
			store: store,
			request: asyncIdempotencyRequest{
				scopeKey:   caller + "\x00" + target,
				target:     target,
				requestKey: requestKey,
				bodyDigest: hex.EncodeToString(digest[:]),
			},
		}
		execution := &types.Execution{
			ExecutionID: "mvp-r11-qa-execution-" + caller,
			RunID:       "mvp-r11-qa-run-" + caller,
			Status:      types.ExecutionStatusQueued,
		}
		if err := adapter.CreateExecutionRecord(request.Context(), execution); err != nil {
			var replay *asyncIdempotencyReplayError
			var conflict *asyncIdempotencyConflictError
			switch {
			case errors.As(err, &replay):
				writer.WriteHeader(http.StatusAccepted)
				_ = json.NewEncoder(writer).Encode(mvpR11QAHTTPResult{ExecutionID: replay.execution.ExecutionID, RunID: replay.execution.RunID, Status: "replay"})
			case errors.As(err, &conflict):
				writer.WriteHeader(http.StatusConflict)
				_ = json.NewEncoder(writer).Encode(mvpR11QAHTTPResult{Error: conflict.Error()})
			default:
				writer.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(writer).Encode(mvpR11QAHTTPResult{Error: err.Error()})
			}
			return
		}

		writer.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(writer).Encode(mvpR11QAHTTPResult{ExecutionID: execution.ExecutionID, RunID: execution.RunID, Status: "created"})
	}))
}

func postMVPR11QA(t *testing.T, client *http.Client, endpoint, caller, target, key string, payload []byte) (int, mvpR11QAHTTPResult) {
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

	var result mvpR11QAHTTPResult
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return response.StatusCode, result
}

func r11QACreateCount(store *liveIdempotencyCanaryStore) int {
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.createCount
}

func TestMVPR11(t *testing.T) {
	store := newLiveIdempotencyCanaryStore()
	server := newMVPR11QAHTTPServer(store)
	defer server.Close()

	payload := []byte(`{"input":{"value":"r11-qa"}}`)
	const (
		caller = "r11-qa-caller"
		target = "agent-1.echo"
		key    = "mvp-r11-qa-key"
	)

	t.Run("positive request and replay preserve one durable effect", func(t *testing.T) {
		status, created := postMVPR11QA(t, server.Client(), server.URL, caller, target, key, payload)
		if status != http.StatusAccepted || created.Status != "created" || created.ExecutionID == "" || created.RunID == "" {
			t.Fatalf("unexpected created response: status=%d result=%+v", status, created)
		}

		status, replay := postMVPR11QA(t, server.Client(), server.URL, caller, target, key, payload)
		if status != http.StatusAccepted || replay.Status != "replay" || replay.ExecutionID != created.ExecutionID || replay.RunID != created.RunID {
			t.Fatalf("unexpected replay response: status=%d created=%+v replay=%+v", status, created, replay)
		}
		if count := r11QACreateCount(store); count != 1 {
			t.Fatalf("expected exactly one durable effect, got %d", count)
		}
	})

	t.Run("different body for same caller target and key fails closed", func(t *testing.T) {
		status, result := postMVPR11QA(t, server.Client(), server.URL, caller, target, key, []byte(`{"input":{"value":"tampered"}}`))
		if status != http.StatusConflict || result.Error == "" {
			t.Fatalf("expected conflict for changed request body: status=%d result=%+v", status, result)
		}
		if count := r11QACreateCount(store); count != 1 {
			t.Fatalf("conflicting retry produced a durable effect: count=%d", count)
		}
	})

	t.Run("cancelled request is rejected before persistence", func(t *testing.T) {
		before := r11QACreateCount(store)
		cancelled, cancel := context.WithCancel(context.Background())
		cancel()
		adapter := &asyncIdempotentExecutionStore{
			store: store,
			request: asyncIdempotencyRequest{
				scopeKey:   "r11-cancelled\x00" + target,
				target:     target,
				requestKey: "r11-cancelled-key",
				bodyDigest: "r11-cancelled-digest",
			},
		}
		err := adapter.CreateExecutionRecord(cancelled, &types.Execution{ExecutionID: "r11-cancelled-execution"})
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context cancellation, got %v", err)
		}
		if after := r11QACreateCount(store); after != before {
			t.Fatalf("cancelled request changed durable count: before=%d after=%d", before, after)
		}
	})

	t.Run("replay survives disposable HTTP process restart", func(t *testing.T) {
		server.Close()
		restarted := newMVPR11QAHTTPServer(store)
		defer restarted.Close()

		status, replay := postMVPR11QA(t, restarted.Client(), restarted.URL, caller, target, key, payload)
		if status != http.StatusAccepted || replay.Status != "replay" || replay.ExecutionID == "" || replay.RunID == "" {
			t.Fatalf("expected durable replay after restart: status=%d result=%+v", status, replay)
		}
		if count := r11QACreateCount(store); count != 1 {
			t.Fatalf("restart replay produced a duplicate durable effect: count=%d", count)
		}
	})
}
