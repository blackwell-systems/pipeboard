# Integration Test Scripts

This directory contains integration test scripts for pipeboard that require real environments (Docker, SSH, etc.) rather than unit tests.

## Available Scripts

### `test-ssh-peer.sh`

End-to-end integration test for SSH peer synchronization.

**Purpose**: Validates that pipeboard can successfully sync clipboard content between machines over SSH.

**Requirements**:
- Docker
- Bash
- Internet connection (for Alpine images and Go dependencies)

**Usage**:
```bash
./test-ssh-peer.sh
```

**What it tests**:
1. Basic text transfer (sender → receiver)
2. Multiline content transfer
3. Bidirectional transfer (receiver → sender)

**Test environment**:
- Two Alpine Linux containers
- Bidirectional SSH with key-based auth
- Xvfb virtual X server for clipboard access
- xclip with custom wrapper for headless operation
- pipeboard built from source in both containers

**Duration**: ~2 minutes (first run with image pulls/builds), ~30 seconds (subsequent runs)

See [Testing Documentation](../docs/testing.md) for detailed information.

## Adding New Integration Tests

When adding new integration test scripts:

1. Name them descriptively: `test-<feature>.sh`
2. Make them executable: `chmod +x test-<feature>.sh`
3. Include cleanup logic (use trap for EXIT)
4. Provide clear success/failure output
5. Document in this README
6. Add to [docs/testing.md](../docs/testing.md)
