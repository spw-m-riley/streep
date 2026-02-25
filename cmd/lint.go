package cmd

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"streep/internal/workflow"
)

const lintUsage = `Lint GitHub Actions workflow files.

Rules:
  - deprecated action versions
  - deprecated ::set-output / ::save-state commands
  - missing top-level permissions block
  - unreachable jobs (needs references missing jobs)
  - undeclared workflow_dispatch inputs referenced in expressions

Usage:
  streep lint [path] [--fix]

Flags:
  --fix    Apply safe fixes for deprecated action versions
`

func executeLint(args []string, stdout io.Writer, stderr io.Writer) error {
	_ = stderr

	fix := false
	positional := make([]string, 0, 1)
	for _, arg := range args {
		switch arg {
		case "-h", "--help", "help":
			_, err := io.WriteString(stdout, lintUsage)
			return err
		case "--fix":
			fix = true
		default:
			if strings.HasPrefix(arg, "-") {
				return fmt.Errorf("unknown flag %q", arg)
			}
			positional = append(positional, arg)
		}
	}
	if len(positional) > 1 {
		return fmt.Errorf("streep lint accepts at most one path argument")
	}

	dir := "."
	if len(positional) == 1 {
		dir = positional[0]
	}

	workflowsDir := filepath.Join(dir, ".github", "workflows")
	overviews, err := workflow.LoadWorkflowOverviews(workflowsDir)
	if err != nil {
		return err
	}
	if len(overviews) == 0 {
		fmt.Fprintf(stdout, "No workflow files found in %s\n", workflowsDir)
		return nil
	}

	result, err := workflow.LintDir(workflowsDir, fix)
	if err != nil {
		return err
	}
	compositeIssues, err := workflow.LintCompositeActionDir(filepath.Join(dir, ".github", "actions"))
	if err != nil {
		return err
	}
	result.Issues = append(result.Issues, compositeIssues...)
	sort.Slice(result.Issues, func(i, j int) bool {
		if result.Issues[i].File != result.Issues[j].File {
			return result.Issues[i].File < result.Issues[j].File
		}
		if result.Issues[i].Rule != result.Issues[j].Rule {
			return result.Issues[i].Rule < result.Issues[j].Rule
		}
		return result.Issues[i].Message < result.Issues[j].Message
	})

	if len(result.Issues) == 0 {
		fmt.Fprintln(stdout, "✔ No lint issues found.")
	} else {
		for _, issue := range result.Issues {
			fmt.Fprintf(stdout, "✗ %s: %s (%s)\n", issue.File, issue.Message, issue.Rule)
		}
		fmt.Fprintf(stdout, "\nFound %d issue(s).\n", len(result.Issues))
	}

	if fix {
		fmt.Fprintf(stdout, "Applied %d action version fix(es) across %d file(s).\n", result.FixedActions, result.ChangedFiles)
	}
	return nil
}
