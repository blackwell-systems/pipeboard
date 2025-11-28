# pipeboard

[![Go Reference](https://pkg.go.dev/badge/github.com/blackwell-systems/pipeboard.svg)](https://pkg.go.dev/github.com/blackwell-systems/pipeboard)
[![CI](https://github.com/blackwell-systems/pipeboard/actions/workflows/ci.yml/badge.svg)](https://github.com/blackwell-systems/pipeboard/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/blackwell-systems/pipeboard)](https://goreportcard.com/report/github.com/blackwell-systems/pipeboard)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)
[![Release](https://img.shields.io/github/v/release/blackwell-systems/pipeboard)](https://github.com/blackwell-systems/pipeboard/releases)

A tiny cross-platform clipboard CLI with peer-to-peer SSH sync and optional S3 remote slots.

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

## Why pipeboard?

| Problem | Without pipeboard | With pipeboard |
|---------|-------------------|----------------|
| Different clipboard tools per OS | `pbcopy` on Mac, `xclip` on Linux, `clip.exe` on WSL | `pipeboard copy/paste` everywhere |
| Copy between machines | Manual SSH, scp, or shared files | `pipeboard send dev` |
| Persistent clipboard across reboots | Not possible | `pipeboard push/pull` via S3 |
| Share config snippets async | Slack/email yourself | `pipeboard push kube` on laptop, `pipeboard pull kube` on server |

## Usage

### Local Clipboard

```bash
# Copy from stdin
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

### Direct Peer-to-Peer (SSH)

Send clipboard contents directly between machines via SSH:

```bash
# Send local clipboard to peer's clipboard
pipeboard send dev

# Receive peer's clipboard into local clipboard
pipeboard recv dev

# View peer's clipboard without copying locally
pipeboard peek dev
```

**Example workflow:**

```bash
# On laptop: send to devbox
pipeboard send dev

# On laptop: receive from devbox
pipeboard recv dev

# On laptop: peek at devbox clipboard
pipeboard peek dev | head
```

### Remote Slots (S3)

Store clipboard contents in named slots for async access:

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

**Example workflow:**

```bash
# On laptop
cat ~/.kube/config | pipeboard copy
pipeboard push kube

# On server
pipeboard pull kube
pipeboard paste > ~/.kube/config
```

**Tip:** Add an alias for convenience:

```bash
alias pb='pipeboard'
```

### Transforms (fx)

Define custom clipboard transforms in your config, then run them with `pipeboard fx`:

```bash
# Format JSON in clipboard
pipeboard fx pretty-json

# Preview transform output without modifying clipboard
pipeboard fx strip-ansi --dry-run

# List available transforms
pipeboard fx --list
```

**Example transforms in config:**

```yaml
fx:
  pretty-json:
    cmd: ["jq", "."]
    description: "Format JSON"

  strip-ansi:
    shell: "sed 's/\\x1b\\[[0-9;]*m//g'"
    description: "Remove ANSI escape codes"

  redact-secrets:
    shell: "sed -E 's/AKIA[0-9A-Z]{16}/<AWS_KEY>/g'"
    description: "Redact AWS keys"

  yaml-to-json:
    cmd: ["yq", "-o", "json"]
    description: "Convert YAML to JSON"

  base64-decode:
    cmd: ["base64", "-d"]
    description: "Decode base64"
```

**Use cases:**
- Pretty-print JSON/YAML before pasting
- Redact secrets before sharing in Slack/LLMs
- Strip ANSI codes from terminal output
- Convert between formats (JSON↔YAML, base64)
- Normalize configs before pushing to S3

## Configuration

Config file: `~/.config/pipeboard/config.yaml`

```yaml
version: 1

# Default settings
defaults:
  peer: dev                  # default peer for send/recv/peek (optional)

# Direct peer-to-peer via SSH
peers:
  dev:
    ssh: devbox              # SSH host/alias from ~/.ssh/config
    remote_cmd: pipeboard    # optional, default: pipeboard
  mac:
    ssh: dayna-mac.local
  wsl:
    ssh: wsl-host

# Clipboard transforms
fx:
  pretty-json:
    cmd: ["jq", "."]
    description: "Format JSON"
  strip-ansi:
    shell: "sed 's/\\x1b\\[[0-9;]*m//g'"

# Optional remote sync backend
sync:
  backend: s3                # or "none"
  encryption: aes256         # optional: client-side encryption
  passphrase: your-secret    # required if encryption is aes256
  ttl_days: 30               # optional: auto-expire slots after N days
  s3:
    bucket: your-bucket-name
    region: us-west-2
    prefix: username/slots/  # optional
    profile: pipeboard       # optional AWS profile
    sse: AES256              # optional: AES256 or aws:kms (server-side)
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

## Commands

| Command | Description |
|---------|-------------|
| `copy [text]` | Copy stdin or provided text to clipboard |
| `copy --image` | Copy PNG image from stdin to clipboard |
| `paste` | Output clipboard contents to stdout |
| `paste --image` | Paste clipboard image as PNG to stdout |
| `clear` | Clear the clipboard |
| `backend` | Show detected clipboard backend |
| `doctor` | Check dependencies and environment |
| `fx <name>` | Run clipboard transform (in-place) |
| `fx <name> --dry-run` | Preview transform without modifying clipboard |
| `fx --list` | List available transforms |
| `send [peer]` | Send clipboard to peer via SSH (uses default if no peer) |
| `recv [peer]` | Receive from peer via SSH (uses default if no peer) |
| `peek [peer]` | Print peer's clipboard to stdout (uses default if no peer) |
| `push <name>` | Push clipboard to remote slot |
| `pull <name>` | Pull from remote slot to clipboard |
| `show <name>` | Print remote slot to stdout |
| `slots` | List remote slots |
| `rm <name>` | Delete a remote slot |
| `history` | Show recent operations |
| `help` | Show help |
| `version` | Show version |

## Supported Backends

### Local Clipboard

| Platform | Backend | Tools |
|----------|---------|-------|
| macOS | darwin-pasteboard | pbcopy, pbpaste |
| Linux (Wayland) | wayland-wl-copy | wl-copy, wl-paste |
| Linux (X11) | x11-xclip | xclip or xsel |
| WSL | wsl-clip | clip.exe, powershell.exe |

Backend detection is automatic based on environment variables (`WAYLAND_DISPLAY`, `DISPLAY`) and available commands.

### Remote Sync

| Backend | Storage | Auth |
|---------|---------|------|
| s3 | AWS S3 | AWS credentials/profile |

## Requirements

### Local Clipboard

- **macOS**: No additional tools needed (pbcopy/pbpaste are built-in)
- **Wayland**: `wl-clipboard` package
- **X11**: `xclip` or `xsel` package
- **WSL**: Windows clipboard tools should be available automatically

### Peer-to-Peer (SSH)

- SSH access to peer machines
- pipeboard installed on remote machines

### Remote Sync (S3)

- AWS credentials configured (via environment, profile, or IAM role)
- S3 bucket with appropriate permissions

## Security

pipeboard handles potentially sensitive clipboard data. See [SECURITY.md](SECURITY.md) for details on:
- Data handling (no logging, no telemetry)
- Transport security (SSH, HTTPS/S3)
- Configuration recommendations

## Trademarks

Blackwell Systems™ and the Blackwell Systems logo are trademarks of Dayna Blackwell. You may use the name "Blackwell Systems" to refer to this project, but you may not use the name or logo in a way that suggests endorsement or official affiliation without prior written permission. See [BRAND.md](BRAND.md) for usage guidelines.

## License

MIT
