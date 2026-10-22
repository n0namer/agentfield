package handlers

import (
	"net/http"
	"testing"
)

func TestMVPR12HTTPReplay(t *testing.T) {
	t.Run("same idempotency key returns the original execution once", func(t *testing.T) {
		store := newLiveIdempotencyCanaryStore()
		server := newMVPR10DisposableHTTPServer(store)
		defer server.Close()

		payload := []byte(`{"input":{"value":"alpha"}}`)
		status, first := postMVPR10(t, server.Client(), server.URL, "owner-a", "agent-1.echo", "r12-replay-key", payload)
		if status != http.StatusAccepted || first.Status != "created" || first.ExecutionID == "" || first.RunID == "" {
			t.Fatalf("unexpected first result: status=%d result=%+v", status, first)
		}

		status, replay := postMVPR10(t, server.Client(), server.URL, "owner-a", "agent-1.echo", "r12-replay-key", payload)
		if status != http.StatusAccepted || replay.Status != "replay" || replay.ExecutionID != first.ExecutionID || replay.RunID != first.RunID {
			t.Fatalf("unexpected replay result: status=%d first=%+v replay=%+v", status, first, replay)
		}

		store.mu.Lock()
		defer store.mu.Unlock()
		if store.createCount != 1 {
			t.Fatalf("expected one durable effect for replayed request, got %d", store.createCount)
		}
	})

	t.Run("foreign owner cannot replay another owner's execution", func(t *testing.T) {
		store := newLiveIdempotencyCanaryStore()
		server := newMVPR10DisposableHTTPServer(store)
		defer server.Close()

		payload := []byte(`{"input":{"value":"alpha"}}`)
		status, owner := postMVPR10(t, server.Client(), server.URL, "owner-a", "agent-1.echo", "r12-owner-key", payload)
		if status != http.StatusAccepted || owner.Status != "created" || owner.ExecutionID == "" {
			t.Fatalf("unexpected owner result: status=%d result=%+v", status, owner)
		}

		status, foreign := postMVPR10(t, server.Client(), server.URL, "owner-b", "agent-1.echo", "r12-owner-key", payload)
		if status != http.StatusAccepted || foreign.Status != "created" || foreign.ExecutionID == "" {
			t.Fatalf("expected caller-isolated creation, got status=%d result=%+v", status, foreign)
		}
		if foreign.ExecutionID == owner.ExecutionID || foreign.RunID == owner.RunID {
			t.Fatalf("foreign caller received owner execution: owner=%+v foreign=%+v", owner, foreign)
		}

		status, ownerReplay := postMVPR10(t, server.Client(), server.URL, "owner-a", "agent-1.echo", "r12-owner-key", payload)
		if status != http.StatusAccepted || ownerReplay.Status != "replay" || ownerReplay.ExecutionID != owner.ExecutionID || ownerReplay.RunID != owner.RunID {
			t.Fatalf("owner replay drifted after foreign request: status=%d owner=%+v replay=%+v", status, owner, ownerReplay)
		}

		store.mu.Lock()
		defer store.mu.Unlock()
		if store.createCount != 2 {
			t.Fatalf("expected one isolated effect per caller, got %d", store.createCount)
		}
	})

	t.Run("digest mismatch is rejected without a second durable effect", func(t *testing.T) {
		store := newLiveIdempotencyCanaryStore()
		server := newMVPR10DisposableHTTPServer(store)
		defer server.Close()

		status, first := postMVPR10(t, server.Client(), server.URL, "owner-a", "agent-1.echo", "r12-digest-key", []byte(`{"input":{"value":"alpha"}}`))
		if status != http.StatusAccepted || first.Status != "created" {
			t.Fatalf("unexpected first result: status=%d result=%+v", status, first)
		}

		status, conflict := postMVPR10(t, server.Client(), server.URL, "owner-a", "agent-1.echo", "r12-digest-key", []byte(`{"input":{"value":"different"}}`))
		if status != http.StatusConflict || conflict.Error == "" {
			t.Fatalf("expected digest conflict, got status=%d result=%+v", status, conflict)
		}

		store.mu.Lock()
		defer store.mu.Unlock()
		if store.createCount != 1 {
			t.Fatalf("digest conflict created an additional durable effect: %d", store.createCount)
		}
	})

	t.Run("replay remains bound after HTTP server restart", func(t *testing.T) {
		store := newLiveIdempotencyCanaryStore()
		server := newMVPR10DisposableHTTPServer(store)

		payload := []byte(`{"input":{"value":"alpha"}}`)
		status, first := postMVPR10(t, server.Client(), server.URL, "owner-a", "agent-1.echo", "r12-restart-key", payload)
		if status != http.StatusAccepted || first.Status != "created" || first.ExecutionID == "" || first.RunID == "" {
			server.Close()
			t.Fatalf("unexpected first result: status=%d result=%+v", status, first)
		}
		server.Close()

		restarted := newMVPR10DisposableHTTPServer(store)
		defer restarted.Close()
		status, replay := postMVPR10(t, restarted.Client(), restarted.URL, "owner-a", "agent-1.echo", "r12-restart-key", payload)
		if status != http.StatusAccepted || replay.Status != "replay" || replay.ExecutionID != first.ExecutionID || replay.RunID != first.RunID {
			t.Fatalf("unexpected restart replay: status=%d first=%+v replay=%+v", status, first, replay)
		}

		store.mu.Lock()
		defer store.mu.Unlock()
		if store.createCount != 1 {
			t.Fatalf("restart replay created an additional durable effect: %d", store.createCount)
		}
	})
}
