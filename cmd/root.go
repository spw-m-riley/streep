package cmd

import (
	"fmt"
	"io"
	"strings"
)

const rootUsage = `streep helps prepare local projects for act.

Usage:
  streep <command>

Commands:
  new         Create new streep resources
  check       Validate that act credential files are ready
  rehearse    Dry-run workflows locally with act -n
  perform     Run workflows locally with act
  clean       Remove local act runtime files
  doctor      Diagnose local act readiness
  edit        Edit .secrets/.env/.vars/.input files
  explain     Explain workflow intent and structure
  lint        Lint workflow files for common issues
  bundle      Bundle actions for offline use
  hook        Manage git hooks for workflow checks
  diff        Show workflow changes vs a git revision
  fingerprint Generate or compare run fingerprints
  policy      Run workflow security policy checks
  diagnose    Analyze run logs and suggest fixes
  version     Print the version
  update      Check for a newer version
  completion  Generate shell completion scripts
  help        Show help for a command

Run "streep help <command>" for more information about a command.
`

func Execute(args []string, stdout io.Writer, stderr io.Writer) error {
	if stdout == nil || stderr == nil {
		return fmt.Errorf("stdout and stderr are required")
	}

	if len(args) == 0 {
		_, err := io.WriteString(stdout, rootUsage)
		return err
	}

	switch args[0] {
	case "help":
		if len(args) < 2 {
			_, err := io.WriteString(stdout, rootUsage)
			return err
		}
		return Execute([]string{args[1], "--help"}, stdout, stderr)
	case "-h", "--help":
		_, err := io.WriteString(stdout, rootUsage)
		return err
	case "new":
		return executeNew(args[1:], stdout, stderr)
	case "check":
		return executeCheck(args[1:], stdout, stderr)
	case "rehearse":
		return executeRehearse(args[1:], stdout, stderr)
	case "perform":
		return executePerform(args[1:], stdout, stderr)
	case "clean":
		return executeClean(args[1:], stdout, stderr)
	case "doctor":
		return executeDoctor(args[1:], stdout, stderr)
	case "edit":
		return executeEdit(args[1:], stdout, stderr)
	case "explain":
		return executeExplain(args[1:], stdout, stderr)
	case "lint":
		return executeLint(args[1:], stdout, stderr)
	case "bundle":
		return executeBundle(args[1:], stdout, stderr)
	case "hook":
		return executeHook(args[1:], stdout, stderr)
	case "diff":
		return executeDiff(args[1:], stdout, stderr)
	case "fingerprint":
		return executeFingerprint(args[1:], stdout, stderr)
	case "policy":
		return executePolicy(args[1:], stdout, stderr)
	case "diagnose":
		return executeDiagnose(args[1:], stdout, stderr)
	case "version":
		return executeVersion(args[1:], stdout, stderr)
	case "update":
		return executeUpdate(args[1:], stdout, stderr)
	case "completion":
		return executeCompletion(args[1:], stdout, stderr)
	default:
		return fmt.Errorf("unknown command %q\n\n%s", args[0], strings.TrimSpace(rootUsage))
	}
}

func isHelp(arg string) bool {
	return arg == "help" || arg == "-h" || arg == "--help"
}
