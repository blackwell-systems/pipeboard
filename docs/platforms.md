# Platform Support

pipeboard automatically detects your platform and uses the appropriate clipboard backend.

## Supported Platforms

| Platform | Backend | Text | Image | Notes |
|----------|---------|------|-------|-------|
| macOS | `darwin-pasteboard` | pbcopy/pbpaste | pngpaste/impbcopy | Image tools optional |
| Linux (Wayland) | `wayland-wl-copy` | wl-copy/wl-paste | Native | Needs wl-clipboard |
| Linux (X11) | `x11-xclip` | xclip | xclip | Needs xclip |
| Windows | `windows-clip` | clip.exe/PowerShell | PowerShell | Native |
| WSL | `wsl-clip` | clip.exe/PowerShell | Paste only | Limited image support |

## Check Your Backend

```bash
pipeboard backend
# darwin-pasteboard

pipeboard doctor
# Full environment diagnostics
```

## macOS

**Text clipboard:** Built-in, no dependencies.

```bash
echo "hello" | pipeboard copy
pipeboard paste
```

**Image clipboard:** Requires optional tools.

```bash
# Install image tools
brew install pngpaste
brew install impbcopy   # or: go install github.com/parro-it/impbcopy@latest

# Copy image
cat screenshot.png | pipeboard copy --image

# Paste image
pipeboard paste --image > clipboard.png
```

## Linux (Wayland)

**Text clipboard:** Requires wl-clipboard.

```bash
# Install
sudo apt install wl-clipboard    # Debian/Ubuntu
sudo dnf install wl-clipboard    # Fedora
sudo pacman -S wl-clipboard       # Arch
```

**Image clipboard:** Native support via wl-clipboard.

```bash
cat image.png | pipeboard copy --image
pipeboard paste --image > out.png
```

## Linux (X11)

**Text clipboard:** Requires xclip (preferred) or xsel.

```bash
# Install
sudo apt install xclip    # Debian/Ubuntu
sudo dnf install xclip    # Fedora
sudo pacman -S xclip      # Arch
```

**Image clipboard:** Requires xclip.

```bash
cat image.png | pipeboard copy --image
pipeboard paste --image > out.png
```

## Windows

**Text clipboard:** Built-in via clip.exe and PowerShell.

```bash
echo "hello" | pipeboard copy
pipeboard paste
```

**Image clipboard:** PowerShell-based.

```bash
cat screenshot.png | pipeboard copy --image
pipeboard paste --image > clipboard.png
```

## WSL (Windows Subsystem for Linux)

**Text clipboard:** Uses clip.exe for copy, PowerShell for paste.

```bash
echo "hello" | pipeboard copy
pipeboard paste
```

**Image clipboard:** Limited support (paste only).

```bash
# Paste works
pipeboard paste --image > clipboard.png

# Copy may not work (WSL limitation)
```

## Backend Detection

pipeboard detects your backend automatically:

1. **macOS:** Always uses `darwin-pasteboard`
2. **Windows:** Uses `windows-clip`
3. **WSL:** Detected via `/proc/version`, uses `wsl-clip`
4. **Wayland:** Detected via `WAYLAND_DISPLAY` env var
5. **X11:** Detected via `DISPLAY` env var

Force a specific backend (not recommended):

```bash
PIPEBOARD_BACKEND=x11-xclip pipeboard copy
```

## Troubleshooting

### "backend unknown is missing required tools"

Your platform wasn't detected. Check:

```bash
# Are you in a graphical session?
echo $DISPLAY
echo $WAYLAND_DISPLAY

# On X11, is xclip installed?
which xclip

# On Wayland, is wl-clipboard installed?
which wl-copy
```

### "image copy not supported on backend"

Your backend doesn't support image operations, or required tools are missing.

**macOS:**
```bash
brew install pngpaste impbcopy
```

**X11:**
```bash
sudo apt install xclip   # xsel doesn't support images
```

### WSL clipboard issues

WSL clipboard access can be flaky. Ensure:

1. You're running WSL 2
2. Windows clipboard is accessible
3. Try running `clip.exe` directly to test

### SSH peer not working

1. Verify SSH access: `ssh <peer> echo ok`
2. Verify pipeboard installed remotely: `ssh <peer> pipeboard version`
3. Check peer config in `~/.config/pipeboard/config.yaml`

## Platform-Specific Tips

### macOS

Add to shell profile for convenience:

```bash
alias pb='pipeboard'
alias pbc='pipeboard copy'
alias pbp='pipeboard paste'
```

### Linux

If running headless (no X11/Wayland), clipboard operations won't work. Consider using S3 slots for remote workflows.

### Windows/WSL

PowerShell execution policy may affect clipboard operations. If issues occur:

```powershell
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
```
