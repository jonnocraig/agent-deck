# Session Handoff - 2026-02-22 (Session 5)

## What Was Accomplished This Session

### Bug Fixes (manual testing of Vagrant mode)
1. **Stale binary bug**: OAuth code wasn't compiled into running binary. Root cause: `build/agent-deck` wasn't rebuilt after adding `oauth.go`. Added CLAUDE.md rule: always rebuild before testing.
2. **Skill not discovered**: `EnsureSudoSkill()` wrote `operating-in-vagrant.md` as standalone file. Claude Code requires skills as directories with `SKILL.md`. Fixed to write `operating-in-vagrant/SKILL.md`. Added legacy cleanup for `vagrant-sudo.md` and `operating-in-vagrant.md`.
3. **Skill not pre-loaded**: Claude Code lazy-loads skills (descriptions only in context). Added step 6 to `SyncClaudeConfig()` that writes `~/.claude/CLAUDE.md` inside the VM with the operating-in-vagrant skill body. CLAUDE.md is always loaded in full at session start.

### New Files Created
- `CLAUDE.md` — project-level instructions with critical "rebuild before testing" rule

### Plan Created (Not Yet Implemented)
- **User Profile Sync**: `/Users/jon_ec/.claude/plans/snug-marinating-teapot.md`
- Goal: sync user's full Claude Code profile (skills, commands, agents, rules, CLAUDE.md) to the VM
- Approach: batch tar transfer with symlink dereferencing, single SSH call for all 4 directories
- Status: **Plan approved, implementation NOT started**

## Files Changed This Session (Uncommitted)

| File | Type | Description |
|------|------|-------------|
| `CLAUDE.md` | NEW | Project instructions with rebuild-before-test rule |
| `internal/vagrant/skill.go` | MODIFIED | Write skill as directory `operating-in-vagrant/SKILL.md`, cleanup legacy files |
| `internal/vagrant/skill_test.go` | MODIFIED | Updated paths to `operating-in-vagrant/SKILL.md`, added legacy cleanup test |
| `internal/vagrant/e2e_test.go` | MODIFIED | Updated skill path assertion |
| `internal/vagrant/sync.go` | MODIFIED | Added `getVMClaudeMD()`, step 6 writes `~/.claude/CLAUDE.md` to VM |
| `internal/vagrant/sync_test.go` | MODIFIED | Added `TestGetVMClaudeMD`, `TestSyncClaudeConfigWritesClaudeMD` |

## Also Uncommitted from Previous Sessions (4 sessions of work)

| File | Change |
|------|--------|
| `internal/vagrant/oauth.go` | NEW — OAuth credential extraction |
| `internal/vagrant/oauth_test.go` | NEW — 6 OAuth tests |
| `internal/vagrant/sync.go` | `stripHostOnlyFields()`, `injectVMFields()`, OAuth sync, CLAUDE.md sync |
| `internal/vagrant/sync_test.go` | OAuth, onboarding, config stripping, CLAUDE.md tests |
| `internal/vagrant/mcp.go` | `CLAUDE_CODE_OAUTH_TOKEN` env var |
| `internal/vagrant/mcp_test.go` | Updated env var test cases |
| `internal/vagrant/vagrantfile.go` | PATH provisioning |
| `internal/vagrant/skill.go` | Full skill rewrite + directory format |
| `cmd/agent-deck/main.go` | Blank import for vagrant |
| `internal/session/instance.go` | `buildVagrantClaudeCommand()` |
| `internal/ui/home.go` | `[Vagrant]` badge |
| `internal/ui/styles.go` | `ColorBlue` |

## Current State

- **Branch**: `main` (5 sessions of uncommitted work)
- **NOTHING IS COMMITTED** — all work is uncommitted on main
- **Binary**: `build/agent-deck` rebuilt with all changes (skill dir fix + CLAUDE.md sync)
- **Tests**: All pass (`go test ./internal/vagrant/ -count=1 -short`)
- **Manual test results**: OAuth works, skill discovered, CLAUDE.md pending retest

## Important Context

- `EnsureSudoSkill()` now writes `operating-in-vagrant/SKILL.md` (directory format) and cleans up legacy standalone `.md` files
- `getVMClaudeMD()` strips YAML frontmatter from `GetVagrantSudoSkill()` and returns just the body for CLAUDE.md
- Step 6 of `SyncClaudeConfig()` writes `~/.claude/CLAUDE.md` to VM — this is a user-level file that Claude Code always loads on session start
- Build output MUST go to `build/agent-deck` (homebrew symlink from `/opt/homebrew/bin/`)
- **ALWAYS rebuild before manual testing**: `go build -o build/agent-deck ./cmd/agent-deck/`
- Plan for user profile sync is at `/Users/jon_ec/.claude/plans/snug-marinating-teapot.md`

## Next Steps (in order)

1. **Implement user profile sync** (plan at `.claude/plans/snug-marinating-teapot.md`):
   - Create `internal/vagrant/sync_batch.go` — tar infrastructure with symlink dereferencing
   - Create `internal/vagrant/sync_batch_test.go` — tar tests
   - Add `extractTarInVMFunc` to Manager struct in `manager.go`
   - Modify `sync.go` step 6 to merge user CLAUDE.md with VM context
   - Add steps 7-10: sync skills/commands/agents/rules via combined tar
   - Add sync tests
2. **Commit ALL changes to git** (5 sessions of uncommitted work)
3. Fresh VM test with `vagrant destroy` + `vagrant up`
4. Verify full profile sync works inside VM
5. Continue `feat/kanban` work in worktree

## Commands to Run First

```bash
go test ./internal/vagrant/ -count=1 -short       # verify tests still pass
go build -o build/agent-deck ./cmd/agent-deck/     # rebuild binary
```
