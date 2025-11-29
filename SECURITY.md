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
  - **Security note**: If your clipboard contains sensitive data, the history files will also contain that data. Consider disabling history or periodically clearing `~/.config/pipeboard/`

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

## Reporting Vulnerabilities

If you discover a security vulnerability, please open an issue or contact the maintainers directly.

## Supported Versions

Only the latest release is supported with security updates.
