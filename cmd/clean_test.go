package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCleanShowsHelp(t *testing.T) {
	var out bytes.Buffer
	if err := executeClean([]string{"--help"}, &out, &bytes.Buffer{}); err != nil {
		t.Fatalf("executeClean() error: %v", err)
	}
	if !strings.Contains(out.String(), "streep clean") {
		t.Fatalf("expected help text, got: %q", out.String())
	}
}

func TestCleanDryRun(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, ".secrets"), "x=y\n")
	writeCheckFile(t, filepath.Join(dir, ".actrc"), "--secret-file .secrets\n")
	if err := os.MkdirAll(filepath.Join(dir, ".artifacts"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeCheckFile(t, filepath.Join(dir, ".artifacts", "a.txt"), "data\n")

	var out bytes.Buffer
	if err := executeClean([]string{dir}, &out, &bytes.Buffer{}); err != nil {
		t.Fatalf("executeClean() error: %v", err)
	}
	if !strings.Contains(out.String(), "Dry-run") {
		t.Fatalf("expected dry-run output, got:\n%s", out.String())
	}
	if _, err := os.Stat(filepath.Join(dir, ".secrets")); err != nil {
		t.Fatalf("expected file to remain after dry-run: %v", err)
	}
}

func TestCleanForceRemovesTargets(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, ".secrets"), "x=y\n")
	writeCheckFile(t, filepath.Join(dir, ".env"), "x=y\n")
	writeCheckFile(t, filepath.Join(dir, ".vars"), "x=y\n")
	writeCheckFile(t, filepath.Join(dir, ".input"), "x=y\n")
	writeCheckFile(t, filepath.Join(dir, ".actrc"), "--secret-file .secrets\n")
	if err := os.MkdirAll(filepath.Join(dir, ".artifacts"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeCheckFile(t, filepath.Join(dir, ".artifacts", "a.txt"), "data\n")
	if err := os.MkdirAll(filepath.Join(dir, ".act", "cache"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeCheckFile(t, filepath.Join(dir, ".act", "cache", "cached.txt"), "data\n")

	var out bytes.Buffer
	if err := executeClean([]string{dir, "--force"}, &out, &bytes.Buffer{}); err != nil {
		t.Fatalf("executeClean() error: %v", err)
	}
	if !strings.Contains(out.String(), "Removed:") {
		t.Fatalf("expected removed output, got:\n%s", out.String())
	}

	for _, rel := range []string{".secrets", ".env", ".vars", ".input", ".actrc"} {
		if _, err := os.Stat(filepath.Join(dir, rel)); !os.IsNotExist(err) {
			t.Fatalf("expected %s to be removed, stat err: %v", rel, err)
		}
	}

	artifactsEntries, err := os.ReadDir(filepath.Join(dir, ".artifacts"))
	if err != nil {
		t.Fatalf("readdir .artifacts: %v", err)
	}
	if len(artifactsEntries) != 0 {
		t.Fatalf("expected .artifacts contents to be removed, got %d entries", len(artifactsEntries))
	}

	cacheEntries, err := os.ReadDir(filepath.Join(dir, ".act", "cache"))
	if err != nil {
		t.Fatalf("readdir .act/cache: %v", err)
	}
	if len(cacheEntries) != 0 {
		t.Fatalf("expected .act/cache contents to be removed, got %d entries", len(cacheEntries))
	}
}

func TestCleanNothingToClean(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	if err := executeClean([]string{dir}, &out, &bytes.Buffer{}); err != nil {
		t.Fatalf("executeClean() error: %v", err)
	}
	if !strings.Contains(out.String(), "Nothing to clean") {
		t.Fatalf("expected nothing-to-clean output, got:\n%s", out.String())
	}
}

func TestCleanUnknownFlagReturnsError(t *testing.T) {
	var out bytes.Buffer
	if err := executeClean([]string{"--unknown"}, &out, &bytes.Buffer{}); err == nil {
		t.Fatal("expected error for unknown flag, got nil")
	}
}
