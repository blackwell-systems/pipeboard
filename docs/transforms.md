# Transforms (fx)

Transforms let you process clipboard contents in-place using external commands. Define them once in your config, use them forever.

## Basic Usage

```bash
# Run a transform
pipeboard fx pretty-json

# Preview without modifying clipboard
pipeboard fx pretty-json --dry-run

# List available transforms
pipeboard fx --list
```

## Chaining Transforms

Run multiple transforms in sequence. Output from each step feeds into the next.

```bash
# Chain: strip ANSI codes → redact secrets → format JSON
pipeboard fx strip-ansi redact-secrets pretty-json
```

**Safety guarantees:**
- If any transform in the chain fails, the clipboard is unchanged
- Empty output is treated as an error (clipboard unchanged)
- `--dry-run` prints final result to stdout, never touches clipboard

## Defining Transforms

Add transforms to your config file (`~/.config/pipeboard/config.yaml`):

```yaml
fx:
  pretty-json:
    cmd: ["jq", "."]
    description: "Format JSON"

  strip-ansi:
    shell: "sed 's/\\x1b\\[[0-9;]*m//g'"
    description: "Remove ANSI escape codes"

  redact-secrets:
    shell: "sed -E 's/(AKIA[0-9A-Z]{16}|sk-[a-zA-Z0-9]{48})/<REDACTED>/g'"
    description: "Redact AWS keys and OpenAI tokens"
```

### cmd vs shell

**`cmd`** — Array of command and arguments. Safer, no shell interpretation.

```yaml
pretty-json:
  cmd: ["jq", "."]
```

**`shell`** — String passed to `/bin/sh -c`. Supports pipes, redirection, shell features.

```yaml
strip-ansi:
  shell: "sed 's/\\x1b\\[[0-9;]*m//g'"
```

Use `cmd` when possible. Use `shell` when you need shell features.

## Example Transforms

### JSON

```yaml
fx:
  pretty-json:
    cmd: ["jq", "."]
    description: "Format JSON with jq"

  compact-json:
    cmd: ["jq", "-c", "."]
    description: "Compact JSON to single line"

  json-keys:
    cmd: ["jq", "keys"]
    description: "Extract JSON keys"
```

### YAML

```yaml
fx:
  yaml-to-json:
    cmd: ["yq", "-o", "json"]
    description: "Convert YAML to JSON"

  json-to-yaml:
    cmd: ["yq", "-P"]
    description: "Convert JSON to YAML"
```

### Text Processing

```yaml
fx:
  strip-ansi:
    shell: "sed 's/\\x1b\\[[0-9;]*m//g'"
    description: "Remove ANSI escape codes"

  sort-lines:
    shell: "sort | uniq"
    description: "Sort and deduplicate lines"

  trim:
    shell: "sed 's/^[[:space:]]*//;s/[[:space:]]*$//'"
    description: "Trim leading/trailing whitespace"

  lowercase:
    cmd: ["tr", "A-Z", "a-z"]
    description: "Convert to lowercase"

  uppercase:
    cmd: ["tr", "a-z", "A-Z"]
    description: "Convert to uppercase"
```

### Encoding

```yaml
fx:
  base64-encode:
    cmd: ["base64"]
    description: "Encode as base64"

  base64-decode:
    cmd: ["base64", "-d"]
    description: "Decode base64"

  url-encode:
    cmd: ["python3", "-c", "import sys,urllib.parse;print(urllib.parse.quote(sys.stdin.read().strip()))"]
    description: "URL encode"

  url-decode:
    cmd: ["python3", "-c", "import sys,urllib.parse;print(urllib.parse.unquote(sys.stdin.read().strip()))"]
    description: "URL decode"
```

### Security

```yaml
fx:
  redact-secrets:
    shell: "sed -E 's/(AKIA[0-9A-Z]{16}|sk-[a-zA-Z0-9]{48}|ghp_[a-zA-Z0-9]{36})/<REDACTED>/g'"
    description: "Redact AWS keys, OpenAI tokens, GitHub tokens"

  redact-emails:
    shell: "sed -E 's/[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}/<EMAIL>/g'"
    description: "Redact email addresses"

  redact-ips:
    shell: "sed -E 's/[0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3}/<IP>/g'"
    description: "Redact IP addresses"
```

### Development

```yaml
fx:
  markdown-to-html:
    cmd: ["pandoc", "-f", "markdown", "-t", "html"]
    description: "Convert Markdown to HTML"

  html-to-text:
    cmd: ["pandoc", "-f", "html", "-t", "plain"]
    description: "Strip HTML to plain text"

  sql-format:
    cmd: ["sqlformat", "-r", "-k", "upper", "-"]
    description: "Format SQL"
```

## Real-World Workflows

### Share logs without secrets

```bash
# Copy log output, then:
pipeboard fx strip-ansi redact-secrets
# Now safe to paste into Slack/ticket
```

### Format JSON from API response

```bash
curl -s api.example.com/data | pipeboard copy
pipeboard fx pretty-json
pipeboard paste  # nicely formatted
```

### Decode JWT payload

```yaml
fx:
  jwt-payload:
    shell: "cut -d. -f2 | base64 -d 2>/dev/null | jq ."
    description: "Decode JWT payload"
```

```bash
pipeboard fx jwt-payload --dry-run
```

### Clean up copied code

```bash
pipeboard fx strip-ansi trim
```
