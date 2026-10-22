package harness

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestR24GoRunnerRestart(t *testing.T) {
	t.Run("construct_runner", func(t *testing.T) {
		runner := NewRunner(Options{Provider: "codex"})
		if runner == nil || runner.Logger == nil || runner.DefaultOptions.Provider != "codex" {
			t.Fatal("runner construction did not preserve its configuration")
		}
	})

	t.Run("unknown_provider_has_no_schema_effect", func(t *testing.T) {
		project := filepath.Join(t.TempDir(), "not-created")
		runner := NewRunner(Options{Provider: "r24-unsupported-provider", ProjectDir: project})
		result, err := runner.Run(context.Background(), "offline boundary", map[string]any{"type": "object"}, nil, Options{})
		if err == nil || result != nil {
			t.Fatalf("unsupported provider must fail without a result: result=%+v err=%v", result, err)
		}
		if _, statErr := os.Stat(project); !errors.Is(statErr, os.ErrNotExist) {
			t.Fatalf("provider rejection must precede schema output creation: %v", statErr)
		}
	})

	t.Run("canceled_call_has_no_success", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		runner := NewRunner(Options{Provider: "codex", BinPath: filepath.Join(t.TempDir(), "absent-provider")})
		result, err := runner.Run(ctx, "must not dispatch", nil, nil, Options{})
		if !errors.Is(err, context.Canceled) || result != nil {
			t.Fatalf("cancellation must win without a result: result=%+v err=%v", result, err)
		}
	})

	t.Run("process_restart_before_dispatch", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		executable, err := os.Executable()
		if err != nil {
			t.Fatal(err)
		}
		command := exec.CommandContext(ctx, executable, "-test.run=^TestR24GoRunnerRestartProcessHelper$", "-test.count=1")
		command.Stdin = nil
		stdout, err := command.StdoutPipe()
		if err != nil {
			t.Fatal(err)
		}
		if err := command.Start(); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = command.Process.Kill() })
		ready := make([]byte, 1)
		if _, err := stdout.Read(ready); err != nil || ready[0] != 'R' {
			_ = command.Process.Kill()
			_ = command.Wait()
			t.Fatalf("child did not reach pre-dispatch boundary: ready=%q err=%v", ready, err)
		}
		if err := command.Process.Kill(); err != nil {
			_ = command.Wait()
			t.Fatal(err)
		}
		if err := command.Wait(); err == nil {
			t.Fatal("killed process was reported as successful")
		}
		if ctx.Err() != nil {
			t.Fatalf("process fixture timed out: %v", ctx.Err())
		}
		restarted := exec.CommandContext(ctx, executable, "-test.run=^TestR24GoRunnerRestart$/^unknown_provider_has_no_schema_effect$", "-test.count=1")
		if output, err := restarted.CombinedOutput(); err != nil {
			t.Fatalf("fresh process failed rejection boundary: %v\n%s", err, output)
		}
	})
}

func TestR24GoRunnerRestartProcessHelper(t *testing.T) {
	if len(os.Args) < 2 || os.Args[1] != "-test.run=^TestR24GoRunnerRestartProcessHelper$" {
		return
	}
	if NewRunner(Options{Provider: "codex"}) == nil {
		t.Fatal("runner construction failed")
	}
	if _, err := os.Stdout.Write([]byte("R")); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Hour)
	t.Fatal("parent did not interrupt the child")
}
