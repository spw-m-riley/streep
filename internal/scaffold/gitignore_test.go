package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureGitignoreEntriesCreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".gitignore")

	if err := EnsureGitignoreEntries(path); err != nil {
		t.Fatalf("EnsureGitignoreEntries() error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read .gitignore: %v", err)
	}
	content := string(data)
	for _, entry := range []string{".secrets", ".env", ".vars"} {
		if !strings.Contains(content, entry) {
			t.Errorf("expected %q in .gitignore, got:\n%s", entry, content)
		}
	}
}

func TestEnsureGitignoreEntriesAppendsToExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".gitignore")
	if err := os.WriteFile(path, []byte("node_modules\ndist\n"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	if err := EnsureGitignoreEntries(path); err != nil {
		t.Fatalf("EnsureGitignoreEntries() error: %v", err)
	}

	data, _ := os.ReadFile(path)
	content := string(data)

	if !strings.Contains(content, "node_modules") {
		t.Error("existing content was removed")
	}
	for _, entry := range []string{".secrets", ".env", ".vars"} {
		if !strings.Contains(content, entry) {
			t.Errorf("expected %q in .gitignore", entry)
		}
	}
}

func TestEnsureGitignoreEntriesIdempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".gitignore")
	existing := ".secrets\n.env\n.vars\n"
	if err := os.WriteFile(path, []byte(existing), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	if err := EnsureGitignoreEntries(path); err != nil {
		t.Fatalf("EnsureGitignoreEntries() error: %v", err)
	}

	data, _ := os.ReadFile(path)
	// Should not have duplicated any entries
	count := strings.Count(string(data), ".secrets")
	if count != 1 {
		t.Errorf("expected .secrets to appear once, got %d times", count)
	}
}
