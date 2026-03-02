# Current Tasks

## In Progress

- [ ] **Manual Testing of Kanban Mode** — ctrl+k to enter kanban mode, need a group with kanban enabled
  - Feature branch merged into main worktree locally (merge commit `1dc8cae`)
  - Binary rebuilt at `/Users/jon_ec/code/research/agent-deck/build/agent-deck`
  - Symlink at `/opt/homebrew/bin/agent-deck` points to that binary
  - User could not see kanban board — likely because no group has `kanban_enabled=1` in SQLite
  - Need to enable kanban for a group to populate the board

## Completed This Session (2026-02-28 — Wave 6 + Merge)

- [x] Wave 6 tasks 6.1-6.3 (Go code): Conductor lifecycle, zen consensus gates, YOLO UI indicators
- [x] Wave 6 tasks 7.1-7.4 (Skills): Skipped project-level copies — skills already exist as user-level
- [x] Quality gates: code-review + security-review (fixed 3 CRITICAL, 4 HIGH, 2 MEDIUM issues)
  - Fixed race conditions on `currentColumn`, `retryCount`, `ContinuationID` (mutex + getter)
  - Fixed variable shadowing in `ProgressColumns` (loop var `c` -> `col`)
  - Fixed `truncateTitle` byte-vs-rune truncation
  - Added `backoffDelay()` with linear backoff for retry loops
  - Added `sanitizeLogField()` / `sanitizePromptField()` for log + MCP prompt injection
- [x] Wave 6 committed as `8796964`
- [x] Feature branch merged into main worktree (`1dc8cae`)
  - Resolved conflict: `home.go` — accepted feature branch version (extracted code)
  - Fixed `ShowDeleteSession` — 4th param `vagrantMode` added to 2 call sites
- [x] Binary rebuilt from main worktree for manual testing

## Completed Previous Sessions

- Session 15: Wave 5 — Transitions (ef44b0f)
- Session 14: Wave 4 — Navigation & Detail (26b94dd)
- Session 13: Wave 3 — Board UI (146dc92)
- Session 12: Synced JSON agent plan with MD (6295382)
- Session 11: Wave 2 — Data Layer (9b7ea91)
- Session 10: Wave 1 — home.go decomposition (b80441d)
- Sessions 7-9: Kanban brainstorm, design doc, plan enrichment
- Sessions 1-6: Vagrant mode implementation, OAuth, config sync, MCP

## Pending

- [ ] **Enable kanban for a test group** — run SQL to set `kanban_enabled=1` for a group
- [ ] **Push feature branch to origin** — user will manually create PR after testing
- [ ] **Uncommitted change on main worktree** — `home.go` has group dialog kanban enablement (not from feature branch)
- [ ] Extract transition handlers from home.go to kanban_transition_handlers.go (code review suggestion)
- [ ] Add YAML config caching to avoid per-call file I/O (low priority)
- [ ] `Rollback` method doesn't set `OriginalError` on `RollbackError`
- [ ] Consider adding `-race` flag to CI test runs
- [ ] Remove unused `readLogContent` helper in kanban_conductor_test.go
- [ ] Remove unused `width` parameter from `renderYOLOProgress` and `renderGateStatus`

## Blocked

- None
