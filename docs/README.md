# pipeboard

[![CI](https://github.com/blackwell-systems/pipeboard/actions/workflows/ci.yml/badge.svg)](https://github.com/blackwell-systems/pipeboard/actions/workflows/ci.yml) [![codecov](https://codecov.io/gh/blackwell-systems/pipeboard/branch/main/graph/badge.svg)](https://codecov.io/gh/blackwell-systems/pipeboard) [![Go Report Card](https://goreportcard.com/badge/github.com/blackwell-systems/pipeboard)](https://goreportcard.com/report/github.com/blackwell-systems/pipeboard) [![Go Reference](https://pkg.go.dev/badge/github.com/blackwell-systems/pipeboard.svg)](https://pkg.go.dev/github.com/blackwell-systems/pipeboard) [![Release](https://img.shields.io/github/v/release/blackwell-systems/pipeboard)](https://github.com/blackwell-systems/pipeboard/releases) [![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT) ![Platforms](https://img.shields.io/badge/Platforms-macOS%20%7C%20Linux%20%7C%20Windows%20%7C%20WSL-blue)

**The programmable clipboard router for terminals.**

One command across macOS, Linux, Windows, and WSL. Sync between machines via SSH. Store snippets in S3. Transform clipboard contents with user-defined pipelines.

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

## Four Pillars

### 1. Unified Clipboard

One CLI that works everywhere. No more remembering `pbcopy` vs `xclip` vs `xsel` vs `wl-copy` vs `clip.exe`.

```bash
# Implicit copy (piped input defaults to copy)
echo "hello" | pipeboard

# Explicit copy
echo "hello" | pipeboard copy
pipeboard copy "hello world"

# Paste to stdout
pipeboard paste

# Pipe to other tools
pipeboard paste | jq .

# Copy/paste images (PNG)
cat screenshot.png | pipeboard copy --image
pipeboard paste --image > clipboard.png
```

### 2. SSH Peer Sync

Send clipboard directly between machines over SSH—no cloud, no intermediary.

```bash
pipeboard send dev    # push to dev machine
pipeboard recv mac    # pull from mac
pipeboard peek dev    # view without copying
pipeboard watch dev   # real-time bidirectional sync
```

Requires pipeboard installed on both machines. See [Sync & Remote](sync.md) for setup.

### 3. S3 Remote Slots

Named, persistent clipboard storage for async workflows. Client-side AES-256-GCM encryption.

```bash
pipeboard push kube   # save for later
pipeboard pull kube   # on another machine, hours later
pipeboard show kube   # view without copying
pipeboard slots       # list all slots
pipeboard rm kube     # delete slot
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

### 4. Programmable Transforms

User-defined pipelines to process clipboard in-place. Chain multiple transforms.

```bash
# Single transform
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

Transforms are defined in your config. See [Transforms](transforms.md) for examples.

## Clipboard History

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

## Command Quick Reference

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
| `history [--json]` | Show recent operations |
| `history --local` | Show clipboard content history |
| `history --local --search <q>` | Search clipboard history |
| `recall <index>` | Restore from clipboard history |

**Setup:**
| Command | Description |
|---------|-------------|
| `init` | Interactive configuration wizard |
| `completion <shell>` | Generate shell completions |

See [Commands](commands.md) for full details and flags.

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

See [Configuration](configuration.md) for all options.

## Platform Support

| Platform | Clipboard | Image | Notes |
|----------|-----------|-------|-------|
| macOS | ✓ built-in | ✓ requires pngpaste/impbcopy | |
| Linux (Wayland) | ✓ wl-clipboard | ✓ native | |
| Linux (X11) | ✓ xclip or xsel | ✓ xclip only | |
| Windows | ✓ clip.exe + PowerShell | ✓ PowerShell | |
| WSL | ✓ clip.exe | paste only | |

Backend detection is automatic. Run `pipeboard doctor` to check your setup.

See [Platforms](platforms.md) for platform-specific details.

## Requirements

- **Local clipboard**: Platform tools (see above), auto-detected
- **SSH sync**: SSH access + pipeboard on remote machines
- **S3 slots**: AWS credentials (env, profile, or IAM role)

## Documentation

- [Getting Started](getting-started.md) — Installation and first commands
- [Commands](commands.md) — Full command reference with all flags
- [Transforms](transforms.md) — Build your own clipboard pipelines
- [Configuration](configuration.md) — Config file reference
- [Sync & Remote](sync.md) — SSH peers and S3 slots
- [Platforms](platforms.md) — Platform-specific details

---

pipeboard™ is a product of Blackwell Systems™.
