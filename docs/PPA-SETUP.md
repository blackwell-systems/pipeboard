# Ubuntu PPA Setup Guide

This guide explains how to set up and maintain the pipeboard PPA (Personal Package Archive) for Ubuntu users.

## Table of Contents

1. [One-Time Setup](#one-time-setup)
2. [Uploading New Releases](#uploading-new-releases)
3. [User Installation](#user-installation)
4. [Troubleshooting](#troubleshooting)

## One-Time Setup

### 1. Create Launchpad Account

1. Go to [Launchpad.net](https://launchpad.net) and create an account
2. Complete your profile information
3. Note your Launchpad username (e.g., `blackwell-systems`)

### 2. Generate GPG Key

If you don't already have a GPG key:

```bash
# Generate new GPG key
gpg --full-generate-key

# Choose:
# - RSA and RSA (default)
# - 4096 bits
# - No expiration (or set your preference)
# - Your name: Blackwell Systems
# - Your email: blackwellsystems@protonmail.com
```

List your keys and note the key ID:

```bash
gpg --list-secret-keys --keyid-format=long

# Output will show something like:
# sec   rsa4096/ABCD1234EFGH5678 2025-12-02 [SC]
#                               ^^^^^^^^^^^^^^^^ This is your key ID
```

### 3. Upload GPG Key to Launchpad

Export your public key:

```bash
# Replace ABCD1234EFGH5678 with your actual key ID
gpg --armor --export ABCD1234EFGH5678 > my-key.asc
```

Upload to Launchpad:

1. Go to https://launchpad.net/~YOUR_USERNAME/+editpgpkeys
2. Paste the contents of `my-key.asc`
3. Click "Import Key"
4. Launchpad will send you an encrypted email
5. Decrypt the email and click the confirmation link

```bash
# To decrypt the email body:
gpg --decrypt
# Paste the encrypted message, press Ctrl+D
```

### 4. Upload GPG Key to Keyserver

```bash
# Upload to Ubuntu keyserver
gpg --keyserver keyserver.ubuntu.com --send-keys ABCD1234EFGH5678
```

### 5. Create PPA on Launchpad

1. Go to https://launchpad.net/~YOUR_USERNAME
2. Click "Create a new PPA"
3. Fill in details:
   - **URL:** `pipeboard`
   - **Display name:** `Pipeboard`
   - **Description:** Cross-platform clipboard CLI with SSH + S3 sync
4. Click "Activate"

Your PPA URL will be: `ppa:blackwell-systems/pipeboard`

### 6. Install Required Tools

On Ubuntu/Debian:

```bash
# Install packaging tools
sudo apt update
sudo apt install -y \
    devscripts \
    debhelper \
    dh-golang \
    dput \
    gnupg \
    git \
    golang-go

# Verify installation
debuild --version
dput --version
```

### 7. Configure dput

Create or edit `~/.dput.cf`:

```ini
[ppa:blackwell-systems/pipeboard]
fqdn = ppa.launchpad.net
method = ftp
incoming = ~blackwell-systems/ubuntu/pipeboard/
login = anonymous
allow_unsigned_uploads = 0
```

## Uploading New Releases

### Automatic Upload (Recommended)

Use the provided script to build and upload for all Ubuntu releases:

```bash
# Set your GPG key ID (optional, uses default if not set)
export GPG_KEY=ABCD1234EFGH5678

# Run the upload script
cd /path/to/pipeboard
./scripts/ppa-upload.sh
```

The script will:
1. Build source packages for noble, mantic, jammy, and focal
2. Upload each to your PPA
3. Clean up build artifacts

### Manual Upload (Advanced)

For a single Ubuntu release:

```bash
# Update changelog
DEBEMAIL="blackwellsystems@protonmail.com" \
DEBFULLNAME="Blackwell Systems" \
dch --newversion 0.7.2-1ubuntu1~noble1 \
    --distribution noble \
    "New upstream release"

# Build source package
debuild -S -sa -kABCD1234EFGH5678

# Upload to PPA
dput ppa:blackwell-systems/pipeboard ../pipeboard_0.7.2-1ubuntu1~noble1_source.changes
```

### Monitor Build Status

After uploading, monitor the build progress:

1. Go to https://launchpad.net/~blackwell-systems/+archive/ubuntu/pipeboard
2. Click on "View package details"
3. Check build status for each architecture (amd64, arm64, etc.)

Builds typically take 5-30 minutes depending on queue length.

## User Installation

### For Ubuntu Users

Users can install pipeboard from your PPA with:

```bash
# Add PPA
sudo add-apt-repository ppa:blackwell-systems/pipeboard

# Update and install
sudo apt update
sudo apt install pipeboard

# Verify installation
pipeboard version
```

### Supported Ubuntu Releases

- **Ubuntu 24.04 LTS (Noble Numbat)** - Recommended
- **Ubuntu 23.10 (Mantic Minotaur)**
- **Ubuntu 22.04 LTS (Jammy Jellyfish)**
- **Ubuntu 20.04 LTS (Focal Fossa)**

## Troubleshooting

### Build Failures

**Problem:** Build fails with "Unmet build dependencies"

**Solution:** Update `debian/control` with correct dependencies for that Ubuntu version.

**Problem:** Build fails with "Original tarball not found"

**Solution:** Use `-sa` flag with debuild to include original source:
```bash
debuild -S -sa
```

### Upload Failures

**Problem:** "Signature verification failed"

**Solution:** Ensure your GPG key is:
1. Uploaded to Launchpad
2. Confirmed via email
3. Uploaded to Ubuntu keyserver

```bash
# Re-upload to keyserver
gpg --keyserver keyserver.ubuntu.com --send-keys ABCD1234EFGH5678
```

**Problem:** "Could not find person or team"

**Solution:** Check your dput configuration matches your Launchpad username exactly.

### Version Conflicts

**Problem:** "File already exists in PPA"

**Solution:** Increment the Ubuntu revision number:
```bash
# Instead of: 0.7.2-1ubuntu1~noble1
# Use:        0.7.2-2ubuntu1~noble1
#                    ^ increment this
```

### Testing Before Upload

Test package build locally without uploading:

```bash
# Build locally (doesn't sign or upload)
debuild -us -uc

# Install locally to test
sudo dpkg -i ../pipeboard_*.deb

# Test functionality
pipeboard --help
pipeboard copy "test"
pipeboard paste
```

## Best Practices

1. **Version numbering:**
   - Upstream version: `0.7.2` (from git tag)
   - Debian revision: `-1ubuntu1`
   - Release suffix: `~noble1` (for Ubuntu Noble)

2. **Always test locally** before uploading to PPA

3. **Build for multiple releases** to maximize user coverage

4. **Keep changelog updated** with meaningful descriptions

5. **Monitor build logs** for warnings and errors

6. **Sign all uploads** with your GPG key

## Automation

### GitHub Actions Integration

You can automate PPA uploads on new releases by adding to `.github/workflows/release.yml`:

```yaml
- name: Upload to PPA
  if: startsWith(github.ref, 'refs/tags/v')
  env:
    GPG_KEY: ${{ secrets.GPG_KEY_ID }}
    GPG_PRIVATE_KEY: ${{ secrets.GPG_PRIVATE_KEY }}
  run: |
    # Import GPG key
    echo "$GPG_PRIVATE_KEY" | gpg --import

    # Run upload script
    ./scripts/ppa-upload.sh
```

Store your GPG private key in GitHub Secrets:

```bash
# Export private key
gpg --armor --export-secret-keys ABCD1234EFGH5678 > private.key

# Add to GitHub Secrets as GPG_PRIVATE_KEY
```

## Resources

- [Launchpad PPA Help](https://help.launchpad.net/Packaging/PPA)
- [Ubuntu Packaging Guide](https://packaging.ubuntu.com/)
- [Debian Policy Manual](https://www.debian.org/doc/debian-policy/)
- [dput Documentation](https://manpages.ubuntu.com/manpages/noble/man1/dput.1.html)

## Support

For issues with the PPA setup:
- Email: blackwellsystems@protonmail.com
- GitHub Issues: https://github.com/blackwell-systems/pipeboard/issues
- Launchpad Questions: https://answers.launchpad.net/pipeboard
