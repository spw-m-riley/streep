package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLintDirFindsIssues(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "ci.yml"), `
on:
  workflow_dispatch:
    inputs:
      env:
        description: env
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - run: echo "::set-output name=x::y"
      - run: echo ${{ github.event.inputs.missing_input }}
  test:
    runs-on: ubuntu-latest
    needs: [does_not_exist]
    steps:
      - run: echo ok
`)

	result, err := LintDir(dir, false)
	if err != nil {
		t.Fatalf("LintDir() error: %v", err)
	}
	if len(result.Issues) == 0 {
		t.Fatal("expected lint issues")
	}

	rules := map[string]bool{}
	for _, issue := range result.Issues {
		rules[issue.Rule] = true
	}
	for _, rule := range []string{
		"missing-permissions",
		"deprecated-action-version",
		"deprecated-set-output",
		"unreachable-job",
		"undeclared-input",
	} {
		if !rules[rule] {
			t.Fatalf("expected rule %q in issues, got %+v", rule, result.Issues)
		}
	}
}

func TestLintDirFixesDeprecatedActionVersions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ci.yml")
	writeFile(t, path, `
name: CI
on: [push]
permissions:
  contents: read
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
`)

	result, err := LintDir(dir, true)
	if err != nil {
		t.Fatalf("LintDir() error: %v", err)
	}
	if result.FixedActions != 1 || result.ChangedFiles != 1 {
		t.Fatalf("expected one fixed action in one file, got %+v", result)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if !strings.Contains(string(data), "actions/checkout@v4") {
		t.Fatalf("expected file to be updated to v4, got:\n%s", data)
	}
}
