package hook

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const managedMarker = "# streep-managed-hook"

const preCommitScript = `#!/bin/sh
# streep-managed-hook
set -e
staged="$(git diff --cached --name-only)"
echo "$staged" | grep -E '^\.github/workflows/.*\.ya?ml$' >/dev/null 2>&1 || exit 0
streep lint
`

const prePushScript = `#!/bin/sh
# streep-managed-hook
set -e
streep check
`

// Install writes streep-managed pre-commit and pre-push hooks.
func Install(repoDir string) (int, error) {
	if repoDir == "" {
		repoDir = "."
	}

	hooksDir := filepath.Join(repoDir, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		return 0, fmt.Errorf("failed to create hooks directory: %w", err)
	}

	written := 0
	if err := writeHook(filepath.Join(hooksDir, "pre-commit"), preCommitScript); err != nil {
		return written, err
	}
	written++
	if err := writeHook(filepath.Join(hooksDir, "pre-push"), prePushScript); err != nil {
		return written, err
	}
	written++

	return written, nil
}

// Uninstall removes streep-managed hooks and leaves unmanaged hooks untouched.
func Uninstall(repoDir string) (int, error) {
	if repoDir == "" {
		repoDir = "."
	}

	hooksDir := filepath.Join(repoDir, ".git", "hooks")
	removed := 0
	for _, name := range []string{"pre-commit", "pre-push"} {
		ok, err := removeManagedHook(filepath.Join(hooksDir, name))
		if err != nil {
			return removed, err
		}
		if ok {
			removed++
		}
	}
	return removed, nil
}

func writeHook(path, content string) error {
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		return fmt.Errorf("failed to write hook %s: %w", filepath.Base(path), err)
	}
	return nil
}

func removeManagedHook(path string) (bool, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to read hook %s: %w", filepath.Base(path), err)
	}
	if !strings.Contains(string(data), managedMarker) {
		return false, nil
	}
	if err := os.Remove(path); err != nil {
		return false, fmt.Errorf("failed to remove hook %s: %w", filepath.Base(path), err)
	}
	return true, nil
}
