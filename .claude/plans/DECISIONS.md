# Architecture Decisions

## 2026-02-14 - Vagrant Mode: Wrapper Command Approach (Approach 1)

**Context**: User wants a "Just do it" checkbox that spawns Claude Code in an isolated Vagrant VM with `--dangerously-skip-permissions` and sudo access. Three approaches evaluated by multi-perspective brainstorm.
**Decision**: Wrapper Command approach -- add checkbox, auto-manage VM lifecycle, wrap commands via `vagrant ssh -c`. No provider abstraction, no security hardening beyond VM isolation.
**Consequences**: Minimal complexity (4 modified + 5 new files). VirtualBox dependency. First boot latency (5-10 min). Bidirectional sync risk accepted as intentional.

## 2026-02-14 - Vagrant Mode: Force skip-permissions when vagrant mode enabled

**Context**: Should the user be able to disable `--dangerously-skip-permissions` while in vagrant mode?
**Decision**: Force it on automatically. The entire purpose of vagrant mode is unrestricted access in a safe sandbox.
**Consequences**: Simpler UX. Users who want restricted mode should not use vagrant mode.

## 2026-02-14 - MCP Compatibility: SSH Reverse Tunnels + SendEnv

**Context**: MCP tools configured in agent-deck need to work inside the Vagrant VM. HTTP MCPs bind to localhost (unreachable via NAT), env vars need secure transport.
**Decision**: SSH reverse tunnels (`-R PORT:localhost:PORT`) for HTTP/SSE MCPs. SSH `SendEnv`/`AcceptEnv` for env var transport. No URL rewriting needed. VM sshd configured with `AcceptEnv *` during provisioning.
**Consequences**: HTTP MCPs work even when bound to 127.0.0.1 only. Env vars never visible in `ps aux`. No quoting/escaping issues. Requires sshd config in provisioning.

## 2026-02-14 - Crash Recovery: Two-Phase Health Check

**Context**: Vagrant VM can crash or hang independently. Hypervisor may report "running" even when guest is frozen.
**Decision**: Two-phase health check: Phase 1 = `vagrant status --machine-readable` (~200ms), Phase 2 = `vagrant ssh -c 'echo pong'` with 5s timeout (only when "running"). 30s interval.
**Consequences**: Detects both hypervisor-level crashes and in-VM hangs. ~200ms overhead when healthy, up to 5s when hung.

## 2026-02-14 - Crash Recovery: agent-deck Crash is Automatic

**Context**: What happens when agent-deck itself crashes while a vagrant session is running?
**Decision**: No special handling needed. tmux session survives, VM survives, Claude survives. On restart, `ReconnectSessionLazy()` reconnects to existing tmux session. `HealthCheck()` confirms VM state on next poll.
**Consequences**: Zero-effort recovery for agent-deck crashes. Only edge case: crash during `vagrant up` before Claude launches -- handled by restart flow detecting partial VM state.

## 2026-02-14 - Vagrantfile Customization via config.toml

**Context**: Users need to customize the VM beyond memory/CPU -- custom packages, port forwarding, provisioning scripts, or their own Vagrantfile entirely.
**Decision**: Expand `[vagrant]` config. `provision_packages` appends to base set (not replaces). `provision_packages_exclude` removes from base. `vagrant.vagrantfile` disables auto-generation entirely.
**Consequences**: Users can't accidentally remove required packages. Three tiers: defaults (zero config), tuning (set packages/ports), full override (custom Vagrantfile).

## 2026-02-14 - MCP Config Generation on Session Start (Bug Fix)

**Context**: `regenerateMCPConfig()` was only called in `Restart()`, not in `Start()` or `StartWithMessage()`. New sessions never generated `.mcp.json` from config.toml.
**Decision**: Call `regenerateMCPConfig()` in both `Start()` and `StartWithMessage()` for Claude sessions.
**Consequences**: MCP tools available from the first session start.

## 2026-02-14 - Multiple Sessions Per Project: User Prompt

**Context**: Multiple vagrant sessions from the same project need to handle VM sharing.
**Decision**: When creating a new session and a VM is already running for the same project, prompt the user: "Share existing VM" or "Create separate VM". Share reuses VM with multiple SSH connections. Separate uses `VAGRANT_DOTFILE_PATH` for isolated VMs.
**Consequences**: Users choose their trade-off. Share = faster, shared filesystem. Separate = full isolation, longer startup, port conflict risk (mitigated by auto_correct).

## 2026-02-14 - VagrantProvider Interface for Testability

**Context**: Unit tests for instance.go lifecycle methods can't run in CI without Vagrant installed.
**Decision**: Extract `VagrantProvider` interface. Production uses `Manager`, tests use `MockVagrantProvider`.
**Consequences**: All lifecycle unit tests runnable in CI. Standard Go interface pattern.

## 2026-02-14 - Credential Guard: Multi-Layer Protection

**Context**: Credential files in project directory are synced into VM. Claude has unrestricted access.
**Decision**: Three layers: (1) Skill warns Claude not to read credential files, (2) PreToolUse hook blocks Read/View/Cat of known credential patterns, (3) rsync mode excludes credential files from sync.
**Consequences**: Hard guardrail via hook. Soft guidance via skill. Physical exclusion for rsync users.

## 2026-02-14 - Provision Drift Detection via Config Hash

**Context**: VM packages become outdated when config.toml changes but VM isn't recreated.
**Decision**: SHA-256 hash of Vagrantfile inputs stored in `.vagrant/agent-deck-config.sha256`. On drift, auto re-provision via `vagrant provision`. Box changes require manual destroy+recreate.
**Consequences**: Config changes apply automatically without destroying VM. Box changes logged as warning.

## 2026-02-14 - Windows Unsupported for v1

**Context**: Windows has different path handling, SSH clients, and no `syscall.Statfs`.
**Decision**: Document Windows as unsupported. macOS and Linux only.
**Consequences**: Reduced scope. Windows support may be added later.
