# Current Tasks

## Pending — Wave 7: Critical Bug Fixes (4 tasks)

### CRITICAL — Must fix before merge

- [ ] task-7.1 (golang-pro): Race condition on `vmOpDone` channel — `stopVagrant()`, `destroyVagrant()`, and `waitForVagrantOp()` all read/write `i.vmOpDone` without mutex protection. Only `vmOpInFlight` uses atomic. Fix: protect all `vmOpDone` access with `i.mu`, or capture channel reference under lock before select.
  - Files: `internal/session/instance.go:1245-1246, 1275-1276, 1302-1311`

- [ ] task-7.2 (golang-pro): Race condition on `cleanShutdown` — written from goroutine in `stopVagrant()` (line 1265) without lock, read elsewhere under `i.mu.Lock()` (line 1748). Fix: convert to `atomic.Bool` or wrap write in `i.mu.Lock()`.
  - Files: `internal/session/instance.go:1265, 1748`

- [ ] task-7.3 (golang-pro): TOCTOU race in `healthCache.isValid()` then `get()` — two separate RLock acquisitions allow concurrent `set()` between calls. Fix: combine into single `getIfValid() (VMHealth, bool)` method.
  - Files: `internal/vagrant/health.go:60-67, 20-35`

- [ ] task-7.4 (golang-pro): `SetDotfilePath` not mutex-protected — writes `m.dotfilePath` without lock while `vagrantCmd()` reads it without lock. Fix: guard both with `m.mu`.
  - Files: `internal/vagrant/sessions.go:71-73`, `internal/vagrant/manager.go:48`

## Pending — Wave 8: High Priority Fixes (6 tasks)

### HIGH — Should fix before merge

- [ ] task-8.1 (golang-pro): `EnsureRunning(nil)` deadlock — creates stdout pipe but never reads it when `onPhase` is nil. Vagrant process blocks once OS pipe buffer fills, causing `cmd.Wait()` to hang. Bridge adapter's `Boot()` calls `EnsureRunning(nil)`. Fix: skip pipe creation when `onPhase` is nil, use `cmd.Run()` directly.
  - Files: `internal/vagrant/manager.go:87-121`, `internal/vagrant/bridge.go:26`

- [ ] task-8.2 (golang-pro): Swallowed errors in `writeLockfile` — `MkdirAll`, `json.Marshal`, `os.WriteFile` all silently discard errors. Lockfile corruption causes incorrect `IsLastSession` decisions and premature VM destruction. Fix: return errors, propagate from `RegisterSession`/`UnregisterSession`, or at minimum log them.
  - Files: `internal/vagrant/sessions.go:78-93`

- [ ] task-8.3 (golang-pro): `initCache()` not thread-safe — nil check + assignment on `m.cache` without lock. Two concurrent `HealthCheck()` calls can both enter the block. Fix: use `sync.Once`.
  - Files: `internal/vagrant/health.go:46-52`

- [ ] task-8.4 (golang-pro): Global mutable `VagrantProviderFactory` — package-level var set via `init()` is not thread-safe for parallel tests. Fix: use `sync.Once` or `atomic.Value`.
  - Files: `internal/session/vagrant_iface.go:47`, `internal/vagrant/bridge.go:7-9`

- [ ] task-8.5 (golang-pro): `DestroySuspendedVMs` removes `.vagrant` dir even on destroy failure — orphans the VM permanently. Fix: skip cleanup when `vagrant destroy -f` fails. Also add Vagrantfile existence check to prevent path traversal.
  - Files: `internal/ui/cleanup_dialog.go:338-358`

- [ ] task-8.6 (golang-pro): Health checks bypass `vagrantCmd()` — `runVagrantStatus()` and `runSSHProbe()` use `exec.Command` directly, missing `VAGRANT_DOTFILE_PATH`. Functional bug: health checks query wrong VM in multi-session-isolation mode. Fix: use `m.vagrantCmd()` helper.
  - Files: `internal/vagrant/health.go:94-109, 112-130`

## Completed — Session 10

- [x] Installed `gopls-lsp` Claude Code plugin from marketplace
- [x] Installed `gopls v0.21.1` binary (`go install golang.org/x/tools/gopls@latest`)
- [x] Verified gopls works: diagnostics, symbols, definitions, references all functional
- [x] Added "Go Tooling" section to CLAUDE.md documenting gopls-lsp plugin
- [x] Added gopls-lsp reference to vagrant mode agent plan header

## Completed — Session 9

### Wave 5: UI Integration — 3/3 complete (task-5.1 done in Wave 2)
- [x] task-5.2 (sonnet): Boot progress display in list.go + instance.go fields
- [x] task-5.3 (sonnet): Stale VM cleanup dialog (Shift+D) — cleanup_dialog.go + home.go keybinding
- [x] task-5.4 (haiku): Apple Silicon kext error surfacing in TUI

### Wave 6: Testing & Hardening — complete
- [x] task-6.1: MockVagrantProvider + interface check (bridge_test.go)
- [x] task-6.2: Manager unit tests (existing + enhanced)
- [x] task-6.3: MCP unit tests (existing, 78-93% coverage)
- [x] task-6.4: Health check unit tests (health_test.go extended)
- [x] task-6.5: Instance lifecycle unit tests (instance_test.go extended)
- [x] task-6.6: Credential guard tests (existing, 80% coverage)
- [x] task-6.7: Config drift tests (drift_test.go extended)
- [x] task-6.8: Tips tests (existing, fully covered)
- [x] task-6.9: UI tests (claudeoptions_test.go created)
- [x] task-6.10: Documentation updates (README.md + config-reference.md)

### Cleanup
- [x] Removed junk files from commit (agentic-deck binary, settings.local.json, excalidraw.log)
- [x] Added .gitignore entries
- [x] Created CLAUDE.md
- [x] Installed golang-pro subagent at ~/.claude/agents/golang-pro.md

### Wave 5+6 Quality Gates — PASSED
- [x] `go build ./...` — PASS
- [x] `go vet ./...` — PASS
- [x] `go test ./internal/session/` — PASS (52.1% coverage)
- [x] `go test ./internal/vagrant/` — PASS (72.1% coverage)
- [x] `go test ./internal/ui/` — PASS (34.0% coverage)

## Completed — Sessions 1-8

- [x] Multi-perspective brainstorm (Architect, Implementer, Devil's Advocate, Security Analyst)
- [x] Wrote design document: `docs/plans/2026-02-14-vagrant-mode-design.md`
- [x] Multi-model design review with Gemini 3 Pro + GPT 5.1 Codex — 22 issues amended
- [x] Multi-model consensus review of plan with Gemini 3 Pro + GPT 5.2
- [x] Created implementation plan using `agentic-ai-plan` skill — 34 tasks, 6 waves
- [x] Rebased feature/vagrant branch from feature/teammate-mode onto main (v0.16.0)
- [x] Wave 1 (Foundation): 4/4 complete
- [x] Wave 2 (Core Infrastructure): 8/8 complete
- [x] Wave 3 (MCP & Skills): 4/4 complete
- [x] Wave 4 (Instance Lifecycle): 5/5 complete

## Progress Summary

| Wave | Tasks | Status | Notes |
|------|-------|--------|-------|
| 1 Foundation | 4/4 | Done | |
| 2 Core Infrastructure | 8/8 | Done | manager.go split into 9 files |
| 3 MCP & Skills | 4/4 | Done | |
| 4 Instance Lifecycle | 5/5 | Done | bridge adapter pattern |
| 5 UI Integration | 3/3 | Done | task-5.1 done in Wave 2 |
| 6 Testing & Hardening | 10/10 | Done | 72.1% vagrant, 52.1% session coverage |
| 7 Critical Bug Fixes | 0/4 | Pending | 4 race conditions from code review |
| 8 High Priority Fixes | 0/6 | Pending | deadlock, swallowed errors, path traversal |

**Total: 34/44 tasks complete (77%) — 10 bug fix tasks remaining**
