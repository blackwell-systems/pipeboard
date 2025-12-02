# AUR (Arch User Repository) Setup Guide

This guide explains how to submit and maintain the pipeboard package on the AUR (Arch User Repository).

## Table of Contents

1. [One-Time Setup](#one-time-setup)
2. [Updating AUR Package](#updating-aur-package)
3. [User Installation](#user-installation)
4. [Troubleshooting](#troubleshooting)
5. [Best Practices](#best-practices)

## One-Time Setup

### 1. Create AUR Account

1. Go to [AUR website](https://aur.archlinux.org)
2. Click "Register" in top-right
3. Fill in account details:
   - **Username:** (e.g., `blackwell-systems`)
   - **Email:** Your email address
   - **Real Name:** Blackwell Systems
4. Confirm email address

### 2. Set Up SSH Key

AUR uses SSH for authentication. Generate and upload your SSH key:

```bash
# Generate SSH key (if you don't have one)
ssh-keygen -t ed25519 -C "aur@pipeboard" -f ~/.ssh/aur

# Add to SSH config for convenience
cat >> ~/.ssh/config <<EOF

Host aur.archlinux.org
  IdentityFile ~/.ssh/aur
  User aur
EOF

# Copy public key
cat ~/.ssh/aur.pub
```

Upload your public key to AUR:
1. Go to https://aur.archlinux.org/account/
2. Paste your public key in "SSH Public Key" field
3. Click "Update"

### 3. Test SSH Connection

```bash
ssh -T aur@aur.archlinux.org

# Expected output:
# Hi blackwell-systems! You've successfully authenticated, but AUR does not provide shell access.
```

### 4. Install Required Tools

```bash
# On Arch Linux
sudo pacman -S git base-devel openssh

# Verify installation
makepkg --version
git --version
```

### 5. Create AUR Package (First Time)

#### Initial Package Submission

```bash
# Clone empty AUR repository
git clone ssh://aur@aur.archlinux.org/pipeboard.git pipeboard-aur
cd pipeboard-aur

# Copy PKGBUILD and .SRCINFO from your main repo
cp /path/to/pipeboard/aur/PKGBUILD .
cp /path/to/pipeboard/aur/.SRCINFO .

# Add files
git add PKGBUILD .SRCINFO

# Commit and push
git commit -m "Initial commit: pipeboard 0.7.2"
git push origin master
```

**Note:** Package name `pipeboard` must be available. Check at https://aur.archlinux.org/packages/pipeboard

## Updating AUR Package

### Automatic Update (Recommended)

Use the provided script to update PKGBUILD and push to AUR:

```bash
cd /path/to/pipeboard

# Run the update script
./scripts/aur-update.sh
```

The script will:
1. Update `pkgver` in PKGBUILD to match git tag
2. Update checksums with `updpkgsums`
3. Generate `.SRCINFO` file
4. Optionally test build the package
5. Push to AUR repository

### Manual Update (Advanced)

For manual control over the process:

```bash
cd /path/to/pipeboard

# 1. Update version in PKGBUILD
vim aur/PKGBUILD
# Change: pkgver=0.7.2
# Reset: pkgrel=1

# 2. Update checksums
cd aur
updpkgsums PKGBUILD

# 3. Generate .SRCINFO
makepkg --printsrcinfo > .SRCINFO

# 4. Test build (optional but recommended)
makepkg --syncdeps --clean

# 5. Push to AUR
AUR_REPO="$HOME/.cache/pipeboard-aur"
cp PKGBUILD .SRCINFO "$AUR_REPO/"
cd "$AUR_REPO"
git add PKGBUILD .SRCINFO
git commit -m "Update to version 0.7.2"
git push
```

## PKGBUILD Explanation

### Key Components

```bash
# Maintainer information
# Maintainer: Blackwell Systems <blackwellsystems@protonmail.com>

# Package metadata
pkgname=pipeboard              # Package name (must match AUR repo)
pkgver=0.7.2                   # Version from git tag
pkgrel=1                       # Package release (reset to 1 on version bump)
pkgdesc="..."                  # Short description
arch=('x86_64' 'aarch64')      # Supported architectures
url="..."                      # Project homepage
license=('MIT')                # License type

# Dependencies
depends=('glibc')              # Runtime dependencies
optdepends=(...)               # Optional dependencies
makedepends=('go>=1.21')       # Build-time dependencies

# Source and checksums
source=("$pkgname-$pkgver.tar.gz::...")
sha256sums=('SKIP')            # Use 'SKIP' or actual sha256

# Build function
build() {
    cd "$pkgname-$pkgver"
    export CGO_ENABLED=0
    go build -ldflags "-s -w -X main.version=$pkgver" -o pipeboard .
}

# Optional test function
check() {
    cd "$pkgname-$pkgver"
    go test -race ./...
}

# Install function
package() {
    cd "$pkgname-$pkgver"
    install -Dm755 pipeboard "$pkgdir/usr/bin/pipeboard"
    install -Dm644 LICENSE "$pkgdir/usr/share/licenses/$pkgname/LICENSE"
    install -Dm644 man/man1/pipeboard.1 "$pkgdir/usr/share/man/man1/pipeboard.1"
}
```

### .SRCINFO File

The `.SRCINFO` file is auto-generated from PKGBUILD:

```bash
# Generate .SRCINFO
makepkg --printsrcinfo > .SRCINFO
```

**Important:** Always regenerate `.SRCINFO` after editing `PKGBUILD`!

## User Installation

### With AUR Helper (Recommended)

Users can install pipeboard using an AUR helper:

```bash
# Using yay
yay -S pipeboard

# Using paru
paru -S pipeboard

# Using trizen
trizen -S pipeboard
```

### Manual Installation

Without an AUR helper:

```bash
# Clone AUR package
git clone https://aur.archlinux.org/pipeboard.git
cd pipeboard

# Build and install
makepkg -si
```

## Troubleshooting

### Build Failures

**Problem:** "ERROR: One or more files did not pass the validity check!"

**Solution:** Update checksums:
```bash
cd aur
updpkgsums PKGBUILD
makepkg --printsrcinfo > .SRCINFO
```

**Problem:** "==> ERROR: A failure occurred in build()"

**Solution:** Test build locally to debug:
```bash
cd aur
makepkg --syncdeps --clean --log
# Check build.log for errors
```

### Push Failures

**Problem:** "Permission denied (publickey)"

**Solution:** Check SSH key setup:
```bash
# Test connection
ssh -T aur@aur.archlinux.org

# If fails, check SSH config
cat ~/.ssh/config

# Verify key is uploaded to AUR
# Visit: https://aur.archlinux.org/account/
```

**Problem:** "remote: error: hook declined to update refs/heads/master"

**Solution:** .SRCINFO might be out of sync:
```bash
makepkg --printsrcinfo > .SRCINFO
git add .SRCINFO
git commit --amend
git push --force
```

### Version Mismatch

**Problem:** "ERROR: pkgver is not allowed to contain colons, forward slashes, hyphens or whitespace."

**Solution:** Ensure version format is correct:
```bash
# Good: 0.7.2
# Bad:  v0.7.2 (no 'v' prefix)
# Bad:  0.7.2-beta (no hyphens, use 0.7.2.beta)
```

## Best Practices

### 1. Version Numbering

- **pkgver:** Should match upstream version (0.7.2)
- **pkgrel:** Reset to 1 on version bump, increment for package fixes
- **epoch:** Only use if version scheme changes

Examples:
```bash
# New release
pkgver=0.7.2
pkgrel=1

# Fix PKGBUILD issue (no upstream change)
pkgver=0.7.2
pkgrel=2

# Downgrade version (rare, requires epoch)
epoch=1
pkgver=0.7.1
pkgrel=1
```

### 2. Always Test Builds

Before pushing to AUR:

```bash
cd aur
makepkg --syncdeps --clean --rmdeps
sudo pacman -U pipeboard-*.pkg.tar.zst
pipeboard --version
```

### 3. Update Checklist

When releasing a new version:

- [ ] Update `pkgver` in PKGBUILD
- [ ] Reset `pkgrel` to 1
- [ ] Update checksums with `updpkgsums`
- [ ] Regenerate `.SRCINFO`
- [ ] Test build locally
- [ ] Commit and push to AUR
- [ ] Verify package page updates

### 4. Responding to Comments

AUR users may leave comments on your package:
- Check regularly at https://aur.archlinux.org/packages/pipeboard
- Respond to build issues promptly
- Thank users for feedback

### 5. Orphaning/Disowning

If you can no longer maintain the package:

1. Go to https://aur.archlinux.org/packages/pipeboard
2. Click "Disown Package"
3. Someone else can adopt it

## Automation

### GitHub Actions Integration

Automate AUR updates on new releases:

```yaml
# .github/workflows/aur-update.yml
name: Update AUR Package

on:
  release:
    types: [published]

jobs:
  update-aur:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install dependencies
        run: |
          sudo apt-get update
          sudo apt-get install -y pacman-package-manager

      - name: Setup SSH key
        run: |
          mkdir -p ~/.ssh
          echo "${{ secrets.AUR_SSH_KEY }}" > ~/.ssh/aur
          chmod 600 ~/.ssh/aur
          ssh-keyscan aur.archlinux.org >> ~/.ssh/known_hosts

      - name: Update and push to AUR
        run: ./scripts/aur-update.sh
```

Store your AUR SSH private key in GitHub Secrets as `AUR_SSH_KEY`.

### Automatic Version Detection

The update script automatically detects version from git tags:

```bash
# Get version from latest tag
VERSION=$(git describe --tags --abbrev=0 | sed 's/^v//')

# Update PKGBUILD
sed -i "s/^pkgver=.*/pkgver=$VERSION/" PKGBUILD
```

## Resources

- [AUR Official Wiki](https://wiki.archlinux.org/title/AUR)
- [PKGBUILD Guide](https://wiki.archlinux.org/title/PKGBUILD)
- [AUR Submission Guidelines](https://wiki.archlinux.org/title/AUR_submission_guidelines)
- [Arch Package Guidelines](https://wiki.archlinux.org/title/Arch_package_guidelines)
- [makepkg Manual](https://man.archlinux.org/man/makepkg.8)

## Support

For issues with the AUR package:
- **AUR Comments:** https://aur.archlinux.org/packages/pipeboard
- **GitHub Issues:** https://github.com/blackwell-systems/pipeboard/issues
- **Email:** blackwellsystems@protonmail.com

## Common AUR Commands

```bash
# Test build package
makepkg --syncdeps --clean

# Build and install
makepkg -si

# Update checksums
updpkgsums

# Generate .SRCINFO
makepkg --printsrcinfo > .SRCINFO

# Clean build artifacts
makepkg --clean

# Check PKGBUILD for errors
namcap PKGBUILD

# Check built package
namcap pipeboard-*.pkg.tar.zst

# Search AUR
yay -Ss pipeboard

# Get package info
yay -Si pipeboard

# Check package files
pacman -Ql pipeboard
```
