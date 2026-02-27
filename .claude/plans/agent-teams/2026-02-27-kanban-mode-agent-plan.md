# Kanban Mode — Agent Team Plan

> Source: `docs/plans/2026-02-27-kanban-mode-design.md`
> Generated: 2026-02-27

**Goal:** Add a 6-column kanban board to agent-deck for managing AI coding agent sessions through a development workflow (Backlog → Design → Plan → Implement → Review → Done) with skill automation and autonomous YOLO mode.
**Architecture:** Go 1.24 TUI with Bubble Tea + lipgloss, SQLite persistence, message-driven Elm architecture, 3-tier config, zen MCP consensus gates
**Tech Stack:** Go 1.24, Bubble Tea v1.3.10, lipgloss v1.1.0, SQLite, tmux, zen MCP

---

## Summary

| Metric | Value |
|--------|-------|
| Total Tasks | 24 |
| Total Waves | 6 |
| Avg Parallelism | 4.0x |
| Max Parallelism | 6 |
| Critical Path | 14 tasks |

---

## Wave Execution Strategy

### Wave 1: Foundation — 3 tasks (sequential)

| Task | Name | Model | Agents | Complexity | MCP Tools |
|------|------|-------|--------|------------|-----------|
| 0.1 | Analyze home.go extraction boundaries | opus | architect | high | — |
| 0.2 | Execute home.go decomposition | sonnet | refactor-expert, golang-pro | high | — |
| 0.3 | Verify functional equivalence | haiku | build-error-resolver | low | — |

**Checkpoint:** Review that home.go decomposition is clean — 7 new files created, all tests pass, zero behavioral changes. This is a separate PR before any kanban features.

```
    ┌─────────────────────────────────────────────────────────────────────┐
    │   ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░   │
    │   ░░                                                          ░░   │
    │   ░░    ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓    ░░   │
    │   ░░    ▓▓  ╔═══════════════════════════════════════╗  ▓▓    ░░   │
    │   ░░    ▓▓  ║   W A V E  1 : F O U N D A T I O N   ║  ▓▓    ░░   │
    │   ░░    ▓▓  ║     9031 lines → 7 clean modules     ║  ▓▓    ░░   │
    │   ░░    ▓▓  ╚═══════════════════════════════════════╝  ▓▓    ░░   │
    │   ░░    ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓    ░░   │
    │   ░░               3 tasks  ·  sequential  ·  PR #1          ░░   │
    │   ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░   │
    └─────────────────────────────────────────────────────────────────────┘
```

---

### Wave 2: Data Layer — 3 tasks (1.1 ‖ 1.2, then 1.3)

| Task | Name | Model | Agents | Complexity | MCP Tools |
|------|------|-------|--------|------------|-----------|
| 1.1 | Define KanbanColumn type + Instance fields | sonnet | tdd-guide, golang-pro | medium | — |
| 1.2 | Create SQLite migrations | sonnet | tdd-guide, golang-pro | medium | — |
| 1.3 | Add CRUD methods + GroupKanbanConfig persistence | sonnet | tdd-guide, golang-pro | medium | — |

**Checkpoint:** Review schema design — new Instance fields, migration SQL, CRUD methods. Verify migration runs cleanly on existing databases.

```
    ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░
    ░░                                                                ░░
    ░░     ┌──────┐  ┌──────┐  ┌──────┐  ┌──────┐  ┌──────┐         ░░
    ░░     │██████│  │██████│  │██████│  │██████│  │██████│         ░░
    ░░     │  ID  │──│COLUMN│──│ SORT │──│ DESC │──│  AC  │         ░░
    ░░     │██████│  │██████│  │██████│  │██████│  │██████│         ░░
    ░░     └──────┘  └──────┘  └──────┘  └──────┘  └──────┘         ░░
    ░░            W A V E  2 : D A T A  L A Y E R                    ░░
    ░░         3 tasks  ·  2 parallel + 1  ·  SQLite ready           ░░
    ░░                                                                ░░
    ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░
```

---

### Wave 3: Board UI — 4 tasks (2.1 ‖ 2.2 ‖ 2.3, then 2.4)

| Task | Name | Model | Agents | Complexity | MCP Tools |
|------|------|-------|--------|------------|-----------|
| 2.1 | Implement KanbanBoard component | sonnet | tdd-guide, golang-pro | medium | context7 |
| 2.2 | Implement KanbanCard component | sonnet | tdd-guide, golang-pro | medium | — |
| 2.3 | Implement KanbanSidebar component | sonnet | tdd-guide, golang-pro | medium | — |
| 2.4 | Integrate kanban into home.go routing | sonnet | tdd-guide, golang-pro, code-reviewer | high | — |

**Checkpoint:** Review visual layout — 6-column board, card rendering, sidebar. Build binary and visually verify the board renders correctly at different terminal widths.

```
    ╔══════════════════════════════════════════════════════════════════╗
    ║  ┌────┐ ┌────┐ ┌────┐ ┌────┐ ┌────┐ ┌────┐                   ║
    ║  │ BL │ │ DE │ │ PL │ │ IM │ │ RE │ │ DO │                   ║
    ║  │    │ │    │ │    │ │    │ │    │ │    │                   ║
    ║  │ ▓▓ │ │ ▓▓ │ │    │ │ ▓▓ │ │ ▓▓ │ │    │                   ║
    ║  │ ▓▓ │ │    │ │    │ │ ▓▓ │ │    │ │    │                   ║
    ║  │ ▓▓ │ │    │ │    │ │    │ │    │ │    │                   ║
    ║  └────┘ └────┘ └────┘ └────┘ └────┘ └────┘                   ║
    ║        W A V E  3 : B O A R D  U I                            ║
    ║     4 tasks  ·  3 parallel + 1  ·  kanban visible             ║
    ╚══════════════════════════════════════════════════════════════════╝
```

---

### Wave 4: Navigation & Detail — 4 tasks (3.1 ‖ 4.1, then 3.2 ‖ 4.2)

| Task | Name | Model | Agents | Complexity | MCP Tools |
|------|------|-------|--------|------------|-----------|
| 3.1 | Implement KanbanNav 2D cursor | sonnet | tdd-guide, golang-pro | high | — |
| 4.1 | Implement KanbanDetail panel | sonnet | tdd-guide, golang-pro | medium | — |
| 3.2 | Add vertical scroll within columns | sonnet | tdd-guide, golang-pro | medium | — |
| 4.2 | Implement edit mode for detail panel | sonnet | tdd-guide, golang-pro | medium | — |

**Checkpoint:** Test keyboard navigation end-to-end — h/j/k/l, Tab, 1-6, Enter, Space, e, Esc. Verify detail panel shows all fields correctly.

```
          ┌───────────────────────────────────────────────┐
          │         ◆ ─ ─ ─ ─ ─ ─ ► ◆                    │
          │         │               │                     │
          │    h ◄──┼──► l     j ◄──┼──► k                │
          │         │               │                     │
          │         ▼               ▼                     │
          │    ┌─────────────────────────┐                │
          │    │   Space → Detail Panel  │                │
          │    │   e → Edit   Esc → Back │                │
          │    └─────────────────────────┘                │
          │   W A V E  4 : N A V  &  D E T A I L         │
          │      4 tasks  ·  2×2 parallel  ·  interactive │
          └───────────────────────────────────────────────┘
```

---

### Wave 5: Transitions — 3 tasks (sequential: 5.1 → 5.2 → 5.3)

| Task | Name | Model | Agents | Complexity | MCP Tools |
|------|------|-------|--------|------------|-----------|
| 5.1 | Implement TransitionEngine interface | sonnet | tdd-guide, golang-pro | medium | — |
| 5.2 | Implement 3-tier config resolution | sonnet | tdd-guide, golang-pro | medium | — |
| 5.3 | Implement skill triggers + rollback | sonnet | tdd-guide, golang-pro, code-reviewer | high | — |

**Checkpoint:** Review transition rules, config resolution order (SQLite > YAML > defaults), skill trigger mechanism, and rollback behavior. Test forward and backward column moves.

```
    ░▒▓█▓▒░  ░▒▓█▓▒░  ░▒▓█▓▒░  ░▒▓█▓▒░  ░▒▓█▓▒░  ░▒▓█▓▒░
       │         │         │         │         │         │
       └────►────┘────►────┘────►────┘────►────┘────►────┘
              ↑                                    │
              └────────────── rollback ◄───────────┘
                                                   ✗
         W A V E  5 : T R A N S I T I O N S
         3 tasks  ·  sequential  ·  skills wired
    ░▒▓█▓▒░  ░▒▓█▓▒░  ░▒▓█▓▒░  ░▒▓█▓▒░  ░▒▓█▓▒░  ░▒▓█▓▒░
```

---

### Wave 6: Automation & Skills — 7 tasks (6.1 first, then 6.2 ‖ 6.3 ‖ 7.1 ‖ 7.2 ‖ 7.3 ‖ 7.4)

| Task | Name | Model | Agents | Complexity | MCP Tools |
|------|------|-------|--------|------------|-----------|
| 6.1 | Implement Conductor lifecycle | opus | tdd-guide, golang-pro, architect | high | zen |
| 6.2 | Implement zen consensus gate protocol | opus | tdd-guide, golang-pro | high | zen |
| 6.3 | Implement YOLO UI indicators | sonnet | tdd-guide, golang-pro | medium | — |
| 7.1 | Create agentic-ai-backlog skill | sonnet | — | medium | — |
| 7.2 | Create agentic-ai-review skill | sonnet | — | high | — |
| 7.3 | Create agentic-ai-done skill | sonnet | — | medium | — |
| 7.4 | Create self-evolve skill | sonnet | — | medium | — |

**Checkpoint:** Full end-to-end review — YOLO mode, consensus gates, all 4 skills, conductor logging. This is the final wave before release.

```
                              *    .  *       .             *
       *   .        *          .     *    .    *    .        *
    .    *    .  ╔══════════════════════════════════╗    .    *
   .        .    ║  ★  A L L   W A V E S   ★       ║  .        .
     *  .     *  ║     C O M P L E T E !           ║     *  .
  .        .     ╚══════════╦═══╦══════════════════╝  .        .
       .    *         ░░░░░░║   ║░░░░░░        *    .
    *     .     *     ░░████║   ║████░░     *     .     *
       .         ░░░░░█████║   ║█████░░░░░         .
    .    *    ░░░░████████║     ║████████░░░░    *    .
          ░░░░███████████╚═════╝███████████░░░░
      ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
         24 tasks  ·  6 waves  ·  4.0x parallelism
            🤖 YOLO MODE ENGAGED  ·  KANBAN LIVE
```

---

## Phase 0: Refactor

### Task 0.1: Analyze home.go extraction boundaries

**Agent Orchestration:**
- Model: opus (Deep analysis of 9031-line file structure)
- Agents: architect
- MCP Tools: — (none)
- Skills: —
- Permissions: default
- Complexity: high
- Wave: 1 | Sequential with: 0.2

**Files:**
- Read: `internal/ui/home.go`
- Create: extraction plan document (ephemeral)

**Steps:**
- [ ] 1. Read all 9031 lines of home.go, catalog every function, type, and method
- [ ] 2. Identify logical groupings: sidebar rendering, session list/grid, card rendering, navigation state, detail/preview logic, dialog management
- [ ] 3. Map each group to target file: kanban_board.go, kanban_card.go, kanban_sidebar.go, kanban_detail.go, kanban_nav.go
- [ ] 4. Identify shared state and helper functions that must stay in home.go
- [ ] 5. Document extraction plan with function-to-file mapping
- [ ] 6. Create stub plans for kanban_transition.go and kanban_conductor.go (no existing code to extract)

**Progress:** 0/6 steps (0%) — pending

---

### Task 0.2: Execute home.go decomposition

**Agent Orchestration:**
- Model: sonnet (Standard refactoring with clear plan from 0.1)
- Agents: refactor-expert, golang-pro
- MCP Tools: — (none)
- Skills: —
- Permissions: acceptEdits
- Complexity: high
- Wave: 1 | Blocked by: 0.1

**Files:**
- Create: `internal/ui/kanban_board.go`, `internal/ui/kanban_card.go`, `internal/ui/kanban_sidebar.go`, `internal/ui/kanban_detail.go`, `internal/ui/kanban_nav.go`, `internal/ui/kanban_transition.go`, `internal/ui/kanban_conductor.go`
- Modify: `internal/ui/home.go`

**Steps:**
- [ ] 1. Create kanban_sidebar.go — extract sidebar/group rendering functions
- [ ] 2. Create kanban_board.go — extract board/grid layout functions
- [ ] 3. Create kanban_card.go — extract card/session rendering functions
- [ ] 4. Create kanban_nav.go — extract navigation state machine logic
- [ ] 5. Create kanban_detail.go — extract preview/detail panel logic
- [ ] 6. Create kanban_transition.go — empty file with package declaration + interfaces
- [ ] 7. Create kanban_conductor.go — empty file with package declaration + interfaces
- [ ] 8. Update home.go imports and cross-references
- [ ] 9. Run `go build ./...` — verify compilation

**Progress:** 0/9 steps (0%) — pending

---

### Task 0.3: Verify functional equivalence

**Agent Orchestration:**
- Model: haiku (Simple verification commands)
- Agents: build-error-resolver (if issues arise)
- MCP Tools: — (none)
- Skills: —
- Permissions: acceptEdits
- Complexity: low
- Wave: 1 | Blocked by: 0.2

**Files:**
- None (verification only)

**Steps:**
- [ ] 1. Run `go build -o build/agent-deck ./cmd/agent-deck/`
- [ ] 2. Run `go test ./... -count=1 -short`
- [ ] 3. Run `go vet ./...`
- [ ] 4. Verify line counts (home.go should be significantly smaller)
- [ ] 5. If any failures, fix and re-verify

**Progress:** 0/5 steps (0%) — pending

---

## Phase 1: Data Layer

### Task 1.1: Define KanbanColumn type + Instance fields

**Agent Orchestration:**
- Model: sonnet (Standard implementation with TDD)
- Agents: tdd-guide, golang-pro
- MCP Tools: — (none)
- Skills: tdd-workflow
- Permissions: acceptEdits
- Complexity: medium
- Wave: 2 | Parallel with: 1.2

**Files:**
- Modify: `internal/session/instance.go`
- Test: `internal/session/instance_test.go`

**Steps:**
- [ ] 1. Write tests for KanbanColumn type (validation, string conversion) — RED
- [ ] 2. Define KanbanColumn type with 6 constants (backlog, design, plan, implement, review, done)
- [ ] 3. Write tests for new Instance fields — RED
- [ ] 4. Add fields to Instance: KanbanColumn, KanbanSortOrder, KanbanLastMoved, Description, AcceptCriteria
- [ ] 5. Add AutomationMode type + YOLOConfig struct
- [ ] 6. Run tests — GREEN
- [ ] 7. Refactor if needed — IMPROVE

**Progress:** 0/7 steps (0%) — pending

---

### Task 1.2: Create SQLite migrations

**Agent Orchestration:**
- Model: sonnet (Database migration work)
- Agents: tdd-guide, golang-pro
- MCP Tools: — (none)
- Skills: tdd-workflow
- Permissions: default (user reviews SQL)
- Complexity: medium
- Wave: 2 | Parallel with: 1.1

**Files:**
- Modify: `internal/session/migration.go`, `internal/session/storage.go`
- Test: `internal/session/storage_test.go`

**Steps:**
- [ ] 1. Write migration tests (table creation, column additions) — RED
- [ ] 2. Add ALTER TABLE statements for instances (kanban_column, kanban_sort_order, kanban_last_moved, description, accept_criteria, automation_mode, yolo_config_json)
- [ ] 3. Create group_kanban_configs table
- [ ] 4. Create column_skill_mappings table with foreign key
- [ ] 5. Run migration tests — GREEN
- [ ] 6. Test migration on fresh database AND on existing database with data

**Progress:** 0/6 steps (0%) — pending

---

### Task 1.3: Add CRUD methods + GroupKanbanConfig persistence

**Agent Orchestration:**
- Model: sonnet (Standard CRUD implementation)
- Agents: tdd-guide, golang-pro
- MCP Tools: — (none)
- Skills: tdd-workflow
- Permissions: acceptEdits
- Complexity: medium
- Wave: 2 | Blocked by: 1.1, 1.2

**Files:**
- Modify: `internal/session/storage.go`
- Test: `internal/session/storage_test.go`

**Steps:**
- [ ] 1. Write tests for kanban CRUD methods — RED
- [ ] 2. Implement UpdateKanbanColumn(), UpdateSortOrder(), UpdateDescription(), UpdateAcceptCriteria()
- [ ] 3. Write tests for GroupKanbanConfig — RED
- [ ] 4. Implement GetGroupKanbanConfig(), SetGroupKanbanConfig()
- [ ] 5. Implement GetColumnSkillMappings(), SetColumnSkillMapping()
- [ ] 6. Run all tests — GREEN
- [ ] 7. Verify no regressions in existing storage tests

**Progress:** 0/7 steps (0%) — pending

---

## Phase 2: Board UI

### Task 2.1: Implement KanbanBoard component

**Agent Orchestration:**
- Model: sonnet (UI component implementation)
- Agents: tdd-guide, golang-pro
- MCP Tools: context7 (optional — Bubble Tea docs reference)
- Skills: tdd-workflow
- Permissions: acceptEdits
- Complexity: medium
- Wave: 3 | Parallel with: 2.2, 2.3

**Files:**
- Modify: `internal/ui/kanban_board.go`
- Test: `internal/ui/kanban_board_test.go`

**Steps:**
- [ ] 1. Write tests for 6-column layout rendering (golden files) — RED
- [ ] 2. Implement KanbanBoard struct with Init(), Update(), View()
- [ ] 3. Use lipgloss JoinHorizontal for column layout
- [ ] 4. Handle terminal width distribution (equal columns minus sidebar)
- [ ] 5. Render column headers with card counts
- [ ] 6. Run tests — GREEN
- [ ] 7. Handle 80-col terminal graceful degradation

**Progress:** 0/7 steps (0%) — pending

---

### Task 2.2: Implement KanbanCard component

**Agent Orchestration:**
- Model: sonnet (UI component)
- Agents: tdd-guide, golang-pro
- MCP Tools: — (none)
- Skills: tdd-workflow
- Permissions: acceptEdits
- Complexity: medium
- Wave: 3 | Parallel with: 2.1, 2.3

**Files:**
- Modify: `internal/ui/kanban_card.go`
- Test: `internal/ui/kanban_card_test.go`

**Steps:**
- [ ] 1. Write tests for card rendering (compact format, status icons) — RED
- [ ] 2. Implement KanbanCard struct with View()
- [ ] 3. Use existing StatusIndicator() for status display
- [ ] 4. Truncate title to column width with ellipsis
- [ ] 5. Show selected card highlight styling
- [ ] 6. Run tests — GREEN

**Progress:** 0/6 steps (0%) — pending

---

### Task 2.3: Implement KanbanSidebar component

**Agent Orchestration:**
- Model: sonnet (UI component)
- Agents: tdd-guide, golang-pro
- MCP Tools: — (none)
- Skills: tdd-workflow
- Permissions: acceptEdits
- Complexity: medium
- Wave: 3 | Parallel with: 2.1, 2.2

**Files:**
- Modify: `internal/ui/kanban_sidebar.go`
- Test: `internal/ui/kanban_sidebar_test.go`

**Steps:**
- [ ] 1. Write tests for group list rendering — RED
- [ ] 2. Implement KanbanSidebar struct with Init(), Update(), View()
- [ ] 3. Show group list with kanban toggle indicator (K icon)
- [ ] 4. Handle j/k navigation within sidebar
- [ ] 5. Fixed 20-column width
- [ ] 6. Run tests — GREEN

**Progress:** 0/6 steps (0%) — pending

---

### Task 2.4: Integrate kanban into home.go routing

**Agent Orchestration:**
- Model: sonnet (Integration work)
- Agents: tdd-guide, golang-pro, code-reviewer
- MCP Tools: — (none)
- Skills: tdd-workflow
- Permissions: acceptEdits
- Complexity: high
- Wave: 3 | Blocked by: 2.1, 2.2, 2.3

**Files:**
- Modify: `internal/ui/home.go`
- Test: `internal/ui/home_test.go`

**Steps:**
- [ ] 1. Write integration tests for kanban mode toggle — RED
- [ ] 2. Add kanban mode flag to home model state
- [ ] 3. Route Update() messages to kanban components when in kanban mode
- [ ] 4. Wire KanbanBoard, KanbanCard, KanbanSidebar into View()
- [ ] 5. Add K key on sidebar to toggle kanban for selected group
- [ ] 6. Run tests — GREEN
- [ ] 7. Build binary and manually verify kanban view renders

**Progress:** 0/7 steps (0%) — pending

---

## Phase 3: Navigation

### Task 3.1: Implement KanbanNav 2D cursor

**Agent Orchestration:**
- Model: sonnet (State machine implementation)
- Agents: tdd-guide, golang-pro
- MCP Tools: — (none)
- Skills: tdd-workflow
- Permissions: acceptEdits
- Complexity: high
- Wave: 4 | Parallel with: 4.1

**Files:**
- Modify: `internal/ui/kanban_nav.go`
- Test: `internal/ui/kanban_nav_test.go`

**Steps:**
- [ ] 1. Write tests for all navigation transitions (h/l, j/k, Tab, 1-6, Enter, Space) — RED
- [ ] 2. Implement KanbanNav struct with col, row, focus panel state
- [ ] 3. Implement h/l (column), j/k (card) movement with boundary clamping
- [ ] 4. Implement Tab focus cycling (sidebar ↔ board ↔ detail)
- [ ] 5. Implement 1-6 column jump
- [ ] 6. Implement Enter (AttachSessionMsg) and Space (DetailToggleMsg)
- [ ] 7. Handle empty columns (skip when navigating h/l)
- [ ] 8. Run tests — GREEN

**Progress:** 0/8 steps (0%) — pending

---

### Task 3.2: Add vertical scroll within columns

**Agent Orchestration:**
- Model: sonnet (Scroll behavior)
- Agents: tdd-guide, golang-pro
- MCP Tools: — (none)
- Skills: tdd-workflow
- Permissions: acceptEdits
- Complexity: medium
- Wave: 4 | Blocked by: 3.1

**Files:**
- Modify: `internal/ui/kanban_nav.go`, `internal/ui/kanban_board.go`

**Steps:**
- [ ] 1. Write tests for scroll behavior (overflow, indicators) — RED
- [ ] 2. Add scrollOffset per column to KanbanNav
- [ ] 3. Calculate visible card window based on terminal height
- [ ] 4. Render scroll indicators: ▲ N above / ▼ N below
- [ ] 5. Auto-scroll to keep selected card visible
- [ ] 6. Run tests — GREEN

**Progress:** 0/6 steps (0%) — pending

---

## Phase 4: Detail Panel

### Task 4.1: Implement KanbanDetail panel

**Agent Orchestration:**
- Model: sonnet (UI component)
- Agents: tdd-guide, golang-pro
- MCP Tools: — (none)
- Skills: tdd-workflow
- Permissions: acceptEdits
- Complexity: medium
- Wave: 4 | Parallel with: 3.1

**Files:**
- Modify: `internal/ui/kanban_detail.go`
- Test: `internal/ui/kanban_detail_test.go`

**Steps:**
- [ ] 1. Write tests for panel rendering with all fields — RED
- [ ] 2. Implement KanbanDetail struct with Init(), Update(), View()
- [ ] 3. Render all 14 fields from design spec (title, description, AC, column, status, tool, worktree, branch, project, model, MCPs, prompt, auto-trigger, YOLO)
- [ ] 4. Handle Space toggle visibility
- [ ] 5. Panel height = 40% of terminal height when open
- [ ] 6. Run tests — GREEN

**Progress:** 0/6 steps (0%) — pending

---

### Task 4.2: Implement edit mode for detail panel

**Agent Orchestration:**
- Model: sonnet (Edit behavior)
- Agents: tdd-guide, golang-pro
- MCP Tools: — (none)
- Skills: tdd-workflow
- Permissions: acceptEdits
- Complexity: medium
- Wave: 4 | Blocked by: 4.1

**Files:**
- Modify: `internal/ui/kanban_detail.go`

**Steps:**
- [ ] 1. Write tests for edit mode transitions and saves — RED
- [ ] 2. Add `e` key handler to enter edit mode
- [ ] 3. Implement text input for title, textarea for description and AC
- [ ] 4. Implement Shift+Y (YOLO toggle) and Shift+A (auto-trigger toggle)
- [ ] 5. Add Esc to exit edit mode without saving, Enter/Tab to save and advance
- [ ] 6. Persist edits to SQLite via storage CRUD methods
- [ ] 7. Run tests — GREEN

**Progress:** 0/7 steps (0%) — pending

---

## Phase 5: Transitions

### Task 5.1: Implement TransitionEngine interface

**Agent Orchestration:**
- Model: sonnet (Interface + core logic)
- Agents: tdd-guide, golang-pro
- MCP Tools: — (none)
- Skills: tdd-workflow
- Permissions: acceptEdits
- Complexity: medium
- Wave: 5 | Sequential first in chain

**Files:**
- Modify: `internal/ui/kanban_transition.go`
- Test: `internal/ui/kanban_transition_test.go`

**Steps:**
- [ ] 1. Write tests for transition validation — RED
- [ ] 2. Define TransitionEngine interface (RequestMove, ResolveSkill, IsValidMove)
- [ ] 3. Define MoveRequest, MoveResult, SkillMapping types
- [ ] 4. Implement IsValidMove() (forward always valid, backward with confirm)
- [ ] 5. Implement basic RequestMove() without skill triggers
- [ ] 6. Run tests — GREEN

**Progress:** 0/6 steps (0%) — pending

---

### Task 5.2: Implement 3-tier config resolution

**Agent Orchestration:**
- Model: sonnet (Config system)
- Agents: tdd-guide, golang-pro
- MCP Tools: — (none)
- Skills: tdd-workflow
- Permissions: acceptEdits
- Complexity: medium
- Wave: 5 | Blocked by: 5.1

**Files:**
- Modify: `internal/ui/kanban_transition.go`

**Steps:**
- [ ] 1. Write tests for config resolution order (SQLite > YAML > defaults) — RED
- [ ] 2. Implement default skill-column mappings (6 columns → 6 skills)
- [ ] 3. Implement YAML config parsing for ~/.config/agent-deck/kanban.yaml
- [ ] 4. Implement ResolveSkillForColumn() with 3-tier resolution
- [ ] 5. Run tests — GREEN

**Progress:** 0/5 steps (0%) — pending

---

### Task 5.3: Implement skill triggers + rollback

**Agent Orchestration:**
- Model: sonnet (Async skill execution)
- Agents: tdd-guide, golang-pro, code-reviewer
- MCP Tools: — (none)
- Skills: tdd-workflow
- Permissions: acceptEdits
- Complexity: high
- Wave: 5 | Blocked by: 5.2

**Files:**
- Modify: `internal/ui/kanban_transition.go`, `internal/ui/home.go`

**Steps:**
- [ ] 1. Write tests for skill execution flow (success + failure) — RED
- [ ] 2. Implement async skill execution: card moves to "processing" state immediately
- [ ] 3. Launch skill in background goroutine via tmux send-keys
- [ ] 4. Send tea.Msg on completion (skillSuccessMsg or skillFailureMsg)
- [ ] 5. Implement rollback: move card back to original column on failure
- [ ] 6. Add confirmation dialog for backward column moves
- [ ] 7. Show error in detail panel with [r]etry [v]iew logs [Esc]ignore
- [ ] 8. Run tests — GREEN

**Progress:** 0/8 steps (0%) — pending

---

## Phase 6: Conductor

### Task 6.1: Implement Conductor lifecycle

**Agent Orchestration:**
- Model: opus (Complex orchestration design)
- Agents: tdd-guide, golang-pro, architect
- MCP Tools: zen (consensus, thinkdeep — for gate design reference)
- Skills: tdd-workflow
- Permissions: default (user reviews orchestration logic)
- Complexity: high
- Wave: 6 | First in wave, blocks 6.2 and 6.3

**Files:**
- Modify: `internal/ui/kanban_conductor.go`
- Test: `internal/ui/kanban_conductor_test.go`
- Read: `internal/session/conductor.go` (reference existing conductor pattern)

**Steps:**
- [ ] 1. Write tests for conductor state machine (spawn, run, terminate, resume) — RED
- [ ] 2. Read existing conductor.go (1903 lines) to understand current patterns
- [ ] 3. Implement Conductor struct with session binding and YOLO config
- [ ] 4. Implement spawn (Shift+Y toggle creates conductor)
- [ ] 5. Implement run loop: for each column → execute skill → wait → gate → advance/pause
- [ ] 6. Implement terminate conditions (Done reached, YOLO disabled, max retries)
- [ ] 7. Implement resume (re-enable YOLO → continue from current column)
- [ ] 8. Run tests — GREEN

**Progress:** 0/8 steps (0%) — pending

---

### Task 6.2: Implement zen consensus gate protocol

**Agent Orchestration:**
- Model: opus (Complex MCP integration)
- Agents: tdd-guide, golang-pro
- MCP Tools: zen (consensus, thinkdeep — for integration testing)
- Skills: tdd-workflow
- Permissions: default
- Complexity: high
- Wave: 6 | Blocked by: 6.1 | Parallel with: 6.3, 7.1-7.4

**Files:**
- Modify: `internal/ui/kanban_conductor.go`

**Steps:**
- [ ] 1. Write tests for gate configurations per transition — RED
- [ ] 2. Define gate configs: models, thinking modes, pass thresholds per column pair
- [ ] 3. Implement consensus call builder (step field, relevant_files, models, stances)
- [ ] 4. Implement blinded consensus evaluation (step 1 proposal, steps 2+ internal notes)
- [ ] 5. Implement pass/fail determination from model verdicts
- [ ] 6. Implement mixed consensus escalation (thinkdeep with same continuation_id)
- [ ] 7. Save outputs >50K chars to temp file before passing as file path
- [ ] 8. Reuse continuation_id across all gates for same session
- [ ] 9. Run tests — GREEN

**Progress:** 0/9 steps (0%) — pending

---

### Task 6.3: Implement YOLO UI indicators

**Agent Orchestration:**
- Model: sonnet (UI indicators)
- Agents: tdd-guide, golang-pro
- MCP Tools: — (none)
- Skills: tdd-workflow
- Permissions: acceptEdits
- Complexity: medium
- Wave: 6 | Blocked by: 6.1 | Parallel with: 6.2, 7.1-7.4

**Files:**
- Modify: `internal/ui/kanban_card.go`, `internal/ui/kanban_conductor.go`

**Steps:**
- [ ] 1. Write tests for YOLO card indicators — RED
- [ ] 2. Add robot icon prefix on YOLO-enabled cards
- [ ] 3. Add progress bar: `Backlog ✓ → Design ✓ → Plan ⚙ → Implement → Review → Done`
- [ ] 4. Add gate status: `Gate: gemini ✓ gpt-5.2 ⏳ o3 ⏳`
- [ ] 5. Add conductor logging to .agent-deck/conductor-log.md
- [ ] 6. Run tests — GREEN

**Progress:** 0/6 steps (0%) — pending

---

## Phase 7: Skills

### Task 7.1: Create agentic-ai-backlog skill

**Agent Orchestration:**
- Model: sonnet (Skill creation with reference files)
- Agents: —
- MCP Tools: — (none)
- Skills: —
- Permissions: acceptEdits
- Complexity: medium
- Wave: 6 | Parallel with: 6.2, 6.3, 7.2, 7.3, 7.4

**Files:**
- Create: `~/.claude/skills/agentic-ai-backlog/SKILL.md`
- Create: `~/.claude/skills/agentic-ai-backlog/references/debt-detection-patterns.md`
- Create: `~/.claude/skills/agentic-ai-backlog/references/security-checklist.md`
- Create: `~/.claude/skills/agentic-ai-backlog/references/performance-metrics.md`
- Create: `~/.claude/skills/agentic-ai-backlog/references/prioritization-matrix.md`

**Steps:**
- [ ] 1. Create skill directory structure
- [ ] 2. Write SKILL.md with YAML frontmatter, workflow checklist, RICE prioritization
- [ ] 3. Create debt-detection-patterns.md (TODO/FIXME patterns, code smells, unused deps)
- [ ] 4. Create security-checklist.md (OWASP top 10, dependency audit)
- [ ] 5. Create performance-metrics.md (N+1 queries, missing indexes, bundle size)
- [ ] 6. Create prioritization-matrix.md (RICE framework template)

**Progress:** 0/6 steps (0%) — pending

---

### Task 7.2: Create agentic-ai-review skill

**Agent Orchestration:**
- Model: sonnet (Complex agent team architecture)
- Agents: —
- MCP Tools: — (none)
- Skills: —
- Permissions: acceptEdits
- Complexity: high
- Wave: 6 | Parallel with: 6.2, 6.3, 7.1, 7.3, 7.4

**Files:**
- Create: `~/.claude/skills/agentic-ai-review/SKILL.md`
- Create: `~/.claude/skills/agentic-ai-review/references/agent-team-guidelines.md`
- Create: `~/.claude/skills/agentic-ai-review/references/chrome-review-guide.md`
- Create: `~/.claude/skills/agentic-ai-review/references/ui-detection-rules.md`
- Create: `~/.claude/skills/agentic-ai-review/references/e2e-test-requirements.md`
- Create: `~/.claude/skills/agentic-ai-review/references/quality-gates.md`

**Steps:**
- [ ] 1. Create skill directory structure
- [ ] 2. Write SKILL.md with agent team architecture (coordinator + 4 sub-agents)
- [ ] 3. Define wave execution: Wave 1 (static-checks ‖ chrome-visual), Wave 2 (e2e-tests ‖ code-review)
- [ ] 4. Create agent-team-guidelines.md (model selection, context isolation rules)
- [ ] 5. Create chrome-review-guide.md (screenshot commands, Lighthouse, console checks)
- [ ] 6. Create ui-detection-rules.md (visual regression, breakpoint testing)
- [ ] 7. Create e2e-test-requirements.md (Playwright patterns, critical flows)
- [ ] 8. Create quality-gates.md (BLOCK criteria, severity definitions)

**Progress:** 0/8 steps (0%) — pending

---

### Task 7.3: Create agentic-ai-done skill

**Agent Orchestration:**
- Model: sonnet (Safety protocol design)
- Agents: —
- MCP Tools: — (none)
- Skills: —
- Permissions: acceptEdits
- Complexity: medium
- Wave: 6 | Parallel with: 6.2, 6.3, 7.1, 7.2, 7.4

**Files:**
- Create: `~/.claude/skills/agentic-ai-done/SKILL.md`
- Create: `~/.claude/skills/agentic-ai-done/references/merge-conflict-resolution.md`
- Create: `~/.claude/skills/agentic-ai-done/references/semver-rules.md`
- Create: `~/.claude/skills/agentic-ai-done/references/changelog-format.md`
- Create: `~/.claude/skills/agentic-ai-done/references/merge-strategies.md`

**Steps:**
- [ ] 1. Create skill directory structure
- [ ] 2. Write SKILL.md with safety protocol (5 pre-merge checks), merge strategy auto-detection
- [ ] 3. Create merge-conflict-resolution.md (git merge-tree dry run, conflict patterns)
- [ ] 4. Create semver-rules.md (conventional commit parsing → version bump)
- [ ] 5. Create changelog-format.md (Keep a Changelog template)
- [ ] 6. Create merge-strategies.md (PR merge vs direct merge decision tree)

**Progress:** 0/6 steps (0%) — pending

---

### Task 7.4: Create self-evolve skill

**Agent Orchestration:**
- Model: sonnet (Learning system design)
- Agents: —
- MCP Tools: — (none)
- Skills: —
- Permissions: acceptEdits
- Complexity: medium
- Wave: 6 | Parallel with: 6.2, 6.3, 7.1, 7.2, 7.3

**Files:**
- Create: `~/.claude/skills/self-evolve/SKILL.md`
- Create: `~/.claude/skills/self-evolve/references/pattern-recognition.md`
- Create: `~/.claude/skills/self-evolve/references/learning-categories.md`
- Create: `~/.claude/skills/self-evolve/references/anti-pattern-detection.md`
- Create: `~/.claude/skills/self-evolve/references/suggestion-templates.md`

**Steps:**
- [ ] 1. Create skill directory structure
- [ ] 2. Write SKILL.md with storage structure, confidence scoring (HIGH/MEDIUM/LOW), cross-validation
- [ ] 3. Create pattern-recognition.md (heuristics for repeated behaviors)
- [ ] 4. Create learning-categories.md (codebase-patterns, user-behaviors, testing, learned)
- [ ] 5. Create anti-pattern-detection.md (contradiction detection, stale pattern pruning)
- [ ] 6. Create suggestion-templates.md (user-facing suggestion formats)

**Progress:** 0/6 steps (0%) — pending
