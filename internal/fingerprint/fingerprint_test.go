package fingerprint

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildIsDeterministic(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".github", "workflows"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".github", "workflows", "ci.yml"), []byte("on: [push]\n"), 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".actrc"), []byte("--secret-file .secrets\n"), 0o644); err != nil {
		t.Fatalf("write .actrc: %v", err)
	}

	a, err := Build(dir)
	if err != nil {
		t.Fatalf("Build() error: %v", err)
	}
	b, err := Build(dir)
	if err != nil {
		t.Fatalf("Build() second error: %v", err)
	}

	if a.Digest != b.Digest {
		t.Fatalf("expected deterministic digest, got %s vs %s", a.Digest, b.Digest)
	}
}

func TestWriteAndLoad(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".github", "workflows"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".github", "workflows", "ci.yml"), []byte("on: [push]\n"), 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}

	written, path, err := WriteCurrent(dir)
	if err != nil {
		t.Fatalf("WriteCurrent() error: %v", err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if written.Digest != loaded.Digest {
		t.Fatalf("expected loaded digest %s, got %s", written.Digest, loaded.Digest)
	}
}
