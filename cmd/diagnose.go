package cmd

import (
	"fmt"
	"io"
	"os"

	"streep/internal/diagnose"
)

const diagnoseUsage = `Analyze an act run log and suggest likely fixes.

Usage:
  streep diagnose <run-log>

Example:
  streep diagnose .act/latest.log
`

func executeDiagnose(args []string, stdout io.Writer, stderr io.Writer) error {
	_ = stderr

	if len(args) == 0 || isHelp(args[0]) {
		_, err := io.WriteString(stdout, diagnoseUsage)
		return err
	}
	if len(args) != 1 {
		return fmt.Errorf("usage: streep diagnose <run-log>")
	}

	logPath := args[0]
	content, err := os.ReadFile(logPath)
	if err != nil {
		return fmt.Errorf("failed to read log %s: %w", logPath, err)
	}

	findings := diagnose.AnalyzeLog(string(content))
	if len(findings) == 0 {
		fmt.Fprintln(stdout, "No known failure patterns matched.")
		fmt.Fprintln(stdout, "Try: streep doctor, streep lint, and inspect the failing step logs directly.")
		return nil
	}

	fmt.Fprintln(stdout, "Likely root causes:")
	for i, f := range findings {
		fmt.Fprintf(stdout, "%d) [%s] %s\n", i+1, f.Rule, f.Reason)
		fmt.Fprintf(stdout, "   Suggestion: %s\n", f.Suggestion)
	}
	return nil
}
