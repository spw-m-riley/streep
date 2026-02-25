package cmd

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"streep/internal/fingerprint"
)

const fingerprintUsage = `Create or compare deterministic workflow run fingerprints.

Usage:
  streep fingerprint [path]
  streep fingerprint compare <run1> <run2>

Examples:
  streep fingerprint
  streep fingerprint /path/to/repo
  streep fingerprint compare .act/run-fingerprint other-fingerprint.json
`

func executeFingerprint(args []string, stdout io.Writer, stderr io.Writer) error {
	_ = stderr

	if len(args) > 0 && isHelp(args[0]) {
		_, err := io.WriteString(stdout, fingerprintUsage)
		return err
	}

	if len(args) > 0 && args[0] == "compare" {
		if len(args) != 3 {
			return fmt.Errorf("usage: streep fingerprint compare <run1> <run2>")
		}
		left, err := fingerprint.Load(args[1])
		if err != nil {
			return fmt.Errorf("read %s: %w", args[1], err)
		}
		right, err := fingerprint.Load(args[2])
		if err != nil {
			return fmt.Errorf("read %s: %w", args[2], err)
		}

		if left.Digest == right.Digest {
			fmt.Fprintln(stdout, "✔ Fingerprints match.")
		} else {
			fmt.Fprintln(stdout, "✗ Fingerprints differ.")
			fmt.Fprintf(stdout, "  left:  %s\n", left.Digest)
			fmt.Fprintf(stdout, "  right: %s\n", right.Digest)
		}
		return nil
	}

	dir := "."
	if len(args) == 1 {
		dir = args[0]
	} else if len(args) > 1 {
		return fmt.Errorf("streep fingerprint accepts at most one path argument")
	}

	data, path, err := fingerprint.WriteCurrent(dir)
	if err != nil {
		return err
	}
	relPath := path
	if rel, err := filepath.Rel(dir, path); err == nil {
		relPath = rel
	}
	fmt.Fprintf(stdout, "Fingerprint: %s\n", data.Digest)
	fmt.Fprintf(stdout, "Wrote %s\n", strings.TrimSpace(relPath))
	return nil
}
