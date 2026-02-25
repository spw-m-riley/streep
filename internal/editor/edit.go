package editor

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// Options configures file editing behaviour.
type Options struct {
	FilePath     string
	TemplatePath string
	Redact       bool
	In           io.Reader
	Out          io.Writer
	Err          io.Writer
	Editor       string // optional override (defaults to $EDITOR)
}

// Edit edits a dotenv-style file either via $EDITOR or interactive prompts.
func Edit(opts Options) error {
	if opts.FilePath == "" {
		return fmt.Errorf("file path is required")
	}
	if opts.TemplatePath == "" {
		return fmt.Errorf("template path is required")
	}

	if err := ensureFileFromTemplate(opts.FilePath, opts.TemplatePath); err != nil {
		return err
	}

	editor := opts.Editor
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if strings.TrimSpace(editor) != "" {
		return runEditor(editor, opts)
	}

	return promptEdit(opts)
}

func ensureFileFromTemplate(filePath, templatePath string) error {
	if _, err := os.Stat(templatePath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("template file %s not found (run 'streep new role')", templatePath)
		}
		return fmt.Errorf("failed to read template %s: %w", templatePath, err)
	}

	if _, err := os.Stat(filePath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check %s: %w", filePath, err)
	}

	data, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("failed to read template %s: %w", templatePath, err)
	}
	if err := os.WriteFile(filePath, data, 0o644); err != nil {
		return fmt.Errorf("failed to create %s: %w", filePath, err)
	}
	return nil
}

func runEditor(editorCmd string, opts Options) error {
	parts := strings.Fields(editorCmd)
	if len(parts) == 0 {
		return fmt.Errorf("invalid $EDITOR value")
	}

	args := append(parts[1:], opts.FilePath)
	cmd := exec.Command(parts[0], args...)
	cmd.Stdin = firstNonNilReader(opts.In, os.Stdin)
	cmd.Stdout = firstNonNilWriter(opts.Out, os.Stdout)
	cmd.Stderr = firstNonNilWriter(opts.Err, os.Stderr)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("editor command failed: %w", err)
	}
	return nil
}

func promptEdit(opts Options) error {
	keys, err := readDotenvKeys(opts.TemplatePath)
	if err != nil {
		return err
	}
	current, err := readDotenvValues(opts.FilePath)
	if err != nil {
		return err
	}

	in := firstNonNilReader(opts.In, os.Stdin)
	out := firstNonNilWriter(opts.Out, os.Stdout)
	scanner := bufio.NewScanner(in)

	updated := map[string]string{}
	for _, key := range keys {
		cur := strings.TrimSpace(current[key])
		display := cur
		if opts.Redact && display != "" {
			display = "******"
		}
		fmt.Fprintf(out, "%s [%s]: ", key, display)
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return err
			}
			return fmt.Errorf("input closed before all keys were entered")
		}
		next := strings.TrimSpace(scanner.Text())
		if next == "" {
			updated[key] = cur
		} else {
			updated[key] = next
		}
	}

	var b strings.Builder
	for _, key := range keys {
		fmt.Fprintf(&b, "%s=%s\n", key, updated[key])
	}
	if err := os.WriteFile(opts.FilePath, []byte(b.String()), 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", opts.FilePath, err)
	}
	return nil
}

func readDotenvKeys(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var keys []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, _, _ := strings.Cut(line, "=")
		key = strings.TrimPrefix(key, "export ")
		key = strings.TrimSpace(key)
		if key != "" {
			keys = append(keys, key)
		}
	}
	return keys, scanner.Err()
}

func readDotenvValues(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	result := map[string]string{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, _ := strings.Cut(line, "=")
		key = strings.TrimPrefix(key, "export ")
		key = strings.TrimSpace(key)
		if key != "" {
			result[key] = strings.Trim(strings.TrimSpace(value), `"'`)
		}
	}
	return result, scanner.Err()
}

func firstNonNilReader(a io.Reader, b io.Reader) io.Reader {
	if a != nil {
		return a
	}
	return b
}

func firstNonNilWriter(a io.Writer, b io.Writer) io.Writer {
	if a != nil {
		return a
	}
	return b
}
