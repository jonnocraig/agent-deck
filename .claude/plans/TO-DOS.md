# Current Tasks

## In Progress

- [ ] Execute Wave 5 (UI Integration) — 4 tasks, all parallel

## Completed This Session (Session 8 — Wave 4 Implementation)

### Wave 3 Quality Gates — Re-run and Passed
- [x] `go test ./internal/vagrant/` — PASS
- [x] `go test ./internal/session/` — PASS
- [x] `go build ./...` — PASS

### Wave 4: Instance Lifecycle — 5/5 complete
- [x] task-4.1 (opus): Vagrant lifecycle hooks — applyVagrantWrapper, stopVagrant/destroyVagrant goroutines, waitForVagrantOp, IsVagrantMode(), import cycle resolution via bridge adapter pattern
- [x] task-4.2 (sonnet): Restart flow — restartVagrantSession() with state machine for running/suspended/aborted/not_created
- [x] task-4.3 (sonnet): Health check in UpdateStatus() — periodic VM health checks with configurable interval
- [x] task-4.4 (haiku): Config drift in applyVagrantWrapper — re-provision on drift during Start
- [x] task-4.5 (implemented in main session): Multi-session share/separate flow + fork inheritance + 4 new tests

### Import Cycle Resolution
- [x] Created `internal/session/vagrant_iface.go` — VagrantVM interface + VMHealthResult + VagrantProviderFactory
- [x] Created `internal/vagrant/bridge.go` — vagrantVMAdapter that bridges Manager → VagrantVM interface
- [x] Removed direct `vagrant` import from `instance.go`

### Wave 4 Quality Gates — PASSED
- [x] `go build ./...` — PASS
- [x] `go vet ./...` — PASS
- [x] `go test ./internal/session/` — PASS
- [x] `go test ./internal/vagrant/` — PASS

## Completed Previous Sessions

- [x] Multi-perspective brainstorm (Architect, Implementer, Devil's Advocate, Security Analyst)
- [x] Wrote design document: `docs/plans/2026-02-14-vagrant-mode-design.md`
- [x] Multi-model design review with Gemini 3 Pro + GPT 5.1 Codex — 22 issues amended
- [x] Multi-model consensus review of plan with Gemini 3 Pro + GPT 5.2
- [x] Created implementation plan using `agentic-ai-plan` skill — 34 tasks, 6 waves
- [x] Rebased feature/vagrant branch from feature/teammate-mode onto main (v0.16.0)
- [x] Wave 1 (Foundation): 4/4 complete
- [x] Wave 2 (Core Infrastructure): 8/8 complete
- [x] Wave 3 (MCP & Skills): 4/4 complete

## Pending — Waves 5-6 (14 tasks remaining)

### Wave 5: UI Integration (4 tasks, all parallel)
- [ ] task-5.1 (sonnet): Vagrant mode checkbox in TUI — modify claudeoptions.go (NOTE: checkbox already added in Wave 2, may need verification/updates only)
- [ ] task-5.2 (sonnet): Boot progress display in session list — modify sessionlist.go (doesn't exist yet, check list.go)
- [ ] task-5.3 (sonnet): Stale VM cleanup dialog (Shift+D) — create cleanup_dialog.go, modify app.go (app.go doesn't exist, check home.go)
- [ ] task-5.4 (haiku): Apple Silicon kext detection in TUI — modify instance.go

### Wave 6: Testing & Hardening (10 tasks, all parallel)
- [ ] task-6.1 (haiku): MockVagrantProvider + interface check
- [ ] task-6.2 (sonnet): Manager unit tests
- [ ] task-6.3 (haiku): MCP unit tests
- [ ] task-6.4 (haiku): Health check unit tests
- [ ] task-6.5 (sonnet): Instance lifecycle unit tests with mock
- [ ] task-6.6 (haiku): Credential guard tests
- [ ] task-6.7 (haiku): Config drift tests
- [ ] task-6.8 (haiku): Tips tests
- [ ] task-6.9 (sonnet): UI tests
- [ ] task-6.10 (haiku): Documentation updates

## Blocked

- None

## Known Issues

- task-5.1 may be partially done already (TUI checkbox added in Wave 2 as part of task-5.1 scheduled early)
- task-5.3 lists `cleanup_dialog.go` as "modify" but it doesn't exist — should be "create"
- task-5.3 references `app.go` which doesn't exist — the main TUI file is `home.go`
- task-5.2 references `sessionlist.go` which doesn't exist — session list rendering is in `list.go`
- Wave 6 tasks may find many tests already exist from earlier waves (each file got `_test.go` in Wave 2-3)
