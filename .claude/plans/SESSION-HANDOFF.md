# Session Handoff - 2026-02-21 04:30

## What Was Accomplished (This Session — Host-Side)

- Fixed 4 config sync bugs from Vagrant VM testing screenshots (installMethod, plugins, PATH, OAuth)
- Rewrote `operating-in-vagrant` skill following "Supercharged Claude" blog post + Anthropic skill best practices
- Added generic `stripJSONKeys()` helper, `stripHostOnlyFields()`, `stripSettingsForVM()`
- Added Vagrantfile provisioning for `~/.local/bin` + claude symlink + PATH
- Added 6 new tests, updated all skill/e2e tests
- All vagrant tests pass (81s suite), binary builds clean

## What Was Accomplished (Previous VM Session)

- Created `vagrant-vm-setup` skill (SKILL.md + MCP-SETUP.md + TROUBLESHOOTING.md)
- Installed MCP packages globally in VM (memory, sequential-thinking, filesystem)
- Configured `/vagrant/.mcp.json` with working STDIO-based MCP servers
- Created worktree at `/vagrant/worktrees/feat-kanban` on `feat/kanban` branch

## Files Modified This Session (Host-Side)

| File | Change |
|------|--------|
| `internal/vagrant/sync.go` | Added `stripHostOnlyFields()`, `stripSettingsForVM()`, refactored to `stripJSONKeys()` |
| `internal/vagrant/sync_test.go` | Added 6 new tests for config stripping functions |
| `internal/vagrant/vagrantfile.go` | Added `~/.local/bin` creation, claude symlink, PATH setup to provisioning |
| `internal/vagrant/skill.go` | Rewrote skill as `operating-in-vagrant` with capabilities-first mindset |
| `internal/vagrant/skill_test.go` | Updated tests for new skill content, filename, frontmatter |
| `internal/vagrant/e2e_test.go` | Updated skill filename and name assertions |

## Also Modified from Earlier Sessions (Still Uncommitted)

| File | Change |
|------|--------|
| `cmd/agent-deck/main.go` | Blank import for vagrant provider |
| `internal/session/instance.go` | `buildVagrantClaudeCommand()` |
| `internal/ui/claudeoptions.go` | Claude options changes |
| `internal/ui/home.go` | `[Vagrant]` badge |
| `internal/ui/styles.go` | `ColorBlue` in theme |

## Current State

- **Branch**: `main` (1 commit ahead of origin, many uncommitted changes)
- **NOTHING IS COMMITTED** — all work from 3 sessions is uncommitted on main
- **Binary**: `build/agent-deck` is up to date with all fixes
- **VM**: Vagrant Ubuntu 24.04, has MCP packages installed, running
- **Skill renamed**: `vagrant-sudo.md` → `operating-in-vagrant.md`
- **Two skills exist**: `operating-in-vagrant` (embedded in Go, written to project) + `vagrant-vm-setup` (created inside VM at `.claude/skills/`)

## Important Context

- Build output MUST go to `build/agent-deck` (homebrew symlink from `/opt/homebrew/bin/`)
- `EnsureVagrantfile()` skips if Vagrantfile exists — **DELETE IT** to regenerate with new provisioning
- Existing Vagrantfile + `.vagrant/` in repo root are from testing — not committed
- `stripJSONKeys()` is the generic helper; `stripMCPServers()` still exists wrapping it
- MCP packages are `@modelcontextprotocol/server-*` (NOT `@anthropic-ai/mcp-*`)
- Pool socket MCPs don't work in VM — always use STDIO
- Node.js 18.x in VM despite some packages wanting 20+
- Tokyo Night theme: `#2ac3de` for blue (dark mode)
- Agent-deck is a Go 1.24 TUI app using Bubble Tea, sessions are tmux-based

## Next Steps (in order)

1. **Commit all host-side fixes** — many modified files on `main`
2. Delete `Vagrantfile` + `rm -rf .vagrant/` for fresh test
3. Test `vagrant up` from scratch → verify config stripping + PATH fixes
4. Verify `operating-in-vagrant.md` skill loads inside VM
5. Test VM lifecycle (suspend on stop, destroy on delete)
6. Continue `feat/kanban` work in worktree

## Commands to Run First

```bash
go build -o build/agent-deck ./cmd/agent-deck/   # rebuild binary
go test ./internal/vagrant/ -v -count=1            # run vagrant tests
rm Vagrantfile && rm -rf .vagrant/                 # force fresh VM
```
