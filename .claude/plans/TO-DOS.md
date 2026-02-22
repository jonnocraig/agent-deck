# Current Tasks

## In Progress

- [ ] **Implement user profile sync to Vagrant VM** (plan: `.claude/plans/snug-marinating-teapot.md`)
  - [ ] Add `extractTarInVMFunc` field to Manager struct (`manager.go`)
  - [ ] Create `internal/vagrant/sync_batch.go` — tar infrastructure with symlink dereferencing
  - [ ] Create `internal/vagrant/sync_batch_test.go` — tar tests (7 test cases)
  - [ ] Modify `sync.go` step 6: merge user CLAUDE.md with VM context
  - [ ] Add steps 7-10 to `sync.go`: sync skills/commands/agents/rules via combined tar
  - [ ] Add sync_test.go tests for CLAUDE.md merge and profile sync
  - [ ] Rebuild binary and run full test suite

## Completed This Session (2026-02-22 — Bug fixes + profile sync planning)

- [x] Fixed stale binary bug — OAuth code not compiled. Added CLAUDE.md rebuild rule
- [x] Fixed skill discovery — changed from standalone `.md` to directory `SKILL.md` format
- [x] Fixed skill pre-loading — added `~/.claude/CLAUDE.md` sync to VM via `SyncClaudeConfig()`
- [x] Created project CLAUDE.md with rebuild-before-testing rule
- [x] Added legacy skill file cleanup (vagrant-sudo.md, operating-in-vagrant.md)
- [x] Added `TestEnsureSudoSkillCleansUpLegacyFiles` test
- [x] Added `TestGetVMClaudeMD` and `TestSyncClaudeConfigWritesClaudeMD` tests
- [x] Created and approved plan for user profile sync (`.claude/plans/snug-marinating-teapot.md`)

## Completed Previous Sessions

- [x] OAuth credential forwarding (oauth.go, sync step 5, CLAUDE_CODE_OAUTH_TOKEN)
- [x] Config stripping (stripHostOnlyFields, stripSettingsForVM, stripJSONKeys)
- [x] PATH provisioning (~/.local/bin, claude symlink)
- [x] Skill rewrite (operating-in-vagrant)
- [x] MCP in VM (stripMCPServers, WriteMCPJsonForVagrant)
- [x] buildVagrantClaudeCommand (tmux-free)
- [x] [Vagrant] badge
- [x] Full vagrant mode implementation (33 files, 8051 lines)

## Pending

- [ ] **Commit ALL changes to git** (5 sessions of uncommitted work on main)
- [ ] Delete Vagrantfile + `.vagrant/` for fresh VM test
- [ ] Test full profile sync inside VM (skills, commands, agents, rules visible)
- [ ] Test VM suspend on session stop, destroy on session delete
- [ ] Test multi-session VM sharing
- [ ] Upgrade Node.js in VM from 18.x to 20.x
- [ ] Continue `feat/kanban` work in worktree

## Blocked

- None
