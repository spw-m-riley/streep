package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"streep/internal/system"
	"streep/internal/workflow"
)

const doctorUsage = `Diagnose local readiness for running GitHub Actions with act.

Checks:
  - act installed and version available
  - docker installed and daemon reachable
  - .actrc present
  - required values in .secrets/.env/.vars/.input
  - generated .act/events/*.json payload files
  - .artifacts/ present when artifact actions are used

Usage:
  streep doctor [path] [--json]

If no path is given, the current directory is used.
`

func executeDoctor(args []string, stdout io.Writer, stderr io.Writer) error {
	_ = stderr
	args, jsonMode := splitJSONFlag(args)
	if jsonMode {
		var human bytes.Buffer
		err := executeDoctor(args, &human, stderr)
		if jsonErr := writeWrappedJSON(stdout, human.String(), err); jsonErr != nil {
			return jsonErr
		}
		return err
	}

	for _, arg := range args {
		if isHelp(arg) {
			_, err := io.WriteString(stdout, doctorUsage)
			return err
		}
	}

	dir := "."
	if len(args) == 1 {
		dir = args[0]
	} else if len(args) > 1 {
		return fmt.Errorf("streep doctor accepts at most one path argument")
	}

	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("failed to read target directory %q: %w", dir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("target path %q is not a directory", dir)
	}

	issues := 0

	// act presence/version
	if v, err := actVersion(); err != nil {
		fmt.Fprintf(stdout, "✗ act: %v\n", err)
		issues++
	} else {
		fmt.Fprintf(stdout, "✔ act: %s\n", v)
	}

	// docker presence/daemon
	if v, err := system.DockerStatus(); err != nil {
		fmt.Fprintf(stdout, "✗ docker: %v\n", err)
		issues++
	} else {
		fmt.Fprintf(stdout, "✔ docker: %s\n", v)
	}

	// .actrc present
	actrcPath := filepath.Join(dir, ".actrc")
	if _, err := os.Stat(actrcPath); err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintln(stdout, "✗ config: .actrc not found (run 'streep new role')")
			issues++
		} else {
			return fmt.Errorf("failed to stat .actrc: %w", err)
		}
	} else {
		fmt.Fprintln(stdout, "✔ config: .actrc present")
	}

	// Credential files populated
	credentialIssueCount, err := doctorCredentialChecks(dir, stdout)
	if err != nil {
		return err
	}
	issues += credentialIssueCount

	// Event files populated
	eventsIssueCount, err := doctorEventChecks(dir, stdout)
	if err != nil {
		return err
	}
	issues += eventsIssueCount

	// Artifact path validation (based on workflow usage)
	artifactIssueCount, err := doctorArtifactChecks(dir, stdout)
	if err != nil {
		return err
	}
	issues += artifactIssueCount

	if issues == 0 {
		fmt.Fprintln(stdout, "\nAll checks passed.")
	} else {
		fmt.Fprintf(stdout, "\n%d issue(s) found.\n", issues)
	}

	return nil
}

func actVersion() (string, error) {
	actPath, err := exec.LookPath("act")
	if err != nil {
		return "", fmt.Errorf("act not found in PATH")
	}
	out, err := exec.Command(actPath, "--version").Output()
	if err != nil {
		return "", fmt.Errorf("failed to read act version: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func doctorCredentialChecks(dir string, out io.Writer) (int, error) {
	issues := 0
	checked := 0
	for _, cf := range checkFiles {
		examplePath := filepath.Join(dir, cf.example)
		realPath := filepath.Join(dir, cf.real)

		if _, err := os.Stat(examplePath); os.IsNotExist(err) {
			continue
		}

		required, err := readDotenvKeys(examplePath)
		if err != nil {
			return 0, fmt.Errorf("failed to read %s: %w", cf.example, err)
		}
		if len(required) == 0 {
			continue
		}
		checked++

		if _, err := os.Stat(realPath); os.IsNotExist(err) {
			fmt.Fprintf(out, "✗ %s: %s not found (copy from %s)\n", cf.label, cf.real, cf.example)
			issues++
			continue
		}

		actual, err := readDotenvValues(realPath)
		if err != nil {
			return 0, fmt.Errorf("failed to read %s: %w", cf.real, err)
		}

		var missing []string
		for _, key := range required {
			v, ok := actual[key]
			if !ok || strings.TrimSpace(v) == "" {
				missing = append(missing, key)
			}
		}

		if len(missing) > 0 {
			fmt.Fprintf(out, "✗ %s: missing or empty values in %s: %s\n", cf.label, cf.real, strings.Join(missing, ", "))
			issues++
		} else {
			fmt.Fprintf(out, "✔ %s: all %d key(s) present in %s\n", cf.label, len(required), cf.real)
		}
	}

	if checked == 0 {
		fmt.Fprintln(out, "⚠ credentials: no .*.example files found to validate")
		issues++
	}
	return issues, nil
}

func doctorEventChecks(dir string, out io.Writer) (int, error) {
	eventsDir := filepath.Join(dir, ".act", "events")
	entries, err := os.ReadDir(eventsDir)
	if os.IsNotExist(err) {
		fmt.Fprintln(out, "✗ events: .act/events not found (run 'streep new role')")
		return 1, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to read .act/events: %w", err)
	}

	count := 0
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			count++
		}
	}

	if count == 0 {
		fmt.Fprintln(out, "✗ events: no .json payload files found in .act/events")
		return 1, nil
	}

	fmt.Fprintf(out, "✔ events: found %d payload file(s) in .act/events\n", count)
	return 0, nil
}

func doctorArtifactChecks(dir string, out io.Writer) (int, error) {
	refs, err := workflow.ScanDir(filepath.Join(dir, ".github", "workflows"))
	if err != nil {
		return 0, fmt.Errorf("failed to scan workflows for artifact usage: %w", err)
	}

	if !workflow.DetectsArtifactActions(refs.UsesActions) {
		fmt.Fprintln(out, "✔ artifacts: not required by workflows")
		return 0, nil
	}

	artifactsDir := filepath.Join(dir, ".artifacts")
	if _, err := os.Stat(artifactsDir); os.IsNotExist(err) {
		fmt.Fprintln(out, "✗ artifacts: upload/download-artifact is used but .artifacts/ is missing")
		return 1, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to stat .artifacts: %w", err)
	}

	fmt.Fprintln(out, "✔ artifacts: .artifacts/ present")
	return 0, nil
}
