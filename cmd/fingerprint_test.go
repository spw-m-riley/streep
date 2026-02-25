package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFingerprintShowsHelp(t *testing.T) {
	var out bytes.Buffer
	if err := executeFingerprint([]string{"--help"}, &out, &bytes.Buffer{}); err != nil {
		t.Fatalf("executeFingerprint() error: %v", err)
	}
	if !strings.Contains(out.String(), "streep fingerprint") {
		t.Fatalf("expected help output, got: %q", out.String())
	}
}

func TestFingerprintWritesFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".github", "workflows"), 0o755); err != nil {
		t.Fatalf("mkdir workflows: %v", err)
	}
	writeCheckFile(t, filepath.Join(dir, ".github", "workflows", "ci.yml"), "on: [push]\n")

	var out bytes.Buffer
	if err := executeFingerprint([]string{dir}, &out, &bytes.Buffer{}); err != nil {
		t.Fatalf("executeFingerprint() error: %v", err)
	}
	if !strings.Contains(out.String(), "Fingerprint:") {
		t.Fatalf("expected fingerprint output, got:\n%s", out.String())
	}
	if _, err := os.Stat(filepath.Join(dir, ".act", "run-fingerprint")); err != nil {
		t.Fatalf("expected run-fingerprint file, stat error: %v", err)
	}
}

func TestFingerprintCompare(t *testing.T) {
	dir := t.TempDir()
	pathA := filepath.Join(dir, "a.json")
	pathB := filepath.Join(dir, "b.json")
	writeCheckFile(t, pathA, `{"version":1,"platform":"x/y","digest":"abc","files":[]}`+"\n")
	writeCheckFile(t, pathB, `{"version":1,"platform":"x/y","digest":"abc","files":[]}`+"\n")

	var out bytes.Buffer
	if err := executeFingerprint([]string{"compare", pathA, pathB}, &out, &bytes.Buffer{}); err != nil {
		t.Fatalf("executeFingerprint(compare) error: %v", err)
	}
	if !strings.Contains(out.String(), "Fingerprints match") {
		t.Fatalf("expected match output, got:\n%s", out.String())
	}
}
