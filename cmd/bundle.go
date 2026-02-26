package cmd

import (
	"fmt"
	"io"
	"strings"

	"streep/internal/bundle"
)

const bundleUsage = `Bundle dependencies for offline act execution.

Usage:
  streep bundle <command>

Available Commands:
  actions     Download and lock workflow uses: action dependencies

Run "streep bundle actions --help" for usage.
`

const bundleActionsUsage = `Download and lock remote workflow actions for offline use.

Scans .github/workflows for remote uses: references (owner/repo@ref),
resolves refs to commit SHAs, downloads archives into .act/bundle/, and writes
.act/bundle.lock.

Usage:
  streep bundle actions [path]
`

func executeBundle(args []string, stdout io.Writer, stderr io.Writer) error {
	if len(args) == 0 || isHelp(args[0]) {
		_, err := io.WriteString(stdout, bundleUsage)
		return err
	}

	switch args[0] {
	case "actions":
		return executeBundleActions(args[1:], stdout, stderr)
	default:
		return fmt.Errorf("unknown command %q for \"streep bundle\"\n\n%s", args[0], strings.TrimSpace(bundleUsage))
	}
}

func executeBundleActions(args []string, stdout io.Writer, stderr io.Writer) error {
	_ = stderr
	for _, arg := range args {
		if isHelp(arg) {
			_, err := io.WriteString(stdout, bundleActionsUsage)
			return err
		}
	}

	dir := "."
	if len(args) == 1 {
		dir = args[0]
	} else if len(args) > 1 {
		return fmt.Errorf("streep bundle actions accepts at most one path argument")
	}

	result, err := bundle.BundleActions(bundle.Options{RepoDir: dir, Progress: stdout})
	if err != nil {
		return err
	}

	if len(result.Entries) == 0 {
		fmt.Fprintln(stdout, "No remote workflow actions found to bundle.")
		fmt.Fprintf(stdout, "Wrote lock file: %s\n", result.LockPath)
		return nil
	}

	fmt.Fprintf(stdout, "Bundled %d action(s):\n", len(result.Entries))
	for _, entry := range result.Entries {
		fmt.Fprintf(stdout, "  - %s -> %s\n", entry.Ref, entry.Path)
	}
	fmt.Fprintf(stdout, "Wrote lock file: %s\n", result.LockPath)
	return nil
}
