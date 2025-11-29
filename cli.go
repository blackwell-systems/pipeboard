package main

import (
	"fmt"
	"os"
)

// ANSI color codes for terminal output
const (
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorReset  = "\033[0m"
)

// useColor returns true if color output should be used
func useColor() bool {
	// Disable color if NO_COLOR is set (https://no-color.org/)
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	// Disable color if not a terminal
	if os.Getenv("TERM") == "dumb" {
		return false
	}
	// Check if stdout is a terminal (basic heuristic)
	fi, err := os.Stderr.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// commandHelp provides per-command help text
var commandHelp = map[string]string{
	"copy": `Usage: pipeboard copy [text] [--image]

Copy text or image to clipboard.

Options:
  --image, -i    Copy PNG image from stdin instead of text

Examples:
  echo "hello" | pipeboard copy     Copy text from stdin
  pipeboard copy "hello world"      Copy provided text
  cat image.png | pipeboard copy --image`,

	"paste": `Usage: pipeboard paste [--image]

Paste clipboard contents to stdout.

Options:
  --image, -i    Paste clipboard image as PNG

Examples:
  pipeboard paste                   Print clipboard text
  pipeboard paste | jq .            Pipe to other commands
  pipeboard paste --image > out.png`,

	"clear": `Usage: pipeboard clear

Clear the clipboard contents (best-effort, may not work on all platforms).`,

	"backend": `Usage: pipeboard backend

Show the detected clipboard backend for your platform.
Useful for debugging clipboard issues.`,

	"doctor": `Usage: pipeboard doctor [--json]

Run environment checks to verify clipboard tools are available.
Shows detected backend, available commands, and any issues.

Options:
  --json     Output in JSON format`,

	"push": `Usage: pipeboard push <name>

Push current clipboard contents to a remote slot.

Arguments:
  name    Slot name (e.g., "work", "snippet", "tmp")

Examples:
  pipeboard push work               Push to "work" slot
  pipeboard push kube && ssh server "pipeboard pull kube"`,

	"pull": `Usage: pipeboard pull <name>

Pull a remote slot into the local clipboard.

Arguments:
  name    Slot name to pull

Examples:
  pipeboard pull work               Pull "work" slot to clipboard`,

	"show": `Usage: pipeboard show <name>

Print remote slot contents to stdout without modifying local clipboard.

Arguments:
  name    Slot name to show

Examples:
  pipeboard show work               Print slot contents
  pipeboard show work | jq .        Pipe to other commands`,

	"slots": `Usage: pipeboard slots [--json]

List all remote slots with size and age.

Options:
  --json     Output in JSON format`,

	"rm": `Usage: pipeboard rm <name>

Delete a remote slot.

Arguments:
  name    Slot name to delete`,

	"send": `Usage: pipeboard send [peer]

Send local clipboard directly to a peer's clipboard via SSH.

Arguments:
  peer    Peer name from config (optional, uses defaults.peer if omitted)

Examples:
  pipeboard send                    Send to default peer
  pipeboard send devbox             Send to "devbox" peer`,

	"recv": `Usage: pipeboard recv [peer]

Receive peer's clipboard into local clipboard via SSH.

Arguments:
  peer    Peer name from config (optional, uses defaults.peer if omitted)`,

	"peek": `Usage: pipeboard peek [peer]

Print peer's clipboard to stdout without modifying local clipboard.

Arguments:
  peer    Peer name from config (optional, uses defaults.peer if omitted)`,

	"history": `Usage: pipeboard history [--fx] [--slots] [--peer] [--local] [--json]

Show recent clipboard operations.

Options:
  --fx       Filter to fx transforms only
  --slots    Filter to push/pull/show/rm only
  --peer     Filter to send/recv/peek only
  --local    Show local clipboard history (content snapshots)
  --json     Output in JSON format

Examples:
  pipeboard history                 Show all history
  pipeboard history --fx            Show only transforms
  pipeboard history --local         Show clipboard content history
  pipeboard history --json          Output as JSON`,

	"fx": `Usage: pipeboard fx <name> [name2...] [--dry-run] [--list]

Run transforms on clipboard contents.

Options:
  --dry-run    Preview output without modifying clipboard
  --list       List available transforms from config

Examples:
  pipeboard fx pretty-json              Format JSON in clipboard
  pipeboard fx strip-ansi pretty-json   Chain multiple transforms
  pipeboard fx uppercase --dry-run      Preview without changing clipboard
  pipeboard fx --list                   Show available transforms`,

	"init": `Usage: pipeboard init

Interactive configuration wizard to set up pipeboard.

Creates ~/.config/pipeboard/config.yaml with:
  - Sync backend (local, S3, or none)
  - Peer connections for SSH clipboard sharing
  - Example transforms for clipboard processing

Run this when first installing pipeboard.`,

	"completion": `Usage: pipeboard completion <shell>

Generate shell completion scripts.

Supported shells:
  bash    Bash completion
  zsh     Zsh completion
  fish    Fish completion

Installation:
  # Bash (add to ~/.bashrc)
  source <(pipeboard completion bash)

  # Zsh (add to ~/.zshrc)
  source <(pipeboard completion zsh)

  # Fish
  pipeboard completion fish > ~/.config/fish/completions/pipeboard.fish`,

	"watch": `Usage: pipeboard watch [peer]

Watch and sync clipboard in real-time with a peer.

Monitors both local and remote clipboards, automatically syncing changes
bidirectionally. Great for pair programming or keeping clipboards in sync
across machines.

Arguments:
  peer    Peer name from config (optional, uses defaults.peer if omitted)

Examples:
  pipeboard watch                    Sync with default peer
  pipeboard watch dev                Sync with "dev" peer

Press Ctrl+C to stop watching.`,

	"recall": `Usage: pipeboard recall <index>

Restore a previous clipboard entry from local history.

Use 'pipeboard history --local' to see available entries with their indices.
Index 1 is the most recent entry.

Arguments:
  index   Entry number from history (1 = most recent)

Examples:
  pipeboard history --local          Show clipboard history
  pipeboard recall 1                 Restore most recent entry
  pipeboard recall 3                 Restore third most recent entry`,
}

// stdinHasData returns true if stdin is a pipe (not a terminal)
func stdinHasData() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	// Check if stdin is a pipe or has data
	return (fi.Mode() & os.ModeCharDevice) == 0
}

// hasHelpFlag checks if args contain -h or --help
func hasHelpFlag(args []string) bool {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" {
			return true
		}
	}
	return false
}

// printCommandHelp prints help for a specific command
func printCommandHelp(cmd string) {
	if help, ok := commandHelp[cmd]; ok {
		fmt.Println(help)
	} else {
		printHelp()
	}
}

func printHelp() {
	fmt.Println(`pipeboard - the programmable clipboard router for terminals

Usage:
  pipeboard <command> [args...]
  <stdin> | pipeboard              Piped input defaults to copy

Local clipboard:
  copy [text]          Copy stdin or provided text to clipboard
  copy --image         Copy PNG image from stdin to clipboard
  paste                Paste clipboard contents to stdout
  paste --image        Paste clipboard image as PNG to stdout
  clear                Clear clipboard (best-effort)
  backend              Show detected clipboard backend
  doctor [--json]      Run environment checks

Transforms (programmable clipboard pipelines):
  fx <name> [name2...] Run transform(s) on clipboard (chained, in-place)
  fx <name> --dry-run  Preview output without modifying clipboard
  fx --list            List available transforms

  Chaining: pipeboard fx strip-ansi pretty-json
  Safety: clipboard unchanged if any transform fails

Direct peer-to-peer (SSH):
  send [peer]          Send local clipboard to peer's clipboard
  recv [peer]          Receive peer's clipboard into local clipboard
  peek [peer]          Print peer's clipboard to stdout (no local change)
  watch [peer]         Real-time bidirectional clipboard sync
                       (peer defaults to 'defaults.peer' in config)

Remote slots (S3 or local backend):
  push <name>          Push clipboard to remote slot
  pull <name>          Pull remote slot into clipboard
  show <name>          Print remote slot to stdout
  slots [--json]       List remote slots
  rm <name>            Delete remote slot

History:
  history [--json]     Show recent operations (most recent first)
  history --fx         Filter to fx transforms only
  history --slots      Filter to push/pull/show/rm only
  history --peer       Filter to send/recv/peek only
  history --local      Show local clipboard history (content snapshots)
  recall <index>       Restore entry from clipboard history

Setup:
  init                 Interactive configuration wizard
  completion <shell>   Generate shell completions (bash/zsh/fish)

Other:
  <command> --help     Show help for a specific command
  help                 Show this help
  version              Show version

Config: ~/.config/pipeboard/config.yaml

  defaults:
    peer: dev              # default peer for send/recv/peek

  peers:
    dev:
      ssh: devbox

  fx:                      # clipboard transforms
    pretty-json:
      cmd: ["jq", "."]
      description: "Format JSON"
    strip-ansi:
      shell: "sed 's/\\x1b\\[[0-9;]*m//g'"

  sync:
    backend: local         # or "s3" for cloud sync
    encryption: aes256     # client-side encryption (optional)
    passphrase: secret     # encryption passphrase
    ttl_days: 30           # auto-expire slots (optional)
    # For S3 backend:
    # s3:
    #   bucket: my-bucket
    #   region: us-west-2

Examples:
  echo "hello" | pipeboard             # implicit copy
  pipeboard paste | jq .
  pipeboard fx pretty-json           # format JSON in clipboard
  pipeboard fx strip-ansi --dry-run  # preview transform
  pipeboard send                     # uses default peer
  pipeboard send dev
  pipeboard push kube && ssh server "pipeboard pull kube"
  cat screenshot.png | pipeboard copy --image
  pipeboard paste --image > clipboard.png`)
}

// printError prints an error message to stderr with optional color
func printError(err error) {
	if useColor() {
		fmt.Fprintf(os.Stderr, "%spipeboard: %v%s\n", colorRed, err, colorReset)
	} else {
		fmt.Fprintf(os.Stderr, "pipeboard: %v\n", err)
	}
}
