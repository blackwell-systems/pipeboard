# Getting Started

## Installation

<!-- tabs:start -->

#### **Homebrew (macOS/Linux)**

```bash
brew install blackwell-systems/tap/pipeboard
```

#### **Go**

```bash
go install github.com/blackwell-systems/pipeboard@latest
```

#### **Download Binary**

**macOS (Apple Silicon)**
```bash
curl -LO https://github.com/blackwell-systems/pipeboard/releases/latest/download/pipeboard_darwin_arm64.tar.gz
tar xzf pipeboard_darwin_arm64.tar.gz
sudo mv pipeboard /usr/local/bin/
```

**macOS (Intel)**
```bash
curl -LO https://github.com/blackwell-systems/pipeboard/releases/latest/download/pipeboard_darwin_amd64.tar.gz
tar xzf pipeboard_darwin_amd64.tar.gz
sudo mv pipeboard /usr/local/bin/
```

**Linux (x86_64)**
```bash
curl -LO https://github.com/blackwell-systems/pipeboard/releases/latest/download/pipeboard_linux_amd64.tar.gz
tar xzf pipeboard_linux_amd64.tar.gz
sudo mv pipeboard /usr/local/bin/
```

**Linux (ARM64)**
```bash
curl -LO https://github.com/blackwell-systems/pipeboard/releases/latest/download/pipeboard_linux_arm64.tar.gz
tar xzf pipeboard_linux_arm64.tar.gz
sudo mv pipeboard /usr/local/bin/
```

#### **From Source**

```bash
git clone https://github.com/blackwell-systems/pipeboard.git
cd pipeboard
go build
sudo mv pipeboard /usr/local/bin/
```

<!-- tabs:end -->

## Uninstall

<!-- tabs:start -->

#### **Quick Uninstall**

```bash
curl -sSL https://raw.githubusercontent.com/blackwell-systems/pipeboard/main/uninstall.sh | sh
```

#### **Homebrew**

```bash
brew uninstall pipeboard
```

#### **Manual**

```bash
# Remove binary
sudo rm /usr/local/bin/pipeboard

# Remove config (optional)
rm -rf ~/.config/pipeboard
```

<!-- tabs:end -->

## Verify Installation

```bash
pipeboard version
# pipeboard v0.5.1

pipeboard doctor
# Checks your environment and reports any issues
```

## Basic Usage

### Copy and Paste

```bash
# Implicit copy (piped input defaults to copy)
echo "hello world" | pipeboard

# Explicit copy from stdin
echo "hello world" | pipeboard copy

# Copy text directly
pipeboard copy "hello world"

# Paste to stdout
pipeboard paste

# Pipe to other tools
pipeboard paste | jq .

# Clear clipboard
pipeboard clear
```

### Check Your Backend

pipeboard automatically detects your clipboard backend:

```bash
pipeboard backend
# darwin-pasteboard (on macOS)
# wayland-wl-copy (on Wayland)
# x11-xclip (on X11)
# windows-clip (on Windows)
```

## Create a Config File

For advanced features (transforms, sync, peers), create a config file:

```bash
mkdir -p ~/.config/pipeboard
```

```yaml
# ~/.config/pipeboard/config.yaml
version: 1

# Default peer for send/recv/peek
defaults:
  peer: dev

# SSH peers for direct sync
peers:
  dev:
    ssh: devbox

# Clipboard transforms
fx:
  pretty-json:
    cmd: ["jq", "."]
    description: "Format JSON"

# S3 remote storage (optional)
sync:
  backend: s3
  s3:
    bucket: my-pipeboard
    region: us-west-2
```

## Next Steps

- [Commands](commands.md) — Full command reference
- [Transforms](transforms.md) — Build clipboard pipelines
- [Sync & Remote](sync.md) — SSH peers and S3 slots
- [Configuration](configuration.md) — All config options
