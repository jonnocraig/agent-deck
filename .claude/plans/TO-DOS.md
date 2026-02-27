# Current Tasks

## In Progress

- [ ] **Kanban Mode Feature** (design: `docs/plans/2026-02-27-kanban-mode-design.md`)
  - [x] Brainstorm with 5-perspective agent team (Architect, Implementer, Devil's Advocate, User Advocate, Skill Designer)
  - [x] Clone and analyze 3 reference repos (agtx, vibe-kanban, claude-vibekanban)
  - [x] Read Anthropic skill best practices
  - [x] Read zen/PAL MCP consensus best practices
  - [x] Write full design document (737 lines)
  - [x] Commit design document
  - [x] Multi-model design review (Gemini 3.1 Pro advocate, Gemini 2.5 Pro challenger)
  - [x] Adopt Phase 0 refactoring from review feedback
  - [x] Create implementation plan (`agentic-ai-plan`) — 24 tasks, 6 waves
  - [ ] **Phase 0: Refactor** — Decompose home.go (separate PR, zero features, functionally identical)
  - [ ] **Phase 1: Data Layer** — SQLite migration, Instance fields, GroupKanbanConfig
  - [ ] **Phase 2: Board UI** — KanbanBoard, KanbanCard, KanbanSidebar rendering
  - [ ] **Phase 3: Navigation** — KanbanNav 2D cursor, Tab focus, column jump, scroll
  - [ ] **Phase 4: Detail Panel** — KanbanDetail editable fields, Space toggle, edit mode
  - [ ] **Phase 5: Transitions** — TransitionEngine, skill triggers, 3-tier config, rollback
  - [ ] **Phase 6: Conductor** — YOLO mode, zen consensus gates, conductor lifecycle
  - [ ] **Phase 7: Skills** — 4 new skills (backlog, review, done, self-evolve)

## Completed This Session (2026-02-27 — Design Review + Plan Enrichment)

- [x] Ran multi-model design review via zen consensus (Gemini 3.1 Pro + Gemini 2.5 Pro)
- [x] GPT-5.2 quota exhausted — used Gemini 2.5 Pro as challenger instead
- [x] Gemini 3.1 Pro: 8/10 confidence, validated KanbanColumn/Status separation, Elm alignment, component decomposition
- [x] Gemini 2.5 Pro: challenged scope, terminal width, 3-tier config, YOLO reliability, self-evolve value
- [x] User adopted only Phase 0 (home.go refactoring as separate PR), kept all other design elements
- [x] Updated design doc with Phase 0 in implementation phases table
- [x] Updated session handoff and todos with Phase 0
- [x] Ran agentic-ai-plan skill to enrich plan
- [x] Confirmed correct file paths: storage.go (not statedb.go), migration.go, conductor.go already exists
- [x] Generated 24 tasks across 8 phases with agent orchestration metadata
- [x] Assigned 6 execution waves with user checkpoints and quality gates
- [x] Wrote 3 output files (JSON, XML, MD) to .claude/plans/agent-teams/

## Completed Previous Sessions

- [x] Full kanban mode design brainstorm (session 7)
- [x] Design document written and committed (97a87a4)
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
