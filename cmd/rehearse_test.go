package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
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

func TestRehearsUnknownFlagReturnsError(t *testing.T) {
	var out bytes.Buffer
	if err := executeRehearse([]string{"--unknown"}, &out, &bytes.Buffer{}); err == nil {
		t.Fatal("expected error for unknown flag, got nil")
	}
}

func TestRehearsUsesConfigDefaultsAndPassthrough(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script test helper is unix-only")
	}

	origWD, _ := os.Getwd()
	origPath := os.Getenv("PATH")
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(origWD)            //nolint:errcheck
	defer os.Setenv("PATH", origPath) //nolint:errcheck

	if err := os.MkdirAll(filepath.Join(dir, ".streep"), 0o755); err != nil {
		t.Fatalf("mkdir .streep: %v", err)
	}
	writeCheckFile(t, filepath.Join(dir, ".streep", "config.yaml"), `
defaults:
  event: pull_request
  job: build
  workflow: .github/workflows/ci.yml
`)
	writeCheckFile(t, filepath.Join(dir, ".actrc"), "--secret-file .secrets\n")

	actArgs := filepath.Join(dir, "act-args.txt")
	fakeAct := filepath.Join(dir, "act")
	writeCheckFile(t, fakeAct, "#!/bin/sh\nprintf '%s\\n' \"$@\" > \""+actArgs+"\"\n")
	if err := os.Chmod(fakeAct, 0o755); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	if err := os.Setenv("PATH", dir+":"+origPath); err != nil {
		t.Fatalf("setenv PATH: %v", err)
	}

	var out bytes.Buffer
	if err := executeRehearse([]string{"--", "--verbose"}, &out, &bytes.Buffer{}); err != nil {
		t.Fatalf("executeRehearse() error: %v", err)
	}

	raw, err := os.ReadFile(actArgs)
	if err != nil {
		t.Fatalf("read fake act args: %v", err)
	}
	got := string(raw)
	for _, want := range []string{"-n", "-j", "build", "-W", ".github/workflows/ci.yml", "pull_request", "--verbose"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected arg %q in fake act invocation, got:\n%s", want, got)
		}
	}
}
