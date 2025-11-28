# pipeboard

A tiny cross-platform clipboard CLI with optional remote sync.

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

# Check detected backend
pipeboard backend

# Diagnose issues
pipeboard doctor
```

### Remote Sync

Sync clipboard contents across machines via S3:

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
# On laptop:
cat ~/.kube/config | pipeboard copy
pipeboard push kube

# On server:
pipeboard pull kube
pipeboard paste > ~/.kube/config
```

**Tip:** Add an alias for convenience:

```bash
alias pb='pipeboard'
```

## Configuration

Remote sync requires a config file at `~/.config/pipeboard/config.yaml`:

```yaml
backend: s3

s3:
  bucket: your-bucket-name
  region: us-west-2
  prefix: your-username/slots/   # optional
  profile: pipeboard             # optional AWS profile
  sse: AES256                    # optional: AES256 or aws:kms
```

Environment variable overrides:

```bash
PIPEBOARD_S3_BUCKET=my-bucket pipeboard push test
PIPEBOARD_S3_REGION=us-east-1 pipeboard slots
```

All environment variables:
- `PIPEBOARD_CONFIG` - config file path
- `PIPEBOARD_BACKEND` - backend type (s3)
- `PIPEBOARD_S3_BUCKET` - S3 bucket name
- `PIPEBOARD_S3_REGION` - AWS region
- `PIPEBOARD_S3_PREFIX` - key prefix
- `PIPEBOARD_S3_PROFILE` - AWS profile name
- `PIPEBOARD_S3_SSE` - server-side encryption

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

## Commands

| Command | Description |
|---------|-------------|
| `copy [text]` | Copy stdin or provided text to clipboard |
| `paste` | Output clipboard contents to stdout |
| `clear` | Clear the clipboard |
| `backend` | Show detected clipboard backend |
| `doctor` | Check dependencies and environment |
| `push <name>` | Push clipboard to remote slot |
| `pull <name>` | Pull from remote slot to clipboard |
| `show <name>` | Print remote slot to stdout |
| `slots` | List remote slots |
| `rm <name>` | Delete a remote slot |
| `help` | Show help |
| `version` | Show version |

## Requirements

### Local Clipboard

- **macOS**: No additional tools needed (pbcopy/pbpaste are built-in)
- **Wayland**: `wl-clipboard` package
- **X11**: `xclip` or `xsel` package
- **WSL**: Windows clipboard tools should be available automatically

### Remote Sync

- AWS credentials configured (via environment, profile, or IAM role)
- S3 bucket with appropriate permissions

## License

MIT
