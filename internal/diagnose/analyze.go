package diagnose

import (
	"sort"
	"strings"
)

// Finding is a likely root cause extracted from a run log.
type Finding struct {
	Rule       string
	Reason     string
	Suggestion string
}

type matcher struct {
	rule       string
	reason     string
	suggestion string
	tokens     []string
}

var matchers = []matcher{
	{
		rule:       "docker-daemon",
		reason:     "Docker daemon is not reachable.",
		suggestion: "Start Docker Desktop (or docker daemon) and rerun `streep doctor`.",
		tokens:     []string{"cannot connect to the docker daemon", "is the docker daemon running"},
	},
	{
		rule:       "docker-socket-permission",
		reason:     "Permission denied while accessing Docker socket.",
		suggestion: "Ensure your user can access /var/run/docker.sock (docker group / Docker Desktop permissions).",
		tokens:     []string{"permission denied", "/var/run/docker.sock"},
	},
	{
		rule:       "github-token-missing",
		reason:     "GITHUB_TOKEN is missing or invalid in .secrets — act cannot clone remote actions from GitHub.",
		suggestion: "Set a valid GitHub PAT as GITHUB_TOKEN in .secrets. Run `streep check` to validate.",
		tokens:     []string{"authentication required", "invalid username or token"},
	},
	{
		rule:       "missing-secrets-file",
		reason:     "act could not read a required credentials file.",
		suggestion: "Run `streep new role`, then copy and fill .secrets/.env/.vars/.input before running.",
		tokens:     []string{"no such file or directory", ".secrets"},
	},
	{
		rule:       "missing-env-file",
		reason:     "act could not read .env file.",
		suggestion: "Create .env from .env.example and fill required values.",
		tokens:     []string{"no such file or directory", ".env"},
	},
	{
		rule:       "action-resolution",
		reason:     "act failed to resolve a remote action.",
		suggestion: "Check network access and action reference; consider `streep bundle actions` for offline runs.",
		tokens:     []string{"unable to resolve action", "repository not found"},
	},
	{
		rule:       "yaml-validation",
		reason:     "Workflow YAML appears invalid.",
		suggestion: "Run `streep lint` and fix the YAML/workflow validation errors.",
		tokens:     []string{"workflow is not valid", "yaml: line"},
	},
	{
		rule:       "missing-gh-cli",
		reason:     "Workflow expects gh CLI but it's not installed in the runner image.",
		suggestion: "Switch runner image or install gh in a setup step.",
		tokens:     []string{"gh: command not found"},
	},
	{
		rule:       "codeql-tooling",
		reason:     "CodeQL tooling likely unavailable in the selected runner image.",
		suggestion: "Use a fuller image mapping (`-P ...`) or install required tooling in steps.",
		tokens:     []string{"codeql", "not found"},
	},
}

// AnalyzeLog returns likely root causes from an act log.
func AnalyzeLog(log string) []Finding {
	lower := strings.ToLower(log)
	seen := map[string]bool{}
	var findings []Finding
	for _, m := range matchers {
		if seen[m.rule] {
			continue
		}
		if hasAllTokens(lower, m.tokens) {
			seen[m.rule] = true
			findings = append(findings, Finding{
				Rule:       m.rule,
				Reason:     m.reason,
				Suggestion: m.suggestion,
			})
		}
	}
	sort.Slice(findings, func(i, j int) bool { return findings[i].Rule < findings[j].Rule })
	return findings
}

func hasAllTokens(logLower string, tokens []string) bool {
	for _, tok := range tokens {
		if !strings.Contains(logLower, tok) {
			return false
		}
	}
	return true
}
