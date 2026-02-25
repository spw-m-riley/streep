package cmd

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiagnoseShowsHelp(t *testing.T) {
	var out bytes.Buffer
	if err := executeDiagnose([]string{"--help"}, &out, &bytes.Buffer{}); err != nil {
		t.Fatalf("executeDiagnose() error: %v", err)
	}
	if !strings.Contains(out.String(), "streep diagnose") {
		t.Fatalf("expected help output, got: %q", out.String())
	}
}

func TestDiagnoseFindings(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "run.log")
	writeCheckFile(t, logPath, "Cannot connect to the Docker daemon at unix:///var/run/docker.sock. Is the docker daemon running?\n")

	var out bytes.Buffer
	if err := executeDiagnose([]string{logPath}, &out, &bytes.Buffer{}); err != nil {
		t.Fatalf("executeDiagnose() error: %v", err)
	}
	if !strings.Contains(out.String(), "docker-daemon") {
		t.Fatalf("expected docker finding, got:\n%s", out.String())
	}
}

func TestDiagnoseNoFindings(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "run.log")
	writeCheckFile(t, logPath, "everything passed\n")

	var out bytes.Buffer
	if err := executeDiagnose([]string{logPath}, &out, &bytes.Buffer{}); err != nil {
		t.Fatalf("executeDiagnose() error: %v", err)
	}
	if !strings.Contains(out.String(), "No known failure patterns matched") {
		t.Fatalf("expected fallback output, got:\n%s", out.String())
	}
}
