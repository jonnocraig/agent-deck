# Session Handoff - 2026-02-14 (Session 3)

## What Was Accomplished

- Multi-model design review of Vagrant Mode design doc using Gemini 3 Pro + GPT 5.1 Codex (GPT 5.2 had connection errors)
- Synthesized 22 issues from three perspectives (mine, Gemini, GPT), prioritized by severity
- Worked through ALL 22 issues sequentially with user, amending design doc for each:
  - #1 SSH reverse tunnels for MCP network binding
  - #2 SSH SendEnv/AcceptEnv for env var transport
  - #3 Goroutine + done channel for async suspend/start race condition
  - #4 SSH agent forwarding for git push
  - #5 Boot phase parser + 100 loading tips (50 Vagrant + 50 world facts, researched by background agents)
  - #6 Two-phase health check with SSH liveness probe
  - #7 Nested virtualization for Docker-in-VM
  - #8 Multi-session prompt: share VM or create separate VM
  - #9 Stale VM warning (3+ threshold) + Shift+D cleanup dialog
  - #10 Health check interval 60s -> 30s
  - #11 Disk space preflight (block <5GB, warn 5-10GB)
  - #12 Combined preflight (Vagrant + VirtualBox version + disk)
  - #13 Port forwarding auto_correct: true
  - #14 provision_packages append semantics + exclude list
  - #15 inotify polling env vars auto-injected + skill guidance
  - #16 Apple Silicon kernel extension stderr detection
  - #17 Auto-hostname from project name
  - #18 Windows documented as unsupported
  - #19 Proxy env var auto-forwarding + docs
  - #20 Credential guard: skill + PreToolUse hook + rsync exclude
  - #21 Config hash + auto re-provision on drift
  - #22 VagrantProvider interface for CI testability
- Design doc grew from ~960 lines to ~2000 lines

## Current State

- Branch: `feature/vagrant` (based on upstream/main v0.16.0)
- Last commit: `47e1d20` — "feat: enrich vagrant mode design doc and generate MCP config on session start"
- Uncommitted changes: design doc + plan files (need to commit)
- Design doc: `docs/plans/2026-02-14-vagrant-mode-design.md` (~2000 lines, fully reviewed)

## Open Issues

- Skeleton repo PR still not created (`git@github.com:jonnocraig/skeleton.git` branch `feature/agentic-ai-skills`)
- No PR created yet for `feature/vagrant`

## Next Steps (in order)

1. Run `/catchup` to restore context in new session
2. Create implementation plan using `agentic-ai-plan` skill
3. Set up git worktree for isolated implementation
4. Execute plan with agent team using `agentic-ai-implement`
5. Create PR for `feature/vagrant` -> upstream
6. Create PR for skeleton repo `feature/agentic-ai-skills` branch

## Important Context

- Design doc is the source of truth: `docs/plans/2026-02-14-vagrant-mode-design.md` (~2000 lines)
- Decisions recorded in `.claude/plans/DECISIONS.md` (12 decisions)
- The feature adds 5 new files: `internal/vagrant/manager.go`, `internal/vagrant/provider.go`, `internal/vagrant/skill.go`, `internal/vagrant/mcp.go`, `internal/vagrant/tips.go`
- The feature modifies 4 files: `claudeoptions.go`, `tooloptions.go`, `instance.go`, `userconfig.go`
- Agent-deck is a Go 1.24 TUI app using Bubble Tea, sessions are tmux-based
- MCP connectivity: SSH reverse tunnels for HTTP MCPs, SendEnv/AcceptEnv for env vars
- `VagrantProvider` interface in `provider.go` — `instance.go` uses interface, not concrete Manager
- `provision_packages` APPENDS to base set (not replaces). `provision_packages_exclude` for removals.
- Credential guard: PreToolUse hook auto-injected for vagrant sessions blocks reading .env, .key, etc.
- Config drift detection: SHA-256 hash triggers auto re-provision on config change
- Polling env vars (CHOKIDAR_USEPOLLING, WATCHPACK_POLLING) auto-injected for VirtualBox sync
- Proxy env vars auto-forwarded from host (forward_proxy_env config toggle)
- Windows is NOT supported in v1
- Multi-session: user prompted to share VM or create separate (VAGRANT_DOTFILE_PATH)
- Port forwards use auto_correct: true to handle collisions
- Upstream is at v0.16.0 with Slack integration (#169) and Claude lifecycle hooks

## Commands to Run First

```bash
# Check branch status
git status
git log --oneline -5

# Read the design doc
cat docs/plans/2026-02-14-vagrant-mode-design.md

# Verify build
go build ./...
go test ./internal/session/ -count=1
```
