# Agent Deck

Terminal session manager for AI coding agents (Claude Code, Gemini CLI, OpenCode). Go 1.24 TUI built with Bubble Tea + tmux.

## Commands

```bash
go build ./...                    # Build
go test -race ./...               # Test all (with race detector)
go test ./internal/session/       # Test single package
go vet ./...                      # Vet
make build                        # Build binary to ./build/agent-deck
make lint                         # golangci-lint
make ci                           # Pre-push checks (lint + test + build)
```

## Structure

```
cmd/agent-deck/     Entry point, CLI commands (cobra-style subcommands)
internal/
  session/          Session & group CRUD, tool detection, vagrant integration
  ui/               Bubble Tea TUI components (home, list, dialogs, styles)
  tmux/             tmux session control, status detection
  vagrant/          VM lifecycle, provisioning, sync, MCP tunneling
  mcppool/          MCP unix socket pooling
  git/              Git worktree operations
  statedb/          SQLite state persistence
  logging/          Structured logging (slog + lumberjack)
  platform/         OS-specific operations (macOS/Linux)
  profile/          Profile management
  clipboard/        Clipboard operations
  update/           Auto-update mechanism
  experiments/      Feature flags
conductor/          Background daemon scripts & templates
skills/             Claude Code skill documentation
```

## Conventions

- **Concurrency**: `sync.RWMutex` for shared state, getter/setter methods (`GetStatus()`/`SetStatus()`), `atomic.Bool` for flags
- **Errors**: Wrap with context (`fmt.Errorf("context: %w", err)`), structured logging via slog
- **Logging**: Component-based (`logging.ForComponent(logging.CompSession)`), fields via `slog.String()`
- **Config**: TOML at `~/.agent-deck/config.toml`, profiles at `~/.agent-deck/profiles/<name>/`
- **Types**: Status as typed constants (`type Status string`), interfaces for abstractions
- **Tests**: `_test.go` alongside source, `testify` for assertions, `skipIfNoTmuxServer(t)` for integration tests

## Architecture Notes

- **Bridge adapter pattern** (vagrant): `session/vagrant_iface.go` defines `VagrantVM` interface, `vagrant/bridge.go` implements it. Breaks `session -> vagrant -> session` import cycle. Never import `vagrant` from `session` directly.
- **MCP pool**: Unix socket multiplexing reduces memory ~85-90% for shared MCP servers across sessions.
- **State**: SQLite via `modernc.org/sqlite` (pure Go, no CGO). Sessions and groups persisted.
