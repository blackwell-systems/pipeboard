# Sync & Remote

pipeboard supports two sync methods:

1. **SSH Peer Sync** — Direct machine-to-machine transfer over SSH
2. **S3 Remote Slots** — Persistent named storage in S3

## SSH Peer Sync

Send clipboard directly between machines over SSH. No cloud, no intermediary. Requires pipeboard installed on both machines.

### Setup

1. Ensure SSH access to the remote machine
2. Ensure pipeboard is installed on the remote machine
3. Add the peer to your config:

```yaml
# ~/.config/pipeboard/config.yaml
peers:
  dev:
    ssh: devbox           # SSH host from ~/.ssh/config
  mac:
    ssh: user@192.168.1.10
  server:
    ssh: prod-server
```

### Commands

```bash
# Send local clipboard to peer
pipeboard send dev

# Receive peer's clipboard
pipeboard recv dev

# View peer's clipboard (no local change)
pipeboard peek dev
```

### Default Peer

Set a default peer to avoid typing it every time:

```yaml
defaults:
  peer: dev
```

```bash
# Uses default peer
pipeboard send
pipeboard recv
pipeboard peek
```

### How It Works

1. `send` runs `pipeboard copy` on the remote via SSH, piping local clipboard
2. `recv` runs `pipeboard paste` on the remote, copying output to local clipboard
3. `peek` runs `pipeboard paste` on the remote, printing to stdout

This means:
- Both machines need pipeboard installed
- SSH key auth recommended (avoid password prompts)
- Works across different platforms (Mac → Linux, etc.)

## S3 Remote Slots

Named, persistent clipboard storage backed by S3. Perfect for:

- Async workflows (copy now, paste hours later)
- Cross-network transfers (machines not on same network)
- Backup important snippets

### Setup

1. Configure AWS credentials (env vars, profile, or IAM role)
2. Create an S3 bucket
3. Add sync config:

```yaml
# ~/.config/pipeboard/config.yaml
sync:
  backend: s3
  s3:
    bucket: my-pipeboard-bucket
    region: us-west-2
    prefix: clips/              # optional key prefix
```

### Commands

```bash
# Push clipboard to a named slot
pipeboard push myslot

# Pull from slot to clipboard
pipeboard pull myslot

# View slot without modifying clipboard
pipeboard show myslot

# List all slots
pipeboard slots

# Delete a slot
pipeboard rm myslot
```

### Encryption

Enable client-side AES-256-GCM encryption:

```yaml
sync:
  backend: s3
  encryption: aes256
  passphrase: ${PIPEBOARD_PASSPHRASE}  # use env var
  s3:
    bucket: my-pipeboard-bucket
    region: us-west-2
```

With encryption:
- Data is encrypted before upload
- Only you can decrypt (passphrase required)
- S3 stores ciphertext only

### TTL / Auto-Expiry

Automatically expire old slots:

```yaml
sync:
  backend: s3
  ttl_days: 30              # auto-delete after 30 days
  s3:
    bucket: my-pipeboard-bucket
    region: us-west-2
```

> **Note:** TTL uses S3 lifecycle rules. You may need to configure the bucket lifecycle policy.

### Server-Side Encryption

Enable S3 server-side encryption (in addition to optional client-side):

```yaml
sync:
  backend: s3
  s3:
    bucket: my-pipeboard-bucket
    region: us-west-2
    sse: AES256              # or aws:kms
```

### AWS Authentication

pipeboard uses the standard AWS SDK credential chain:

1. Environment variables (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`)
2. Shared credentials file (`~/.aws/credentials`)
3. IAM role (EC2, ECS, Lambda)

Specify a named profile:

```yaml
sync:
  backend: s3
  s3:
    bucket: my-pipeboard-bucket
    region: us-west-2
    profile: my-aws-profile
```

Or via environment:

```bash
export PIPEBOARD_S3_PROFILE=my-aws-profile
```

## Environment Variables

Override sync settings with environment variables:

```bash
PIPEBOARD_BACKEND          # sync backend type
PIPEBOARD_S3_BUCKET        # S3 bucket name
PIPEBOARD_S3_REGION        # AWS region
PIPEBOARD_S3_PREFIX        # key prefix
PIPEBOARD_S3_PROFILE       # AWS profile name
PIPEBOARD_S3_SSE           # server-side encryption
PIPEBOARD_PASSPHRASE       # encryption passphrase
```

## Example Workflows

### Share kubeconfig between machines

```bash
# On laptop
cat ~/.kube/config | pipeboard copy
pipeboard push kube

# On server (later)
pipeboard pull kube
pipeboard paste > ~/.kube/config
```

### Quick snippet sharing with team

```bash
# You
pipeboard push meeting-notes

# Teammate (with same S3 access)
pipeboard pull meeting-notes
```

### Backup before experimenting

```bash
# Save current clipboard
pipeboard push backup

# Do risky things...

# Restore if needed
pipeboard pull backup
```
