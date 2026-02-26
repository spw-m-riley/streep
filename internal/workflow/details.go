package workflow

import (
	"bytes"
	"fmt"

	"gopkg.in/yaml.v3"
)

// Details is a compact workflow summary used for diffs.
type Details struct {
	Jobs    []string
	Events  []string
	Secrets []string
}

// ParseDetails parses workflow YAML bytes and extracts jobs/events/secrets.
func ParseDetails(data []byte) (Details, error) {
	var root yaml.Node
	dec := yaml.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&root); err != nil {
		return Details{}, err
	}
	if root.Kind != yaml.DocumentNode || len(root.Content) == 0 {
		return Details{}, fmt.Errorf("invalid workflow document")
	}
	doc := root.Content[0]

	jobsSet := map[string]struct{}{}
	visitMappingValue(doc, "jobs", func(jobsNode *yaml.Node) {
		if jobsNode.Kind != yaml.MappingNode {
			return
		}
		for i := 0; i+1 < len(jobsNode.Content); i += 2 {
			jobsSet[jobsNode.Content[i].Value] = struct{}{}
		}
	})

	secretsSet := map[string]struct{}{}
	var scalars []string
	collectScalarValues(doc, &scalars)
	for _, s := range scalars {
		for _, m := range reSecrets.FindAllStringSubmatch(s, -1) {
			secretsSet[m[1]] = struct{}{}
		}
	}

	return Details{
		Jobs:    mapKeys(jobsSet),
		Events:  collectWorkflowEvents(doc),
		Secrets: mapKeys(secretsSet),
	}, nil
}
