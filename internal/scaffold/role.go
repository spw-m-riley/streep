package scaffold

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"streep/internal/workflow"
)

// RoleOptions configures the behaviour of NewRole.
type RoleOptions struct {
	Dir   string
	Force bool
	Out   io.Writer
	// Arch overrides runtime.GOARCH for testing.
	Arch string
}

// NewRole scaffolds act-oriented files in opts.Dir:
//
//   - .secrets.example  (discovered secrets + always GITHUB_TOKEN)
//   - .env.example      (discovered env vars)
//   - .vars.example     (discovered repository vars)
//   - .input.example    (workflow_dispatch inputs, if any)
//   - .actrc            (wires files into act flags + -P runner mappings)
//   - .artifacts/       (created when artifact actions detected)
//   - .gitignore        (guarded block protecting real credential files)
func NewRole(opts RoleOptions) error {
	targetDir := opts.Dir
	if targetDir == "" {
		targetDir = "."
	}

	info, err := os.Stat(targetDir)
	if err != nil {
		return fmt.Errorf("failed to read target directory %q: %w", targetDir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("target path %q is not a directory", targetDir)
	}

	refs, err := workflow.ScanDir(filepath.Join(targetDir, ".github", "workflows"))
	if err != nil {
		return fmt.Errorf("failed to scan workflow files: %w", err)
	}

	arch := opts.Arch
	if arch == "" {
		arch = runtime.GOARCH
	}

	out := opts.Out
	if out == nil {
		out = io.Discard
	}

	usesArtifacts := workflow.DetectsArtifactActions(refs.UsesActions)
	allInputs := mergeWorkflowInputs(refs.WorkflowInputs)

	files := buildScaffoldFiles(refs, arch, usesArtifacts, allInputs)
	for _, f := range files {
		path := filepath.Join(targetDir, filepath.FromSlash(f.path))
		if err := writeScaffoldFile(path, f.content, opts.Force); err != nil {
			return err
		}
		fmt.Fprintf(out, "  wrote %s\n", f.path)
	}

	if usesArtifacts {
		artifactsDir := filepath.Join(targetDir, ".artifacts")
		if err := os.MkdirAll(artifactsDir, 0o755); err != nil {
			return fmt.Errorf("failed to create .artifacts directory: %w", err)
		}
		gitkeep := filepath.Join(artifactsDir, ".gitkeep")
		if err := writeScaffoldFile(gitkeep, "", opts.Force); err != nil && !os.IsExist(err) {
			return err
		}
		fmt.Fprintf(out, "  wrote .artifacts/\n")
	}

	gitignorePath := filepath.Join(targetDir, ".gitignore")
	extraEntries := []string{}
	if usesArtifacts {
		extraEntries = append(extraEntries, ".artifacts/")
	}
	if err := EnsureGitignoreEntries(gitignorePath, extraEntries...); err != nil {
		return err
	}

	// Generate .act/events/<event>.json for each discovered trigger event.
	if len(refs.Events) > 0 {
		if err := writeEventFiles(targetDir, refs.Events, opts.Force); err != nil {
			return err
		}
		for _, ev := range refs.Events {
			fmt.Fprintf(out, "  wrote .act/events/%s.json\n", ev)
		}
	}

	printNextSteps(out, targetDir, allInputs, refs)
	return nil
}

type scaffoldFile struct {
	path    string
	content string
}

func buildScaffoldFiles(refs workflow.References, arch string, usesArtifacts bool, allInputs []string) []scaffoldFile {
	files := []scaffoldFile{
		{path: ".secrets.example", content: buildDotenvFile("Copy to .secrets and provide real values", withGitHubToken(refs.Secrets))},
		{path: ".env.example", content: buildDotenvFile("Copy to .env and provide real values", refs.Env)},
		{path: ".vars.example", content: buildDotenvFile("Copy to .vars and provide real values", refs.Vars)},
	}

	if len(allInputs) > 0 {
		files = append(files, scaffoldFile{
			path:    ".input.example",
			content: buildDotenvFile("Copy to .input and provide workflow_dispatch input values", allInputs),
		})
	}

	files = append(files, scaffoldFile{
		path:    ".actrc",
		content: buildActrc(refs.Runners, refs.SelfHosted, arch, usesArtifacts, len(allInputs) > 0),
	})

	return files
}

// runnerImageMap maps known GitHub-hosted runner labels to catthehacker act images.
var runnerImageMap = map[string]string{
	"ubuntu-latest": "catthehacker/ubuntu:act-latest",
	"ubuntu-24.04":  "catthehacker/ubuntu:act-24.04",
	"ubuntu-22.04":  "catthehacker/ubuntu:act-22.04",
	"ubuntu-20.04":  "catthehacker/ubuntu:act-20.04",
}

func buildActrc(runners []string, selfHosted [][]string, arch string, usesArtifacts bool, hasInputs bool) string {
	var b strings.Builder
	b.WriteString("--secret-file .secrets\n")
	b.WriteString("--env-file .env\n")
	b.WriteString("--var-file .vars\n")

	if hasInputs {
		b.WriteString("--input-file .input\n")
	}

	for _, r := range runners {
		if img, ok := runnerImageMap[r]; ok {
			fmt.Fprintf(&b, "-P %s=%s\n", r, img)
		}
	}

	// Self-hosted runners all map to a single image (act limitation).
	if len(selfHosted) > 0 {
		b.WriteString("-P self-hosted=catthehacker/ubuntu:act-latest\n")
	}

	if arch == "arm64" {
		b.WriteString("--container-architecture linux/amd64\n")
	}

	if usesArtifacts {
		b.WriteString("--artifact-server-path .artifacts\n")
	}

	return b.String()
}

// mergeWorkflowInputs flattens all workflow_dispatch input names into a sorted deduped list.
func mergeWorkflowInputs(m map[string][]string) []string {
	seen := map[string]struct{}{}
	for _, inputs := range m {
		for _, k := range inputs {
			seen[k] = struct{}{}
		}
	}
	result := make([]string, 0, len(seen))
	for k := range seen {
		result = append(result, k)
	}
	sort.Strings(result)
	return result
}

// withGitHubToken ensures GITHUB_TOKEN is always the first entry.
func withGitHubToken(secrets []string) []string {
	seen := map[string]bool{"GITHUB_TOKEN": true}
	result := []string{"GITHUB_TOKEN"}
	for _, s := range secrets {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

func buildDotenvFile(comment string, keys []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n", comment)
	for _, k := range keys {
		if k == "GITHUB_TOKEN" {
			b.WriteString("# Required by act to clone remote actions. Must be a classic PAT with 'repo' scope.\n")
			b.WriteString("# Create one at: https://github.com/settings/tokens (Tokens (classic))\n")
		}
		fmt.Fprintf(&b, "%s=\n", k)
	}
	return b.String()
}

func printNextSteps(out io.Writer, dir string, inputs []string, refs workflow.References) {
	fmt.Fprintf(out, "✔ Initialized role scaffold in %s\n", dir)

	if refs.MatrixCount > 0 {
		fmt.Fprintf(out, "  Matrix: ~%d combination(s) detected across workflows\n", refs.MatrixCount)
	}

	if warning := selfHostedWarning(refs.SelfHosted); warning != "" {
		fmt.Fprint(out, warning)
	}

	fmt.Fprintf(out, "\nNext steps:\n")
	fmt.Fprintf(out, "  1. cp .secrets.example .secrets  →  fill in GITHUB_TOKEN and other secrets\n")
	fmt.Fprintf(out, "  2. cp .env.example .env          →  fill in environment values\n")
	fmt.Fprintf(out, "  3. cp .vars.example .vars        →  fill in repository variable values\n")
	step := 4
	if len(inputs) > 0 {
		fmt.Fprintf(out, "  %d. cp .input.example .input      →  fill in workflow_dispatch input values\n", step)
		step++
	}
	if len(refs.Events) > 0 {
		fmt.Fprintf(out, "  %d. act                           →  run your workflows locally\n", step)
		step++
		fmt.Fprintf(out, "\nEvent payload files generated in .act/events/:\n")
		for _, ev := range refs.Events {
			fmt.Fprintf(out, "  act %s -e .act/events/%s.json\n", ev, ev)
		}
	} else {
		fmt.Fprintf(out, "  %d. act                           →  run your workflows locally\n", step)
	}
}

func writeScaffoldFile(path string, content string, force bool) error {
	if !force {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("refusing to overwrite existing file %q (use --force)", path)
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("failed to check file %q: %w", path, err)
		}
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("failed to create directory for %q: %w", path, err)
	}

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("failed to write file %q: %w", path, err)
	}

	return nil
}
