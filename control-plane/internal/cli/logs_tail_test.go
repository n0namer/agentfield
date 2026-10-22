package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestTailFileDataPreservesTailBytes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "native-windows.log")
	if err := os.WriteFile(path, []byte("one\n\ntwo\nthree"), 0o600); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		n    int
		want string
	}{
		{n: 0, want: ""},
		{n: 1, want: "three"},
		{n: 2, want: "two\nthree"},
		{n: 3, want: "\ntwo\nthree"},
		{n: 10, want: "one\n\ntwo\nthree"},
	}
	for _, tc := range cases {
		got, err := tailFileData(path, tc.n)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != tc.want {
			t.Errorf("tailFileData(%d) = %q, want %q", tc.n, got, tc.want)
		}
	}
}

func TestTailFileDataPreservesTrailingNewline(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.log")
	if err := os.WriteFile(path, []byte("one\ntwo\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := tailFileData(path, 1)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "two\n" {
		t.Fatalf("got %q, want %q", got, "two\n")
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("stdout closed") }

func TestFollowFilePropagatesInitialOutputError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.log")
	if err := os.WriteFile(path, []byte("line\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := followFileWith(path, 1, failingWriter{}, os.Open, os.Stat, func() {})
	if err == nil || !strings.Contains(err.Error(), "stdout closed") {
		t.Fatalf("error = %v, want stdout error", err)
	}
}

func TestFollowFileContextCanBeCancelled(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.log")
	if err := os.WriteFile(path, []byte("line\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var got strings.Builder
	err := followFileWithContext(ctx, path, 1, &got, os.Open, os.Stat, func() {})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if got.String() != "line\n" {
		t.Fatalf("output = %q, want %q", got.String(), "line\\n")
	}
}

// Contract: Unix keeps using tail(1). Windows uses the Go-native reader in
// tailLogs/followLogs and must not construct a shell command.
func TestTailCommandArgs(t *testing.T) {
	cases := []struct {
		name     string
		goos     string
		file     string
		n        int
		follow   bool
		wantProg string
		wantArgs []string
	}{
		{
			name: "unix no follow", goos: "linux",
			file: "/var/log/agent.log", n: 7,
			wantProg: "tail",
			wantArgs: []string{"-n", "7", "/var/log/agent.log"},
		},
		{
			name: "unix follow", goos: "darwin",
			file: "/var/log/agent.log", n: 10, follow: true,
			wantProg: "tail",
			wantArgs: []string{"-n", "10", "-f", "/var/log/agent.log"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			prog, args := tailCommandArgs(tc.file, tc.n, tc.follow)
			if prog != tc.wantProg {
				t.Fatalf("program = %q; want %q", prog, tc.wantProg)
			}
			if !reflect.DeepEqual(args, tc.wantArgs) {
				t.Fatalf("args = %q; want %q", args, tc.wantArgs)
			}
		})
	}
}

// Contract: tailCommand builds an exec.Cmd from the host platform's
// tailCommandArgs — the first Args entry is the program itself.
func TestTailCommandUsesHostGOOS(t *testing.T) {
	cmd := tailCommand("/var/log/agent.log", 5, false)
	prog, args := tailCommandArgs("/var/log/agent.log", 5, false)
	want := append([]string{prog}, args...)
	if !reflect.DeepEqual(cmd.Args, want) {
		t.Fatalf("cmd.Args = %q; want %q", cmd.Args, want)
	}
}
