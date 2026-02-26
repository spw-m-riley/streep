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

// WorkflowOverview is a structured summary of a single workflow file.
type WorkflowOverview struct {
	File               string
	Name               string
	Events             []string
	HasPermissions     bool
	Jobs               []JobOverview
	UsesActions        []string
	DispatchInputs     []string
	ReferencedInputs   []string
	DeprecatedCommands []string
}

// JobOverview is a summary of a workflow job.
type JobOverview struct {
	ID               string
	Needs            []string
	MatrixDimensions map[string][]string
}

var reEventInputRef = regexp.MustCompile(`github\.event\.inputs\.([A-Za-z_][A-Za-z0-9_-]*)`)

// LoadWorkflowOverviews parses all workflow YAML files in dir and returns summaries.
func LoadWorkflowOverviews(dir string) ([]WorkflowOverview, error) {
	files, err := listWorkflowFiles(dir)
	if err != nil {
		return nil, err
	}

	overviews := make([]WorkflowOverview, 0, len(files))
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}

		var root yaml.Node
		dec := yaml.NewDecoder(bytes.NewReader(data))
		if err := dec.Decode(&root); err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", filepath.Base(f), err)
		}

		ov := WorkflowOverview{
			File: filepath.Base(f),
		}

		if root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
			doc := root.Content[0]
			ov.Name = mappingScalar(doc, "name")
			ov.HasPermissions = mappingHasKey(doc, "permissions")
			ov.Events = collectWorkflowEvents(doc)
			ov.DispatchInputs = collectWorkflowDispatchInputs(doc)
			ov.Jobs = collectJobs(doc)
			ov.UsesActions = collectUsesActions(doc)
			ov.ReferencedInputs = collectReferencedInputs(doc)
			ov.DeprecatedCommands = collectDeprecatedCommands(doc)
		}

		overviews = append(overviews, ov)
	}

	return overviews, nil
}

// RenderJobGraph renders a compact dependency view for workflow jobs.
func RenderJobGraph(jobs []JobOverview) string {
	if len(jobs) == 0 {
		return "  (no jobs)\n"
	}
	var b strings.Builder
	for _, job := range jobs {
		if len(job.Needs) == 0 {
			fmt.Fprintf(&b, "  - %s\n", job.ID)
			continue
		}
		fmt.Fprintf(&b, "  - %s <- [%s]\n", job.ID, strings.Join(job.Needs, ", "))
	}
	return b.String()
}

// MatrixCombinations returns cartesian product count for matrix dimensions.
func MatrixCombinations(dimensions map[string][]string) int {
	if len(dimensions) == 0 {
		return 0
	}
	total := 1
	for _, values := range dimensions {
		if len(values) == 0 {
			continue
		}
		total *= len(values)
	}
	return total
}

// ExpandMatrix returns cartesian rows for matrix dimensions.
// maxRows <= 0 means no cap.
func ExpandMatrix(dimensions map[string][]string, maxRows int) []map[string]string {
	if len(dimensions) == 0 {
		return nil
	}

	keys := make([]string, 0, len(dimensions))
	for k := range dimensions {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	rows := make([]map[string]string, 0)
	var walk func(idx int, row map[string]string)
	walk = func(idx int, row map[string]string) {
		if maxRows > 0 && len(rows) >= maxRows {
			return
		}
		if idx == len(keys) {
			cp := make(map[string]string, len(row))
			for k, v := range row {
				cp[k] = v
			}
			rows = append(rows, cp)
			return
		}
		key := keys[idx]
		values := dimensions[key]
		for _, v := range values {
			row[key] = v
			walk(idx+1, row)
			if maxRows > 0 && len(rows) >= maxRows {
				return
			}
		}
		delete(row, key)
	}

	walk(0, map[string]string{})
	return rows
}

func collectJobs(doc *yaml.Node) []JobOverview {
	var jobs []JobOverview
	visitMappingValue(doc, "jobs", func(jobsNode *yaml.Node) {
		if jobsNode.Kind != yaml.MappingNode {
			return
		}
		for i := 0; i+1 < len(jobsNode.Content); i += 2 {
			id := jobsNode.Content[i].Value
			jobNode := jobsNode.Content[i+1]
			job := JobOverview{
				ID:               id,
				Needs:            collectNeeds(jobNode),
				MatrixDimensions: collectMatrixDimensions(jobNode),
			}
			jobs = append(jobs, job)
		}
	})
	sort.Slice(jobs, func(i, j int) bool { return jobs[i].ID < jobs[j].ID })
	return jobs
}

func collectNeeds(jobNode *yaml.Node) []string {
	var needs []string
	visitMappingValue(jobNode, "needs", func(needsNode *yaml.Node) {
		switch needsNode.Kind {
		case yaml.ScalarNode:
			if needsNode.Value != "" {
				needs = append(needs, needsNode.Value)
			}
		case yaml.SequenceNode:
			for _, n := range needsNode.Content {
				if n.Kind == yaml.ScalarNode && n.Value != "" {
					needs = append(needs, n.Value)
				}
			}
		}
	})
	sort.Strings(needs)
	return needs
}

func collectMatrixDimensions(jobNode *yaml.Node) map[string][]string {
	dimensions := map[string][]string{}
	visitMappingValue(jobNode, "strategy", func(strategyNode *yaml.Node) {
		visitMappingValue(strategyNode, "matrix", func(matrixNode *yaml.Node) {
			if matrixNode.Kind != yaml.MappingNode {
				return
			}
			for i := 0; i+1 < len(matrixNode.Content); i += 2 {
				k := matrixNode.Content[i].Value
				v := matrixNode.Content[i+1]
				if k == "include" || k == "exclude" {
					continue
				}
				if v.Kind != yaml.SequenceNode {
					continue
				}
				values := make([]string, 0, len(v.Content))
				for _, item := range v.Content {
					values = append(values, item.Value)
				}
				if len(values) > 0 {
					dimensions[k] = values
				}
			}
		})
	})
	return dimensions
}

func collectWorkflowEvents(doc *yaml.Node) []string {
	set := map[string]struct{}{}
	visitMappingValue(doc, "on", func(onNode *yaml.Node) {
		switch onNode.Kind {
		case yaml.ScalarNode:
			if onNode.Value != "" {
				set[onNode.Value] = struct{}{}
			}
		case yaml.SequenceNode:
			for _, n := range onNode.Content {
				if n.Kind == yaml.ScalarNode && n.Value != "" {
					set[n.Value] = struct{}{}
				}
			}
		case yaml.MappingNode:
			for i := 0; i < len(onNode.Content); i += 2 {
				set[onNode.Content[i].Value] = struct{}{}
			}
		}
	})
	return mapKeys(set)
}

func collectUsesActions(doc *yaml.Node) []string {
	set := map[string]struct{}{}
	visitMappingValue(doc, "jobs", func(jobsNode *yaml.Node) {
		for i := 1; i < len(jobsNode.Content); i += 2 {
			jobNode := jobsNode.Content[i]
			visitMappingValue(jobNode, "steps", func(stepsNode *yaml.Node) {
				for _, step := range stepsNode.Content {
					visitMappingValue(step, "uses", func(useNode *yaml.Node) {
						if useNode.Kind == yaml.ScalarNode && useNode.Value != "" {
							set[useNode.Value] = struct{}{}
						}
					})
				}
			})
		}
	})
	return mapKeys(set)
}

func collectReferencedInputs(doc *yaml.Node) []string {
	set := map[string]struct{}{}
	var scalars []string
	collectScalarValues(doc, &scalars)
	for _, s := range scalars {
		for _, m := range reEventInputRef.FindAllStringSubmatch(s, -1) {
			set[m[1]] = struct{}{}
		}
	}
	return mapKeys(set)
}

func collectDeprecatedCommands(doc *yaml.Node) []string {
	set := map[string]struct{}{}
	var scalars []string
	collectScalarValues(doc, &scalars)
	for _, s := range scalars {
		if strings.Contains(s, "::set-output") {
			set["set-output"] = struct{}{}
		}
		if strings.Contains(s, "::save-state") {
			set["save-state"] = struct{}{}
		}
	}
	return mapKeys(set)
}

func listWorkflowFiles(dir string) ([]string, error) {
	patterns := []string{
		filepath.Join(dir, "*.yml"),
		filepath.Join(dir, "*.yaml"),
	}
	var files []string
	for _, p := range patterns {
		matches, err := filepath.Glob(p)
		if err != nil {
			return nil, err
		}
		files = append(files, matches...)
	}
	sort.Strings(files)
	return files, nil
}

func mappingScalar(node *yaml.Node, key string) string {
	value := ""
	visitMappingValue(node, key, func(v *yaml.Node) {
		if v.Kind == yaml.ScalarNode {
			value = v.Value
		}
	})
	return value
}

func mappingHasKey(node *yaml.Node, key string) bool {
	if node == nil || node.Kind != yaml.MappingNode {
		return false
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return true
		}
	}
	return false
}

func collectScalarValues(node *yaml.Node, out *[]string) {
	if node == nil {
		return
	}
	if node.Kind == yaml.ScalarNode {
		*out = append(*out, node.Value)
		return
	}
	for _, child := range node.Content {
		collectScalarValues(child, out)
	}
}

func mapKeys(m map[string]struct{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
