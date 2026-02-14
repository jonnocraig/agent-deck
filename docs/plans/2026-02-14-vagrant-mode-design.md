# Vagrant Mode ("Just Do It") -- Design Document

> Generated: 2026-02-14
> Brainstorm perspectives: Architect, Implementer, Devil's Advocate, Security Analyst
> Chosen approach: Wrapper Command (Approach 1)

## Summary

Add a "Just do it (vagrant sudo)" checkbox to the New Session dialog, below "Teammate mode". When checked, agent-deck auto-manages a Vagrant VM lifecycle and runs Claude Code inside it with `--dangerously-skip-permissions` and sudo access. Based on [Running Claude Code Dangerously Safely](https://blog.emilburzo.com/2026/01/running-claude-code-dangerously-safely/).

## Acceptance Criteria

- [ ] Checkbox "Just do it (vagrant sudo)" appears below Teammate mode in NewDialog and ForkDialog
- [ ] Checking it forces skipPermissions on
- [ ] Session start runs `vagrant up` if VM not running, then wraps command in `vagrant ssh -c "..."`
- [ ] Session stop runs `vagrant suspend` (async)
- [ ] Session delete runs `vagrant destroy -f` (async)
- [ ] Vagrantfile auto-generated if missing (Ubuntu 24.04, 4GB RAM, 2 CPUs, docker/node/git/claude-code)
- [ ] Static skill preloaded giving Claude sudo guidance
- [ ] Existing Vagrantfile respected (no overwrite)
- [ ] Graceful error when Vagrant not installed
- [ ] `[vagrant]` section in config.toml for VM resources, lifecycle, provisioning, and network settings
- [ ] Vagrantfile generated from config.toml settings (packages, ports, box, resources)
- [ ] Custom provisioning script supported via `vagrant.provision_script`
- [ ] Custom Vagrantfile template supported via `vagrant.vagrantfile` (disables auto-generation)
- [ ] Port forwarding configurable via `vagrant.port_forwards`
- [ ] Additional env vars passed to VM sessions via `vagrant.env`
- [ ] MCP tools from config.toml work inside the Vagrant VM
- [ ] HTTP/SSE MCPs reachable from VM via host gateway IP rewrite
- [ ] STDIO MCPs available in VM (npm packages provisioned, pool sockets bypassed)
- [ ] Global/User scope Claude MCP configs propagated into VM
- [ ] VM crash detected and surfaced as "VM crashed — press R to restart"
- [ ] Press R on crashed session: recovers VM and resumes Claude conversation
- [ ] agent-deck crash recovery: reconnects to surviving tmux+VM+Claude automatically
- [ ] Host reboot recovery: press R recreates VM and resumes Claude via session ID

## Non-Goals

- Provider abstraction (Lima/Docker/OrbStack) -- YAGNI, Vagrant only for now
- Bidirectional sync hardening (one-way sync, credential filtering, etc.) -- users opting in understand the trade-offs
- Network firewall/whitelist inside VM -- out of scope for v1
- Prompt injection detection -- out of scope for v1
- CLI flag support (`agent-deck add --vagrant`) -- TUI only for now

## Context & Constraints

**Article approach:** Vagrant + VirtualBox VM with bidirectional shared folder sync. Claude runs with `--dangerously-skip-permissions` and sudo inside VM. VM protects host from accidental damage.

**User decisions:**
- VM lifecycle: Auto-managed (up on start, suspend on stop, destroy on delete)
- Vagrantfile: Auto-generated if missing, user's Vagrantfile respected
- Skill: Static file bundled with agent-deck
- Skip perms: Force-enabled when vagrant mode checked

**Known trade-offs:**
- VirtualBox on Apple Silicon is recent (VB 7.2) but Linux VMs work well
- First boot latency: 5-10 min (box download + provision); subsequent: 30-60s
- Bidirectional sync means Claude can modify host files (by design)
- Vagrant ecosystem declining, but still functional and widely installed

## Exploration Findings

### Perspective: Architect
- Clean separation via `internal/vagrant/` package with `VagrantManager`
- Lifecycle hooks into existing Start/Stop/Delete flow
- Command wrapping via `vagrant ssh -c "cd /vagrant && ..."`
- VM state tracked via `vagrant status --machine-readable` (no new DB fields)
- Config via `[vagrant]` TOML section with sensible defaults

### Perspective: Implementer
- 4 modified files: `claudeoptions.go`, `tooloptions.go`, `instance.go`, `userconfig.go`
- 3 new files: `internal/vagrant/manager.go`, `internal/vagrant/skill.go`, `internal/vagrant/mcp.go`
- Follows exact same pattern as teammate-mode checkbox (bool field, space toggle, renderCheckboxLine)
- Vagrantfile template embedded in Go code with `fmt.Sprintf` for memory/CPU interpolation
- MCP config generation uses VM-aware variant that rewrites URLs and bypasses pool sockets

### Perspective: Devil's Advocate
- VirtualBox dependency is a concern (Apple Silicon beta, declining ecosystem)
- Startup latency breaks the instant-session UX
- File sync performance with node_modules is poor
- Simpler alternative: wrapper command config or skill-file-only
- Recommended validating demand before building full feature
- **Counterpoint:** The feature is explicitly requested, matches an article users will follow, and the checkbox is opt-in. Power users who check "Just do it" accept these trade-offs.

### Perspective: Security Analyst
- VM escape: Low likelihood, Critical impact. Mitigate with VB feature disabling (audio, USB, clipboard off)
- Bidirectional sync: .git/hooks and package.json scripts can be weaponized. Accepted risk for v1.
- Credential exposure: API key passed via inline env var, not written to disk
- Network: VM has full internet. Accepted risk (matches article's approach)
- Prompt injection: Skill tells Claude it has sudo. Accepted risk with clear skill scoping.
- Resource limits: Vagrantfile template caps memory/CPU via config

## Approaches Considered

### Approach 1: Wrapper Command (Selected)
Add checkbox, auto-manage VM lifecycle, wrap commands. Follows article exactly. Minimal complexity. 4 modified + 2 new files.

**Pros:** Matches article, follows existing patterns, clean separation, opt-in UX
**Cons:** VirtualBox dependency, startup latency, bidirectional sync risks

### Approach 2: Provider-Agnostic VM
Same as Approach 1 but with `VMProvider` interface for Vagrant/Lima/Docker.
**Rejected:** YAGNI, premature abstraction, only Vagrant tested in article.

### Approach 3: Security-Hardened VM
Approach 1 + one-way sync, credential filtering, network whitelist, scoped skill.
**Rejected:** Contradicts "just do it" UX, over-hardens a feature whose value is unrestricted access.

## Design

### Architecture

```
User checks "Just do it" checkbox
           |
           v
ClaudeOptionsPanel.useVagrantMode = true --> forces skipPermissions = true
           |
           v
ClaudeOptions.UseVagrantMode: true (JSON-serialized, stored in SQLite)
           |
           v
instance.Start() --> VagrantManager.EnsureRunning()
                     VagrantManager.EnsureSudoSkill()
                     VagrantManager.SyncClaudeConfig()         // propagate global/user MCPs to VM
                     WriteMCPJsonForVagrant(enabledNames)       // rewrite .mcp.json for VM access
                     VagrantManager.WrapCommand(cmd, mcpEnvVars)
                     --> tmux launches wrapped command

instance.Stop()  --> tmux kill --> VagrantManager.Suspend() (async)
instance.Delete()--> tmux kill --> VagrantManager.Destroy() (async)

UpdateStatus() ─── every 60s ──→ VagrantManager.HealthCheck()
                                  |
                                  +--→ VM aborted/poweroff → StatusError + "VM crashed"
                                  +--→ VM running → no action

instance.Restart() (press R) ──→ restartVagrantSession()
                                  |
                                  +--→ VM running   → skip vagrant up, respawn Claude
                                  +--→ VM suspended → vagrant resume, respawn Claude
                                  +--→ VM aborted   → vagrant destroy + up, respawn Claude
                                  +--→ VM not_created → vagrant up, respawn Claude

agent-deck restart ──→ ReconnectSessionLazy() finds tmux session
                       UseVagrantMode restored from ToolOptionsJSON
                       HealthCheck() confirms VM state on next poll
```

### Components

**`internal/vagrant/manager.go`** -- VagrantManager struct
```go
type Manager struct {
    projectPath string
    settings    VagrantSettings
}

func NewManager(projectPath string) *Manager
func (m *Manager) IsInstalled() bool          // exec.LookPath("vagrant")
func (m *Manager) EnsureRunning() error       // vagrant up if not running
func (m *Manager) Suspend() error             // vagrant suspend
func (m *Manager) Resume() error              // vagrant resume (suspended → running, faster than up)
func (m *Manager) Destroy() error             // vagrant destroy -f
func (m *Manager) ForceRestart() error        // vagrant destroy -f && vagrant up
func (m *Manager) Reload() error              // vagrant reload (restarts VM, re-mounts shared folders)
func (m *Manager) WrapCommand(cmd string, mcpEnvVars map[string]string) string
func (m *Manager) EnsureVagrantfile() error   // generate if missing, includes MCP npm packages
func (m *Manager) EnsureSudoSkill() error     // write skill to project
func (m *Manager) Status() (string, error)    // running/suspended/not_created/aborted/poweroff
func (m *Manager) HealthCheck() (VMHealth, error) // cached status check, 60s TTL
func (m *Manager) GetMCPPackages() []string   // extract npm packages from config.toml MCPs
func (m *Manager) SyncClaudeConfig() error    // copy host Claude configs to VM with URL rewrites
```

**`internal/vagrant/mcp.go`** -- MCP config generation for Vagrant
```go
func WriteMCPJsonForVagrant(projectPath string, enabledNames []string, hostGatewayIP string) error
func RewriteURLForVagrant(url, hostGatewayIP string) string
func CollectMCPEnvVars(enabledNames []string) map[string]string
```

**`internal/vagrant/skill.go`** -- Static skill content
```go
func GetVagrantSudoSkill() string // returns skill markdown
```

**`internal/ui/claudeoptions.go`** -- Checkbox
- `useVagrantMode bool` field
- Renders below "Teammate mode"
- Space toggles, forces skipPermissions when checked

**`internal/session/tooloptions.go`** -- Options struct
- `UseVagrantMode bool` field with JSON tag
- `ToArgs()` ensures `--dangerously-skip-permissions` when vagrant mode on

**`internal/session/instance.go`** -- Lifecycle hooks and crash recovery
- `applyVagrantWrapper()` method called in Start()/StartWithMessage()
- `restartVagrantSession()` method: VM-aware restart with state detection and recovery
- Calls `WriteMCPJsonForVagrant()` instead of `WriteMCPJsonFromConfig()` when vagrant mode active
- Calls `SyncClaudeConfig()` to propagate global/user MCPs into VM
- Collects MCP env vars via `CollectMCPEnvVars()` and passes to `WrapCommand()`
- Suspend hook in Stop(), Destroy hook in Delete()
- `UpdateStatus()` extended: periodic VM health check (60s interval) for vagrant sessions
- New fields: `vagrantManager`, `lastVMHealthCheck`, `cleanShutdown`
- Contextual error messages for VM crash states (aborted, poweroff, not_created)

**`internal/session/userconfig.go`** -- Config
- `VagrantSettings` struct (see [Vagrant Configuration Reference](#vagrant-configuration-reference) for full schema)
- `PortForward` struct for port forwarding rules
- `[vagrant]` TOML section
- `GetVagrantSettings()` with defaults (4096MB, 2 CPUs, auto_suspend=true, host_gateway_ip="10.0.2.2")

```go
type VagrantSettings struct {
    MemoryMB            int               `toml:"memory_mb"`             // Default: 4096
    CPUs                int               `toml:"cpus"`                  // Default: 2
    Box                 string            `toml:"box"`                   // Default: "bento/ubuntu-24.04"
    AutoSuspend         bool              `toml:"auto_suspend"`          // Default: true
    AutoDestroy         bool              `toml:"auto_destroy"`          // Default: false
    HostGatewayIP       string            `toml:"host_gateway_ip"`       // Default: "10.0.2.2"
    SyncedFolderType    string            `toml:"synced_folder_type"`    // Default: "virtualbox"
    ProvisionPackages   []string          `toml:"provision_packages"`    // Default: see below
    NpmPackages         []string          `toml:"npm_packages"`          // Additional global npm packages
    ProvisionScript     string            `toml:"provision_script"`      // Path to custom shell script
    Vagrantfile         string            `toml:"vagrantfile"`           // Path to custom Vagrantfile (disables generation)
    HealthCheckInterval int               `toml:"health_check_interval"` // Default: 60 (seconds)
    PortForwards        []PortForward     `toml:"port_forwards"`         // Port forwarding rules
    Env                 map[string]string `toml:"env"`                   // Additional env vars for VM sessions
}

type PortForward struct {
    Guest    int    `toml:"guest"`    // Port inside VM
    Host     int    `toml:"host"`     // Port on host
    Protocol string `toml:"protocol"` // Default: "tcp"
}
```

### Vagrantfile Template

The Vagrantfile is auto-generated by `EnsureVagrantfile()` from `config.toml` settings. All values below are interpolated from the `[vagrant]` section. If a user provides their own Vagrantfile (file already exists or `vagrant.vagrantfile` points to a template), auto-generation is skipped entirely.

```ruby
Vagrant.configure("2") do |config|
  config.vm.box = "%s"  # from vagrant.box (default: "bento/ubuntu-24.04")
  config.vm.synced_folder ".", "/vagrant", type: "%s"  # from vagrant.synced_folder_type

  # Port forwarding from vagrant.port_forwards
  # config.vm.network "forwarded_port", guest: 3000, host: 3000, protocol: "tcp"

  config.vm.provider "virtualbox" do |vb|
    vb.memory = "%d"  # from vagrant.memory_mb
    vb.cpus = %d      # from vagrant.cpus
    vb.gui = false
    vb.customize ["modifyvm", :id, "--audio", "none"]
    vb.customize ["modifyvm", :id, "--usb", "off"]
  end

  config.vm.provision "shell", inline: <<-SHELL
    export DEBIAN_FRONTEND=noninteractive
    apt-get update
    # Packages from vagrant.provision_packages
    apt-get install -y docker.io nodejs npm git unzip curl build-essential
    # Claude Code (always installed)
    npm install -g @anthropic-ai/claude-code --no-audit
    # Additional npm packages from vagrant.npm_packages
    # npm install -g @custom/tool --no-audit
    # MCP packages from config.toml [mcps.*] (auto-extracted)
    # npm install -g @anthropics/exa-mcp @anthropics/slack-mcp --no-audit
    usermod -aG docker vagrant
    chown -R vagrant:vagrant /vagrant
  SHELL

  # Custom provisioning script from vagrant.provision_script (if set)
  # config.vm.provision "shell", path: "/path/to/custom-provision.sh"
end
```

**Generation logic in `EnsureVagrantfile()`:**
1. Check if `Vagrantfile` already exists in project dir → skip (log warning about manual MCP provisioning)
2. Check if `vagrant.vagrantfile` is set → copy that file instead of generating
3. Otherwise, generate from template using `[vagrant]` settings:
   - Interpolate `box`, `memory_mb`, `cpus`, `synced_folder_type`
   - Add `config.vm.network "forwarded_port"` lines for each `port_forwards` entry
   - Join `provision_packages` into `apt-get install -y` command
   - Join `npm_packages` + auto-extracted MCP packages into `npm install -g` command
   - If `provision_script` is set, add `config.vm.provision "shell", path:` line

### Command Wrapping

Input: `claude --session-id UUID --dangerously-skip-permissions`

Output: `vagrant ssh -c 'cd /vagrant && ANTHROPIC_API_KEY=... EXA_API_KEY=... claude --session-id UUID --dangerously-skip-permissions'`

API key passed as inline env var (not written to disk). MCP-specific env vars from `MCPDef.Env` and user-defined vars from `vagrant.env` also passed inline. `vagrant ssh -c` is run from the project directory where Vagrantfile lives.

### Error Handling

| Scenario | Handling |
|----------|----------|
| Vagrant not installed | Return error with install URL. UI shows toast. |
| VirtualBox not installed | `vagrant up` fails. Error shown to user. |
| First boot (box download) | Output visible in tmux session. |
| `vagrant up` fails | Error returned, session not created. |
| Existing Vagrantfile | Used as-is, no overwrite. |
| VM already running | Detected via status, skip `vagrant up`. |
| Suspend fails | Warning logged, non-blocking. |
| Destroy fails | Warning logged, session deleted anyway. |
| VM crashes (OOM/panic) | Health check detects within 60s. StatusError + contextual message. Press R to recover. |
| VirtualBox crashes | Same as VM crash. `vagrant status` shows "aborted". |
| agent-deck crashes | Automatic recovery on restart via tmux reconnection. No action needed. |
| Host reboots | Press R: `vagrant up` recreates VM, Claude resumes via session ID. |
| VM hangs (unresponsive) | No activity detected by `UpdateStatus()`. User kills + restarts. |
| Shared folder breaks | Claude errors visible in tmux. Press R triggers `vagrant reload`. |

### Static Skill Content

The skill tells Claude:
- It is running in an isolated Ubuntu 24.04 VM
- sudo access is available
- Docker, Node.js, Git are pre-installed
- Project files are at /vagrant (synced to host)
- VM can be destroyed and recreated anytime
- Use Docker for services when possible
- Changes in /vagrant are reflected on the host

### MCP Compatibility

MCP tools configured in agent-deck's `config.toml` must work inside the Vagrant VM where Claude Code runs. There are three transport types and three scopes to handle.

#### Problem

When Claude Code runs inside the VM via `vagrant ssh -c`, the MCP configs written to `.mcp.json` reference host-side resources:

| MCP Type | Host Config | Problem in VM |
|----------|------------|---------------|
| STDIO | `command: "npx", args: ["-y", "@pkg/mcp"]` | `npx` package may not be installed in VM |
| Pool socket | `command: "agent-deck", args: ["mcp-proxy", "/tmp/agentdeck-mcp-NAME.sock"]` | Host Unix socket inaccessible from VM |
| HTTP/SSE | `url: "http://localhost:30000/mcp/"` | `localhost` in VM is the VM itself, not the host |
| Global scope | `~/.claude/.claude.json` → `mcpServers` | File exists on host only |
| User scope | `~/.claude.json` → `mcpServers` | File exists on host only |

#### Solution: VM-Aware MCP Config Generation

**New function: `WriteMCPJsonForVagrant()`** -- variant of `WriteMCPJsonFromConfig()` called when `UseVagrantMode == true`.

```go
func WriteMCPJsonForVagrant(projectPath string, enabledNames []string, hostGatewayIP string) error
```

Differences from the normal write path:

1. **STDIO MCPs**: Written as plain STDIO config (no pool socket references). Pool is always bypassed for vagrant sessions. The STDIO commands work because MCP npm packages are installed in the VM during provisioning.

2. **HTTP/SSE MCPs**: URL rewritten from `localhost`/`127.0.0.1` to `10.0.2.2` (VirtualBox NAT host gateway). The VM can reach host-bound ports via this IP without explicit port forwarding.

3. **Pool sockets**: Never referenced. Vagrant sessions always use STDIO fallback for MCPs that would normally use pool sockets.

4. **Env vars**: MCP-specific env vars from `MCPDef.Env` are passed through the `vagrant ssh -c` command as inline env vars alongside `ANTHROPIC_API_KEY`.

#### Rewrite Logic

```go
func rewriteURLForVagrant(url, hostGatewayIP string) string {
    // Replace localhost/127.0.0.1 with VirtualBox NAT host gateway
    url = strings.Replace(url, "://localhost", "://"+hostGatewayIP, 1)
    url = strings.Replace(url, "://127.0.0.1", "://"+hostGatewayIP, 1)
    return url
}
```

Default `hostGatewayIP` is `10.0.2.2` (VirtualBox NAT mode). Configurable in `[vagrant]` TOML section for non-standard network setups.

#### STDIO MCP Provisioning

The Vagrantfile template's provisioning script is enhanced to install STDIO MCP npm packages. `VagrantManager.EnsureVagrantfile()` reads `config.toml` MCPs and generates install commands:

```ruby
config.vm.provision "shell", inline: <<-SHELL
  export DEBIAN_FRONTEND=noninteractive
  apt-get update
  apt-get install -y docker.io nodejs npm git unzip curl build-essential
  npm install -g @anthropic-ai/claude-code --no-audit
  # MCP packages from config.toml (auto-generated)
  npm install -g @anthropics/exa-mcp @anthropics/slack-mcp --no-audit
  usermod -aG docker vagrant
  chown -R vagrant:vagrant /vagrant
SHELL
```

MCP package extraction heuristic: for MCPs where `command == "npx"` and args contain `-y`, the next arg after `-y` is the package name. For MCPs where `command == "node"` or custom binaries, skip (user is responsible for provisioning).

If the Vagrantfile is user-provided (already exists), MCP packages are NOT injected. A warning is logged suggesting the user install MCP tools manually.

#### Global/User Scope MCP Propagation

`VagrantManager.SyncClaudeConfig()` copies host-side Claude configs into the VM with URL rewrites:

```go
func (m *Manager) SyncClaudeConfig() error
```

Steps:
1. Read host's `$CLAUDE_CONFIG_DIR/.claude.json` (global MCPs)
2. Read host's `~/.claude.json` (user MCPs)
3. Rewrite any `localhost`/`127.0.0.1` URLs to `10.0.2.2`
4. Write to VM via `vagrant ssh -c "mkdir -p ~/.claude && cat > ~/.claude/.claude.json << 'HEREDOC' ... HEREDOC"`
5. Run once during `EnsureRunning()`, after VM is up but before Claude launches

#### Command Wrapping (Updated)

The wrapped command now includes MCP env vars:

```
vagrant ssh -c 'cd /vagrant && ANTHROPIC_API_KEY=... EXA_API_KEY=... claude --session-id UUID --dangerously-skip-permissions'
```

All env vars from enabled MCP definitions (`MCPDef.Env`) are collected and passed inline.

#### Updated VagrantManager API

```go
func (m *Manager) WriteMCPJsonForVagrant(projectPath string, enabledNames []string) error
func (m *Manager) SyncClaudeConfig() error
func (m *Manager) GetMCPPackages() []string  // extract npm packages from config.toml MCPs
func (m *Manager) WrapCommand(cmd string, mcpEnvVars map[string]string) string
```

#### Limitations (v1)

- STDIO MCPs using non-npm commands (python, custom binaries) must be manually installed in the VM by the user
- MCP env vars containing single quotes will be escaped but complex values may need manual handling
- Pool socket MCPs fall back to direct STDIO (higher memory usage in VM, one MCP process per session)
- If user provides their own Vagrantfile, MCP packages are not auto-provisioned
- `10.0.2.2` is the default VirtualBox NAT gateway; non-standard VirtualBox network configs may need `[vagrant] host_gateway_ip` override

#### Error Handling (MCP-specific)

| Scenario | Handling |
|----------|----------|
| STDIO MCP command not found in VM | Claude Code reports tool unavailable. Non-blocking. |
| HTTP MCP unreachable from VM | Claude Code reports connection error. Non-blocking. |
| Host gateway IP wrong | MCP connections timeout. User configures `host_gateway_ip` in config.toml. |
| Global config sync fails | Warning logged. Local `.mcp.json` MCPs still work. |
| MCP npm install fails during provisioning | Provisioning continues. Warning in tmux output. |

### Crash Recovery & Resilience

Vagrant sessions introduce failure modes that don't exist with direct tmux sessions. The VM, VirtualBox, or agent-deck itself can crash independently. Recovery must be seamless -- the user should be able to press `R` (restart) and resume where they left off.

#### Failure Scenarios

| Scenario | What Dies | What Survives | Detection | Recovery |
|----------|-----------|---------------|-----------|----------|
| VM crashes (OOM, kernel panic) | VM + Claude | tmux pane (shows exit), agent-deck, SQLite state | `vagrant ssh -c` exits non-zero; `UpdateStatus()` → StatusError | Press R: `vagrant up` → respawn Claude with `--resume` |
| VirtualBox crashes | VM + Claude | tmux pane, agent-deck, SQLite state | Same as above | Press R: same recovery flow |
| agent-deck crashes | TUI process | tmux session (still running `vagrant ssh -c`), VM, Claude | On restart: `ReconnectSessionLazy()` finds tmux session | Automatic: reconnects to existing tmux+VM+Claude |
| agent-deck + VM crash (host reboot) | Everything | SQLite state on disk | On restart: tmux session not found, `vagrant status` → not_created | Press R: `vagrant up` → fresh Claude with `--resume` |
| VM hangs (unresponsive) | Nothing (but frozen) | tmux pane (frozen), agent-deck | `UpdateStatus()` detects no activity for extended period | User kills session, restarts. VM force-destroyed. |
| Shared folder sync breaks | File I/O inside VM | VM, Claude (but erroring), agent-deck | Claude reports file errors in tmux output | Press R: `vagrant reload` (restarts VM, re-mounts folders) |

#### VM Health Check

Add a periodic VM health check to the existing `UpdateStatus()` polling loop. Only runs for vagrant-mode sessions.

```go
func (m *Manager) HealthCheck() (VMHealth, error)
```

**VMHealth struct:**
```go
type VMHealth struct {
    State    string // "running", "suspended", "not_created", "aborted", "poweroff"
    Healthy  bool   // true if running and responsive
    Message  string // human-readable status
}
```

**Integration with `UpdateStatus()`** (`instance.go`):
- For vagrant sessions, piggyback on the existing 500ms status polling
- VM health check runs at a lower frequency: every 60 seconds (configurable)
- Uses `vagrant status --machine-readable` (~100-200ms, acceptable overhead)
- Cached in-memory with 60s TTL to avoid hammering vagrant CLI
- If VM state is "aborted" or "poweroff" while session expects "running" → set `StatusError`

```go
// In UpdateStatus(), after existing status detection:
if i.IsVagrantMode() && time.Since(i.lastVMHealthCheck) > vmHealthCheckInterval {
    i.lastVMHealthCheck = time.Now()
    health, err := i.vagrantManager.HealthCheck()
    if err == nil && !health.Healthy {
        i.setStatus(StatusError)
        i.lastError = fmt.Errorf("VM %s: %s", health.State, health.Message)
    }
}
```

#### Restart Flow for Vagrant Sessions

The existing `Restart()` method (press `R`) is extended with vagrant-aware recovery.

```
User presses R on a vagrant session
          |
          v
   Check VM status via vagrant status --machine-readable
          |
          +---> VM running? ─── yes ──→ Skip vagrant up
          |                              |
          +---> VM suspended? ─ yes ──→ vagrant resume
          |                              |
          +---> VM aborted/   ─ yes ──→ vagrant destroy -f
          |     poweroff?                vagrant up
          |                              |
          +---> VM not_created? ─ yes ─→ vagrant up
          |                              |
          v                              v
   Re-sync MCP config           Re-sync MCP config
   (WriteMCPJsonForVagrant)     (WriteMCPJsonForVagrant)
   (SyncClaudeConfig)           (SyncClaudeConfig)
          |                              |
          v                              v
   Respawn tmux pane with wrapped command:
   vagrant ssh -c 'cd /vagrant && ANTHROPIC_API_KEY=... claude --resume SESSION_ID ...'
```

**Key detail**: Claude Code's `--resume` / `--session-id` flag means Claude can pick up the conversation even after the VM is destroyed and recreated. Claude stores conversation state server-side, not inside the VM.

```go
func (i *Instance) restartVagrantSession() error {
    mgr := vagrant.NewManager(i.ProjectPath)

    // 1. Detect VM state and recover
    health, err := mgr.HealthCheck()
    if err != nil {
        return fmt.Errorf("failed to check VM health: %w", err)
    }

    switch health.State {
    case "running":
        // VM is fine, just respawn Claude
    case "saved", "suspended":
        if err := mgr.Resume(); err != nil {
            return fmt.Errorf("failed to resume VM: %w", err)
        }
    case "aborted", "poweroff":
        // VM crashed or was forced off -- destroy and recreate
        _ = mgr.Destroy() // ignore error, may already be gone
        if err := mgr.EnsureRunning(); err != nil {
            return fmt.Errorf("failed to restart VM after crash: %w", err)
        }
    case "not_created":
        if err := mgr.EnsureRunning(); err != nil {
            return fmt.Errorf("failed to create VM: %w", err)
        }
    }

    // 2. Re-sync configs
    mgr.SyncClaudeConfig()
    WriteMCPJsonForVagrant(i.ProjectPath, i.getEnabledMCPs(), mgr.Settings().HostGatewayIP)

    // 3. Respawn tmux pane with wrapped command
    // (uses existing respawn-pane -k pattern)
    return nil
}
```

#### agent-deck Crash Recovery

When agent-deck itself crashes, the recovery is largely automatic thanks to existing infrastructure:

1. **tmux session survives**: `vagrant ssh -c "... claude ..."` continues running in tmux. The VM and Claude are unaffected.
2. **SQLite state persists**: Instance data including `UseVagrantMode` in `ToolOptionsJSON` survives on disk.
3. **On restart**: `ReconnectSessionLazy()` finds the existing tmux session by name, reconnects, and `UpdateStatus()` starts polling again.
4. **No VM action needed**: The VM is already running. `HealthCheck()` confirms this on the next poll cycle.

The only edge case: if agent-deck crashes during `vagrant up` (before Claude launches), the tmux session may show a partial vagrant output. On restart:
- `UpdateStatus()` detects session as error/inactive
- User presses `R` → restart flow checks VM status → either VM is running (continue) or partially up (destroy + recreate)

#### Ungraceful Shutdown Detection

A new `Instance` field tracks whether the previous shutdown was clean:

```go
type Instance struct {
    // ... existing fields ...
    vagrantManager      *vagrant.Manager `json:"-"`
    lastVMHealthCheck   time.Time        `json:"-"`
    cleanShutdown       bool             `json:"-"` // set to true on graceful Stop/Suspend
}
```

On startup, if `UseVagrantMode` is true and `cleanShutdown` is false (default for loaded instances), the first `UpdateStatus()` call triggers an immediate VM health check instead of waiting 60s. This catches cases where the host rebooted or agent-deck was killed.

#### VagrantManager.Resume() Method

New method for resuming suspended VMs (distinct from `EnsureRunning()` which does a full `vagrant up`):

```go
func (m *Manager) Resume() error   // vagrant resume (for suspended VMs)
```

`vagrant resume` is faster than `vagrant up` for suspended VMs (~5s vs ~30-60s).

#### Error Messages in UI

When a vagrant session enters `StatusError`, the session list shows contextual messages:

| VM State | UI Message |
|----------|-----------|
| aborted | `VM crashed — press R to restart` |
| poweroff | `VM powered off — press R to restart` |
| not_created | `VM destroyed — press R to recreate` |
| running (but Claude exited) | `Claude exited — press R to resume` |
| unknown | `VM status unknown — press R to retry` |

These messages replace the generic error indicator for vagrant sessions and are stored in `Instance.lastError`.

#### Updated VagrantManager API (with recovery)

```go
func (m *Manager) HealthCheck() (VMHealth, error)    // vagrant status --machine-readable, cached 60s
func (m *Manager) Resume() error                     // vagrant resume (suspended → running)
func (m *Manager) ForceRestart() error               // vagrant destroy -f && vagrant up
func (m *Manager) Reload() error                     // vagrant reload (restarts VM, re-mounts sync)
```

#### Updated Acceptance Criteria (recovery)

- [ ] VM crash detected via health check within 60s, session shows "VM crashed" error
- [ ] Press R on crashed vagrant session: destroys old VM, creates new, resumes Claude
- [ ] agent-deck crash + restart: automatically reconnects to running vagrant session
- [ ] Host reboot + restart: press R recreates VM and resumes Claude conversation
- [ ] Suspended VM correctly resumed (not full `vagrant up`) on restart
- [ ] Vagrant reload triggered when shared folder sync breaks

### Testing Strategy

**Unit tests:**
- `TestWrapCommand` -- command wrapping with quote escaping
- `TestWrapCommandWithMCPEnvVars` -- env var forwarding in wrapped command
- `TestWrapCommandWithVagrantEnvVars` -- vagrant.env vars merged into wrapped command
- `TestEnsureVagrantfile` -- generation when missing, skip when exists
- `TestEnsureVagrantfileWithMCPs` -- npm MCP packages in provisioning script
- `TestEnsureVagrantfileWithCustomPackages` -- provision_packages and npm_packages in template
- `TestEnsureVagrantfileWithPortForwards` -- port forwarding lines in generated Vagrantfile
- `TestEnsureVagrantfileWithProvisionScript` -- custom script provisioner appended
- `TestEnsureVagrantfileCustomTemplate` -- vagrant.vagrantfile copies user's file
- `TestEnsureVagrantfileExistingRespected` -- existing Vagrantfile never overwritten
- `TestGetVagrantSudoSkill` -- skill content validation
- `TestWriteMCPJsonForVagrant` -- STDIO fallback, no pool references
- `TestRewriteURLForVagrant` -- localhost/127.0.0.1 → host gateway rewrite
- `TestGetMCPPackages` -- npm package extraction from config.toml MCPs
- `TestHealthCheck` -- VM state parsing from `vagrant status --machine-readable`
- `TestHealthCheckCaching` -- 60s TTL, no redundant subprocess calls
- `TestRestartVagrantSession` -- state-based recovery (running/suspended/aborted/not_created)
- `TestVMHealthToErrorMessage` -- contextual error messages per VM state
- `TestGetVagrantSettingsDefaults` -- default values for all VagrantSettings fields
- `TestGetVagrantSettingsOverrides` -- config.toml values override defaults

**UI tests:**
- Checkbox renders after Teammate mode
- Space toggles vagrant mode
- Checking vagrant forces skipPermissions on
- Error state shows "VM crashed — press R to restart" (not generic error)

**Manual integration tests (requires Vagrant):**
- Full lifecycle: create -> up -> claude runs -> stop -> suspend
- Vagrantfile generation in empty project
- Vagrantfile generation with custom provision_packages, npm_packages, port_forwards
- Vagrantfile generation with provision_script (custom script runs during provisioning)
- Custom Vagrantfile via vagrant.vagrantfile (user's file copied, no auto-generation)
- Skill preloading and Claude discovery
- HTTP MCP reachable from VM via 10.0.2.2 rewrite
- STDIO MCP (npx-based) works inside VM after provisioning
- Global MCP config propagated to VM's ~/.claude/.claude.json
- Port forwarding: dev server in VM accessible on host via forwarded port
- vagrant.env vars available inside VM session
- VM crash recovery: force-kill VirtualBox → press R → VM recreated, Claude resumes
- agent-deck crash recovery: kill agent-deck → restart → session auto-reconnects
- Suspended VM resume: suspend → press R → `vagrant resume` (not full `vagrant up`)
- Shared folder recovery: corrupt sync → press R → `vagrant reload` re-mounts

## User Documentation

This section describes Vagrant Mode from the user's perspective. During implementation, this content should be added to the project README and the config reference (`skills/agent-deck/references/config-reference.md`).

### Overview

Vagrant Mode runs Claude Code inside an isolated VirtualBox VM with full sudo access and `--dangerously-skip-permissions` enabled. The VM protects your host machine from accidental damage while giving Claude unrestricted access to install packages, modify system files, run Docker containers, and execute arbitrary commands.

**How it works:**
1. You check "Just do it (vagrant sudo)" when creating a session
2. agent-deck generates a `Vagrantfile` in your project directory (if one doesn't exist)
3. On session start, agent-deck runs `vagrant up` to boot the VM
4. Claude Code launches inside the VM via `vagrant ssh -c`
5. Your project files are bidirectionally synced between host and VM at `/vagrant`
6. On session stop, the VM is suspended. On session delete, the VM is destroyed.

**What Claude gets inside the VM:**
- Full `sudo` access (no password)
- Docker, Node.js, Git, and build tools pre-installed
- `--dangerously-skip-permissions` (all tool calls auto-approved)
- MCP tools from your `config.toml` (auto-provisioned)
- Your project files at `/vagrant` (synced with host)

### Prerequisites

- [Vagrant](https://www.vagrantup.com/downloads) installed and on PATH
- [VirtualBox](https://www.virtualbox.org/wiki/Downloads) 7.0+ (7.2+ recommended for Apple Silicon)
- Sufficient disk space for the VM base box (~2GB first download, cached afterward)

### Enabling Vagrant Mode

**In the TUI:**
1. Press `N` to create a new session (or `F` to fork)
2. Navigate to "Just do it (vagrant sudo)" checkbox
3. Press `Space` to enable — this automatically enables "Skip permissions" too
4. Press `Enter` to create the session

First boot downloads the base box and provisions the VM. Subsequent boots resume from a suspended state.

### Vagrant Configuration Reference

All Vagrant Mode settings live in the `[vagrant]` section of `~/.agent-deck/config.toml`. The config controls how the Vagrantfile is generated, similar to how `[mcps.*]` controls `.mcp.json` generation.

```toml
[vagrant]
# VM Resources
memory_mb = 4096                    # RAM in MB
cpus = 2                            # CPU cores
box = "bento/ubuntu-24.04"          # Vagrant box name

# Lifecycle
auto_suspend = true                 # Suspend VM on session stop (false = leave running)
auto_destroy = false                # Destroy VM on session delete (true = always clean slate)
health_check_interval = 60          # Seconds between VM health checks

# Network
host_gateway_ip = "10.0.2.2"       # VirtualBox NAT gateway (for MCP URL rewriting)
synced_folder_type = "virtualbox"   # Sync type: "virtualbox", "rsync", "nfs"

# Provisioning — what gets installed in the VM
provision_packages = [              # apt packages (replaces defaults when set)
  "docker.io", "nodejs", "npm", "git",
  "unzip", "curl", "build-essential"
]
npm_packages = []                   # Additional global npm packages (claude-code always included)
provision_script = ""               # Path to custom shell script, run after default provisioning

# Override — use your own Vagrantfile instead of auto-generating
vagrantfile = ""                    # Path to custom Vagrantfile (disables auto-generation entirely)

# Port forwarding
[[vagrant.port_forwards]]
guest = 3000
host = 3000
protocol = "tcp"                    # Optional, default: "tcp"

[[vagrant.port_forwards]]
guest = 5432
host = 15432

# Additional env vars available in VM sessions (alongside ANTHROPIC_API_KEY)
[vagrant.env]
DATABASE_URL = "postgres://localhost:5432/dev"
CUSTOM_VAR = "value"
```

#### VM Resources

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `memory_mb` | int | `4096` | RAM allocated to VM in megabytes. |
| `cpus` | int | `2` | CPU cores allocated to VM. |
| `box` | string | `"bento/ubuntu-24.04"` | Vagrant box to use. Any box from [Vagrant Cloud](https://app.vagrantup.com/boxes/search) works. |

#### Lifecycle

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `auto_suspend` | bool | `true` | Suspend VM when session stops. Set `false` to leave VM running between sessions. |
| `auto_destroy` | bool | `false` | Destroy VM when session is deleted. Set `true` for always-clean-slate workflow. |
| `health_check_interval` | int | `60` | Seconds between VM health checks. Lower values detect crashes faster but add overhead. |

#### Network

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `host_gateway_ip` | string | `"10.0.2.2"` | IP the VM uses to reach the host. Default is VirtualBox NAT gateway. Override for custom network configs. |
| `synced_folder_type` | string | `"virtualbox"` | Shared folder sync method: `"virtualbox"` (default, easiest), `"rsync"` (one-way, faster for large dirs), `"nfs"` (fast, requires NFS on host). |

#### Port Forwarding

Forward ports from guest VM to host. Useful when Claude starts dev servers, databases, or other services inside the VM.

```toml
[[vagrant.port_forwards]]
guest = 3000    # Port inside VM
host = 3000     # Port on host machine
protocol = "tcp" # Optional, default: "tcp"
```

Each `[[vagrant.port_forwards]]` entry becomes a `config.vm.network "forwarded_port"` line in the generated Vagrantfile.

#### Provisioning

Control what software is installed in the VM. This is analogous to how `[mcps.*]` defines what MCP tools are available — `[vagrant]` defines what system packages and tools are available inside the VM.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `provision_packages` | array | `["docker.io", "nodejs", "npm", "git", "unzip", "curl", "build-essential"]` | apt packages installed during provisioning. Setting this **replaces** the defaults (not appends). |
| `npm_packages` | array | `[]` | Additional npm packages installed globally. `@anthropic-ai/claude-code` is always included regardless. MCP packages from `[mcps.*]` are also auto-included. |
| `provision_script` | string | `""` | Path to a custom shell script run after default provisioning. Use for project-specific setup (databases, language runtimes, config files). |

**Example: Add Python and Redis to the VM:**
```toml
[vagrant]
provision_packages = [
  "docker.io", "nodejs", "npm", "git", "unzip", "curl", "build-essential",
  "python3", "python3-pip", "redis-server"
]
npm_packages = ["typescript", "tsx"]
provision_script = "scripts/vm-setup.sh"
```

**Example: `scripts/vm-setup.sh`:**
```bash
#!/bin/bash
# Custom provisioning — runs as root inside the VM
pip3 install poetry
systemctl enable redis-server
systemctl start redis-server
```

#### Custom Vagrantfile

For full control, point to your own Vagrantfile. When set, agent-deck skips auto-generation entirely and copies this file instead.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `vagrantfile` | string | `""` | Path to a custom Vagrantfile. Relative paths resolved from `~/.agent-deck/`. When set, all other `[vagrant]` provisioning settings are ignored — the custom file is used as-is. |

```toml
[vagrant]
vagrantfile = "~/vagrant-templates/heavy-dev.Vagrantfile"
```

**When using a custom Vagrantfile:**
- You are responsible for installing Claude Code (`npm install -g @anthropic-ai/claude-code`)
- MCP npm packages are NOT auto-provisioned (a warning is logged)
- The VM must have `/vagrant` synced folder for project file access
- VM resources (memory/CPU), port forwards, and provision settings from `config.toml` are ignored

#### Environment Variables

Additional env vars passed to Claude inside the VM, alongside `ANTHROPIC_API_KEY` and MCP-specific env vars.

```toml
[vagrant.env]
DATABASE_URL = "postgres://localhost:5432/dev"
NODE_ENV = "development"
```

These are appended to the `vagrant ssh -c` command as inline env vars.

### Customization Comparison: MCP vs Vagrant

The `[vagrant]` config follows the same pattern as `[mcps.*]`: you declare what you want in `config.toml`, and agent-deck generates the appropriate file.

| Aspect | MCP Tools | Vagrant Mode |
|--------|-----------|-------------|
| **Config source** | `[mcps.*]` in config.toml | `[vagrant]` in config.toml |
| **Generated file** | `.mcp.json` in project dir | `Vagrantfile` in project dir |
| **Skip generation** | N/A (always generated) | Set `vagrant.vagrantfile` to use your own |
| **Existing file** | Overwritten each session start | Respected — never overwritten |
| **Packages** | MCP npm packages auto-extracted | `provision_packages` + `npm_packages` |
| **Env vars** | `mcps.NAME.env` per MCP | `vagrant.env` for all sessions |
| **Custom setup** | `mcps.NAME.server` for auto-start | `vagrant.provision_script` for custom provisioning |

### Examples

#### Minimal (defaults)

```toml
# No [vagrant] section needed — defaults work out of the box
# Just check "Just do it" in the TUI
```

#### Web Development

```toml
[vagrant]
memory_mb = 8192
cpus = 4
npm_packages = ["typescript", "tsx", "pnpm"]

[[vagrant.port_forwards]]
guest = 3000
host = 3000

[[vagrant.port_forwards]]
guest = 5173
host = 5173
```

#### Data Science / ML

```toml
[vagrant]
memory_mb = 16384
cpus = 8
box = "bento/ubuntu-24.04"
provision_packages = [
  "docker.io", "nodejs", "npm", "git", "curl", "build-essential",
  "python3", "python3-pip", "python3-venv", "libpq-dev"
]
provision_script = "scripts/ml-setup.sh"

[[vagrant.port_forwards]]
guest = 8888
host = 8888
```

#### Full Custom Vagrantfile

```toml
[vagrant]
vagrantfile = "~/.agent-deck/vagrant-templates/my-devbox.Vagrantfile"
host_gateway_ip = "10.0.2.2"  # Still used for MCP URL rewriting
```

### Recovery & Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| "VM crashed — press R to restart" | VM ran out of memory or VirtualBox crashed | Press `R`. Increase `memory_mb` if recurring. |
| "VM powered off — press R to restart" | VM was manually shut down | Press `R` to boot. |
| "VM destroyed — press R to recreate" | Host rebooted or VM was manually destroyed | Press `R`. Claude conversation resumes via session ID. |
| First boot very slow | Downloading base box + provisioning | Normal. Subsequent boots use cached box (~30-60s). |
| MCP tool unavailable inside VM | npm package not installed in VM | Add to `npm_packages` or install in `provision_script`. |
| Can't reach host service from VM | Wrong gateway IP or service not listening on all interfaces | Check `host_gateway_ip`. Ensure host service binds to `0.0.0.0`, not just `127.0.0.1`. |
| File sync slow with node_modules | VirtualBox shared folders struggle with many small files | Use `synced_folder_type = "rsync"` or add `node_modules` to `.rsyncignore`. |
| Port conflict on host | Another process using the forwarded port | Change `host` port in `[[vagrant.port_forwards]]`. |

### Config Reference Update (Implementation Note)

During implementation, add a `[vagrant]` section to `skills/agent-deck/references/config-reference.md` following the same format as existing sections (`[claude]`, `[mcps.*]`, etc.). Include the full key table, examples, and a link to this design document for architectural context.

## Next Steps

- [ ] Create implementation plan (use `agentic-ai-plan`)
- [ ] Set up worktree for implementation
- [ ] Execute plan with agent team
