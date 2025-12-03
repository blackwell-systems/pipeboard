# Architecture

This page describes the high-level architecture and component interactions of the pipeboard system.

## System Overview

```mermaid
graph TB
    subgraph "User Interface"
        CLI[pipeboard CLI]
        SHELL[Shell Aliases]
    end

    subgraph "Core Components"
        CLIPBOARD[Clipboard Manager]
        TRANSFORM[Transform Engine]
        HISTORY[History Tracker]
        CONFIG[Configuration]
    end

    subgraph "Storage Backends"
        LOCAL[Local Slots]
        S3[S3 Remote]
        SSH[SSH Peer]
    end

    subgraph "Platform Adapters"
        MACOS[macOS pbcopy/pbpaste]
        LINUX_WL[Linux Wayland]
        LINUX_X[Linux X11]
        WINDOWS[Windows clip.exe]
    end

    CLI --> CLIPBOARD
    CLI --> TRANSFORM
    CLI --> HISTORY
    CLIPBOARD --> TRANSFORM
    TRANSFORM --> LOCAL
    TRANSFORM --> S3
    TRANSFORM --> SSH

    CLIPBOARD --> MACOS
    CLIPBOARD --> LINUX_WL
    CLIPBOARD --> LINUX_X
    CLIPBOARD --> WINDOWS

    CONFIG --> CLIPBOARD
    CONFIG --> TRANSFORM
```

## Data Flow

The lifecycle of clipboard data through pipeboard:

```mermaid
flowchart LR
    A[Copy] --> B{Transform?}
    B -->|Yes| C[Apply Transforms]
    B -->|No| D[Raw Data]
    C --> D
    D --> E{Storage}

    E --> F[Local Slot]
    E --> G[S3 Remote]
    E --> H[SSH Peer]

    F --> I[History]
    G --> I
    H --> I

    I --> J[Paste/Recall]
```

## Component Architecture

### CLI Entry Point

The `pipeboard` command provides the main interface:

```mermaid
graph LR
    A[pipeboard] --> B{Command}
    B --> C[copy]
    B --> D[paste]
    B --> E[slot]
    B --> F[sync]
    B --> G[history]
    B --> H[config]
    B --> I[transform]

    C --> C1[read/write]
    C --> C2[pipe]

    E --> E1[save]
    E --> E2[load]
    E --> E3[list]

    F --> F1[push]
    F --> F2[pull]
    F --> F3[status]
```

### Transform Pipeline

Transforms process clipboard data before storage:

```mermaid
flowchart TD
    A[Input Data] --> B{Has Transforms?}
    B -->|No| G[Output Data]
    B -->|Yes| C[Load Transform Chain]

    C --> D[Transform 1]
    D --> E[Transform 2]
    E --> F[Transform N]
    F --> G

    subgraph "Built-in Transforms"
        T1[trim]
        T2[lowercase]
        T3[uppercase]
        T4[base64]
        T5[json-format]
        T6[url-encode]
    end

    subgraph "Custom Transforms"
        T7[User Scripts]
        T8[Shell Commands]
    end
```

## Sync Architecture

Pipeboard supports multiple sync backends:

```mermaid
sequenceDiagram
    participant User
    participant CLI as pipeboard CLI
    participant Local as Local Slots
    participant Remote as Remote Backend

    User->>CLI: pipeboard copy
    CLI->>Local: Save to slot

    User->>CLI: pipeboard push
    CLI->>Local: Read slot data
    CLI->>Remote: Upload (S3/SSH)

    User->>CLI: pipeboard pull (other machine)
    CLI->>Remote: Download data
    Remote-->>CLI: Return encrypted data
    CLI->>Local: Save to local slot

    User->>CLI: pipeboard paste
    CLI->>Local: Read slot
    CLI-->>User: Output to clipboard
```

### Sync Backends

```mermaid
graph TB
    subgraph "Sync Options"
        A[pipeboard sync]
    end

    subgraph "Backend Implementations"
        B[Local Filesystem]
        C[S3 Bucket]
        D[SSH Transfer]
    end

    subgraph "Features"
        E[Encryption]
        F[Compression]
        G[History]
        H[Conflict Resolution]
    end

    A --> B
    A --> C
    A --> D

    B --> E
    C --> E
    D --> E

    E --> F
    F --> G
    G --> H
```

## Platform Abstraction

Cross-platform clipboard access:

```mermaid
graph LR
    A[Clipboard Manager] --> B{Detect Platform}

    B --> C[macOS]
    B --> D[Linux]
    B --> E[Windows]
    B --> F[WSL]

    C --> C1[pbcopy/pbpaste]

    D --> D1{Display Server}
    D1 --> D2[Wayland: wl-copy/wl-paste]
    D1 --> D3[X11: xclip/xsel]

    E --> E1[clip.exe + PowerShell]

    F --> F1[clip.exe via /mnt/c]
```

## Directory Structure

```
pipeboard/
├── main.go                 # Main entry point
├── cli.go                  # CLI interface and help text
├── clipboard.go            # Clipboard operations (copy/paste)
├── backend.go              # Platform clipboard detection
├── config.go               # Configuration management
├── crypto.go               # AES-256-GCM encryption
├── history.go              # Command and clipboard history
├── local.go                # Local filesystem backend
├── remote.go               # S3 backend with compression
├── peer.go                 # SSH peer sync
├── watch.go                # Real-time bidirectional sync
├── slots.go                # Remote slot management
├── fx.go                   # Transform pipelines
├── init.go                 # Interactive setup wizard
├── completion.go           # Shell completions
├── run.go                  # Subprocess execution
├── *_test.go               # Test files
└── docs/                   # Documentation
```

## Configuration Flow

```mermaid
flowchart TD
    A[pipeboard Command] --> B[Load Config]
    B --> C[config.yaml]

    C --> D{Config Type}
    D --> E[Clipboard Settings]
    D --> F[Transform Rules]
    D --> G[Sync Settings]
    D --> H[Storage Backends]

    E --> I[Execute Command]
    F --> I
    G --> I
    H --> I

    I --> J{Success?}
    J -->|Yes| K[Update History]
    J -->|No| L[Error Handler]
```

## Security Architecture

```mermaid
graph TB
    subgraph "Data at Rest"
        A[Local Slots] --> E[AES-256 Encryption]
        B[S3 Storage] --> E
        C[SSH Transfer] --> E
    end

    subgraph "Encryption Layer"
        E --> F[User Key/Password]
        F --> G[Key Derivation]
    end

    subgraph "Data in Transit"
        H[Network Transfer]
        I[TLS/SSH]
    end

    E --> H
    H --> I
```

## Transform System

Transforms are applied in a configurable chain:

```mermaid
flowchart LR
    A[Raw Input] --> B{Transform 1}
    B --> C{Transform 2}
    C --> D{Transform N}
    D --> E[Processed Output]

    subgraph "Transform Types"
        F[Text Processing]
        G[Encoding]
        H[Formatting]
        I[Custom Scripts]
    end

    B -.-> F
    C -.-> G
    D -.-> I
```

## Error Handling

```mermaid
flowchart TD
    A[Operation] --> B{Check}
    B -->|Valid| C[Execute]
    B -->|Invalid| D[Validate Input]

    C --> E{Success?}
    E -->|Yes| F[Log Success]
    E -->|No| G[Error Handler]

    G --> H{Error Type}
    H --> I[Network Error: Retry]
    H --> J[Permission Error: Report]
    H --> K[Config Error: Guide User]

    D --> L[Return Error Message]
```

## Key Design Decisions

### 1. Slot-Based Storage
Instead of a single clipboard history, pipeboard uses named "slots" for organized clipboard management. This allows:
- Categorization (work vs personal)
- Long-term storage
- Easy retrieval by name

### 2. Transform Pipeline
Transforms are composable and can be chained. This enables:
- Reusable data processing
- Custom workflows
- Plugin-like extensibility

### 3. P2P Sync
Direct machine-to-machine transfer via SSH reduces cloud dependencies:
- No intermediary storage required
- Lower latency
- Privacy-focused

### 4. Encryption by Default
All stored data is encrypted before writing to disk or network:
- AES-256 encryption
- User-controlled keys
- Zero-knowledge architecture

### 5. Platform Abstraction
Unified clipboard API regardless of OS:
- Automatic platform detection
- Graceful fallbacks
- Consistent behavior across platforms

## Performance Characteristics

| Operation | Typical Latency |
|-----------|----------------|
| Local copy/paste | < 10ms |
| Transform application | 10-50ms |
| Local slot save | < 20ms |
| S3 sync | 100-500ms |
| SSH peer transfer | 50-200ms |
| History query | < 5ms |

## Extensibility Points

1. **Custom Transforms** - User-defined scripts in `~/.config/pipeboard/transforms/`
2. **Storage Backends** - Pluggable storage interface
3. **Platform Support** - New platform adapters via clipboard interface
4. **Sync Protocols** - Additional sync methods beyond S3/SSH

## Comparison: Pipeboard vs Traditional Clipboard

| Feature | Traditional Clipboard | Pipeboard |
|---------|---------------------|-----------|
| Cross-machine sync | No | Yes (S3, SSH, P2P) |
| Multiple slots | No (single clipboard) | Yes (named slots) |
| Transform pipeline | No | Yes (chainable) |
| History | Limited | Full history with search |
| Encryption | No | Yes (AES-256) |
| Platform support | OS-specific | Unified API |

## Data Model

### Slot Structure

```json
{
  "name": "slot-name",
  "content": "clipboard data",
  "transforms": ["trim", "lowercase"],
  "created": "2025-12-01T12:00:00Z",
  "modified": "2025-12-01T12:00:00Z",
  "metadata": {
    "size": 1024,
    "encoding": "utf-8",
    "encrypted": true
  }
}
```

### Configuration Schema

```yaml
clipboard:
  default_slot: "main"
  auto_save: true

transforms:
  - name: "cleanup"
    steps: ["trim", "normalize"]

sync:
  backend: "s3"
  auto_sync: false

storage:
  s3:
    bucket: "my-clipboard"
    region: "us-east-1"
  ssh:
    host: "peer-machine"
    path: "/home/user/.pipeboard/slots"
```

## See Also

- [Roadmap](roadmap.md) - Future enhancement ideas and feature proposals
