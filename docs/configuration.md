# Configuration

pipeboard uses a YAML config file at `~/.config/pipeboard/config.yaml`.

## Full Example

```yaml
version: 1

# Default settings
defaults:
  peer: dev                    # default peer for send/recv/peek

# SSH peers for direct sync
peers:
  dev:
    ssh: devbox                # SSH host from ~/.ssh/config
  mac:
    ssh: dayna-mac.local
  server:
    ssh: user@prod.example.com

# Clipboard transforms
fx:
  pretty-json:
    cmd: ["jq", "."]
    description: "Format JSON"

  compact-json:
    cmd: ["jq", "-c", "."]
    description: "Compact JSON"

  strip-ansi:
    shell: "sed 's/\\x1b\\[[0-9;]*m//g'"
    description: "Remove ANSI codes"

  redact-secrets:
    shell: "sed -E 's/(AKIA[0-9A-Z]{16}|sk-[a-zA-Z0-9]{48})/<REDACTED>/g'"
    description: "Redact AWS/OpenAI keys"

  base64-decode:
    cmd: ["base64", "-d"]
    description: "Decode base64"

# Slot aliases (shortcuts)
aliases:
  k: kube-config
  p: prod-secrets
  aws: aws-credentials

# Clipboard history
history:
  limit: 50              # max entries (default: 20)
  ttl_days: 30           # auto-delete after 30 days
  no_duplicates: true    # skip duplicate content

# S3 remote storage
sync:
  backend: s3
  encryption: aes256           # client-side encryption
  passphrase: ${PIPEBOARD_PASSPHRASE}
  ttl_days: 30                 # auto-expire slots
  s3:
    bucket: my-pipeboard
    region: us-west-2
    prefix: clips/
    sse: AES256                # server-side encryption
    profile: default           # AWS profile
```

## Reference

### version

Config file version. Currently `1`.

```yaml
version: 1
```

### defaults

Default settings.

```yaml
defaults:
  peer: dev    # default peer for send/recv/peek commands
```

### peers

SSH peers for direct clipboard sync.

```yaml
peers:
  <name>:
    ssh: <host>    # SSH host (from ~/.ssh/config or user@host)
```

Example:

```yaml
peers:
  dev:
    ssh: devbox
  mac:
    ssh: dayna@macbook.local
  prod:
    ssh: deploy@prod.example.com:2222
```

### fx

Clipboard transforms. See [Transforms](transforms.md) for details.

```yaml
fx:
  <name>:
    cmd: [...]           # command as array (preferred)
    # OR
    shell: "..."         # shell command string
    description: "..."   # optional description for --list
```

**cmd** — Array of command and arguments. No shell interpretation.

```yaml
pretty-json:
  cmd: ["jq", "."]
```

**shell** — String passed to `/bin/sh -c`. Supports pipes, redirection.

```yaml
strip-ansi:
  shell: "sed 's/\\x1b\\[[0-9;]*m//g'"
```

### aliases

Slot name shortcuts for frequently used slots.

```yaml
aliases:
  <short>: <full-slot-name>
```

Example:

```yaml
aliases:
  k: kube-config
  p: prod-secrets
  aws: aws-credentials
  db: database-connection-string
```

Now use short names with any slot command:

```bash
pipeboard push k       # pushes to "kube-config"
pipeboard pull p       # pulls from "prod-secrets"
pipeboard show aws     # shows "aws-credentials"
pipeboard rm db        # removes "database-connection-string"
```

**Note:** Aliases only apply to slot commands (`push`, `pull`, `show`, `rm`). The full slot name is always used for storage.

### history

Clipboard history settings.

```yaml
history:
  limit: 50           # max clipboard history entries (default: 20)
  ttl_days: 30        # auto-delete entries older than N days (0 = never)
  no_duplicates: true # skip entries with same content (checks all history)
```

**Options:**

| Option | Default | Description |
|--------|---------|-------------|
| `limit` | `20` | Maximum number of clipboard history entries to keep |
| `ttl_days` | `0` | Auto-delete entries older than N days (0 = disabled) |
| `no_duplicates` | `false` | Skip duplicate content across all history entries |

**Note:** Without `no_duplicates`, pipeboard only checks if new content matches the *most recent* entry. With `no_duplicates: true`, it checks all entries.

### sync

Remote storage configuration.

```yaml
sync:
  backend: s3              # "s3" or "local"
  encryption: aes256       # optional: client-side encryption
  passphrase: <string>     # encryption passphrase (use env var)
  ttl_days: <number>       # optional: auto-expire after N days
  s3:
    bucket: <bucket-name>  # required for s3
    region: <aws-region>   # required for s3
    prefix: <key-prefix>   # optional: prefix for S3 keys
    sse: <AES256|aws:kms>  # optional: server-side encryption
    profile: <profile>     # optional: AWS profile name
  local:
    path: <directory>      # optional: defaults to ~/.config/pipeboard/slots
```

**Backends:**

- `s3` — Store slots in AWS S3 (requires bucket, region)
- `local` — Store slots on local filesystem (zero config needed)

## Environment Variables

Environment variables override config file settings.

### Config Location

```bash
PIPEBOARD_CONFIG           # path to config file
```

### Sync Settings

```bash
PIPEBOARD_BACKEND          # sync backend (s3)
PIPEBOARD_PASSPHRASE       # encryption passphrase
```

### S3 Settings

```bash
PIPEBOARD_S3_BUCKET        # bucket name
PIPEBOARD_S3_REGION        # AWS region
PIPEBOARD_S3_PREFIX        # key prefix
PIPEBOARD_S3_PROFILE       # AWS profile
PIPEBOARD_S3_SSE           # server-side encryption
```

### AWS Credentials

Standard AWS SDK environment variables:

```bash
AWS_ACCESS_KEY_ID
AWS_SECRET_ACCESS_KEY
AWS_SESSION_TOKEN
AWS_PROFILE
AWS_REGION
```

## Config File Location

Default: `~/.config/pipeboard/config.yaml`

Override with environment variable:

```bash
export PIPEBOARD_CONFIG=/path/to/config.yaml
pipeboard <command>
```

## Minimal Configs

### Local only (no config needed)

pipeboard works without any config for local clipboard operations:

```bash
echo "hello" | pipeboard copy
pipeboard paste
```

### Just transforms

```yaml
version: 1
fx:
  pretty-json:
    cmd: ["jq", "."]
```

### Just SSH peers

```yaml
version: 1
defaults:
  peer: dev
peers:
  dev:
    ssh: devbox
```

### Just local slots

```yaml
version: 1
sync:
  backend: local
```

### Just S3 slots

```yaml
version: 1
sync:
  backend: s3
  s3:
    bucket: my-bucket
    region: us-west-2
```

### Just slot aliases

```yaml
version: 1
aliases:
  k: kube-config
  p: prod-secrets
```

### Local slots with aliases

```yaml
version: 1
sync:
  backend: local
aliases:
  k: kube-config
  p: prod-secrets
  aws: aws-credentials
```
