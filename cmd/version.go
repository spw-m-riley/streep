package cmd

import (
	"fmt"
	"io"
)

// Version, Commit, and Date are set at build time via ldflags.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

func executeVersion(_ []string, stdout io.Writer, _ io.Writer) error {
	_, err := fmt.Fprintf(stdout, "%s (commit %s, built %s)\n", Version, Commit, Date)
	return err
}
