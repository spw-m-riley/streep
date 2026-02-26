package cmd

import (
	"bytes"
	"encoding/json"
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
	_ = executeCheck([]string{dir}, &out, &bytes.Buffer{})
	if !strings.Contains(out.String(), ".secrets not found") {
		t.Errorf("expected missing file message, got:\n%s", out.String())
	}
}

func TestCheckFailsWhenValueEmpty(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, ".secrets.example"), "GITHUB_TOKEN=\nDEPLOY_KEY=\n")
	writeCheckFile(t, filepath.Join(dir, ".secrets"), "GITHUB_TOKEN=ghp_abc123\nDEPLOY_KEY=\n")

	var out bytes.Buffer
	_ = executeCheck([]string{dir}, &out, &bytes.Buffer{})
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

func TestCheckWarnsForNonClassicPAT(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, ".secrets.example"), "GITHUB_TOKEN=\n")
	writeCheckFile(t, filepath.Join(dir, ".secrets"), "GITHUB_TOKEN=ghs_notaclassicpat\n")

	var out bytes.Buffer
	_ = executeCheck([]string{dir}, &out, &bytes.Buffer{})
	if !strings.Contains(out.String(), "ghp_") {
		t.Errorf("expected classic PAT warning, got:\n%s", out.String())
	}
}

func TestCheckJSONOutput(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, ".secrets.example"), "GITHUB_TOKEN=\n")

	var out bytes.Buffer
	err := executeCheck([]string{dir, "--json"}, &out, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected check failure error in json mode, got nil")
	}

	var payload commandJSONResult
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("json unmarshal: %v\n%s", err, out.String())
	}
	if payload.OK {
		t.Fatalf("expected ok=false payload, got %+v", payload)
	}
	if payload.Error == "" {
		t.Fatalf("expected error text in payload, got %+v", payload)
	}
}

func writeCheckFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writeCheckFile(%s): %v", path, err)
	}
}
