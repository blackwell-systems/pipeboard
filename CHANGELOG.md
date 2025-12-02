# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **SSH peer integration test** - Automated Docker-based test infrastructure in `scripts/test-ssh-peer.sh`
  - Tests bidirectional clipboard sync over SSH using Alpine containers
  - Includes xclip wrapper for headless clipboard access via Xvfb
  - 3 test scenarios: basic transfer, multiline content, bidirectional sync
- **Testing documentation** - Comprehensive guide in `docs/testing.md`
  - Unit test instructions and examples
  - Integration test architecture diagrams
  - Troubleshooting and manual testing guides
- **Scripts directory** - New `scripts/` directory for integration test scripts
- **Clipboard test suite** - New `clipboard_test.go` with comprehensive coverage for clipboard operations
  - Tests for `readClipboard()` and `writeClipboard()` functions
  - Tests for cmdClear fallback path, cmdBackend Notes output
  - Tests for image mode error handling
  - Tests with various data sizes (empty, small, medium, large)

### Changed
- Test coverage increased to 500+ tests (400 test functions, up from 471 test runs)
- Improved watch.go test coverage with tests for `readRemoteClipboard()` and `sendToRemote()`
- Improved clipboard.go test coverage with 14 new tests for core clipboard functions
- Improved history.go test coverage with 29 new tests for error paths and edge cases:
  - Helper functions: `truncateString()`, `isSlotCommand()`, `isPeerCommand()`
  - `getClipboardHistoryLimit()` with custom/zero/negative limits
  - Encryption with preview handling and decryption paths
  - Search on encrypted content
  - JSON marshaling error paths
  - Invalid input validation (negative index, multiple args)
  - Multiple filter combinations
- Added Testing page to documentation navigation

## [0.7.1] - 2025-12-01

### Changed
- Test coverage improved to 71.9% with 471 tests (up from 414)
- New `cli_test.go` with tests for debugLog, printInfo, printError, useColor
- Added tests for peer commands (cmdSend, cmdRecv, cmdPeek error paths)
- Added tests for slots commands (cmdPush, cmdPull, cmdShow, cmdSlots, cmdRm)
- Added tests for parseGlobalFlags and run() with global flags
- cmdShow, cmdSlots, cmdRm now at 100%/93.5%/100% coverage

## [0.7.0] - 2025-11-30

### Added
- **Global flags** - `--quiet` (`-q`) suppresses output, `--debug` enables debug logging
- **Configurable history limit** - `history.limit` in config (default: 20)
- **Debian/Ubuntu packages** - `.deb` packages in GitHub releases
- **RPM packages** - `.rpm` packages for Fedora/RHEL in GitHub releases
- **AUR package** - `yay -S pipeboard` for Arch Linux users
- **Cosign release signing** - All releases cryptographically signed with keyless Cosign
- **CodeQL security scanning** - Continuous security analysis in CI
- **Dependabot** - Automated dependency updates
- **Dynamic version** - Version now set at build time via ldflags
- **Benchmark tests** - 20 performance benchmarks for encryption, compression, etc.
- **ARCHITECTURE.md** - Developer documentation for codebase structure

### Changed
- Improved community infrastructure (issue templates, PR template, CODE_OF_CONDUCT.md)

## [0.6.0] - 2025-11-29

### Added
- **History search** - `pipeboard history --local --search <query>` to filter clipboard history by content
- **Encrypted clipboard history** - When encryption is enabled, clipboard history is automatically encrypted at rest using the sync passphrase
- **Automatic compression** - Data larger than 1KB is automatically gzip-compressed when stored in slots (both S3 and local)
- **MIME type detection** - Content type is automatically detected and stored in slot metadata
- **Retry with exponential backoff** - S3 operations automatically retry on transient network errors with jitter
- **Watch mode** - `pipeboard watch [peer]` for real-time bidirectional clipboard sync with a peer
- **Local clipboard history** - `pipeboard history --local` shows clipboard content snapshots
- **Recall command** - `pipeboard recall <index>` restores previous clipboard entries
- **JSON output** - `--json` flag for `history`, `slots`, and `doctor` commands
- **Man pages** - `man pipeboard` now available (installed with Homebrew)
- **Shell completions** - `pipeboard completion bash|zsh|fish` generates shell completion scripts for tab completion
- **Interactive setup wizard** - `pipeboard init` guides users through initial configuration
- **Homebrew formula** - `brew install blackwell-systems/homebrew-tap/pipeboard` now available
- **Docsify documentation site** - Dark-themed docs with cover page, matching Blackwell Systems style
- **Codecov integration** - Code coverage tracking in CI
- **Local filesystem backend** - Zero-config slot storage at `~/.config/pipeboard/slots/`
- **Per-command help** - `pipeboard <command> --help` shows detailed help for each command
- **Colored error messages** - Errors now display in red (respects NO_COLOR and TERM=dumb)
- **Implicit copy** - `echo "hello" | pipeboard` now defaults to copy when stdin has data
- **CONTRIBUTING.md** - Contributor guidelines for open source participation

### Changed
- Test coverage improved to 69.9% with 414 tests (init.go, watch.go, history, recall, encryption)
- Help text now documents local backend alongside S3

### Fixed
- Watch mode echo prevention logic now correctly preserves the prevention window
- TTL parsing now validates input and uses defaults for invalid values
- Backend detection is now cached for performance (avoids repeated detection calls)
- S3 slot listing now uses pagination to handle more than 1000 slots

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

[Unreleased]: https://github.com/blackwell-systems/pipeboard/compare/v0.7.1...HEAD
[0.7.1]: https://github.com/blackwell-systems/pipeboard/compare/v0.7.0...v0.7.1
[0.7.0]: https://github.com/blackwell-systems/pipeboard/compare/v0.6.0...v0.7.0
[0.6.0]: https://github.com/blackwell-systems/pipeboard/compare/v0.5.1...v0.6.0
[0.5.1]: https://github.com/blackwell-systems/pipeboard/compare/v0.5.0...v0.5.1
[0.5.0]: https://github.com/blackwell-systems/pipeboard/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/blackwell-systems/pipeboard/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/blackwell-systems/pipeboard/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/blackwell-systems/pipeboard/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/blackwell-systems/pipeboard/releases/tag/v0.1.0
