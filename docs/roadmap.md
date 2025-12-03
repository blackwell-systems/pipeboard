# Roadmap

This page tracks potential enhancements and feature ideas for pipeboard. Items are organized by theme, not priority.

## Sync & Collaboration

### Real-time Sync
WebSocket-based instant sync between machines, beyond the current polling-based watch mode.

### Conflict Resolution
CRDT-based merge for concurrent edits when multiple machines modify the same slot.

### Team Slots
Shared slots with access control—push to a team slot, teammates can pull. Requires auth layer.

### Offline Queue
Queue S3 operations when offline, automatically sync when connectivity returns.

### Remote History
Sync clipboard history to S3, not just named slots. Access your history from any machine.

## Transforms & Processing

### Built-in Transforms
Ship common transforms out of the box:
- `pretty-json`, `minify-json`
- `base64-encode`, `base64-decode`
- `url-encode`, `url-decode`
- `sha256`, `md5`
- `sort-lines`, `unique-lines`
- `strip-ansi`, `strip-html`

### Auto-Transform Rules
Automatically apply transforms based on content detection:
```yaml
auto_transforms:
  - match: "application/json"
    apply: pretty-json
  - match: "^\\s*<"
    apply: format-xml
```

### Clipboard OCR
Extract text from images in clipboard using Tesseract or cloud OCR.

### Clipboard Diff
Compare two slots or history entries:
```bash
pipeboard diff slot1 slot2
pipeboard diff --history 1 3
```

## Storage & Organization

### Slot Aliases
Shortcuts for frequently used slots:
```yaml
aliases:
  k: kube-config
  p: prod-secrets
```
```bash
pipeboard pull k    # expands to kube-config
```

### Clipboard Templates
Parameterized templates with placeholders:
```bash
pipeboard template api-request --var endpoint=/users --var token=$TOKEN
```

### Clipboard Pinning
Pin important items to prevent accidental overwrite:
```bash
pipeboard pin important-config
pipeboard copy "new stuff"    # doesn't overwrite pinned slot
```

### Smart TTL
Different TTL per slot based on naming conventions or tags:
```yaml
ttl_rules:
  - match: "tmp-*"
    ttl_days: 1
  - match: "secrets-*"
    ttl_days: 7
```

### Export/Import
Backup all slots to a single encrypted archive:
```bash
pipeboard export --all > backup.pb
pipeboard import < backup.pb
```

## Search & Discovery

### Fuzzy Search
Fuzzy matching in history search:
```bash
pipeboard history --local --fuzzy "kubctl"    # matches "kubectl"
```

### Content Tags
Tag slots for organization and filtering:
```bash
pipeboard push config --tag work --tag k8s
pipeboard slots --tag work
```

## Integration & Extensibility

### Plugin System
Dynamic loading of transforms and backends from external packages.

### Webhooks
Trigger HTTP webhooks on clipboard events:
```yaml
webhooks:
  on_push:
    - url: https://example.com/hook
      slots: ["deploy-*"]
```

### Browser Extension
Sync browser clipboard with pipeboard. Copy in browser, paste in terminal.

### Mobile Apps
iOS/Android apps for clipboard access on the go. Push from phone, pull on laptop.

### IDE Integration
VS Code / JetBrains plugins for direct slot access from editor.

## Platform & Performance

### Native GUI
Optional system tray app for quick access without terminal.

### Streaming Large Files
Stream large clipboard contents instead of loading entirely into memory.

### Compression Options
Configurable compression algorithms (zstd, lz4) for different size/speed tradeoffs.

---

## Contributing Ideas

Have an idea? Open an issue on [GitHub](https://github.com/blackwell-systems/pipeboard/issues) with the `enhancement` label.

When proposing features, consider:
- **Use case**: What problem does this solve?
- **Scope**: Is this a core feature or a plugin?
- **Compatibility**: Does it work across all platforms?
