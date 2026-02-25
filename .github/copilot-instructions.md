# Copilot Instructions

## Build & Test

```bash
go build ./...
go test ./...

# Run a single test
go test ./cmd -run TestCheckPassesWhenAllValuesPresent
go test ./internal/workflow -run TestScanDir
```

No linter is configured. No Makefile.

## Architecture

`streep` is a CLI tool that wraps [`act`](https://github.com/nektos/act) to make local GitHub Actions execution easier. It scans `.github/workflows/`, generates configuration files act needs (`.secrets`, `.env`, `.vars`, `.actrc`), and provides commands for checking, running, linting, and diagnosing local workflow runs.

**Layers:**

- `main.go` — entry point; calls `cmd.Execute(os.Args[1:], os.Stdout, os.Stderr)` and exits on error
- `cmd/` — command dispatch and CLI logic; one file per command (e.g. `cmd/check.go`)
- `internal/` — business logic packages imported by `cmd/`:
  - `workflow` — parses and scans workflow YAML files
  - `scaffold` — generates `.secrets.example`, `.env.example`, `.vars.example`, `.actrc`, event JSON files
  - `bundle`, `diagnose`, `editor`, `fingerprint`, `hook`, `policy`, `system` — one concern each

The only external dependency is `gopkg.in/yaml.v3`.

## Key Conventions

**No CLI framework.** Routing is a hand-rolled `switch args[0]` in `cmd/root.go`. Each command is a function `execute<Name>(args []string, stdout io.Writer, stderr io.Writer) error`.

**I/O is always injected.** Commands never write to `os.Stdout`/`os.Stderr` directly — they use the passed `io.Writer` parameters. Tests pass `bytes.Buffer` instances.

**Help text is a package-level `const`.** Each `cmd/*.go` file defines a `const <name>Usage` string; commands check `isHelp(args[0])` and write it to `stdout`.

**Tests use `t.TempDir()`.** All file-based tests create a temp directory, write fixture files into it, and pass the path to the function under test. No global state.

**`internal/workflow` operates on `yaml.Node` trees**, not unmarshaled structs. This allows structured navigation of YAML while excluding comments from expression scanning. The helper `visitMappingValue(node, key, fn)` is the standard way to traverse mapping nodes.

**Output uses `✔`/`✗` prefix** for pass/fail status lines (e.g. in `check`, `doctor`).

**`--force` guards all file writes.** `writeScaffoldFile` refuses to overwrite existing files unless `force` is true, returning a descriptive error.
