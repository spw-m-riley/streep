package diagnose

import "testing"

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
