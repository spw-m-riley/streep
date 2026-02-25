package scaffold

import (
	"fmt"
	"os"
	"strings"
)

const gitignoreBlockHeader = "\n# streep — act local run files (do not commit real values)\n"

var defaultGitignoreEntries = []string{".secrets", ".env", ".vars"}

// EnsureGitignoreEntries appends a guarded block to the .gitignore at path
// (creating it if absent) ensuring .secrets, .env, .vars and any extra entries are listed.
// It is a no-op for entries that already exist in the file.
func EnsureGitignoreEntries(path string, extra ...string) error {
	var existing string
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read %s: %w", path, err)
	}
	existing = string(data)

	allEntries := append(defaultGitignoreEntries, extra...) //nolint:gocritic
	var missing []string
	for _, entry := range allEntries {
		if !containsLine(existing, entry) {
			missing = append(missing, entry)
		}
	}
	if len(missing) == 0 {
		return nil
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", path, err)
	}
	defer f.Close()

	var block strings.Builder
	block.WriteString(gitignoreBlockHeader)
	for _, entry := range missing {
		block.WriteString(entry)
		block.WriteString("\n")
	}

	_, err = fmt.Fprint(f, block.String())
	return err
}

// containsLine reports whether text contains a line equal to target (trimmed).
func containsLine(text, target string) bool {
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) == target {
			return true
		}
	}
	return false
}
