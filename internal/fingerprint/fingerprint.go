package fingerprint

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// FileDigest is the digest of a single file participating in a fingerprint.
type FileDigest struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

// Data represents a deterministic project fingerprint snapshot.
type Data struct {
	Version  int          `json:"version"`
	Platform string       `json:"platform"`
	Digest   string       `json:"digest"`
	Files    []FileDigest `json:"files"`
}

// Build computes a deterministic fingerprint for repoDir.
func Build(repoDir string) (Data, error) {
	if repoDir == "" {
		repoDir = "."
	}

	paths, err := candidateFiles(repoDir)
	if err != nil {
		return Data{}, err
	}

	files := make([]FileDigest, 0, len(paths))
	for _, rel := range paths {
		abs := filepath.Join(repoDir, filepath.FromSlash(rel))
		content, err := os.ReadFile(abs)
		if err != nil {
			return Data{}, fmt.Errorf("read %s: %w", rel, err)
		}
		sum := sha256.Sum256(content)
		files = append(files, FileDigest{
			Path:   rel,
			SHA256: hex.EncodeToString(sum[:]),
		})
	}

	platform := runtime.GOOS + "/" + runtime.GOARCH
	digest := overallDigest(platform, files)
	return Data{
		Version:  1,
		Platform: platform,
		Digest:   digest,
		Files:    files,
	}, nil
}

// WriteCurrent computes and writes the fingerprint to .act/run-fingerprint.
func WriteCurrent(repoDir string) (Data, string, error) {
	data, err := Build(repoDir)
	if err != nil {
		return Data{}, "", err
	}

	path := filepath.Join(repoDir, ".act", "run-fingerprint")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return Data{}, "", fmt.Errorf("create .act directory: %w", err)
	}

	encoded, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return Data{}, "", err
	}
	encoded = append(encoded, '\n')
	if err := os.WriteFile(path, encoded, 0o644); err != nil {
		return Data{}, "", fmt.Errorf("write fingerprint: %w", err)
	}
	return data, path, nil
}

// Load reads a fingerprint from disk.
func Load(path string) (Data, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Data{}, err
	}
	var data Data
	if err := json.Unmarshal(content, &data); err != nil {
		return Data{}, err
	}
	return data, nil
}

func candidateFiles(repoDir string) ([]string, error) {
	workflowPatterns := []string{
		filepath.Join(repoDir, ".github", "workflows", "*.yml"),
		filepath.Join(repoDir, ".github", "workflows", "*.yaml"),
	}

	set := map[string]struct{}{}
	for _, p := range workflowPatterns {
		matches, err := filepath.Glob(p)
		if err != nil {
			return nil, err
		}
		for _, m := range matches {
			rel, err := filepath.Rel(repoDir, m)
			if err != nil {
				return nil, err
			}
			set[filepath.ToSlash(rel)] = struct{}{}
		}
	}

	for _, rel := range []string{
		".actrc",
		".secrets",
		".env",
		".vars",
		".input",
		".act/bundle.lock",
	} {
		abs := filepath.Join(repoDir, filepath.FromSlash(rel))
		if _, err := os.Stat(abs); err == nil {
			set[rel] = struct{}{}
		}
	}

	paths := make([]string, 0, len(set))
	for p := range set {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	return paths, nil
}

func overallDigest(platform string, files []FileDigest) string {
	var b strings.Builder
	b.WriteString("version:1\n")
	b.WriteString("platform:")
	b.WriteString(platform)
	b.WriteByte('\n')
	for _, f := range files {
		b.WriteString(f.Path)
		b.WriteByte(':')
		b.WriteString(f.SHA256)
		b.WriteByte('\n')
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}
