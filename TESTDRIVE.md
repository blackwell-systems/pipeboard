# Test Drive pipeboard

Try pipeboard without installing it on your system. This demo creates two Docker containers connected via SSH, letting you experience all pipeboard features in an isolated environment.

## Quick Start

```bash
git clone https://github.com/blackwell-systems/pipeboard.git
cd pipeboard
./scripts/demo.sh
```

## Requirements

- Docker installed and running
- ~500MB disk space (Alpine Linux + Go toolchain)
- 2-3 minutes for first-time setup

## What You Get

The demo creates two "machines" that can send clipboard data to each other:

```
┌─────────────────┐         ┌─────────────────┐
│   Machine A     │◄──SSH──►│   Machine B     │
│                 │         │                 │
│  pipeboard      │         │  pipeboard      │
│  xclip          │         │  xclip          │
└─────────────────┘         └─────────────────┘
```

Both machines have:
- pipeboard built from source
- SSH configured for passwordless access
- 14 transforms (upper, lower, pretty-json, base64, sha256, etc.)
- Local encrypted slots with example data
- Pre-populated clipboard history
- Duplicate detection enabled

## Demo Walkthrough

### 1. Start the Demo

```bash
./scripts/demo.sh
```

You'll be dropped into Machine A. Open a second terminal for Machine B:

```bash
docker exec -it pipeboard-demo-b bash
```

### 2. Basic Clipboard

On Machine A:
```bash
echo "hello world" | pipeboard    # copy
pipeboard paste                    # paste
```

### 3. Send Between Machines

On Machine A:
```bash
echo "secret message from A" | pipeboard
pipeboard send
```

On Machine B:
```bash
pipeboard paste    # see the message!
```

### 4. Try Transforms

14 transforms are pre-configured:

```bash
pipeboard fx --list              # see all transforms

# Text transforms
echo "hello" | pipeboard
pipeboard fx upper && pipeboard paste    # HELLO
pipeboard fx reverse && pipeboard paste  # OLLEH

# JSON formatting
echo '{"a":1,"b":2}' | pipeboard
pipeboard fx pretty-json && pipeboard paste

# Encoding
echo "secret" | pipeboard
pipeboard fx base64-encode && pipeboard paste    # c2VjcmV0Cg==

# Chain multiple transforms
echo "Hello World" | pipeboard
pipeboard fx lower sha256 && pipeboard paste     # hash of lowercase

# Utility transforms
pipeboard fx word-count      # count words
pipeboard fx line-count      # count lines
pipeboard fx sort-lines      # sort lines
pipeboard fx unique          # remove duplicates
```

### 5. Clipboard History

```bash
echo "first" | pipeboard
echo "second" | pipeboard
echo "third" | pipeboard

pipeboard history --local    # see all entries
pipeboard recall 2           # restore "second"
pipeboard paste              # second
```

### 6. Peek Without Copying

On Machine B, view Machine A's clipboard without modifying your own:
```bash
pipeboard peek    # shows what's on Machine A
```

### 7. Named Slots (Encrypted Storage)

The demo has pre-created encrypted slots:

```bash
pipeboard slots                  # list available slots
pipeboard pull kube-example      # restore to clipboard
pipeboard paste                  # see the content
pipeboard show secrets-example   # view without copying

# Create your own
echo "my important note" | pipeboard
pipeboard push my-notes          # save with encryption
pipeboard slots                  # see it in the list
```

### 8. Bidirectional Sync

On Machine B:
```bash
echo "reply from B" | pipeboard
pipeboard send
```

On Machine A:
```bash
pipeboard paste    # reply from B
```

### 9. Watch Mode (Real-time Sync)

Try real-time bidirectional sync:

On Machine A:
```bash
pipeboard watch    # starts watching
```

On Machine B (in another terminal):
```bash
echo "instant sync!" | pipeboard
```

Machine A's clipboard updates automatically! Press Ctrl+C to stop watching.

## Cleanup

Exit the demo shell and press Ctrl+C, or run:

```bash
./scripts/demo.sh --cleanup
```

This removes the containers and network.

## What's Different in Production?

The demo uses Docker containers to simulate multiple machines. In real use:

| Demo | Production |
|------|------------|
| Docker containers | Real machines or VMs |
| Auto-configured SSH | Your existing SSH config |
| xclip in Xvfb | Your native clipboard (pbcopy, wl-copy, etc.) |
| Pre-built transforms | Your custom transforms |

## Next Steps

Liked what you saw? Install pipeboard for real:

```bash
# macOS
brew install blackwell-systems/homebrew-tap/pipeboard

# Go install
go install github.com/blackwell-systems/pipeboard@latest

# From source
git clone https://github.com/blackwell-systems/pipeboard.git
cd pipeboard && go build && sudo mv pipeboard /usr/local/bin/
```

See the [Getting Started guide](docs/getting-started.md) for full setup instructions.

## Troubleshooting

**Demo hangs on "Installing dependencies"**
- First run downloads ~500MB of packages. Subsequent runs are faster.

**"Docker daemon is not running"**
- Start Docker Desktop or run `sudo systemctl start docker`

**SSH errors**
- The demo auto-configures SSH. If issues persist, try `./scripts/demo.sh --cleanup` and restart.

**Clipboard operations fail**
- The demo uses xclip with Xvfb. Ensure DISPLAY=:99 is set (auto-configured in bash).
