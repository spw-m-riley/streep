package workflow

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// References holds all values discovered across workflow files.
type References struct {
	Secrets        []string            // names from ${{ secrets.NAME }}
	Env            []string            // names from ${{ env.NAME }}
	Vars           []string            // names from ${{ vars.NAME }}
	Runners        []string            // unique scalar runs-on values (github-hosted)
	SelfHosted     [][]string          // unique self-hosted label sets (each element is one runs-on array)
	UsesActions    []string            // unique uses: action references
	WorkflowInputs map[string][]string // workflow_dispatch input names keyed by workflow filename
	Events         []string            // unique on: trigger event names
	MatrixCount    int                 // total number of matrix combinations across all workflows
}

var (
	reSecrets = regexp.MustCompile(`\${{\s*secrets\.([A-Za-z_][A-Za-z0-9_]*)\s*}}`)
	reEnv     = regexp.MustCompile(`\${{\s*env\.([A-Za-z_][A-Za-z0-9_]*)\s*}}`)
	reVars    = regexp.MustCompile(`\${{\s*vars\.([A-Za-z_][A-Za-z0-9_]*)\s*}}`)
)

// ScanDir reads all *.yml and *.yaml files in dir and returns discovered references.
// If dir does not exist, empty References are returned without error.
// The scan operates on the parsed YAML node tree so that comments are excluded.
func ScanDir(dir string) (References, error) {
	secrets := map[string]struct{}{}
	env := map[string]struct{}{}
	vars := map[string]struct{}{}
	runners := map[string]struct{}{}
	actions := map[string]struct{}{}
	events := map[string]struct{}{}
	workflowInputs := map[string][]string{}

	// Self-hosted label sets deduplicated by their canonical string key.
	selfHostedSeen := map[string]bool{}
	var selfHostedSets [][]string
	totalMatrixCount := 0

	patterns := []string{
		filepath.Join(dir, "*.yml"),
		filepath.Join(dir, "*.yaml"),
	}

	var files []string
	for _, p := range patterns {
		matches, err := filepath.Glob(p)
		if err != nil {
			return References{}, err
		}
		files = append(files, matches...)
	}

	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			return References{}, err
		}

		var root yaml.Node
		dec := yaml.NewDecoder(bytes.NewReader(data))
		if err := dec.Decode(&root); err != nil {
			return References{}, err
		}

		// Expression scanning on all scalars
		var scalars []string
		collectScalars(&root, &scalars)
		for _, s := range scalars {
			for _, m := range reSecrets.FindAllStringSubmatch(s, -1) {
				secrets[m[1]] = struct{}{}
			}
			for _, m := range reEnv.FindAllStringSubmatch(s, -1) {
				env[m[1]] = struct{}{}
			}
			for _, m := range reVars.FindAllStringSubmatch(s, -1) {
				vars[m[1]] = struct{}{}
			}
		}

		// Structured extraction using the YAML node tree
		if root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
			doc := root.Content[0]
			collectRunners(doc, runners, selfHostedSeen, &selfHostedSets)
			collectActions(doc, actions)
			collectEvents(doc, events)
			if inputs := collectWorkflowDispatchInputs(doc); len(inputs) > 0 {
				workflowInputs[filepath.Base(f)] = inputs
			}
			totalMatrixCount += countMatrixCombinations(doc)
		}
	}

	return References{
		Secrets:        sortedKeys(secrets),
		Env:            sortedKeys(env),
		Vars:           sortedKeys(vars),
		Runners:        sortedKeys(runners),
		SelfHosted:     selfHostedSets,
		UsesActions:    sortedKeys(actions),
		WorkflowInputs: workflowInputs,
		Events:         sortedKeys(events),
		MatrixCount:    totalMatrixCount,
	}, nil
}

// collectRunners walks a workflow document node and collects runs-on values.
// Scalar values (github-hosted) go into the runners set.
// Sequence values that contain "self-hosted" are collected as label sets.
func collectRunners(doc *yaml.Node, githubHosted map[string]struct{}, selfHostedSeen map[string]bool, selfHostedSets *[][]string) {
	visitMappingValue(doc, "jobs", func(jobsNode *yaml.Node) {
		for i := 1; i < len(jobsNode.Content); i += 2 {
			jobConfig := jobsNode.Content[i]
			visitMappingValue(jobConfig, "runs-on", func(v *yaml.Node) {
				switch v.Kind {
				case yaml.ScalarNode:
					githubHosted[v.Value] = struct{}{}
				case yaml.SequenceNode:
					var labels []string
					for _, item := range v.Content {
						if item.Kind == yaml.ScalarNode {
							labels = append(labels, item.Value)
						}
					}
					if len(labels) > 0 {
						isSelfHosted := false
						for _, l := range labels {
							if l == "self-hosted" {
								isSelfHosted = true
								break
							}
						}
						if isSelfHosted {
							key := strings.Join(labels, ",")
							if !selfHostedSeen[key] {
								selfHostedSeen[key] = true
								*selfHostedSets = append(*selfHostedSets, labels)
							}
						} else {
							// Non-self-hosted sequence (rare) — use first label as runner name.
							githubHosted[labels[0]] = struct{}{}
						}
					}
				}
			})
		}
	})
}

// collectEvents collects unique trigger event names from the top-level "on:" key.
func collectEvents(doc *yaml.Node, out map[string]struct{}) {
	visitMappingValue(doc, "on", func(onNode *yaml.Node) {
		switch onNode.Kind {
		case yaml.ScalarNode:
			out[onNode.Value] = struct{}{}
		case yaml.SequenceNode:
			for _, item := range onNode.Content {
				if item.Kind == yaml.ScalarNode {
					out[item.Value] = struct{}{}
				}
			}
		case yaml.MappingNode:
			for i := 0; i < len(onNode.Content); i += 2 {
				out[onNode.Content[i].Value] = struct{}{}
			}
		}
	})
}

// countMatrixCombinations counts the total expanded matrix combinations in a workflow document.
// It does not account for include/exclude modifiers — this is an approximation for user guidance.
func countMatrixCombinations(doc *yaml.Node) int {
	total := 0
	visitMappingValue(doc, "jobs", func(jobsNode *yaml.Node) {
		for i := 1; i < len(jobsNode.Content); i += 2 {
			jobConfig := jobsNode.Content[i]
			visitMappingValue(jobConfig, "strategy", func(stratNode *yaml.Node) {
				visitMappingValue(stratNode, "matrix", func(matrixNode *yaml.Node) {
					if matrixNode.Kind != yaml.MappingNode {
						return
					}
					combinations := 1
					for j := 0; j+1 < len(matrixNode.Content); j += 2 {
						key := matrixNode.Content[j].Value
						val := matrixNode.Content[j+1]
						// Skip include/exclude keys
						if key == "include" || key == "exclude" {
							continue
						}
						if val.Kind == yaml.SequenceNode {
							combinations *= len(val.Content)
						}
					}
					total += combinations
				})
			})
		}
	})
	return total
}

// collectActions walks a workflow document node and collects all uses: values.
func collectActions(doc *yaml.Node, out map[string]struct{}) {
	visitMappingValue(doc, "jobs", func(jobsNode *yaml.Node) {
		for i := 1; i < len(jobsNode.Content); i += 2 {
			jobConfig := jobsNode.Content[i]
			visitMappingValue(jobConfig, "steps", func(stepsNode *yaml.Node) {
				for _, step := range stepsNode.Content {
					visitMappingValue(step, "uses", func(v *yaml.Node) {
						if v.Kind == yaml.ScalarNode && v.Value != "" {
							out[v.Value] = struct{}{}
						}
					})
				}
			})
		}
	})
}

// collectWorkflowDispatchInputs returns input names declared under on.workflow_dispatch.inputs.
func collectWorkflowDispatchInputs(doc *yaml.Node) []string {
	var inputs []string
	visitMappingValue(doc, "on", func(onNode *yaml.Node) {
		// "on" can be a scalar, sequence, or mapping
		if onNode.Kind != yaml.MappingNode {
			return
		}
		visitMappingValue(onNode, "workflow_dispatch", func(wdNode *yaml.Node) {
			if wdNode.Kind != yaml.MappingNode {
				return
			}
			visitMappingValue(wdNode, "inputs", func(inputsNode *yaml.Node) {
				if inputsNode.Kind != yaml.MappingNode {
					return
				}
				// keys at even indices are input names
				for i := 0; i < len(inputsNode.Content); i += 2 {
					inputs = append(inputs, inputsNode.Content[i].Value)
				}
			})
		})
	})
	sort.Strings(inputs)
	return inputs
}

// visitMappingValue calls fn with the value node for the given key in a YAML mapping node.
func visitMappingValue(node *yaml.Node, key string, fn func(*yaml.Node)) {
	if node == nil || node.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			fn(node.Content[i+1])
			return
		}
	}
}

func collectScalars(n *yaml.Node, out *[]string) {
	if n == nil {
		return
	}
	if n.Kind == yaml.ScalarNode {
		*out = append(*out, n.Value)
		return
	}
	for _, child := range n.Content {
		collectScalars(child, out)
	}
}

func sortedKeys(m map[string]struct{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
