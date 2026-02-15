# Current Tasks

## In Progress

- [ ] Wave 3 quality gates (build passed, vet passed, tests interrupted -- need re-run)
- [ ] Execute remaining waves with `agentic-ai-implement` skill

## Completed This Session (Session 7 -- Implementation Sessions 2+3)

### Wave 1 (Foundation) -- 4/4 complete
- [x] task-1.1: VagrantSettings struct + config parsing (userconfig.go)
- [x] task-1.2: VagrantProvider interface (provider.go)
- [x] task-1.3: UseVagrantMode in ClaudeOptions (tooloptions.go)
- [x] task-3.3: Loading tips -- 100 tips (tips.go)

### Wave 2 (Core Infrastructure) -- 8/8 complete
- [x] task-2.1: Manager struct + basic lifecycle (manager.go)
- [x] task-2.2: Vagrantfile template generation (vagrantfile.go)
- [x] task-2.3: Boot phase parser (bootphase.go)
- [x] task-2.4: Preflight checks (preflight.go)
- [x] task-2.5: Two-phase health check (health.go)
- [x] task-2.6: Config drift detection (drift.go)
- [x] task-2.7: Multi-session tracking (sessions.go)
- [x] task-5.1: TUI vagrant checkbox (claudeoptions.go)

### Wave 3 (MCP & Skills) -- 4/4 complete
- [x] task-3.1: MCP config for Vagrant (mcp.go)
- [x] task-3.2: Sudo skill + credential guard (skill.go)
- [x] task-3.4: Command wrapping SSH tunnels (wrap.go)
- [x] task-3.5: SyncClaudeConfig (sync.go)

### Quality Gates Passed
- [x] Wave 1 quality gates (build, vet, tests)
- [x] Wave 2 quality gates (build, vet, vagrant tests, session tests)
- [ ] Wave 3 quality gates (build PASS, vet PASS, tests INTERRUPTED)

## Completed Previous Sessions

- [x] Multi-perspective brainstorm (Architect, Implementer, Devil's Advocate, Security Analyst)
- [x] Wrote design document: `docs/plans/2026-02-14-vagrant-mode-design.md`
- [x] Multi-model design review with Gemini 3 Pro + GPT 5.1 Codex -- 22 issues amended
- [x] Multi-model consensus review of plan with Gemini 3 Pro + GPT 5.2
- [x] Created implementation plan using `agentic-ai-plan` skill -- 34 tasks, 6 waves
- [x] Rebased feature/vagrant branch from feature/teammate-mode onto main (v0.16.0)

## Pending -- Waves 4-6 (18 tasks remaining)

### Wave 4: Instance Lifecycle (5 tasks, partially serialized)
- [ ] task-4.1: Start hook (instance.go) -- MUST run first
- [ ] task-4.2: Health monitor goroutine (instance.go) -- parallel with 4.3, 4.4
- [ ] task-4.3: Config drift on start (instance.go) -- parallel with 4.2, 4.4
- [ ] task-4.4: Multi-session prompt (instance.go) -- parallel with 4.2, 4.3
- [ ] task-4.5: Stop/restart hooks (instance.go) -- MUST run last

### Wave 5: UI Integration (3 tasks)
- [ ] task-5.2: Boot progress display in session list
- [ ] task-5.3: Stale VM cleanup dialog (Shift+D)
- [ ] task-5.4: Apple Silicon kext detection in TUI

### Wave 6: Hardening & Polish (10 tasks)
- [ ] task-6.1: Manager unit tests
- [ ] task-6.2: MCP unit tests
- [ ] task-6.3: Health check unit tests
- [ ] task-6.4: Instance lifecycle unit tests
- [ ] task-6.5: Credential guard unit tests
- [ ] task-6.6: Drift detection unit tests
- [ ] task-6.7: Tips unit tests
- [ ] task-6.8: UI checkbox unit tests
- [ ] task-6.9: MockVagrantProvider
- [ ] task-6.10: Documentation (config-reference.md + README)

## Blocked

- None

## Known Issues

- task-5.3 lists `cleanup_dialog.go` as "modify" but it doesn't exist -- should be "create"
