package bundle

import (
	"archive/zip"
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBundleActionsDownloadsArchivesAndWritesLock(t *testing.T) {
	dir := t.TempDir()
	wfDir := filepath.Join(dir, ".github", "workflows")
	if err := os.MkdirAll(wfDir, 0o755); err != nil {
		t.Fatalf("mkdir workflows: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wfDir, "ci.yml"), []byte(`
on: [push]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: ./local-action
`), 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}

	sha := strings.Repeat("a", 40)
	archive := zipArchive(t, "checkout-"+sha+"/action.yml", "name: checkout\n")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/repos/actions/checkout/commits/v4":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"sha":"` + sha + `"}`))
		case r.URL.Path == "/actions/checkout/zip/"+sha:
			w.Header().Set("Content-Type", "application/zip")
			w.Write(archive)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	oldAPI := githubAPIBaseURL
	oldCode := codeloadBaseURL
	oldClient := defaultHTTPClient
	githubAPIBaseURL = server.URL
	codeloadBaseURL = server.URL
	defaultHTTPClient = server.Client()
	defer func() {
		githubAPIBaseURL = oldAPI
		codeloadBaseURL = oldCode
		defaultHTTPClient = oldClient
	}()

	result, err := BundleActions(Options{RepoDir: dir})
	if err != nil {
		t.Fatalf("BundleActions() error: %v", err)
	}
	if len(result.Entries) != 1 {
		t.Fatalf("expected one bundled action, got %d", len(result.Entries))
	}
	if result.Entries[0].Ref != "actions/checkout@v4" {
		t.Fatalf("unexpected ref: %+v", result.Entries[0])
	}
	if result.Entries[0].SHA != sha {
		t.Fatalf("unexpected SHA: %+v", result.Entries[0])
	}

	actionFile := filepath.Join(dir, ".act", "bundle", "actions", "checkout@"+sha, "action.yml")
	if _, err := os.Stat(actionFile); err != nil {
		t.Fatalf("expected extracted action file, stat error: %v", err)
	}

	lockData, err := os.ReadFile(filepath.Join(dir, ".act", "bundle.lock"))
	if err != nil {
		t.Fatalf("read lock file: %v", err)
	}
	if !strings.Contains(string(lockData), "actions/checkout@v4") {
		t.Fatalf("expected lock file to contain action ref, got:\n%s", lockData)
	}
}

func TestBundleActionsNoRemoteActions(t *testing.T) {
	dir := t.TempDir()
	wfDir := filepath.Join(dir, ".github", "workflows")
	if err := os.MkdirAll(wfDir, 0o755); err != nil {
		t.Fatalf("mkdir workflows: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wfDir, "ci.yml"), []byte(`
on: [push]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: ./local-action
`), 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}

	result, err := BundleActions(Options{RepoDir: dir})
	if err != nil {
		t.Fatalf("BundleActions() error: %v", err)
	}
	if len(result.Entries) != 0 {
		t.Fatalf("expected no entries, got %d", len(result.Entries))
	}
	if _, err := os.Stat(filepath.Join(dir, ".act", "bundle.lock")); err != nil {
		t.Fatalf("expected bundle.lock to exist, stat error: %v", err)
	}
}

func zipArchive(t *testing.T, path, content string) []byte {
	t.Helper()
	var b bytes.Buffer
	zw := zip.NewWriter(&b)
	w, err := zw.Create(path)
	if err != nil {
		t.Fatalf("zip create: %v", err)
	}
	if _, err := w.Write([]byte(content)); err != nil {
		t.Fatalf("zip write: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zip close: %v", err)
	}
	return b.Bytes()
}
