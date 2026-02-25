package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckPassesWhenAllValuesPresent(t *testing.T) {
	dir := t.TempDir()

	writeCheckFile(t, filepath.Join(dir, ".secrets.example"), "GITHUB_TOKEN=\nDEPLOY_KEY=\n")
	writeCheckFile(t, filepath.Join(dir, ".secrets"), "GITHUB_TOKEN=ghp_abc123\nDEPLOY_KEY=ssh-rsa AAAA\n")
	writeCheckFile(t, filepath.Join(dir, ".env.example"), "APP_ENV=\n")
	writeCheckFile(t, filepath.Join(dir, ".env"), "APP_ENV=local\n")

	var out bytes.Buffer
	if err := executeCheck([]string{dir}, &out, &bytes.Buffer{}); err != nil {
		t.Fatalf("executeCheck() error: %v", err)
	}
	if !strings.Contains(out.String(), "All checks passed") {
		t.Errorf("expected all checks passed, got:\n%s", out.String())
	}
}

func TestCheckFailsWhenRealFileMissing(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, ".secrets.example"), "GITHUB_TOKEN=\n")

	var out bytes.Buffer
	if err := executeCheck([]string{dir}, &out, &bytes.Buffer{}); err != nil {
		t.Fatalf("executeCheck() error: %v", err)
	}
	if !strings.Contains(out.String(), ".secrets not found") {
		t.Errorf("expected missing file message, got:\n%s", out.String())
	}
}

func TestCheckFailsWhenValueEmpty(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, ".secrets.example"), "GITHUB_TOKEN=\nDEPLOY_KEY=\n")
	writeCheckFile(t, filepath.Join(dir, ".secrets"), "GITHUB_TOKEN=ghp_abc123\nDEPLOY_KEY=\n")

	var out bytes.Buffer
	if err := executeCheck([]string{dir}, &out, &bytes.Buffer{}); err != nil {
		t.Fatalf("executeCheck() error: %v", err)
	}
	if !strings.Contains(out.String(), "DEPLOY_KEY") {
		t.Errorf("expected DEPLOY_KEY to be flagged, got:\n%s", out.String())
	}
}

func TestCheckSkipsWhenNoExampleFile(t *testing.T) {
	dir := t.TempDir()
	// Only .env.example exists, no secrets example at all
	writeCheckFile(t, filepath.Join(dir, ".env.example"), "APP_ENV=\n")
	writeCheckFile(t, filepath.Join(dir, ".env"), "APP_ENV=local\n")

	var out bytes.Buffer
	if err := executeCheck([]string{dir}, &out, &bytes.Buffer{}); err != nil {
		t.Fatalf("executeCheck() error: %v", err)
	}
	if !strings.Contains(out.String(), "All checks passed") {
		t.Errorf("expected all checks passed (only env checked), got:\n%s", out.String())
	}
}

func TestCheckShowsHelp(t *testing.T) {
	var out bytes.Buffer
	if err := executeCheck([]string{"--help"}, &out, &bytes.Buffer{}); err != nil {
		t.Fatalf("executeCheck() error: %v", err)
	}
	if !strings.Contains(out.String(), "streep check") {
		t.Errorf("expected help text, got: %q", out.String())
	}
}

func writeCheckFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writeCheckFile(%s): %v", path, err)
	}
}
