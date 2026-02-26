package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"streep/internal/fingerprint"
)

const performUsage = `Run your GitHub Actions workflows locally using act.

Runs "act [event]" with flags from the local .actrc.
If .actrc is not present, run "streep new role" first.

Usage:
  streep perform [event] [--job JOB] [--workflow FILE] [-- <act args>]

Examples:
  streep perform
  streep perform pull_request
  streep perform pull_request --job test
  streep perform push --workflow .github/workflows/ci.yml
  streep perform -- --verbose
`

func executePerform(args []string, stdout io.Writer, stderr io.Writer) error {
	primaryArgs, passthroughArgs := splitPassthroughArgs(args)
	event := ""
	job := ""
	workflowFile := ""
	positional := make([]string, 0, 1)

	for i := 0; i < len(primaryArgs); i++ {
		arg := primaryArgs[i]
		switch arg {
		case "-h", "--help", "help":
			_, err := io.WriteString(stdout, performUsage)
			return err
		case "--job", "-j":
			if i+1 >= len(primaryArgs) {
				return fmt.Errorf("missing value for %s", arg)
			}
			i++
			job = primaryArgs[i]
		case "--workflow", "-W":
			if i+1 >= len(primaryArgs) {
				return fmt.Errorf("missing value for %s", arg)
			}
			i++
			workflowFile = primaryArgs[i]
		default:
			if strings.HasPrefix(arg, "-") {
				return fmt.Errorf("unknown flag %q", arg)
			}
			positional = append(positional, arg)
		}
	}

	if len(positional) > 1 {
		return fmt.Errorf("streep perform accepts at most one event argument")
	}
	if len(positional) == 1 {
		event = positional[0]
	}
	cfg, err := loadStreepConfig(".")
	if err != nil {
		return err
	}
	if event == "" && cfg.Defaults.Event != "" {
		event = cfg.Defaults.Event
	}
	if job == "" && cfg.Defaults.Job != "" {
		job = cfg.Defaults.Job
	}
	if workflowFile == "" && cfg.Defaults.Workflow != "" {
		workflowFile = cfg.Defaults.Workflow
	}

	if _, err := os.Stat(".actrc"); os.IsNotExist(err) {
		fmt.Fprintln(stdout, "⚠  .actrc not found.")
		fmt.Fprintln(stdout, "   Run 'streep new role' first to scaffold your project for act.")
		return nil
	}

	actPath, err := exec.LookPath("act")
	if err != nil {
		return fmt.Errorf("act not found in PATH — install it from https://github.com/nektos/act")
	}

	cmdArgs := make([]string, 0, 8+len(passthroughArgs))
	if job != "" {
		cmdArgs = append(cmdArgs, "-j", job)
	}
	if workflowFile != "" {
		cmdArgs = append(cmdArgs, "-W", workflowFile)
	}
	if event != "" {
		cmdArgs = append(cmdArgs, event)
	}

	payloadEvent := event
	if payloadEvent == "" {
		payloadEvent = "push"
	}
	payloadPath := filepath.Join(".act", "events", payloadEvent+".json")
	if _, err := os.Stat(payloadPath); err == nil {
		cmdArgs = append(cmdArgs, "-e", payloadPath)
	}
	if len(passthroughArgs) > 0 {
		cmdArgs = append(cmdArgs, passthroughArgs...)
	}

	fmt.Fprintf(stdout, "Performing: act %s\n\n", strings.Join(cmdArgs, " "))

	if err := runAct(actPath, cmdArgs, stdout, stderr); err != nil {
		return err
	}

	if fp, path, err := fingerprint.WriteCurrent("."); err == nil {
		fmt.Fprintf(stdout, "\nFingerprint: %s\nWrote %s\n", fp.Digest, path)
	}
	return nil
}
