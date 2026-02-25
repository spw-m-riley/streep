package diagnose

import (
	"strings"
	"testing"
)

func TestAnalyzeLogFindings(t *testing.T) {
	log := `
Error: Cannot connect to the Docker daemon at unix:///var/run/docker.sock. Is the docker daemon running?
gh: command not found
`
	findings := AnalyzeLog(log)
	if len(findings) < 2 {
		t.Fatalf("expected at least 2 findings, got %d (%+v)", len(findings), findings)
	}
}

func TestAnalyzeLogNoFindings(t *testing.T) {
	findings := AnalyzeLog("all good")
	if len(findings) != 0 {
		t.Fatalf("expected no findings, got %+v", findings)
	}
}

func TestAnalyzeLogGitHubTokenMissing(t *testing.T) {
	log := `authentication required: Invalid username or token. Password authentication is not supported for Git operations.`
	findings := AnalyzeLog(log)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d (%+v)", len(findings), findings)
	}
	if findings[0].Rule != "github-token-missing" {
		t.Errorf("expected rule github-token-missing, got %q", findings[0].Rule)
	}
	if !strings.Contains(findings[0].Suggestion, "GITHUB_TOKEN") {
		t.Errorf("expected suggestion to mention GITHUB_TOKEN, got %q", findings[0].Suggestion)
	}
}
