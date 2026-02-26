package cmd

import (
	"fmt"
	"io"
	"strings"

	"streep/internal/scaffold"
)

const newRoleUsage = `Initialize act-oriented scaffold files in a directory.

Usage:
  streep new role [path] [--force]

Flags:
  --force     Overwrite existing scaffold files
`

func executeNewRole(args []string, stdout io.Writer, stderr io.Writer) error {
	_ = stderr

	force := false
	positionalArgs := make([]string, 0, 1)

	for _, arg := range args {
		switch arg {
		case "-h", "--help", "help":
			_, err := io.WriteString(stdout, newRoleUsage)
			return err
		case "--force":
			force = true
		default:
			if strings.HasPrefix(arg, "-") {
				return fmt.Errorf("unknown flag %q", arg)
			}
			positionalArgs = append(positionalArgs, arg)
		}
	}

	if len(positionalArgs) > 1 {
		return fmt.Errorf("streep new role accepts at most one path argument")
	}

	dir := "."
	if len(positionalArgs) == 1 {
		dir = positionalArgs[0]
	}
	cfg, err := loadStreepConfig(dir)
	if err != nil {
		return err
	}

	return scaffold.NewRole(scaffold.RoleOptions{
		Dir:            dir,
		Force:          force,
		Out:            stdout,
		RunnerImageMap: cfg.RunnerImages,
	})
}
