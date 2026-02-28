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
| Total Tasks | 28 |
| Total Waves | 6 |
| Avg Parallelism | 4.0x |
| Max Parallelism | 6 |
| Critical Path | 16 tasks |

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

### Wave 2: Data Layer — 5 tasks (1.1 ‖ 1.2, then 1.3 ‖ 1.4, then 1.5)

| Task | Name | Model | Agents | Complexity | MCP Tools |
|------|------|-------|--------|------------|-----------|
| 1.1 | Define KanbanColumn type + Instance fields | sonnet | tdd-guide, golang-pro | medium | — |
| 1.2 | Create SQLite migrations | sonnet | tdd-guide, golang-pro | medium | — |
| 1.3 | Add CRUD methods + GroupKanbanConfig persistence | sonnet | tdd-guide, golang-pro | medium | — |
| 1.4 | Define Bubble Tea message types for kanban | sonnet | golang-pro | medium | — |
| 1.5 | Implement sort order calculation + rebalancing | sonnet | tdd-guide, golang-pro | medium | — |

**Checkpoint:** Review schema design — new Instance fields, migration SQL, CRUD methods, message types, sort order logic. Verify migration runs cleanly on existing databases.

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

### Wave 4: Navigation & Detail — 6 tasks (3.1 ‖ 4.1, then 3.2 ‖ 4.2, then 3.3 ‖ 3.4)

| Task | Name | Model | Agents | Complexity | MCP Tools |
|------|------|-------|--------|------------|-----------|
| 3.1 | Implement KanbanNav 2D cursor | sonnet | tdd-guide, golang-pro | high | — |
| 4.1 | Implement KanbanDetail panel | sonnet | tdd-guide, golang-pro | medium | — |
| 3.2 | Add vertical scroll within columns | sonnet | tdd-guide, golang-pro | medium | — |
| 4.2 | Implement edit mode for detail panel | sonnet | tdd-guide, golang-pro | medium | — |
| 3.3 | Implement `n` (new session) and `d` (delete session) in board | sonnet | tdd-guide, golang-pro | medium | — |
| 3.4 | Implement `m` (move card) interaction flow | sonnet | tdd-guide, golang-pro | medium | — |

**Checkpoint:** Test keyboard navigation end-to-end — h/j/k/l, Tab, 1-6, Enter, Space, e, Esc, n, d, m. Verify detail panel shows all fields correctly. Verify card creation, deletion, and move flows.

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
- [ ] 5. Create golden file test helper in `internal/ui/testutil_test.go` with `-update` flag pattern for updating golden files. Create `internal/ui/testdata/` directory for golden file storage
- [ ] 6. If any failures, fix and re-verify

**Progress:** 0/6 steps (0%) — pending

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
- [ ] 1. Write tests for KanbanColumn type — RED:
  - `TestKanbanColumnValid`: all 6 values ("backlog", "design", "plan", "implement", "review", "done") are valid
  - `TestKanbanColumnInvalid`: empty string, "unknown", "BACKLOG" (case), "planning" return error
  - `TestKanbanColumnString`: each constant returns its lowercase string representation
  - `TestParseKanbanColumn`: "backlog" → KanbanBacklog, "done" → KanbanDone, "invalid" → error
  - `TestKanbanColumnOrder`: verify column index (Backlog=0, Design=1, ..., Done=5) for sort order calculations
  - `TestKanbanColumnNilSafe`: nil *KanbanColumn handled without panic in comparison/string methods
- [ ] 2. Define KanbanColumn type with 6 constants (backlog, design, plan, implement, review, done). Add `ParseKanbanColumn(s string) (KanbanColumn, error)` and `(k KanbanColumn) Index() int` methods
- [ ] 3. Write tests for new Instance fields — RED:
  - `TestInstanceKanbanFields`: Instance with all new fields serializes/deserializes correctly
  - `TestInstanceDefaultValues`: new Instance has nil KanbanColumn, 0 sort order, nil KanbanLastMoved, empty Description/AcceptCriteria
  - `TestInstanceAutomationMode`: "interactive" is default, "yolo" is valid, "" defaults to "interactive"
  - `TestYOLOConfigJSON`: YOLOConfig serializes to/from JSON correctly including all fields
- [ ] 4. Add fields to Instance: KanbanColumn (*KanbanColumn), KanbanSortOrder (int), KanbanLastMoved (*time.Time), Description (string), AcceptCriteria (string), AutomationMode (AutomationMode), YOLOConfig (*YOLOConfig)
- [ ] 5. Add AutomationMode type (AutomationInteractive, AutomationYOLO) + YOLOConfig struct with fields: Mode, ConsensusModels, PassThreshold, PauseOnFail, MaxRetries, SkipColumns, ContinuationID
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
- [ ] 1. Write migration tests — RED:
  - `TestMigrationFreshDB`: migration on empty database creates all columns and tables
  - `TestMigrationExistingData`: migration on DB with existing sessions preserves all data, new columns have correct defaults (kanban_column=NULL, kanban_sort_order=0, description='', accept_criteria='', automation_mode='interactive')
  - `TestMigrationIdempotent`: running migration twice doesn't error
  - `TestGroupKanbanConfigsTable`: table created with group_path PK, kanban_enabled, timestamps
  - `TestColumnSkillMappingsTable`: table created with FK to group_kanban_configs, UNIQUE(group_path, column_name)
- [ ] 2. Add ALTER TABLE statements for instances (kanban_column TEXT, kanban_sort_order INTEGER DEFAULT 0, kanban_last_moved TIMESTAMP, description TEXT DEFAULT '', accept_criteria TEXT DEFAULT '', automation_mode TEXT DEFAULT 'interactive', yolo_config_json TEXT)
- [ ] 3. Create group_kanban_configs table (group_path TEXT PK, kanban_enabled BOOLEAN DEFAULT 0, created_at, updated_at)
- [ ] 4. Create column_skill_mappings table (id INTEGER PK AUTOINCREMENT, group_path TEXT NOT NULL, column_name TEXT NOT NULL, skill_name TEXT NOT NULL, auto_trigger BOOLEAN DEFAULT 0, trigger_on_enter BOOLEAN DEFAULT 1, FK group_path, UNIQUE(group_path, column_name))
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
- [ ] 1. Write tests for kanban CRUD methods — RED:
  - `TestUpdateKanbanColumn`: set column to "design", read back, verify. Set nil (remove from kanban), verify
  - `TestUpdateKanbanColumn_InvalidColumn`: "invalid" → error, column unchanged
  - `TestUpdateSortOrder`: set sort order, read back, verify
  - `TestUpdateDescription`: set description, read back, verify. Empty string allowed
  - `TestUpdateAcceptCriteria`: set AC, read back, verify. Multi-line string preserved
  - `TestUpdateAutomationMode`: set "yolo", read back. Set "interactive", read back. Invalid → error
  - `TestUpdateYOLOConfig`: serialize YOLOConfig to JSON, store, read back, deserialize, verify all fields
  - `TestGetSessionsByColumn`: insert 3 sessions in "backlog", 2 in "design", query by column, verify counts and order
- [ ] 2. Implement UpdateKanbanColumn(), UpdateSortOrder(), UpdateDescription(), UpdateAcceptCriteria(), UpdateAutomationMode(), UpdateYOLOConfig()
- [ ] 3. Write tests for GroupKanbanConfig — RED:
  - `TestSetGroupKanbanConfig`: enable kanban for group, read back, verify kanban_enabled=true
  - `TestSetGroupKanbanConfig_Toggle`: enable then disable, verify
  - `TestGetGroupKanbanConfig_NotFound`: returns default (kanban_enabled=false)
  - `TestSetColumnSkillMapping`: set mapping for "backlog" → "agentic-ai-backlog", read back
  - `TestSetColumnSkillMapping_Override`: set mapping twice, second overrides first
  - `TestGetColumnSkillMappings`: set 3 mappings, get all for group, verify 3 returned
- [ ] 4. Implement GetGroupKanbanConfig(), SetGroupKanbanConfig()
- [ ] 5. Implement GetColumnSkillMappings(), SetColumnSkillMapping(), GetSessionsByColumn()
- [ ] 6. Run all tests — GREEN
- [ ] 7. Verify no regressions in existing storage tests

**Progress:** 0/7 steps (0%) — pending

---

### Task 1.4: Define Bubble Tea message types for kanban

**Agent Orchestration:**
- Model: sonnet (Type definitions with clear spec from design doc)
- Agents: golang-pro
- MCP Tools: — (none)
- Skills: —
- Permissions: acceptEdits
- Complexity: medium
- Wave: 2 | Parallel with: 1.3 | Blocked by: 1.1

**Context:** The design doc (Message Flow section) defines 15+ message types that drive the entire Kanban UI. ALL UI tasks in Waves 3-6 depend on these types being defined first. Without these, parallel agents will invent inconsistent message structs.

**Files:**
- Create: `internal/ui/kanban_messages.go`
- Test: `internal/ui/kanban_messages_test.go`

**Steps:**
- [ ] 1. Write tests verifying all message types implement tea.Msg interface — RED:
  - `TestAllMessagesImplementMsg`: each message type satisfies `tea.Msg` interface
- [ ] 2. Define user input messages:
  - `FocusChangedMsg { Panel FocusPanel }` — Tab switches focus (FocusPanel enum: Sidebar, Board, Detail)
  - `ColumnChangedMsg { Column int }` — h/l navigates columns
  - `CardChangedMsg { Row int }` — j/k navigates cards
  - `ColumnJumpMsg { Column int }` — 1-6 direct column jump
  - `DetailToggleMsg {}` — Space toggles detail panel
  - `AttachSessionMsg { SessionID string }` — Enter opens tmux session
  - `EditModeMsg { Enabled bool }` — e enters edit, Esc exits
  - `MoveInitiatedMsg { SessionID string }` — m starts move mode
  - `MoveConfirmedMsg { SessionID string, FromColumn KanbanColumn, ToColumn KanbanColumn }` — Enter confirms move target
  - `YOLOModeToggleMsg { SessionID string }` — Shift+Y toggles YOLO
  - `AutoTriggerToggleMsg { SessionID string }` — Shift+A toggles auto-trigger
  - `CreateSessionMsg { Column KanbanColumn }` — n creates session in column
  - `DeleteSessionMsg { SessionID string }` — d deletes session
  - `KanbanToggleMsg { GroupPath string }` — K toggles kanban for group
- [ ] 3. Define system/response messages:
  - `ShowConfirmDialogMsg { Title, Body string, OnConfirm tea.Msg }` — backward move confirmation
  - `ExecuteTransitionMsg { Request MoveRequest }` — confirmed transition to execute
  - `SkillTriggeredMsg { SessionID string, SkillName string, Column KanbanColumn }` — skill started
  - `SkillCompletedMsg { SessionID string, Success bool, Error error, Output string }` — skill finished
  - `BoardRefreshMsg {}` — refresh board data from storage
  - `RollbackMsg { SessionID string, FromColumn, ToColumn KanbanColumn, Error error }` — move failed, rolling back
  - `ErrorDisplayMsg { SessionID string, Error error, Actions []ErrorAction }` — show error with action options
  - `ConductorStatusMsg { SessionID string, Column KanbanColumn, GateStatus map[string]string }` — YOLO progress update
- [ ] 4. Define supporting types:
  - `FocusPanel` enum: `PanelSidebar`, `PanelBoard`, `PanelDetail`
  - `ErrorAction` struct: `Key rune`, `Label string`, `Msg tea.Msg`
- [ ] 5. Run tests — GREEN

**Progress:** 0/5 steps (0%) — pending

---

### Task 1.5: Implement sort order calculation + rebalancing

**Agent Orchestration:**
- Model: sonnet (Algorithm implementation with TDD)
- Agents: tdd-guide, golang-pro
- MCP Tools: — (none)
- Skills: tdd-workflow
- Permissions: acceptEdits
- Complexity: medium
- Wave: 2 | Blocked by: 1.1, 1.3

**Context:** Design doc Sort Order section specifies: `sort_order = 1000 * column_index + position * 100`. Insert between: `(above + below) / 2`. Rebalance when gap < 10.

**Files:**
- Create: `internal/session/kanban_sort.go`
- Test: `internal/session/kanban_sort_test.go`

**Steps:**
- [ ] 1. Write tests for sort order calculation — RED:
  - `TestInitialSortOrder`: first card in Backlog → 0, first in Design → 1000, first in Done → 5000
  - `TestAppendSortOrder`: 3 cards in Backlog → sort orders 0, 100, 200
  - `TestInsertBetween`: insert between sort_order 100 and 200 → 150
  - `TestInsertBetween_NarrowGap`: insert between 100 and 101 → triggers rebalance
  - `TestRebalance`: 5 cards with gaps <10 → rebalanced to 0, 100, 200, 300, 400
  - `TestColumnMoveSortOrder`: card moves from Backlog(sort=150) to Design → gets sort_order = 1000 + next_position
  - `TestEmptyColumnSortOrder`: insert into empty column → column_base + 0
- [ ] 2. Implement `CalculateInitialSortOrder(column KanbanColumn, existingCards []Instance) int`
- [ ] 3. Implement `CalculateInsertBetween(above, below int) int`
- [ ] 4. Implement `NeedsRebalance(sortOrders []int) bool` — returns true if min gap < 10
- [ ] 5. Implement `Rebalance(cards []Instance) []Instance` — redistributes sort orders evenly (immutable, returns new slice). **IMPORTANT:** storage layer must support batch update for rebalanced sort orders — add `BatchUpdateSortOrders(updates map[string]int) error` to storage.go to avoid N individual DB writes causing UI stutter
- [ ] 6. Implement `CalculateMoveSortOrder(targetColumn KanbanColumn, existingCards []Instance) int`
- [ ] 7. Run tests — GREEN

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
- [ ] 1. Write tests for 6-column layout rendering — RED:
  - `TestBoardView_120Cols`: 120-col terminal with 20-col sidebar → 6 columns at ~16 chars each, all visible
  - `TestBoardView_160Cols`: 160-col terminal → 6 columns at ~23 chars each, comfortable
  - `TestBoardView_80Cols`: graceful degradation — **hide sidebar, show only 3 columns (current + neighbors) with `◄` / `►` scroll indicators** (design doc error handling table)
  - `TestBoardView_ColumnHeaders`: each column shows abbreviated header + card count: "BL(3)", "DE(1)", etc.
  - `TestBoardView_EmptyBoard`: empty group shows board with "Press `n` to create a session" hint (design doc error handling)
  - `TestBoardView_CardsInColumns`: cards sorted by sort_order within each column
  - Golden file: `testdata/board_120cols.golden`, `testdata/board_80cols.golden`
- [ ] 2. Implement KanbanBoard struct with Init(), Update(), View(). Fields: `columns [6][]Instance`, `width int`, `height int`, `selectedCol int`, `scrollOffset int` (for <80 col degraded mode)
- [ ] 3. Use lipgloss JoinHorizontal for column layout. Use existing `ensureExactWidth()` for column alignment
- [ ] 4. Handle terminal width distribution: `colWidth = (termWidth - sidebarWidth) / 6`
- [ ] 5. Render column headers with card counts: abbreviated names "BL", "DE", "PL", "IM", "RE", "DO"
- [ ] 6. Run tests — GREEN
- [ ] 7. Implement 80-col degradation: hide sidebar, show 3 columns (current + left/right neighbors), render `◄ N` / `► N` scroll indicators

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
- [ ] 1. Write tests for card rendering — RED:
  - `TestCardView_Compact`: renders status icon + truncated title in ~15 chars (e.g., "● fix-auth...")
  - `TestCardView_StatusIcons`: running=`●`, waiting=`◐`, idle=`○`, error=`✗` (reuse existing StatusIndicator())
  - `TestCardView_Selected`: selected card has highlight border/background styling via lipgloss
  - `TestCardView_Unselected`: unselected card has default styling
  - `TestCardView_TitleTruncation`: title longer than column width truncated with "..." ellipsis
  - `TestCardView_ShortTitle`: title shorter than column width renders without truncation
  - `TestCardView_YOLOIcon`: YOLO-enabled card prefixed with robot icon (handled in Task 6.3, but struct field needed now)
  - Golden file: `testdata/card_compact.golden`
- [ ] 2. Implement KanbanCard struct with View(). Fields: `instance *Instance`, `selected bool`, `width int`
- [ ] 3. Use existing StatusIndicator() for status display
- [ ] 4. Truncate title to column width with ellipsis using `runewidth.Truncate()` or equivalent
- [ ] 5. Show selected card highlight styling via lipgloss.NewStyle().Background()/Border()
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
- [ ] 1. Write tests for group list rendering — RED:
  - `TestSidebarView_Groups`: renders list of groups with names, selected group highlighted
  - `TestSidebarView_KanbanIndicator`: groups with kanban enabled show "[K]" suffix; others don't
  - `TestSidebarView_FixedWidth`: sidebar always renders at exactly 20 columns
  - `TestSidebarView_AllSessionsFirst`: "All Sessions" group always listed first
  - `TestSidebarNav_JK`: j moves selection down, k moves up, wraps at boundaries
  - `TestSidebarNav_Focused`: sidebar only responds to j/k when it has focus (FocusPanel == PanelSidebar)
- [ ] 2. Implement KanbanSidebar struct with Init(), Update(), View(). Fields: `groups []*Group`, `selectedIdx int`, `focused bool`, `height int`
- [ ] 3. Show group list with kanban toggle indicator ("[K]" suffix for kanban-enabled groups)
- [ ] 4. Handle j/k navigation within sidebar (only when focused). Clamp selection to 0..len(groups)-1
- [ ] 5. Fixed 20-column width via lipgloss.Width(20) or ensureExactWidth
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
- [ ] 1. Write integration tests for kanban mode toggle — RED:
  - `TestHomeKanbanRouting`: when selected group has kanban enabled, View() renders KanbanBoard+KanbanSidebar instead of session list
  - `TestHomeKanbanRouting_AllSessions`: "All Sessions" group ALWAYS renders flat list view, NEVER kanban, even if other groups have kanban enabled (design doc AC line 21)
  - `TestHomeKanbanToggle`: pressing `K` on sidebar group toggles kanban_enabled in storage, triggers BoardRefreshMsg
  - `TestHomeKanbanMessages`: FocusChangedMsg, ColumnChangedMsg, CardChangedMsg routed to correct sub-components
  - `TestHomeKanbanDisabled`: groups without kanban show standard session list/grid view
- [ ] 2. Add kanban mode flag to home model state: `kanbanMode bool`, `kanbanBoard *KanbanBoard`, `kanbanSidebar *KanbanSidebar`, `kanbanDetail *KanbanDetail`
- [ ] 3. Route Update() messages to kanban components when kanbanMode is true. Pass FocusChangedMsg, ColumnChangedMsg, CardChangedMsg, etc. to correct component
- [ ] 4. Wire KanbanBoard, KanbanCard, KanbanSidebar into View() using lipgloss.JoinHorizontal (sidebar | board)
- [ ] 5. Implement `K` key handler on sidebar: toggle kanban for selected group via SetGroupKanbanConfig(), send KanbanToggleMsg
- [ ] 6. Implement "All Sessions" exception: if selected group is the root/all-sessions group, always render flat list regardless of kanban state
- [ ] 7. Run tests — GREEN
- [ ] 8. Build binary (`go build -o build/agent-deck ./cmd/agent-deck/`) and manually verify kanban view renders

**Progress:** 0/8 steps (0%) — pending

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
- [ ] 1. Write tests for all navigation transitions — RED:
  - `TestNav_HLColumns`: h moves left (col 2→1), l moves right (col 1→2). Clamp at 0 and 5
  - `TestNav_JKCards`: j moves down (row 0→1), k moves up (row 1→0). Clamp at 0 and len(cards)-1
  - `TestNav_Tab`: Tab cycles PanelSidebar → PanelBoard → PanelDetail → PanelSidebar. Only cycles to PanelDetail if detail panel is open
  - `TestNav_ColumnJump`: keys 1→col 0, 2→col 1, ..., 6→col 5. Row resets to 0 on column change
  - `TestNav_Enter`: sends AttachSessionMsg with current card's SessionID. No-op if column is empty
  - `TestNav_Space`: sends DetailToggleMsg. Opens if closed, closes if open
  - `TestNav_EmptyColumn`: h/l skips empty columns. If all columns empty, stays at current position
  - `TestNav_ColumnChange_RowClamp`: when moving from column with 5 cards to column with 2, row clamped to 1 (max index)
  - `TestNav_FocusGuard`: h/l/j/k/1-6 only work when focus==PanelBoard. Tab works from any panel
- [ ] 2. Implement KanbanNav struct: `col int`, `row int`, `focus FocusPanel`, `detailOpen bool`, `columnCardCounts [6]int`
- [ ] 3. Implement h/l (column), j/k (card) movement with boundary clamping. On column change, clamp row to min(currentRow, newColumnCardCount-1)
- [ ] 4. Implement Tab focus cycling: Sidebar → Board → Detail (if open) → Sidebar
- [ ] 5. Implement 1-6 column jump: `ColumnJumpMsg{Column: key - '1'}`, reset row to 0
- [ ] 6. Implement Enter → `AttachSessionMsg{SessionID}` and Space → `DetailToggleMsg{}`
- [ ] 7. Handle empty columns: h/l skips columns where `columnCardCounts[col] == 0`
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
- [ ] 1. Write tests for scroll behavior — RED:
  - `TestScroll_NoOverflow`: 3 cards, terminal fits 10 cards → no scroll indicators, scrollOffset=0
  - `TestScroll_Overflow`: 15 cards, terminal fits 8 → shows "▼ 7 below" at bottom
  - `TestScroll_ScrollDown`: selected card at row 9, viewport shows rows 2-9, "▲ 2 above" shown
  - `TestScroll_AutoFollow`: moving j/k auto-adjusts scrollOffset to keep selected card in viewport
  - `TestScroll_BothIndicators`: scrolled to middle → "▲ N above" and "▼ M below" both shown
  - `TestScroll_PerColumn`: each column has independent scrollOffset
- [ ] 2. Add `scrollOffsets [6]int` to KanbanNav (per-column scroll state)
- [ ] 3. Calculate visible card window: `visibleCards = (termHeight - headerHeight - footerHeight) / cardHeight`
- [ ] 4. Render scroll indicators: `▲ N above` at top / `▼ N below` at bottom of column
- [ ] 5. Auto-scroll: when selected row < scrollOffset → scrollOffset = row. When selected row >= scrollOffset + visibleCards → scrollOffset = row - visibleCards + 1
- [ ] 6. Run tests — GREEN

**Progress:** 0/6 steps (0%) — pending

---

### Task 3.3: Implement `n` (new session) and `d` (delete session) in board

**Agent Orchestration:**
- Model: sonnet (Keyboard handler implementation)
- Agents: tdd-guide, golang-pro
- MCP Tools: — (none)
- Skills: tdd-workflow
- Permissions: acceptEdits
- Complexity: medium
- Wave: 4 | Blocked by: 3.1

**Context:** Design doc Keyboard Navigation table defines `n` = create new session in focused column, `d` = delete session with confirmation. These are essential board interactions.

**Files:**
- Modify: `internal/ui/kanban_board.go`, `internal/ui/home.go`
- Test: `internal/ui/kanban_board_test.go`

**Steps:**
- [ ] 1. Write tests for session creation/deletion — RED:
  - `TestCreateSession_N`: pressing `n` with board focused sends CreateSessionMsg{Column: currentCol}
  - `TestCreateSession_EmptyColumn`: pressing `n` in empty column creates session, card appears
  - `TestDeleteSession_D`: pressing `d` sends ShowConfirmDialogMsg with "Delete session?" prompt
  - `TestDeleteSession_Confirm`: confirming deletion sends DeleteSessionMsg{SessionID}, card removed
  - `TestDeleteSession_Cancel`: canceling deletion returns to board, no change
  - `TestDeleteSession_Empty`: pressing `d` with no card selected is no-op
- [ ] 2. Implement `n` handler: send CreateSessionMsg with current column. On response, create session via storage, assign sort order via CalculateInitialSortOrder, send BoardRefreshMsg
- [ ] 3. Implement `d` handler: send ShowConfirmDialogMsg, on confirm → delete session via storage, send BoardRefreshMsg
- [ ] 4. Run tests — GREEN

**Progress:** 0/4 steps (0%) — pending

---

### Task 3.4: Implement `m` (move card) interaction flow

**Agent Orchestration:**
- Model: sonnet (Multi-step interaction flow)
- Agents: tdd-guide, golang-pro
- MCP Tools: — (none)
- Skills: tdd-workflow
- Permissions: acceptEdits
- Complexity: medium
- Wave: 4 | Blocked by: 3.1

**Context:** Design doc Keyboard Navigation: `m` = move card (then h/l to select target, Enter to confirm). This is a modal interaction — board enters "move mode" where h/l highlights target columns.

**Files:**
- Modify: `internal/ui/kanban_board.go`, `internal/ui/kanban_nav.go`
- Test: `internal/ui/kanban_board_test.go`

**Steps:**
- [ ] 1. Write tests for move interaction — RED:
  - `TestMove_Initiate`: pressing `m` enters move mode, sends MoveInitiatedMsg. Target column highlighted differently
  - `TestMove_SelectTarget`: in move mode, h/l selects target column (visual indicator on target)
  - `TestMove_Confirm`: in move mode, Enter sends MoveConfirmedMsg{SessionID, FromColumn, ToColumn}
  - `TestMove_Cancel`: in move mode, Esc cancels move, returns to normal navigation. No MoveConfirmedMsg fired, UI reflects original column/sort order unchanged
  - `TestMove_SameColumn`: confirming move to same column is no-op
  - `TestMove_BackwardConfirm`: moving backward (e.g., Implement→Design) triggers ShowConfirmDialogMsg before executing (design doc: "backward with confirmation")
- [ ] 2. Add `moveMode bool`, `moveTargetCol int` to KanbanNav state
- [ ] 3. Implement move mode: `m` sets moveMode=true, moveTargetCol=currentCol. h/l changes moveTargetCol. Enter confirms. Esc cancels
- [ ] 4. Implement visual indicator: target column header highlighted in move mode
- [ ] 5. Run tests — GREEN

**Progress:** 0/5 steps (0%) — pending

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
- [ ] 1. Write tests for panel rendering with all fields — RED:
  - `TestDetailView_AllFields`: panel renders all 14 fields from design spec: Title, Description, AC, Column, Status, Tool, Worktree Path, Worktree Branch, Project Path, Model, MCP Servers, Last Prompt, Auto-trigger, YOLO Mode
  - `TestDetailView_EditableMarkers`: editable fields (Title, Description, AC, Column, Auto-trigger, YOLO) show edit indicators; read-only fields don't
  - `TestDetailView_Height`: panel height = 40% of terminal height
  - `TestDetailView_SpaceToggle`: Space opens panel (visible=true), Space again closes (visible=false)
  - `TestDetailView_NilSession`: with no session selected, panel shows "No session selected"
  - `TestDetailView_LongContent`: description/AC with long text wraps within panel width, doesn't overflow
- [ ] 2. Implement KanbanDetail struct with Init(), Update(), View(). Fields: `instance *Instance`, `visible bool`, `editing bool`, `focusedField int`, `height int`
- [ ] 3. Render all 14 fields. Editable: Title (text input), Description (textarea), AcceptCriteria (textarea), KanbanColumn (dropdown-style), Auto-trigger (toggle), YOLO (toggle). Read-only: Status, Tool, Worktree Path/Branch, Project Path, Model, MCP Servers, Last Prompt
- [ ] 4. Handle Space toggle: DetailToggleMsg flips `visible` state
- [ ] 5. Panel height = 40% of terminal height when open. Board takes remaining 60%
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
- [ ] 1. Write tests for edit mode transitions and saves — RED:
  - `TestEditMode_Enter`: pressing `e` sets editing=true, focuses first editable field (Title)
  - `TestEditMode_Escape`: pressing Esc exits edit mode WITHOUT saving, restores original values
  - `TestEditMode_TabAdvance`: Tab moves focus to next editable field (Title→Description→AC→Column→Auto-trigger→YOLO)
  - `TestEditMode_SaveTitle`: editing title, pressing Enter saves to storage, sends BoardRefreshMsg
  - `TestEditMode_SaveDescription`: editing multi-line description, Tab/Enter saves
  - `TestEditMode_YOLOToggle`: Shift+Y toggles AutomationMode between "interactive" and "yolo" in storage
  - `TestEditMode_AutoTriggerToggle`: Shift+A toggles auto_trigger for current session's column config
  - `TestEditMode_ColumnChange`: changing column in detail panel triggers MoveConfirmedMsg (same as `m` key)
  - `TestEditMode_OnlyWhenOpen`: `e` is no-op when detail panel is not visible
- [ ] 2. Add `e` key handler: only works when detail panel is visible and focus==PanelDetail. Sets editing=true
- [ ] 3. Implement text input for title (bubbletea textinput), textarea for description and AC (bubbletea textarea)
- [ ] 4. Implement Shift+Y → YOLOModeToggleMsg, Shift+A → AutoTriggerToggleMsg
- [ ] 5. Esc exits edit mode (discard changes by reloading from Instance). Enter/Tab saves current field and advances to next
- [ ] 6. Persist edits to SQLite: UpdateDescription(), UpdateAcceptCriteria(), UpdateAutomationMode(), etc. from storage CRUD (Task 1.3)
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
- [ ] 1. Write tests for transition validation — RED:
  - `TestIsValidMove_Forward`: Backlog→Design=valid, Backlog→Implement=valid (skip allowed), Design→Done=valid
  - `TestIsValidMove_Backward`: Implement→Design=valid but NeedsConfirm=true, Review→Backlog=valid+NeedsConfirm
  - `TestIsValidMove_SameColumn`: Design→Design=invalid (no-op)
  - `TestIsValidMove_NilColumn`: nil→Backlog=valid (first assignment)
  - `TestRequestMove_Forward`: forward move returns Executed=true, NeedsConfirm=false
  - `TestRequestMove_Backward`: backward move returns Executed=false, NeedsConfirm=true
  - `TestRequestMove_ForceConfirm`: ForceConfirm=true always requires confirmation even for forward
- [ ] 2. Define TransitionEngine interface: `RequestMove(req MoveRequest) MoveResult`, `ResolveSkill(groupPath string, column KanbanColumn) (SkillMapping, error)`, `IsValidMove(from, to KanbanColumn) bool`
- [ ] 3. Define types: MoveRequest (SessionID, FromColumn, ToColumn, GroupPath, ForceConfirm), MoveResult (Executed, NeedsConfirm, SkillName, Error), SkillMapping (SkillName, AutoTrigger)
- [ ] 4. Define error types: `TransitionError` (wraps move failure), `SkillFailedError` (skill execution failed), `RollbackError` (rollback also failed — critical)
- [ ] 5. Implement IsValidMove(): forward always valid, backward valid but NeedsConfirm=true, same column invalid
- [ ] 6. Implement basic RequestMove() without skill triggers: validates move, updates column in storage, updates sort order via CalculateMoveSortOrder
- [ ] 7. Run tests — GREEN

**Progress:** 0/7 steps (0%) — pending

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
- [ ] 1. Write tests for config resolution order — RED:
  - `TestResolveSkill_DefaultsOnly`: no YAML, no SQLite config → returns hardcoded defaults (backlog→agentic-ai-backlog, design→agentic-ai-brainstorm, plan→agentic-ai-plan, implement→agentic-ai-implement, review→agentic-ai-review, done→agentic-ai-done). All auto_trigger=false
  - `TestResolveSkill_YAMLOverride`: YAML sets backlog skill to "custom-backlog" → returns "custom-backlog", other columns still use defaults
  - `TestResolveSkill_SQLiteOverride`: SQLite has mapping for design→"my-design-skill" → returns "my-design-skill" even if YAML has different value (SQLite > YAML > defaults)
  - `TestResolveSkill_NoYAMLFile`: ~/.config/agent-deck/kanban.yaml doesn't exist → falls through to defaults (no error)
  - `TestResolveSkill_InvalidYAML`: malformed YAML → log warning, fall through to defaults
  - `TestResolveSkill_AutoTrigger`: config with auto_trigger=true returns SkillMapping{AutoTrigger: true}
- [ ] 2. Implement default skill-column mappings as a hardcoded map: `var defaultSkillMappings = map[KanbanColumn]SkillMapping{...}`
- [ ] 3. Define YAML schema and implement parsing:
  ```yaml
  # ~/.config/agent-deck/kanban.yaml
  columns:
    backlog:
      skill: agentic-ai-backlog
      auto_trigger: false
    design:
      skill: agentic-ai-brainstorm
      auto_trigger: false
    plan:
      skill: agentic-ai-plan
      auto_trigger: false
    implement:
      skill: agentic-ai-implement
      auto_trigger: false
    review:
      skill: agentic-ai-review
      auto_trigger: false
    done:
      skill: agentic-ai-done
      auto_trigger: false
  ```
- [ ] 4. Implement ResolveSkillForColumn(groupPath, column): check SQLite (GetColumnSkillMappings) → check YAML → return default
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
- [ ] 1. Write tests for skill execution flow — RED:
  - `TestSkillTrigger_Success`: MoveConfirmedMsg with auto_trigger=true → SkillTriggeredMsg sent → skill completes → SkillCompletedMsg{Success:true} → card stays in new column, BoardRefreshMsg
  - `TestSkillTrigger_Failure`: skill fails → SkillCompletedMsg{Success:false, Error} → RollbackMsg sent → card returns to original column → ErrorDisplayMsg with actions: `[r]etry [v]iew logs [Esc]ignore`
  - `TestSkillTrigger_NoAutoTrigger`: auto_trigger=false → move succeeds without triggering skill
  - `TestSkillTrigger_BackwardConfirm`: backward move → ShowConfirmDialogMsg → user confirms → ExecuteTransitionMsg → skill NOT triggered (design: "Skill won't re-run")
  - `TestSkillTrigger_BackwardCancel`: backward confirm canceled → card stays in original column
  - `TestSkillTrigger_ProcessingState`: during skill execution, card shows "processing" indicator
  - `TestRollback_Success`: rollback restores original column and sort order
  - `TestRollback_Failure`: rollback also fails → RollbackError logged, card stuck in "error" state, user must manually fix
  - `TestErrorDisplay_Retry`: user presses `r` on error → re-triggers skill
  - `TestErrorDisplay_ViewLogs`: user presses `v` → opens detail panel with error logs
  - `TestErrorDisplay_Ignore`: user presses Esc → dismisses error, card stays in new column without skill
- [ ] 2. Implement async skill execution: on ExecuteTransitionMsg, card moves immediately. If skill configured, launch in goroutine
- [ ] 3. Launch skill in background via tmux send-keys: `tmux send-keys -t {sessionID} "/<skill-name>" Enter`
- [ ] 4. Send tea.Msg on completion: SkillCompletedMsg{SessionID, Success, Error, Output}
- [ ] 5. Implement rollback: on failure, UpdateKanbanColumn back to FromColumn, restore original sort order
- [ ] 6. Add confirmation dialog for backward moves: ShowConfirmDialogMsg{Title: "Move backward?", Body: "Move to {column}? Skill won't re-run."}
- [ ] 7. Show error in detail panel: ErrorDisplayMsg with ErrorActions — `ErrorAction{Key: 'r', Label: "retry"}`, `ErrorAction{Key: 'v', Label: "view logs"}`, `ErrorAction{Key: '\x1b', Label: "ignore"}`
- [ ] 8. Handle edge case: skill needs user input → card shows `◐ waiting` status, user presses Enter to attach and provide input (design doc error handling)
- [ ] 9. Run tests — GREEN

**Progress:** 0/9 steps (0%) — pending

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
- [ ] 1. Write tests for conductor state machine — RED:
  - `TestConductor_Spawn`: YOLOModeToggleMsg → conductor created with session binding, starts at current column
  - `TestConductor_RunLoop`: conductor advances session: Backlog→Design→Plan→...→Done. Each step: skill → gate → advance
  - `TestConductor_Terminate_Done`: session reaches Done column → conductor stops, logs completion
  - `TestConductor_Terminate_Disabled`: user sends YOLOModeToggleMsg again → conductor stops gracefully
  - `TestConductor_Terminate_MaxRetries`: 3 failures in same column → conductor pauses, notifies user
  - `TestConductor_Resume`: user re-enables YOLO → new conductor spawns, resumes from current column
  - `TestConductor_GateFail`: consensus gate fails → conductor pauses at current column, sends ConductorStatusMsg
  - `TestConductor_SkipColumns`: YOLOConfig.SkipColumns=["backlog"] → conductor skips Backlog, starts at Design
  - `TestConductor_Crash`: conductor goroutine panics → session stays at current column, user can re-enable YOLO (design doc error handling)
  - `TestConductor_Logging`: all gate decisions logged to `.agent-deck/conductor-log.md`
- [ ] 2. Read existing conductor.go (1903 lines) to understand current patterns for goroutine lifecycle, message passing
- [ ] 3. Implement Conductor struct: `sessionID string`, `yoloConfig YOLOConfig`, `currentColumn KanbanColumn`, `retryCount int`, `continuationID string`, `running bool`
- [ ] 4. Implement spawn: YOLOModeToggleMsg → create Conductor in goroutine, bind to session
- [ ] 5. Implement run loop: for each column → resolve skill → execute → wait for completion → run consensus gate → if pass: advance → if fail: retry or pause
- [ ] 6. Implement terminate: Done reached || YOLO disabled || retryCount >= MaxRetries
- [ ] 7. Implement resume: new conductor reads current column from storage, continues from there
- [ ] 8. Implement logging: append gate decisions to `.agent-deck/conductor-log.md` with timestamps
- [ ] 9. Run tests — GREEN

**Progress:** 0/9 steps (0%) — pending

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
- [ ] 1. Write tests for gate configurations per transition — RED:
  - `TestGateConfig_BacklogToDesign`: 2 models (gemini-3.1-pro for, gpt-5.2 against), thinking=medium, threshold=both>=7/10
  - `TestGateConfig_DesignToPlan`: 2 models, thinking=high, threshold=both>=7/10
  - `TestGateConfig_PlanToImplement`: 3 models (gemini-3.1-pro for, gpt-5.2 against, o3 neutral), thinking=high, threshold=2/3>=7/10
  - `TestGateConfig_ImplementToReview`: 2 models, thinking=high, threshold=both>=7/10
  - `TestGateConfig_ReviewToDone`: 3 models, thinking=max, threshold=UNANIMOUS>=8/10 (strictest gate)
  - `TestConsensusCallBuilder`: generates correct zen consensus params (step, step_number, total_steps, models, stances, relevant_files)
  - `TestConsensusBlinded`: step 1 has frozen proposal, steps 2+ have internal notes only
  - `TestPassFail_Unanimous`: all models pass → gate passes
  - `TestPassFail_Majority`: 2/3 pass → gate passes (for majority threshold)
  - `TestPassFail_Fail`: 1/3 pass → gate fails
  - `TestMixedEscalation`: models disagree → escalates to thinkdeep with same continuation_id
  - `TestGateTimeout`: consensus call exceeds 5 min → retry once, then pause (design doc error handling)
- [ ] 2. Define gate configs as a map: `var gateConfigs = map[[2]KanbanColumn]GateConfig{...}` with models, thinking, threshold per transition
- [ ] 3. Implement consensus call builder: creates zen consensus params. Step 1 `step` field is the evaluation question. `relevant_files` points to skill output artifacts. Temperature=0.2
- [ ] 4. Implement blinded consensus: step 1 proposal frozen, steps 2+ internal notes (not sent to models)
- [ ] 5. Implement pass/fail: parse model verdicts from consensus response, apply threshold logic
- [ ] 6. Implement escalation: on mixed results, call `mcp__zen__thinkdeep` with same `continuation_id` for context revival
- [ ] 7. Implement large output handling: if skill output >50K chars, save to temp file, pass path in `relevant_files` (NEVER inline)
- [ ] 8. Reuse `continuation_id` across all gates for same session (store in Conductor struct, pass to YOLOConfig)
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
- [ ] 1. Write tests for YOLO card indicators — RED:
  - `TestYOLOCard_RobotIcon`: card with AutomationMode="yolo" has robot icon prefix in View()
  - `TestYOLOCard_NoIcon`: card with AutomationMode="interactive" has no robot icon
  - `TestYOLOProgress_AllPending`: progress shows `BL → DE → PL → IM → RE → DO` (all pending)
  - `TestYOLOProgress_MidWay`: session at Plan → `BL ✓ → DE ✓ → PL ⚙ → IM → RE → DO`
  - `TestYOLOProgress_Complete`: session at Done → all checkmarks
  - `TestYOLOGateStatus_InProgress`: gate running → `Gate: gemini ✓ gpt-5.2 ⏳ o3 ⏳`
  - `TestYOLOGateStatus_Passed`: all models passed → `Gate: ✓ passed`
  - `TestYOLOGateStatus_Failed`: gate failed → `Gate: ✗ failed — [r]etry`
- [ ] 2. Modify KanbanCard.View(): if instance.AutomationMode == "yolo", prefix title with robot icon
- [ ] 3. Add progress bar to detail panel (when YOLO session selected): render column progress with ✓ (passed), ⚙ (current), blank (pending)
- [ ] 4. Add gate status line to detail panel: show per-model status from ConductorStatusMsg
- [ ] 5. Conductor logging already handled in Task 6.1 step 8
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
