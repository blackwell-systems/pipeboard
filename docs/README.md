# pipeboard

**The programmable clipboard router for terminals.**

One command across macOS, Linux, and WSL. Sync between machines via SSH. Store snippets in S3. Transform clipboard contents with user-defined pipelines.

## Why pipeboard?

Terminal users copy and paste hundreds of times a day—config snippets, JSON payloads, API keys, log excerpts. Each time you reach for a GUI or email yourself a snippet, you break flow.

pipeboard keeps everything in the terminal:

| Workflow | Without pipeboard | With pipeboard |
|----------|-------------------|----------------|
| Cross-platform scripts | `pbcopy` on Mac, `xclip` on Linux, `clip.exe` on WSL | `pipeboard copy` everywhere |
| Send clipboard to another machine | SSH, scp, or paste into Slack | `pipeboard send dev` |
| Grab that config you saved yesterday | Scroll through chat history | `pipeboard pull kube` |
| Format JSON before pasting | Copy → browser → formatter → copy | `pipeboard fx pretty-json` |
| Redact secrets before sharing | Manual find-and-replace | `pipeboard fx redact-secrets` |

## Four Pillars

### 1. Unified Clipboard

One CLI that works everywhere. No more remembering `pbcopy` vs `xclip` vs `xsel` vs `wl-copy` vs `clip.exe`.

```bash
echo "hello" | pipeboard copy
pipeboard paste
```

### 2. SSH Peer Sync

Send clipboard directly between machines over SSH—no cloud, no intermediary.

```bash
pipeboard send dev    # push to dev machine
pipeboard recv mac    # pull from mac
```

### 3. S3 Remote Slots

Named, persistent clipboard storage for async workflows. Client-side AES-256 encryption.

```bash
pipeboard push kube   # save for later
pipeboard pull kube   # on another machine, hours later
```

### 4. Programmable Transforms

User-defined pipelines to process clipboard in-place. Chain multiple transforms.

```bash
pipeboard fx pretty-json
pipeboard fx strip-ansi redact-secrets pretty-json
```

## Quick Install

<!-- tabs:start -->

#### **Homebrew**

```bash
brew install blackwell-systems/tap/pipeboard
```

#### **Go**

```bash
go install github.com/blackwell-systems/pipeboard@latest
```

#### **Binary**

```bash
# macOS (Apple Silicon)
curl -LO https://github.com/blackwell-systems/pipeboard/releases/latest/download/pipeboard_darwin_arm64.tar.gz
tar xzf pipeboard_darwin_arm64.tar.gz
sudo mv pipeboard /usr/local/bin/

# Linux (x86_64)
curl -LO https://github.com/blackwell-systems/pipeboard/releases/latest/download/pipeboard_linux_amd64.tar.gz
tar xzf pipeboard_linux_amd64.tar.gz
sudo mv pipeboard /usr/local/bin/
```

<!-- tabs:end -->

## Next Steps

- [Getting Started](getting-started.md) — Installation and first commands
- [Commands](commands.md) — Full command reference
- [Transforms](transforms.md) — Build your own clipboard pipelines
- [Configuration](configuration.md) — Config file reference
