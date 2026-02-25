package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestBundleShowsHelp(t *testing.T) {
	var out bytes.Buffer
	if err := executeBundle([]string{"--help"}, &out, &bytes.Buffer{}); err != nil {
		t.Fatalf("executeBundle() error: %v", err)
	}
	if !strings.Contains(out.String(), "streep bundle") {
		t.Fatalf("expected help text, got: %q", out.String())
	}
}

func TestBundleUnknownSubcommand(t *testing.T) {
	err := executeBundle([]string{"unknown"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error for unknown subcommand")
	}
}

func TestBundleActionsNoRemoteActions(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	if err := executeBundleActions([]string{dir}, &out, &bytes.Buffer{}); err != nil {
		t.Fatalf("executeBundleActions() error: %v", err)
	}
	if !strings.Contains(out.String(), "No remote workflow actions found to bundle.") {
		t.Fatalf("expected no-actions output, got:\n%s", out.String())
	}
}
