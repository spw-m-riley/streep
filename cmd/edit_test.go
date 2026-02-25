package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestEditShowsHelp(t *testing.T) {
	var out bytes.Buffer
	if err := executeEdit([]string{"--help"}, &out, &bytes.Buffer{}); err != nil {
		t.Fatalf("executeEdit() error: %v", err)
	}
	if !strings.Contains(out.String(), "streep edit") {
		t.Fatalf("expected help text, got: %q", out.String())
	}
}

func TestEditUnknownTarget(t *testing.T) {
	err := executeEdit([]string{"unknown"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected unknown target error")
	}
}

func TestEditWithEditor(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script helper is unix-only")
	}

	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, ".env.example"), "APP_ENV=\n")
	editorPath := filepath.Join(dir, "fake-editor")
	writeCheckFile(t, editorPath, "#!/bin/sh\necho \"APP_ENV=local\" > \"$1\"\n")
	if err := os.Chmod(editorPath, 0o755); err != nil {
		t.Fatalf("chmod: %v", err)
	}

	oldEditor := os.Getenv("EDITOR")
	if err := os.Setenv("EDITOR", editorPath); err != nil {
		t.Fatalf("setenv EDITOR: %v", err)
	}
	defer os.Setenv("EDITOR", oldEditor) //nolint:errcheck

	var out bytes.Buffer
	if err := executeEdit([]string{"env", dir}, &out, &bytes.Buffer{}); err != nil {
		t.Fatalf("executeEdit() error: %v", err)
	}
	if !strings.Contains(out.String(), "Updated .env") {
		t.Fatalf("expected success output, got:\n%s", out.String())
	}
}

func TestEditValidationFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script helper is unix-only")
	}

	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, ".vars.example"), "CHANNEL=\n")
	editorPath := filepath.Join(dir, "fake-editor")
	writeCheckFile(t, editorPath, "#!/bin/sh\necho \"CHANNEL=\" > \"$1\"\n")
	if err := os.Chmod(editorPath, 0o755); err != nil {
		t.Fatalf("chmod: %v", err)
	}

	oldEditor := os.Getenv("EDITOR")
	if err := os.Setenv("EDITOR", editorPath); err != nil {
		t.Fatalf("setenv EDITOR: %v", err)
	}
	defer os.Setenv("EDITOR", oldEditor) //nolint:errcheck

	err := executeEdit([]string{"vars", dir}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "missing or empty values") {
		t.Fatalf("unexpected error: %v", err)
	}
}
