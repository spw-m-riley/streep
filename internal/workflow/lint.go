package workflow

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// LintIssue is a single workflow lint finding.
type LintIssue struct {
	File    string
	Rule    string
	Message string
}

// LintResult contains lint findings and optional auto-fix metadata.
type LintResult struct {
	Issues       []LintIssue
	FixedActions int
	ChangedFiles int
}

// LintDir checks workflow files in dir and optionally applies action-version fixes.
func LintDir(dir string, fix bool) (LintResult, error) {
	files, err := listWorkflowFiles(dir)
	if err != nil {
		return LintResult{}, err
	}

	result := LintResult{}
	for _, f := range files {
		changed, fixed, issues, err := lintWorkflowFile(f, fix)
		if err != nil {
			return LintResult{}, err
		}
		result.Issues = append(result.Issues, issues...)
		result.FixedActions += fixed
		if changed {
			result.ChangedFiles++
		}
	}

	sort.Slice(result.Issues, func(i, j int) bool {
		if result.Issues[i].File != result.Issues[j].File {
			return result.Issues[i].File < result.Issues[j].File
		}
		if result.Issues[i].Rule != result.Issues[j].Rule {
			return result.Issues[i].Rule < result.Issues[j].Rule
		}
		return result.Issues[i].Message < result.Issues[j].Message
	})
	return result, nil
}

func lintWorkflowFile(path string, fix bool) (changed bool, fixedActions int, issues []LintIssue, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, 0, nil, err
	}

	var root yaml.Node
	dec := yaml.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&root); err != nil {
		return false, 0, nil, fmt.Errorf("failed to parse %s: %w", filepath.Base(path), err)
	}
	if root.Kind != yaml.DocumentNode || len(root.Content) == 0 {
		return false, 0, nil, nil
	}
	doc := root.Content[0]
	file := filepath.Base(path)

	if !mappingHasKey(doc, "permissions") {
		issues = append(issues, LintIssue{
			File:    file,
			Rule:    "missing-permissions",
			Message: "missing top-level permissions block",
		})
	}

	declaredInputs := make(map[string]struct{})
	for _, key := range collectWorkflowDispatchInputs(doc) {
		declaredInputs[key] = struct{}{}
	}
	for _, ref := range collectReferencedInputs(doc) {
		if _, ok := declaredInputs[ref]; !ok {
			issues = append(issues, LintIssue{
				File:    file,
				Rule:    "undeclared-input",
				Message: fmt.Sprintf("github.event.inputs.%s is referenced but not declared under workflow_dispatch.inputs", ref),
			})
		}
	}

	jobIDs := map[string]struct{}{}
	visitMappingValue(doc, "jobs", func(jobsNode *yaml.Node) {
		for i := 0; i+1 < len(jobsNode.Content); i += 2 {
			jobID := jobsNode.Content[i].Value
			jobIDs[jobID] = struct{}{}
		}
	})

	visitMappingValue(doc, "jobs", func(jobsNode *yaml.Node) {
		for i := 0; i+1 < len(jobsNode.Content); i += 2 {
			jobID := jobsNode.Content[i].Value
			jobNode := jobsNode.Content[i+1]

			for _, need := range collectNeeds(jobNode) {
				if _, ok := jobIDs[need]; !ok {
					issues = append(issues, LintIssue{
						File:    file,
						Rule:    "unreachable-job",
						Message: fmt.Sprintf("job %q needs %q, but %q does not exist", jobID, need, need),
					})
				}
			}

			visitMappingValue(jobNode, "steps", func(stepsNode *yaml.Node) {
				for _, step := range stepsNode.Content {
					visitMappingValue(step, "uses", func(useNode *yaml.Node) {
						if useNode.Kind != yaml.ScalarNode || useNode.Value == "" {
							return
						}
						replacement, deprecated := DeprecatedActionVersions[useNode.Value]
						if !deprecated {
							return
						}
						issues = append(issues, LintIssue{
							File:    file,
							Rule:    "deprecated-action-version",
							Message: fmt.Sprintf("replace %q with %q", useNode.Value, replacement),
						})
						if fix {
							useNode.Value = replacement
							changed = true
							fixedActions++
						}
					})

					visitMappingValue(step, "run", func(runNode *yaml.Node) {
						if runNode.Kind != yaml.ScalarNode {
							return
						}
						if strings.Contains(runNode.Value, "::set-output") {
							issues = append(issues, LintIssue{
								File:    file,
								Rule:    "deprecated-set-output",
								Message: "uses deprecated ::set-output command",
							})
						}
						if strings.Contains(runNode.Value, "::save-state") {
							issues = append(issues, LintIssue{
								File:    file,
								Rule:    "deprecated-save-state",
								Message: "uses deprecated ::save-state command",
							})
						}
					})
				}
			})
		}
	})

	if fix && changed {
		var out bytes.Buffer
		enc := yaml.NewEncoder(&out)
		enc.SetIndent(2)
		if err := enc.Encode(&root); err != nil {
			enc.Close()
			return false, 0, nil, err
		}
		if err := enc.Close(); err != nil {
			return false, 0, nil, err
		}
		if err := os.WriteFile(path, out.Bytes(), 0o644); err != nil {
			return false, 0, nil, err
		}
	}

	return changed, fixedActions, issues, nil
}
