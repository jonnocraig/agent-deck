# Session Handoff - 2026-02-27 (Session 7 — Kanban Mode Design)

## What Was Accomplished This Session

1. **Full kanban mode design brainstorm** with 5-perspective agent team
   - Architect: Data model, component decomposition, message flow, 3-tier config
   - Implementer: 7.5:1 reuse ratio, ~1065 new lines, lipgloss layout approach
   - Devil's Advocate: Terminal width concerns, scope risk, simpler alternatives
   - User Advocate: Navigation flow, progressive disclosure, error states, keyboard design
   - Skill Designer: 4 skill designs with frontmatter, checklists, reference files

2. **Cloned and analyzed 3 reference repositories**
   - `agtx` (git@github.com:fynnfluegge/agtx.git) — Rust kanban for coding agents, 5-column board, plugin system
   - `vibe-kanban` (git@github.com:BloopAI/vibe-kanban.git) — Rust+React rich kanban, sub-issues, MCP server
   - `claude-vibekanban` (git@github.com:ericblue/claude-vibekanban.git) — 17 slash commands, epic-based workflow

3. **Read and integrated best practices**
   - Anthropic skill best practices (platform.claude.com)
   - Zen/PAL MCP consensus best practices (blinded consensus, file paths, continuation IDs)

4. **Design decisions validated with user**
   - Groups on left sidebar, kanban board on right
   - 6 columns always visible (no horizontal scroll)
   - Enter=attach to tmux, Space=detail panel, e=edit mode
   - Auto-trigger skills on column move (configurable confirm vs auto per session)
   - Review skill uses agent team for screenshots (never in coordinator context)
   - Review skill uses Claude Chrome extension for ALL work types
   - YOLO mode: Conductor agent with zen consensus gates
   - self-evolve: markdown files in ~/.claude/learned/ with confidence scoring
   - Done: auto-detect PR vs direct merge, safety protocol

5. **Wrote and committed design document** (97a87a4, 737 lines)
   - `docs/plans/2026-02-27-kanban-mode-design.md`

## Current State

- **Branch**: `feature/KanbanMode` (worktree at `.worktrees/feature-KanbanMode`)
- **Last commit**: `97a87a4` — `docs: add kanban mode design document`
- **Design complete, implementation NOT started**
- **Tests**: All pass (`go test ./... -count=1 -short`)
- **Working tree**: Clean (design doc committed)

## Important Context

- **Design doc is the source of truth**: `docs/plans/2026-02-27-kanban-mode-design.md`
- **Reference repos cloned to /tmp/**: agtx, vibe-kanban, claude-vibekanban, pal-mcp-server (may need re-cloning after restart)
- **home.go is 9031 lines** — needs decomposition into kanban_board.go, kanban_card.go, kanban_sidebar.go, kanban_detail.go, kanban_nav.go, kanban_transition.go, kanban_conductor.go
- **KanbanColumn is separate from Status** — workflow stage != session lifecycle
- **3-tier config**: per-group SQLite > user YAML (~/.config/agent-deck/kanban.yaml) > hardcoded defaults
- **Sort order**: `1000 * column_index + position * 100` (from vibe-kanban)
- **YOLO consensus gates**: Use `mcp__zen__consensus` with blinded evaluation, file-based context, continuation IDs
- **Review agent team**: static-checks (haiku), chrome-visual (sonnet), e2e-tests (sonnet), code-review (sonnet)
- **Build output**: `go build -o build/agent-deck ./cmd/agent-deck/` (symlink from /opt/homebrew/bin/)
- **ALWAYS rebuild before manual testing**

## Next Steps (in order)

1. **Create implementation plan** — use `agentic-ai-plan` with `docs/plans/2026-02-27-kanban-mode-design.md` as input
2. **Phase 1: Data Layer** — SQLite migration, add KanbanColumn/Description/AcceptCriteria/AutomationMode to Instance, GroupKanbanConfig table
3. **Phase 2: Board UI** — KanbanBoard, KanbanCard, KanbanSidebar components
4. **Phase 3: Navigation** — KanbanNav 2D cursor, Tab focus, h/j/k/l/1-6 keys
5. **Phase 4: Detail Panel** — KanbanDetail with editable fields, Space toggle
6. **Phase 5: Transitions** — TransitionEngine, skill triggers, 3-tier config, rollback
7. **Phase 6: Conductor** — YOLO mode, zen consensus gates, conductor lifecycle
8. **Phase 7: Skills** — Create 4 new skills: agentic-ai-backlog, agentic-ai-review, agentic-ai-done, self-evolve

## Commands to Run First

```bash
go test ./... -count=1 -short                          # verify tests pass
go build -o build/agent-deck ./cmd/agent-deck/         # rebuild binary
cat docs/plans/2026-02-27-kanban-mode-design.md        # review design doc
```
