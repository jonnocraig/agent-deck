# Vagrant Mode (Just Do It) — Agent Team Plan

> Source: `docs/plans/2026-02-14-vagrant-mode-design.md`
> Generated: 2026-02-15 (updated: 2026-02-15)

**Goal:** Add 'Just do it (vagrant sudo)' checkbox to agent-deck that auto-manages a Vagrant VM lifecycle and runs Claude Code inside it with --dangerously-skip-permissions and sudo access.

**Architecture:** Wrapper Command approach — checkbox in TUI, VagrantManager handles VM lifecycle, commands wrapped via 'vagrant ssh --' with SSH reverse tunnels and SendEnv. VagrantProvider interface for testability.

**Tech Stack:** Go 1.24, Bubble Tea TUI, Vagrant, VirtualBox, tmux, SSH

---

## Summary

| Metric | Value |
|--------|-------|
| Total Tasks | 34 |
| Total Waves | 6 |
| Avg Parallelism | 5.7x |
| Max Parallelism | 10 |
| Critical Path | 6 tasks |

---

## Wave Execution Strategy

### Wave 1: Foundation — 4 tasks (parallelism=4)

| Task | Name | Model | Agents | Complexity | Files |
|------|------|-------|--------|------------|-------|
| 1.1 | VagrantSettings struct + config parsing | sonnet | tdd-guide | medium | userconfig.go |
| 1.2 | VagrantProvider interface | sonnet | — | low | provider.go |
| 1.3 | UseVagrantMode in ClaudeOptions | haiku | tdd-guide | low | tooloptions.go |
| 3.3 | Loading tips (100 tips) | haiku | — | low | tips.go |

**Checkpoint:** Wave 1 complete: Config types, VagrantProvider interface, ClaudeOptions field, and loading tips established.

Review items:
- [ ] VagrantSettings struct fields match design doc
- [ ] VagrantProvider interface covers all Manager methods
- [ ] UseVagrantMode field in ClaudeOptions with JSON tag
- [ ] 100 tips present (50 Vagrant + 50 world facts)

Quality gates: code-review

---

### Wave 2: Core Infrastructure — 8 tasks (parallelism=8)

| Task | Name | Model | Agents | Complexity | Files |
|------|------|-------|--------|------------|-------|
| 2.1 | Manager struct + basic lifecycle | sonnet | tdd-guide | high | manager.go |
| 2.2 | Vagrantfile template generation | sonnet | tdd-guide | high | vagrantfile.go |
| 2.3 | Boot phase parser | sonnet | tdd-guide | medium | bootphase.go |
| 2.4 | Preflight checks | sonnet | tdd-guide | medium | preflight.go |
| 2.5 | Two-phase health check | sonnet | tdd-guide | medium | health.go |
| 2.6 | Config drift detection | haiku | tdd-guide | low | drift.go |
| 2.7 | Multi-session tracking | sonnet | tdd-guide | medium | sessions.go |
| 5.1 | Vagrant mode checkbox in TUI | sonnet | code-reviewer | medium | claudeoptions.go |

**Checkpoint:** Wave 2 complete: VagrantManager with full lifecycle, Vagrantfile generation, preflight checks, health monitoring, drift detection, multi-session support, and TUI checkbox.

Review items:
- [ ] Manager methods match VagrantProvider interface
- [ ] Vagrantfile template interpolation correct
- [ ] Preflight thresholds (5GB/10GB) match design
- [ ] Health check two-phase logic
- [ ] Config hash deterministic
- [ ] Each Wave 2 task writes to its own file (no manager.go contention)
- [ ] TUI checkbox renders below Teammate mode

Quality gates: code-review, build-verification

---

### Wave 3: MCP & Skills — 4 tasks (parallelism=4)

| Task | Name | Model | Agents | Complexity | Files |
|------|------|-------|--------|------------|-------|
| 3.1 | MCP config for Vagrant | sonnet | tdd-guide | high | mcp.go |
| 3.2 | Static skill + credential guard hook | sonnet | tdd-guide, security-reviewer | medium | skill.go |
| 3.4 | Command wrapping with SSH tunnels | sonnet | tdd-guide | high | wrap.go |
| 3.5 | SyncClaudeConfig | sonnet | tdd-guide | medium | sync.go |

**Checkpoint:** Wave 3 complete: MCP config generation for Vagrant, SSH reverse tunnel support, sudo skill with credential guard, and command wrapping. (Tips moved to Wave 1.)

Review items:
- [ ] WriteMCPJsonForVagrant bypasses pool sockets
- [ ] CollectHTTPMCPPorts extracts localhost ports correctly
- [ ] Credential guard hook patterns complete
- [ ] WrapCommand format matches design
- [ ] Each Wave 3 task writes to its own file (wrap.go, sync.go)

Quality gates: code-review, security-review

---

### Wave 4: Instance Lifecycle — 5 tasks (parallelism=3)

> **Execution order:** task-4.1 first → [task-4.2, task-4.3, task-4.4] parallel → task-4.5 last (all modify instance.go)

| Task | Name | Model | Agents | Complexity | Files |
|------|------|-------|--------|------------|-------|
| 4.1 | Instance vagrant lifecycle hooks | opus | tdd-guide, code-reviewer | high | instance.go |
| 4.2 | Vagrant restart flow | sonnet | tdd-guide | medium | instance.go |
| 4.3 | Health check integration in UpdateStatus | sonnet | tdd-guide | medium | instance.go |
| 4.4 | Config drift + re-provision in Start | haiku | tdd-guide | low | instance.go |
| 4.5 | Multi-session prompt + share/separate flow | sonnet | tdd-guide | medium | instance.go |

**Checkpoint:** Wave 4 complete: Instance lifecycle fully integrated with Vagrant — start/stop/restart hooks, health monitoring, config drift handling, and multi-session prompting.

Review items:
- [ ] vagrantProvider interface used (not concrete Manager)
- [ ] vmOpDone channel + vmOpInFlight atomic pattern
- [ ] restartVagrantSession state machine covers all states
- [ ] Config drift triggers Provision() not Destroy()
- [ ] auto_suspend/auto_destroy gated on VagrantSettings
- [ ] HealthCheckInterval read from settings (not hardcoded 30s)

Quality gates: code-review, security-review

---

### Wave 5: UI Integration — 3 tasks (parallelism=3)

| Task | Name | Model | Agents | Complexity | Files |
|------|------|-------|--------|------------|-------|
| 5.2 | Boot progress display | sonnet | — | medium | sessionlist.go |
| 5.3 | Stale VM cleanup (Shift+D) | sonnet | — | medium | app.go, cleanup_dialog.go |
| 5.4 | Apple Silicon kext detection in TUI | haiku | — | low | instance.go |

**Checkpoint:** Wave 5 complete: Boot progress display, stale VM cleanup dialog, and Apple Silicon detection wired up. (TUI checkbox moved to Wave 2.)

Review items:
- [ ] Boot phase + elapsed timer in session list
- [ ] Shift+D cleanup dialog updates session status after destroy
- [ ] Session status updated post-cleanup

Quality gates: code-review

---

### Wave 6: Hardening & Polish — 10 tasks (parallelism=10)

| Task | Name | Model | Agents | Complexity | Files |
|------|------|-------|--------|------------|-------|
| 6.1 | MockVagrantProvider + interface check | haiku | — | low | mock_provider_test.go |
| 6.2 | Manager unit tests | sonnet | tdd-guide | medium | manager_test.go, vagrantfile_test.go, bootphase_test.go, preflight_test.go, sessions_test.go, wrap_test.go, sync_test.go |
| 6.3 | MCP unit tests | haiku | tdd-guide | low | mcp_test.go |
| 6.4 | Health check unit tests | haiku | tdd-guide | low | health_test.go |
| 6.5 | Instance lifecycle unit tests with mock | sonnet | tdd-guide | medium | instance_test.go |
| 6.6 | Credential guard tests | haiku | tdd-guide | low | skill_test.go |
| 6.7 | Config drift tests | haiku | — | low | drift_test.go |
| 6.8 | Tips tests | haiku | — | low | tips_test.go |
| 6.9 | UI tests | sonnet | tdd-guide | medium | claudeoptions_test.go |
| 6.10 | Documentation updates | haiku | doc-updater | low | config-reference.md, README.md |

**Checkpoint:** Wave 6 complete: All unit tests written and passing. Mock provider enables CI testing. Full coverage across manager, MCP, health, lifecycle, credentials, drift, tips, UI, and documentation.

Review items:
- [ ] All tests pass
- [ ] go build ./... succeeds
- [ ] go test ./internal/vagrant/... -count=1 passes
- [ ] go test ./internal/session/... -count=1 passes
- [ ] config-reference.md [vagrant] section complete

Quality gates: code-review, build-verification, security-review

---

## Dependency Summary

```
Wave 1 (Foundation + Tips) → Wave 2 (Core Manager + TUI Checkbox) → Wave 3 (MCP & Skills) → Wave 4 (Instance Integration, serialized) → Wave 5 (UI Polish) → Wave 6 (Testing + Docs)
```

## Task Details

### task-1.1: VagrantSettings struct + config parsing

- **Model:** sonnet
- **Wave:** 1
- **Complexity:** medium
- **Blocked by:** none
- **Modifies:** internal/session/userconfig.go
- **Tests:** internal/session/userconfig_test.go

**Steps:**
1. Write tests for VagrantSettings defaults and overrides: TestGetVagrantSettingsDefaults, TestGetVagrantSettingsOverrides
2. Add VagrantSettings struct with all fields: 14+ fields: MemoryMB, CPUs, Box, AutoSuspend, AutoDestroy, HostGatewayIP, SyncedFolderType, ProvisionPackages, ProvisionPkgExclude, NpmPackages, ProvisionScript, Vagrantfile, HealthCheckInterval, PortForwards, Env, ForwardProxyEnv
3. Add PortForward struct: Guest, Host, Protocol fields with TOML tags
4. Add Vagrant field to UserConfig struct: Vagrant VagrantSettings `toml:"vagrant"`
5. Add GetVagrantSettings() with defaults: 4096MB, 2 CPUs, bento/ubuntu-24.04, auto_suspend=true, health_check_interval=30, forward_proxy_env=true
6. Run tests: Verify defaults and TOML override parsing `go test ./internal/session/ -run TestGetVagrantSettings -count=1 -v`

### task-1.2: VagrantProvider interface

- **Model:** sonnet
- **Wave:** 1
- **Complexity:** low
- **Blocked by:** none
- **Creates:** internal/vagrant/provider.go

**Steps:**
1. Create internal/vagrant/ package directory: mkdir -p internal/vagrant `mkdir -p internal/vagrant`
2. Define VagrantProvider interface: All 24 methods from design doc: IsInstalled, PreflightCheck, EnsureRunning, Suspend, Resume, Destroy, ForceRestart, Reload, Provision, Status, HealthCheck, WrapCommand, EnsureVagrantfile, EnsureSudoSkill, SyncClaudeConfig, GetMCPPackages, HasConfigDrift, WriteConfigHash, RegisterSession, UnregisterSession, SessionCount, IsLastSession, SetDotfilePath, CheckDiskSpace, IsBoxCached
3. Define BootPhase type and constants: 8 phases: Downloading, Importing, Booting, Network, Mounting, Provisioning, NpmInstall, Ready
4. Define VMHealth struct: State, Healthy, Responsive, Message fields

### task-1.3: UseVagrantMode in ClaudeOptions

- **Model:** haiku
- **Wave:** 1
- **Complexity:** low
- **Blocked by:** none
- **Modifies:** internal/session/tooloptions.go
- **Tests:** internal/session/tooloptions_test.go

**Steps:**
1. Write test for UseVagrantMode forcing skip permissions: TestClaudeOptionsVagrantModeForceSkipPermissions
2. Add UseVagrantMode field to ClaudeOptions: UseVagrantMode bool `json:"use_vagrant_mode,omitempty"`
3. Update ToArgs to force --dangerously-skip-permissions when UseVagrantMode: if o.UseVagrantMode { o.SkipPermissions = true }
4. Update ToArgsForFork similarly: Same force logic for fork path
5. Run tests: Verify skip permissions forced `go test ./internal/session/ -run TestClaudeOptions -count=1 -v`

### task-2.1: Manager struct + basic lifecycle

- **Model:** sonnet
- **Wave:** 2
- **Complexity:** high
- **Blocked by:** task-1.1, task-1.2
- **Creates:** internal/vagrant/manager.go
- **Tests:** internal/vagrant/manager_test.go

**Steps:**
1. Write tests for Manager struct creation and IsInstalled: TestNewManager, TestIsInstalled
2. Define Manager struct: projectPath, settings, dotfilePath, sessions, mu fields
3. Implement NewManager constructor: Accept projectPath and VagrantSettings
4. Implement IsInstalled via exec.LookPath: Returns bool
5. Implement vagrantCmd helper: Sets Dir, VAGRANT_DOTFILE_PATH env if set
6. Implement Status via vagrant status --machine-readable: Parse machine-readable output for state
7. Implement EnsureRunning with phase callback: vagrant up with stdout parsing for BootPhase, Apple Silicon kext detection
8. Implement Suspend, Resume, Destroy, ForceRestart, Reload, Provision: Each wraps vagrant command
9. Verify Manager implements VagrantProvider interface: Compile-time check: var _ VagrantProvider = (*Manager)(nil)
10. Run tests: Unit tests for command construction and status parsing `go test ./internal/vagrant/ -run TestManager -count=1 -v`

### task-2.2: Vagrantfile template generation

- **Model:** sonnet
- **Wave:** 2
- **Complexity:** high
- **Blocked by:** task-1.1, task-1.2
- **Creates:** internal/vagrant/vagrantfile.go
- **Tests:** internal/vagrant/vagrantfile_test.go

**Steps:**
1. Write tests for EnsureVagrantfile: TestEnsureVagrantfile, TestEnsureVagrantfileWithMCPs, TestEnsureVagrantfileWithCustomPackages, TestEnsureVagrantfileWithPortForwards, TestEnsureVagrantfileWithProvisionScript, TestEnsureVagrantfileCustomTemplate, TestEnsureVagrantfileExistingRespected, TestEnsureVagrantfilePortForwardsAutoCorrect
2. Write tests for package resolution: TestProvisionPackagesAppendToBase, TestProvisionPackagesExclude, TestProvisionPackagesExcludeAndAppend, TestProvisionPackagesEmptyUsesBaseOnly
3. Write tests for hostname generation: TestHostnameFromProjectName, TestHostnameSanitization, TestHostnameTruncation
4. Implement resolvedPackages() helper: Base set minus excludes plus additions
5. Implement hostname sanitization: Lowercase, replace non-alnum with -, truncate 63 chars, RFC 1123
6. Implement EnsureVagrantfile: Template with box, hostname, memory, cpus, sync type, port forwards, packages, npm packages, provision script, proxy env, AcceptEnv *
7. Handle rsync credential exclusion: rsync__exclude list for credential files when synced_folder_type=rsync
8. Run tests: Verify all generation scenarios `go test ./internal/vagrant/ -run TestEnsureVagrantfile -count=1 -v`

### task-2.3: Boot phase parser

- **Model:** sonnet
- **Wave:** 2
- **Complexity:** medium
- **Blocked by:** task-1.2
- **Creates:** internal/vagrant/bootphase.go
- **Tests:** internal/vagrant/bootphase_test.go

**Steps:**
1. Write tests for boot phase parsing: TestBootPhaseParser, TestBootPhaseDownloadDetection
2. Implement parseBootPhase function: Match vagrant machine-readable output patterns to BootPhase constants
3. Implement wrapVagrantUpError: Apple Silicon kernel extension detection from stderr
4. Write test for Apple Silicon detection: TestWrapVagrantUpErrorAppleSiliconKext, TestWrapVagrantUpErrorNonAppleSilicon, TestWrapVagrantUpErrorUnrelatedFailure
5. Run tests: Verify phase parsing and error wrapping `go test ./internal/vagrant/ -run TestBootPhase -count=1 -v`

### task-2.4: Preflight checks

- **Model:** sonnet
- **Wave:** 2
- **Complexity:** medium
- **Blocked by:** task-1.1, task-1.2
- **Creates:** internal/vagrant/preflight.go
- **Tests:** internal/vagrant/preflight_test.go

**Steps:**
1. Write preflight tests: TestPreflightCheckBlocksBelowMinimum, TestPreflightCheckWarnsLowSpace, TestPreflightCheckPassesAboveThreshold, TestPreflightCheckAccountsForBoxCache, TestIsBoxCached, TestPreflightCheckVBoxMissing, TestPreflightCheckVBoxTooOld, TestPreflightCheckVBoxAppleSiliconWarning, TestCheckVBoxInstalledParsesVersion
2. Implement CheckDiskSpace: syscall.Statfs on project directory filesystem
3. Implement CheckVBoxInstalled: exec.LookPath + VBoxManage --version parsing
4. Implement IsBoxCached: vagrant box list --machine-readable parsing
5. Implement PreflightCheck: Combined: Vagrant + VBox + disk space with thresholds 5GB/10GB
6. Run tests: Verify all preflight scenarios `go test ./internal/vagrant/ -run TestPreflight -count=1 -v`

### task-2.5: Two-phase health check

- **Model:** sonnet
- **Wave:** 2
- **Complexity:** medium
- **Blocked by:** task-1.2
- **Creates:** internal/vagrant/health.go
- **Tests:** internal/vagrant/health_test.go

**Steps:**
1. Write health check tests: TestHealthCheckPhase1, TestHealthCheckPhase2LivenessProbe, TestHealthCheckPhase2Timeout, TestHealthCheckCaching
2. Implement HealthCheck Phase 1: vagrant status --machine-readable → VMHealth
3. Implement HealthCheck Phase 2: vagrant ssh -c 'echo pong' with 5s timeout (context.WithTimeout)
4. Add 30s TTL caching for Phase 1: In-memory cache with lastHealthCheck timestamp
5. Implement vmStateMessage helper: Map VM states to human-readable messages
6. Run tests: Verify two-phase logic and caching `go test ./internal/vagrant/ -run TestHealthCheck -count=1 -v`

### task-2.6: Config drift detection

- **Model:** haiku
- **Wave:** 2
- **Complexity:** low
- **Blocked by:** task-1.1, task-1.2
- **Creates:** internal/vagrant/drift.go
- **Tests:** internal/vagrant/drift_test.go

**Steps:**
1. Write config drift tests: TestConfigHashDeterministic, TestConfigHashChangesOnPackageAdd, TestConfigHashChangesOnProvisionScript, TestHasConfigDriftDetectsChange, TestHasConfigDriftFalseOnFirstRun, TestBoxChangeLogsWarning
2. Implement configHash(): SHA-256 of box + packages + npm packages + provision script content + port forwards
3. Implement HasConfigDrift(): Compare stored hash file to current hash
4. Implement WriteConfigHash(): Write hash to .vagrant/agent-deck-config.sha256
5. Run tests: Verify hash determinism and drift detection `go test ./internal/vagrant/ -run TestConfigHash -count=1 -v`

### task-2.7: Multi-session tracking

- **Model:** sonnet
- **Wave:** 2
- **Complexity:** medium
- **Blocked by:** task-1.2
- **Creates:** internal/vagrant/sessions.go
- **Tests:** internal/vagrant/sessions_test.go

**Steps:**
1. Write multi-session tests: TestRegisterUnregisterSession, TestSharedVMSuspendOnlyOnLastSession, TestSharedVMDestroyOnlyWhenAllDeleted, TestSeparateVMDotfilePath, TestSeparateVMCleanup, TestDetectRunningVMWithExistingSession, TestDetectRunningVMOrphaned
2. Implement RegisterSession/UnregisterSession: Thread-safe session tracking with mutex
3. Implement SessionCount/IsLastSession: Query session list
4. Implement SetDotfilePath: Set VAGRANT_DOTFILE_PATH for isolated VMs
5. Implement lockfile persistence: JSON lockfile at .vagrant/agent-deck.lock
6. Run tests: Verify session tracking and isolation `go test ./internal/vagrant/ -run TestSession -count=1 -v`

### task-3.1: MCP config for Vagrant

- **Model:** sonnet
- **Wave:** 3
- **Complexity:** high
- **Blocked by:** task-2.1
- **Creates:** internal/vagrant/mcp.go
- **Tests:** internal/vagrant/mcp_test.go

**Steps:**
1. Write MCP tests: TestWriteMCPJsonForVagrant, TestCollectHTTPMCPPorts, TestCollectHTTPMCPPortsDedup, TestGetMCPPackages, TestCollectEnvVarNames
2. Implement CollectHTTPMCPPorts: Extract unique ports from localhost/127.0.0.1 HTTP/SSE URLs, sorted and deduplicated
3. Implement CollectEnvVarNames: Merge MCP env var names + vagrant.env names + ANTHROPIC_API_KEY
4. Implement WriteMCPJsonForVagrant: STDIO fallback (no pool sockets), URLs unchanged, env vars collected
5. Implement GetMCPPackages: Extract npm packages from npx-based MCPs
6. Run tests: Verify MCP config generation `go test ./internal/vagrant/ -run TestMCP -count=1 -v`

### task-3.2: Static skill + credential guard hook

- **Model:** sonnet
- **Wave:** 3
- **Complexity:** medium
- **Blocked by:** task-1.2
- **Creates:** internal/vagrant/skill.go
- **Tests:** internal/vagrant/skill_test.go

**Steps:**
1. Write skill tests: TestGetVagrantSudoSkill, TestSudoSkillMentionsInotify, TestSudoSkillMentionsCredentialWarning, TestCredentialGuardHookInjected, TestCredentialGuardHookBlocksEnvFile, TestCredentialGuardHookBlocksSSHKey, TestCredentialGuardHookAllowsNormalFiles, TestCredentialGuardHookNotInjectedForNonVagrant
2. Implement GetVagrantSudoSkill(): Skill markdown with VM info, sudo access, Docker/Node/Git, /vagrant path, inotify/polling guidance, credential warning
3. Implement EnsureSudoSkill(): Write skill file to project .claude/skills/ directory
4. Implement credential guard hook injection: Write PreToolUse hook to .claude/settings.local.json (merge, don't overwrite)
5. Implement GetCredentialGuardHook(): Returns hook JSON for Read|View|Cat matcher with credential patterns
6. Run tests: Verify skill content and hook injection `go test ./internal/vagrant/ -run TestSkill -count=1 -v`

### task-3.3: Loading tips (100 tips)

- **Model:** haiku
- **Wave:** 1
- **Complexity:** low
- **Blocked by:** none
- **Creates:** internal/vagrant/tips.go
- **Tests:** internal/vagrant/tips_test.go

**Steps:**
1. Write tips tests: TestGetRandomTip, TestGetNextTipRotation
2. Define Tip struct: Text, Source, Category fields
3. Embed 50 Vagrant best practice tips: From design doc Loading Tips Content section
4. Embed 50 world fact tips: From design doc Loading Tips Content section
5. Implement GetRandomTip() and GetNextTip(index): Random selection and sequential rotation
6. Run tests: Verify tip retrieval and rotation `go test ./internal/vagrant/ -run TestTip -count=1 -v`

### task-3.4: Command wrapping with SSH tunnels

- **Model:** sonnet
- **Wave:** 3
- **Complexity:** high
- **Blocked by:** task-2.1
- **Creates:** internal/vagrant/wrap.go
- **Tests:** internal/vagrant/wrap_test.go

**Steps:**
1. Write WrapCommand tests: TestWrapCommand, TestWrapCommandWithSendEnv, TestWrapCommandWithVagrantEnvVars, TestWrapCommandWithTunnels, TestPollingEnvVarsInjectedForVirtualBox, TestPollingEnvVarsNotInjectedForNFS, TestPollingEnvVarsUserOverride, TestProxyEnvVarsForwardedWhenSet, TestProxyEnvVarsNotForwardedWhenUnset, TestProxyEnvVarsDisabledByConfig, TestProxyEnvVarsUserOverride
2. Implement WrapCommand: vagrant ssh -- with -R flags, -o SendEnv flags, -t PTY allocation, cd /vagrant && command
3. Implement collectProxyEnvVars: Detect host proxy env vars, deduplicate uppercase/lowercase
4. Add polling env var auto-injection: CHOKIDAR_USEPOLLING, WATCHPACK_POLLING, TSC_WATCHFILE when synced_folder_type=virtualbox
5. Run tests: Verify command format with all combinations `go test ./internal/vagrant/ -run TestWrapCommand -count=1 -v`

### task-3.5: SyncClaudeConfig

- **Model:** sonnet
- **Wave:** 3
- **Complexity:** medium
- **Blocked by:** task-2.1
- **Creates:** internal/vagrant/sync.go
- **Tests:** internal/vagrant/sync_test.go

**Steps:**
1. Write SyncClaudeConfig tests: Test config file reading and SSH copy command generation
2. Implement SyncClaudeConfig: Read host Claude configs, extract HTTP/SSE ports, write to VM via vagrant ssh -c heredoc
3. Run tests: Verify config sync `go test ./internal/vagrant/ -run TestSyncClaudeConfig -count=1 -v`

### task-4.1: Instance vagrant lifecycle hooks

- **Model:** opus
- **Wave:** 4
- **Complexity:** high
- **Blocked by:** task-3.1, task-3.2, task-3.4, task-3.5
- **Modifies:** internal/session/instance.go
- **Tests:** internal/session/instance_test.go

**Steps:**
1. Add vagrant fields to Instance struct: vagrantProvider VagrantProvider, lastVMHealthCheck, cleanShutdown, vmOpDone chan, vmOpInFlight atomic.Bool
2. Add IsVagrantMode() helper: Check UseVagrantMode from persisted tool options
3. Implement applyVagrantWrapper in Start/StartWithMessage: Call PreflightCheck, EnsureVagrantfile, EnsureSudoSkill, EnsureRunning, WriteMCPJsonForVagrant, SyncClaudeConfig, CollectEnvVarNames, CollectHTTPMCPPorts, WrapCommand
4. Implement stopVagrant with goroutine + done channel: Non-blocking suspend, signal vmOpDone on completion
5. Implement destroyVagrant with goroutine + done channel: Non-blocking destroy, signal vmOpDone on completion
6. Implement waitForVagrantOp: Select on vmOpDone with 60s timeout
7. Wire up Stop() and Delete() hooks: Call stopVagrant/destroyVagrant when vagrant mode. Gate suspend on VagrantSettings.AutoSuspend (skip if false). Gate destroy on VagrantSettings.AutoDestroy and session count for shared VMs. Surface preflight warnings as TUI toasts (not just logs).
8. Wire up Start() to wait on in-flight ops: Call waitForVagrantOp before applyVagrantWrapper

### task-4.2: Vagrant restart flow

- **Model:** sonnet
- **Wave:** 4
- **Complexity:** medium
- **Blocked by:** task-4.1
- **Modifies:** internal/session/instance.go
- **Tests:** internal/session/instance_test.go

**Steps:**
1. Write restart tests: TestRestartVagrantSession for running/suspended/aborted/not_created states
2. Implement restartVagrantSession: State machine: check HealthCheck → branch by state → re-sync configs → respawn tmux
3. Wire into Restart() method: Call restartVagrantSession when IsVagrantMode()
4. Run tests: Verify all recovery paths `go test ./internal/session/ -run TestRestart -count=1 -v`

### task-4.3: Health check integration in UpdateStatus

- **Model:** sonnet
- **Wave:** 4
- **Complexity:** medium
- **Blocked by:** task-4.1
- **Modifies:** internal/session/instance.go
- **Tests:** internal/session/instance_test.go

**Steps:**
1. Write health check integration tests: TestVMHealthToErrorMessage, TestMockProviderHealthCheckIntegration
2. Add health check to UpdateStatus: Read HealthCheckInterval from VagrantSettings (default 30s, configurable). Poll at that interval for vagrant sessions, set StatusError on unhealthy
3. Add immediate health check on startup for ungraceful shutdown: If cleanShutdown is false, check immediately
4. Add contextual error messages: Map VM states to user-facing messages: 'VM crashed', 'VM unresponsive', etc.
5. Run tests: Verify health check triggers error state `go test ./internal/session/ -run TestVMHealth -count=1 -v`

### task-4.4: Config drift + re-provision in Start

- **Model:** haiku
- **Wave:** 4
- **Complexity:** low
- **Blocked by:** task-4.1
- **Modifies:** internal/session/instance.go
- **Tests:** internal/session/instance_test.go

**Steps:**
1. Write drift integration tests: TestProvisionCalledOnDrift
2. Add drift check after EnsureRunning in Start: HasConfigDrift → EnsureVagrantfile → Provision → WriteConfigHash → toast
3. Run tests: Verify drift triggers re-provision `go test ./internal/session/ -run TestProvision -count=1 -v`

### task-4.5: Multi-session prompt + share/separate flow

- **Model:** sonnet
- **Wave:** 4
- **Complexity:** medium
- **Blocked by:** task-4.1, task-4.4
- **Modifies:** internal/session/instance.go
- **Tests:** internal/session/instance_test.go

**Steps:**
1. Write multi-session integration tests: TestForkInheritsVMSharingChoice
2. Implement VM ownership detection in Start: Check vagrant status + existing sessions for same project
3. Add share/separate handling: Share: RegisterSession + skip vagrant up. Separate: SetDotfilePath + vagrant up
4. Fork inheritance: Forked session inherits parent's share/separate decision
5. Run tests: Verify multi-session flow `go test ./internal/session/ -run TestMultiSession -count=1 -v`

### task-5.1: Vagrant mode checkbox in TUI

- **Model:** sonnet
- **Wave:** 2
- **Complexity:** medium
- **Blocked by:** task-1.3
- **Modifies:** internal/ui/claudeoptions.go

**Steps:**
1. Add useVagrantMode field to ClaudeOptionsPanel: Bool field, focus index after teammate mode
2. Add checkbox rendering: renderCheckboxLine for 'Just do it (vagrant sudo)' below Teammate mode
3. Implement Space toggle + force skipPermissions: When vagrant toggled on, force skipPermissions on. When toggled off, restore previous value
4. Update focusCount for both NewDialog and ForkDialog: Increment by 1 for the new checkbox
5. Update GetOptions to include UseVagrantMode: Return UseVagrantMode in ClaudeOptions
6. Build and verify: go build ./... `go build ./...`

### task-5.2: Boot progress display

- **Model:** sonnet
- **Wave:** 5
- **Complexity:** medium
- **Blocked by:** task-4.1, task-3.3
- **Modifies:** internal/ui/sessionlist.go

**Steps:**
1. Add boot phase and elapsed timer to session list rendering: Show 'my-project ⟳ Provisioning... (2m 34s)' for vagrant sessions during boot
2. Add tip display in detail pane: Show rotating tips (8s interval) in right-side detail pane during boot
3. Stop tips on BootPhaseReady: Clear tip display and hide box when VM reaches ready state
4. Build and verify: go build ./... `go build ./...`

### task-5.3: Stale VM cleanup (Shift+D)

- **Model:** sonnet
- **Wave:** 5
- **Complexity:** medium
- **Blocked by:** task-4.1
- **Modifies:** internal/ui/app.go, internal/ui/cleanup_dialog.go

**Steps:**
1. Implement checkStaleSuspendedVMs: Background goroutine on startup + after session stop, threshold=3
2. Implement ListSuspendedAgentDeckVMs: vagrant global-status --machine-readable, cross-reference with agent-deck sessions
3. Create cleanup dialog TUI component: Shift+D keybinding, checkbox list with project paths, suspend age, estimated size
4. Implement DestroySuspendedVMs: Sequential vagrant destroy -f for selected VMs, progress inline
5. Update associated session status after VM destruction: After destroying VMs, update agent-deck session state/DB to reflect VM no longer exists. Sessions should show 'VM destroyed' status.
6. Build and verify: go build ./... `go build ./...`

### task-5.4: Apple Silicon kext detection in TUI

- **Model:** haiku
- **Wave:** 5
- **Complexity:** low
- **Blocked by:** task-4.1, task-2.3
- **Modifies:** internal/session/instance.go

**Steps:**
1. Ensure wrapVagrantUpError is called in applyVagrantWrapper: Catch stderr from vagrant up and wrap with user-friendly message
2. Surface wrapped error in TUI: Show 'VirtualBox requires approval in System Settings' in session status
3. Build and verify: go build ./... `go build ./...`

### task-6.1: MockVagrantProvider + interface check

- **Model:** haiku
- **Wave:** 6
- **Complexity:** low
- **Blocked by:** task-2.1
- **Creates:** internal/vagrant/mock_provider_test.go
- **Tests:** internal/vagrant/mock_provider_test.go

**Steps:**
1. Create MockVagrantProvider struct: Configurable return values per method: installed, status, health, errors, sessionCount, configDrift, wrappedCmd
2. Implement all VagrantProvider methods on mock: Return configured values
3. Add compile-time interface check: var _ VagrantProvider = (*MockVagrantProvider)(nil)
4. Add compile-time Manager check: TestManagerImplementsVagrantProvider
5. Run tests: Verify compilation `go test ./internal/vagrant/ -run TestManager -count=1 -v`

### task-6.2: Manager unit tests

- **Model:** sonnet
- **Wave:** 6
- **Complexity:** medium
- **Blocked by:** task-2.1, task-2.2, task-2.3, task-2.4, task-2.5, task-2.6, task-2.7, task-3.4
- **Modifies:** internal/vagrant/manager_test.go, internal/vagrant/vagrantfile_test.go, internal/vagrant/bootphase_test.go, internal/vagrant/preflight_test.go, internal/vagrant/sessions_test.go, internal/vagrant/wrap_test.go, internal/vagrant/sync_test.go
- **Tests:** internal/vagrant/manager_test.go

**Steps:**
1. Verify all manager tests pass: Run full test suite for vagrant package `go test ./internal/vagrant/ -count=1 -v`
2. Add any missing test cases from design doc: Cross-reference testing strategy section
3. Check coverage: go test ./internal/vagrant/ -coverprofile=coverage.out `go test ./internal/vagrant/ -coverprofile=coverage.out && go tool cover -func=coverage.out`

### task-6.3: MCP unit tests

- **Model:** haiku
- **Wave:** 6
- **Complexity:** low
- **Blocked by:** task-3.1
- **Modifies:** internal/vagrant/mcp_test.go
- **Tests:** internal/vagrant/mcp_test.go

**Steps:**
1. Verify all MCP tests pass: Run MCP-specific tests `go test ./internal/vagrant/ -run TestMCP -count=1 -v`
2. Add edge case tests: Empty MCPs, non-localhost URLs, duplicate ports

### task-6.4: Health check unit tests

- **Model:** haiku
- **Wave:** 6
- **Complexity:** low
- **Blocked by:** task-2.5
- **Modifies:** internal/vagrant/health_test.go
- **Tests:** internal/vagrant/health_test.go

**Steps:**
1. Verify health check tests pass: Run health-specific tests `go test ./internal/vagrant/ -run TestHealthCheck -count=1 -v`
2. Add timeout edge case: TestHealthCheckPhase2Timeout confirms 5s behavior

### task-6.5: Instance lifecycle unit tests with mock

- **Model:** sonnet
- **Wave:** 6
- **Complexity:** medium
- **Blocked by:** task-4.1, task-4.2, task-4.3, task-6.1
- **Modifies:** internal/session/instance_test.go
- **Tests:** internal/session/instance_test.go

**Steps:**
1. Write mock provider lifecycle tests: TestMockProviderStartLifecycle, TestMockProviderStopSuspends, TestMockProviderHealthCheckIntegration, TestMockProviderRestartRecovery, TestStartWaitsForInFlightSuspend, TestStartTimeoutOnHungSuspend
2. Run tests: Verify all lifecycle paths with mock `go test ./internal/session/ -run TestMockProvider -count=1 -v`

### task-6.6: Credential guard tests

- **Model:** haiku
- **Wave:** 6
- **Complexity:** low
- **Blocked by:** task-3.2
- **Modifies:** internal/vagrant/skill_test.go
- **Tests:** internal/vagrant/skill_test.go

**Steps:**
1. Verify credential guard tests pass: Run credential-specific tests `go test ./internal/vagrant/ -run TestCredential -count=1 -v`
2. Add rsync exclusion test: TestRsyncExcludesCredentialFiles

### task-6.7: Config drift tests

- **Model:** haiku
- **Wave:** 6
- **Complexity:** low
- **Blocked by:** task-2.6
- **Modifies:** internal/vagrant/drift_test.go
- **Tests:** internal/vagrant/drift_test.go

**Steps:**
1. Verify config drift tests pass: Run drift-specific tests `go test ./internal/vagrant/ -run TestConfigHash -count=1 -v`

### task-6.8: Tips tests

- **Model:** haiku
- **Wave:** 6
- **Complexity:** low
- **Blocked by:** task-3.3
- **Modifies:** internal/vagrant/tips_test.go
- **Tests:** internal/vagrant/tips_test.go

**Steps:**
1. Verify tips tests pass: Run tip-specific tests `go test ./internal/vagrant/ -run TestTip -count=1 -v`

### task-6.9: UI tests

- **Model:** sonnet
- **Wave:** 6
- **Complexity:** medium
- **Blocked by:** task-5.1
- **Modifies:** internal/ui/claudeoptions_test.go
- **Tests:** internal/ui/claudeoptions_test.go

**Steps:**
1. Write UI tests: TestCheckboxRendersAfterTeammateMode, TestSpaceTogglesVagrantMode, TestVagrantForcesSkipPermissions, TestErrorShowsVMCrashedMessage
2. Run tests: Verify UI behavior `go test ./internal/ui/ -run TestCheckbox -count=1 -v`

### task-6.10: Documentation updates

- **Model:** haiku
- **Wave:** 6
- **Complexity:** low
- **Blocked by:** task-5.1
- **Modifies:** skills/agent-deck/references/config-reference.md, README.md

**Steps:**
1. Add [vagrant] section to config-reference.md: Full key table (memory_mb, cpus, box, auto_suspend, auto_destroy, health_check_interval, host_gateway_ip, synced_folder_type, provision_packages, provision_packages_exclude, npm_packages, provision_script, vagrantfile, port_forwards, env, forward_proxy_env), examples, and link to design doc
2. Add Vagrant Mode overview to README: User-facing overview: prerequisites (Vagrant, VirtualBox 7.0+), how to enable (TUI checkbox), what Claude gets inside VM, recovery/troubleshooting table
3. Add Vagrant Mode examples to README: Minimal, web dev, data science, custom Vagrantfile examples from design doc

## Phases (Conceptual Grouping)

### phase-1: Foundation (Config & Types)
Establish the type system, interfaces, and config structures that all subsequent tasks depend on.
Tasks: task-1.1, task-1.2, task-1.3

### phase-2: Core Vagrant Manager
Build the VagrantManager with full lifecycle management, Vagrantfile generation, preflight checks, health monitoring, drift detection, and multi-session support.
Tasks: task-2.1, task-2.2, task-2.3, task-2.4, task-2.5, task-2.6, task-2.7

### phase-3: MCP & Skill Integration
MCP config generation for Vagrant VMs, SSH reverse tunnel support, sudo skill with credential guard, loading tips, and command wrapping.
Tasks: task-3.1, task-3.2, task-3.3, task-3.4, task-3.5

### phase-4: Instance Lifecycle Integration
Wire Vagrant lifecycle into instance.go — start/stop/restart hooks, health monitoring, config drift handling, and multi-session prompting.
Tasks: task-4.1, task-4.2, task-4.3, task-4.4, task-4.5

### phase-5: UI Integration
TUI checkbox, boot progress display, stale VM cleanup dialog, and Apple Silicon detection.
Tasks: task-5.1, task-5.2, task-5.3, task-5.4

### phase-6: Testing & Hardening
Mock provider, comprehensive unit tests across all modules, coverage verification, and documentation.
Tasks: task-6.1, task-6.2, task-6.3, task-6.4, task-6.5, task-6.6, task-6.7, task-6.8, task-6.9, task-6.10

