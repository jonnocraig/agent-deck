# Current Tasks

## In Progress

- [ ] **Implement user profile sync to Vagrant VM** (design: `.claude/plans/2026-02-22-user-profile-sync-design.md`)
  - [ ] Create implementation plan (`agentic-ai-plan`)
  - [ ] Create `internal/vagrant/sync_profile.go` — tar creation, symlink validation, VM sync
  - [ ] Create `internal/vagrant/sync_profile_test.go` — 13 test cases
  - [ ] Modify `sync.go` step 6: `getVMClaudeMD()` → `getMergedVMClaudeMD()` (append host CLAUDE.md)
  - [ ] Add step 7 to `sync.go`: call `syncProfileToVM()`
  - [ ] Update `sync_test.go` for CLAUDE.md merge + profile sync call
  - [ ] Rebuild binary and run full test suite

## Completed This Session (2026-02-22 — Commit + design brainstorm)

- [x] Committed 5 sessions of work as `355417e` (13 files, 811 insertions)
- [x] Brainstormed profile sync with 4-agent team (Architect, Implementer, Devil's Advocate, Security)
- [x] Wrote design document: `.claude/plans/2026-02-22-user-profile-sync-design.md`
- [x] Validated all design sections with user (architecture, components, errors, testing, security)

## Completed Previous Sessions

- [x] OAuth credential forwarding (oauth.go, sync step 5, CLAUDE_CODE_OAUTH_TOKEN)
- [x] Config stripping (stripHostOnlyFields, stripSettingsForVM, stripJSONKeys)
- [x] PATH provisioning (~/.local/bin, claude symlink)
- [x] Skill rewrite (operating-in-vagrant) + directory format fix
- [x] CLAUDE.md pre-loading in VM
- [x] MCP in VM (stripMCPServers, WriteMCPJsonForVagrant)
- [x] buildVagrantClaudeCommand (tmux-free)
- [x] [Vagrant] badge
- [x] Full vagrant mode implementation (33 files, 8051 lines)
- [x] Project CLAUDE.md with rebuild-before-testing rule

## Pending

- [ ] Delete Vagrantfile + `.vagrant/` for fresh VM test
- [ ] Test full profile sync inside VM (skills, commands, agents, rules visible)
- [ ] Test VM suspend on session stop, destroy on session delete
- [ ] Test multi-session VM sharing
- [ ] **Increase default VM RAM from 4GB to 16GB** (`vagrantfile.go:162`)
- [ ] Upgrade Node.js in VM from 18.x to 20.x
- [ ] Continue `feat/kanban` work in worktree

## Blocked

- None
