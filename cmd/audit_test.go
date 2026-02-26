package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAuditPasses(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{".secrets", ".env", ".vars", ".input"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x=y\n"), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(".secrets\n.env\n.vars\n.input\n"), 0o644); err != nil {
		t.Fatalf("write .gitignore: %v", err)
	}

	var out bytes.Buffer
	if err := executeAudit([]string{dir}, &out, &bytes.Buffer{}); err != nil {
		t.Fatalf("executeAudit() error: %v", err)
	}
	if !strings.Contains(out.String(), "Audit passed.") {
		t.Fatalf("expected pass message, got:\n%s", out.String())
	}
}

func TestAuditFindsIssues(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".secrets"), []byte("x=y\n"), 0o644); err != nil {
		t.Fatalf("write .secrets: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(".secrets\n.env\n"), 0o644); err != nil {
		t.Fatalf("write .gitignore: %v", err)
	}

	var out bytes.Buffer
	err := executeAudit([]string{dir}, &out, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected audit failure, got nil")
	}
	if !strings.Contains(out.String(), "Audit found") {
		t.Fatalf("expected issue summary, got:\n%s", out.String())
	}
}

func TestAuditJSONOutput(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(".secrets\n.env\n.vars\n.input\n"), 0o644); err != nil {
		t.Fatalf("write .gitignore: %v", err)
	}

	var out bytes.Buffer
	if err := executeAudit([]string{dir, "--json"}, &out, &bytes.Buffer{}); err != nil {
		t.Fatalf("executeAudit() error: %v", err)
	}
	var payload commandJSONResult
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("json unmarshal: %v\n%s", err, out.String())
	}
	if !payload.OK {
		t.Fatalf("expected ok=true payload, got %+v", payload)
	}
}
