package editor

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestEditPromptModeUpdatesValues(t *testing.T) {
	dir := t.TempDir()
	template := filepath.Join(dir, ".secrets.example")
	target := filepath.Join(dir, ".secrets")

	if err := os.WriteFile(template, []byte("GITHUB_TOKEN=\nAPI_KEY=\n"), 0o644); err != nil {
		t.Fatalf("write template: %v", err)
	}
	if err := os.WriteFile(target, []byte("GITHUB_TOKEN=old-token\nAPI_KEY=old-key\n"), 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}

	in := strings.NewReader("\nnew-api-key\n")
	var out bytes.Buffer
	err := Edit(Options{
		FilePath:     target,
		TemplatePath: template,
		Redact:       true,
		In:           in,
		Out:          &out,
		Editor:       " ",
	})
	if err != nil {
		t.Fatalf("Edit() error: %v", err)
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	got := string(data)
	if !strings.Contains(got, "GITHUB_TOKEN=old-token") {
		t.Fatalf("expected kept value for token, got:\n%s", got)
	}
	if !strings.Contains(got, "API_KEY=new-api-key") {
		t.Fatalf("expected updated value for api key, got:\n%s", got)
	}
	if !strings.Contains(out.String(), "GITHUB_TOKEN [******]") {
		t.Fatalf("expected redacted prompt output, got:\n%s", out.String())
	}
}

func TestEditEditorMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script helper is unix-only")
	}

	dir := t.TempDir()
	template := filepath.Join(dir, ".env.example")
	target := filepath.Join(dir, ".env")
	editorPath := filepath.Join(dir, "fake-editor")

	if err := os.WriteFile(template, []byte("APP_ENV=\n"), 0o644); err != nil {
		t.Fatalf("write template: %v", err)
	}
	if err := os.WriteFile(editorPath, []byte("#!/bin/sh\necho \"APP_ENV=local\" > \"$1\"\n"), 0o755); err != nil {
		t.Fatalf("write editor: %v", err)
	}

	err := Edit(Options{
		FilePath:     target,
		TemplatePath: template,
		Editor:       editorPath,
		Out:          &bytes.Buffer{},
		Err:          &bytes.Buffer{},
	})
	if err != nil {
		t.Fatalf("Edit() error: %v", err)
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if !strings.Contains(string(data), "APP_ENV=local") {
		t.Fatalf("expected fake editor to write target file, got:\n%s", data)
	}
}

func TestEditRequiresTemplate(t *testing.T) {
	dir := t.TempDir()
	err := Edit(Options{
		FilePath:     filepath.Join(dir, ".vars"),
		TemplatePath: filepath.Join(dir, ".vars.example"),
		In:           strings.NewReader(""),
		Out:          &bytes.Buffer{},
		Editor:       " ",
	})
	if err == nil {
		t.Fatal("expected error for missing template")
	}
}
