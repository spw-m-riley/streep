package cmd

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"streep/internal/workflow"
)

const explainUsage = `Explain workflows in human-readable form.

Shows:
  - trigger events
  - job dependency graph
  - required secrets/env/vars
  - external actions used
  - matrix expansion per job
  - warnings (self-hosted, large matrix, missing permissions, deprecated commands)

Usage:
  streep explain [path]

If no path is given, the current directory is used.
`

func executeExplain(args []string, stdout io.Writer, stderr io.Writer) error {
	_ = stderr

	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			_, err := io.WriteString(stdout, explainUsage)
			return err
		}
	}

	dir := "."
	if len(args) == 1 {
		dir = args[0]
	} else if len(args) > 1 {
		return fmt.Errorf("streep explain accepts at most one path argument")
	}

	workflowsDir := filepath.Join(dir, ".github", "workflows")
	refs, err := workflow.ScanDir(workflowsDir)
	if err != nil {
		return err
	}
	overviews, err := workflow.LoadWorkflowOverviews(workflowsDir)
	if err != nil {
		return err
	}

	if len(overviews) == 0 {
		fmt.Fprintf(stdout, "No workflow files found in %s\n", workflowsDir)
		return nil
	}

	fmt.Fprintf(stdout, "Workflow explanation for %s\n\n", dir)

	printStringSection(stdout, "Triggers", refs.Events)

	fmt.Fprintln(stdout, "Job graph")
	for _, ov := range overviews {
		label := ov.File
		if ov.Name != "" {
			label = fmt.Sprintf("%s (%s)", ov.File, ov.Name)
		}
		fmt.Fprintf(stdout, "  %s\n", label)
		fmt.Fprint(stdout, workflow.RenderJobGraph(ov.Jobs))
	}
	fmt.Fprintln(stdout)

	printStringSection(stdout, "Required secrets", refs.Secrets)
	printStringSection(stdout, "Required env vars", refs.Env)
	printStringSection(stdout, "Required repository vars", refs.Vars)
	printStringSection(stdout, "External actions", refs.UsesActions)

	fmt.Fprintln(stdout, "Matrix expansion")
	matrixLines := 0
	for _, ov := range overviews {
		for _, job := range ov.Jobs {
			if len(job.MatrixDimensions) == 0 {
				continue
			}
			matrixLines++
			count := workflow.MatrixCombinations(job.MatrixDimensions)
			fmt.Fprintf(stdout, "  %s/%s: %d combination(s)\n", ov.File, job.ID, count)
			rows := workflow.ExpandMatrix(job.MatrixDimensions, 8)
			for _, row := range rows {
				fmt.Fprintf(stdout, "    - %s\n", formatMatrixRow(row))
			}
			if len(rows) < count {
				fmt.Fprintf(stdout, "    ... and %d more\n", count-len(rows))
			}
		}
	}
	if matrixLines == 0 {
		fmt.Fprintln(stdout, "  - none")
	}
	fmt.Fprintln(stdout)

	warnings := collectExplainWarnings(overviews, refs)
	fmt.Fprintln(stdout, "Warnings")
	if len(warnings) == 0 {
		fmt.Fprintln(stdout, "  - none")
	} else {
		for _, w := range warnings {
			fmt.Fprintf(stdout, "  - %s\n", w)
		}
	}

	return nil
}

func printStringSection(out io.Writer, title string, values []string) {
	fmt.Fprintln(out, title)
	if len(values) == 0 {
		fmt.Fprintln(out, "  - none")
		fmt.Fprintln(out)
		return
	}
	for _, v := range values {
		fmt.Fprintf(out, "  - %s\n", v)
	}
	fmt.Fprintln(out)
}

func formatMatrixRow(row map[string]string) string {
	keys := make([]string, 0, len(row))
	for k := range row {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", k, row[k]))
	}
	return strings.Join(parts, ", ")
}

func collectExplainWarnings(overviews []workflow.WorkflowOverview, refs workflow.References) []string {
	var warnings []string

	if len(refs.SelfHosted) > 0 {
		for _, labels := range refs.SelfHosted {
			warnings = append(warnings, fmt.Sprintf("self-hosted runner label set detected: [%s]", strings.Join(labels, ", ")))
		}
	}

	if refs.MatrixCount > 10 {
		warnings = append(warnings, fmt.Sprintf("large matrix detected (~%d combinations) — local runs may be slow", refs.MatrixCount))
	}

	var missingPermissions []string
	var deprecatedCmd []string
	for _, ov := range overviews {
		if !ov.HasPermissions {
			missingPermissions = append(missingPermissions, ov.File)
		}
		if len(ov.DeprecatedCommands) > 0 {
			deprecatedCmd = append(deprecatedCmd, fmt.Sprintf("%s (%s)", ov.File, strings.Join(ov.DeprecatedCommands, ", ")))
		}
	}
	if len(missingPermissions) > 0 {
		sort.Strings(missingPermissions)
		warnings = append(warnings, fmt.Sprintf("missing top-level permissions block: %s", strings.Join(missingPermissions, ", ")))
	}
	if len(deprecatedCmd) > 0 {
		sort.Strings(deprecatedCmd)
		warnings = append(warnings, fmt.Sprintf("deprecated workflow commands found: %s", strings.Join(deprecatedCmd, "; ")))
	}

	sort.Strings(warnings)
	return warnings
}
