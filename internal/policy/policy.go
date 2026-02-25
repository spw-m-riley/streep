package policy

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

var pinnedSHAPattern = regexp.MustCompile(`^[a-f0-9]{40}$`)

// Finding is a policy violation in a workflow file.
type Finding struct {
	File    string
	Rule    string
	Message string
}

type config struct {
	Rules struct {
		WriteAllPermissions *bool `yaml:"write_all_permissions"`
		PullRequestTarget   *bool `yaml:"pull_request_target"`
		UnpinnedActions     *bool `yaml:"unpinned_actions"`
	} `yaml:"rules"`
}

// CheckDir evaluates workflow files against policy rules.
func CheckDir(repoDir string) ([]Finding, error) {
	cfg, err := loadConfig(repoDir)
	if err != nil {
		return nil, err
	}

	files, err := workflowFiles(repoDir)
	if err != nil {
		return nil, err
	}

	var findings []Finding
	for _, f := range files {
		fileFindings, err := checkFile(f, cfg)
		if err != nil {
			return nil, err
		}
		findings = append(findings, fileFindings...)
	}

	sort.Slice(findings, func(i, j int) bool {
		if findings[i].File != findings[j].File {
			return findings[i].File < findings[j].File
		}
		if findings[i].Rule != findings[j].Rule {
			return findings[i].Rule < findings[j].Rule
		}
		return findings[i].Message < findings[j].Message
	})
	return findings, nil
}

func checkFile(path string, cfg config) ([]Finding, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var root yaml.Node
	dec := yaml.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&root); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", filepath.Base(path), err)
	}
	if root.Kind != yaml.DocumentNode || len(root.Content) == 0 {
		return nil, nil
	}
	doc := root.Content[0]
	file := filepath.Base(path)

	var findings []Finding
	if enabled(cfg.Rules.WriteAllPermissions) && hasWriteAllPermissions(doc) {
		findings = append(findings, Finding{
			File:    file,
			Rule:    "write-all-permissions",
			Message: "permissions uses write-all",
		})
	}
	if enabled(cfg.Rules.PullRequestTarget) && hasEvent(doc, "pull_request_target") {
		findings = append(findings, Finding{
			File:    file,
			Rule:    "pull-request-target",
			Message: "uses pull_request_target; ensure untrusted code is not executed with elevated token access",
		})
	}
	if enabled(cfg.Rules.UnpinnedActions) {
		for _, use := range collectUses(doc) {
			if !isUnpinnedRemoteAction(use) {
				continue
			}
			findings = append(findings, Finding{
				File:    file,
				Rule:    "unpinned-action",
				Message: fmt.Sprintf("action %q is not pinned to a full commit SHA", use),
			})
		}
	}
	return findings, nil
}

func enabled(v *bool) bool {
	if v == nil {
		return true
	}
	return *v
}

func loadConfig(repoDir string) (config, error) {
	var cfg config
	path := filepath.Join(repoDir, ".streep", "policy.yaml")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse policy config: %w", err)
	}
	return cfg, nil
}

func workflowFiles(repoDir string) ([]string, error) {
	patterns := []string{
		filepath.Join(repoDir, ".github", "workflows", "*.yml"),
		filepath.Join(repoDir, ".github", "workflows", "*.yaml"),
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

func hasWriteAllPermissions(doc *yaml.Node) bool {
	permissionsNode := mapValue(doc, "permissions")
	if permissionsNode == nil {
		return false
	}
	switch permissionsNode.Kind {
	case yaml.ScalarNode:
		return strings.EqualFold(strings.TrimSpace(permissionsNode.Value), "write-all")
	case yaml.MappingNode:
		for i := 0; i+1 < len(permissionsNode.Content); i += 2 {
			val := permissionsNode.Content[i+1]
			if val.Kind == yaml.ScalarNode && strings.EqualFold(strings.TrimSpace(val.Value), "write-all") {
				return true
			}
		}
	}
	return false
}

func hasEvent(doc *yaml.Node, event string) bool {
	onNode := mapValue(doc, "on")
	if onNode == nil {
		return false
	}
	switch onNode.Kind {
	case yaml.ScalarNode:
		return onNode.Value == event
	case yaml.SequenceNode:
		for _, item := range onNode.Content {
			if item.Kind == yaml.ScalarNode && item.Value == event {
				return true
			}
		}
	case yaml.MappingNode:
		for i := 0; i < len(onNode.Content); i += 2 {
			if onNode.Content[i].Value == event {
				return true
			}
		}
	}
	return false
}

func collectUses(doc *yaml.Node) []string {
	set := map[string]struct{}{}
	jobsNode := mapValue(doc, "jobs")
	if jobsNode == nil || jobsNode.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(jobsNode.Content); i += 2 {
		jobNode := jobsNode.Content[i+1]
		stepsNode := mapValue(jobNode, "steps")
		if stepsNode == nil || stepsNode.Kind != yaml.SequenceNode {
			continue
		}
		for _, step := range stepsNode.Content {
			useNode := mapValue(step, "uses")
			if useNode != nil && useNode.Kind == yaml.ScalarNode && useNode.Value != "" {
				set[useNode.Value] = struct{}{}
			}
		}
	}
	values := make([]string, 0, len(set))
	for v := range set {
		values = append(values, v)
	}
	sort.Strings(values)
	return values
}

func isUnpinnedRemoteAction(use string) bool {
	if strings.HasPrefix(use, "./") || strings.HasPrefix(use, "docker://") {
		return false
	}
	left, ref, ok := strings.Cut(use, "@")
	if !ok || ref == "" || left == "" {
		return false
	}
	parts := strings.Split(left, "/")
	if len(parts) < 2 {
		return false
	}
	return !pinnedSHAPattern.MatchString(ref)
}

func mapValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}
