package harness

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

type waveLifecycleProvider struct {
	calls       int
	prompts     []string
	resumeIDs   []string
	writeOutput bool
}

func (p *waveLifecycleProvider) Execute(_ context.Context, prompt string, opts Options) (*RawResult, error) {
	p.calls++
	p.prompts = append(p.prompts, prompt)
	p.resumeIDs = append(p.resumeIDs, opts.ResumeSessionID)
	if p.writeOutput {
		content, err := json.Marshal(map[string]string{"answer": "recovered"})
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile(OutputPath(opts.ProjectDir), content, 0o600); err != nil {
			return nil, err
		}
		return &RawResult{
			Metrics: Metrics{
				Model:           "recovery-model",
				SessionID:       "recovered-session",
				NumTurns:        2,
				InputTokens:     11,
				OutputTokens:    7,
				CacheReadTokens: 3,
				CostUSD:         floatPtr(1.25),
			},
		}, nil
	}
	return &RawResult{Result: "ok"}, nil
}

func TestWaveGoalLifecycle_ProviderSelectionStructuredRecoveryAndAccounting(t *testing.T) {
	provider, err := BuildProvider(ProviderCodex, "")
	if err != nil || provider == nil {
		t.Fatalf("codex provider selection failed: provider=%T err=%v", provider, err)
	}
	if _, err := BuildProvider("not-a-provider", ""); err == nil || !strings.Contains(err.Error(), "unknown harness provider") {
		t.Fatalf("wrong provider model must fail deterministically, got %v", err)
	}

	dir := t.TempDir()
	providerDouble := &waveLifecycleProvider{writeOutput: true}
	initial := &RawResult{
		IsError:      true,
		ErrorMessage: "agent session crashed",
		FailureType:  FailureCrash,
		Metrics: Metrics{
			Model:        "wrong-model",
			SessionID:    "crashed-session",
			InputTokens:  5,
			OutputTokens: 2,
			CostUSD:      floatPtr(0.75),
		},
	}
	type output struct {
		Answer string `json:"answer"`
	}
	var dest output
	result := (&Runner{Logger: log.New(io.Discard, "", 0)}).handleSchemaWithRetry(
		context.Background(), initial,
		map[string]any{
			"type":     "object",
			"required": []any{"answer"},
			"properties": map[string]any{
				"answer": map[string]any{"type": "string"},
			},
		},
		&dest, dir, time.Now(), providerDouble,
		Options{SchemaMaxRetries: 1, ProjectDir: dir}, "complete the goal", false,
	)

	if result.IsError {
		t.Fatalf("crashed session should recover: %s", result.ErrorMessage)
	}
	if dest.Answer != "recovered" || providerDouble.calls != 1 {
		t.Fatalf("unexpected recovery state: dest=%+v calls=%d", dest, providerDouble.calls)
	}
	if providerDouble.resumeIDs[0] != "" {
		t.Fatalf("crash recovery must not resume the failed session: %q", providerDouble.resumeIDs[0])
	}
	if result.SessionID != "recovered-session" || result.Model != "wrong-model" {
		t.Fatalf("deterministic post-state lost lifecycle metadata: session=%q model=%q", result.SessionID, result.Model)
	}
	if result.NumTurns != 2 || result.CostUSD == nil || *result.CostUSD != 2.0 {
		t.Fatalf("metrics were not accumulated across attempts: turns=%d cost=%v", result.NumTurns, result.CostUSD)
	}
	assertTokenField(t, result, "InputTokens", 16)
	assertTokenField(t, result, "OutputTokens", 9)
	assertTokenField(t, result, "CacheReadTokens", 3)
	if _, err := os.Stat(OutputPath(dir)); err != nil {
		t.Fatalf("recovery artifact missing: %v", err)
	}
}

func TestWaveGoalLifecycle_CancellationWinsOverLateProviderSuccess(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	calls := 0
	provider := &funcProvider{fn: func(context.Context, string, Options) (*RawResult, error) {
		calls++
		return &RawResult{Result: "late"}, nil
	}}
	_, err := (&Runner{}).executeWithRetry(ctx, provider, "goal", Options{MaxRetries: 1})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled context must be returned, got %v", err)
	}
	if calls != 0 {
		t.Fatalf("canceled context must prevent provider side effects, calls=%d", calls)
	}

	ctx, cancel = context.WithCancel(context.Background())
	provider = &funcProvider{fn: func(context.Context, string, Options) (*RawResult, error) {
		cancel()
		return &RawResult{Result: "late"}, nil
	}}
	_, err = (&Runner{}).executeWithRetry(ctx, provider, "goal", Options{MaxRetries: 1})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation during provider execution must win, got %v", err)
	}
}

func TestWaveGoalLifecycle_NilProviderResultIsBoundedError(t *testing.T) {
	provider := &funcProvider{fn: func(context.Context, string, Options) (*RawResult, error) {
		return nil, nil
	}}
	_, err := (&Runner{}).executeWithRetry(context.Background(), provider, "goal", Options{MaxRetries: 1})
	if err == nil || !strings.Contains(err.Error(), "nil provider result") {
		t.Fatalf("nil provider result must be rejected, got %v", err)
	}
}

func assertTokenField(t *testing.T, result *Result, field string, want int) {
	t.Helper()
	value := reflect.ValueOf(result).Elem().FieldByName(field)
	if !value.IsValid() {
		t.Fatalf("result missing token field %s", field)
	}
	if value.Int() != int64(want) {
		t.Fatalf("%s=%d, want %d", field, value.Int(), want)
	}
}

func floatPtr(value float64) *float64 { return &value }
