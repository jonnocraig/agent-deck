# Session Handoff - 2026-02-22 (Session 6)

## What Was Accomplished This Session

1. **Committed 5 sessions of work** to git as `355417e` on `main` (13 files, 811 insertions)
   - OAuth forwarding, skill directory fix, CLAUDE.md pre-loading, onboarding bypass, env var forwarding
2. **Brainstormed user profile sync** with 4-perspective agent team (Architect, Implementer, Devil's Advocate, Security Analyst)
3. **Wrote design document**: `.claude/plans/2026-02-22-user-profile-sync-design.md`
   - Approach: Go `archive/tar` with security hardening
   - In-memory tar, stdin pipe to `vagrant ssh`, single SSH call
   - Symlink validation (restrict to ~/.claude/ and ~/.agents/)
   - CLAUDE.md merge (VM context first, user content appended)
   - 100MB size limit, .DS_Store exclusion, operating-in-vagrant/ exclusion
   - New file: `sync_profile.go` (~150 lines) + `sync_profile_test.go` (~300 lines)

## Current State

- **Branch**: `main` (1 commit ahead of origin: `355417e`)
- **All previous work committed** — clean working tree (except plan files and untracked dirs)
- **Design complete, implementation NOT started**
- **Tests**: All pass (`go test ./internal/vagrant/ -count=1 -short`)

## Important Context

- Design doc is the source of truth: `.claude/plans/2026-02-22-user-profile-sync-design.md`
- The old plan reference (`snug-marinating-teapot.md`) no longer exists — replaced by the design doc
- `SyncClaudeConfig()` currently has 6 steps; profile sync adds step 6 modification + step 7
- Step 6 change: `getVMClaudeMD()` → `getMergedVMClaudeMD()` (append host CLAUDE.md)
- Step 7 new: `syncProfileToVM()` using `createProfileTar()` + stdin pipe
- Package-level injectable: `var createProfileTarFunc = createProfileTar`
- Manager struct needs NO new fields (tar creation is pure function, VM pipe uses existing vagrantCmd)
- Build output MUST go to `build/agent-deck` (homebrew symlink from `/opt/homebrew/bin/`)
- **ALWAYS rebuild before manual testing**: `go build -o build/agent-deck ./cmd/agent-deck/`

## Next Steps (in order)

1. **Create implementation plan** (use `agentic-ai-plan` with the design doc as input)
2. **Implement user profile sync**:
   - Create `internal/vagrant/sync_profile.go` — tar creation, symlink validation, VM sync
   - Create `internal/vagrant/sync_profile_test.go` — 13 test cases
   - Modify `internal/vagrant/sync.go` — step 6 CLAUDE.md merge, step 7 profile sync call
   - Update `internal/vagrant/sync_test.go` — merged CLAUDE.md tests
3. **Rebuild binary and run full test suite**
4. **Commit and push**
5. **Fresh VM test** (`vagrant destroy` + `vagrant up`)
6. **Increase default VM RAM** from 4GB to 16GB (`vagrantfile.go:162`, `settings.MemoryMB` default)
7. Continue `feat/kanban` work in worktree

## Commands to Run First

```bash
go test ./internal/vagrant/ -count=1 -short       # verify tests still pass
go build -o build/agent-deck ./cmd/agent-deck/     # rebuild binary
```
