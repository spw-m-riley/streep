package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDoctorShowsHelp(t *testing.T) {
	var out bytes.Buffer
	if err := executeDoctor([]string{"--help"}, &out, &bytes.Buffer{}); err != nil {
		t.Fatalf("executeDoctor() error: %v", err)
	}
	if !strings.Contains(out.String(), "streep doctor") {
		t.Fatalf("expected help text, got: %q", out.String())
	}
}

func TestDoctorHealthyProject(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script test helper is unix-only")
	}

	dir := t.TempDir()
	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	setupFakeActAndDocker(t, binDir, false)

	oldPath := os.Getenv("PATH")
	if err := os.Setenv("PATH", binDir+":"+oldPath); err != nil {
		t.Fatalf("setenv PATH: %v", err)
	}
	defer os.Setenv("PATH", oldPath) //nolint:errcheck

	writeDoctorFixture(t, dir, true)

	var out bytes.Buffer
	if err := executeDoctor([]string{dir}, &out, &bytes.Buffer{}); err != nil {
		t.Fatalf("executeDoctor() error: %v", err)
	}

	got := out.String()
	for _, want := range []string{
		"✔ act:",
		"✔ docker:",
		"✔ config: .actrc present",
		"✔ secrets:",
		"✔ env:",
		"✔ vars:",
		"✔ events:",
		"✔ artifacts:",
		"All checks passed.",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in output, got:\n%s", want, got)
		}
	}
}

func TestDoctorReportsArtifactIssue(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script test helper is unix-only")
	}

	dir := t.TempDir()
	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	setupFakeActAndDocker(t, binDir, false)

	oldPath := os.Getenv("PATH")
	if err := os.Setenv("PATH", binDir+":"+oldPath); err != nil {
		t.Fatalf("setenv PATH: %v", err)
	}
	defer os.Setenv("PATH", oldPath) //nolint:errcheck

	writeDoctorFixture(t, dir, false)

	var out bytes.Buffer
	if err := executeDoctor([]string{dir}, &out, &bytes.Buffer{}); err != nil {
		t.Fatalf("executeDoctor() error: %v", err)
	}
	if !strings.Contains(out.String(), "artifacts: upload/download-artifact is used but .artifacts/ is missing") {
		t.Fatalf("expected artifact issue, got:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "issue(s) found") {
		t.Fatalf("expected issue summary, got:\n%s", out.String())
	}
}

func TestDoctorJSONOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script test helper is unix-only")
	}

	dir := t.TempDir()
	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	setupFakeActAndDocker(t, binDir, false)

	oldPath := os.Getenv("PATH")
	if err := os.Setenv("PATH", binDir+":"+oldPath); err != nil {
		t.Fatalf("setenv PATH: %v", err)
	}
	defer os.Setenv("PATH", oldPath) //nolint:errcheck

	writeDoctorFixture(t, dir, true)

	var out bytes.Buffer
	if err := executeDoctor([]string{dir, "--json"}, &out, &bytes.Buffer{}); err != nil {
		t.Fatalf("executeDoctor() error: %v", err)
	}
	var payload commandJSONResult
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("json unmarshal: %v\n%s", err, out.String())
	}
	if !payload.OK {
		t.Fatalf("expected ok=true payload, got %+v", payload)
	}
	if !strings.Contains(payload.Output, "All checks passed.") {
		t.Fatalf("expected wrapped doctor output, got %+v", payload)
	}
}

func setupFakeActAndDocker(t *testing.T, dir string, dockerInfoShouldFail bool) {
	t.Helper()

	actScript := "#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then\n  echo \"act version 0.2.75\"\n  exit 0\nfi\nexit 0\n"
	writeCheckFile(t, filepath.Join(dir, "act"), actScript)
	if err := os.Chmod(filepath.Join(dir, "act"), 0o755); err != nil {
		t.Fatalf("chmod act: %v", err)
	}

	dockerScript := "#!/bin/sh\nif [ \"$1\" = \"info\" ]; then\n  exit 0\nfi\nif [ \"$1\" = \"--version\" ]; then\n  echo \"Docker version 26.1.0\"\n  exit 0\nfi\nexit 0\n"
	if dockerInfoShouldFail {
		dockerScript = "#!/bin/sh\nif [ \"$1\" = \"info\" ]; then\n  echo \"daemon unavailable\" 1>&2\n  exit 1\nfi\nif [ \"$1\" = \"--version\" ]; then\n  echo \"Docker version 26.1.0\"\n  exit 0\nfi\nexit 0\n"
	}
	writeCheckFile(t, filepath.Join(dir, "docker"), dockerScript)
	if err := os.Chmod(filepath.Join(dir, "docker"), 0o755); err != nil {
		t.Fatalf("chmod docker: %v", err)
	}
}

func writeDoctorFixture(t *testing.T, dir string, withArtifactsDir bool) {
	t.Helper()

	writeCheckFile(t, filepath.Join(dir, ".actrc"), "--secret-file .secrets\n")
	writeCheckFile(t, filepath.Join(dir, ".secrets.example"), "GITHUB_TOKEN=\nAPI_KEY=\n")
	writeCheckFile(t, filepath.Join(dir, ".secrets"), "GITHUB_TOKEN=x\nAPI_KEY=y\n")
	writeCheckFile(t, filepath.Join(dir, ".env.example"), "APP_ENV=\n")
	writeCheckFile(t, filepath.Join(dir, ".env"), "APP_ENV=local\n")
	writeCheckFile(t, filepath.Join(dir, ".vars.example"), "CHANNEL=\n")
	writeCheckFile(t, filepath.Join(dir, ".vars"), "CHANNEL=dev\n")

	if err := os.MkdirAll(filepath.Join(dir, ".act", "events"), 0o755); err != nil {
		t.Fatalf("mkdir events: %v", err)
	}
	writeCheckFile(t, filepath.Join(dir, ".act", "events", "push.json"), "{}\n")

	wfDir := filepath.Join(dir, ".github", "workflows")
	if err := os.MkdirAll(wfDir, 0o755); err != nil {
		t.Fatalf("mkdir workflows: %v", err)
	}
	writeCheckFile(t, filepath.Join(wfDir, "ci.yml"), `
on:
  push:
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/upload-artifact@v4
        with:
          name: out
          path: out/
`)

	if withArtifactsDir {
		if err := os.MkdirAll(filepath.Join(dir, ".artifacts"), 0o755); err != nil {
			t.Fatalf("mkdir .artifacts: %v", err)
		}
	}
}
