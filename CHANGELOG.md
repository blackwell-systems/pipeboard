# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/blackwell-systems/pipeboard/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/blackwell-systems/pipeboard/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/blackwell-systems/pipeboard/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/blackwell-systems/pipeboard/releases/tag/v0.1.0
