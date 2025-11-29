# Claude Code Instructions for Pipeboard

## Git Safety Rules

**CRITICAL: Follow these rules to prevent merge conflicts and diverging branches.**

1. **Always sync before working:**
   - Run `git fetch && git status` at the start of every session
   - If the branch has diverged from remote, STOP and ask the user before proceeding
   - Run `git pull --rebase` before making any commits

2. **Never force push:**
   - Do not use `git push --force` or `git push -f`
   - If a push is rejected, ask the user how to proceed

3. **Check before committing:**
   - Run `git status` before staging changes
   - Ensure you're on the correct branch
   - Verify no unexpected changes are staged

4. **One session at a time:**
   - If you detect uncommitted changes you didn't make, ask the user
   - If remote has commits not in local, pull before continuing

## Project Overview

Pipeboard is a cross-platform clipboard manager written in Go. It supports multiple backends (local filesystem, S3, SSH) for syncing clipboard content across machines.

## Development

- Language: Go 1.21+
- Run tests: `go test -v -race ./...`
- Build: `go build .`
- Lint: `golangci-lint run`

## Code Style

- Follow standard Go conventions
- Keep functions focused and small
- Write tests for new functionality
