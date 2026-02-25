package bundle

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"streep/internal/workflow"
)

var (
	shaPattern        = regexp.MustCompile(`^[a-f0-9]{40}$`)
	githubAPIBaseURL  = "https://api.github.com"
	codeloadBaseURL   = "https://codeload.github.com"
	defaultHTTPClient = &http.Client{Timeout: 30 * time.Second}
)

// Options configures action bundling.
type Options struct {
	RepoDir string
	Token   string
}

// Entry describes one bundled action.
type Entry struct {
	Ref string `yaml:"ref"`
	SHA string `yaml:"sha"`
	// Path is a repository-relative path to the local bundle copy.
	Path string `yaml:"path"`
}

// Result describes the completed bundle operation.
type Result struct {
	Entries  []Entry
	LockPath string
}

// BundleActions downloads and locks all remote actions referenced by workflow files.
func BundleActions(opts Options) (Result, error) {
	repoDir := opts.RepoDir
	if repoDir == "" {
		repoDir = "."
	}
	token := strings.TrimSpace(opts.Token)
	if token == "" {
		token = strings.TrimSpace(os.Getenv("GITHUB_TOKEN"))
	}

	refs, err := workflow.ScanDir(filepath.Join(repoDir, ".github", "workflows"))
	if err != nil {
		return Result{}, fmt.Errorf("failed to scan workflows: %w", err)
	}

	actions := collectRemoteActions(refs.UsesActions)
	sort.Slice(actions, func(i, j int) bool { return actions[i].Ref < actions[j].Ref })

	entries := make([]Entry, 0, len(actions))
	for _, a := range actions {
		sha, err := resolveSHA(a.Owner, a.Repo, a.RequestedRef, token)
		if err != nil {
			return Result{}, fmt.Errorf("resolve %s: %w", a.Ref, err)
		}

		relPath := filepath.ToSlash(filepath.Join(".act", "bundle", a.Owner, a.Repo+"@"+sha))
		destDir := filepath.Join(repoDir, filepath.FromSlash(relPath))
		if err := os.RemoveAll(destDir); err != nil {
			return Result{}, fmt.Errorf("clear existing bundle %s: %w", destDir, err)
		}
		if err := os.MkdirAll(destDir, 0o755); err != nil {
			return Result{}, fmt.Errorf("create bundle dir %s: %w", destDir, err)
		}

		if err := downloadAndExtract(a.Owner, a.Repo, sha, destDir, token); err != nil {
			return Result{}, fmt.Errorf("download %s: %w", a.Ref, err)
		}

		entries = append(entries, Entry{
			Ref:  a.Ref,
			SHA:  sha,
			Path: relPath,
		})
	}

	lockPath := filepath.Join(repoDir, ".act", "bundle.lock")
	if err := writeLockFile(lockPath, entries); err != nil {
		return Result{}, err
	}

	return Result{
		Entries:  entries,
		LockPath: lockPath,
	}, nil
}

type remoteAction struct {
	Ref          string
	Owner        string
	Repo         string
	RequestedRef string
}

func collectRemoteActions(uses []string) []remoteAction {
	seen := map[string]struct{}{}
	var result []remoteAction

	for _, use := range uses {
		a, ok := parseRemoteAction(use)
		if !ok {
			continue
		}
		if _, exists := seen[a.Ref]; exists {
			continue
		}
		seen[a.Ref] = struct{}{}
		result = append(result, a)
	}
	return result
}

func parseRemoteAction(use string) (remoteAction, bool) {
	if strings.HasPrefix(use, "./") || strings.HasPrefix(use, "docker://") {
		return remoteAction{}, false
	}
	left, ref, ok := strings.Cut(use, "@")
	if !ok || ref == "" {
		return remoteAction{}, false
	}
	parts := strings.Split(left, "/")
	if len(parts) < 2 {
		return remoteAction{}, false
	}
	owner, repo := parts[0], parts[1]
	return remoteAction{
		Ref:          owner + "/" + repo + "@" + ref,
		Owner:        owner,
		Repo:         repo,
		RequestedRef: ref,
	}, true
}

func resolveSHA(owner, repo, requestedRef, token string) (string, error) {
	if shaPattern.MatchString(requestedRef) {
		return requestedRef, nil
	}

	u := fmt.Sprintf("%s/repos/%s/%s/commits/%s", strings.TrimSuffix(githubAPIBaseURL, "/"), owner, repo, url.PathEscape(requestedRef))
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := defaultHTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("github API returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload struct {
		SHA string `json:"sha"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}
	if !shaPattern.MatchString(payload.SHA) {
		return "", fmt.Errorf("invalid SHA from github API for %s/%s@%s", owner, repo, requestedRef)
	}
	return payload.SHA, nil
}

func downloadAndExtract(owner, repo, sha, destDir, token string) error {
	u := fmt.Sprintf("%s/%s/%s/zip/%s", strings.TrimSuffix(codeloadBaseURL, "/"), owner, repo, sha)
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := defaultHTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("codeload returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	reader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return err
	}

	for _, f := range reader.File {
		parts := strings.SplitN(filepath.ToSlash(f.Name), "/", 2)
		if len(parts) < 2 {
			continue
		}
		rel := filepath.Clean(parts[1])
		if rel == "." || strings.HasPrefix(rel, "..") {
			continue
		}

		target := filepath.Join(destDir, filepath.FromSlash(rel))
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		in, err := f.Open()
		if err != nil {
			return err
		}
		mode := f.Mode().Perm()
		if mode == 0 {
			mode = 0o644
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
		if err != nil {
			in.Close()
			return err
		}
		if _, err := io.Copy(out, in); err != nil {
			out.Close()
			in.Close()
			return err
		}
		if err := out.Close(); err != nil {
			in.Close()
			return err
		}
		if err := in.Close(); err != nil {
			return err
		}
	}
	return nil
}

func writeLockFile(path string, entries []Entry) error {
	sort.Slice(entries, func(i, j int) bool { return entries[i].Ref < entries[j].Ref })

	payload := struct {
		Actions []Entry `yaml:"actions"`
	}{
		Actions: entries,
	}
	data, err := yaml.Marshal(payload)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
