package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const cleanUsage = `Clean local act runtime files created by streep.

By default this is a dry-run and prints what would be removed.
Use --force to actually remove files.

Usage:
  streep clean [path] [--force]

Examples:
  streep clean
  streep clean --force
  streep clean /path/to/repo --force
`

func executeClean(args []string, stdout io.Writer, stderr io.Writer) error {
	_ = stderr

	force := false
	positional := make([]string, 0, 1)
	for _, arg := range args {
		switch arg {
		case "-h", "--help", "help":
			_, err := io.WriteString(stdout, cleanUsage)
			return err
		case "--force":
			force = true
		default:
			if strings.HasPrefix(arg, "-") {
				return fmt.Errorf("unknown flag %q", arg)
			}
			positional = append(positional, arg)
		}
	}

	if len(positional) > 1 {
		return fmt.Errorf("streep clean accepts at most one path argument")
	}

	dir := "."
	if len(positional) == 1 {
		dir = positional[0]
	}

	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("failed to read target directory %q: %w", dir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("target path %q is not a directory", dir)
	}

	fileTargets := []string{".secrets", ".env", ".vars", ".input", ".actrc", filepath.Join(".act", "run-fingerprint")}
	dirTargets := []string{".artifacts", filepath.Join(".act", "cache"), filepath.Join(".act", "events")}
	var existing []string

	for _, rel := range fileTargets {
		if _, err := os.Stat(filepath.Join(dir, rel)); err == nil {
			existing = append(existing, rel)
		}
	}
	for _, rel := range dirTargets {
		if _, err := os.Stat(filepath.Join(dir, rel)); err == nil {
			existing = append(existing, rel+"/")
		}
	}

	if len(existing) == 0 {
		fmt.Fprintln(stdout, "Nothing to clean.")
		return nil
	}

	if !force {
		fmt.Fprintln(stdout, "Dry-run: would remove")
		for _, rel := range existing {
			fmt.Fprintf(stdout, "  - %s\n", rel)
		}
		fmt.Fprintln(stdout, "\nRe-run with --force to remove these files.")
		return nil
	}

	var removed []string
	for _, rel := range fileTargets {
		p := filepath.Join(dir, rel)
		if _, err := os.Stat(p); err == nil {
			if err := os.Remove(p); err != nil {
				return fmt.Errorf("failed to remove %s: %w", rel, err)
			}
			removed = append(removed, rel)
		}
	}
	for _, rel := range dirTargets {
		p := filepath.Join(dir, rel)
		if _, err := os.Stat(p); err == nil {
			if err := removeDirContents(p); err != nil {
				return fmt.Errorf("failed to clean %s: %w", rel, err)
			}
			removed = append(removed, rel+"/")
		}
	}

	fmt.Fprintf(stdout, "Removed: %s\n", strings.Join(removed, " "))
	return nil
}

func removeDirContents(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := os.RemoveAll(filepath.Join(dir, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}
