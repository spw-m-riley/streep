package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestPerformShowsHelp(t *testing.T) {
	var out bytes.Buffer
	if err := executePerform([]string{"--help"}, &out, &bytes.Buffer{}); err != nil {
		t.Fatalf("executePerform() error: %v", err)
	}
	if !strings.Contains(out.String(), "streep perform") {
		t.Fatalf("expected help text, got: %q", out.String())
	}
}

func TestPerformWhenNoActrc(t *testing.T) {
	orig, _ := os.Getwd()
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(orig) //nolint:errcheck

	var out bytes.Buffer
	if err := executePerform(nil, &out, &bytes.Buffer{}); err != nil {
		t.Fatalf("executePerform() error: %v", err)
	}
	if !strings.Contains(out.String(), ".actrc not found") {
		t.Fatalf("expected .actrc warning, got: %q", out.String())
	}
}

func TestPerformAddsEventPayloadAndPassesFlags(t *testing.T) {
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

	writeCheckFile(t, filepath.Join(dir, ".actrc"), "--secret-file .secrets\n")
	if err := os.MkdirAll(filepath.Join(dir, ".act", "events"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeCheckFile(t, filepath.Join(dir, ".act", "events", "pull_request.json"), "{}\n")

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
	err := executePerform([]string{"pull_request", "--job", "build", "--workflow", ".github/workflows/ci.yml"}, &out, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("executePerform() error: %v", err)
	}

	raw, err := os.ReadFile(actArgs)
	if err != nil {
		t.Fatalf("read fake act args: %v", err)
	}
	got := string(raw)
	for _, want := range []string{"-j", "build", "-W", ".github/workflows/ci.yml", "pull_request", "-e", ".act/events/pull_request.json"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected arg %q in fake act invocation, got:\n%s", want, got)
		}
	}
}
