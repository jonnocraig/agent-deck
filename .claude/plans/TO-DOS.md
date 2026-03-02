# Current Tasks

## In Progress

- [ ] **Manual Testing of Kanban Mode** — user retesting with updated binary
  - Binary rebuilt to main worktree path (symlink now points to correct binary)
  - Previous test showed sidebar unresponsive + space not working — was stale binary
  - Need user to confirm fixes work after rebuild

## Completed This Session (2026-03-02 — Session 17: Bug Fixes + Card Redesign)

- [x] Bug 1: Group filtering — ctrl+k now auto-selects current group in sidebar
- [x] Bug 2: Sessions with nil KanbanColumn default to Backlog in board + findKanbanCard
- [x] Bug 3: Space key toggles detail panel (kanbanDetailOpen + case " ": handler)
- [x] Bug 4: Enter key attaches to session (fixed by nil KanbanColumn handling)
- [x] Bug 5: Card redesign — bordered boxes with tool badge + selection indicator
- [x] Fixed binary symlink issue — must build to main worktree path

## Completed Previous Sessions

- Session 16: Wave 6 — Conductor, Zen Gates, YOLO UI (8796964)
- Session 15: Wave 5 — Transitions (ef44b0f)
- Session 14: Wave 4 — Navigation & Detail (26b94dd)
- Session 13: Wave 3 — Board UI (146dc92)
- Session 12: Synced JSON agent plan with MD (6295382)
- Session 11: Wave 2 — Data Layer (9b7ea91)
- Session 10: Wave 1 — home.go decomposition (b80441d)
- Sessions 7-9: Kanban brainstorm, design doc, plan enrichment
- Sessions 1-6: Vagrant mode implementation, OAuth, config sync, MCP

## Pending

- [ ] **Commit bug fixes** — `feat(kanban): fix group filtering, card styling, key handling`
- [ ] **Push feature branch to origin** — user will manually create PR after testing
- [ ] **Detail panel rendering** — Space toggles flag but renderKanbanLayout may not render detail panel yet
- [ ] **Tab focus cycle** — should include PanelDetail when detail is open
- [ ] Extract transition handlers from home.go to kanban_transition_handlers.go
- [ ] Add YAML config caching to avoid per-call file I/O (low priority)
- [ ] `Rollback` method doesn't set `OriginalError` on `RollbackError`
- [ ] Consider adding `-race` flag to CI test runs
- [ ] Remove unused `readLogContent` helper in kanban_conductor_test.go
- [ ] Remove unused `width` parameter from `renderYOLOProgress` and `renderGateStatus`

## Blocked

- None
