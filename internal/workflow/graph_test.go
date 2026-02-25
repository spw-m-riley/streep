package workflow

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadWorkflowOverviews(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "ci.yml"), `
name: CI
on:
  push:
  workflow_dispatch:
    inputs:
      environment:
        description: env
permissions:
  contents: read
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: echo "hello"
  test:
    runs-on: ubuntu-latest
    needs: build
    strategy:
      matrix:
        os: [ubuntu-latest, ubuntu-22.04]
        node: [18, 20]
    steps:
      - run: echo "::set-output name=x::y"
      - run: echo ${{ github.event.inputs.environment }}
`)
	writeFile(t, filepath.Join(dir, "deploy.yaml"), `
name: Deploy
on: [pull_request]
jobs:
  deploy:
    runs-on: ubuntu-latest
    needs: [build]
    steps:
      - uses: actions/upload-artifact@v4
`)

	got, err := LoadWorkflowOverviews(dir)
	if err != nil {
		t.Fatalf("LoadWorkflowOverviews() error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 overviews, got %d", len(got))
	}

	ci := got[0]
	if ci.File != "ci.yml" {
		t.Fatalf("expected first file to be ci.yml, got %s", ci.File)
	}
	if !ci.HasPermissions {
		t.Fatalf("expected permissions to be detected")
	}
	if !reflect.DeepEqual(ci.Events, []string{"push", "workflow_dispatch"}) {
		t.Fatalf("events mismatch: %v", ci.Events)
	}
	if !reflect.DeepEqual(ci.DispatchInputs, []string{"environment"}) {
		t.Fatalf("dispatch inputs mismatch: %v", ci.DispatchInputs)
	}
	if !reflect.DeepEqual(ci.ReferencedInputs, []string{"environment"}) {
		t.Fatalf("referenced inputs mismatch: %v", ci.ReferencedInputs)
	}
	if !reflect.DeepEqual(ci.DeprecatedCommands, []string{"set-output"}) {
		t.Fatalf("deprecated commands mismatch: %v", ci.DeprecatedCommands)
	}
	if len(ci.Jobs) != 2 {
		t.Fatalf("expected 2 jobs, got %d", len(ci.Jobs))
	}
	if ci.Jobs[1].ID != "test" || !reflect.DeepEqual(ci.Jobs[1].Needs, []string{"build"}) {
		t.Fatalf("unexpected test job dependency: %+v", ci.Jobs[1])
	}
	if MatrixCombinations(ci.Jobs[1].MatrixDimensions) != 4 {
		t.Fatalf("expected matrix combinations = 4, got %d", MatrixCombinations(ci.Jobs[1].MatrixDimensions))
	}

	graph := RenderJobGraph(ci.Jobs)
	if !strings.Contains(graph, "test <- [build]") {
		t.Fatalf("expected dependency in rendered graph, got:\n%s", graph)
	}
}

func TestExpandMatrix(t *testing.T) {
	rows := ExpandMatrix(map[string][]string{
		"os":   []string{"ubuntu", "macos"},
		"node": []string{"18", "20"},
	}, 0)
	if len(rows) != 4 {
		t.Fatalf("expected 4 rows, got %d", len(rows))
	}

	limited := ExpandMatrix(map[string][]string{
		"os":   []string{"ubuntu", "macos"},
		"node": []string{"18", "20"},
	}, 2)
	if len(limited) != 2 {
		t.Fatalf("expected 2 rows with limit, got %d", len(limited))
	}
}
