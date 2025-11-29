# pipeboard

[![CI](https://github.com/blackwell-systems/pipeboard/actions/workflows/ci.yml/badge.svg)](https://github.com/blackwell-systems/pipeboard/actions/workflows/ci.yml) [![Tests](https://img.shields.io/badge/tests-400-success)](https://github.com/blackwell-systems/pipeboard) [![codecov](https://codecov.io/gh/blackwell-systems/pipeboard/branch/main/graph/badge.svg)](https://codecov.io/gh/blackwell-systems/pipeboard) [![Go Report Card](https://goreportcard.com/badge/github.com/blackwell-systems/pipeboard)](https://goreportcard.com/report/github.com/blackwell-systems/pipeboard) [![Go Reference](https://pkg.go.dev/badge/github.com/blackwell-systems/pipeboard.svg)](https://pkg.go.dev/github.com/blackwell-systems/pipeboard) [![Release](https://img.shields.io/github/v/release/blackwell-systems/pipeboard)](https://github.com/blackwell-systems/pipeboard/releases) [![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT) ![Platforms](https://img.shields.io/badge/Platforms-macOS%20%7C%20Linux%20%7C%20Windows%20%7C%20WSL-blue) [![Homebrew Tap](https://img.shields.io/badge/Homebrew-blackwell--systems%2Ftap-black.svg?logo=homebrew)](https://github.com/blackwell-systems/homebrew-tap) [![Go Install](https://img.shields.io/badge/go_install-@latest-00ADD8?logo=go)](https://pkg.go.dev/github.com/blackwell-systems/pipeboard) [![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](https://github.com/blackwell-systems/pipeboard/issues)

**The programmable clipboard router for terminals.**

One command across macOS, Linux, Windows, and WSL. Sync between machines via SSH. Store snippets in S3. Transform clipboard contents with user-defined pipelines.

## Installation

**Homebrew (macOS/Linux):**
```bash
brew install blackwell-systems/tap/pipeboard
```

**Go:**
```bash
go install github.com/blackwell-systems/pipeboard@latest
```

**Download binary:**
```bash
# macOS (Apple Silicon)
curl -LO https://github.com/blackwell-systems/pipeboard/releases/latest/download/pipeboard_darwin_arm64.tar.gz
tar xzf pipeboard_darwin_arm64.tar.gz
sudo mv pipeboard /usr/local/bin/

# Linux (x86_64)
curl -LO https://github.com/blackwell-systems/pipeboard/releases/latest/download/pipeboard_linux_amd64.tar.gz
tar xzf pipeboard_linux_amd64.tar.gz
sudo mv pipeboard /usr/local/bin/
```

**From source:**
```bash
git clone https://github.com/blackwell-systems/pipeboard.git
cd pipeboard && go build
```

## Uninstall

**Quick uninstall:**
```bash
curl -sSL https://raw.githubusercontent.com/blackwell-systems/pipeboard/main/uninstall.sh | sh
```

**Homebrew:**
```bash
brew uninstall pipeboard
```

**Manual:**
```bash
sudo rm /usr/local/bin/pipeboard
rm -rf ~/.config/pipeboard  # optional: remove config
```

## Why pipeboard?

**Stop context-switching.** Terminal users copy and paste hundreds of times a day—config snippets, JSON payloads, API keys, log excerpts. Each time you reach for a GUI or email yourself a snippet, you break flow.

pipeboard keeps everything in the terminal:

| Workflow | Without pipeboard | With pipeboard |
|----------|-------------------|----------------|
| Cross-platform scripts | `pbcopy` on Mac, `xclip` on Linux, `clip.exe` on WSL | `pipeboard copy` everywhere |
| Send clipboard to another machine | SSH, scp, or paste into Slack | `pipeboard send dev` |
| Grab that config you saved yesterday | Scroll through chat history | `pipeboard pull kube` |
| Format JSON before pasting into docs | Copy → browser → jsonformatter.org → copy | `pipeboard fx pretty-json` |
| Redact secrets before sharing with LLMs | Manual find-and-replace | `pipeboard fx redact-secrets` |
| Screenshot to clipboard | Platform-specific tools | `pipeboard paste --image > out.png` |

**Four pillars:**
1. **Unified clipboard** — One CLI, every platform (macOS, Linux, Windows, WSL)
2. **SSH peer sync** — Direct machine-to-machine transfer, no cloud required
3. **S3 remote slots** — Named, persistent storage for async workflows
4. **Programmable transforms** — User-defined pipelines to process clipboard in-place

## Usage

### Local Clipboard

```bash
# Implicit copy (piped input defaults to copy)
echo "hello" | pipeboard

# Explicit copy from stdin
echo "hello" | pipeboard copy

# Copy text directly
pipeboard copy "hello world"

# Paste to stdout
pipeboard paste

# Pipe to other tools
pipeboard paste | jq .

# Clear clipboard
pipeboard clear

# Copy/paste images (PNG)
cat screenshot.png | pipeboard copy --image
pipeboard paste --image > clipboard.png

# Check detected backend
pipeboard backend

# Diagnose issues
pipeboard doctor
```

### Transforms (fx)

Define transforms in your config, then process clipboard contents in-place:

```bash
# Format JSON in clipboard
pipeboard fx pretty-json

# Chain multiple transforms
pipeboard fx strip-ansi redact-secrets pretty-json

# Preview without modifying clipboard
pipeboard fx strip-ansi --dry-run

# List available transforms
pipeboard fx --list
```

**Safety guarantees:**
- `--dry-run` prints to stdout, never touches clipboard
- If any transform in a chain fails, clipboard is unchanged
- Empty output = error, clipboard unchanged

Transforms are defined in your config with either `cmd` (array) or `shell` (string) syntax:

```yaml
fx:
  pretty-json:
    cmd: ["jq", "."]
    description: "Format JSON"

  strip-ansi:
    shell: "sed 's/\\x1b\\[[0-9;]*m//g'"
    description: "Remove ANSI escape codes"

  redact-secrets:
    shell: "sed -E 's/(AKIA[0-9A-Z]{16}|sk-[a-zA-Z0-9]{48})/<REDACTED>/g'"
    description: "Redact AWS keys and OpenAI tokens"

  yaml-to-json:
    cmd: ["yq", "-o", "json"]
    description: "Convert YAML to JSON"

  base64-decode:
    cmd: ["base64", "-d"]
    description: "Decode base64"

  sort-lines:
    shell: "sort | uniq"
    description: "Sort and deduplicate lines"
```

**Why this matters:** Every day you copy JSON that needs formatting, logs with ANSI codes, configs with secrets you need to redact before sharing. Instead of context-switching to browser tools or writing one-off scripts, define the transform once and run it forever.

### Peer-to-Peer Sync (SSH)

Send clipboard directly between machines over SSH—no cloud, no intermediary:

```bash
# Send local clipboard to peer's clipboard
pipeboard send dev

# Receive peer's clipboard into local clipboard
pipeboard recv dev

# View peer's clipboard without copying locally
pipeboard peek dev

# Real-time bidirectional sync (great for pair programming)
pipeboard watch dev
```

Requires pipeboard installed on both machines. Peers are defined in config:

```yaml
peers:
  dev:
    ssh: devbox              # SSH host/alias from ~/.ssh/config
  mac:
    ssh: dayna-mac.local
```

### Remote Slots (S3)

Named, persistent clipboard storage for async workflows:

```bash
# Push local clipboard to a named slot
pipeboard push myslot

# Pull from a slot to local clipboard
pipeboard pull myslot

# View slot contents without copying to clipboard
pipeboard show myslot

# List all slots
pipeboard slots

# Delete a slot
pipeboard rm myslot
```

**Example:** Share a kubeconfig between laptop and server.

```bash
# On laptop
cat ~/.kube/config | pipeboard copy
pipeboard push kube

# On server (hours later)
pipeboard pull kube
pipeboard paste > ~/.kube/config
```

Features: client-side AES-256-GCM encryption, configurable TTL/expiry, server-side encryption (SSE).

### Clipboard History

Track and restore previous clipboard contents:

```bash
# Show clipboard content history
pipeboard history --local

# Search history for specific content
pipeboard history --local --search "api key"

# Restore most recent entry
pipeboard recall 1

# Restore third most recent
pipeboard recall 3
```

The local clipboard history stores up to 20 entries with automatic deduplication. When encryption is enabled in your sync config, clipboard history is also encrypted at rest.

**Tip:** Add an alias for convenience:

```bash
alias pb='pipeboard'
```

## Configuration

Config file: `~/.config/pipeboard/config.yaml`

```yaml
version: 1

defaults:
  peer: dev                  # default for send/recv/peek

peers:
  dev:
    ssh: devbox              # SSH host from ~/.ssh/config
  mac:
    ssh: dayna-mac.local

fx:
  pretty-json:
    cmd: ["jq", "."]
  redact-secrets:
    shell: "sed -E 's/AKIA[0-9A-Z]{16}/<REDACTED>/g'"

sync:
  backend: s3
  encryption: aes256         # client-side AES-256-GCM
  passphrase: ${PIPEBOARD_PASSPHRASE}
  ttl_days: 30               # auto-expire after 30 days
  s3:
    bucket: my-pipeboard
    region: us-west-2
    prefix: clips/
    sse: AES256              # server-side encryption
```

### Environment Variables

```bash
PIPEBOARD_CONFIG           # config file path
PIPEBOARD_BACKEND          # sync backend type
PIPEBOARD_S3_BUCKET        # S3 bucket name
PIPEBOARD_S3_REGION        # AWS region
PIPEBOARD_S3_PREFIX        # key prefix
PIPEBOARD_S3_PROFILE       # AWS profile name
PIPEBOARD_S3_SSE           # server-side encryption
```

## Command Reference

**Local clipboard:**
| Command | Description |
|---------|-------------|
| `copy [text]` | Copy stdin or text to clipboard |
| `copy --image` | Copy PNG from stdin |
| `paste` | Output clipboard to stdout |
| `paste --image` | Output clipboard image as PNG |
| `clear` | Clear clipboard |
| `backend` | Show detected backend |
| `doctor [--json]` | Check environment |

**Transforms:**
| Command | Description |
|---------|-------------|
| `fx <name> [name2...]` | Run transform(s) in-place (chainable) |
| `fx <name> --dry-run` | Preview without modifying clipboard |
| `fx --list` | List available transforms |

**SSH peer sync:**
| Command | Description |
|---------|-------------|
| `send [peer]` | Send clipboard to peer |
| `recv [peer]` | Receive from peer |
| `peek [peer]` | View peer's clipboard |
| `watch [peer]` | Real-time bidirectional sync |

**S3 remote slots:**
| Command | Description |
|---------|-------------|
| `push <slot>` | Push clipboard to slot |
| `pull <slot>` | Pull from slot |
| `show <slot>` | View slot contents |
| `slots [--json]` | List all slots |
| `rm <slot>` | Delete slot |

**History:**
| Command | Description |
|---------|-------------|
| `history [--json]` | Show recent operations (newest first) |
| `history --fx` | Filter to fx transforms |
| `history --slots` | Filter to push/pull/show/rm |
| `history --peer` | Filter to send/recv/peek/watch |
| `history --local` | Show clipboard content history |
| `history --local --search <query>` | Search clipboard history |
| `recall <index>` | Restore from clipboard history |

**Setup:**
| Command | Description |
|---------|-------------|
| `init` | Interactive configuration wizard |
| `completion <shell>` | Generate shell completions (bash/zsh/fish) |

## Platform Support

| Platform | Clipboard | Image | Notes |
|----------|-----------|-------|-------|
| macOS | ✓ built-in | ✓ requires pngpaste/impbcopy | |
| Linux (Wayland) | ✓ wl-clipboard | ✓ native | |
| Linux (X11) | ✓ xclip or xsel | ✓ xclip only | |
| Windows | ✓ clip.exe + PowerShell | ✓ PowerShell | |
| WSL | ✓ clip.exe | paste only | |

Backend detection is automatic. Run `pipeboard doctor` to check your setup.

## Requirements

- **Local clipboard**: Platform tools (see above), auto-detected
- **SSH sync**: SSH access + pipeboard on remote machines
- **S3 slots**: AWS credentials (env, profile, or IAM role)

## Security

pipeboard handles potentially sensitive clipboard data. See [SECURITY.md](SECURITY.md) for details on:
- Data handling (no logging, no telemetry)
- Transport security (SSH, HTTPS/S3)
- Configuration recommendations

## Trademarks

pipeboard™ is a product of Blackwell Systems™.

Blackwell Systems™ and the Blackwell Systems logo are trademarks of Dayna Blackwell. You may use the name "Blackwell Systems" to refer to this project, but you may not use the name or logo in a way that suggests endorsement or official affiliation without prior written permission. See [BRAND.md](BRAND.md) for usage guidelines.

## License

MIT
