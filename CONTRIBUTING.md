# Contributing to pipeboard

Thanks for your interest in contributing to pipeboard! This document outlines the process for contributing to the project.

## Getting Started

1. **Fork the repository** on GitHub
2. **Clone your fork** locally:
   ```bash
   git clone https://github.com/YOUR_USERNAME/pipeboard.git
   cd pipeboard
   ```
3. **Build and test** to make sure everything works:
   ```bash
   go build
   go test ./...
   ```

## Development Setup

### Requirements

- Go 1.21 or later
- For clipboard operations: platform-specific tools (see README)
- For S3 sync testing: AWS credentials or localstack

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with race detection
go test -race ./...
```

### Building

```bash
# Build binary
go build

# Build for specific platform
GOOS=linux GOARCH=amd64 go build
```

## Making Changes

### Code Style

- Follow standard Go conventions (`gofmt`, `go vet`)
- Keep functions focused and small
- Add comments for exported functions
- Error messages should be lowercase and not end with punctuation

### Commit Messages

- Use present tense ("Add feature" not "Added feature")
- Keep the first line under 72 characters
- Reference issues when relevant: "Fix clipboard detection (#42)"

### Pull Request Process

1. **Create a branch** for your changes:
   ```bash
   git checkout -b feature/my-feature
   ```

2. **Make your changes** and commit them

3. **Run tests** to ensure nothing is broken:
   ```bash
   go test ./...
   go vet ./...
   ```

4. **Push** to your fork and open a Pull Request

5. **Describe your changes** in the PR description:
   - What does this PR do?
   - Why is this change needed?
   - How was it tested?

## What to Contribute

### Good First Issues

Look for issues labeled `good first issue` - these are specifically chosen as accessible entry points for new contributors.

### Areas We'd Love Help With

- **New backends**: Redis, WebDAV, SFTP
- **Platform support**: Better Windows integration, BSD support
- **Transforms**: Useful default transforms (see `fx` in docs)
- **Documentation**: Tutorials, examples, translations
- **Testing**: Increase test coverage, especially for platform-specific code

### Before Starting Major Work

For large changes, please open an issue first to discuss the approach. This helps avoid wasted effort if the change doesn't align with project goals.

## Code of Conduct

Be respectful and constructive. We're all here to build something useful.

## Questions?

- Open an issue for bugs or feature requests
- Check existing issues before opening a new one

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

---

pipeboard™ is a product of Blackwell Systems™.
