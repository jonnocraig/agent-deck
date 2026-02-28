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
  - [ ] **Phase 3: Navigation** — KanbanNav 2D cursor, Tab focus, column jump, scroll (**NEXT**)
  - [ ] **Phase 4: Detail Panel** — KanbanDetail editable fields, Space toggle, edit mode
  - [ ] **Phase 5: Transitions** — TransitionEngine, skill triggers, 3-tier config, rollback
  - [ ] **Phase 6: Conductor** — YOLO mode, zen consensus gates, conductor lifecycle
  - [ ] **Phase 7: Skills** — 4 new skills (backlog, review, done, self-evolve)

## Completed This Session (2026-02-28 — Wave 3 Implementation)

- [x] Task 2.1: KanbanBoard 6-column layout, responsive breakpoints, empty state (6 tests)
- [x] Task 2.2: KanbanCard compact rendering, status icons, YOLO badge, truncation (10 tests)
- [x] Task 2.3: KanbanSidebar group list, [K] indicator, j/k nav, fixed 20-col (8 tests)
- [x] Task 2.4: Integrate kanban into home.go — ctrl+k toggle, View/Update routing (7 tests)
- [x] Wave 3 quality gates: build pass, vet pass, test-suite pass (16/16 packages, 31 kanban tests)
- [x] Wave 3 committed: 146dc92

## Completed Previous Session (2026-02-27 — Wave 2 Implementation)

- [x] Task 1.1: Define KanbanColumn type + Instance fields (7 tests pass)
- [x] Task 1.2: Create SQLite migrations — schema v2, 7 ALTER TABLEs, 2 new tables (8 tests pass)
- [x] Task 1.3: Add CRUD methods + GroupKanbanConfig persistence (17 tests pass)
- [x] Task 1.4: Define Bubble Tea message types — 22 messages, FocusPanel enum, ErrorAction (4 tests pass)
- [x] Task 1.5: Implement sort order calculation + rebalancing (8 tests pass)
- [x] Wave 2 quality gates: build-check pass, vet-check pass, test-suite pass (16/16 packages)
- [x] Wave 2 committed: 9b7ea91

## Completed Previous Sessions

- [x] Wave 1 implementation (session 10) — home.go decomposition, committed b80441d
- [x] Plan review + TDD enhancement (session 9)
- [x] Design review + plan enrichment (session 8)
- [x] Full kanban mode design brainstorm (session 7)
- [x] Design document written and committed (97a87a4)
- [x] User profile sync design
- [x] Full vagrant mode implementation (33 files, 8051 lines)

## Pending

- [ ] **Implement user profile sync** (design: `.claude/plans/2026-02-22-user-profile-sync-design.md`)
- [ ] Delete Vagrantfile + `.vagrant/` for fresh VM test
- [ ] Increase default VM RAM from 4GB to 16GB (`vagrantfile.go:162`)
- [ ] Upgrade Node.js in VM from 18.x to 20.x

## Blocked

- None
