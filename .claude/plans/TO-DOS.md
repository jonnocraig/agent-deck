# Current Tasks

## In Progress

- [ ] Vagrant Mode ("Just Do It") feature -- design review complete, awaiting implementation

## Completed This Session

- [x] Multi-model design review with Gemini 3 Pro + GPT 5.1 Codex -- 22 issues identified
- [x] Issue #1: MCP network binding -- replaced URL rewriting with SSH reverse tunnels (-R)
- [x] Issue #2: Env var transport -- replaced inline vars with SSH SendEnv/AcceptEnv
- [x] Issue #3: Race condition async suspend/start -- goroutine + done channel + spinner
- [x] Issue #4: SSH agent forwarding -- config.ssh.forward_agent = true
- [x] Issue #5: No boot progress -- phase parser + 100 loading tips (50 Vagrant + 50 world facts)
- [x] Issue #6: Health check misses hangs -- two-phase: status + SSH liveness probe
- [x] Issue #7: Docker nested virt -- --nested-hw-virt on
- [x] Issue #8: Multiple sessions per project -- prompt: share VM or create separate VM
- [x] Issue #9: Stale suspended VMs -- warning at 3+ + Shift+D cleanup dialog
- [x] Issue #10: Health check interval -- 60s to 30s
- [x] Issue #11: No disk space preflight -- block <5GB, warn 5-10GB
- [x] Issue #12: No VirtualBox check -- combined preflight (Vagrant + VBox + disk)
- [x] Issue #13: Port forwarding collisions -- auto_correct: true
- [x] Issue #14: provision_packages replaces -- changed to append + exclude
- [x] Issue #15: inotify broken on VBox sync -- polling env vars + skill guidance
- [x] Issue #16: Apple Silicon kext approval -- stderr parsing + System Settings guidance
- [x] Issue #17: Missing hostname -- auto-set agentdeck-<project>
- [x] Issue #18: No Windows support -- documented as unsupported v1
- [x] Issue #19: Enterprise proxy -- auto-forward host proxy vars + docs
- [x] Issue #20: Credential leakage -- skill warning + PreToolUse hook + rsync exclude
- [x] Issue #21: No box update path -- config hash + auto re-provision
- [x] Issue #22: CI testing needs interface -- VagrantProvider interface + mock

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

## Pending

- [ ] Create implementation plan using `agentic-ai-plan` skill
- [ ] Set up git worktree for implementation
- [ ] Execute plan with agent team using `agentic-ai-implement`
- [ ] Create PR for `feature/vagrant` -> upstream
- [ ] Create PR for skeleton repo `feature/agentic-ai-skills` branch

## Blocked

- None
