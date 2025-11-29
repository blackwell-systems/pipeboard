# Commands

## Local Clipboard

### copy

Copy content to the clipboard.

```bash
# Implicit copy (piped input defaults to copy)
echo "hello" | pipeboard

# Explicit copy from stdin
echo "hello" | pipeboard copy

# Copy text argument
pipeboard copy "hello world"

# Copy PNG image from stdin
cat screenshot.png | pipeboard copy --image
```

**Flags:**
- `--image`, `-i` — Copy PNG data from stdin

### paste

Output clipboard contents to stdout.

```bash
# Paste to stdout
pipeboard paste

# Pipe to other tools
pipeboard paste | jq .

# Save to file
pipeboard paste > output.txt

# Paste image as PNG
pipeboard paste --image > clipboard.png
```

**Flags:**
- `--image`, `-i` — Output clipboard image as PNG

### clear

Clear the clipboard contents.

```bash
pipeboard clear
```

### backend

Show the detected clipboard backend.

```bash
pipeboard backend
# darwin-pasteboard
```

### doctor

Run environment diagnostics.

```bash
pipeboard doctor
```

Reports:
- Detected backend
- Available clipboard tools
- Missing dependencies
- Image support status

## Transforms (fx)

### fx

Run transforms on clipboard contents. See [Transforms](transforms.md) for details.

```bash
# Run single transform
pipeboard fx pretty-json

# Chain multiple transforms
pipeboard fx strip-ansi redact-secrets pretty-json

# Preview without modifying clipboard
pipeboard fx pretty-json --dry-run

# List available transforms
pipeboard fx --list
```

**Flags:**
- `--dry-run` — Print result to stdout, don't modify clipboard
- `--list` — List available transforms from config

## SSH Peer Sync

### send

Send local clipboard to a peer's clipboard via SSH.

```bash
# Send to default peer
pipeboard send

# Send to specific peer
pipeboard send dev
```

### recv

Receive a peer's clipboard into local clipboard.

```bash
# Receive from default peer
pipeboard recv

# Receive from specific peer
pipeboard recv mac
```

### peek

View a peer's clipboard without modifying local clipboard.

```bash
# Peek at default peer
pipeboard peek

# Peek at specific peer
pipeboard peek dev
```

## S3 Remote Slots

### push

Push clipboard to a named remote slot.

```bash
pipeboard push myslot
pipeboard push kube-config
```

### pull

Pull from a remote slot into clipboard.

```bash
pipeboard pull myslot
pipeboard pull kube-config
```

### show

View slot contents without modifying clipboard.

```bash
pipeboard show myslot
```

### slots

List all remote slots.

```bash
pipeboard slots
```

Output includes:
- Slot name
- Size
- Age
- TTL/expiry status

### rm

Delete a remote slot.

```bash
pipeboard rm myslot
```

## History

### history

Show recent pipeboard operations.

```bash
# Show all history (most recent first)
pipeboard history

# Filter to fx transforms only
pipeboard history --fx

# Filter to slot operations (push/pull/show/rm)
pipeboard history --slots

# Filter to peer operations (send/recv/peek)
pipeboard history --peer
```

**Flags:**
- `--fx` — Show only transform operations
- `--slots` — Show only slot operations (push/pull/show/rm)
- `--peer` — Show only peer operations (send/recv/peek)

## Setup

### init

Interactive configuration wizard to set up pipeboard.

```bash
pipeboard init
```

This guides you through:
- Choosing a sync backend (local, S3, or none)
- Configuring SSH peers for clipboard sharing
- Adding example transforms (pretty-json, sort-lines, etc.)

Creates `~/.config/pipeboard/config.yaml` with your choices.

### completion

Generate shell completion scripts for tab completion.

```bash
# Bash (add to ~/.bashrc)
source <(pipeboard completion bash)

# Zsh (add to ~/.zshrc)
source <(pipeboard completion zsh)

# Fish
pipeboard completion fish > ~/.config/fish/completions/pipeboard.fish
```

**Supported shells:** bash, zsh, fish

## Other

### Per-Command Help

Get detailed help for any command:

```bash
pipeboard copy --help
pipeboard push --help
pipeboard fx --help
```

### help

Show help message.

```bash
pipeboard help
pipeboard -h
pipeboard --help
```

### version

Show version.

```bash
pipeboard version
pipeboard -v
pipeboard --version
```
