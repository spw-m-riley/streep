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
  verify      Verify .act/bundle.lock against current workflow refs

Run "streep bundle actions --help" for usage.
`

const bundleActionsUsage = `Download and lock remote workflow actions for offline use.

Scans .github/workflows for remote uses: references (owner/repo@ref),
resolves refs to commit SHAs, downloads archives into .act/bundle/, and writes
.act/bundle.lock.

Usage:
  streep bundle actions [path]
`

const bundleVerifyUsage = `Verify .act/bundle.lock against current workflow references.

Checks for:
  - workflow refs missing from lock
  - lock refs no longer used by workflows
  - refs whose resolved SHA drifted from locked SHA

Usage:
  streep bundle verify [path]
`

func executeBundle(args []string, stdout io.Writer, stderr io.Writer) error {
	if len(args) == 0 || isHelp(args[0]) {
		_, err := io.WriteString(stdout, bundleUsage)
		return err
	}

	switch args[0] {
	case "actions":
		return executeBundleActions(args[1:], stdout, stderr)
	case "verify":
		return executeBundleVerify(args[1:], stdout, stderr)
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

func executeBundleVerify(args []string, stdout io.Writer, stderr io.Writer) error {
	_ = stderr
	for _, arg := range args {
		if isHelp(arg) {
			_, err := io.WriteString(stdout, bundleVerifyUsage)
			return err
		}
	}

	dir := "."
	if len(args) == 1 {
		dir = args[0]
	} else if len(args) > 1 {
		return fmt.Errorf("streep bundle verify accepts at most one path argument")
	}

	result, err := bundle.VerifyLock(bundle.Options{RepoDir: dir})
	if err != nil {
		return err
	}
	if result.IsClean() {
		fmt.Fprintln(stdout, "✔ bundle.lock matches current workflow action refs.")
		return nil
	}

	for _, ref := range result.Missing {
		fmt.Fprintf(stdout, "✗ missing in lock: %s\n", ref)
	}
	for _, ref := range result.Extra {
		fmt.Fprintf(stdout, "✗ extra in lock: %s\n", ref)
	}
	for _, drift := range result.Stale {
		fmt.Fprintf(stdout, "✗ stale lock entry: %s (locked %s, resolved %s)\n", drift.Ref, drift.LockedSHA, drift.ResolvedSHA)
	}
	return fmt.Errorf("bundle lock verification failed")
}
