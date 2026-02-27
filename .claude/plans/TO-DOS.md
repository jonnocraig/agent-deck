# Current Tasks

## In Progress

- [ ] **Kanban Mode Feature** (design: `docs/plans/2026-02-27-kanban-mode-design.md`)
  - [x] Brainstorm with 5-perspective agent team
  - [x] Clone and analyze 3 reference repos
  - [x] Write full design document (737 lines)
  - [x] Multi-model design review + Phase 0 adoption
  - [x] Create implementation plan — 28 tasks, 6 waves
  - [x] Cross-model plan review + TDD enhancement
  - [x] **Phase 0: Refactor** — home.go decomposed into 7 kanban_*.go files (DONE, needs commit+PR)
  - [ ] **Commit + PR for Phase 0** — commit the decomposition, create PR, merge before Wave 2
  - [ ] **Phase 1: Data Layer** — SQLite migration, Instance fields, GroupKanbanConfig
  - [ ] **Phase 2: Board UI** — KanbanBoard, KanbanCard, KanbanSidebar rendering
  - [ ] **Phase 3: Navigation** — KanbanNav 2D cursor, Tab focus, column jump, scroll
  - [ ] **Phase 4: Detail Panel** — KanbanDetail editable fields, Space toggle, edit mode
  - [ ] **Phase 5: Transitions** — TransitionEngine, skill triggers, 3-tier config, rollback
  - [ ] **Phase 6: Conductor** — YOLO mode, zen consensus gates, conductor lifecycle
  - [ ] **Phase 7: Skills** — 4 new skills (backlog, review, done, self-evolve)

## Completed This Session (2026-02-27 — Wave 1 Implementation)

- [x] Task 0.1: Analyzed home.go extraction boundaries (opus/architect)
- [x] Task 0.2: Executed home.go decomposition — 7 new files created, home.go 9031→6038 lines
- [x] Task 0.3: Verified functional equivalence — build, test, vet all pass
- [x] Wave 1 quality gates: build-check pass, vet-check pass, test-suite pass (16/16 packages)

## Completed Previous Sessions

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
