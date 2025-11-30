# Pipeboard Architecture

This document describes the internal architecture of pipeboard for developers who want to contribute or understand how the codebase works.

## Overview

Pipeboard is a single-binary CLI tool written in Go. It has no external runtime dependencies—all clipboard operations use platform-native tools via subprocess calls.

```
┌─────────────────────────────────────────────────────────────────┐
│                          CLI Layer                               │
│  main.go → cli.go → command handlers                            │
└─────────────────────────────────────────────────────────────────┘
                                │
        ┌───────────────────────┼───────────────────────┐
        ▼                       ▼                       ▼
┌───────────────┐      ┌───────────────┐      ┌───────────────┐
│   Clipboard   │      │    Storage    │      │     Peer      │
│   Operations  │      │   Backends    │      │     Sync      │
│  clipboard.go │      │ local.go      │      │   peer.go     │
│  backend.go   │      │ remote.go     │      │   watch.go    │
└───────────────┘      └───────────────┘      └───────────────┘
        │                       │                       │
        ▼                       ▼                       ▼
┌─────────────────────────────────────────────────────────────────┐
│                       Support Layer                              │
│  config.go │ crypto.go │ history.go │ fx.go │ completion.go    │
└─────────────────────────────────────────────────────────────────┘
```

## File Structure

| File | Purpose | Key Functions |
|------|---------|---------------|
| `main.go` | Entry point, command routing | `run()`, `main()` |
| `cli.go` | Help text, colors, stdin detection | `printHelp()`, `useColor()` |
| `clipboard.go` | Copy/paste/clear commands | `cmdCopy()`, `cmdPaste()`, `readClipboard()`, `writeClipboard()` |
| `backend.go` | Platform clipboard detection | `detectBackend()`, `detectDarwin()`, `detectX11()`, `detectWayland()` |
| `local.go` | Local filesystem backend | `LocalBackend.Push()`, `LocalBackend.Pull()` |
| `remote.go` | S3 backend with compression | `S3Backend.Push()`, `S3Backend.Pull()`, `compressData()` |
| `peer.go` | SSH peer sync | `cmdSend()`, `cmdRecv()`, `cmdPeek()` |
| `watch.go` | Real-time bidirectional sync | `cmdWatch()`, `watchLoop()` |
| `slots.go` | Remote slot management | `cmdPush()`, `cmdPull()`, `cmdSlots()` |
| `fx.go` | Transform pipelines | `cmdFx()`, `runTransform()` |
| `config.go` | YAML config + env vars | `loadConfig()`, `validateSyncConfig()` |
| `crypto.go` | AES-256-GCM encryption | `encrypt()`, `decrypt()`, `deriveKey()` |
| `history.go` | Command & clipboard history | `recordHistory()`, `recordClipboardHistory()`, `cmdRecall()` |
| `init.go` | Interactive setup wizard | `cmdInit()`, `generateConfigYAML()` |
| `completion.go` | Shell completions | `cmdCompletion()` |
| `run.go` | Subprocess execution | `runCommand()`, `runWithInput()` |

## Core Abstractions

### RemoteBackend Interface

Both S3 and local filesystem implement this interface:

```go
type RemoteBackend interface {
    Push(slot string, data []byte, meta map[string]string) error
    Pull(slot string) ([]byte, map[string]string, error)
    List() ([]RemoteSlot, error)
    Delete(slot string) error
}
```

This allows seamless switching between storage backends via configuration.

### Backend (Clipboard)

The `Backend` struct represents a platform clipboard tool:

```go
type Backend struct {
    Kind     BackendKind  // Darwin, X11, Wayland, WSL, Windows
    CopyCmd  []string     // e.g., ["pbcopy"] or ["xclip", "-selection", "clipboard"]
    PasteCmd []string     // e.g., ["pbpaste"]
}
```

Detection order: Darwin → Wayland → X11 → WSL → Windows

## Data Flow

### Copy Operation
```
stdin/args → cmdCopy() → writeClipboard() → backend.CopyCmd subprocess
                      → recordClipboardHistory() → ~/.config/pipeboard/clipboard_history.json
```

### Push to Remote
```
clipboard → cmdPush() → compressData() (if >1KB)
                     → encrypt() (if enabled)
                     → S3Backend.Push() / LocalBackend.Push()
                     → recordHistory()
```

### Peer Send (SSH)
```
clipboard → cmdSend() → SSH subprocess
                     → remote pipeboard copy
                     → recordHistory()
```

### Watch Mode
```
watchLoop() ─┬─► poll local clipboard (SHA256 hash)
             │   └─► if changed: sendToRemote()
             │
             └─► poll remote clipboard
                 └─► if changed: writeClipboard()
```

## Configuration

Config is loaded from `~/.config/pipeboard/config.yaml` with environment variable overrides:

```
loadConfig() → applyDefaults()
            → applyEnvOverrides()
            → applyLegacyConfig()
            → validateSyncConfig()
```

Priority: Environment variables > Config file > Defaults

Key environment variables:
- `PIPEBOARD_BACKEND` - sync backend (s3, local)
- `PIPEBOARD_S3_*` - S3 configuration
- `PIPEBOARD_PASSPHRASE` - encryption key

## Encryption

Client-side encryption uses:
- **Algorithm**: AES-256-GCM (authenticated encryption)
- **Key derivation**: PBKDF2 with 100,000 iterations
- **Salt**: 16 random bytes (stored with ciphertext)
- **Nonce**: 12 random bytes (stored with ciphertext)

Format: `salt (16 bytes) || nonce (12 bytes) || ciphertext`

## History

Two types of history:
1. **Command history** (`~/.config/pipeboard/history.json`) - tracks push/pull/send/recv operations
2. **Clipboard history** (`~/.config/pipeboard/clipboard_history.json`) - snapshots of clipboard content

Both are encrypted when `sync.encryption` is enabled.

## Error Handling

- All command functions return `error`
- Errors are printed to stderr via `printError()`
- Exit code 0 for success, 1 for any error
- Colors respect `NO_COLOR` env var and `TERM=dumb`

## Testing

Tests are colocated with source files (`*_test.go`). Key patterns:
- Table-driven tests for configuration validation
- Temp directories for file system tests
- Mock backends for S3 testing
- Integration tests that spawn actual clipboard tools

Run tests: `go test -v -race ./...`

## Build

Version is injected at build time via ldflags:

```bash
go build -ldflags "-X main.version=1.2.3" .
```

GoReleaser handles this automatically for releases.

## Adding a New Command

1. Add handler function in appropriate file (e.g., `clipboard.go`)
2. Register in `commands` map in `main.go`
3. Add help text in `commandHelp` map in `cli.go`
4. Add shell completion in `completion.go`
5. Add tests in `*_test.go`
6. Update man page in `man/man1/pipeboard.1`

## Adding a New Backend

1. Implement `RemoteBackend` interface
2. Add configuration struct in `config.go`
3. Add factory function (e.g., `newMyBackend()`)
4. Update `newRemoteBackendFromConfig()` in `remote.go`
5. Add validation in `validateSyncConfig()`
