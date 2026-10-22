//go:build windows

package harness

import (
	"os/exec"
	"strconv"
)

// setProcessGroup is a no-op on Windows (process-group semantics differ).
func setProcessGroup(_ *exec.Cmd) {}

// killProcessGroup terminates the child and its descendants. A coding-agent CLI
// commonly starts its real runtime below a .cmd or launcher process; killing
// only the leader leaves that runtime alive after cancellation or timeout.
// taskkill /T is Windows' process-tree equivalent of the POSIX group kill used
// by cli_unix.go. Keep Process.Kill as a best-effort fallback when taskkill is
// unavailable or races a naturally exiting child.
func killProcessGroup(c *exec.Cmd) {
	if c.Process == nil || c.Process.Pid <= 0 {
		return
	}
	_ = exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(c.Process.Pid)).Run()
	_ = c.Process.Kill()
}
