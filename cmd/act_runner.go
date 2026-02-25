package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"streep/internal/diagnose"
)

const actLogPath = ".act/latest.log"

// runAct executes act with the given args, tees output to .act/latest.log,
// and on failure automatically diagnoses the log for known issues.
func runAct(actPath string, cmdArgs []string, stdout io.Writer, stderr io.Writer) error {
	if err := os.MkdirAll(filepath.Dir(actLogPath), 0o755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}
	logFile, err := os.Create(actLogPath)
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}
	defer logFile.Close()

	var logBuf bytes.Buffer
	combined := io.MultiWriter(&logBuf, logFile)

	cmd := exec.Command(actPath, cmdArgs...)
	cmd.Stdout = io.MultiWriter(stdout, combined)
	cmd.Stderr = io.MultiWriter(stderr, combined)

	runErr := cmd.Run()
	if runErr != nil {
		findings := diagnose.AnalyzeLog(logBuf.String())
		if len(findings) > 0 {
			fmt.Fprintln(stderr, "\n── streep diagnose ──────────────────────────────")
			for i, f := range findings {
				fmt.Fprintf(stderr, "%d) [%s] %s\n", i+1, f.Rule, f.Reason)
				fmt.Fprintf(stderr, "   Suggestion: %s\n", f.Suggestion)
			}
			fmt.Fprintln(stderr, "─────────────────────────────────────────────────")
		}
		fmt.Fprintf(stderr, "\nLog saved to %s\n", actLogPath)
	}
	return runErr
}
