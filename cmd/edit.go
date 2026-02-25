package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"streep/internal/editor"
)

const editUsage = `Edit populated act credential files.

Uses $EDITOR when available; otherwise falls back to an interactive prompt.
Templates from .*.example files are used as the key manifest.

Usage:
  streep edit <secrets|env|vars|input> [path]

Examples:
  streep edit secrets
  streep edit env
  streep edit vars /path/to/repo
`

type editTarget struct {
	example string
	real    string
	redact  bool
}

var editTargets = map[string]editTarget{
	"secrets": {example: ".secrets.example", real: ".secrets", redact: true},
	"env":     {example: ".env.example", real: ".env", redact: false},
	"vars":    {example: ".vars.example", real: ".vars", redact: false},
	"input":   {example: ".input.example", real: ".input", redact: false},
}

func executeEdit(args []string, stdout io.Writer, stderr io.Writer) error {
	if len(args) == 0 || isHelp(args[0]) {
		_, err := io.WriteString(stdout, editUsage)
		return err
	}
	if len(args) > 2 {
		return fmt.Errorf("streep edit accepts a target and optional path")
	}

	targetName := args[0]
	target, ok := editTargets[targetName]
	if !ok {
		return fmt.Errorf("unknown edit target %q (expected one of: secrets, env, vars, input)", targetName)
	}

	dir := "."
	if len(args) == 2 {
		dir = args[1]
	}

	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("failed to read target directory %q: %w", dir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("target path %q is not a directory", dir)
	}

	templatePath := filepath.Join(dir, target.example)
	realPath := filepath.Join(dir, target.real)
	if err := editor.Edit(editor.Options{
		FilePath:     realPath,
		TemplatePath: templatePath,
		Redact:       target.redact,
		In:           os.Stdin,
		Out:          stdout,
		Err:          stderr,
	}); err != nil {
		return err
	}

	required, err := readDotenvKeys(templatePath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", target.example, err)
	}
	values, err := readDotenvValues(realPath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", target.real, err)
	}

	var missing []string
	for _, key := range required {
		v, ok := values[key]
		if !ok || strings.TrimSpace(v) == "" {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("%s has missing or empty values: %s", target.real, strings.Join(missing, ", "))
	}

	fmt.Fprintf(stdout, "✔ Updated %s\n", target.real)
	return nil
}
