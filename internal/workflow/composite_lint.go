package workflow

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

var bashismPattern = regexp.MustCompile(`\[\[|\$\(|\bsource\s+|set -euo pipefail`)

// LintCompositeActionDir scans .github/actions for composite action shell-safety issues.
func LintCompositeActionDir(actionsDir string) ([]LintIssue, error) {
	if _, err := os.Stat(actionsDir); os.IsNotExist(err) {
		return nil, nil
	}

	var files []string
	err := filepath.WalkDir(actionsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		name := strings.ToLower(d.Name())
		if name == "action.yml" || name == "action.yaml" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)

	var issues []LintIssue
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}

		var root yaml.Node
		dec := yaml.NewDecoder(bytes.NewReader(data))
		if err := dec.Decode(&root); err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", file, err)
		}
		if root.Kind != yaml.DocumentNode || len(root.Content) == 0 {
			continue
		}
		doc := root.Content[0]

		var runsNode *yaml.Node
		visitMappingValue(doc, "runs", func(v *yaml.Node) { runsNode = v })
		if runsNode == nil || runsNode.Kind != yaml.MappingNode {
			continue
		}

		using := mappingScalar(runsNode, "using")
		if using != "composite" {
			continue
		}

		visitMappingValue(runsNode, "steps", func(stepsNode *yaml.Node) {
			if stepsNode.Kind != yaml.SequenceNode {
				return
			}
			for idx, step := range stepsNode.Content {
				run := mappingScalar(step, "run")
				if run == "" {
					continue
				}
				shell := mappingScalar(step, "shell")
				if shell == "bash" {
					continue
				}
				if bashismPattern.MatchString(run) {
					issues = append(issues, LintIssue{
						File:    filepath.ToSlash(file),
						Rule:    "composite-shell-safety",
						Message: fmt.Sprintf("step %d uses bash-style syntax but shell is %q (set shell: bash)", idx+1, shell),
					})
				}
			}
		})
	}

	sort.Slice(issues, func(i, j int) bool {
		if issues[i].File != issues[j].File {
			return issues[i].File < issues[j].File
		}
		return issues[i].Message < issues[j].Message
	})
	return issues, nil
}
