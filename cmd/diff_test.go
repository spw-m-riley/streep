package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiffShowsHelp(t *testing.T) {
	var out bytes.Buffer
	if err := executeDiff([]string{"--help"}, &out, &bytes.Buffer{}); err != nil {
		t.Fatalf("executeDiff() error: %v", err)
	}
	if !strings.Contains(out.String(), "streep diff") {
		t.Fatalf("expected help output, got: %q", out.String())
	}
}

func TestDiffReportsWorkflowDelta(t *testing.T) {
	repo := t.TempDir()

	runGit(t, repo, "init")
	runGit(t, repo, "config", "user.email", "test@example.com")
	runGit(t, repo, "config", "user.name", "Test User")

	wfDir := filepath.Join(repo, ".github", "workflows")
	if err := os.MkdirAll(wfDir, 0o755); err != nil {
		t.Fatalf("mkdir workflows: %v", err)
	}
	writeCheckFile(t, filepath.Join(wfDir, "ci.yml"), `
on: [push]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - run: echo ${{ secrets.OLD_SECRET }} ${{ env.OLD_ENV }} ${{ vars.OLD_VAR }}
`)
	runGit(t, repo, "add", ".")
	runGit(t, repo, "commit", "-m", "first")

	writeCheckFile(t, filepath.Join(wfDir, "ci.yml"), `
on:
  push:
  pull_request:
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - run: echo ${{ secrets.NEW_SECRET }} ${{ env.NEW_ENV }} ${{ vars.NEW_VAR }}
  test:
    runs-on: ubuntu-latest
    needs: build
    steps:
      - run: echo ok
`)
	runGit(t, repo, "add", ".")
	runGit(t, repo, "commit", "-m", "second")

	var out bytes.Buffer
	if err := executeDiff([]string{"HEAD~1", repo}, &out, &bytes.Buffer{}); err != nil {
		t.Fatalf("executeDiff() error: %v", err)
	}
	got := out.String()
	for _, want := range []string{
		"modified workflow: .github/workflows/ci.yml",
		"jobs added: test",
		"events added: pull_request",
		"secrets added: NEW_SECRET",
		"secrets removed: OLD_SECRET",
		"env added: NEW_ENV",
		"env removed: OLD_ENV",
		"vars added: NEW_VAR",
		"vars removed: OLD_VAR",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in output, got:\n%s", want, got)
		}
	}
}

func TestDiffNoDelta(t *testing.T) {
	repo := t.TempDir()

	runGit(t, repo, "init")
	runGit(t, repo, "config", "user.email", "test@example.com")
	runGit(t, repo, "config", "user.name", "Test User")

	wfDir := filepath.Join(repo, ".github", "workflows")
	if err := os.MkdirAll(wfDir, 0o755); err != nil {
		t.Fatalf("mkdir workflows: %v", err)
	}
	writeCheckFile(t, filepath.Join(wfDir, "ci.yml"), `
on: [push]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - run: echo hi
`)
	runGit(t, repo, "add", ".")
	runGit(t, repo, "commit", "-m", "first")
	runGit(t, repo, "commit", "--allow-empty", "-m", "second")

	var out bytes.Buffer
	if err := executeDiff([]string{"HEAD~1", repo}, &out, &bytes.Buffer{}); err != nil {
		t.Fatalf("executeDiff() error: %v", err)
	}
	if strings.Contains(out.String(), "modified workflow") {
		t.Errorf("expected no delta output, got:\n%s", out.String())
	}
}

func TestDiffAddedRemovedWorkflow(t *testing.T) {
	repo := t.TempDir()

	runGit(t, repo, "init")
	runGit(t, repo, "config", "user.email", "test@example.com")
	runGit(t, repo, "config", "user.name", "Test User")

	wfDir := filepath.Join(repo, ".github", "workflows")
	if err := os.MkdirAll(wfDir, 0o755); err != nil {
		t.Fatalf("mkdir workflows: %v", err)
	}
	writeCheckFile(t, filepath.Join(wfDir, "ci.yml"), `
on: [push]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - run: echo ok
`)
	runGit(t, repo, "add", ".")
	runGit(t, repo, "commit", "-m", "first")

	writeCheckFile(t, filepath.Join(wfDir, "release.yml"), `
on: [push]
jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - run: echo release
`)
	if err := os.Remove(filepath.Join(wfDir, "ci.yml")); err != nil {
		t.Fatalf("remove ci.yml: %v", err)
	}
	runGit(t, repo, "add", ".")
	runGit(t, repo, "commit", "-m", "second")

	var out bytes.Buffer
	if err := executeDiff([]string{"HEAD~1", repo}, &out, &bytes.Buffer{}); err != nil {
		t.Fatalf("executeDiff() error: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "added workflow") {
		t.Errorf("expected added workflow, got:\n%s", got)
	}
	if !strings.Contains(got, "removed workflow") {
		t.Errorf("expected removed workflow, got:\n%s", got)
	}
}

func TestDiffJSONOutput(t *testing.T) {
	repo := t.TempDir()

	runGit(t, repo, "init")
	runGit(t, repo, "config", "user.email", "test@example.com")
	runGit(t, repo, "config", "user.name", "Test User")

	wfDir := filepath.Join(repo, ".github", "workflows")
	if err := os.MkdirAll(wfDir, 0o755); err != nil {
		t.Fatalf("mkdir workflows: %v", err)
	}
	writeCheckFile(t, filepath.Join(wfDir, "ci.yml"), `
on: [push]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - run: echo ${{ secrets.OLD_SECRET }}
`)
	runGit(t, repo, "add", ".")
	runGit(t, repo, "commit", "-m", "first")

	writeCheckFile(t, filepath.Join(wfDir, "ci.yml"), `
on: [push]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - run: echo ${{ secrets.NEW_SECRET }}
`)
	runGit(t, repo, "add", ".")
	runGit(t, repo, "commit", "-m", "second")

	var out bytes.Buffer
	err := executeDiff([]string{"HEAD~1", repo, "--json"}, &out, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("executeDiff() error: %v", err)
	}

	var payload commandJSONResult
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("json unmarshal: %v\n%s", err, out.String())
	}
	if !payload.OK {
		t.Fatalf("expected ok=true payload, got: %+v", payload)
	}
	if !strings.Contains(payload.Output, "modified workflow") {
		t.Fatalf("expected workflow delta in json output payload, got:\n%s", payload.Output)
	}
}

func TestDiffUsesConfigDefaultBaseline(t *testing.T) {
	repo := t.TempDir()
	origWD, _ := os.Getwd()
	if err := os.Chdir(repo); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(origWD) //nolint:errcheck

	runGit(t, repo, "init")
	runGit(t, repo, "config", "user.email", "test@example.com")
	runGit(t, repo, "config", "user.name", "Test User")

	wfDir := filepath.Join(repo, ".github", "workflows")
	if err := os.MkdirAll(wfDir, 0o755); err != nil {
		t.Fatalf("mkdir workflows: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(repo, ".streep"), 0o755); err != nil {
		t.Fatalf("mkdir .streep: %v", err)
	}
	writeCheckFile(t, filepath.Join(repo, ".streep", "config.yaml"), "defaults:\n  diff_base: HEAD~2\n")

	writeCheckFile(t, filepath.Join(wfDir, "ci.yml"), "on: [push]\n")
	runGit(t, repo, "add", ".")
	runGit(t, repo, "commit", "-m", "first")
	writeCheckFile(t, filepath.Join(wfDir, "ci.yml"), "on: [push]\nname: second\n")
	runGit(t, repo, "add", ".")
	runGit(t, repo, "commit", "-m", "second")
	runGit(t, repo, "commit", "--allow-empty", "-m", "third")

	var out bytes.Buffer
	if err := executeDiff(nil, &out, &bytes.Buffer{}); err != nil {
		t.Fatalf("executeDiff() error: %v", err)
	}
	if !strings.Contains(out.String(), "Workflow delta vs HEAD~2") {
		t.Fatalf("expected config baseline to be used, got:\n%s", out.String())
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}
