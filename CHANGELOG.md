# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **Docsify documentation site** - Dark-themed docs with cover page, matching Blackwell Systems style
- **Codecov integration** - Code coverage tracking in CI
- **Local filesystem backend** - Zero-config slot storage at `~/.config/pipeboard/slots/`
- **Per-command help** - `pipeboard <command> --help` shows detailed help for each command
- **Colored error messages** - Errors now display in red (respects NO_COLOR and TERM=dumb)
- **Implicit copy** - `echo "hello" | pipeboard` now defaults to copy when stdin has data
- **CONTRIBUTING.md** - Contributor guidelines for open source participation

### Changed
- Test coverage improved from 43% to 66%
- Help text now documents local backend alongside S3

## [0.5.1] - 2025-11-29

### Fixed
- **WSL2 backend detection** - Now correctly falls through to WSL backend when Wayland tools are missing (WSLg sets WAYLAND_DISPLAY but may not have wl-clipboard)
- **Actionable error hints** - Missing tool errors now include platform-specific installation instructions
- **install.sh** - Fixed filename mismatch with GoReleaser output format

## [0.5.0] - 2025-11-28

### Added
- **Chained transforms** - `pipeboard fx name1 name2 name3` runs multiple transforms in sequence
  - Output from each transform feeds into the next
  - If any step fails, clipboard is unchanged (safety guarantee)
- **History filters** - `pipeboard history --fx`, `--slots`, `--peer` to filter operations
- **Native Windows support** - `BackendWindows` for direct Windows clipboard (not just WSL)
  - Full image copy/paste support via PowerShell

### Changed
- History now shows most recent operations first
- Improved help text with chaining examples and safety guarantees

## [0.4.0] - 2025-01-28

### Added
- **Programmable clipboard transforms** - `pipeboard fx <name>` runs user-defined transforms on clipboard contents
  - Configure transforms in `fx` section of config with `cmd` (array) or `shell` (string) syntax
  - `fx --list` shows available transforms
  - `fx <name> --dry-run` previews output without modifying clipboard
  - Example transforms: `pretty-json` (jq), `strip-ansi`, `redact-secrets`, `yaml-to-json`

## [0.3.0] - 2025-01-28

### Added
- **Image clipboard support** - `copy --image` and `paste --image` flags for PNG image clipboard operations
  - macOS: Requires `pngpaste` and `impbcopy` (install via Homebrew)
  - Linux Wayland: Native support via `wl-copy`/`wl-paste`
  - Linux X11: Native support via `xclip` (xsel does not support images)
  - WSL/Windows: Paste only via PowerShell

## [0.2.0] - 2025-01-28

### Added
- **Default peer** - `pipeboard send`, `recv`, and `peek` without argument uses `defaults.peer` from config
- **Client-side encryption** - AES-256-GCM encryption for S3 slot contents with PBKDF2 key derivation
- **Slot expiry/TTL** - Auto-delete slots after configurable TTL via `sync.ttl_days` config
- **History command** - `pipeboard history` to show recent operations (push, pull, send, recv, peek)

### Security
- Client-side encryption uses secure cryptographic primitives (AES-256-GCM, PBKDF2 with 100k iterations)
- Encrypted payloads stored in S3 cannot be decrypted without the passphrase

## [0.1.0] - 2025-01-28

### Added
- **Local clipboard commands**: `copy`, `paste`, `clear`, `backend`, `doctor`
- **Cross-platform backend detection**: macOS (pbcopy), Linux Wayland (wl-copy), Linux X11 (xclip/xsel), WSL (clip.exe)
- **SSH peer-to-peer sync**: `send`, `recv`, `peek` commands for direct clipboard transfer between machines
- **S3 remote slots**: `push`, `pull`, `show`, `slots`, `rm` commands for persistent clipboard storage
- **YAML configuration**: `~/.config/pipeboard/config.yaml` with peers and sync settings
- **Environment variable overrides**: `PIPEBOARD_CONFIG`, `PIPEBOARD_S3_*` variables
- **GitHub Actions CI/CD**: Automated testing and cross-platform releases via GoReleaser
- **Documentation**: README, SECURITY.md, BRAND.md

### Security
- No logging of clipboard contents
- No telemetry or data collection
- SSH transport for peer sync
- Optional S3 server-side encryption (AES256/KMS)

[Unreleased]: https://github.com/blackwell-systems/pipeboard/compare/v0.5.0...HEAD
[0.5.0]: https://github.com/blackwell-systems/pipeboard/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/blackwell-systems/pipeboard/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/blackwell-systems/pipeboard/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/blackwell-systems/pipeboard/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/blackwell-systems/pipeboard/releases/tag/v0.1.0
