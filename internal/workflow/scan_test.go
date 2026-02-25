package workflow

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestScanDir(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "ci.yml"), `
name: CI
on:
  push:
  workflow_dispatch:
    inputs:
      environment:
        description: Target environment
        type: string
      debug:
        description: Enable debug
        type: boolean
jobs:
  build:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        node: [18, 20]
        os: [ubuntu-latest, ubuntu-22.04]
    env:
      APP_ENV: ${{ env.APP_ENV }}
    steps:
      - uses: actions/checkout@v4
      - run: echo ${{ secrets.GITHUB_TOKEN }}
      - run: echo ${{ secrets.DEPLOY_KEY }}
      - run: echo ${{ vars.RELEASE_CHANNEL }}
      - uses: actions/upload-artifact@v4
        with:
          name: dist
          path: dist/
`)
	writeFile(t, filepath.Join(dir, "deploy.yaml"), `
name: Deploy
on:
  pull_request:
jobs:
  deploy:
    runs-on: ubuntu-22.04
    steps:
      - uses: actions/checkout@v4
      - run: echo ${{ secrets.DEPLOY_KEY }}
      - run: echo ${{ secrets.NPM_TOKEN }}
      - run: echo ${{ vars.DEPLOY_ENV }}
      - run: echo ${{ env.APP_ENV }}
      - uses: actions/download-artifact@v4
        with:
          name: dist
  infra:
    runs-on: [self-hosted, linux, x64]
    steps:
      - run: echo deploying
`)

	got, err := ScanDir(dir)
	if err != nil {
		t.Fatalf("ScanDir() error: %v", err)
	}

	wantSecrets := []string{"DEPLOY_KEY", "GITHUB_TOKEN", "NPM_TOKEN"}
	wantEnv := []string{"APP_ENV"}
	wantVars := []string{"DEPLOY_ENV", "RELEASE_CHANNEL"}
	wantRunners := []string{"ubuntu-22.04", "ubuntu-latest"}
	wantEvents := []string{"pull_request", "push", "workflow_dispatch"}

	if !reflect.DeepEqual(got.Secrets, wantSecrets) {
		t.Errorf("Secrets: got %v, want %v", got.Secrets, wantSecrets)
	}
	if !reflect.DeepEqual(got.Env, wantEnv) {
		t.Errorf("Env: got %v, want %v", got.Env, wantEnv)
	}
	if !reflect.DeepEqual(got.Vars, wantVars) {
		t.Errorf("Vars: got %v, want %v", got.Vars, wantVars)
	}
	if !reflect.DeepEqual(got.Runners, wantRunners) {
		t.Errorf("Runners: got %v, want %v", got.Runners, wantRunners)
	}
	if !reflect.DeepEqual(got.Events, wantEvents) {
		t.Errorf("Events: got %v, want %v", got.Events, wantEvents)
	}

	// Self-hosted labels
	if len(got.SelfHosted) != 1 {
		t.Errorf("SelfHosted: expected 1 set, got %v", got.SelfHosted)
	} else {
		want := []string{"self-hosted", "linux", "x64"}
		if !reflect.DeepEqual(got.SelfHosted[0], want) {
			t.Errorf("SelfHosted[0]: got %v, want %v", got.SelfHosted[0], want)
		}
	}

	// Matrix: 2 nodes × 2 os = 4 combinations
	if got.MatrixCount != 4 {
		t.Errorf("MatrixCount: got %d, want 4", got.MatrixCount)
	}

	// Artifact actions
	foundUpload, foundDownload := false, false
	for _, a := range got.UsesActions {
		if containsStr(a, "upload-artifact") {
			foundUpload = true
		}
		if containsStr(a, "download-artifact") {
			foundDownload = true
		}
	}
	if !foundUpload {
		t.Errorf("UsesActions: expected upload-artifact, got %v", got.UsesActions)
	}
	if !foundDownload {
		t.Errorf("UsesActions: expected download-artifact, got %v", got.UsesActions)
	}

	// Workflow dispatch inputs
	inputs, ok := got.WorkflowInputs["ci.yml"]
	if !ok {
		t.Fatalf("WorkflowInputs: expected entry for ci.yml, got %v", got.WorkflowInputs)
	}
	wantInputs := []string{"debug", "environment"}
	if !reflect.DeepEqual(inputs, wantInputs) {
		t.Errorf("WorkflowInputs[ci.yml]: got %v, want %v", inputs, wantInputs)
	}
}

func TestScanDirEmptyDir(t *testing.T) {
	dir := t.TempDir()
	got, err := ScanDir(dir)
	if err != nil {
		t.Fatalf("ScanDir() error: %v", err)
	}
	if len(got.Secrets)+len(got.Env)+len(got.Vars)+len(got.Runners)+len(got.UsesActions) != 0 {
		t.Errorf("expected empty references, got %+v", got)
	}
}

func TestScanDirIgnoresComments(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "ci.yml"), `
on: [push]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      # - run: echo ${{ secrets.COMMENTED_OUT_SECRET }}
      - run: echo ${{ secrets.REAL_SECRET }}
`)
	got, err := ScanDir(dir)
	if err != nil {
		t.Fatalf("ScanDir() error: %v", err)
	}

	for _, s := range got.Secrets {
		if s == "COMMENTED_OUT_SECRET" {
			t.Error("scanner should not extract keys from YAML comments")
		}
	}

	found := false
	for _, s := range got.Secrets {
		if s == "REAL_SECRET" {
			found = true
		}
	}
	if !found {
		t.Error("scanner should extract REAL_SECRET from live step")
	}
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write fixture %s: %v", path, err)
	}
}
