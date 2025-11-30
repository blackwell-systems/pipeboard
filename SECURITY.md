# Security Policy

## Overview

pipeboard is a clipboard utility that handles potentially sensitive data. This document describes its security model.

## Data Handling

- **No logging**: pipeboard does not log clipboard contents to stdout during normal operation
- **No telemetry**: No data is sent to third parties
- **Local history**: pipeboard maintains an optional local clipboard history for the `recall` command:
  - Operation history stored in `~/.config/pipeboard/history.json` (command metadata only, no content)
  - Clipboard content history stored in `~/.config/pipeboard/clipboard_history.json` (up to 20 entries with full content)
  - History files are stored with `0600` permissions
  - **Encrypted history**: When encryption is enabled in your sync config (`encryption: aes256`), clipboard history content and previews are automatically encrypted using the same passphrase
  - **Security note**: If your clipboard contains sensitive data and encryption is not enabled, the history files will contain that data in plaintext. Enable encryption or periodically clear `~/.config/pipeboard/`

## Transport Security

### SSH Peer Sync (`send`/`recv`/`peek`)

- Uses your existing SSH configuration and authentication
- Data is encrypted in transit via SSH
- No additional credentials required beyond SSH access

### S3 Remote Slots (`push`/`pull`/`show`/`slots`/`rm`)

- Uses AWS SDK with standard credential chain (env vars, profiles, IAM roles)
- Supports server-side encryption (`sse: AES256` or `sse: aws:kms` in config)
- Data is encrypted in transit via HTTPS
- Bucket permissions are your responsibility

## Configuration

- Config file: `~/.config/pipeboard/config.yaml`
- Permissions: Recommend `chmod 600` for config files containing sensitive paths
- No secrets stored: AWS credentials come from standard AWS credential sources, not the config file

## Recommendations

1. **Use SSE for S3**: Enable `sse: AES256` or `sse: aws:kms` in your config
2. **Restrict S3 bucket access**: Use IAM policies to limit who can read/write slots
3. **SSH key auth**: Use SSH keys rather than passwords for peer sync
4. **Don't commit configs**: Add `config.yaml` to `.gitignore` if it contains sensitive peer names

## Verifying Releases

All releases are signed using [Cosign](https://github.com/sigstore/cosign) keyless signing with GitHub Actions OIDC.

To verify a release:

```bash
# Install cosign
# macOS: brew install cosign
# Linux: https://github.com/sigstore/cosign/releases

# Download the release files
curl -LO https://github.com/blackwell-systems/pipeboard/releases/download/v0.6.0/checksums.txt
curl -LO https://github.com/blackwell-systems/pipeboard/releases/download/v0.6.0/checksums.txt.sig
curl -LO https://github.com/blackwell-systems/pipeboard/releases/download/v0.6.0/checksums.txt.sig.pem

# Verify the signature
cosign verify-blob \
  --signature checksums.txt.sig \
  --certificate checksums.txt.sig.pem \
  --certificate-identity-regexp "https://github.com/blackwell-systems/pipeboard" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
  checksums.txt

# Then verify your downloaded binary against checksums.txt
sha256sum -c checksums.txt --ignore-missing
```

## Reporting Vulnerabilities

If you discover a security vulnerability, please report it privately via [GitHub Security Advisories](https://github.com/blackwell-systems/pipeboard/security/advisories/new) or email security@blackwell-systems.com.

Please do not open public issues for security vulnerabilities.

## Supported Versions

Only the latest release is supported with security updates.
