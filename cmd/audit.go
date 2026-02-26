package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const auditUsage = `Audit local safety settings for streep-managed sensitive files.

Checks:
  - restrictive permissions on .secrets/.env/.vars/.input when present
  - .gitignore entries for .secrets/.env/.vars/.input

Usage:
  streep audit [path] [--json]
`

func executeAudit(args []string, stdout io.Writer, stderr io.Writer) error {
	_ = stderr
	args, jsonMode := splitJSONFlag(args)
	if jsonMode {
		var human bytes.Buffer
		err := executeAudit(args, &human, stderr)
		if jsonErr := writeWrappedJSON(stdout, human.String(), err); jsonErr != nil {
			return jsonErr
		}
		return err
	}

	for _, arg := range args {
		if isHelp(arg) {
			_, err := io.WriteString(stdout, auditUsage)
			return err
		}
	}

	dir := "."
	if len(args) == 1 {
		dir = args[0]
	} else if len(args) > 1 {
		return fmt.Errorf("streep audit accepts at most one path argument")
	}

	issues := 0
	sensitiveFiles := []string{".secrets", ".env", ".vars", ".input"}
	for _, rel := range sensitiveFiles {
		path := filepath.Join(dir, rel)
		info, err := os.Stat(path)
		if os.IsNotExist(err) {
			fmt.Fprintf(stdout, "✔ permissions: %s not present (skipped)\n", rel)
			continue
		}
		if err != nil {
			return fmt.Errorf("failed to stat %s: %w", rel, err)
		}
		if info.Mode().Perm()&0o077 != 0 {
			fmt.Fprintf(stdout, "✗ permissions: %s is too permissive (mode %04o)\n", rel, info.Mode().Perm())
			issues++
		} else {
			fmt.Fprintf(stdout, "✔ permissions: %s mode is restrictive (%04o)\n", rel, info.Mode().Perm())
		}
	}

	gitignorePath := filepath.Join(dir, ".gitignore")
	giData, err := os.ReadFile(gitignorePath)
	if os.IsNotExist(err) {
		fmt.Fprintln(stdout, "✗ .gitignore not found")
		issues++
	} else if err != nil {
		return fmt.Errorf("failed to read .gitignore: %w", err)
	} else {
		lines := make(map[string]bool)
		for _, line := range strings.Split(string(giData), "\n") {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" || strings.HasPrefix(trimmed, "#") {
				continue
			}
			lines[trimmed] = true
		}
		for _, rel := range sensitiveFiles {
			if lines[rel] {
				fmt.Fprintf(stdout, "✔ gitignore: %s is ignored\n", rel)
			} else {
				fmt.Fprintf(stdout, "✗ gitignore: missing entry for %s\n", rel)
				issues++
			}
		}
	}

	if issues == 0 {
		fmt.Fprintln(stdout, "\nAudit passed.")
		return nil
	}
	fmt.Fprintf(stdout, "\nAudit found %d issue(s).\n", issues)
	return fmt.Errorf("audit failed")
}
