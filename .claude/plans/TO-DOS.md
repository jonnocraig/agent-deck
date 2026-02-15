# Current Tasks

## In Progress

- [ ] Vagrant Mode ("Just Do It") feature -- plan reviewed and amended, awaiting implementation

## Completed This Session (Session 5)

- [x] Multi-model consensus review of plan vs design doc with Gemini 3 Pro + GPT 5.2
- [x] C1: Split manager.go into 9 concern-separated files to eliminate Wave 2/3 file contention
- [x] C2: Split Wave 3 file targets (wrap.go, sync.go) for parallel safety
- [x] C3: Fixed Wave 4 dependency chain -- serialized instance.go edits (4.1 -> [4.2,4.3,4.4] -> 4.5)
- [x] G1: Added task-6.10 for documentation (config-reference.md + README)
- [x] G2: Explicit AutoSuspend/AutoDestroy gating + toast surfacing in task-4.1
- [x] G3: HealthCheckInterval read from config (not hardcoded) in task-4.3
- [x] G4: Fixed 60s->30s TTL inconsistency across 4 locations in design doc
- [x] G5: task-5.3 now updates session status after VM destruction
- [x] G6: Wave 6 test tasks reference correct split files
- [x] O1: Moved task-3.3 (tips) to Wave 1 -- no dependencies, pure data
- [x] O2: Moved task-5.1 (checkbox) to Wave 2 -- only depends on Wave 1
- [x] Updated meta: 34 tasks, max parallelism 10, avg 5.7
- [x] Validated JSON + cross-referenced all wave/task/dependency/phase relationships (0 errors)
- [x] Regenerated MD plan from JSON (727 lines)

## Completed Previous Sessions

- [x] Multi-perspective brainstorm (Architect, Implementer, Devil's Advocate, Security Analyst)
- [x] Wrote design document: `docs/plans/2026-02-14-vagrant-mode-design.md`
- [x] Added MCP compatibility, crash recovery, user documentation sections
- [x] Expanded VagrantSettings struct (6 -> 14+ fields)
- [x] Fixed MCP regen on Start() -- regenerateMCPConfig() in Start() and StartWithMessage()
- [x] Rebased feature/teammate-mode onto upstream/main (v0.16.0)
- [x] Created feature/vagrant branch and pushed to origin
- [x] Copied agentic-ai skills to global ~/.claude/skills/
- [x] Pushed skills to skeleton repo on feature/agentic-ai-skills branch
- [x] Multi-model design review with Gemini 3 Pro + GPT 5.1 Codex -- 22 issues amended
- [x] Created implementation plan using `agentic-ai-plan` skill -- 27 tasks, 6 waves, 3 output files

## Pending

- [ ] Set up git worktree for implementation
- [ ] Execute plan with agent team using `agentic-ai-implement`
- [ ] Create PR for `feature/vagrant` -> upstream
- [ ] Create PR for skeleton repo `feature/agentic-ai-skills` branch

## Blocked

- None
