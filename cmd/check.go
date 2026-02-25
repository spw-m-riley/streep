package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

const checkUsage = `Validate that act credential files are ready to use.

Reads the .example files as the required-key manifest and checks the real
files (.secrets, .env, .vars, .input) for missing files or empty values.

Usage:
  streep check [path]

If no path is given, the current directory is used.
`

// checkFile pairs an example template with the real file it should be copied to.
type checkFile struct {
	example string
	real    string
	label   string
}

var checkFiles = []checkFile{
	{example: ".secrets.example", real: ".secrets", label: "secrets"},
	{example: ".env.example", real: ".env", label: "env"},
	{example: ".vars.example", real: ".vars", label: "vars"},
	{example: ".input.example", real: ".input", label: "inputs"},
}

func executeCheck(args []string, stdout io.Writer, stderr io.Writer) error {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			_, err := io.WriteString(stdout, checkUsage)
			return err
		}
	}

	dir := "."
	if len(args) == 1 {
		dir = args[0]
	} else if len(args) > 1 {
		return fmt.Errorf("streep check accepts at most one path argument")
	}

	allPassed := true
	for _, cf := range checkFiles {
		examplePath := dir + "/" + cf.example
		realPath := dir + "/" + cf.real

		// If no example file, skip this pair entirely
		if _, err := os.Stat(examplePath); os.IsNotExist(err) {
			continue
		}

		required, err := readDotenvKeys(examplePath)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", cf.example, err)
		}
		if len(required) == 0 {
			continue
		}

		// Check real file exists
		if _, err := os.Stat(realPath); os.IsNotExist(err) {
			fmt.Fprintf(stdout, "✗ %s: %s not found (copy from %s)\n", cf.label, cf.real, cf.example)
			allPassed = false
			continue
		}

		actual, err := readDotenvValues(realPath)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", cf.real, err)
		}

		var missing []string
		for _, key := range required {
			v, ok := actual[key]
			if !ok || strings.TrimSpace(v) == "" {
				missing = append(missing, key)
			}
		}

		if len(missing) > 0 {
			fmt.Fprintf(stdout, "✗ %s: missing or empty values in %s: %s\n", cf.label, cf.real, strings.Join(missing, ", "))
			allPassed = false
		} else {
			fmt.Fprintf(stdout, "✔ %s: all %d key(s) present in %s\n", cf.label, len(required), cf.real)
		}
	}

	if allPassed {
		fmt.Fprintln(stdout, "\nAll checks passed — you're ready to run act.")
	} else {
		fmt.Fprintln(stdout, "\nSome checks failed — fill in the missing values before running act.")
	}

	return nil
}

// readDotenvKeys returns all key names from a dotenv file (ignoring comments and blank lines).
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

// readDotenvValues returns a map of key→value from a dotenv file.
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
