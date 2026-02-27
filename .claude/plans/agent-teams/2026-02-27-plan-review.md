# Plan Review — Design ↔ Agent Plan Cross-Reference

> Reviewers: Gemini 2.5 Pro (via clink) + Claude Opus 4.6
> Date: 2026-02-27
> Files reviewed: design doc (740 lines) + agent plan (877 lines)

---

## CRITICAL

### 1. Missing Task: Bubble Tea Message Type Definitions

**Reference:** Design doc lines 243-273 (Message Flow section)
**Impact:** The entire UI communication model depends on ~11 specific `tea.Msg` types:
- `FocusChangedMsg`, `ColumnChangedMsg`, `CardChangedMsg`, `DetailToggleMsg`
- `AttachSessionMsg`, `EditModeMsg`, `MoveInitiatedMsg`, `MoveConfirmedMsg`
- `YOLOModeToggleMsg`, `ColumnJumpMsg`, `SkillTriggeredMsg`
- Plus downstream: `BoardRefreshMsg`, `RollbackMsg`, `ErrorDisplayMsg`, `ShowConfirmDialogMsg`, `ExecuteTransitionMsg`

No task defines these. LLMs implementing Tasks 2.1-5.3 will each independently invent inconsistent message structs, causing integration failures.

**Fix:** Add Task 1.4 (Wave 2) — "Define all Bubble Tea message types in `internal/ui/kanban_messages.go`". List every message struct with fields. This must precede all UI tasks.

---

## HIGH

### 2. Missing Keyboard Shortcuts: `n`, `d`, `K`, `m`

**Reference:** Design doc lines 395-413 (Keyboard Navigation table)
**Impact:** Four keyboard interactions from the design have no implementation task:
- `n` — Create new session in focused column (line 410)
- `d` — Delete session with confirmation (line 411)
- `K` — Toggle kanban mode for group (line 412)
- `m` — Move card: press m, then h/l to select target, Enter to confirm (line 407)

Task 5.3 mentions rollback on move failure, but the actual `m` key interaction flow (initiate → target selection → confirm) is unplanned.

**Fix:** Add steps to existing tasks:
- `n` and `d`: Add to Task 3.1 (KanbanNav) or create Task 3.3
- `K`: Add to Task 2.4 (home.go integration)
- `m`: Add to Task 5.3 (skill triggers) — implement the full move UX flow

### 3. Missing Sort Order Logic

**Reference:** Design doc lines 379-387 (Sort Order section)
**Impact:** The design specifies `sort_order = 1000 * column_index + position * 100` with insert-between logic `(above + below) / 2` and rebalance when gap < 10. Task 1.1 adds the `KanbanSortOrder` field, but no task implements the calculation, insertion, or rebalancing algorithms.

**Fix:** Add Task 1.4b (Wave 2) — "Implement sort order calculation, insertion, and rebalancing" with tests for:
- Initial assignment on kanban enable
- Insert between two existing cards
- Rebalance when gap < 10
- Sort order update on column move

### 4. E2E and Integration Testing Underspecified

**Reference:** Design doc lines 545-558 (Testing Strategy table)
**Impact:** The design mandates specific integration tests:
- "Skill trigger via tmux send-keys with mock tmux" (80% coverage)
- "Config resolution 3-tier" (90% coverage)
- "SQLite migration + CRUD with temp DB" (90% coverage)
- Two E2E flows (manual + script)

The plan's TDD steps are formulaic ("Write tests for X — RED") without specifying:
- What assertions to make
- What edge cases to cover
- What mocks/fixtures are needed

**Fix:** Enhance every RED step with specific test case descriptions. Add Wave 7 "QA & Release" with integration test tasks.

### 5. Design Doc Has Wrong File Path

**Reference:** Design doc lines 292-294 and 722
**Impact:** Design references `internal/statedb/statedb.go` but actual files are `internal/session/storage.go` (706 lines) and `internal/session/migration.go` (230 lines). The agent plan correctly uses the right paths, but an LLM reading the design doc cross-reference will be confused.

**Fix:** Update design doc: replace `internal/statedb/statedb.go` with `internal/session/storage.go` and `internal/session/migration.go`.

---

## MEDIUM

### 6. "All Sessions" Flat List Behavior Missing

**Reference:** Design doc line 21 (Acceptance Criteria)
**Impact:** AC explicitly states: "All Sessions" default group shows a flat list (not kanban). No task handles this conditional rendering. Task 2.4 wires kanban into home.go but doesn't account for the "All Sessions" exception.

**Fix:** Add step to Task 2.4: "Implement conditional rendering — 'All Sessions' group always shows flat list view, other groups show kanban when enabled."

### 7. Graceful Degradation Not Specified in Tasks

**Reference:** Design doc lines 530-543 (Error Handling table)
**Impact:** The design has 12 specific error/edge case scenarios with explicit handling. These are not copied into the relevant task steps. An LLM implementing Task 2.1 won't know to hide the sidebar at <80 cols unless it reads the error table separately.

**Fix:** For each task, embed the relevant error handling scenario directly in the steps:
- Task 2.1: "Terminal < 80 cols: hide sidebar, show 3 columns with scroll indicators"
- Task 3.1: "Empty columns: skip when navigating h/l"
- Task 5.3: "Skill fails: rollback to previous column, show `[r]etry [v]iew logs [Esc]ignore`"

### 8. TDD Steps Lack Specificity for LLM Implementation

**Reference:** All task steps
**Impact:** Every task follows the pattern "Write tests for X — RED" without specifying WHAT to test. For example, Task 1.1 step 1 says "Write tests for KanbanColumn type (validation, string conversion)" but doesn't specify:
- Valid inputs: "backlog", "design", "plan", "implement", "review", "done"
- Invalid inputs: "", "unknown", "BACKLOG" (case sensitivity?)
- Conversion: KanbanColumn → string, string → KanbanColumn
- Edge: nil pointer handling for *KanbanColumn fields

**Fix:** Each RED step should list 3-5 specific test cases with inputs and expected outputs. Example:
```
Write tests for KanbanColumn type — RED:
- TestKanbanColumnValid: all 6 values are valid
- TestKanbanColumnInvalid: empty string, "unknown", "BACKLOG" return error
- TestKanbanColumnString: each constant returns its lowercase string
- TestParseKanbanColumn: "backlog" → KanbanBacklog, "invalid" → error
- TestKanbanColumnNil: nil *KanbanColumn handled without panic
```

---

## LOW

### 9. YAML Config Schema Not Defined

**Reference:** Design doc line 362 (3-Tier Config Resolution)
**Impact:** Task 5.2 parses `~/.config/agent-deck/kanban.yaml` but no task defines what this file looks like. The implementing LLM will invent a schema.

**Fix:** Add a schema definition step to Task 5.2 (or as a sub-task). Define:
```yaml
# ~/.config/agent-deck/kanban.yaml
columns:
  backlog:
    skill: agentic-ai-backlog
    auto_trigger: false
  design:
    skill: agentic-ai-brainstorm
    auto_trigger: false
  # ...
```

### 10. No Golden File Test Helper

**Reference:** Agent plan Task 2.1 mentions "golden files"
**Impact:** Multiple UI tasks reference golden file testing but no task creates the test infrastructure.

**Fix:** Add step to Task 0.3 (Verify equivalence): "Create golden file test helper in `internal/ui/testutil_test.go` with `updateGolden` flag pattern."

### 11. No Custom Error Types

**Impact:** Multiple tasks need to handle errors (skill failures, merge conflicts, validation errors) but no shared error types are defined.

**Fix:** Define error types in the relevant files as they're needed — no separate task required. Add a note to Task 5.1 to define `TransitionError`, `SkillFailedError`, `RollbackError` types.

---

## Summary of Required Plan Changes

| # | Severity | Change | Where |
|---|----------|--------|-------|
| 1 | CRITICAL | Add Task 1.4: Define all `tea.Msg` types in `kanban_messages.go` | Wave 2 |
| 2 | HIGH | Add `n`, `d`, `K`, `m` key handlers to tasks 2.4, 3.1, 5.3 | Waves 3-5 |
| 3 | HIGH | Add sort order logic task (calculation + rebalancing) | Wave 2 |
| 4 | HIGH | Enhance all TDD RED steps with specific test cases | All tasks |
| 5 | HIGH | Fix design doc: statedb.go → storage.go/migration.go | Design doc |
| 6 | MEDIUM | Add "All Sessions" flat list conditional to Task 2.4 | Wave 3 |
| 7 | MEDIUM | Embed error handling scenarios in relevant task steps | Waves 3-5 |
| 8 | MEDIUM | Make TDD steps more specific (inputs, outputs, edge cases) | All tasks |
| 9 | LOW | Define kanban.yaml schema in Task 5.2 | Wave 5 |
| 10 | LOW | Add golden file helper to Task 0.3 | Wave 1 |
| 11 | LOW | Define error types in Task 5.1 | Wave 5 |
