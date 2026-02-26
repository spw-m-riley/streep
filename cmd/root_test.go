package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteShowsRootHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := Execute(nil, &stdout, &stderr); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if !strings.Contains(stdout.String(), "streep <command>") {
		t.Fatalf("expected root usage, got: %q", stdout.String())
	}
}

func TestExecuteRunsNewRole(t *testing.T) {
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := Execute([]string{"new", "role", dir}, &stdout, &stderr); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	for _, rel := range []string{".secrets.example", ".env.example", ".vars.example", ".actrc"} {
		path := filepath.Join(dir, filepath.FromSlash(rel))
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s to exist: %v", path, err)
		}
	}
}

func TestExecuteUnknownCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Execute([]string{"unknown"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for unknown command")
	}

	if !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecuteTopLevelCommandsRegistered(t *testing.T) {
	cases := []struct {
		args []string
		want string
	}{
		{args: []string{"perform", "--help"}, want: "streep perform"},
		{args: []string{"clean", "--help"}, want: "streep clean"},
		{args: []string{"doctor", "--help"}, want: "streep doctor"},
		{args: []string{"edit", "--help"}, want: "streep edit"},
		{args: []string{"explain", "--help"}, want: "streep explain"},
		{args: []string{"lint", "--help"}, want: "streep lint"},
		{args: []string{"bundle", "--help"}, want: "streep bundle"},
		{args: []string{"hook", "--help"}, want: "streep hook"},
		{args: []string{"diff", "--help"}, want: "streep diff"},
		{args: []string{"fingerprint", "--help"}, want: "streep fingerprint"},
		{args: []string{"policy", "--help"}, want: "streep policy"},
		{args: []string{"audit", "--help"}, want: "streep audit"},
		{args: []string{"diagnose", "--help"}, want: "streep diagnose"},
	}

	for _, tc := range cases {
		var out bytes.Buffer
		err := Execute(tc.args, &out, &bytes.Buffer{})
		if err != nil {
			t.Fatalf("Execute(%v) error: %v", tc.args, err)
		}
		if !strings.Contains(out.String(), tc.want) {
			t.Fatalf("expected %q in output for args %v, got:\n%s", tc.want, tc.args, out.String())
		}
	}
}
