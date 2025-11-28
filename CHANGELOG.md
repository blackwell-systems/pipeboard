# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Planned
- **Default peer** - `pipeboard send` without argument uses `defaults.peer` from config
- **Encryption at rest** - Client-side encryption for S3 slot contents
- **Slot expiry** - Auto-delete slots after configurable TTL
- **History command** - `pipeboard history` to show recent local operations

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

[Unreleased]: https://github.com/blackwell-systems/pipeboard/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/blackwell-systems/pipeboard/releases/tag/v0.1.0
