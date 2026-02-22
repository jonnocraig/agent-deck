# Current Tasks

## In Progress

- [ ] End-to-end manual testing of Vagrant mode feature (needs fresh `vagrant destroy` + `vagrant up`)
- [ ] Kanban feature on `feat/kanban` branch (worktree at `/vagrant/worktrees/feat-kanban`)

## Completed This Session (2026-02-21 session 3 — host-side)

- [x] Fixed: `installMethod is native` errors — added `stripHostOnlyFields()` to strip `installMethod`, `oauthAccount`
- [x] Fixed: `22 plugins failed to install` — added `stripSettingsForVM()` to strip `enabledPlugins`, `hooks`
- [x] Fixed: `~/.local/bin` not in PATH — added provisioning to create dir, symlink claude, update PATH
- [x] Refactored `stripMCPServers()` → generic `stripJSONKeys()` helper
- [x] Enhanced skill: `operating-in-vagrant` (was `vagrant-sudo`) — "Supercharged Claude" mindset, capabilities-first, host networking, Docker patterns
- [x] Added 6 new tests for config stripping
- [x] Updated all skill tests and e2e test for renamed skill file

## Completed Previous Sessions

- [x] Diagnosed and fixed MCP tools not loading inside VM (VM session)
- [x] Created `vagrant-vm-setup` skill (SKILL.md + MCP-SETUP.md + TROUBLESHOOTING.md) (VM session)
- [x] Installed MCP packages globally in VM, configured `.mcp.json` (VM session)
- [x] Created worktree at `/vagrant/worktrees/feat-kanban` (VM session)
- [x] Fixed: `internal/vagrant` package never imported — blank import in `main.go`
- [x] Fixed: tmux commands leaking into vagrant SSH — `buildVagrantClaudeCommand()`
- [x] Fixed: Build output going to wrong location
- [x] Fixed: settings.json and statusline.sh not synced into VM
- [x] Fixed: Host MCP servers failing inside VM — `stripMCPServers()`
- [x] Added `[Vagrant]` badge to session list and preview pane
- [x] Design document and full vagrant mode implementation (33 files, 8051 lines)

## Pending

- [ ] **Commit all host-side fixes to git** (many modified files, uncommitted on main)
- [ ] Delete existing Vagrantfile + `.vagrant/` for fresh VM test with new provisioning
- [ ] Test with `vagrant up` from scratch to verify config stripping + PATH fixes
- [ ] Verify `operating-in-vagrant.md` skill loads inside VM
- [ ] Restart Claude Code inside VM to verify MCP servers load from `.mcp.json`
- [ ] Test VM suspend on session stop, destroy on session delete
- [ ] Test multi-session VM sharing
- [ ] Consider adding MCP npm packages to Vagrantfile provisioning script
- [ ] Upgrade Node.js in VM from 18.x to 20.x (some MCP packages want 20+)

## Blocked

- None
