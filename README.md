# pipeboard

A tiny cross-platform clipboard CLI.

## Installation

```bash
go install github.com/blackwell-systems/pipeboard@latest
```

Or build from source:

```bash
git clone https://github.com/blackwell-systems/pipeboard.git
cd pipeboard
go build -o pipeboard
```

## Usage

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

# Check detected backend
pipeboard backend

# Diagnose issues
pipeboard doctor
```

**Tip:** Add an alias for convenience:

```bash
alias pb='pipeboard'
```

## Supported Backends

| Platform | Backend | Tools |
|----------|---------|-------|
| macOS | darwin-pasteboard | pbcopy, pbpaste |
| Linux (Wayland) | wayland-wl-copy | wl-copy, wl-paste |
| Linux (X11) | x11-xclip | xclip or xsel |
| WSL | wsl-clip | clip.exe, powershell.exe |

Backend detection is automatic based on environment variables (`WAYLAND_DISPLAY`, `DISPLAY`) and available commands.

## Commands

| Command | Description |
|---------|-------------|
| `copy [text]` | Copy stdin or provided text to clipboard |
| `paste` | Output clipboard contents to stdout |
| `clear` | Clear the clipboard |
| `backend` | Show detected clipboard backend |
| `doctor` | Check dependencies and environment |
| `help` | Show help |
| `version` | Show version |

## Requirements

Install the appropriate clipboard tools for your platform:

- **macOS**: No additional tools needed (pbcopy/pbpaste are built-in)
- **Wayland**: `wl-clipboard` package
- **X11**: `xclip` or `xsel` package
- **WSL**: Windows clipboard tools should be available automatically

## License

MIT
