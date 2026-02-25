package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestCompletionBash(t *testing.T) {
	var out bytes.Buffer
	if err := executeCompletion([]string{"bash"}, &out, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "complete -F _streep streep") {
		t.Errorf("expected bash completion, got: %q", out.String())
	}
}

func TestCompletionZsh(t *testing.T) {
	var out bytes.Buffer
	if err := executeCompletion([]string{"zsh"}, &out, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "#compdef streep") {
		t.Errorf("expected zsh completion, got: %q", out.String())
	}
}

func TestCompletionFish(t *testing.T) {
	var out bytes.Buffer
	if err := executeCompletion([]string{"fish"}, &out, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "complete -c streep") {
		t.Errorf("expected fish completion, got: %q", out.String())
	}
}

func TestCompletionUnknownShell(t *testing.T) {
	err := executeCompletion([]string{"powershell"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "unknown shell") {
		t.Errorf("expected unknown shell error, got: %v", err)
	}
}

func TestCompletionHelp(t *testing.T) {
	var out bytes.Buffer
	if err := executeCompletion([]string{"--help"}, &out, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "bash") {
		t.Errorf("expected usage in output, got: %q", out.String())
	}
}
