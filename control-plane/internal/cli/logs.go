package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Agent-Field/agentfield/control-plane/internal/logger"
	"github.com/Agent-Field/agentfield/control-plane/internal/packages"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	logsFollow bool
	logsTail   int
	logsJSON   bool
)

// NewLogsCommand creates the logs command
func NewLogsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs <agent-node-name>",
		Short: "View logs for a AgentField agent node",
		Long: `Display logs for an installed AgentField agent node package.

Shows the most recent log entries from the agent node's log file.

Examples:
  af logs email-helper
  af logs data-analyzer --follow
  af logs email-helper --json --tail 100`,
		Args: cobra.ExactArgs(1),
		RunE: runLogsCommand,
	}

	cmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "Follow log output")
	cmd.Flags().IntVarP(&logsTail, "tail", "n", 50, "Number of lines to show from the end")
	cmd.Flags().BoolVar(&logsJSON, "json", false, "Emit a machine-readable JSON envelope ({node, log_path, lines}) instead of raw log text")

	return cmd
}

func runLogsCommand(cmd *cobra.Command, args []string) error {
	agentNodeName := args[0]

	if logsJSON {
		return runLogsCommandJSON(agentNodeName)
	}

	logViewer := &LogViewer{
		AgentFieldHome: getAgentFieldHomeDir(),
		Follow:         logsFollow,
		Tail:           logsTail,
	}

	if err := logViewer.ViewLogs(agentNodeName); err != nil {
		logger.Logger.Error().Err(err).Msg("Failed to view logs")
		return fmt.Errorf("failed to view logs: %w", err)
	}

	return nil
}

// runLogsCommandJSON emits the last --tail log lines as a JSON envelope.
// --follow is a live stream and cannot be combined with a JSON snapshot.
func runLogsCommandJSON(agentNodeName string) error {
	if logsFollow {
		return nodeJSONError("invalid_flags", "--follow cannot be combined with --json", "Poll 'af logs <node> --json --tail N' instead of following.")
	}

	logViewer := &LogViewer{
		AgentFieldHome: getAgentFieldHomeDir(),
		Tail:           logsTail,
	}

	logPath, lines, err := logViewer.CollectLogLines(agentNodeName)
	if err != nil {
		return nodeJSONError("logs_failed", err.Error(), "Run 'af list --json' to see installed nodes.")
	}

	return nodeJSONSuccess(map[string]interface{}{
		"node":     agentNodeName,
		"log_path": logPath,
		"lines":    lines,
	})
}

// LogViewer handles viewing agent node logs
type LogViewer struct {
	AgentFieldHome string
	Follow         bool
	Tail           int
}

// ViewLogs displays logs for an agent node
func (lv *LogViewer) ViewLogs(agentNodeName string) error {
	// Load registry to get log file path
	registryPath := filepath.Join(lv.AgentFieldHome, "installed.yaml")
	registry := &packages.InstallationRegistry{
		Installed: make(map[string]packages.InstalledPackage),
	}

	if data, err := os.ReadFile(registryPath); err == nil {
		if err := yaml.Unmarshal(data, registry); err != nil {
			return fmt.Errorf("failed to parse registry: %w", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to read registry: %w", err)
	}

	agentNode, exists := registry.Installed[agentNodeName]
	if !exists {
		return fmt.Errorf("agent node %s not installed", agentNodeName)
	}

	logFile := agentNode.Runtime.LogFile
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		logger.Logger.Info().Msgf("📝 No logs found for %s", agentNodeName)
		logger.Logger.Info().Msg("💡 Logs will appear here when the agent node is running")
		return nil
	}

	logger.Logger.Info().Msgf("📝 Logs for %s:", agentNodeName)
	logger.Logger.Info().Msgf("📁 %s\n", logFile)

	if lv.Follow {
		return lv.followLogs(logFile)
	} else {
		return lv.tailLogs(logFile, lv.Tail)
	}
}

// CollectLogLines returns the node's log file path and its last Tail lines
// for structured (JSON) output. A missing log file yields empty lines rather
// than an error, mirroring the human command's behavior for nodes that have
// not run yet.
func (lv *LogViewer) CollectLogLines(agentNodeName string) (string, []string, error) {
	registryPath := filepath.Join(lv.AgentFieldHome, "installed.yaml")
	registry := &packages.InstallationRegistry{
		Installed: make(map[string]packages.InstalledPackage),
	}

	if data, err := os.ReadFile(registryPath); err == nil {
		if err := yaml.Unmarshal(data, registry); err != nil {
			return "", nil, fmt.Errorf("failed to parse registry: %w", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", nil, fmt.Errorf("failed to read registry: %w", err)
	}

	agentNode, exists := registry.Installed[agentNodeName]
	if !exists {
		return "", nil, fmt.Errorf("agent node %s not installed", agentNodeName)
	}

	logFile := agentNode.Runtime.LogFile
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		return logFile, []string{}, nil
	}

	lines, err := tailFileLines(logFile, lv.Tail)
	if err != nil {
		return logFile, nil, fmt.Errorf("failed to read log file: %w", err)
	}
	return logFile, lines, nil
}

// tailFileLines returns the last n lines of the file at path.
func tailFileLines(path string, n int) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if len(data) == 0 || n == 0 {
		return []string{}, nil
	}
	text := string(data)
	lines := strings.Split(text, "\n")
	if strings.HasSuffix(text, "\n") {
		lines = lines[:len(lines)-1]
	}
	if n > 0 && len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return lines, nil
}

// tailFileData follows tail -n semantics while preserving every byte in the
// selected records, including the final newline and blank records. A positive
// n selects the last n newline-delimited records; n <= 0 selects nothing.
func tailFileData(path string, n int) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil || n <= 0 || len(data) == 0 {
		if err != nil {
			return nil, err
		}
		return []byte{}, nil
	}
	lines := 1
	for _, b := range data {
		if b == '\n' {
			lines++
		}
	}
	if data[len(data)-1] == '\n' {
		lines--
	}
	if lines <= n {
		return data, nil
	}
	toSkip := lines - n
	for i, b := range data {
		if b == '\n' {
			toSkip--
			if toSkip == 0 {
				return data[i+1:], nil
			}
		}
	}
	return []byte{}, nil
}

// tailLogs shows the last N lines of the log file
func (lv *LogViewer) tailLogs(logFile string, lines int) error {
	if runtime.GOOS == "windows" {
		data, err := tailFileData(logFile, lines)
		if err != nil {
			return err
		}
		_, err = os.Stdout.Write(data)
		return err
	}

	cmd := tailCommand(logFile, lines, false)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// followLogs follows the log file in real-time
func (lv *LogViewer) followLogs(logFile string) error {
	if runtime.GOOS == "windows" {
		return followFile(logFile, 10)
	}

	cmd := tailCommand(logFile, 10, true)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// tailCommand builds the Unix command that prints the last n lines of a file.
func tailCommand(logFile string, n int, follow bool) *exec.Cmd {
	program, args := tailCommandArgs(logFile, n, follow)
	return exec.Command(program, args...)
}

func tailCommandArgs(logFile string, n int, follow bool) (string, []string) {
	args := []string{"-n", fmt.Sprintf("%d", n)}
	if follow {
		args = append(args, "-f")
	}
	return "tail", append(args, logFile)
}

// followFile emits existing trailing lines and then streams appended data.
// It is used on Windows so log following does not depend on an installed shell.
func followFile(path string, n int) error {
	return followFileWithContext(context.Background(), path, n, os.Stdout, os.Open, os.Stat, func() { time.Sleep(100 * time.Millisecond) })
}

// followFileWith is injectable so Windows behavior can be tested on any host.
// It emits the snapshot exactly once, then follows appends. A smaller size
// means truncation and resets to offset zero; a changed SameFile identity
// means rotation/replacement and reopens from its beginning.
func followFileWith(path string, n int, out io.Writer, open func(string) (*os.File, error), stat func(string) (os.FileInfo, error), sleep func()) error {
	return followFileWithContext(context.Background(), path, n, out, open, stat, sleep)
}

// followFileWithContext follows a file using the descriptor used for the
// initial snapshot. This avoids losing bytes appended between snapshot and
// reopen. It preserves bytes verbatim, including CRLF and trailing blanks.
func followFileWithContext(ctx context.Context, path string, n int, out io.Writer, open func(string) (*os.File, error), stat func(string) (os.FileInfo, error), sleep func()) error {
	file, err := open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	current, err := file.Stat()
	if err != nil {
		return err
	}
	data, err := tailFileDataFromOpenFile(file, n)
	if err != nil {
		return err
	}
	if _, err := out.Write(data); err != nil {
		return err
	}
	offset, err := file.Seek(0, io.SeekEnd)
	if err != nil { return err }

	buf := make([]byte, 32*1024)
	for {
		select { case <-ctx.Done(): return ctx.Err(); default: }
		count, readErr := file.Read(buf)
		if count > 0 {
			if _, err := out.Write(buf[:count]); err != nil {
				return err
			}
		}
		if readErr != nil && !errors.Is(readErr, io.EOF) {
			return readErr
		}
		latest, statErr := stat(path)
		if statErr == nil && (!os.SameFile(current, latest) || latest.Size() < offset) {
			file.Close()
			file, err = open(path)
			if err != nil {
				if !waitForFollow(ctx, sleep) { return ctx.Err() }
				continue
			}
			current, err = file.Stat()
			if err != nil { return err }
			if _, err = file.Seek(0, io.SeekStart); err != nil {
				return err
			}
			offset = 0
		}
		if count > 0 {
			offset += int64(count)
		}
		if !waitForFollow(ctx, sleep) { return ctx.Err() }
	}
}

func tailFileDataFromOpenFile(file *os.File, n int) ([]byte, error) {
	if n <= 0 { return []byte{}, nil }
	if _, err := file.Seek(0, io.SeekStart); err != nil { return nil, err }
	data, err := io.ReadAll(file)
	if err != nil { return nil, err }
	if len(data) == 0 { return []byte{}, nil }
	lines := 1
	for _, b := range data { if b == '\n' { lines++ } }
	if data[len(data)-1] == '\n' { lines-- }
	if lines <= n { return data, nil }
	toSkip := lines - n
	for i, b := range data {
		if b == '\n' { toSkip--; if toSkip == 0 { return data[i+1:], nil } }
	}
	return []byte{}, nil
}

func waitForFollow(ctx context.Context, sleep func()) bool {
	if ctx.Err() != nil { return false }
	sleep()
	return ctx.Err() == nil
}
