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
	githubAPIBaseURL = server.URL
	codeloadBaseURL = server.URL
	defer func() {
		githubAPIBaseURL = oldAPI
		codeloadBaseURL = oldCode
	}()

	result, err := BundleActions(Options{RepoDir: dir, HTTPClient: server.Client()})
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

func TestResolveSHAHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusBadGateway)
	}))
	defer server.Close()

	oldAPI := githubAPIBaseURL
	githubAPIBaseURL = server.URL
	defer func() { githubAPIBaseURL = oldAPI }()

	_, err := resolveSHA(server.Client(), "actions", "checkout", "v4", "")
	if err == nil {
		t.Fatal("expected resolveSHA() to return error, got nil")
	}
	if !strings.Contains(err.Error(), "github API returned 502") {
		t.Fatalf("expected github API status error, got: %v", err)
	}
}

func TestDownloadAndExtractHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "missing", http.StatusNotFound)
	}))
	defer server.Close()

	oldCode := codeloadBaseURL
	codeloadBaseURL = server.URL
	defer func() { codeloadBaseURL = oldCode }()

	err := downloadAndExtract(server.Client(), "actions", "checkout", strings.Repeat("a", 40), t.TempDir(), "")
	if err == nil {
		t.Fatal("expected downloadAndExtract() to return error, got nil")
	}
	if !strings.Contains(err.Error(), "codeload returned 404") {
		t.Fatalf("expected codeload status error, got: %v", err)
	}
}

func TestVerifyLockDetectsMissingAndExtra(t *testing.T) {
	dir := t.TempDir()
	wfDir := filepath.Join(dir, ".github", "workflows")
	if err := os.MkdirAll(wfDir, 0o755); err != nil {
		t.Fatalf("mkdir workflows: %v", err)
	}
	sha := strings.Repeat("a", 40)
	writeWorkflow := `
on: [push]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@` + sha + `
`
	if err := os.WriteFile(filepath.Join(wfDir, "ci.yml"), []byte(writeWorkflow), 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".act"), 0o755); err != nil {
		t.Fatalf("mkdir .act: %v", err)
	}
	lock := "actions:\n  - ref: actions/setup-go@" + sha + "\n    sha: " + sha + "\n    path: .act/bundle/actions/setup-go@" + sha + "\n"
	if err := os.WriteFile(filepath.Join(dir, ".act", "bundle.lock"), []byte(lock), 0o644); err != nil {
		t.Fatalf("write lock: %v", err)
	}

	result, err := VerifyLock(Options{RepoDir: dir})
	if err != nil {
		t.Fatalf("VerifyLock() error: %v", err)
	}
	if len(result.Missing) != 1 || result.Missing[0] != "actions/checkout@"+sha {
		t.Fatalf("unexpected missing refs: %+v", result.Missing)
	}
	if len(result.Extra) != 1 || result.Extra[0] != "actions/setup-go@"+sha {
		t.Fatalf("unexpected extra refs: %+v", result.Extra)
	}
}

func TestVerifyLockDetectsStaleEntry(t *testing.T) {
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
`), 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".act"), 0o755); err != nil {
		t.Fatalf("mkdir .act: %v", err)
	}
	oldSHA := strings.Repeat("a", 40)
	newSHA := strings.Repeat("b", 40)
	lock := "actions:\n  - ref: actions/checkout@v4\n    sha: " + oldSHA + "\n    path: .act/bundle/actions/checkout@" + oldSHA + "\n"
	if err := os.WriteFile(filepath.Join(dir, ".act", "bundle.lock"), []byte(lock), 0o644); err != nil {
		t.Fatalf("write lock: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/actions/checkout/commits/v4" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"sha":"` + newSHA + `"}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	oldAPI := githubAPIBaseURL
	githubAPIBaseURL = server.URL
	defer func() { githubAPIBaseURL = oldAPI }()

	result, err := VerifyLock(Options{RepoDir: dir, HTTPClient: server.Client()})
	if err != nil {
		t.Fatalf("VerifyLock() error: %v", err)
	}
	if len(result.Stale) != 1 {
		t.Fatalf("expected one stale entry, got %+v", result.Stale)
	}
	if result.Stale[0].Ref != "actions/checkout@v4" || result.Stale[0].LockedSHA != oldSHA || result.Stale[0].ResolvedSHA != newSHA {
		t.Fatalf("unexpected stale payload: %+v", result.Stale[0])
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
