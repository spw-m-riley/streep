package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"streep/internal/workflow"
)

const diffUsage = `Compare local workflow state with another git revision.

Reports workflow-level deltas for:
  - jobs
  - trigger events
  - required secrets

Usage:
  streep diff [branch] [path]

Defaults:
  branch = HEAD~1
  path   = current directory
`

func executeDiff(args []string, stdout io.Writer, stderr io.Writer) error {
	_ = stderr

	for _, arg := range args {
		if isHelp(arg) {
			_, err := io.WriteString(stdout, diffUsage)
			return err
		}
	}
	if len(args) > 2 {
		return fmt.Errorf("streep diff accepts at most two arguments: [branch] [path]")
	}

	branch := "HEAD~1"
	dir := "."
	if len(args) >= 1 {
		branch = args[0]
	}
	if len(args) == 2 {
		dir = args[1]
	}

	if err := ensureGitRepo(dir); err != nil {
		return err
	}

	localFiles, err := listLocalWorkflowFiles(dir)
	if err != nil {
		return err
	}
	branchFiles, err := listBranchWorkflowFiles(dir, branch)
	if err != nil {
		return err
	}

	allFiles := unionSorted(localFiles, branchFiles)
	fmt.Fprintf(stdout, "Workflow delta vs %s\n", branch)

	changes := 0
	for _, rel := range allFiles {
		localData, localOK := readLocalWorkflow(dir, rel)
		branchData, branchOK, err := readBranchWorkflow(dir, branch, rel)
		if err != nil {
			return err
		}

		switch {
		case localOK && !branchOK:
			changes++
			fmt.Fprintf(stdout, "- added workflow: %s\n", rel)
		case !localOK && branchOK:
			changes++
			fmt.Fprintf(stdout, "- removed workflow: %s\n", rel)
		case localOK && branchOK:
			localDetails, err := workflow.ParseDetails(localData)
			if err != nil {
				return fmt.Errorf("parse local %s: %w", rel, err)
			}
			branchDetails, err := workflow.ParseDetails(branchData)
			if err != nil {
				return fmt.Errorf("parse %s at %s: %w", rel, branch, err)
			}

			anyChange := false
			addedJobs, removedJobs := setDiff(localDetails.Jobs, branchDetails.Jobs)
			addedEvents, removedEvents := setDiff(localDetails.Events, branchDetails.Events)
			addedSecrets, removedSecrets := setDiff(localDetails.Secrets, branchDetails.Secrets)
			if len(addedJobs)+len(removedJobs)+len(addedEvents)+len(removedEvents)+len(addedSecrets)+len(removedSecrets) == 0 {
				continue
			}
			changes++
			fmt.Fprintf(stdout, "- modified workflow: %s\n", rel)
			anyChange = printDiffLine(stdout, "jobs", addedJobs, removedJobs) || anyChange
			anyChange = printDiffLine(stdout, "events", addedEvents, removedEvents) || anyChange
			anyChange = printDiffLine(stdout, "secrets", addedSecrets, removedSecrets) || anyChange
			if !anyChange {
				fmt.Fprintln(stdout, "  (content changed, but no tracked deltas)")
			}
		}
	}

	if changes == 0 {
		fmt.Fprintln(stdout, "No workflow deltas detected.")
	}
	return nil
}

func ensureGitRepo(dir string) error {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--is-inside-work-tree")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("not a git repository: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

func listLocalWorkflowFiles(dir string) ([]string, error) {
	patterns := []string{
		filepath.Join(dir, ".github", "workflows", "*.yml"),
		filepath.Join(dir, ".github", "workflows", "*.yaml"),
	}
	var files []string
	for _, p := range patterns {
		matches, err := filepath.Glob(p)
		if err != nil {
			return nil, err
		}
		for _, m := range matches {
			rel, err := filepath.Rel(dir, m)
			if err != nil {
				return nil, err
			}
			files = append(files, filepath.ToSlash(rel))
		}
	}
	sort.Strings(files)
	return files, nil
}

func listBranchWorkflowFiles(dir, branch string) ([]string, error) {
	cmd := exec.Command("git", "-C", dir, "ls-tree", "-r", "--name-only", branch, "--", ".github/workflows")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to list workflows at %s: %s", branch, strings.TrimSpace(string(out)))
	}

	var files []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasSuffix(line, ".yml") || strings.HasSuffix(line, ".yaml") {
			files = append(files, line)
		}
	}
	sort.Strings(files)
	return files, nil
}

func unionSorted(a, b []string) []string {
	set := map[string]struct{}{}
	for _, v := range a {
		set[v] = struct{}{}
	}
	for _, v := range b {
		set[v] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for v := range set {
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

func readLocalWorkflow(dir, rel string) ([]byte, bool) {
	p := filepath.Join(dir, filepath.FromSlash(rel))
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, false
	}
	return data, true
}

func readBranchWorkflow(dir, branch, rel string) ([]byte, bool, error) {
	spec := branch + ":" + rel
	cmd := exec.Command("git", "-C", dir, "show", spec)
	out, err := cmd.CombinedOutput()
	if err != nil {
		// Missing file in revision is not an error for our purposes.
		text := string(out)
		if strings.Contains(text, "exists on disk, but not in") || strings.Contains(text, "does not exist") || strings.Contains(text, "Path '"+rel+"' does not exist") {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("failed reading %s: %s", spec, strings.TrimSpace(text))
	}
	return out, true, nil
}

func setDiff(current []string, baseline []string) (added []string, removed []string) {
	baseSet := map[string]struct{}{}
	curSet := map[string]struct{}{}
	for _, v := range baseline {
		baseSet[v] = struct{}{}
	}
	for _, v := range current {
		curSet[v] = struct{}{}
		if _, ok := baseSet[v]; !ok {
			added = append(added, v)
		}
	}
	for _, v := range baseline {
		if _, ok := curSet[v]; !ok {
			removed = append(removed, v)
		}
	}
	sort.Strings(added)
	sort.Strings(removed)
	return added, removed
}

func printDiffLine(out io.Writer, label string, added []string, removed []string) bool {
	changed := false
	if len(added) > 0 {
		changed = true
		fmt.Fprintf(out, "  %s added: %s\n", label, strings.Join(added, ", "))
	}
	if len(removed) > 0 {
		changed = true
		fmt.Fprintf(out, "  %s removed: %s\n", label, strings.Join(removed, ", "))
	}
	return changed
}
