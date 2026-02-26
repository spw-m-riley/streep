package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLintShowsHelp(t *testing.T) {
	var out bytes.Buffer
	if err := executeLint([]string{"--help"}, &out, &bytes.Buffer{}); err != nil {
		t.Fatalf("executeLint() error: %v", err)
	}
	if !strings.Contains(out.String(), "streep lint") {
		t.Fatalf("expected help text, got: %q", out.String())
	}
}

func TestLintNoWorkflows(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	if err := executeLint([]string{dir}, &out, &bytes.Buffer{}); err != nil {
		t.Fatalf("executeLint() error: %v", err)
	}
	if !strings.Contains(out.String(), "No workflow files found") {
		t.Fatalf("expected no workflow message, got:\n%s", out.String())
	}
}

func TestLintFindsIssues(t *testing.T) {
	dir := t.TempDir()
	wfDir := filepath.Join(dir, ".github", "workflows")
	if err := os.MkdirAll(wfDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	writeCheckFile(t, filepath.Join(wfDir, "ci.yml"), `
on:
  push:
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v1
      - run: echo "::save-state name=x::y"
`)

	var out bytes.Buffer
	_ = executeLint([]string{dir}, &out, &bytes.Buffer{})
	got := out.String()
	for _, want := range []string{
		"deprecated-action-version",
		"deprecated-save-state",
		"missing-permissions",
		"Found",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in output, got:\n%s", want, got)
		}
	}
}

func TestLintFixUpdatesFile(t *testing.T) {
	dir := t.TempDir()
	wfDir := filepath.Join(dir, ".github", "workflows")
	if err := os.MkdirAll(wfDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	path := filepath.Join(wfDir, "ci.yml")
	writeCheckFile(t, path, `
name: CI
on: [push]
permissions:
  contents: read
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v1
`)

	var out bytes.Buffer
	_ = executeLint([]string{dir, "--fix"}, &out, &bytes.Buffer{})

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if !strings.Contains(string(data), "actions/checkout@v4") {
		t.Fatalf("expected action version fix in file, got:\n%s", data)
	}
	if !strings.Contains(out.String(), "Applied 1 action version fix(es)") {
		t.Fatalf("expected fix summary, got:\n%s", out.String())
	}
}

func TestLintDetectsCompositeShellSafety(t *testing.T) {
	dir := t.TempDir()
	wfDir := filepath.Join(dir, ".github", "workflows")
	if err := os.MkdirAll(wfDir, 0o755); err != nil {
		t.Fatalf("mkdir workflows: %v", err)
	}
	writeCheckFile(t, filepath.Join(wfDir, "ci.yml"), `
on: [push]
permissions:
  contents: read
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - run: echo ok
`)

	actionsDir := filepath.Join(dir, ".github", "actions", "custom")
	if err := os.MkdirAll(actionsDir, 0o755); err != nil {
		t.Fatalf("mkdir actions: %v", err)
	}
	writeCheckFile(t, filepath.Join(actionsDir, "action.yml"), `
runs:
  using: composite
  steps:
    - run: |
        if [[ -f file ]]; then
          echo ok
        fi
`)

	var out bytes.Buffer
	_ = executeLint([]string{dir}, &out, &bytes.Buffer{})
	if !strings.Contains(out.String(), "composite-shell-safety") {
		t.Fatalf("expected composite shell safety rule, got:\n%s", out.String())
	}
}

func TestLintJSONOutput(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	if err := executeLint([]string{dir, "--json"}, &out, &bytes.Buffer{}); err != nil {
		t.Fatalf("executeLint() error: %v", err)
	}
	var payload commandJSONResult
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("json unmarshal: %v\n%s", err, out.String())
	}
	if !payload.OK {
		t.Fatalf("expected ok=true payload, got %+v", payload)
	}
	if !strings.Contains(payload.Output, "No workflow files found") {
		t.Fatalf("expected human output wrapped in json payload, got %+v", payload)
	}
}
