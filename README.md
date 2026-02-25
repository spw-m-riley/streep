# Streep

<p align="center">
  <img src="streep.png" width="800" alt="Streep — pixel art of a developer at a command centre" />
</p>

> *"I think the most liberating thing I did early on was to free myself from any concern with my own face."*  
> — Meryl Streep, on the art of inhabiting a role completely.

[`act`](https://github.com/nektos/act) lets you run GitHub Actions locally — but getting it to actually work requires a frustrating amount of setup: secrets, environment variables, runner images, event payloads, architecture flags… Before you can say "action," you've lost an afternoon.

**Streep** is named after the greatest actor of all time because it does for your CI what Meryl does for every role she takes: it steps in, reads the material, understands exactly what's needed, and delivers a flawless local performance. One command — `streep new role` — and you're ready.

---

## Philosophy

Just as Meryl Streep inhabits every character with precision and no wasted effort, **streep** takes on the configuration work so you don't have to. It scans your workflows, understands their requirements, and builds everything act needs to run. You focus on your code. Streep handles the preparation.

> *Three Academy Award nominations before winning? streep doesn't make you wait that long.*

---

## Quick Start

```bash
streep new role          # prepare the stage
streep check             # make sure your lines are memorised
streep rehearse          # run through it once without the cameras rolling
streep perform           # lights up — action!
```

---

## Commands

### `streep new role` — *Accept the Part*

> Meryl Streep doesn't just show up on set. She researches the accent, the history, the posture, the silence between lines. `streep new role` does the same for your workflows.

Scans `.github/workflows/`, reads every reference your workflows make — secrets, environment variables, repository vars, runner images, artifact actions, workflow dispatch inputs — and generates a complete local configuration for act.

**What it generates:**

| File | Purpose |
|------|---------|
| `.secrets.example` | All discovered secrets (always includes `GITHUB_TOKEN`) |
| `.env.example` | All discovered environment variable references |
| `.vars.example` | All discovered repository var references |
| `.input.example` | `workflow_dispatch` input keys *(only when inputs are declared)* |
| `.actrc` | Pre-wired act configuration |
| `.act/events/<event>.json` | Minimal valid event payloads for each trigger |

**What `.actrc` includes:**

- `--secret-file`, `--env-file`, `--var-file`, `--input-file` for every generated file
- `-P ubuntu-latest=catthehacker/ubuntu:act-latest` (and other Ubuntu variants) for better runner compatibility
- `--container-architecture linux/amd64` automatically when running on Apple Silicon
- `--artifact-server-path .artifacts` when `upload-artifact` / `download-artifact` are detected

**Also:**

- Creates `.artifacts/` when artifact actions are detected
- Appends a guarded block to `.gitignore` covering `.secrets`, `.env`, `.vars`, `.artifacts/`
- Prints exact copy-and-fill instructions as next steps

```bash
streep new role                   # scaffold the current directory
streep new role /path/to/repo     # scaffold a specific path
streep new role --force           # overwrite existing scaffold files
```

---

### `streep check` — *Lines Memorised?*

> Before the cameras roll, every actor runs their lines. `streep check` makes sure you haven't forgotten any.

Reads your `.secrets.example`, `.env.example`, `.vars.example`, and `.input.example` files as the required-key manifest, then checks that the real files exist and have every key filled in.

```
✔ secrets: all 2 key(s) present in .secrets
✗ env: missing or empty values in .env: APP_ENV
✗ vars: .vars not found (copy from .vars.example)

Some checks failed — fill in the missing values before running act.
```

---

### `streep rehearse` — *The Read-Through*

> Every production has a table read before opening night. You catch problems early, nothing is on the line, and everyone finds out what the script actually demands.

Wraps `act -n [event]` — a dry-run that resolves the workflow plan without spinning up containers. Uses flags from `.actrc` automatically.

```bash
streep rehearse                    # dry-run the push event
streep rehearse pull_request
streep rehearse workflow_dispatch
```

If `.actrc` is missing, streep will suggest running `streep new role` first. A performer needs a script before they can rehearse.

---

### `streep perform` — *Lights. Camera. Action.*

> Rehearsals are over. This is the real thing.

Runs `act [event]` for live execution — not a dry-run. Uses your `.actrc`, automatically adds the matching event payload from `.act/events/` if one exists, and records a deterministic fingerprint of the run to `.act/run-fingerprint` on success.

```bash
streep perform                                             # run the default push event
streep perform pull_request                                # run pull_request
streep perform push --workflow .github/workflows/ci.yml   # target a specific workflow
streep perform pull_request --job test                     # target a specific job
```

**Flags:**
- `--job <name>` (`-j`) — run only the specified job
- `--workflow <file>` (`-W`) — target a specific workflow file

---

### `streep clean` — *Strike the Set*

> When the run wraps, the stage crew comes in and clears everything. `streep clean` does the same for your local act environment — with a dry-run by default, because even stage crews take inventory before they start throwing things away.

Removes populated credential and runtime files:

- `.secrets`, `.env`, `.vars`, `.input`, `.actrc`
- contents of `.artifacts/`
- contents of `.act/cache/`

`.example` files and `.gitignore` are always left untouched.

```bash
streep clean             # dry-run: list what would be removed
streep clean --force     # actually remove it
```

---

### `streep doctor` — *The Show Doctor*

> When a production is in trouble, a show doctor is called in — someone who can look at everything with fresh eyes and diagnose exactly what's wrong. `streep doctor` is yours.

Runs a full readiness check before you attempt a performance:

- `act` is installed and available in `PATH`
- Docker is installed and the daemon is reachable
- `.actrc` is present
- `.secrets`, `.env`, `.vars`, and `.input` are all populated (validated against their `.example` counterparts)
- Event payload files exist in `.act/events/`
- `.artifacts/` is present when artifact actions are in use

---

### `streep edit` — *Green Room Rewrites*

> Sometimes the script needs last-minute adjustments before you go on. `streep edit` opens the right file in your `$EDITOR` — or walks you through each key interactively if no editor is configured.

Uses `.*.example` files as the key manifest; secrets are redacted in prompts.

```bash
streep edit secrets     # edit .secrets
streep edit env         # edit .env
streep edit vars        # edit .vars
streep edit input       # edit .input
```

After editing, streep validates that no required keys are empty.

---

### `streep explain` — *Director's Notes*

> Good directors annotate the script. They explain intent, flag dependencies, highlight what the audience will feel and when. `streep explain` does the same for your workflows.

Produces a human-readable summary of everything your workflows do:

- **Triggers** — which events start the workflow
- **Job graph** — dependency relationships between jobs (`needs`)
- **Required credentials** — secrets, env vars, and repository vars referenced
- **External actions** — every `uses:` reference
- **Matrix expansion** — the full set of combinations per job, with row-by-row detail
- **Warnings** — self-hosted runners, oversized matrices, missing `permissions` blocks, deprecated workflow commands

---

### `streep lint` — *The Script Editor's Red Pen*

> No script goes to production without notes. `streep lint` is your script editor: precise, thorough, and entirely without sentiment.

Checks your workflow files and composite actions for:

- **Deprecated action versions** — `actions/checkout@v1` should be `@v4`, and so on
- **Deprecated workflow commands** — `::set-output` and `::save-state` were retired in 2022
- **Missing `permissions` block** — top-level permissions should always be explicit
- **Unreachable jobs** — `needs:` references a job that doesn't exist
- **Undeclared `workflow_dispatch` inputs** — referenced in expressions but not declared
- **Composite action shell safety** — bash-style syntax in steps without `shell: bash`

```bash
streep lint               # report issues
streep lint --fix         # report issues and auto-bump deprecated action versions
```

---

### `streep bundle actions` — *Pack for the Tour*

> A touring production can't depend on the venue having the right props. You pack everything you need. `streep bundle actions` downloads every remote action your workflows reference and locks them to an exact commit SHA, so your runs work offline and deterministically.

Downloads all `uses:` action dependencies into `.act/bundle/` and writes `.act/bundle.lock` with the resolved commit SHAs.

```bash
streep bundle actions
streep bundle actions /path/to/repo
```

---

### `streep hook` — *Technical Crew*

> The technical crew rigs the theatre before the audience arrives. `streep hook` installs git hooks that catch problems before they make it to the stage.

Installs two streep-managed hooks:

- **pre-commit** — runs `streep lint` when workflow files are staged
- **pre-push** — runs `streep check` before every push

Existing unmanaged hooks are never touched. Only hooks with the streep marker are installed or removed.

```bash
streep hook install
streep hook uninstall
```

---

### `streep diff` — *Comparing Drafts*

> Every script goes through revisions. `streep diff` compares the current workflow state against a previous git revision so you can see exactly what changed — not just line diffs, but what those changes mean for your CI.

Reports workflow-level deltas against a git revision (default: `HEAD~1`):

- Added or removed workflow files
- Added or removed jobs
- Added or removed trigger events
- Added or removed required secrets

```bash
streep diff                        # compare to HEAD~1
streep diff main                   # compare to main
streep diff origin/main /path      # compare to origin/main in a specific repo
```

---

### `streep fingerprint` — *A Signature Performance*

> Every Meryl Streep performance is unmistakeable — a unique combination of choices that no one else would make in quite the same way. `streep fingerprint` gives your workflow runs the same treatment: a deterministic hash built from every file that could affect the outcome.

Hashes your workflow files, `.actrc`, credential files, and `bundle.lock` into a single SHA-256 digest and writes it to `.act/run-fingerprint`. This happens automatically after every `streep perform` run.

```bash
streep fingerprint                 # capture a fingerprint now
streep fingerprint /path/to/repo

streep fingerprint compare .act/run-fingerprint other-fingerprint.json
```

Use `compare` to verify two runs were operating on identical inputs — useful for debugging non-deterministic failures.

---

### `streep policy check` — *Studio Safety*

> Studios have policies for a reason. `streep policy check` scans your workflows for security issues that would make a studio lawyer nervous.

Checks for:

- `permissions: write-all` — blanket write access is almost never correct
- `pull_request_target` — powerful and frequently misused; triggers on untrusted code with elevated token access
- Actions not pinned to a full commit SHA — tag-based pins (`@v4`) can be silently overwritten

Rules are configurable via `.streep/policy.yaml`:

```yaml
rules:
  write_all_permissions: true
  pull_request_target: true
  unpinned_actions: true
```

---

### `streep diagnose` — *The Post-Mortem*

> Not every performance lands. When something goes wrong, you need answers fast. `streep diagnose` reads an act run log, matches it against a library of known failure patterns, and tells you exactly what went wrong and how to fix it.

```bash
streep diagnose .act/latest.log
```

Recognises common failure patterns including:

- Docker daemon not running
- Docker socket permission errors
- Missing credentials files
- Failed action resolution (network or reference issues)
- Invalid workflow YAML
- Missing `gh` CLI in runner image
- CodeQL tooling unavailable

When no known pattern matches, streep suggests where to look next.
