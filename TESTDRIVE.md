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
- Example transforms (upper, lower, trim, reverse)
- Clipboard history enabled

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

On either machine:
```bash
echo "hello" | pipeboard
pipeboard fx upper
pipeboard paste    # HELLO

pipeboard fx reverse
pipeboard paste    # OLLEH

pipeboard fx --list    # see all transforms
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

### 7. Bidirectional Sync

On Machine B:
```bash
echo "reply from B" | pipeboard
pipeboard send
```

On Machine A:
```bash
pipeboard paste    # reply from B
```

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
