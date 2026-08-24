package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/viper"
)

func writeTestConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "agentfield.yaml")
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func TestLoadConfigDefaultsExecutionCleanup(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	path := writeTestConfig(t, `
storage:
  mode: postgres
`)
	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatal(err)
	}

	cleanup := cfg.AgentField.ExecutionCleanup
	if !cleanup.Enabled {
		t.Fatal("execution cleanup must default to enabled")
	}
	if cleanup.RetentionPeriod != 24*time.Hour {
		t.Fatalf("retention=%s", cleanup.RetentionPeriod)
	}
	if cleanup.CleanupInterval != time.Hour {
		t.Fatalf("cleanup_interval=%s", cleanup.CleanupInterval)
	}
	if cleanup.BatchSize != 100 {
		t.Fatalf("batch_size=%d", cleanup.BatchSize)
	}
	if cleanup.PreserveRecentDuration != time.Hour {
		t.Fatalf("preserve_recent_duration=%s", cleanup.PreserveRecentDuration)
	}
	if cleanup.StaleExecutionTimeout != 30*time.Minute {
		t.Fatalf("stale_execution_timeout=%s", cleanup.StaleExecutionTimeout)
	}
}

func TestLoadConfigPreservesExplicitDisabledCleanup(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	path := writeTestConfig(t, `
storage:
  mode: postgres
agentfield:
  execution_cleanup:
    enabled: false
    cleanup_interval: 1h
`)
	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.AgentField.ExecutionCleanup.Enabled {
		t.Fatal("explicit execution_cleanup.enabled=false must be preserved")
	}
}
