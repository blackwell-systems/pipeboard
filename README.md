# pipeboard

[![Blackwell Systems™](https://raw.githubusercontent.com/blackwell-systems/blackwell-docs-theme/main/badge-trademark.svg)](https://github.com/blackwell-systems)
[![Platform](https://img.shields.io/badge/Platform-macOS%20%7C%20Linux%20%7C%20Windows%20%7C%20WSL-blue)](https://github.com/blackwell-systems/pipeboard)
[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Go Report Card](https://goreportcard.com/badge/github.com/blackwell-systems/pipeboard)](https://goreportcard.com/report/github.com/blackwell-systems/pipeboard)
[![CI](https://github.com/blackwell-systems/pipeboard/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/blackwell-systems/pipeboard/actions/workflows/ci.yml)

[![Homebrew](https://img.shields.io/badge/Homebrew-blackwell--systems-black?logo=homebrew)](https://github.com/blackwell-systems/homebrew-tap)
[![Go Install](https://img.shields.io/badge/go_install-@latest-00ADD8?logo=go&logoColor=white)](https://pkg.go.dev/github.com/blackwell-systems/pipeboard)
[![Dependabot](https://img.shields.io/badge/Dependabot-enabled-brightgreen?logo=dependabot)](https://github.com/blackwell-systems/pipeboard/security/dependabot)

[![Tests](https://img.shields.io/badge/tests-414-success)](https://github.com/blackwell-systems/pipeboard)
[![codecov](https://codecov.io/gh/blackwell-systems/pipeboard/branch/main/graph/badge.svg)](https://codecov.io/gh/blackwell-systems/pipeboard)
[![Release](https://img.shields.io/github/v/release/blackwell-systems/pipeboard)](https://github.com/blackwell-systems/pipeboard/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](https://github.com/blackwell-systems/pipeboard/issues)

**The programmable clipboard router for terminals.**

One command across macOS, Linux, Windows, and WSL. Sync between machines via SSH. Store snippets in S3. Transform clipboard contents with user-defined pipelines.

## Why pipeboard?

Terminal users copy and paste hundreds of times a day—config snippets, JSON payloads, API keys, log excerpts. Each time you reach for a GUI or email yourself a snippet, you break flow.

| Workflow | Without pipeboard | With pipeboard |
|----------|-------------------|----------------|
| Cross-platform scripts | `pbcopy` on Mac, `xclip` on Linux, `clip.exe` on WSL | `pipeboard copy` everywhere |
| Send clipboard to another machine | SSH, scp, or paste into Slack | `pipeboard send dev` |
| Grab that config you saved yesterday | Scroll through chat history | `pipeboard pull kube` |
| Format JSON before pasting | Copy → browser → formatter → copy | `pipeboard fx pretty-json` |
| Redact secrets before sharing | Manual find-and-replace | `pipeboard fx redact-secrets` |

## Four Pillars

1. **Unified clipboard** — One CLI, every platform (macOS, Linux, Windows, WSL)
2. **SSH peer sync** — Direct machine-to-machine transfer, no cloud required
3. **S3 remote slots** — Named, persistent storage for async workflows
4. **Programmable transforms** — User-defined pipelines to process clipboard in-place

## Quick Start

```bash
# Install
brew install blackwell-systems/homebrew-tap/pipeboard
# or: go install github.com/blackwell-systems/pipeboard@latest

# Copy and paste
echo "hello" | pipeboard
pipeboard paste

# Transform clipboard contents
pipeboard fx pretty-json

# Sync with another machine
pipeboard send dev
pipeboard recv mac

# Store in S3 for later
pipeboard push kube
pipeboard pull kube
```

See [Getting Started](docs/getting-started.md) for all installation methods.

## Platform Support

| Platform | Clipboard | Image | Notes |
|----------|-----------|-------|-------|
| macOS | ✓ built-in | ✓ requires pngpaste/impbcopy | |
| Linux (Wayland) | ✓ wl-clipboard | ✓ native | |
| Linux (X11) | ✓ xclip or xsel | ✓ xclip only | |
| Windows | ✓ clip.exe + PowerShell | ✓ PowerShell | |
| WSL | ✓ clip.exe | paste only | |

Run `pipeboard doctor` to check your setup.

## Documentation

- [Getting Started](docs/getting-started.md) — Installation and first commands
- [Commands](docs/commands.md) — Full command reference
- [Transforms](docs/transforms.md) — Build clipboard pipelines
- [Configuration](docs/configuration.md) — Config file reference
- [Sync & Remote](docs/sync.md) — SSH peers and S3 slots
- [Platforms](docs/platforms.md) — Platform-specific details

## Security

pipeboard handles potentially sensitive clipboard data. See [SECURITY.md](SECURITY.md) for details on data handling, transport security, and configuration recommendations.

## Trademarks

pipeboard™ is a product of Blackwell Systems™. See [BRAND.md](BRAND.md) for usage guidelines.

## License

MIT
