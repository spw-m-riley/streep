package cmd

import (
	"fmt"
	"io"
	"strings"

	"streep/internal/hook"
)

const hookUsage = `Manage git hooks for workflow safety checks.

Usage:
  streep hook <install|uninstall> [path]

Examples:
  streep hook install
  streep hook uninstall
`

func executeHook(args []string, stdout io.Writer, stderr io.Writer) error {
	_ = stderr

	if len(args) == 0 || isHelp(args[0]) {
		_, err := io.WriteString(stdout, hookUsage)
		return err
	}
	if len(args) > 2 {
		return fmt.Errorf("streep hook accepts a subcommand and optional path")
	}

	dir := "."
	if len(args) == 2 {
		dir = args[1]
	}

	switch args[0] {
	case "install":
		n, err := hook.Install(dir)
		if err != nil {
			return err
		}
		fmt.Fprintf(stdout, "Installed %d streep-managed hook(s).\n", n)
		return nil
	case "uninstall":
		n, err := hook.Uninstall(dir)
		if err != nil {
			return err
		}
		fmt.Fprintf(stdout, "Removed %d streep-managed hook(s).\n", n)
		return nil
	default:
		return fmt.Errorf("unknown command %q for \"streep hook\"\n\n%s", args[0], strings.TrimSpace(hookUsage))
	}
}
