package cmd

import (
	"fmt"
	"io"
	"strings"

	"streep/internal/policy"
)

const policyUsage = `Run security policy checks against workflow files.

Usage:
  streep policy check [path]

Rules:
  - permissions: write-all
  - pull_request_target usage
  - actions not pinned to a full commit SHA

Config:
  .streep/policy.yaml
`

func executePolicy(args []string, stdout io.Writer, stderr io.Writer) error {
	_ = stderr

	if len(args) == 0 || isHelp(args[0]) {
		_, err := io.WriteString(stdout, policyUsage)
		return err
	}

	switch args[0] {
	case "check":
		return executePolicyCheck(args[1:], stdout)
	default:
		return fmt.Errorf("unknown command %q for \"streep policy\"\n\n%s", args[0], strings.TrimSpace(policyUsage))
	}
}

func executePolicyCheck(args []string, stdout io.Writer) error {
	for _, arg := range args {
		if isHelp(arg) {
			_, err := io.WriteString(stdout, policyUsage)
			return err
		}
	}

	dir := "."
	if len(args) == 1 {
		dir = args[0]
	} else if len(args) > 1 {
		return fmt.Errorf("streep policy check accepts at most one path argument")
	}

	findings, err := policy.CheckDir(dir)
	if err != nil {
		return err
	}
	if len(findings) == 0 {
		fmt.Fprintln(stdout, "✔ No policy issues found.")
		return nil
	}

	for _, f := range findings {
		fmt.Fprintf(stdout, "✗ %s: %s (%s)\n", f.File, f.Message, f.Rule)
	}
	fmt.Fprintf(stdout, "\nFound %d policy issue(s).\n", len(findings))
	return nil
}
