package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExplainShowsHelp(t *testing.T) {
	var out bytes.Buffer
	if err := executeExplain([]string{"--help"}, &out, &bytes.Buffer{}); err != nil {
		t.Fatalf("executeExplain() error: %v", err)
	}
	if !strings.Contains(out.String(), "streep explain") {
		t.Fatalf("expected help text, got: %q", out.String())
	}
}

func TestExplainNoWorkflows(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	if err := executeExplain([]string{dir}, &out, &bytes.Buffer{}); err != nil {
		t.Fatalf("executeExplain() error: %v", err)
	}
	if !strings.Contains(out.String(), "No workflow files found") {
		t.Fatalf("expected no workflow message, got:\n%s", out.String())
	}
}

func TestExplainOutputsSectionsAndWarnings(t *testing.T) {
	dir := t.TempDir()
	wfDir := filepath.Join(dir, ".github", "workflows")
	if err := os.MkdirAll(wfDir, 0o755); err != nil {
		t.Fatalf("mkdir workflows: %v", err)
	}

	writeCheckFile(t, filepath.Join(wfDir, "ci.yml"), `
name: CI
on:
  push:
  workflow_dispatch:
    inputs:
      environment:
        description: env
jobs:
  build:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        os: [ubuntu-latest, ubuntu-22.04, ubuntu-20.04]
        node: [18, 20, 22, 24]
    steps:
      - uses: actions/checkout@v4
      - run: echo "::set-output name=a::b"
      - run: echo ${{ secrets.API_KEY }}
      - run: echo ${{ github.event.inputs.environment }}
  infra:
    runs-on: [self-hosted, linux, x64]
    needs: build
    steps:
      - run: echo done
`)

	var out bytes.Buffer
	if err := executeExplain([]string{dir}, &out, &bytes.Buffer{}); err != nil {
		t.Fatalf("executeExplain() error: %v", err)
	}
	got := out.String()

	for _, want := range []string{
		"Triggers",
		"Job graph",
		"Required secrets",
		"External actions",
		"Matrix expansion",
		"Warnings",
		"infra <- [build]",
		"ci.yml/build: 12 combination(s)",
		"self-hosted runner label set detected",
		"missing top-level permissions block",
		"deprecated workflow commands found",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in output, got:\n%s", want, got)
		}
	}
}

func TestExplainJSONOutput(t *testing.T) {
	dir := t.TempDir()
	wfDir := filepath.Join(dir, ".github", "workflows")
	if err := os.MkdirAll(wfDir, 0o755); err != nil {
		t.Fatalf("mkdir workflows: %v", err)
	}
	writeCheckFile(t, filepath.Join(wfDir, "ci.yml"), `
on: [push]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: echo ${{ secrets.API_KEY }}
`)

	var out bytes.Buffer
	if err := executeExplain([]string{dir, "--json"}, &out, &bytes.Buffer{}); err != nil {
		t.Fatalf("executeExplain() error: %v", err)
	}

	var payload struct {
		Directory       string   `json:"directory"`
		RequiredSecrets []string `json:"required_secrets"`
		Workflows       []struct {
			File string `json:"File"`
		} `json:"workflows"`
	}
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("json unmarshal: %v\n%s", err, out.String())
	}
	if payload.Directory != dir {
		t.Fatalf("expected directory %q, got %q", dir, payload.Directory)
	}
	if len(payload.Workflows) != 1 {
		t.Fatalf("expected one workflow in json payload, got %+v", payload.Workflows)
	}
	if len(payload.RequiredSecrets) != 1 || payload.RequiredSecrets[0] != "API_KEY" {
		t.Fatalf("unexpected required secrets payload: %+v", payload.RequiredSecrets)
	}
}
