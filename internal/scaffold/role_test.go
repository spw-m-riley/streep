package scaffold

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupWorkflows(t *testing.T, dir string) {
	t.Helper()
	wfDir := filepath.Join(dir, ".github", "workflows")
	if err := os.MkdirAll(wfDir, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	content := `
on:
  push:
  pull_request:
  workflow_dispatch:
    inputs:
      environment:
        description: Target environment
        type: string
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: echo ${{ secrets.DEPLOY_KEY }}
      - run: echo ${{ env.APP_ENV }}
      - run: echo ${{ vars.RELEASE_CHANNEL }}
      - uses: actions/upload-artifact@v4
        with:
          name: dist
          path: dist/
  infra:
    runs-on: [self-hosted, linux, x64]
    steps:
      - run: echo infra
`
	if err := os.WriteFile(filepath.Join(wfDir, "ci.yml"), []byte(content), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
}

func TestNewRoleCreatesExpectedFiles(t *testing.T) {
	dir := t.TempDir()
	setupWorkflows(t, dir)

	var out bytes.Buffer
	if err := NewRole(RoleOptions{Dir: dir, Out: &out, Arch: "amd64"}); err != nil {
		t.Fatalf("NewRole() error: %v", err)
	}

	// Core scaffold files
	for _, rel := range []string{".secrets.example", ".env.example", ".vars.example", ".input.example", ".actrc", ".gitignore"} {
		path := filepath.Join(dir, rel)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected %s to exist: %v", rel, err)
		}
	}

	// .artifacts/ directory created (artifact actions detected)
	if _, err := os.Stat(filepath.Join(dir, ".artifacts")); err != nil {
		t.Errorf("expected .artifacts/ to exist: %v", err)
	}

	secretsData, _ := os.ReadFile(filepath.Join(dir, ".secrets.example"))
	if !strings.Contains(string(secretsData), "GITHUB_TOKEN=") {
		t.Errorf(".secrets.example missing GITHUB_TOKEN")
	}
	if !strings.Contains(string(secretsData), "DEPLOY_KEY=") {
		t.Errorf(".secrets.example missing DEPLOY_KEY")
	}

	envData, _ := os.ReadFile(filepath.Join(dir, ".env.example"))
	if !strings.Contains(string(envData), "APP_ENV=") {
		t.Errorf(".env.example missing APP_ENV")
	}

	varsData, _ := os.ReadFile(filepath.Join(dir, ".vars.example"))
	if !strings.Contains(string(varsData), "RELEASE_CHANNEL=") {
		t.Errorf(".vars.example missing RELEASE_CHANNEL")
	}

	inputData, _ := os.ReadFile(filepath.Join(dir, ".input.example"))
	if !strings.Contains(string(inputData), "environment=") {
		t.Errorf(".input.example missing environment, got:\n%s", inputData)
	}

	actrcData, _ := os.ReadFile(filepath.Join(dir, ".actrc"))
	for _, flag := range []string{
		"--secret-file .secrets",
		"--env-file .env",
		"--var-file .vars",
		"--input-file .input",
		"-P ubuntu-latest=catthehacker/ubuntu:act-latest",
		"-P self-hosted=catthehacker/ubuntu:act-latest",
		"--artifact-server-path .artifacts",
	} {
		if !strings.Contains(string(actrcData), flag) {
			t.Errorf(".actrc missing %q, got:\n%s", flag, actrcData)
		}
	}

	giData, _ := os.ReadFile(filepath.Join(dir, ".gitignore"))
	for _, entry := range []string{".secrets", ".env", ".vars", ".artifacts/"} {
		if !strings.Contains(string(giData), entry) {
			t.Errorf(".gitignore missing %q", entry)
		}
	}

	if !strings.Contains(out.String(), "Initialized role scaffold") {
		t.Errorf("unexpected output: %q", out.String())
	}
	if !strings.Contains(out.String(), "Next steps") {
		t.Errorf("expected next steps in output")
	}

	// Event JSON files for push and pull_request
	for _, ev := range []string{"push", "pull_request", "workflow_dispatch"} {
		path := filepath.Join(dir, ".act", "events", ev+".json")
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected .act/events/%s.json to exist: %v", ev, err)
		}
	}

	// pull_request payload has required fields
	prJSON, _ := os.ReadFile(filepath.Join(dir, ".act", "events", "pull_request.json"))
	if !strings.Contains(string(prJSON), "feature/my-branch") {
		t.Errorf("pull_request.json missing expected content, got:\n%s", prJSON)
	}

	// Self-hosted warning in output
	if !strings.Contains(out.String(), "Self-hosted runners detected") {
		t.Errorf("expected self-hosted warning in output, got:\n%s", out.String())
	}

	// Event command hints in output
	if !strings.Contains(out.String(), "pull_request.json") {
		t.Errorf("expected event file hints in output, got:\n%s", out.String())
	}
}

func TestNewRoleArm64AddsContainerArchitecture(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	if err := NewRole(RoleOptions{Dir: dir, Out: &out, Arch: "arm64"}); err != nil {
		t.Fatalf("NewRole() error: %v", err)
	}
	actrcData, _ := os.ReadFile(filepath.Join(dir, ".actrc"))
	if !strings.Contains(string(actrcData), "--container-architecture linux/amd64") {
		t.Errorf("arm64 host: expected --container-architecture linux/amd64 in .actrc, got:\n%s", actrcData)
	}
}

func TestNewRoleNoWorkflowsFallback(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	if err := NewRole(RoleOptions{Dir: dir, Out: &out}); err != nil {
		t.Fatalf("NewRole() error: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, ".secrets.example"))
	if !strings.Contains(string(data), "GITHUB_TOKEN=") {
		t.Errorf(".secrets.example missing GITHUB_TOKEN in fallback case")
	}
}

func TestNewRoleRefusesOverwriteWithoutForce(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".secrets.example")
	if err := os.WriteFile(path, []byte("ORIGINAL\n"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	err := NewRole(RoleOptions{Dir: dir})
	if err == nil {
		t.Fatal("expected overwrite error")
	}
	if !strings.Contains(err.Error(), "refusing to overwrite existing file") {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := os.ReadFile(path)
	if string(got) != "ORIGINAL\n" {
		t.Fatalf("file was unexpectedly overwritten")
	}
}

func TestNewRoleOverwritesWhenForced(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".secrets.example")
	if err := os.WriteFile(path, []byte("ORIGINAL\n"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	if err := NewRole(RoleOptions{Dir: dir, Force: true}); err != nil {
		t.Fatalf("NewRole() error: %v", err)
	}

	got, _ := os.ReadFile(path)
	if !strings.Contains(string(got), "GITHUB_TOKEN=") {
		t.Fatalf("expected forced overwrite, got: %q", string(got))
	}
}
