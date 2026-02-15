# Vagrant Mode (Just Do It) — Agent Team Plan

> Source: `docs/plans/2026-02-14-vagrant-mode-design.md`
> Generated: 2026-02-15

**Goal:** Add 'Just do it (vagrant sudo)' checkbox to agent-deck that auto-manages a Vagrant VM lifecycle and runs Claude Code inside it with --dangerously-skip-permissions and sudo access.

**Architecture:** Wrapper Command approach — checkbox in TUI, VagrantManager handles VM lifecycle, commands wrapped via 'vagrant ssh --' with SSH reverse tunnels and SendEnv. VagrantProvider interface for testability.

**Tech Stack:** Go 1.24, Bubble Tea TUI, Vagrant, VirtualBox, tmux, SSH

---

## Summary

| Metric | Value |
|--------|-------|
| Total Tasks | 27 |
| Total Waves | 6 |
| Avg Parallelism | 4.5x |
| Max Parallelism | 9 |
| Critical Path | 6 tasks |

---

## Wave Execution Strategy

### Wave 1: Foundation — 3 tasks (parallel)

| Task | Name | Model | Agents | Complexity | MCP Tools |
|------|------|-------|--------|------------|-----------|
| 1.1 | VagrantSettings struct + config parsing | sonnet | tdd-guide | medium | — |
| 1.2 | VagrantProvider interface | sonnet | — | low | — |
| 1.3 | UseVagrantMode in ClaudeOptions | haiku | tdd-guide | low | — |

**Checkpoint:** Review VagrantSettings fields, VagrantProvider interface methods, and UseVagrantMode field.

```
    ┌─────────────────────────────────────────────────────────────────────┐
    │                                                                     │
    │   ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░   │
    │   ░░  ┌─────┐  ┌─────┐  ┌─────┐                               ░░   │
    │   ░░  │Types│──│Iface│──│ Opts│    ◇ FOUNDATION ◇              ░░   │
    │   ░░  └──┬──┘  └──┬──┘  └──┬──┘                               ░░   │
    │   ░░     │        │        │       3 tasks · 3/3 parallel      ░░   │
    │   ░░     ▼        ▼        ▼                                   ░░   │
    │   ░░   ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓   Bedrock laid.              ░░   │
    │   ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░   │
    │           W A V E   1   C O M P L E T E                             │
    └─────────────────────────────────────────────────────────────────────┘
```

---

### Wave 2: Core Infrastructure — 7 tasks (parallel)

| Task | Name | Model | Agents | Complexity | MCP Tools |
|------|------|-------|--------|------------|-----------|
| 2.1 | Manager struct + basic lifecycle | sonnet | tdd-guide | high | — |
| 2.2 | Vagrantfile template generation | sonnet | tdd-guide | high | — |
| 2.3 | Boot phase parser | sonnet | tdd-guide | medium | — |
| 2.4 | Preflight checks | sonnet | tdd-guide | medium | — |
| 2.5 | Two-phase health check | sonnet | tdd-guide | medium | — |
| 2.6 | Config drift detection | haiku | tdd-guide | low | — |
| 2.7 | Multi-session tracking | sonnet | tdd-guide | medium | — |

**Checkpoint:** Review Manager methods match interface, Vagrantfile template, preflight thresholds, health check logic, config hash.

```
         ◆━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━◆
         ┃                                                       ┃
         ┃   ██   ██  █████  ██    ██ ██████     ████████         ┃
         ┃   ██   ██ ██   ██ ██    ██ ██              ██          ┃
         ┃   ██ █ ██ ███████ ██    ██ █████      ██████           ┃
         ┃   ██████  ██   ██  ██  ██  ██         ██              ┃
         ┃    ██ ██  ██   ██   ████   ██████     ████████         ┃
         ┃                                                       ┃
         ┃   ┌────┐ ┌────┐ ┌────┐ ┌────┐ ┌────┐ ┌────┐ ┌────┐  ┃
         ┃   │Mgr │ │VFile│ │Boot│ │Pref│ │Hlth│ │Drft│ │Multi│ ┃
         ┃   └─┬──┘ └─┬──┘ └─┬──┘ └─┬──┘ └─┬──┘ └─┬──┘ └─┬──┘  ┃
         ┃     ▼      ▼      ▼      ▼      ▼      ▼      ▼      ┃
         ┃   ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓   ┃
         ┃       C O R E   I N F R A S T R U C T U R E           ┃
         ┃         7 tasks  ·  7/7 parallel  ·  100%              ┃
         ◆━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━◆
```

---

### Wave 3: MCP & Skills — 5 tasks (parallel)

| Task | Name | Model | Agents | Complexity | MCP Tools |
|------|------|-------|--------|------------|-----------|
| 3.1 | MCP config for Vagrant | sonnet | tdd-guide | high | — |
| 3.2 | Static skill + credential guard hook | sonnet | tdd-guide, security-reviewer | medium | — |
| 3.3 | Loading tips (100 tips) | haiku | — | low | — |
| 3.4 | Command wrapping with SSH tunnels | sonnet | tdd-guide | high | — |
| 3.5 | SyncClaudeConfig | sonnet | tdd-guide | medium | — |

**Checkpoint:** Review MCP config bypass of pool sockets, credential guard patterns, 100 tips, WrapCommand format, config sync.

```
    ┌───────────────────────────────────────────────────────────────┐
    │                  ╔═══════════════════════╗                     │
    │    ◐─────────────║  M C P  &  S K I L L S ║──────────────◑    │
    │    │             ╚═══════════════════════╝              │     │
    │    │                                                    │     │
    │  ┌─┴──┐  ┌─────┐  ┌─────┐  ┌─────┐  ┌─────┐          │     │
    │  │MCP │──│Skill│──│Tips │──│Wrap │──│Sync │          │     │
    │  │.json│  │+Hook│  │×100 │  │SSH-R│  │Cfg  │          │     │
    │  └────┘  └─────┘  └─────┘  └─────┘  └─────┘          │     │
    │    │             ◇ Tunnels Forged ◇                     │     │
    │    ◐───────────── 5 tasks · 5/5 parallel ──────────────◑     │
    │                     W A V E   3                               │
    └───────────────────────────────────────────────────────────────┘
```

---

### Wave 4: Instance Lifecycle — 5 tasks (3 parallel)

| Task | Name | Model | Agents | Complexity | MCP Tools |
|------|------|-------|--------|------------|-----------|
| 4.1 | Instance vagrant lifecycle hooks | opus | tdd-guide, code-reviewer | high | sequential-thinking |
| 4.2 | Vagrant restart flow | sonnet | tdd-guide | medium | — |
| 4.3 | Health check integration in UpdateStatus | sonnet | tdd-guide | medium | — |
| 4.4 | Config drift + re-provision in Start | haiku | tdd-guide | low | — |
| 4.5 | Multi-session prompt + share/separate | sonnet | tdd-guide | medium | — |

**Note:** Tasks 4.2, 4.3, and 4.5 depend on 4.1. Task 4.4 can parallel with 4.1.

**Checkpoint:** Review vagrantProvider interface usage, vmOpDone channel pattern, restart state machine, drift flow.

```
              ⚡━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━⚡
              ┃                                            ┃
              ┃   ┌──Start──┐   ┌──Stop──┐   ┌─Restart─┐  ┃
              ┃   │ up/wrap │──▶│suspend │──▶│recover  │  ┃
              ┃   └────┬────┘   └────┬───┘   └────┬────┘  ┃
              ┃        │             │             │        ┃
              ┃        ▼             ▼             ▼        ┃
              ┃   ┌─Health──┐   ┌─Drift──┐   ┌─Multi──┐   ┃
              ┃   │30s poll │   │re-prov │   │share?  │   ┃
              ┃   └─────────┘   └────────┘   └────────┘   ┃
              ┃                                            ┃
              ┃   I N S T A N C E   L I F E C Y C L E      ┃
              ┃     5 tasks · 3 parallel · Wave 4           ┃
              ⚡━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━⚡
```

---

### Wave 5: UI Integration — 4 tasks (parallel)

| Task | Name | Model | Agents | Complexity | MCP Tools |
|------|------|-------|--------|------------|-----------|
| 5.1 | Vagrant mode checkbox in TUI | sonnet | code-reviewer | medium | — |
| 5.2 | Boot progress display | sonnet | — | medium | context7 |
| 5.3 | Stale VM cleanup (Shift+D) | sonnet | — | medium | — |
| 5.4 | Apple Silicon kext detection in TUI | haiku | — | low | — |

**Checkpoint:** Review checkbox placement, force-skip logic, boot progress rendering, cleanup dialog, kext error message.

```
    ╔══════════════════════════════════════════════════════════════╗
    ║                                                              ║
    ║   ┌──────────────────────────────────────────────┐           ║
    ║   │  New Session                                 │           ║
    ║   │                                              │           ║
    ║   │  [x] Just do it (vagrant sudo)     ← NEW    │           ║
    ║   │  [x] Skip permissions (forced)               │           ║
    ║   │                                              │           ║
    ║   │  my-project  ⟳ Provisioning... (2m 34s)     │           ║
    ║   │  💡 Use NFS for 10x faster I/O...           │           ║
    ║   └──────────────────────────────────────────────┘           ║
    ║                                                              ║
    ║           U I   I N T E G R A T I O N                        ║
    ║             4 tasks · 4/4 parallel · Wave 5                  ║
    ╚══════════════════════════════════════════════════════════════╝
```

---

### Wave 6: Hardening & Polish — 9 tasks (parallel)

| Task | Name | Model | Agents | Complexity | MCP Tools |
|------|------|-------|--------|------------|-----------|
| 6.1 | MockVagrantProvider + interface check | haiku | — | low | — |
| 6.2 | Manager unit tests | sonnet | tdd-guide | medium | — |
| 6.3 | MCP unit tests | haiku | tdd-guide | low | — |
| 6.4 | Health check unit tests | haiku | tdd-guide | low | — |
| 6.5 | Instance lifecycle unit tests with mock | sonnet | tdd-guide | medium | — |
| 6.6 | Credential guard tests | haiku | tdd-guide | low | — |
| 6.7 | Config drift tests | haiku | — | low | — |
| 6.8 | Tips tests | haiku | — | low | — |
| 6.9 | UI tests | sonnet | tdd-guide | medium | context7 |

**Checkpoint:** All tests pass, build succeeds, coverage ≥80%.

```
                        *    .  *       .             *
         *   .        *          .     *    .    *    .        *
      .    *    .  ╔══════════════════════════════════╗    .    *
     .        .    ║  ★  A L L   W A V E S  ★        ║  .        .
       *  .     *  ║     C O M P L E T E !           ║     *  .
    .        .     ╚══════════╦═══╦════════════════════╝  .        .
         .    *         ░░░░░░║   ║░░░░░░        *    .
      *     .     *     ░░████║   ║████░░     *     .     *
         .         ░░░░░█████║   ║█████░░░░░         .
      .    *    ░░░░████████║     ║████████░░░░    *    .
            ░░░░███████████╚═════╝███████████░░░░
        ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
          27 tasks  ·  6 waves  ·  4.5x parallelism
             V A G R A N T   M O D E   R E A D Y
```

**Manual actions:** Integration test with real Vagrant VM, verify MCP connectivity inside VM.

---

## Phase 1: Foundation (Config & Types)

### Task 1.1: VagrantSettings struct + config parsing

**Agent Orchestration:**
- Model: sonnet (Standard struct creation with TDD)
- Agents: tdd-guide
- Skills: tdd-workflow
- Permissions: acceptEdits
- Complexity: medium
- Wave: 1 | Parallel with: 1.2, 1.3

**Files:**
- Modify: `internal/session/userconfig.go`
- Test: `internal/session/userconfig_test.go`

**Steps:**
1. Write tests for VagrantSettings defaults and overrides
2. Add VagrantSettings struct with all 14+ fields
3. Add PortForward struct
4. Add Vagrant field to UserConfig struct
5. Add GetVagrantSettings() with defaults
6. Run tests

**Progress:** 0/6 steps (0%) — pending

---

### Task 1.2: VagrantProvider interface

**Agent Orchestration:**
- Model: sonnet (Interface definition)
- Permissions: acceptEdits
- Complexity: low
- Wave: 1 | Parallel with: 1.1, 1.3

**Files:**
- Create: `internal/vagrant/provider.go`

**Steps:**
1. Create internal/vagrant/ package directory
2. Define VagrantProvider interface (24 methods)
3. Define BootPhase type and 8 constants
4. Define VMHealth struct

**Progress:** 0/4 steps (0%) — pending

---

### Task 1.3: UseVagrantMode in ClaudeOptions

**Agent Orchestration:**
- Model: haiku (Simple field addition)
- Agents: tdd-guide
- Skills: tdd-workflow
- Permissions: acceptEdits
- Complexity: low
- Wave: 1 | Parallel with: 1.1, 1.2

**Files:**
- Modify: `internal/session/tooloptions.go`
- Test: `internal/session/tooloptions_test.go`

**Steps:**
1. Write test for UseVagrantMode forcing skip permissions
2. Add UseVagrantMode field to ClaudeOptions
3. Update ToArgs to force --dangerously-skip-permissions
4. Update ToArgsForFork similarly
5. Run tests

**Progress:** 0/5 steps (0%) — pending

---

## Phase 2: Core Vagrant Manager

### Task 2.1: Manager struct + basic lifecycle

**Agent Orchestration:**
- Model: sonnet (Core implementation with subprocess management)
- Agents: tdd-guide
- Skills: tdd-workflow
- Permissions: acceptEdits
- Complexity: high
- Wave: 2 | Parallel with: 2.2-2.7

**Files:**
- Create: `internal/vagrant/manager.go`
- Test: `internal/vagrant/manager_test.go`

**Steps:**
1. Write tests for Manager creation and IsInstalled
2. Define Manager struct
3. Implement NewManager constructor
4. Implement IsInstalled via exec.LookPath
5. Implement vagrantCmd helper
6. Implement Status via vagrant status --machine-readable
7. Implement EnsureRunning with phase callback
8. Implement Suspend, Resume, Destroy, ForceRestart, Reload, Provision
9. Verify Manager implements VagrantProvider interface
10. Run tests

**Progress:** 0/10 steps (0%) — pending

---

### Task 2.2: Vagrantfile template generation

**Agent Orchestration:**
- Model: sonnet (Template generation with string interpolation)
- Agents: tdd-guide
- Skills: tdd-workflow
- Permissions: acceptEdits
- Complexity: high
- Wave: 2 | Parallel with: 2.1, 2.3-2.7

**Files:**
- Modify: `internal/vagrant/manager.go`
- Test: `internal/vagrant/manager_test.go`

**Steps:**
1. Write tests for EnsureVagrantfile (8 test cases)
2. Write tests for package resolution (4 test cases)
3. Write tests for hostname generation (3 test cases)
4. Implement resolvedPackages() helper
5. Implement hostname sanitization
6. Implement EnsureVagrantfile with full template
7. Handle rsync credential exclusion
8. Run tests

**Progress:** 0/8 steps (0%) — pending

---

### Task 2.3: Boot phase parser

**Agent Orchestration:**
- Model: sonnet (String parsing with pattern matching)
- Agents: tdd-guide
- Skills: tdd-workflow
- Permissions: acceptEdits
- Complexity: medium
- Wave: 2 | Parallel with: 2.1-2.2, 2.4-2.7

**Files:**
- Modify: `internal/vagrant/manager.go`
- Test: `internal/vagrant/manager_test.go`

**Steps:**
1. Write tests for boot phase parsing
2. Implement parseBootPhase function
3. Implement wrapVagrantUpError (Apple Silicon detection)
4. Write test for Apple Silicon detection
5. Run tests

**Progress:** 0/5 steps (0%) — pending

---

### Task 2.4: Preflight checks

**Agent Orchestration:**
- Model: sonnet (System checks with subprocess calls)
- Agents: tdd-guide
- Skills: tdd-workflow
- Permissions: acceptEdits
- Complexity: medium
- Wave: 2 | Parallel with: 2.1-2.3, 2.5-2.7

**Files:**
- Modify: `internal/vagrant/manager.go`
- Test: `internal/vagrant/manager_test.go`

**Steps:**
1. Write preflight tests (9 test cases)
2. Implement CheckDiskSpace (syscall.Statfs)
3. Implement CheckVBoxInstalled (VBoxManage version parsing)
4. Implement IsBoxCached (vagrant box list parsing)
5. Implement PreflightCheck (combined: Vagrant + VBox + disk)
6. Run tests

**Progress:** 0/6 steps (0%) — pending

---

### Task 2.5: Two-phase health check

**Agent Orchestration:**
- Model: sonnet (Subprocess management with timeouts)
- Agents: tdd-guide
- Skills: tdd-workflow
- Permissions: acceptEdits
- Complexity: medium
- Wave: 2 | Parallel with: 2.1-2.4, 2.6-2.7

**Files:**
- Modify: `internal/vagrant/manager.go`
- Test: `internal/vagrant/manager_test.go`

**Steps:**
1. Write health check tests (4 test cases)
2. Implement HealthCheck Phase 1 (vagrant status)
3. Implement HealthCheck Phase 2 (SSH liveness probe, 5s timeout)
4. Add 30s TTL caching for Phase 1
5. Implement vmStateMessage helper
6. Run tests

**Progress:** 0/6 steps (0%) — pending

---

### Task 2.6: Config drift detection

**Agent Orchestration:**
- Model: haiku (Simple hashing and file I/O)
- Agents: tdd-guide
- Skills: tdd-workflow
- Permissions: acceptEdits
- Complexity: low
- Wave: 2 | Parallel with: 2.1-2.5, 2.7

**Files:**
- Modify: `internal/vagrant/manager.go`
- Test: `internal/vagrant/manager_test.go`

**Steps:**
1. Write config drift tests (6 test cases)
2. Implement configHash() with SHA-256
3. Implement HasConfigDrift()
4. Implement WriteConfigHash()
5. Run tests

**Progress:** 0/5 steps (0%) — pending

---

### Task 2.7: Multi-session tracking

**Agent Orchestration:**
- Model: sonnet (Concurrent session tracking with mutex)
- Agents: tdd-guide
- Skills: tdd-workflow
- Permissions: acceptEdits
- Complexity: medium
- Wave: 2 | Parallel with: 2.1-2.6

**Files:**
- Modify: `internal/vagrant/manager.go`
- Test: `internal/vagrant/manager_test.go`

**Steps:**
1. Write multi-session tests (7 test cases)
2. Implement RegisterSession/UnregisterSession
3. Implement SessionCount/IsLastSession
4. Implement SetDotfilePath
5. Implement lockfile persistence
6. Run tests

**Progress:** 0/6 steps (0%) — pending

---

## Phase 3: MCP & Skill Integration

### Task 3.1: MCP config for Vagrant

**Agent Orchestration:**
- Model: sonnet (MCP config integration)
- Agents: tdd-guide
- Skills: tdd-workflow
- Permissions: acceptEdits
- Complexity: high
- Wave: 3 | Parallel with: 3.2-3.5

**Files:**
- Create: `internal/vagrant/mcp.go`
- Test: `internal/vagrant/mcp_test.go`

**Steps:**
1. Write MCP tests (5 test cases)
2. Implement CollectHTTPMCPPorts (port extraction)
3. Implement CollectEnvVarNames (merge MCP + vagrant + ANTHROPIC_API_KEY)
4. Implement WriteMCPJsonForVagrant (STDIO fallback, no pool)
5. Implement GetMCPPackages (npm extraction)
6. Run tests

**Progress:** 0/6 steps (0%) — pending

---

### Task 3.2: Static skill + credential guard hook

**Agent Orchestration:**
- Model: sonnet (Security-sensitive credential guard)
- Agents: tdd-guide, security-reviewer
- Skills: tdd-workflow, security-review
- Permissions: default
- Complexity: medium
- Wave: 3 | Parallel with: 3.1, 3.3-3.5

**Files:**
- Create: `internal/vagrant/skill.go`
- Test: `internal/vagrant/skill_test.go`

**Steps:**
1. Write skill and credential guard tests (8 test cases)
2. Implement GetVagrantSudoSkill()
3. Implement EnsureSudoSkill()
4. Implement credential guard hook injection
5. Implement GetCredentialGuardHook()
6. Run tests

**Progress:** 0/6 steps (0%) — pending

---

### Task 3.3: Loading tips (100 tips)

**Agent Orchestration:**
- Model: haiku (Data-heavy file with simple struct)
- Permissions: acceptEdits
- Complexity: low
- Wave: 3 | Parallel with: 3.1-3.2, 3.4-3.5

**Files:**
- Create: `internal/vagrant/tips.go`
- Test: `internal/vagrant/tips_test.go`

**Steps:**
1. Write tips tests
2. Define Tip struct
3. Embed 50 Vagrant best practice tips
4. Embed 50 world fact tips
5. Implement GetRandomTip() and GetNextTip()
6. Run tests

**Progress:** 0/6 steps (0%) — pending

---

### Task 3.4: Command wrapping with SSH tunnels

**Agent Orchestration:**
- Model: sonnet (Complex string building with SSH flags)
- Agents: tdd-guide
- Skills: tdd-workflow
- Permissions: acceptEdits
- Complexity: high
- Wave: 3 | Parallel with: 3.1-3.3, 3.5

**Files:**
- Modify: `internal/vagrant/manager.go`
- Test: `internal/vagrant/manager_test.go`

**Steps:**
1. Write WrapCommand tests (11 test cases)
2. Implement WrapCommand with -R, -o SendEnv, -t flags
3. Implement collectProxyEnvVars
4. Add polling env var auto-injection for VirtualBox
5. Run tests

**Progress:** 0/5 steps (0%) — pending

---

### Task 3.5: SyncClaudeConfig

**Agent Orchestration:**
- Model: sonnet (File I/O and SSH command construction)
- Agents: tdd-guide
- Skills: tdd-workflow
- Permissions: acceptEdits
- Complexity: medium
- Wave: 3 | Parallel with: 3.1-3.4

**Files:**
- Modify: `internal/vagrant/manager.go`
- Test: `internal/vagrant/manager_test.go`

**Steps:**
1. Write SyncClaudeConfig tests
2. Implement SyncClaudeConfig
3. Run tests

**Progress:** 0/3 steps (0%) — pending

---

## Phase 4: Instance Lifecycle Integration

### Task 4.1: Instance vagrant lifecycle hooks

**Agent Orchestration:**
- Model: opus (Complex integration with goroutine coordination)
- Agents: tdd-guide, code-reviewer
- MCP Tools: sequential-thinking (optional, for concurrency reasoning)
- Skills: tdd-workflow
- Permissions: default
- Complexity: high
- Wave: 4

**Files:**
- Modify: `internal/session/instance.go`
- Test: `internal/session/instance_test.go`

**Steps:**
1. Add vagrant fields to Instance struct
2. Add IsVagrantMode() helper
3. Implement applyVagrantWrapper in Start/StartWithMessage
4. Implement stopVagrant with goroutine + done channel
5. Implement destroyVagrant with goroutine + done channel
6. Implement waitForVagrantOp with 60s timeout
7. Wire up Stop() and Delete() hooks
8. Wire up Start() to wait on in-flight ops

**Progress:** 0/8 steps (0%) — pending

---

### Task 4.2: Vagrant restart flow

**Agent Orchestration:**
- Model: sonnet (State machine implementation)
- Agents: tdd-guide
- Skills: tdd-workflow
- Permissions: acceptEdits
- Complexity: medium
- Wave: 4 | Blocked by: 4.1

**Files:**
- Modify: `internal/session/instance.go`
- Test: `internal/session/instance_test.go`

**Steps:**
1. Write restart tests for all VM states
2. Implement restartVagrantSession state machine
3. Wire into Restart() method
4. Run tests

**Progress:** 0/4 steps (0%) — pending

---

### Task 4.3: Health check integration in UpdateStatus

**Agent Orchestration:**
- Model: sonnet (Integration into existing polling loop)
- Agents: tdd-guide
- Skills: tdd-workflow
- Permissions: acceptEdits
- Complexity: medium
- Wave: 4 | Blocked by: 4.1

**Files:**
- Modify: `internal/session/instance.go`
- Test: `internal/session/instance_test.go`

**Steps:**
1. Write health check integration tests
2. Add 30s health check to UpdateStatus
3. Add immediate check for ungraceful shutdown
4. Add contextual error messages
5. Run tests

**Progress:** 0/5 steps (0%) — pending

---

### Task 4.4: Config drift + re-provision in Start

**Agent Orchestration:**
- Model: haiku (Simple conditional check)
- Agents: tdd-guide
- Skills: tdd-workflow
- Permissions: acceptEdits
- Complexity: low
- Wave: 4 | Parallel with: 4.1

**Files:**
- Modify: `internal/session/instance.go`
- Test: `internal/session/instance_test.go`

**Steps:**
1. Write drift integration tests
2. Add drift check after EnsureRunning in Start
3. Run tests

**Progress:** 0/3 steps (0%) — pending

---

### Task 4.5: Multi-session prompt + share/separate flow

**Agent Orchestration:**
- Model: sonnet (Multi-session logic)
- Agents: tdd-guide
- Skills: tdd-workflow
- Permissions: default
- Complexity: medium
- Wave: 4 | Blocked by: 4.1

**Files:**
- Modify: `internal/session/instance.go`
- Test: `internal/session/instance_test.go`

**Steps:**
1. Write multi-session integration tests
2. Implement VM ownership detection in Start
3. Add share/separate handling
4. Fork inheritance
5. Run tests

**Progress:** 0/5 steps (0%) — pending

---

## Phase 5: UI Integration

### Task 5.1: Vagrant mode checkbox in TUI

**Agent Orchestration:**
- Model: sonnet (TUI component following existing pattern)
- Agents: code-reviewer
- Permissions: acceptEdits
- Complexity: medium
- Wave: 5 | Parallel with: 5.2-5.4

**Files:**
- Modify: `internal/ui/claudeoptions.go`

**Steps:**
1. Add useVagrantMode field
2. Add checkbox rendering below Teammate mode
3. Implement Space toggle + force skipPermissions
4. Update focusCount
5. Update GetOptions
6. Build and verify

**Progress:** 0/6 steps (0%) — pending

---

### Task 5.2: Boot progress display

**Agent Orchestration:**
- Model: sonnet (TUI rendering with timer)
- Permissions: acceptEdits
- Complexity: medium
- Wave: 5 | Parallel with: 5.1, 5.3-5.4

**Files:**
- Modify: `internal/ui/sessionlist.go`

**Steps:**
1. Add boot phase + elapsed timer to session list
2. Add tip display in detail pane (8s rotation)
3. Stop tips on BootPhaseReady
4. Build and verify

**Progress:** 0/4 steps (0%) — pending

---

### Task 5.3: Stale VM cleanup (Shift+D)

**Agent Orchestration:**
- Model: sonnet (TUI dialog with subprocess management)
- Permissions: acceptEdits
- Complexity: medium
- Wave: 5 | Parallel with: 5.1-5.2, 5.4

**Files:**
- Modify: `internal/ui/app.go`, `internal/ui/cleanup_dialog.go`

**Steps:**
1. Implement checkStaleSuspendedVMs
2. Implement ListSuspendedAgentDeckVMs
3. Create cleanup dialog TUI component
4. Implement DestroySuspendedVMs
5. Build and verify

**Progress:** 0/5 steps (0%) — pending

---

### Task 5.4: Apple Silicon kext detection in TUI

**Agent Orchestration:**
- Model: haiku (Simple error surfacing)
- Permissions: acceptEdits
- Complexity: low
- Wave: 5 | Parallel with: 5.1-5.3

**Files:**
- Modify: `internal/session/instance.go`

**Steps:**
1. Ensure wrapVagrantUpError called in applyVagrantWrapper
2. Surface wrapped error in TUI
3. Build and verify

**Progress:** 0/3 steps (0%) — pending

---

## Phase 6: Testing & Hardening

### Task 6.1: MockVagrantProvider + interface check

**Agent Orchestration:**
- Model: haiku (Boilerplate mock struct)
- Permissions: acceptEdits
- Complexity: low
- Wave: 6 | Parallel with: 6.2-6.9

**Files:**
- Create: `internal/vagrant/mock_provider_test.go`

**Steps:**
1. Create MockVagrantProvider struct
2. Implement all VagrantProvider methods
3. Add compile-time interface check (mock)
4. Add compile-time Manager check
5. Run tests

**Progress:** 0/5 steps (0%) — pending

---

### Task 6.2: Manager unit tests — **Progress:** 0/3 (0%) — pending
### Task 6.3: MCP unit tests — **Progress:** 0/2 (0%) — pending
### Task 6.4: Health check unit tests — **Progress:** 0/2 (0%) — pending
### Task 6.5: Instance lifecycle tests with mock — **Progress:** 0/2 (0%) — pending
### Task 6.6: Credential guard tests — **Progress:** 0/2 (0%) — pending
### Task 6.7: Config drift tests — **Progress:** 0/1 (0%) — pending
### Task 6.8: Tips tests — **Progress:** 0/1 (0%) — pending
### Task 6.9: UI tests — **Progress:** 0/2 (0%) — pending
