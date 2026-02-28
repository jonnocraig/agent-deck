# Kanban Mode — Design Document

> Generated: 2026-02-27
> Brainstorm perspectives: Architect, Implementer, Devil's Advocate, User Advocate, Skill Designer
> Chosen approach: Full Kanban (Approach 1)
> Reference implementations: agtx, vibe-kanban, claude-vibekanban

## Summary

Add a kanban board view to agent-deck where groups display on a left sidebar and their sessions render as cards across 6 workflow columns: Backlog, Design, Plan, Implement, Review, Done. Each column maps to a skill that auto-triggers when sessions transition between columns. Sessions can progress autonomously via a Conductor agent using multi-model consensus gates (YOLO mode) or manually with user confirmation (default).

## Acceptance Criteria

- [ ] Groups render in a left sidebar; selecting a group shows its sessions as a kanban board
- [ ] 6 columns visible simultaneously: Backlog, Design, Plan, Implement, Review, Done
- [ ] Sessions render as compact cards within columns with status indicators
- [ ] `Enter` on a card attaches to the tmux session; `Space` opens an editable detail panel
- [ ] Detail panel shows: title, description, acceptance criteria, status, worktree, model, MCP servers, config
- [ ] Moving a card between columns triggers the mapped skill (configurable: confirm vs auto per-session)
- [ ] 3-tier skill-column config: system defaults, user YAML, per-group SQLite
- [ ] "All Sessions" default group shows a flat list (not kanban)
- [ ] Backward column movement allowed with confirmation
- [ ] YOLO mode: Conductor agent manages session progression with zen consensus gates
- [ ] 4 new skills created: agentic-ai-backlog, agentic-ai-review, agentic-ai-done, self-evolve
- [ ] agentic-ai-review delegates screenshots/gifs to parallel agent team (never in coordinator context)
- [ ] agentic-ai-review uses Claude Chrome extension for visual review on ALL work types
- [ ] agentic-ai-done auto-detects PR vs direct merge, handles conflicts safely
- [ ] self-evolve stores learnings in ~/.claude/learned/ as human-readable markdown
- [ ] 80%+ test coverage on all new code

## Non-Goals

- Custom column definitions beyond the 6 defaults (future work)
- Drag-and-drop with mouse (keyboard-only for MVP)
- Cross-group kanban views (aggregated board across multiple groups)
- Real-time collaboration (multi-user editing)
- Mobile/responsive terminal layouts below 80 columns

## Context & Constraints

- **Tech stack**: Go 1.24, Bubble Tea (bubbletea v1.3.10), lipgloss v1.1.0, SQLite
- **Terminal width**: All 6 columns must be visible without horizontal scroll
- **Existing UI**: internal/ui/home.go is 9031 lines — needs decomposition
- **Existing skills**: agentic-ai-brainstorm (Design), agentic-ai-plan (Plan), agentic-ai-implement (Implement)
- **User preference**: Groups on left sidebar, kanban on right, Enter=attach, Space=detail panel
- **Automation**: YOLO mode uses zen MCP consensus; default is interactive with user confirmation
- **Review skill**: Must use Claude Chrome extension; screenshots delegated to agent team

## Exploration Findings

### Perspective: Architect

**Key findings:**
- Separate `KanbanColumn` field from existing `Status` enum — workflow stage is independent from session lifecycle (running/waiting/idle)
- 3-tier config resolution: per-group SQLite > user YAML > hardcoded defaults
- TransitionEngine as a dedicated component handling validation, skill resolution, rollback on failure
- Sort order using `1000 * column_index + position` formula (from vibe-kanban)
- Message-driven architecture using Bubble Tea messages for sidebar-board communication
- Backward compatibility: existing sessions get NULL kanban_column, shown in "Unsorted" virtual column until migrated

### Perspective: Implementer

**Key findings:**
- 7.5:1 reuse-to-new code ratio — GroupTree, Instance, rendering pipeline, status polling all reusable
- ~1065 new lines across 6 new files, ~165 modified lines in 4 existing files
- No new dependencies needed — pure lipgloss with JoinHorizontal/JoinVertical
- Existing `ensureExactWidth()` function critical for column alignment
- Card height of 10 rows (matching agtx) with status indicators from existing `StatusIndicator()`
- KanbanBoard, KanbanSidebar, KanbanDetail as separate components, not embedded in home.go

### Perspective: Devil's Advocate

**Key concerns addressed:**
- **Terminal width**: At 120 cols with 20-col sidebar, each of 6 columns gets ~15 chars. Tight but workable with abbreviated headers and truncated titles. At 160+ cols it's comfortable
- **Scope**: Mitigated by strict 7-phase implementation with each phase independently shippable
- **self-evolve complexity**: Stored as simple markdown with confidence scoring; not ML, just pattern tracking with user review
- **Done skill danger**: Safety protocol with unanimous consensus gate, conflict detection before merge, rollback capability
- **Skill trigger complexity**: Failure causes rollback to previous column with clear error in detail panel

### Perspective: User Advocate

**Key findings:**
- `Enter` = attach to tmux (preserves existing mental model), `Space` = toggle detail panel, `e` = edit mode
- Tab switches focus between sidebar and board
- `h/j/k/l` for column/card navigation, `1-6` for column jump, `m` for move
- Progressive disclosure: kanban is opt-in per group, contextual tips auto-dismiss after 3 uses
- Error states always provide 2-3 action paths (retry, view logs, ignore)
- YOLO cards show robot icon and real-time gate status

### Perspective: Skill Designer

**Key findings:**
- All 4 skills follow Anthropic best practices: <500 line SKILL.md, references/ subdirectory, workflow checklists
- agentic-ai-review uses agent team: static-checks (haiku), chrome-visual (sonnet), e2e-tests (sonnet), code-review (sonnet)
- Chrome-visual agent handles ALL screenshots/gifs — coordinator never loads images into its context
- agentic-ai-done uses safety protocol: tests must pass, conflict detection, PR-first if PR exists
- self-evolve uses confidence scoring (HIGH/MEDIUM/LOW) with user review for MEDIUM patterns
- Data flow between skills: backlog.md → design-spec.md → implementation-plan.md → review-report.md → CHANGELOG.md

## Approaches Considered

### Approach 1: Full Kanban (Selected)

Build the complete kanban board vision with sidebar, 6-column board, detail panel, skill triggers, YOLO mode, and 4 new skills. Strict 7-phase implementation.

**Pros**: Full workflow visualization, skill automation, autonomous progression via Conductor
**Cons**: Larger scope, terminal width constraints at small sizes, 4 skills to build and test

### Approach 2: Status Field Only

Add a workflow status label to sessions in the existing tree view. Filter by status. No visual board.

**Pros**: 5% of complexity, works on all terminal sizes, no UI rewrite
**Cons**: No visual board, can't see all columns at once, less engaging

### Approach 3: Hybrid (Progressive Enhancement)

Start with Approach 2 as MVP, layer kanban board as optional toggle later.

**Pros**: Incremental delivery, lower risk per phase
**Cons**: Two rendering paths to maintain, longer total timeline

## Design

### Architecture & Data Flow

#### Layout

```
┌─ Sidebar ─┐│┌─────────── Kanban Board (6 columns) ──────────┐
│            │││ BL(3) │ DE(1) │ PL(0) │ IM(2) │ RE(1)│ DO(0)│
│  Groups    │││       │       │       │       │      │      │
│            │││ cards  │ cards  │       │ cards  │ card │      │
│ > proj-a   │││       │       │       │       │      │      │
│   proj-b   │││       │       │       │       │      │      │
│   archive  ││├───────────────────────────────────────────────┤
│            │││         Detail Panel (Space to toggle)        │
│            │││  Title: [fix-auth-bug____________]            │
│            │││  Desc:  [Fix the auth token refresh...]      │
│            │││  AC:    [x] Token refresh works              │
│            │││  Column: Implement  Status: running          │
│            │││  Worktree: .worktrees/fix-auth-bug           │
│            │││  Branch: feature/fix-auth-bug                │
│            │││  Tool: claude  Model: sonnet                 │
│            │││  MCPs: supabase, playwright                  │
│            │││  Auto-trigger: [x] YOLO: [ ]                │
└────────────┘│└───────────────────────────────────────────────┘
[Tab]Focus [h/l]Col [j/k]Card [m]Move [1-6]Jump [Space]Detail [?]Help
```

The board splits vertically: top = kanban columns (cards), bottom = detail panel (appears on Space). The detail panel height is ~40% of terminal height when open. Sidebar is fixed at 20 columns.

#### Data Model Changes

```go
// New type: KanbanColumn (workflow stage, separate from session Status)
type KanbanColumn string

const (
    KanbanBacklog   KanbanColumn = "backlog"
    KanbanDesign    KanbanColumn = "design"
    KanbanPlan      KanbanColumn = "plan"
    KanbanImplement KanbanColumn = "implement"
    KanbanReview    KanbanColumn = "review"
    KanbanDone      KanbanColumn = "done"
)

// Added to Instance struct
KanbanColumn    *KanbanColumn // nil = not in kanban view
KanbanSortOrder int           // position within column (1000*col + pos)
KanbanLastMoved *time.Time    // timestamp of last column change
Description     string        // editable session description
AcceptCriteria  string        // editable acceptance criteria

// Automation mode
type AutomationMode string
const (
    AutomationInteractive AutomationMode = "interactive" // default
    AutomationYOLO        AutomationMode = "yolo"        // autonomous
)

// Per-session YOLO config
type YOLOConfig struct {
    Mode              AutomationMode
    ConsensusModels   []string   // models for gate validation
    PassThreshold     string     // "unanimous" | "majority"
    PauseOnFail       bool       // pause and notify on failure
    MaxRetries        int        // per-column retry limit
    SkipColumns       []KanbanColumn
    ContinuationID    string     // zen continuation_id for context revival
}

// Skill-column mapping config
type KanbanColumnConfig struct {
    Column         KanbanColumn
    SkillName      string
    AutoTrigger    bool
    TriggerOnEnter bool
}

// Per-group kanban settings
type GroupKanbanConfig struct {
    GroupPath     string
    KanbanEnabled bool
    ColumnConfigs []KanbanColumnConfig
}
```

#### SQLite Migration

```sql
-- Migration: Add kanban fields to instances
ALTER TABLE instances ADD COLUMN kanban_column TEXT;
ALTER TABLE instances ADD COLUMN kanban_sort_order INTEGER DEFAULT 0;
ALTER TABLE instances ADD COLUMN kanban_last_moved TIMESTAMP;
ALTER TABLE instances ADD COLUMN description TEXT DEFAULT '';
ALTER TABLE instances ADD COLUMN accept_criteria TEXT DEFAULT '';
ALTER TABLE instances ADD COLUMN automation_mode TEXT DEFAULT 'interactive';
ALTER TABLE instances ADD COLUMN yolo_config_json TEXT;

-- New table: per-group kanban config
CREATE TABLE IF NOT EXISTS group_kanban_configs (
    group_path TEXT PRIMARY KEY,
    kanban_enabled BOOLEAN DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- New table: column-skill mappings per group
CREATE TABLE IF NOT EXISTS column_skill_mappings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    group_path TEXT NOT NULL,
    column_name TEXT NOT NULL,
    skill_name TEXT NOT NULL,
    auto_trigger BOOLEAN DEFAULT 0,
    trigger_on_enter BOOLEAN DEFAULT 1,
    FOREIGN KEY (group_path) REFERENCES group_kanban_configs(group_path),
    UNIQUE(group_path, column_name)
);
```

#### Message Flow (Bubble Tea)

```
User Input
    │
    ├─ Tab → FocusChangedMsg (sidebar ↔ board)
    ├─ h/l → ColumnChangedMsg (navigate columns)
    ├─ j/k → CardChangedMsg (navigate cards in column)
    ├─ Space → DetailToggleMsg (open/close detail panel)
    ├─ Enter → AttachSessionMsg (open tmux session)
    ├─ e → EditModeMsg (enable editing in detail panel)
    ├─ m → MoveInitiatedMsg → MoveConfirmedMsg (column transition)
    ├─ Shift+Y → YOLOModeToggleMsg (toggle autonomous mode)
    └─ 1-6 → ColumnJumpMsg (direct column jump)

MoveConfirmedMsg
    │
    ▼
TransitionEngine.RequestMove()
    │
    ├─ NeedsConfirm → ShowConfirmDialogMsg
    │                    │
    │                    └─ User confirms → ExecuteTransitionMsg
    │
    └─ AutoTrigger → ExecuteTransitionMsg
                        │
                        ▼
                  SkillTriggeredMsg
                        │
                        ├─ Success → BoardRefreshMsg
                        └─ Failure → RollbackMsg + ErrorDisplayMsg
```

### Component Breakdown

```
internal/ui/
├── home.go                    # Modified: kanban mode routing (~500 lines added)
├── kanban_board.go            # NEW: Board container + column layout (~400 lines)
├── kanban_card.go             # NEW: Card rendering (compact + expanded) (~200 lines)
├── kanban_sidebar.go          # NEW: Group list sidebar (~200 lines)
├── kanban_detail.go           # NEW: Editable detail panel (~350 lines)
├── kanban_nav.go              # NEW: 2D navigation state machine (~250 lines)
├── kanban_transition.go       # NEW: Column transition engine + skill triggers (~300 lines)
└── kanban_conductor.go        # NEW: YOLO mode conductor orchestration (~400 lines)

internal/session/
├── instance.go                # Modified: +6 fields (kanban, description, AC, automation)
└── groups.go                  # Reused as-is for sidebar data

internal/session/
├── storage.go                 # Modified: migration for new tables + columns, CRUD methods
└── migration.go               # Modified: new migration for kanban schema
```

| Component | Responsibility | Key State |
|-----------|---------------|-----------|
| `KanbanBoard` | 6-column layout, card delegation | `columns [6][]Instance`, `width`, `height` |
| `KanbanCard` | Compact card with status icon + title | `instance *Instance`, `selected bool` |
| `KanbanSidebar` | Group list, selection, expand/collapse | `groups []*Group`, `selectedIdx int` |
| `KanbanDetail` | Editable fields, read-only info, YOLO toggle | `instance *Instance`, `focusedField int`, `editing bool` |
| `KanbanNav` | 2D cursor, focus panel, scroll offsets | `col int`, `row int`, `focus Panel`, `scrollOffsets` |
| `TransitionEngine` | Skill resolution, validation, execution, rollback | `config`, `db *StateDB` |
| `Conductor` | YOLO mode orchestration, zen consensus gates | `sessionID`, `yoloConfig`, `continuationID` |

### Detail Panel Fields

| Field | Type | Editable | Source |
|-------|------|----------|--------|
| Title | text input | Yes | `Instance.Title` |
| Description | textarea | Yes | `Instance.Description` |
| Acceptance Criteria | textarea | Yes | `Instance.AcceptCriteria` |
| Kanban Column | dropdown | Yes (triggers move) | `Instance.KanbanColumn` |
| Status | read-only | No | `Instance.Status` (running/waiting/idle/error) |
| Tool | read-only | No | `Instance.Tool` (claude/gemini/codex/shell) |
| Worktree Path | read-only | No | `Instance.WorktreePath` |
| Worktree Branch | read-only | No | `Instance.WorktreeBranch` |
| Project Path | read-only | No | `Instance.ProjectPath` |
| Model | read-only | No | Extracted from session config |
| MCP Servers | read-only | No | `Instance.LoadedMCPNames` |
| Last Prompt | read-only | No | `Instance.LatestPrompt` |
| Auto-trigger | toggle | Yes | Per-session column config |
| YOLO Mode | toggle | Yes | `Instance.AutomationMode` |

### API / Interface Design

#### Transition Engine

```go
type TransitionEngine interface {
    RequestMove(req MoveRequest) MoveResult
    ResolveSkill(groupPath string, column KanbanColumn) (SkillMapping, error)
    IsValidMove(from, to KanbanColumn) bool
}

type MoveRequest struct {
    SessionID    string
    FromColumn   KanbanColumn
    ToColumn     KanbanColumn
    GroupPath    string
    ForceConfirm bool
}

type MoveResult struct {
    Executed     bool
    NeedsConfirm bool
    SkillName    string
    Error        error
}

type SkillMapping struct {
    SkillName   string
    AutoTrigger bool
}
```

#### 3-Tier Config Resolution

```go
func ResolveSkillForColumn(groupPath string, column KanbanColumn) (SkillMapping, error) {
    // 1. Per-group SQLite config (highest priority)
    // 2. User YAML at ~/.config/agent-deck/kanban.yaml
    // 3. System defaults (lowest priority)
}
```

Default mapping:

| Column | Skill | Default Auto-trigger |
|--------|-------|---------------------|
| Backlog | `agentic-ai-backlog` | No |
| Design | `agentic-ai-brainstorm` | No |
| Plan | `agentic-ai-plan` | No |
| Implement | `agentic-ai-implement` | No |
| Review | `agentic-ai-review` | No |
| Done | `agentic-ai-done` | No |

#### Sort Order

Formula: `sort_order = 1000 * column_index + position * 100`

- Backlog (col 0): 0, 100, 200, 300, ...
- Design (col 1): 1000, 1100, 1200, ...
- Plan (col 2): 2000, 2100, 2200, ...
- Insert between positions: `(above + below) / 2`
- Rebalance when gap < 10

#### Transition Rules

- Forward movement: any number of columns (Backlog → Implement allowed)
- Backward movement: allowed with confirmation dialog
- Skill dependency validation: warn if skipping columns (e.g., no plan exists when moving to Implement)

### Keyboard Navigation

| Key | Context | Action |
|-----|---------|--------|
| `Tab` | Any | Switch focus: sidebar ↔ board |
| `h/l` | Board | Move cursor left/right between columns |
| `j/k` | Board | Move cursor up/down within column |
| `j/k` | Sidebar | Navigate groups |
| `1-6` | Board | Jump to column (1=Backlog, 6=Done) |
| `Enter` | Board (card selected) | Attach to tmux session |
| `Space` | Board (card selected) | Toggle detail panel |
| `e` | Detail panel open | Enter edit mode |
| `Esc` | Edit mode | Exit edit mode / close detail panel |
| `m` | Board (card selected) | Move card (then h/l to select target, Enter to confirm) |
| `Shift+Y` | Detail panel open | Toggle YOLO mode |
| `Shift+A` | Detail panel open | Toggle auto-trigger for session |
| `n` | Board | Create new session in focused column |
| `d` | Board (card selected) | Delete session (with confirmation) |
| `K` | Sidebar (group selected) | Toggle kanban mode for group |

### Automation & Conductor Architecture

#### Two Modes

| Mode | Column Transitions | Quality Gates | Skill Triggers |
|------|-------------------|---------------|----------------|
| **Interactive** (default) | User confirms via dialog | User reviews report | User initiates |
| **YOLO** (autonomous) | Auto-advance on skill + gate pass | Multi-model consensus validates | Auto on column enter |

#### Conductor Agent

When YOLO mode is enabled, agent-deck spawns a Conductor agent that manages the session's lifecycle through the board.

```
┌──────────────────────────────────────────────────────┐
│                  Conductor Agent                      │
│  (Long-lived, manages one session's kanban journey)  │
│                                                       │
│  For each column transition:                         │
│  1. Trigger column skill                             │
│  2. Wait for skill completion                        │
│  3. Validate output via zen consensus                │
│  4. If consensus PASS → advance to next column       │
│  5. If consensus FAIL → pause, notify user           │
└──────────────────────────────────────────────────────┘
```

#### Zen Consensus Gate Protocol

**Rules (MANDATORY):**
1. NEVER inline skill output in `step` or `findings` — use `relevant_files` / `absolute_file_paths`
2. Step 1 `step` field is frozen as `original_proposal` — write it as the exact question all models evaluate
3. Steps 2+ `step` field = internal notes only — NOT sent to other models (blinded consensus)
4. Each (model, stance) pair must be unique
5. `total_steps` = number of models (not models + 1)
6. Reuse `continuation_id` across all gates for the same session (context revival)
7. Save outputs >50K chars to temp file before passing as file path
8. Use `thinking_mode` appropriate to gate severity (medium for early gates, max for Review→Done)
9. Keep temperature at 0.2 for gate decisions

**Gate example (Design → Plan):**

```
Step 1:
  mcp__zen__consensus:
    step: "Evaluate this design document for completeness and
      feasibility. Does it have clear acceptance criteria,
      sound architecture, and address the original requirements?
      Is it ready to proceed to the planning phase?"
    step_number: 1
    total_steps: 2
    next_step_required: true
    findings: "Design skill completed. Document covers architecture,
      components, error handling, and testing strategy."
    models:
      - model: "gemini-3.1-pro-preview"
        stance: "for"
        stance_prompt: "Evaluate design quality and readiness for planning"
      - model: "gpt-5.2"
        stance: "against"
        stance_prompt: "Challenge feasibility, find gaps, question assumptions"
    relevant_files:
      - "/absolute/path/to/docs/plans/design.md"
      - "/absolute/path/to/CLAUDE.md"

Step 2:
  mcp__zen__consensus:
    step: "Gemini approved with 8/10 confidence, noted strong AC coverage"
    step_number: 2
    total_steps: 2
    next_step_required: false
    findings: "Gemini: Design is well-structured. Ready for planning."
    continuation_id: "<from step 1 response>"
```

**Gate configurations per transition:**

| Gate | Models | Thinking | Pass Threshold | Focus |
|------|--------|----------|---------------|-------|
| Backlog → Design | gemini-3.1-pro (for), gpt-5.2 (against) | medium | Both >= 7/10 | Work items well-defined with testable AC |
| Design → Plan | gemini-3.1-pro (for), gpt-5.2 (against) | high | Both >= 7/10 | Design feasible, complete, ready for planning |
| Plan → Implement | gemini-3.1-pro (for), gpt-5.2 (against), o3 (neutral) | high | 2/3 majority >= 7/10 | Plan decomposed correctly, realistic dependencies |
| Implement → Review | gemini-3.1-pro (for), gpt-5.2 (against) | high | Both >= 7/10 | Implementation complete against plan, tests passing |
| Review → Done | gemini-3.1-pro (for), gpt-5.2 (against), o3 (neutral) | max | UNANIMOUS >= 8/10 | Safe to merge, all critical issues resolved |

**Mixed consensus escalation:**

When models disagree, use `mcp__zen__thinkdeep` with the same `continuation_id`:
- thinkdeep sees full consensus history via context revival
- Recommends: advance (with warning), retry skill (with feedback), or pause (notify user)

**Alternative gate type — Claude Agent Team:**

Instead of zen consensus, users can configure a Claude Code agent team:
- Agent 1 (sonnet): "Senior Engineer" — code quality, patterns
- Agent 2 (sonnet): "Security Reviewer" — vulnerabilities, auth
- Agent 3 (haiku): "Build Validator" — compile, test, lint

Configurable per session: `gate_type: "zen-consensus" | "agent-team" | "both"`

#### Conductor Lifecycle

1. **Spawn**: User enables YOLO (Shift+Y) → agent-deck spawns Conductor via Task tool
2. **Run Loop**: For each column: execute skill → wait → run consensus gate → advance or pause
3. **Terminate**: Session reaches Done, user disables YOLO, or max retries exhausted
4. **Resume**: User fixes issues → re-enables YOLO → Conductor resumes from current column
5. **Logging**: All gate decisions logged to `.agent-deck/conductor-log.md`

#### YOLO UI Indicators

Board card: `🤖` icon prefix on YOLO-enabled cards
Detail panel: Progress bar showing `Backlog ✓ → Design ✓ → Plan ⚙ → Implement → Review → Done`
Gate status: `Gate: gemini ✓ gpt-5.2 ⏳ o3 ⏳`

### Error Handling & Edge Cases

| Scenario | Handling |
|----------|----------|
| Skill fails on column move | Rollback: session returns to previous column. Error shown in detail panel with `[r]etry [v]iew logs [Esc]ignore` |
| Merge conflict in Done | Block merge, show conflict files in detail panel, offer `[o]pen editor [s]kip merge` |
| Terminal < 80 cols wide | Degrade: hide sidebar, show only 3 columns (current + neighbors) with scroll indicators |
| Session has no kanban column | Default to Backlog when kanban first enabled for its group |
| User moves session backward | Allow with confirmation: "Move backward to {column}? Skill won't re-run." |
| Skill needs user input | Card shows `◐ waiting` status, user presses Enter to attach and provide input |
| Column has 20+ cards | Vertical scroll within column with `▲ N above` / `▼ N below` indicators |
| Empty group with kanban | Show empty board with "Press `n` to create a session" hint |
| Consensus gate timeout (>5 min) | Retry once, then pause session and notify user |
| YOLO conductor crashes | Session stays at current column, user can re-enable YOLO to respawn conductor |
| Skill output exceeds 50K chars | Save to temp file, pass file path to zen consensus (never inline) |

### Testing Strategy

| Level | Scope | Tool | Coverage Target |
|-------|-------|------|----------------|
| Unit | KanbanNav 2D cursor (left/right/up/down/jump/scroll) | `go test` | 95% |
| Unit | TransitionEngine (validation, skill resolution, rollback) | `go test` | 90% |
| Unit | Sort order calculation (insert, rebalance) | `go test` | 95% |
| Unit | Card rendering (lipgloss output) | `go test` with golden files | 80% |
| Unit | Detail panel field mapping | `go test` | 85% |
| Integration | SQLite migration + CRUD with kanban fields | `go test` with temp DB | 90% |
| Integration | Skill trigger via tmux send-keys | `go test` with mock tmux | 80% |
| Integration | Config resolution (3-tier) | `go test` | 90% |
| E2E | Full workflow: create group → enable kanban → create session → move through columns | Manual + script | N/A |
| E2E | YOLO mode: session progresses autonomously through all columns | Manual | N/A |

### New Skills

#### agentic-ai-backlog (Backlog column)

```yaml
---
name: agentic-ai-backlog
description: Automated backlog grooming - analyze codebase for improvements, gaps, and
  technical debt, then generate prioritized work items with testable acceptance criteria.
  Use when session enters Backlog column or user requests codebase analysis.
argument-hint: "[focus-area]"
---
```

**Workflow:**
1. Scan codebase structure (architecture, dependencies, organization)
2. Detect technical debt (TODO/FIXME, code smells, unused deps, outdated patterns)
3. Identify missing tests (coverage gaps, untested critical paths)
4. Security audit (known vulnerabilities, outdated deps, missing validations)
5. Performance analysis (N+1 queries, missing indexes, bundle size)
6. Documentation gaps (undocumented APIs, stale docs)
7. Prioritize using RICE framework (Reach, Impact, Confidence, Effort)
8. Output to `.agent-deck/backlog.md`

**Reference files:** `debt-detection-patterns.md`, `security-checklist.md`, `performance-metrics.md`, `prioritization-matrix.md`

#### agentic-ai-review (Review column)

```yaml
---
name: agentic-ai-review
description: Comprehensive code review with agent team - validate implementation against
  acceptance criteria, visual review with Claude Chrome extension, run E2E tests.
  Delegates screenshots and visual capture to parallel agents to preserve coordinator context.
  Use when session moves to Review column or user requests implementation review.
argument-hint: "[acceptance-criteria-file]"
---
```

**Agent Team Architecture (MANDATORY — coordinator never takes screenshots):**

```
Review Coordinator (main session)
    │ Spawns in parallel:
    ├─ [haiku]  static-checks: lint, types, security scan
    ├─ [sonnet] chrome-visual: ALL screenshots, gifs, Lighthouse, console checks
    ├─ [sonnet] e2e-tests: Chrome E2E first, then Playwright E2E
    └─ [sonnet] code-review: standards, patterns, AC verification
```

**Wave execution:**
- Wave 1 (parallel): static-checks + chrome-visual
- Wave 2 (parallel, after wave 1): e2e-tests + code-review
- Coordinator synthesizes reports (text summaries only, never raw images)

**Chrome extension usage (ALL work types):**
- UI work: full visual regression (3 breakpoints), design spec comparison, accessibility audit
- Backend work: API response inspection via DevTools, admin dashboard check
- All work: console error check, Lighthouse audit, clean-state screenshots
- Screenshots saved to `.agent-deck/review-screenshots/` (referenced by path, never loaded into coordinator)

**Quality gate:** BLOCK move to Done if any critical issues found.

**Reference files:** `agent-team-guidelines.md`, `chrome-review-guide.md`, `ui-detection-rules.md`, `e2e-test-requirements.md`, `quality-gates.md`

#### agentic-ai-done (Done column)

```yaml
---
name: agentic-ai-done
description: Finalize and merge work - handle git merge (PR or direct), update CHANGELOG.md
  with version numbers, tag release, push to origin, clean up worktree. Auto-detects
  merge strategy based on existing PR. Use when session moves to Done column.
argument-hint: "[version-type]"
---
```

**Safety protocol (ALL checks must pass before merge):**
1. `git status` is clean, all tests pass
2. Base branch (main/master) is up to date: `git fetch origin`
3. Conflict detection: `git merge-tree` dry run
4. If PR exists: all CI checks passed, required reviews approved
5. If ANY check fails: BLOCK merge and report

**Merge strategy auto-detection:**
- If PR exists for branch → merge via `gh pr merge` (squash/rebase per repo settings)
- If no PR → direct `git merge --no-ff` to main/master

**Version determination (semver from commit messages):**
- `BREAKING CHANGE:` or `!:` → major bump
- `feat:` → minor bump
- `fix:`, `chore:`, `docs:` → patch bump

**Post-merge:**
- Update CHANGELOG.md (Keep a Changelog format)
- Create annotated git tag
- Push to origin with `--follow-tags`
- Clean up worktree if temporary
- Broadcast completion summary

**Reference files:** `merge-conflict-resolution.md`, `semver-rules.md`, `changelog-format.md`, `merge-strategies.md`

#### self-evolve (User-level, background)

```yaml
---
name: self-evolve
description: Learn and improve - track repeated patterns, behaviors, and lessons across
  sessions to suggest process improvements. Stores learnings as human-readable markdown
  in ~/.claude/learned/ with confidence scoring. Non-destructive, suggestions only.
argument-hint: "[observation-type]"
---
```

**Storage structure:**
```
~/.claude/learned/
├── codebase-patterns/          # Project-specific conventions
├── user-behaviors/             # Kanban workflow habits, skill usage
├── testing-patterns/           # Reusable test templates
├── learned-behaviors/          # Build error solutions, optimizations
└── meta/
    ├── learning-index.md       # Master index
    ├── confidence-scores.md    # Pattern confidence tracking
    └── review-log.md           # User review history
```

**Confidence scoring:**
- HIGH (0.8-1.0): Pattern seen 5+ times, never failed, aligns with conventions → auto-apply
- MEDIUM (0.5-0.79): Seen 3-4 times, occasional failures → suggest to user, require confirmation
- LOW (0.0-0.49): Seen 1-2 times, experimental → store for review, don't apply

**Bad pattern prevention:**
1. Cross-validate against `~/.claude/rules/` — must not conflict
2. Cross-validate against project `CLAUDE.md` — must align
3. Run through anti-pattern detector
4. User review triggered: on confidence drop, on contradiction, every 30 days, or manually

**Reference files:** `pattern-recognition.md`, `learning-categories.md`, `anti-pattern-detection.md`, `suggestion-templates.md`

### Skill Data Flow

```
Backlog                Design               Plan                 Implement
agentic-ai-backlog → agentic-ai-brainstorm → agentic-ai-plan → agentic-ai-implement
.agent-deck/           docs/plans/            .claude/plans/       Git commits +
backlog.md             *-design.md            agent-teams/         implementation-
                                              *-agent-plan.md      notes.md
        ↓                                                              ↓
     Review                                                         Done
     agentic-ai-review ──────────────────────────────────→ agentic-ai-done
     .agent-deck/                                           CHANGELOG.md
     review-report.md                                       git tag
     review-screenshots/                                    merged to main
```

Each skill reads the previous column's output artifacts. The Conductor (YOLO mode) or user (interactive mode) ensures artifacts exist before advancing.

## Implementation Phases

| Phase | Scope | Dependencies | New Files | Modified Files |
|-------|-------|-------------|-----------|----------------|
| **0: Refactor** | Decompose home.go into kanban component files. Extract existing logic only — zero new features. App must be functionally identical after this phase. Separate PR, merged and stabilized before Phase 1. | None | 7 (kanban_board.go, kanban_card.go, kanban_sidebar.go, kanban_detail.go, kanban_nav.go, kanban_transition.go, kanban_conductor.go) | 1 (home.go) |
| **1: Data Layer** | SQLite migration, Instance fields, persistence, GroupKanbanConfig | Phase 0 | 0 | 3 (storage.go, migration.go, instance.go) |
| **2: Board UI** | KanbanBoard, KanbanCard, KanbanSidebar rendering | Phase 1 | 0 (files created in Phase 0) | 3 (kanban_board.go, kanban_card.go, kanban_sidebar.go) |
| **3: Navigation** | KanbanNav 2D cursor, Tab focus, column jump, scroll | Phase 2 | 0 (file created in Phase 0) | 1 (kanban_nav.go) |
| **4: Detail Panel** | KanbanDetail editable fields, Space toggle, edit mode | Phase 3 | 0 (file created in Phase 0) | 1 (kanban_detail.go) |
| **5: Transitions** | TransitionEngine, skill triggers, 3-tier config, rollback | Phase 4 | 0 (file created in Phase 0) | 1 (kanban_transition.go) |
| **6: Conductor** | YOLO mode, zen consensus gates, conductor lifecycle | Phase 5 | 0 (file created in Phase 0) | 1 (kanban_conductor.go) |
| **7: Skills** | 4 new skills (backlog, review, done, self-evolve) | Phase 5 | 4 skill directories | 0 |

## Next Steps

- [ ] Create implementation plan (use `agentic-ai-plan`)
- [ ] Set up worktree for implementation (already on feature/KanbanMode)
- [ ] Execute plan with agent team (use `agentic-ai-implement`)
- [ ] Phase 0: Decompose home.go (separate PR, no new features)
- [ ] Phase 1: Data layer implementation
- [ ] Phase 2-4: UI components
- [ ] Phase 5-6: Transition engine + conductor
- [ ] Phase 7: New skills
