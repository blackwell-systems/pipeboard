package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func cmdCompletion(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: pipeboard completion <bash|zsh|fish>")
	}

	shell := args[0]
	switch shell {
	case "bash":
		fmt.Print(bashCompletion)
	case "zsh":
		fmt.Print(zshCompletion)
	case "fish":
		fmt.Print(fishCompletion)
	default:
		return fmt.Errorf("unknown shell: %s (supported: bash, zsh, fish)", shell)
	}
	return nil
}

const bashCompletion = `# pipeboard bash completion
# Add to ~/.bashrc or /etc/bash_completion.d/pipeboard

_pipeboard() {
    local cur prev commands
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"

    commands="copy paste clear push pull show slots rm send recv peek watch history recall fx backend doctor init completion help version"

    case "${prev}" in
        pipeboard)
            COMPREPLY=( $(compgen -W "${commands}" -- ${cur}) )
            return 0
            ;;
        completion)
            COMPREPLY=( $(compgen -W "bash zsh fish" -- ${cur}) )
            return 0
            ;;
        fx)
            # Complete with --list, --dry-run, or transform names from config
            local fx_opts="--list --dry-run"
            COMPREPLY=( $(compgen -W "${fx_opts}" -- ${cur}) )
            return 0
            ;;
        push|pull|show|rm)
            # Could complete slot names here if we cached them
            return 0
            ;;
        send|recv|peek|watch)
            # Could complete peer names here if we cached them
            return 0
            ;;
        history)
            COMPREPLY=( $(compgen -W "--fx --slots --peer --local --json" -- ${cur}) )
            return 0
            ;;
        slots|doctor)
            COMPREPLY=( $(compgen -W "--json" -- ${cur}) )
            return 0
            ;;
        copy|paste)
            COMPREPLY=( $(compgen -W "--image" -- ${cur}) )
            return 0
            ;;
        *)
            ;;
    esac

    # Complete with --help for any command
    if [[ ${cur} == -* ]]; then
        COMPREPLY=( $(compgen -W "--help" -- ${cur}) )
        return 0
    fi
}

complete -F _pipeboard pipeboard
`

const zshCompletion = `#compdef pipeboard
# pipeboard zsh completion
# Add to ~/.zshrc or place in $fpath as _pipeboard

_pipeboard() {
    local -a commands
    commands=(
        'copy:Copy text or image to clipboard'
        'paste:Paste from clipboard to stdout'
        'clear:Clear the clipboard'
        'push:Push clipboard to a named slot'
        'pull:Pull from a named slot to clipboard'
        'show:Show contents of a slot without copying'
        'slots:List all available slots'
        'rm:Delete a slot'
        'send:Send clipboard to a peer'
        'recv:Receive clipboard from a peer'
        'peek:View peer clipboard without copying'
        'watch:Real-time bidirectional clipboard sync'
        'history:Show clipboard operation history'
        'recall:Restore entry from clipboard history'
        'fx:Run transforms on clipboard'
        'backend:Show detected clipboard backend'
        'doctor:Check system clipboard setup'
        'init:Initialize pipeboard configuration'
        'completion:Generate shell completions'
        'help:Show help'
        'version:Show version'
    )

    _arguments -C \
        '1: :->command' \
        '*: :->args'

    case $state in
        command)
            _describe -t commands 'pipeboard commands' commands
            ;;
        args)
            case $words[2] in
                completion)
                    _values 'shell' bash zsh fish
                    ;;
                fx)
                    _arguments \
                        '--list[List available transforms]' \
                        '--dry-run[Preview without modifying clipboard]'
                    ;;
                history)
                    _arguments \
                        '--fx[Show only transform operations]' \
                        '--slots[Show only slot operations]' \
                        '--peer[Show only peer operations]' \
                        '--local[Show local clipboard history]' \
                        '--json[Output in JSON format]'
                    ;;
                slots|doctor)
                    _arguments \
                        '--json[Output in JSON format]'
                    ;;
                copy|paste)
                    _arguments \
                        '--image[Copy/paste image instead of text]'
                    ;;
                push|pull|show|rm)
                    # Slot name completion would go here
                    ;;
                send|recv|peek|watch)
                    # Peer name completion would go here
                    ;;
                *)
                    _arguments '--help[Show command help]'
                    ;;
            esac
            ;;
    esac
}

_pipeboard "$@"
`

const fishCompletion = `# pipeboard fish completion
# Add to ~/.config/fish/completions/pipeboard.fish

# Disable file completion by default
complete -c pipeboard -f

# Main commands
complete -c pipeboard -n "__fish_use_subcommand" -a "copy" -d "Copy text or image to clipboard"
complete -c pipeboard -n "__fish_use_subcommand" -a "paste" -d "Paste from clipboard to stdout"
complete -c pipeboard -n "__fish_use_subcommand" -a "clear" -d "Clear the clipboard"
complete -c pipeboard -n "__fish_use_subcommand" -a "push" -d "Push clipboard to a named slot"
complete -c pipeboard -n "__fish_use_subcommand" -a "pull" -d "Pull from a named slot to clipboard"
complete -c pipeboard -n "__fish_use_subcommand" -a "show" -d "Show contents of a slot"
complete -c pipeboard -n "__fish_use_subcommand" -a "slots" -d "List all available slots"
complete -c pipeboard -n "__fish_use_subcommand" -a "rm" -d "Delete a slot"
complete -c pipeboard -n "__fish_use_subcommand" -a "send" -d "Send clipboard to a peer"
complete -c pipeboard -n "__fish_use_subcommand" -a "recv" -d "Receive clipboard from a peer"
complete -c pipeboard -n "__fish_use_subcommand" -a "peek" -d "View peer clipboard"
complete -c pipeboard -n "__fish_use_subcommand" -a "watch" -d "Real-time clipboard sync"
complete -c pipeboard -n "__fish_use_subcommand" -a "history" -d "Show operation history"
complete -c pipeboard -n "__fish_use_subcommand" -a "recall" -d "Restore from clipboard history"
complete -c pipeboard -n "__fish_use_subcommand" -a "fx" -d "Run transforms on clipboard"
complete -c pipeboard -n "__fish_use_subcommand" -a "backend" -d "Show clipboard backend"
complete -c pipeboard -n "__fish_use_subcommand" -a "doctor" -d "Check system setup"
complete -c pipeboard -n "__fish_use_subcommand" -a "init" -d "Initialize configuration"
complete -c pipeboard -n "__fish_use_subcommand" -a "completion" -d "Generate shell completions"
complete -c pipeboard -n "__fish_use_subcommand" -a "help" -d "Show help"
complete -c pipeboard -n "__fish_use_subcommand" -a "version" -d "Show version"

# completion subcommand
complete -c pipeboard -n "__fish_seen_subcommand_from completion" -a "bash zsh fish"

# fx options
complete -c pipeboard -n "__fish_seen_subcommand_from fx" -l list -d "List available transforms"
complete -c pipeboard -n "__fish_seen_subcommand_from fx" -l dry-run -d "Preview without modifying"

# history options
complete -c pipeboard -n "__fish_seen_subcommand_from history" -l fx -d "Show only transforms"
complete -c pipeboard -n "__fish_seen_subcommand_from history" -l slots -d "Show only slot ops"
complete -c pipeboard -n "__fish_seen_subcommand_from history" -l peer -d "Show only peer ops"
complete -c pipeboard -n "__fish_seen_subcommand_from history" -l local -d "Show clipboard history"
complete -c pipeboard -n "__fish_seen_subcommand_from history" -l json -d "Output as JSON"

# slots/doctor options
complete -c pipeboard -n "__fish_seen_subcommand_from slots doctor" -l json -d "Output as JSON"

# copy/paste options
complete -c pipeboard -n "__fish_seen_subcommand_from copy paste" -l image -d "Image mode"

# Global --help
complete -c pipeboard -l help -d "Show help"
`

// Helper to install completions
func installCompletion(shell string) error {
	var path, content string

	switch shell {
	case "bash":
		path = os.ExpandEnv("$HOME/.local/share/bash-completion/completions/pipeboard")
		content = bashCompletion
	case "zsh":
		path = os.ExpandEnv("$HOME/.zsh/completions/_pipeboard")
		content = zshCompletion
	case "fish":
		path = os.ExpandEnv("$HOME/.config/fish/completions/pipeboard.fish")
		content = fishCompletion
	default:
		return fmt.Errorf("unknown shell: %s", shell)
	}

	// Create parent directory
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing completion file: %w", err)
	}

	fmt.Printf("Installed %s completion to %s\n", shell, path)
	return nil
}
