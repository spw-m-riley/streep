package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestHookShowsHelp(t *testing.T) {
	var out bytes.Buffer
	if err := executeHook([]string{"--help"}, &out, &bytes.Buffer{}); err != nil {
		t.Fatalf("executeHook() error: %v", err)
	}
	if !strings.Contains(out.String(), "streep hook") {
		t.Fatalf("expected help text, got: %q", out.String())
	}
}

func TestHookInstallAndUninstall(t *testing.T) {
	dir := t.TempDir()

	var out bytes.Buffer
	if err := executeHook([]string{"install", dir}, &out, &bytes.Buffer{}); err != nil {
		t.Fatalf("executeHook(install) error: %v", err)
	}
	if !strings.Contains(out.String(), "Installed") {
		t.Fatalf("expected install output, got:\n%s", out.String())
	}

	out.Reset()
	if err := executeHook([]string{"uninstall", dir}, &out, &bytes.Buffer{}); err != nil {
		t.Fatalf("executeHook(uninstall) error: %v", err)
	}
	if !strings.Contains(out.String(), "Removed") {
		t.Fatalf("expected uninstall output, got:\n%s", out.String())
	}
}
