# Session Handoff - 2026-02-27 (Session 12 — JSON Plan Sync, Wave 3 Next)

## What Was Accomplished This Session

1. **Synced JSON agent plan with MD** (6295382)
   - JSON was 24 tasks across 2 phases; MD had 28 tasks across 8 phases
   - Added missing tasks: 1.4 (Bubble Tea messages), 1.5 (sort order), 3.3 (n/d create/delete), 3.4 (m move card)
   - Added missing phases: 2-7 (Board UI through Skills)
   - Updated progress tracking: 8 tasks complete (waves 1-2), 20 pending
   - JSON is now the preferred source of truth for `agentic-ai-implement`

2. **Previous session (11): Executed Wave 2: Data Layer** (5 tasks, all complete)
   - Task 1.1 || 1.2 (parallel): KanbanColumn type + SQLite schema v2 migrations
   - Task 1.3 || 1.4 (parallel): CRUD methods + Bubble Tea message types
   - Task 1.5 (sequential): Sort order calculation + rebalancing
   - All committed as 9b7ea91 (+3,027 lines across 11 files)

2. **Wave 2 deliverables:**

   | File | Change | Content |
   |------|--------|---------|
   | instance.go | Modified | KanbanColumn type (6 values), AutomationMode, YOLOConfig, 7 new Instance fields |
   | instance_test.go | Modified | 7 type tests |
   | storage.go | Modified | 7 CRUD methods, GroupKanbanConfig, ColumnSkillMapping, BatchUpdateSortOrders |
   | storage_test.go | Modified | 17 CRUD tests |
   | statedb.go | Modified | Schema v2 (7 ALTER TABLEs, 2 new tables), InstanceRow fields, CRUD |
   | statedb_test.go | Modified | 8 migration/CRUD tests |
   | kanban_transition.go | Modified | Updated to use session.KanbanColumn types |
   | kanban_sort.go | Created | Sort order calc: `1000*col + pos*100`, Rebalance(), NextSortOrder() |
   | kanban_sort_test.go | Created | 8 sort order tests |
   | kanban_messages.go | Created | 22 Bubble Tea messages, FocusPanel enum, ErrorAction struct |
   | kanban_messages_test.go | Created | 4 message tests |

## Previous Sessions Summary

- **Session 10**: Wave 1 — home.go decomposition into 7 kanban_*.go files (b80441d)
- **Session 9**: Plan review (24->28 tasks), TDD test cases enhanced
- **Session 8**: Multi-model design review, Phase 0 added, plan enrichment (6 waves)
- **Session 7**: Full kanban mode brainstorm + design document (737 lines)
- **Sessions 1-6**: Vagrant mode implementation, OAuth, config sync, MCP

## Current State

- **Branch**: `feature/KanbanMode` (worktree at `.worktrees/feature-KanbanMode`)
- **Last commit**: `6295382` — `docs: sync JSON agent plan with MD — add missing tasks and phases`
- **Ahead of origin by 3 commits** (need to push)
- **Tests**: All pass (`go test ./... -count=1 -short` — 16/16 packages)
- **Working tree**: Clean
- **Plan**: 28 tasks, 6 waves — Waves 1-2 done, Waves 3-6 pending

## Important Context

- **Design doc**: `docs/plans/2026-02-27-kanban-mode-design.md` (source of truth)
- **Agent plan (MD)**: `.claude/plans/agent-teams/2026-02-27-kanban-mode-agent-plan.md` (28 tasks, 6 waves)
- **Agent plan (JSON)**: `.claude/plans/agent-teams/2026-02-27-kanban-mode-agent-plan.json` (**preferred source of truth** — fully synced with MD)
- **User chose Opus for all tasks** (not optimizing for cost)
- **Build output**: `go build -o build/agent-deck ./cmd/agent-deck/` (ALWAYS rebuild before testing)
- **statedb uses StateDB.Migrate()** with schema versioning — Wave 2 added schema v2
- **KanbanColumn type**: 6 values defined in `internal/session/instance.go` — Backlog, Design, Plan, Implement, Review, Done
- **22 Bubble Tea messages** defined in `internal/ui/kanban_messages.go` — BoardRefreshMsg, FocusChangedMsg, ColumnChangedMsg, etc.
- **Sort order formula**: `1000 * column_index + position * 100` with Rebalance() for gap closure
- **kanban_board.go** (654 lines) has existing layout/rendering functions extracted from home.go — Wave 3 MODIFIES these
- **kanban_card.go** (267 lines) has existing session card rendering — Wave 3 MODIFIES
- **kanban_sidebar.go** (341 lines) has existing group rendering — Wave 3 MODIFIES
- **home.go** (6038 lines) — Wave 3 Task 2.4 adds kanban routing

## Wave Execution Summary

| Wave | Name | Tasks | Status | Commit |
|------|------|-------|--------|--------|
| 1 | Foundation | 3 | COMPLETE | b80441d |
| 2 | Data Layer | 5 | COMPLETE | 9b7ea91 |
| 3 | Board UI | 4 | **NEXT** | — |
| 4 | Nav & Detail | 6 | PENDING | — |
| 5 | Transitions | 3 | PENDING | — |
| 6 | Automation | 7 | PENDING | — |

## Next Steps (in order)

1. **Execute Wave 3: Board UI** — use `agentic-ai-implement` skill
   - Tasks 2.1 || 2.2 || 2.3 (parallel): KanbanBoard layout, KanbanCard rendering, KanbanSidebar group list
   - Task 2.4 (blocked by 2.1-2.3): Integrate kanban into home.go routing
2. **Commit Wave 3** — commit after all 4 tasks pass quality gates
3. **Execute Wave 4: Nav & Detail** — keyboard navigation + detail panel
4. Continue waves 5-6 per the plan

## Wave 3 Task Details

### Task 2.1: KanbanBoard component (kanban_board.go)
- 6-column layout rendering, column headers with card counts
- 80-col degradation: hide sidebar, show 3 columns with scroll indicators
- Tests: board_120cols, board_160cols, board_80cols, column_headers, empty_board, cards_in_columns

### Task 2.2: KanbanCard component (kanban_card.go)
- Compact card rendering: status icon + truncated title
- Status icons: running=`bullet`, waiting=`half-circle`, idle=`circle`, error=`cross`
- Selected/unselected styling via lipgloss
- Tests: compact, status_icons, selected, unselected, title_truncation

### Task 2.3: KanbanSidebar component (kanban_sidebar.go)
- Group list with kanban toggle indicator "[K]"
- j/k navigation within sidebar (only when focused)
- Fixed 20-column width
- Tests: groups, kanban_indicator, fixed_width, all_sessions_first, jk_nav

### Task 2.4: Integrate kanban into home.go (home.go)
- Route to kanban view when selected group has kanban enabled
- "All Sessions" always renders flat list (never kanban)
- K key toggles kanban for selected group
- Wire FocusChangedMsg, ColumnChangedMsg, CardChangedMsg to components

## Commands to Run First

```bash
go test ./... -count=1 -short                          # verify tests pass
go build -o build/agent-deck ./cmd/agent-deck/         # rebuild binary
wc -l internal/ui/kanban_*.go                          # check kanban file sizes
cat .claude/plans/agent-teams/2026-02-27-kanban-mode-agent-plan.md  # review plan
```
