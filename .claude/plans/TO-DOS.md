# Current Tasks

## In Progress

- [ ] **Kanban Mode Feature** (design: `docs/plans/2026-02-27-kanban-mode-design.md`)
  - [x] Brainstorm with 5-perspective agent team (Architect, Implementer, Devil's Advocate, User Advocate, Skill Designer)
  - [x] Clone and analyze 3 reference repos (agtx, vibe-kanban, claude-vibekanban)
  - [x] Read Anthropic skill best practices
  - [x] Read zen/PAL MCP consensus best practices
  - [x] Write full design document (737 lines)
  - [x] Commit design document
  - [ ] Create implementation plan (`agentic-ai-plan`)
  - [ ] **Phase 1: Data Layer** — SQLite migration, Instance fields, GroupKanbanConfig
  - [ ] **Phase 2: Board UI** — KanbanBoard, KanbanCard, KanbanSidebar rendering
  - [ ] **Phase 3: Navigation** — KanbanNav 2D cursor, Tab focus, column jump, scroll
  - [ ] **Phase 4: Detail Panel** — KanbanDetail editable fields, Space toggle, edit mode
  - [ ] **Phase 5: Transitions** — TransitionEngine, skill triggers, 3-tier config, rollback
  - [ ] **Phase 6: Conductor** — YOLO mode, zen consensus gates, conductor lifecycle
  - [ ] **Phase 7: Skills** — 4 new skills (backlog, review, done, self-evolve)

## Completed This Session (2026-02-27 — Kanban Mode Design Brainstorm)

- [x] Cloned and analyzed agtx (Rust kanban for coding agents) — board layout, task model, plugin system
- [x] Cloned and analyzed vibe-kanban (Rust+React rich kanban) — sub-issues, priorities, tags, workspaces
- [x] Cloned and analyzed claude-vibekanban (Claude commands) — epic workflow, 17 commands, parallel execution
- [x] Read Anthropic skill best practices (platform.claude.com)
- [x] Read zen/PAL MCP consensus best practices (blinded consensus, file paths, continuation IDs)
- [x] Explored agent-deck codebase (245 Go files, 9031-line home.go, session model, group system)
- [x] Explored existing agentic-ai skills (brainstorm, plan, implement — patterns, conventions)
- [x] Asked 4 rounds of clarifying questions (view mode, auto-trigger, learning store, merge strategy)
- [x] Ran 5 parallel exploration agents (Architect, Implementer, Devil's Advocate, User Advocate, Skill Designer)
- [x] Synthesized findings into 3 approaches, recommended Approach 1 (Full Kanban)
- [x] Validated 7 design sections with user (architecture, components, API, errors, testing, skills, automation)
- [x] Revised: Enter=attach, Space=detail panel, Claude Chrome in review, agent team for screenshots
- [x] Added YOLO mode with Conductor agent and zen consensus gates
- [x] Added zen MCP best practices (blinded consensus, file paths, continuation IDs, escalation)
- [x] Wrote and committed design document (97a87a4)

## Completed Previous Sessions

- [x] User profile sync design (brainstorm + design doc)
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

- [ ] **Implement user profile sync** (design: `.claude/plans/2026-02-22-user-profile-sync-design.md`)
- [ ] Delete Vagrantfile + `.vagrant/` for fresh VM test
- [ ] Increase default VM RAM from 4GB to 16GB (`vagrantfile.go:162`)
- [ ] Upgrade Node.js in VM from 18.x to 20.x

## Blocked

- None
