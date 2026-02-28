# Current Tasks

## In Progress

- [ ] **Kanban Mode Feature** (design: `docs/plans/2026-02-27-kanban-mode-design.md`)
  - [x] Brainstorm with 5-perspective agent team
  - [x] Clone and analyze 3 reference repos
  - [x] Write full design document (737 lines)
  - [x] Multi-model design review + Phase 0 adoption
  - [x] Create implementation plan — 28 tasks, 6 waves
  - [x] Cross-model plan review + TDD enhancement
  - [x] **Phase 0: Refactor** — home.go decomposed into 7 kanban_*.go files (b80441d)
  - [x] **Phase 1: Data Layer** — types, schema v2, CRUD, messages, sort order (9b7ea91)
  - [x] **Phase 2: Board UI** — KanbanBoard, KanbanCard, KanbanSidebar rendering (146dc92)
  - [x] **Phase 3: Navigation** — KanbanNav 2D cursor, scroll, n/d/m card interactions (26b94dd)
  - [x] **Phase 4: Detail Panel** — KanbanDetail panel, edit mode (26b94dd)
  - [x] **Phase 5: Transitions** — TransitionEngine, skill triggers, 3-tier config, rollback (**uncommitted**)
  - [ ] **Phase 6: Conductor** — YOLO mode, zen consensus gates, conductor lifecycle (**NEXT**)
  - [ ] **Phase 7: Skills** — 4 new skills (backlog, review, done, self-evolve)

## Completed This Session (2026-02-28 — Wave 5 Implementation)

- [x] Task 5.1: TransitionEngine interface — IsValidMove, RequestMove, error types, TransitionStorage
- [x] Task 5.2: 3-tier config resolution — SQLite > YAML > defaults, KanbanYAMLConfig, functional options
- [x] Task 5.3: Skill triggers + rollback — resolveAutoTriggerSkill, Rollback method, message handlers in home.go
- [x] Quality gates: code-review + security-review (fixed 2 CRITICAL, 3 HIGH, 4 MEDIUM issues)
  - Fixed nil pointer deref in handleSkillCompleted
  - Fixed partial write without rollback in RequestMove (extracted executeStorageMove)
  - Fixed tmux command injection via ValidateSkillName regex
  - Added SkillName to SkillCompletedMsg
  - Added YAML config file size limit (1MB)
  - Added sort order error test + 13 skill name validation tests
- [x] Wave 5 NOT YET COMMITTED — needs commit

## Completed Previous Sessions

- Session 14: Wave 4 — Navigation & Detail (26b94dd)
- Session 13: Wave 3 — Board UI (146dc92)
- Session 12: Synced JSON agent plan with MD (6295382)
- Session 11: Wave 2 — Data Layer (9b7ea91)
- Session 10: Wave 1 — home.go decomposition (b80441d)
- Sessions 7-9: Kanban brainstorm, design doc, plan enrichment
- Sessions 1-6: Vagrant mode implementation, OAuth, config sync, MCP

## Pending

- [ ] **Execute Wave 6: Automation & Skills** (7 tasks)
  - Task 6.1: Conductor lifecycle (kanban_conductor.go)
  - Task 6.2: Zen consensus gate protocol (kanban_conductor.go)
  - Task 6.3: YOLO UI indicators (kanban_card.go, kanban_conductor.go)
  - Tasks 7.1-7.4: 4 new skills (backlog, review, done, self-evolve)
- [ ] Extract transition handlers from home.go to kanban_transition_handlers.go (code review suggestion)
- [ ] Add YAML config caching to avoid per-call file I/O (low priority)
- [ ] **Implement user profile sync** (design: `.claude/plans/2026-02-22-user-profile-sync-design.md`)
- [ ] Delete Vagrantfile + `.vagrant/` for fresh VM test
- [ ] Increase default VM RAM from 4GB to 16GB (`vagrantfile.go:162`)
- [ ] Upgrade Node.js in VM from 18.x to 20.x

## Blocked

- None
