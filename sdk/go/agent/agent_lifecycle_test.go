package agent

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Agent-Field/agentfield/sdk/go/did"
	"github.com/Agent-Field/agentfield/sdk/go/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitialize_AlreadyInitializedIsNoop(t *testing.T) {
	a, err := New(Config{
		NodeID:        "node-1",
		Version:       "1.0.0",
		AgentFieldURL: "https://example.com",
		Logger:        log.New(io.Discard, "", 0),
	})
	require.NoError(t, err)

	// Once initialized, Initialize should return immediately.
	a.initialized = true
	require.NoError(t, a.Initialize(context.Background()))
}

func TestInitialize_WrapsRegisterNodeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "temporarily unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	a, err := New(Config{
		NodeID:        "node-1",
		Version:       "1.0.0",
		AgentFieldURL: server.URL,
		Logger:        log.New(io.Discard, "", 0),
	})
	require.NoError(t, err)
	a.RegisterReasoner("demo", func(context.Context, map[string]any) (any, error) { return nil, nil })

	err = a.Initialize(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "register node:")
}

// Validation contract: a reasoner registered with WithDescription must carry
// that description in the node-registration payload sent to the control plane
// (it was previously local-only, used for CLI help).
func TestRegisterNode_TransmitsReasonerDescription(t *testing.T) {
	var payload types.NodeRegistrationRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			_ = json.NewDecoder(r.Body).Decode(&payload)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"registered"}`))
	}))
	defer server.Close()

	a, err := New(Config{
		NodeID:        "node-desc",
		Version:       "1.0.0",
		AgentFieldURL: server.URL,
		Logger:        log.New(io.Discard, "", 0),
	})
	require.NoError(t, err)
	a.RegisterReasoner("implement_issue",
		func(context.Context, map[string]any) (any, error) { return nil, nil },
		WithDescription("Implement one scoped issue on a branch"),
		WithReasonerTags("entrypoint"),
	)
	a.RegisterReasoner("run_coder",
		func(context.Context, map[string]any) (any, error) { return nil, nil },
	)

	require.NoError(t, a.registerNode(context.Background()))

	byID := map[string]types.ReasonerDefinition{}
	for _, r := range payload.Reasoners {
		byID[r.ID] = r
	}
	require.Len(t, byID, 2)
	assert.Equal(t, "Implement one scoped issue on a branch", byID["implement_issue"].Description)
	assert.Contains(t, byID["implement_issue"].Tags, "entrypoint")
	assert.Empty(t, byID["run_coder"].Description)
}

func TestAgentInstanceIDPropagatesAndChangesPerProcess(t *testing.T) {
	var regs, beats []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var v map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&v))
		id, _ := v["instance_id"].(string)
		if r.Method == http.MethodPost {
			regs = append(regs, id)
		} else if r.Method == http.MethodPatch {
			beats = append(beats, id)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"registered","lease_seconds":300,"next_lease_renewal":"2099-01-01T00:00:00Z"}`))
	}))
	defer srv.Close()
	mk := func() *Agent {
		a, e := New(Config{NodeID: "same-node", Version: "1.0.0", AgentFieldURL: srv.URL, Logger: log.New(io.Discard, "", 0)})
		require.NoError(t, e)
		return a
	}
	a := mk()
	require.NoError(t, a.registerNode(context.Background()))
	require.NoError(t, a.markReady(context.Background()))
	require.NoError(t, a.registerNode(context.Background())) // reconnect, same process identity
	b2 := mk()
	require.NoError(t, b2.registerNode(context.Background()))
	require.Len(t, regs, 3)
	require.Len(t, beats, 1)
	require.Len(t, regs[0], 32)
	require.Len(t, regs[2], 32)
	assert.Equal(t, byte('4'), regs[0][12], "instance id must use UUIDv4 version bits")
	assert.Contains(t, []byte{'8', '9', 'a', 'b'}, regs[0][16], "instance id must use UUID variant bits")
	assert.Equal(t, regs[0], beats[0])
	assert.Equal(t, regs[0], regs[1], "same Agent process must keep one instance id across reconnect")
	assert.NotEqual(t, regs[0], regs[2], "new Agent process needs a fresh instance id")
}

func TestInitialize_ContinuesWhenDIDOrReadyUpdatesFail(t *testing.T) {
	agentDID, _ := testDIDCredentials(t)
	var statusCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/nodes":
			w.WriteHeader(http.StatusOK)
			require.NoError(t, json.NewEncoder(w).Encode(types.NodeRegistrationResponse{
				ID:      "node-1",
				Success: true,
			}))
		case "/api/v1/nodes/node-1/status":
			statusCalls++
			http.Error(w, "status failed", http.StatusBadGateway)
		case "/api/v1/did/register":
			w.Header().Set("Content-Type", "application/json")
			// Successful DID registration followed by invalid credentials exercises
			// the warning-only path inside Initialize.
			require.NoError(t, json.NewEncoder(w).Encode(did.RegistrationResponse{
				Success: true,
				IdentityPackage: did.DIDIdentityPackage{
					AgentDID: did.DIDIdentity{
						DID:           agentDID,
						PrivateKeyJWK: "{invalid",
					},
				},
			}))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	a, err := New(Config{
		NodeID:           "node-1",
		Version:          "1.0.0",
		AgentFieldURL:    server.URL,
		EnableDID:        true,
		DisableLeaseLoop: true,
		Logger:           log.New(io.Discard, "", 0),
	})
	require.NoError(t, err)
	a.RegisterReasoner("demo", func(context.Context, map[string]any) (any, error) { return nil, nil })

	require.NoError(t, a.Initialize(context.Background()))
	assert.True(t, a.initialized)
	assert.Equal(t, 1, statusCalls)
}

func TestWaitForApproval_CompletesAfterPollAndLogsPollingErrors(t *testing.T) {
	var polls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/nodes/node-1":
			polls++
			if polls == 1 {
				// The first poll failing should not abort the approval loop.
				http.Error(w, "try again", http.StatusBadGateway)
				return
			}
			w.WriteHeader(http.StatusOK)
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"id":               "node-1",
				"lifecycle_status": "ready",
			}))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	a, err := New(Config{
		NodeID:        "node-1",
		Version:       "1.0.0",
		AgentFieldURL: server.URL,
		Logger:        log.New(io.Discard, "", 0),
	})
	require.NoError(t, err)

	require.NoError(t, a.waitForApproval(context.Background()))
	assert.GreaterOrEqual(t, polls, 2)
}

func TestShutdownCancelsInFlightReasoners(t *testing.T) {
	a, err := New(Config{NodeID: "node-1", Version: "1.0.0", Logger: log.New(io.Discard, "", 0)})
	require.NoError(t, err)
	ctx, release := a.registerCancellableExecution(context.Background(), "exec-1")
	defer release()
	require.NoError(t, a.shutdown(context.Background()))
	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("shutdown did not cancel in-flight execution")
	}
	a.cancelMu.Lock()
	left := len(a.cancelFuncs)
	a.cancelMu.Unlock()
	assert.Zero(t, left)
}

func TestShutdown_HandlesNilClientAndNilServer(t *testing.T) {
	a, err := New(Config{
		NodeID:  "node-1",
		Version: "1.0.0",
		Logger:  log.New(io.Discard, "", 0),
	})
	require.NoError(t, err)

	require.NoError(t, a.shutdown(context.Background()))
}

func TestRegisteredHeartbeatInterval(t *testing.T) {
	a, err := New(Config{
		NodeID:               "node-1",
		Version:              "1.0.0",
		LeaseRefreshInterval: 15 * time.Second,
		Logger:               log.New(io.Discard, "", 0),
	})
	require.NoError(t, err)

	assert.Equal(t, "15s", a.registeredHeartbeatInterval())
	a.cfg.DisableLeaseLoop = true
	assert.Equal(t, "0s", a.registeredHeartbeatInterval())
}
