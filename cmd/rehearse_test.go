package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestRehearsShowsHelp(t *testing.T) {
	var out bytes.Buffer
	if err := executeRehearse([]string{"--help"}, &out, &bytes.Buffer{}); err != nil {
		t.Fatalf("executeRehearse() error: %v", err)
	}
	if !strings.Contains(out.String(), "streep rehearse") {
		t.Errorf("expected help text, got: %q", out.String())
	}
}

func TestRehearsWhenNoActrc(t *testing.T) {
	// Run from a temp dir that has no .actrc
	orig, _ := os.Getwd()
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(orig) //nolint:errcheck

	var out bytes.Buffer
	if err := executeRehearse(nil, &out, &bytes.Buffer{}); err != nil {
		t.Fatalf("executeRehearse() error: %v", err)
	}
	if !strings.Contains(out.String(), ".actrc not found") {
		t.Errorf("expected .actrc not found message, got: %q", out.String())
	}
}
