package cmd

import (
	"fmt"
	"io"
	"strings"
)

const newUsage = `Create new streep resources.

Usage:
  streep new <command>

Available Commands:
  role        Initialize act-oriented project files

Run "streep new role --help" for usage.
`

func executeNew(args []string, stdout io.Writer, stderr io.Writer) error {
	if len(args) == 0 || isHelp(args[0]) {
		_, err := io.WriteString(stdout, newUsage)
		return err
	}

	switch args[0] {
	case "role":
		return executeNewRole(args[1:], stdout, stderr)
	default:
		return fmt.Errorf("unknown command %q for \"streep new\"\n\n%s", args[0], strings.TrimSpace(newUsage))
	}
}
