# pipeboard

[![Blackwell Systems™](https://raw.githubusercontent.com/blackwell-systems/blackwell-docs-theme/main/badge-trademark.svg)](https://github.com/blackwell-systems)
[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Platform](https://img.shields.io/badge/Platform-macOS%20%7C%20Linux%20%7C%20Windows%20%7C%20WSL-blue)](https://github.com/blackwell-systems/pipeboard)
[![Go Report Card](https://goreportcard.com/badge/github.com/blackwell-systems/pipeboard)](https://goreportcard.com/report/github.com/blackwell-systems/pipeboard)
[![CI](https://github.com/blackwell-systems/pipeboard/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/blackwell-systems/pipeboard/actions/workflows/ci.yml)

[![Homebrew](https://img.shields.io/badge/Homebrew-blackwell--systems-black?logo=homebrew)](https://github.com/blackwell-systems/homebrew-tap)
[![Go Install](https://img.shields.io/badge/go_install-@latest-00ADD8?logo=go&logoColor=white)](https://pkg.go.dev/github.com/blackwell-systems/pipeboard)
[![Dependabot](https://img.shields.io/badge/Dependabot-enabled-brightgreen?logo=dependabot)](https://github.com/blackwell-systems/pipeboard/security/dependabot)
[![Release](https://img.shields.io/github/v/release/blackwell-systems/pipeboard)](https://github.com/blackwell-systems/pipeboard/releases)

[![Tests](https://img.shields.io/badge/tests-565-success)](https://github.com/blackwell-systems/pipeboard)
[![codecov](https://codecov.io/gh/blackwell-systems/pipeboard/branch/main/graph/badge.svg)](https://codecov.io/gh/blackwell-systems/pipeboard)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](https://github.com/blackwell-systems/pipeboard/issues)
[![Sponsor](https://img.shields.io/badge/Sponsor-Buy%20Me%20a%20Coffee-yellow?logo=buy-me-a-coffee&logoColor=white)](https://buymeacoffee.com/blackwellsystems)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

**The programmable clipboard router for terminals.**

One command across macOS, Linux, Windows, and WSL. Sync between machines via SSH or store in S3. Transform clipboard contents with user-defined pipelines. All with client-side encryption and zero telemetry.

## Why pipeboard?

Terminal users copy and paste hundreds of times a day—config snippets, JSON payloads, API keys, log excerpts. Each time you reach for a GUI or email yourself a snippet, you break flow.

| Workflow | Without pipeboard | With pipeboard |
|----------|-------------------|----------------|
| Cross-platform scripts | `pbcopy` on Mac, `xclip` on Linux, `clip.exe` on WSL | `pipeboard copy` everywhere |
| Send clipboard to another machine | SSH, scp, or paste into Slack | `pipeboard send dev` |
| Grab that config you saved yesterday | Scroll through chat history | `pipeboard pull kube` |
| Format JSON before pasting | Copy → browser → formatter → copy | `pipeboard fx pretty-json` |
| Recall what you copied an hour ago | Hope it's still there | `pipeboard recall 1` |

## Core Features

| Feature | Description |
|---------|-------------|
| **Unified clipboard** | One CLI across macOS, Linux, Windows, and WSL |
| **Peer-to-peer sync** | Direct SSH transfer between machines—no cloud required |
| **Named slots** | Persistent storage locally or in S3 for async workflows |
| **Transforms** | User-defined pipelines to process clipboard in-place |
| **History & recall** | Search and restore previous clipboard contents |
| **Watch mode** | Real-time bidirectional sync with a peer |
| **Encryption** | AES-256-GCM client-side encryption for stored content |
| **Zero telemetry** | No tracking, no logging of clipboard contents |

## Quick Start

```bash
# Install
brew install blackwell-systems/homebrew-tap/pipeboard
# or: go install github.com/blackwell-systems/pipeboard@latest

# Copy and paste (works everywhere)
echo "hello" | pipeboard        # implicit copy from stdin
pipeboard paste

# Transform clipboard in-place
pipeboard fx pretty-json        # format JSON
pipeboard fx strip-ansi         # chain multiple transforms

# Sync with another machine (via SSH)
pipeboard send dev              # push to peer
pipeboard recv mac              # pull from peer
pipeboard watch dev             # real-time bidirectional sync

# Store for later (local or S3)
pipeboard push myconfig         # save to named slot
pipeboard pull myconfig         # restore from slot
pipeboard slots                 # list all slots

# Clipboard history
pipeboard history --local       # view recent copies
pipeboard recall 1              # restore previous entry
```

See [Getting Started](docs/getting-started.md) for installation options.

**Want to try first?** Run the [Test Drive](TESTDRIVE.md) - no installation required, just Docker:
```bash
git clone https://github.com/blackwell-systems/pipeboard && cd pipeboard && ./scripts/demo.sh
```

## Platform Support

| Platform | Clipboard | Image | Notes |
|----------|-----------|-------|-------|
| macOS | ✓ built-in | ✓ requires pngpaste/impbcopy | |
| Linux (Wayland) | ✓ wl-clipboard | ✓ native | |
| Linux (X11) | ✓ xclip or xsel | ✓ xclip only | |
| Windows | ✓ clip.exe + PowerShell | ✓ PowerShell | |
| WSL | ✓ clip.exe | paste only | |

Run `pipeboard doctor` to check your setup.

**📚 [Complete Documentation Site](https://blackwell-systems.github.io/pipeboard/)**

- [Test Drive](TESTDRIVE.md) — Try pipeboard without installing (Docker)
- [Getting Started](docs/getting-started.md) — Installation and first commands
- [Commands](docs/commands.md) — Full command reference
- [Transforms](docs/transforms.md) — Build clipboard pipelines
- [Configuration](docs/configuration.md) — Config file reference
- [Sync & Remote](docs/sync.md) — SSH peers and S3 slots
- [Platforms](docs/platforms.md) — Platform-specific details
- [Architecture](docs/architecture.md) — System design and component overview

## Security

pipeboard handles potentially sensitive clipboard data. See [SECURITY.md](SECURITY.md) for details on data handling, transport security, and configuration recommendations.

## Trademarks

pipeboard™ is a product of Blackwell Systems™. See [BRAND.md](BRAND.md) for usage guidelines.

## License

MIT
