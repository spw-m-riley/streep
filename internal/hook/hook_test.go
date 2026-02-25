package hook

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallWritesHooks(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}

	n, err := Install(dir)
	if err != nil {
		t.Fatalf("Install() error: %v", err)
	}
	if n != 2 {
		t.Fatalf("expected 2 hooks written, got %d", n)
	}

	for _, name := range []string{"pre-commit", "pre-push"} {
		path := filepath.Join(dir, ".git", "hooks", name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if !strings.Contains(string(data), managedMarker) {
			t.Fatalf("expected marker in %s", name)
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat %s: %v", name, err)
		}
		if info.Mode()&0o111 == 0 {
			t.Fatalf("expected executable bit set for %s", name)
		}
	}
}

func TestUninstallRemovesManagedHooksOnly(t *testing.T) {
	dir := t.TempDir()
	hooksDir := filepath.Join(dir, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatalf("mkdir hooks: %v", err)
	}

	if err := os.WriteFile(filepath.Join(hooksDir, "pre-commit"), []byte(preCommitScript), 0o755); err != nil {
		t.Fatalf("write pre-commit: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hooksDir, "pre-push"), []byte(prePushScript), 0o755); err != nil {
		t.Fatalf("write pre-push: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hooksDir, "post-commit"), []byte("#!/bin/sh\necho custom\n"), 0o755); err != nil {
		t.Fatalf("write post-commit: %v", err)
	}

	removed, err := Uninstall(dir)
	if err != nil {
		t.Fatalf("Uninstall() error: %v", err)
	}
	if removed != 2 {
		t.Fatalf("expected 2 hooks removed, got %d", removed)
	}

	if _, err := os.Stat(filepath.Join(hooksDir, "pre-commit")); !os.IsNotExist(err) {
		t.Fatalf("expected pre-commit to be removed, stat err: %v", err)
	}
	if _, err := os.Stat(filepath.Join(hooksDir, "pre-push")); !os.IsNotExist(err) {
		t.Fatalf("expected pre-push to be removed, stat err: %v", err)
	}
	if _, err := os.Stat(filepath.Join(hooksDir, "post-commit")); err != nil {
		t.Fatalf("expected unmanaged hook to remain, stat err: %v", err)
	}
}
