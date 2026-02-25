package cmd

import (
	"fmt"
	"io"
)

// Version is set at build time via -ldflags "-X streep/cmd.Version=<version>".
var Version = "dev"

func executeVersion(_ []string, stdout io.Writer, _ io.Writer) error {
	_, err := fmt.Fprintln(stdout, Version)
	return err
}
