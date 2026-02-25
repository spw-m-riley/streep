package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

const rehearseUsage = `Dry-run your GitHub Actions workflows locally using act.

Runs "act -n [event]" with flags from the local .actrc.
If .actrc is not present, run "streep new role" first.

Usage:
  streep rehearse [event]

Examples:
  streep rehearse            # dry-run the default push event
  streep rehearse pull_request
  streep rehearse workflow_dispatch
`

func executeRehearse(args []string, stdout io.Writer, stderr io.Writer) error {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			_, err := io.WriteString(stdout, rehearseUsage)
			return err
		}
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

	cmdArgs := []string{"-n"}
	cmdArgs = append(cmdArgs, args...)

	fmt.Fprintf(stdout, "Running: act %s\n\n", strings.Join(cmdArgs, " "))

	cmd := exec.Command(actPath, cmdArgs...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	return cmd.Run()
}
