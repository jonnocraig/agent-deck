# Agent Deck

Terminal session manager for AI coding agents. Go 1.24 TUI app using Bubble Tea with tmux-based sessions.

## CRITICAL: Rebuild Before Manual Testing

**ALWAYS rebuild the binary before manual testing.** The symlink at `/opt/homebrew/bin/agent-deck` points to `build/agent-deck`. If you don't rebuild, you're testing stale code.

```bash
go build -o build/agent-deck ./cmd/agent-deck/
```

Run this command every time before launching `agent-deck` to test changes. No exceptions.

## Build & Test

```bash
# Build
go build -o build/agent-deck ./cmd/agent-deck/

# Run all tests
go test ./... -count=1 -short

# Run tests for a specific package
go test ./internal/vagrant/ -count=1 -short

# Vet
go vet ./...
```

## Project Structure

- `cmd/agent-deck/` - CLI entrypoint
- `internal/session/` - Session management, tmux integration
- `internal/vagrant/` - Vagrant VM provider (isolation mode)
- `internal/ui/` - Bubble Tea TUI components

## Vagrant Mode

Vagrant sessions run Claude Code inside an isolated VM with `--dangerously-skip-permissions`. Key files:

- `internal/vagrant/sync.go` - Config sync (strips host-only fields, injects VM fields, syncs OAuth)
- `internal/vagrant/oauth.go` - OAuth credential extraction (env var > macOS Keychain > credentials file)
- `internal/vagrant/mcp.go` - MCP config generation for VM (STDIO, no pool sockets)
- `internal/vagrant/vagrantfile.go` - Vagrantfile generation and provisioning

Order matters in `SyncClaudeConfig()`: `stripHostOnlyFields()` runs before `injectVMFields()`.

To test Vagrant changes from scratch: delete `Vagrantfile` + `rm -rf .vagrant/`, then `vagrant up`.
