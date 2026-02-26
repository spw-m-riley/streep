package cmd

import (
	"bytes"
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
      - run: echo ${{ secrets.OLD_SECRET }}
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
      - run: echo ${{ secrets.NEW_SECRET }}
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

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}
