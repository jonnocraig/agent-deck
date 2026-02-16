# Current Tasks

## Completed — Wave 7: Critical Bug Fixes (4 tasks)

- [x] task-7.1: Race condition on `vmOpDone` channel — protected with `i.mu`, capture-before-select pattern
- [x] task-7.2: Race condition on `cleanShutdown` — converted to `atomic.Bool`
- [x] task-7.3: TOCTOU race in `healthCache` — combined into `getIfValid()` method
- [x] task-7.4: `SetDotfilePath` not mutex-protected — guarded with `m.mu`

## Completed — Wave 8: High Priority Fixes (6 tasks)

- [x] task-8.1: `EnsureRunning(nil)` deadlock — skip pipe when `onPhase` is nil, use `CombinedOutput()`
- [x] task-8.2: Swallowed errors in `writeLockfile` — return errors, propagate to callers
- [x] task-8.3: `initCache()` not thread-safe — use `sync.Once`
- [x] task-8.4: Global mutable `VagrantProviderFactory` — use `atomic.Value` with getter/setter
- [x] task-8.5: `DestroySuspendedVMs` cleanup on failure — skip cleanup on destroy fail, add Vagrantfile check
- [x] task-8.6: Health checks bypass `vagrantCmd()` — use `m.vagrantCmd()` + new `vagrantCmdContext()` helper

## Completed — Code Review Fixes (4 issues)

- [x] HIGH-1: `waitForVagrantOp` channel reset — nil out instead of recreating
- [x] HIGH-2: `EnsureRunning` scanner.Err() — moved stderr setup after nil check, added scanner error logging
- [x] MEDIUM-1: `selectedCount` in cleanup dialog — only count `true` values
- [x] LOW-4: `cleanShutdown` never reset on restart — added `Store(false)` in `restartVagrantSession()`

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
| 7 Critical Bug Fixes | 4/4 | Done | 4 race conditions fixed + code review fixes |
| 8 High Priority Fixes | 6/6 | Done | deadlock, errors, thread safety, path traversal |

**Total: 44/44 tasks complete (100%) — all waves done, ready for PR**
