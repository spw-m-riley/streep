package scaffold

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// eventPayload returns a minimal-but-valid JSON event payload for the given GitHub Actions event.
func eventPayload(event string) any {
	switch event {
	case "pull_request", "pull_request_target":
		return map[string]any{
			"pull_request": map[string]any{
				"head": map[string]any{"ref": "feature/my-branch"},
				"base": map[string]any{"ref": "main"},
				"number": 1,
			},
		}
	case "push":
		return map[string]any{
			"ref":    "refs/heads/main",
			"before": "0000000000000000000000000000000000000000",
			"after":  "0000000000000000000000000000000000000001",
		}
	case "release":
		return map[string]any{
			"release": map[string]any{
				"tag_name":   "v1.0.0",
				"draft":      false,
				"prerelease": false,
			},
		}
	case "create":
		return map[string]any{
			"ref":      "refs/heads/main",
			"ref_type": "branch",
		}
	case "workflow_dispatch":
		return map[string]any{"inputs": map[string]any{}}
	case "schedule":
		return map[string]any{}
	default:
		return map[string]any{}
	}
}

// writeEventFiles generates .act/events/<event>.json for each discovered event.
// Existing files are skipped unless force is true.
func writeEventFiles(targetDir string, events []string, force bool) error {
	eventsDir := filepath.Join(targetDir, ".act", "events")
	if err := os.MkdirAll(eventsDir, 0o755); err != nil {
		return fmt.Errorf("failed to create events directory: %w", err)
	}

	for _, event := range events {
		path := filepath.Join(eventsDir, event+".json")
		if !force {
			if _, err := os.Stat(path); err == nil {
				continue // already exists, skip
			}
		}

		payload := eventPayload(event)
		data, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal event payload for %s: %w", event, err)
		}
		if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
			return fmt.Errorf("failed to write event file %s: %w", path, err)
		}
	}
	return nil
}

// selfHostedWarning returns a human-readable guidance string for self-hosted runner label sets.
func selfHostedWarning(sets [][]string) string {
	if len(sets) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n⚠  Self-hosted runners detected:\n")
	for _, labels := range sets {
		fmt.Fprintf(&b, "   runs-on: [%s]\n", strings.Join(labels, ", "))
	}
	b.WriteString("\n   act only supports single-label -P mappings. Add to .actrc:\n")
	b.WriteString("     -P self-hosted=catthehacker/ubuntu:act-latest\n")
	b.WriteString("   Multi-label matching is not supported — all self-hosted labels\n")
	b.WriteString("   will map to the same image. See: https://github.com/nektos/act/issues/1285\n")
	return b.String()
}
