package cmd

import (
	"fmt"
	"io"
)

const completionUsage = `Generate shell completion scripts for streep.

Usage:
  streep completion <shell>

Shells:
  bash    Generate bash completion script
  zsh     Generate zsh completion script
  fish    Generate fish completion script

Examples:
  # Bash (add to ~/.bashrc or ~/.bash_profile)
  source <(streep completion bash)

  # Zsh (add to ~/.zshrc)
  source <(streep completion zsh)

  # Fish (save to completions directory)
  streep completion fish > ~/.config/fish/completions/streep.fish
`

const bashCompletion = `# streep bash completion
_streep() {
    local cur prev commands
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    commands="new check rehearse perform clean doctor edit explain lint bundle hook diff fingerprint policy diagnose version update completion help"

    if [[ ${COMP_CWORD} -eq 1 ]]; then
        COMPREPLY=( $(compgen -W "${commands}" -- "${cur}") )
        return 0
    fi

    case "${prev}" in
        completion)
            COMPREPLY=( $(compgen -W "bash zsh fish" -- "${cur}") )
            return 0
            ;;
        help)
            COMPREPLY=( $(compgen -W "${commands}" -- "${cur}") )
            return 0
            ;;
    esac
}
complete -F _streep streep
`

const zshCompletion = `#compdef streep
# streep zsh completion
_streep() {
    local -a commands
    commands=(
        'new:Create new streep resources'
        'check:Validate that act credential files are ready'
        'rehearse:Dry-run workflows locally with act -n'
        'perform:Run workflows locally with act'
        'clean:Remove local act runtime files'
        'doctor:Diagnose local act readiness'
        'edit:Edit .secrets/.env/.vars/.input files'
        'explain:Explain workflow intent and structure'
        'lint:Lint workflow files for common issues'
        'bundle:Bundle actions for offline use'
        'hook:Manage git hooks for workflow checks'
        'diff:Show workflow changes vs a git revision'
        'fingerprint:Generate or compare run fingerprints'
        'policy:Run workflow security policy checks'
        'diagnose:Analyze run logs and suggest fixes'
        'version:Print the version'
        'update:Check for a newer version'
        'completion:Generate shell completion scripts'
        'help:Show help for a command'
    )

    if (( CURRENT == 2 )); then
        _describe 'command' commands
    elif (( CURRENT == 3 )); then
        case "${words[2]}" in
            completion) _values 'shell' bash zsh fish ;;
            help)       _describe 'command' commands ;;
        esac
    fi
}
_streep "$@"
`

const fishCompletion = `# streep fish completion
set -l commands new check rehearse perform clean doctor edit explain lint bundle hook diff fingerprint policy diagnose version update completion help

complete -c streep -f
complete -c streep -n "__fish_use_subcommand" -a new          -d "Create new streep resources"
complete -c streep -n "__fish_use_subcommand" -a check        -d "Validate that act credential files are ready"
complete -c streep -n "__fish_use_subcommand" -a rehearse     -d "Dry-run workflows locally with act -n"
complete -c streep -n "__fish_use_subcommand" -a perform      -d "Run workflows locally with act"
complete -c streep -n "__fish_use_subcommand" -a clean        -d "Remove local act runtime files"
complete -c streep -n "__fish_use_subcommand" -a doctor       -d "Diagnose local act readiness"
complete -c streep -n "__fish_use_subcommand" -a edit         -d "Edit .secrets/.env/.vars/.input files"
complete -c streep -n "__fish_use_subcommand" -a explain      -d "Explain workflow intent and structure"
complete -c streep -n "__fish_use_subcommand" -a lint         -d "Lint workflow files for common issues"
complete -c streep -n "__fish_use_subcommand" -a bundle       -d "Bundle actions for offline use"
complete -c streep -n "__fish_use_subcommand" -a hook         -d "Manage git hooks for workflow checks"
complete -c streep -n "__fish_use_subcommand" -a diff         -d "Show workflow changes vs a git revision"
complete -c streep -n "__fish_use_subcommand" -a fingerprint  -d "Generate or compare run fingerprints"
complete -c streep -n "__fish_use_subcommand" -a policy       -d "Run workflow security policy checks"
complete -c streep -n "__fish_use_subcommand" -a diagnose     -d "Analyze run logs and suggest fixes"
complete -c streep -n "__fish_use_subcommand" -a version      -d "Print the version"
complete -c streep -n "__fish_use_subcommand" -a update       -d "Check for a newer version"
complete -c streep -n "__fish_use_subcommand" -a completion   -d "Generate shell completion scripts"
complete -c streep -n "__fish_use_subcommand" -a help         -d "Show help for a command"
complete -c streep -n "__fish_seen_subcommand_from completion" -a "bash zsh fish"
`

func executeCompletion(args []string, stdout io.Writer, _ io.Writer) error {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			_, err := io.WriteString(stdout, completionUsage)
			return err
		}
	}

	if len(args) == 0 {
		_, err := io.WriteString(stdout, completionUsage)
		return err
	}

	switch args[0] {
	case "bash":
		_, err := io.WriteString(stdout, bashCompletion)
		return err
	case "zsh":
		_, err := io.WriteString(stdout, zshCompletion)
		return err
	case "fish":
		_, err := io.WriteString(stdout, fishCompletion)
		return err
	default:
		return fmt.Errorf("unknown shell %q — supported shells: bash, zsh, fish", args[0])
	}
}
