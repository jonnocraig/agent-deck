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
- [ ] VM boot progress shown in session list with phase name and elapsed timer
- [ ] Loading tips (Vagrant best practices + world facts) rotate in detail pane during boot
- [ ] Tips stop and disappear when VM reaches ready state
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
- Windows support -- path handling, SSH client differences, and disk check APIs need investigation

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
- 5 new files: `internal/vagrant/manager.go`, `internal/vagrant/provider.go`, `internal/vagrant/skill.go`, `internal/vagrant/mcp.go`, `internal/vagrant/tips.go`
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
- Bidirectional sync: .git/hooks and package.json scripts can be weaponized. Accepted risk for v1. Credential files guarded by PreToolUse hook (blocks Read) and rsync exclusion (when applicable).
- Credential exposure: API key forwarded via SSH SendEnv/AcceptEnv protocol, not visible in command string or `ps aux`
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
instance.Start() --> VagrantManager.Status()
                     |
                     +--→ VM running + other session? → Prompt: Share VM / Create Separate VM
                     |    - Share: RegisterSession(), skip vagrant up
                     |    - Separate: SetDotfilePath(sessionID), vagrant up new VM
                     +--→ VM not running → VagrantManager.EnsureRunning()
                     |
                     VagrantManager.EnsureSudoSkill()
                     VagrantManager.SyncClaudeConfig()         // propagate global/user MCPs to VM
                     WriteMCPJsonForVagrant(enabledNames)       // write .mcp.json (no URL rewriting needed)
                     CollectHTTPMCPPorts(enabledNames)           // ports for SSH reverse tunnels
                     VagrantManager.WrapCommand(cmd, mcpEnvVars, tunnelPorts)
                     --> tmux launches wrapped command

instance.Stop()  --> tmux kill --> VagrantManager.Suspend() (async goroutine, signals done channel)
instance.Delete()--> tmux kill --> VagrantManager.Destroy() (async goroutine, signals done channel)

instance.Start() --> if vmOpInFlight: wait on done channel (show "Waiting for VM...")
                     timeout after 60s → error

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
    projectPath   string
    settings      VagrantSettings
    dotfilePath   string     // set for separate-VM sessions (VAGRANT_DOTFILE_PATH)
    sessions      []string   // session IDs sharing this VM
    mu            sync.Mutex
}

func NewManager(projectPath string) *Manager
func (m *Manager) IsInstalled() bool          // exec.LookPath("vagrant")
func (m *Manager) EnsureRunning() error       // vagrant up if not running
func (m *Manager) Suspend() error             // vagrant suspend
func (m *Manager) Resume() error              // vagrant resume (suspended → running, faster than up)
func (m *Manager) Destroy() error             // vagrant destroy -f
func (m *Manager) ForceRestart() error        // vagrant destroy -f && vagrant up
func (m *Manager) Reload() error              // vagrant reload (restarts VM, re-mounts shared folders)
func (m *Manager) WrapCommand(cmd string, envVarNames []string, tunnelPorts []int) string
func (m *Manager) EnsureVagrantfile() error   // generate if missing, includes MCP npm packages
func (m *Manager) EnsureSudoSkill() error     // write skill to project
func (m *Manager) Status() (string, error)    // running/suspended/not_created/aborted/poweroff
func (m *Manager) HealthCheck() (VMHealth, error) // cached status check, 60s TTL
func (m *Manager) GetMCPPackages() []string   // extract npm packages from config.toml MCPs
func (m *Manager) SyncClaudeConfig() error    // copy host Claude configs to VM with URL rewrites
func (m *Manager) RegisterSession(sessionID string)    // track session sharing this VM
func (m *Manager) UnregisterSession(sessionID string)  // remove session from sharing list
func (m *Manager) SessionCount() int                   // number of sessions sharing this VM
func (m *Manager) IsLastSession(sessionID string) bool // true if this is the only session left
func (m *Manager) SetDotfilePath(sessionID string)     // set VAGRANT_DOTFILE_PATH for separate VM
func (m *Manager) PreflightCheck() error               // combined: Vagrant + VirtualBox + disk space
func (m *Manager) CheckVBoxInstalled() (string, error) // VBoxManage version check
func (m *Manager) CheckDiskSpace() (int64, error)      // available MB on project filesystem
func (m *Manager) IsBoxCached() bool                   // true if vagrant box already downloaded
func (m *Manager) Provision() error                    // vagrant provision on running VM
func (m *Manager) HasConfigDrift() bool                // compare stored hash to current config
func (m *Manager) WriteConfigHash() error              // persist current config hash
```

**`internal/vagrant/mcp.go`** -- MCP config generation and tunnel setup for Vagrant
```go
func WriteMCPJsonForVagrant(projectPath string, enabledNames []string) error
func CollectHTTPMCPPorts(enabledNames []string) []int           // extract ports from HTTP/SSE MCP URLs for reverse tunnels
func CollectEnvVarNames(enabledNames []string, vagrantEnv map[string]string) []string  // names for SendEnv flags
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

**`internal/session/instance.go`** -- Lifecycle hooks, crash recovery, and VM operation coordination
- `applyVagrantWrapper()` method called in Start()/StartWithMessage()
- `restartVagrantSession()` method: VM-aware restart with state detection and recovery
- Calls `WriteMCPJsonForVagrant()` instead of `WriteMCPJsonFromConfig()` when vagrant mode active
- Calls `SyncClaudeConfig()` to propagate global/user MCPs into VM
- Collects env var names via `CollectEnvVarNames()` (MCP env + vagrant.env + ANTHROPIC_API_KEY)
- Collects HTTP ports via `CollectHTTPMCPPorts()` for reverse tunnels
- Sets env var values in subprocess environment, passes names and tunnel ports to `WrapCommand()`
- Suspend hook in Stop(), Destroy hook in Delete() — both run in goroutines, signal `vmOpDone` channel
- Start() waits on `vmOpDone` if an operation is in-flight, with 60s timeout and TUI spinner
- `UpdateStatus()` extended: periodic VM health check (60s interval) for vagrant sessions
- New fields: `vagrantProvider` (interface), `lastVMHealthCheck`, `cleanShutdown`, `vmOpDone chan struct{}`, `vmOpInFlight atomic.Bool`
- Contextual error messages for VM crash states (aborted, poweroff, not_created)

**VM lifecycle coordination:**
```go
// Stop() — non-blocking suspend
func (i *Instance) stopVagrant() {
    i.vmOpInFlight.Store(true)
    go func() {
        defer func() {
            i.vmOpInFlight.Store(false)
            close(i.vmOpDone)  // signal completion
        }()
        _ = i.vagrantManager.Suspend()
    }()
}

// Start() — wait for in-flight ops before proceeding
func (i *Instance) waitForVagrantOp() error {
    if !i.vmOpInFlight.Load() {
        return nil
    }
    select {
    case <-i.vmOpDone:
        i.vmOpDone = make(chan struct{})  // reset for next cycle
        return nil
    case <-time.After(60 * time.Second):
        return fmt.Errorf("timed out waiting for VM operation to complete")
    }
}
```

**`internal/session/userconfig.go`** -- Config
- `VagrantSettings` struct (see [Vagrant Configuration Reference](#vagrant-configuration-reference) for full schema)
- `PortForward` struct for port forwarding rules
- `[vagrant]` TOML section
- `GetVagrantSettings()` with defaults (4096MB, 2 CPUs, auto_suspend=true, health_check_interval=30)

```go
type VagrantSettings struct {
    MemoryMB            int               `toml:"memory_mb"`             // Default: 4096
    CPUs                int               `toml:"cpus"`                  // Default: 2
    Box                 string            `toml:"box"`                   // Default: "bento/ubuntu-24.04"
    AutoSuspend         bool              `toml:"auto_suspend"`          // Default: true
    AutoDestroy         bool              `toml:"auto_destroy"`          // Default: false
    HostGatewayIP       string            `toml:"host_gateway_ip"`       // Default: "10.0.2.2"
    SyncedFolderType    string            `toml:"synced_folder_type"`    // Default: "virtualbox"
    ProvisionPackages   []string          `toml:"provision_packages"`    // Additional packages (appended to base set)
    ProvisionPkgExclude []string          `toml:"provision_packages_exclude"` // Packages to remove from base set
    NpmPackages         []string          `toml:"npm_packages"`          // Additional global npm packages
    ProvisionScript     string            `toml:"provision_script"`      // Path to custom shell script
    Vagrantfile         string            `toml:"vagrantfile"`           // Path to custom Vagrantfile (disables generation)
    HealthCheckInterval int               `toml:"health_check_interval"` // Default: 30 (seconds)
    PortForwards        []PortForward     `toml:"port_forwards"`         // Port forwarding rules
    Env                 map[string]string `toml:"env"`                   // Additional env vars for VM sessions
    ForwardProxyEnv     bool              `toml:"forward_proxy_env"`     // Default: true — auto-forward host proxy vars
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
  config.vm.hostname = "%s"  # "agentdeck-<project-name>" (sanitized)
  config.vm.synced_folder ".", "/vagrant", type: "%s"  # from vagrant.synced_folder_type

  # SSH agent forwarding for git push to private repos
  config.ssh.forward_agent = true

  # Port forwarding from vagrant.port_forwards (auto_correct prevents collisions)
  # config.vm.network "forwarded_port", guest: 3000, host: 3000, protocol: "tcp", auto_correct: true

  config.vm.provider "virtualbox" do |vb|
    vb.memory = "%d"  # from vagrant.memory_mb
    vb.cpus = %d      # from vagrant.cpus
    vb.gui = false
    vb.customize ["modifyvm", :id, "--audio", "none"]
    vb.customize ["modifyvm", :id, "--usb", "off"]
    vb.customize ["modifyvm", :id, "--nested-hw-virt", "on"]  # required for Docker-in-VM
  end

  config.vm.provision "shell", inline: <<-SHELL
    export DEBIAN_FRONTEND=noninteractive
    # Proxy env vars from host (if forward_proxy_env = true and host has proxy set)
    # export HTTP_PROXY="http://proxy.corp:8080"
    # export HTTPS_PROXY="http://proxy.corp:8080"
    # export NO_PROXY="localhost,127.0.0.1"
    apt-get update
    # Base packages (minus provision_packages_exclude) + provision_packages
    apt-get install -y docker.io nodejs npm git unzip curl build-essential
    # Claude Code (always installed)
    npm install -g @anthropic-ai/claude-code --no-audit
    # Additional npm packages from vagrant.npm_packages
    # npm install -g @custom/tool --no-audit
    # MCP packages from config.toml [mcps.*] (auto-extracted)
    # npm install -g @anthropics/exa-mcp @anthropics/slack-mcp --no-audit
    usermod -aG docker vagrant
    chown -R vagrant:vagrant /vagrant
    # Accept all forwarded env vars from host (for API keys, MCP env, vagrant.env)
    echo "AcceptEnv *" >> /etc/ssh/sshd_config
    systemctl restart sshd
  SHELL

  # Custom provisioning script from vagrant.provision_script (if set)
  # config.vm.provision "shell", path: "/path/to/custom-provision.sh"
end
```

**Generation logic in `EnsureVagrantfile()`:**
1. Check if `Vagrantfile` already exists in project dir → skip (log warning about manual MCP provisioning)
2. Check if `vagrant.vagrantfile` is set → copy that file instead of generating
3. Otherwise, generate from template using `[vagrant]` settings:
   - Interpolate `box`, `hostname`, `memory_mb`, `cpus`, `synced_folder_type`
   - Hostname derived from project directory name: `"agentdeck-" + sanitize(filepath.Base(projectPath))`
   - Sanitization: lowercase, replace non-alphanumeric with `-`, trim to 63 chars (RFC 1123)
   - Add `config.vm.network "forwarded_port"` lines for each `port_forwards` entry (with `auto_correct: true`)
   - Compute final package list: `(base_packages - provision_packages_exclude) + provision_packages`
   - Join into `apt-get install -y` command
   - Join `npm_packages` + auto-extracted MCP packages into `npm install -g` command
   - If `provision_script` is set, add `config.vm.provision "shell", path:` line

### VM Boot Progress Feedback

First boot (box download + provisioning) takes 5-10 minutes. Subsequent boots from suspended state take 30-60s. Without progress feedback, users think agent-deck froze.

**Approach:** Parse `vagrant up` stdout for known milestone lines and update session status in real-time. While waiting, display rotating tips (Vagrant best practices + quirky world facts) in the session detail pane.

#### Boot Phase Detection

`VagrantManager.EnsureRunning()` streams `vagrant up` stdout through a phase parser that updates session status via a callback:

```go
type BootPhase string

const (
    BootPhaseDownloading   BootPhase = "Downloading base box..."
    BootPhaseImporting     BootPhase = "Importing base box..."
    BootPhaseBooting       BootPhase = "Booting VM..."
    BootPhaseNetwork       BootPhase = "Configuring network..."
    BootPhaseMounting      BootPhase = "Mounting shared folders..."
    BootPhaseProvisioning  BootPhase = "Provisioning (installing packages)..."
    BootPhaseNpmInstall    BootPhase = "Installing Claude Code & MCP tools..."
    BootPhaseReady         BootPhase = "VM ready — starting Claude..."
)

// EnsureRunning accepts a callback for phase updates
func (m *Manager) EnsureRunning(onPhase func(BootPhase)) error
```

Phase detection parses vagrant's machine-readable output (`--machine-readable` flag) for action keywords:

| Vagrant Output Pattern | Phase |
|----------------------|-------|
| `ui,info,Downloading` | Downloading base box... |
| `ui,info,Importing base box` | Importing base box... |
| `ui,info,Booting VM` / `action,up,start` | Booting VM... |
| `action,up,configure_networks` | Configuring network... |
| `action,up,share_folders` / `action,up,sync_folders` | Mounting shared folders... |
| `action,up,provision` | Provisioning (installing packages)... |
| stdout contains `npm install -g @anthropic-ai/claude-code` | Installing Claude Code & MCP tools... |
| `action,up,complete` / SSH becomes available | VM ready — starting Claude... |

**Apple Silicon kernel extension detection:** On `darwin/arm64`, if `vagrant up` exits with an error, `EnsureRunning()` parses stderr for VirtualBox kernel extension failure patterns (`kernel driver not installed`, `vboxdrv`, `NS_ERROR_FAILURE`). When matched, the error is wrapped with a user-friendly message:

```go
func wrapVagrantUpError(err error, stderr string) error {
    if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
        if strings.Contains(stderr, "kernel driver") ||
           strings.Contains(stderr, "vboxdrv") ||
           strings.Contains(stderr, "NS_ERROR_FAILURE") {
            return fmt.Errorf(
                "VirtualBox requires approval in System Settings → Privacy & Security. " +
                "Approve the VirtualBox kernel extension, then press R to retry. " +
                "Original error: %w", err,
            )
        }
    }
    return err
}
```

#### Session List Display

The session list shows the current phase alongside an elapsed timer:

```
  my-project    ⟳ Provisioning (installing packages)... (2m 34s)
```

When resuming from suspended state, the simpler flow:
```
  my-project    ⟳ Resuming VM... (5s)
```

#### Loading Tips

While the VM boots, the session detail pane (right side) displays rotating tips — a mix of Vagrant/VirtualBox best practices and quirky world facts. Tips rotate every 8 seconds.

**New file: `internal/vagrant/tips.go`** — embedded tip content

```go
type Tip struct {
    Text     string
    Source   string // URL or attribution
    Category string // "vagrant" or "fact"
}

func GetRandomTip() Tip          // returns a random tip
func GetNextTip(index int) Tip   // sequential rotation for deterministic display
```

Tips are compiled into the binary (no external fetch). See [Loading Tips Content](#loading-tips-content) for the full list (50 Vagrant best practices + 50 world facts, shown in shuffled order).

Display format in the detail pane:
```
╭──────────────────────────────────────────────────────────╮
│  💡 Use `synced_folder_type = "nfs"` for 10x faster     │
│     file I/O with large projects.                        │
│                                                          │
│     — vagrantup.com/docs/synced-folders/nfs              │
╰──────────────────────────────────────────────────────────╯
```

Tips stop rotating and the box disappears once the VM reaches `BootPhaseReady`.

### Command Wrapping

Input: `claude --session-id UUID --dangerously-skip-permissions`

Output:
```
vagrant ssh -- \
  -R 30000:localhost:30000 -R 8080:localhost:8080 \
  -o SendEnv=ANTHROPIC_API_KEY -o SendEnv=EXA_API_KEY -o SendEnv=DATABASE_URL \
  -t 'cd /vagrant && claude --session-id UUID --dangerously-skip-permissions'
```

Uses `vagrant ssh --` with SSH flags instead of `vagrant ssh -c`:
- `-R` flags create reverse tunnels for each HTTP/SSE MCP port (from issue #1)
- `-o SendEnv=VAR` flags forward env vars via SSH protocol — values never appear in the command string, avoiding all quoting/escaping issues and command-length limits
- `-t` forces PTY allocation for interactive Claude sessions

**Env var transport:** All env vars (API keys, MCP env, `vagrant.env`) are set in agent-deck's subprocess environment and forwarded via SSH `SendEnv`/`AcceptEnv`. The VM's sshd is configured with `AcceptEnv *` during provisioning. This approach:
- Handles special chars (`$`, `'`, `"`, newlines) without escaping
- Has no command-length limits (SSH protocol, not shell args)
- Keeps secrets out of `ps aux` output on the host
- Works with JSON blobs, base64 tokens, and multi-line values

Command is run from the project directory where Vagrantfile lives.

### Multiple Sessions Per Project

When a user creates a new vagrant-mode session and a VM is already running for the same project directory, agent-deck detects this and prompts the user with two options:

#### Detection

During session creation (`Start()` / `StartWithMessage()`), before calling `EnsureRunning()`:

1. Check `vagrant status --machine-readable` in the project directory
2. If state is "running", check whether another agent-deck session owns this VM
3. If an existing session is using the VM → show prompt
4. If no existing session (orphaned VM) → reuse silently

Ownership is tracked via an in-memory map of `projectPath → []sessionID` on the `VagrantManager`, persisted as a lockfile at `<projectPath>/.vagrant/agent-deck.lock` (JSON with session IDs).

#### User Prompt

```
╭──────────────────────────────────────────────────────────────╮
│  A Vagrant VM is already running for this project.           │
│                                                              │
│  How would you like to proceed?                              │
│                                                              │
│  > Share existing VM   Multiple sessions use the same VM.    │
│                        Faster start, shared filesystem.      │
│                                                              │
│    Create separate VM  Dedicated VM for this session.        │
│                        Full isolation, longer startup.        │
╰──────────────────────────────────────────────────────────────╯
```

The prompt appears in the TUI session creation flow (after pressing Enter on the new session dialog, before `vagrant up`). Arrow keys to select, Enter to confirm.

#### Option B: Share Existing VM

Multiple sessions share the same running VM. Each session gets its own `vagrant ssh` connection with independent tmux panes.

**Behavior:**
- Skip `vagrant up` (VM already running)
- SSH reverse tunnels and SendEnv flags applied per-session (each `vagrant ssh --` invocation gets its own tunnel set)
- Suspend is skipped when *any* sharing session stops — only the last session to stop triggers suspend
- Destroy only runs when the session that *created* the VM is deleted, or when all sharing sessions are deleted
- Health check is shared — one check covers all sessions using the same VM

**Implementation:**
```go
// VagrantManager tracks sessions sharing a VM
type Manager struct {
    projectPath string
    settings    VagrantSettings
    sessions    []string  // session IDs sharing this VM
    mu          sync.Mutex
}

func (m *Manager) RegisterSession(sessionID string)
func (m *Manager) UnregisterSession(sessionID string)
func (m *Manager) SessionCount() int
func (m *Manager) IsLastSession(sessionID string) bool
```

**Stop/Delete logic with shared VM:**
```go
// In instance.Stop():
if mgr.SessionCount() > 1 {
    mgr.UnregisterSession(i.SessionID)
    // Don't suspend — other sessions still using the VM
} else {
    mgr.UnregisterSession(i.SessionID)
    i.stopVagrant()  // last session, suspend the VM
}

// In instance.Delete():
mgr.UnregisterSession(i.SessionID)
if mgr.SessionCount() == 0 {
    i.destroyVagrant()  // no sessions left, destroy VM
}
```

#### Option C: Create Separate VM

A unique VM is created for this session using a session-specific Vagrant environment directory.

**Behavior:**
- A unique `VAGRANT_DOTFILE_PATH` is set per session: `<projectPath>/.vagrant-<sessionID>/`
- Each session gets its own fully independent VM (separate VirtualBox instance)
- Vagrantfile is shared (same project dir), but VM state is isolated
- Suspend/destroy operate independently per session
- Port forwarding conflicts are the user's responsibility (VirtualBox will error on duplicate host ports)

**Implementation:**
```go
// When creating a unique VM, set VAGRANT_DOTFILE_PATH
func (m *Manager) SetDotfilePath(sessionID string) {
    m.dotfilePath = filepath.Join(m.projectPath, ".vagrant-"+sessionID)
}

// All vagrant commands include the env var:
func (m *Manager) vagrantCmd(args ...string) *exec.Cmd {
    cmd := exec.Command("vagrant", args...)
    cmd.Dir = m.projectPath
    if m.dotfilePath != "" {
        cmd.Env = append(os.Environ(), "VAGRANT_DOTFILE_PATH="+m.dotfilePath)
    }
    return cmd
}
```

**Cleanup:** When the session is deleted, the `.vagrant-<sessionID>/` directory is removed after `vagrant destroy -f`.

#### Defaults and Edge Cases

- If the existing VM is **suspended** (not running), no prompt — `vagrant resume` is called directly (existing behavior)
- If the existing VM is **not_created** or **aborted**, no prompt — `vagrant up` creates fresh (existing behavior)
- The prompt only appears when state is **"running"** and another session owns the VM
- **Fork** inherits the parent session's choice: if parent shares, fork shares; if parent has unique VM, fork gets a new unique VM
- `.vagrant-<sessionID>/` directories are added to `.gitignore` suggestions (not auto-modified)

### Stale Suspended VM Cleanup

Suspended VMs consume disk space (~4GB RAM saved to disk per VM). Users who stop sessions but never delete them will accumulate stale VMs over time. agent-deck addresses this with a warning system and a manual cleanup action.

#### Accumulation Warning

When the number of suspended vagrant VMs for the current user exceeds a threshold, agent-deck shows a warning toast in the TUI.

```go
const staleSuspendedVMWarningThreshold = 3

func (app *App) checkStaleSuspendedVMs() {
    // Run on startup and after each session stop
    // Uses `vagrant global-status --machine-readable` to count suspended VMs
    // whose project paths match known agent-deck session paths
    suspended := countSuspendedAgentDeckVMs()
    if suspended >= staleSuspendedVMWarningThreshold {
        app.ShowToast(fmt.Sprintf(
            "%d suspended VMs using disk space. Press Shift+D to clean up.", suspended,
        ))
    }
}
```

The check runs:
1. On agent-deck startup (non-blocking, background goroutine)
2. After each vagrant session stop (the suspend triggers the count check)

Only VMs whose project paths match known agent-deck sessions are counted — other Vagrant VMs are ignored.

#### Manual Cleanup

**TUI action: `Shift+D` — "Destroy suspended VMs"**

Shows a confirmation dialog listing all suspended agent-deck VMs with their project paths and suspend age:

```
╭──────────────────────────────────────────────────────────────╮
│  Destroy suspended Vagrant VMs?                              │
│                                                              │
│  ☑ my-project        suspended 3 days ago     (~4.0 GB)     │
│  ☑ api-service       suspended 12 days ago    (~4.0 GB)     │
│  ☐ ml-experiment     suspended 1 day ago      (~8.0 GB)     │
│                                                              │
│  Space to toggle · Enter to destroy selected · Esc to cancel │
╰──────────────────────────────────────────────────────────────╯
```

- All VMs pre-selected by default
- Space toggles individual VMs
- Enter runs `vagrant destroy -f` for each selected VM (sequentially)
- Progress shown inline as each VM is destroyed
- Associated agent-deck sessions updated to reflect VM destruction

**Implementation:**

```go
type SuspendedVM struct {
    ProjectPath string
    SessionIDs  []string  // agent-deck sessions using this VM
    SuspendedAt time.Time // from .vagrant directory mtime
    MemoryMB    int       // from VagrantSettings for estimated disk usage
}

func ListSuspendedAgentDeckVMs() ([]SuspendedVM, error) {
    // 1. Run `vagrant global-status --machine-readable`
    // 2. Filter for state == "saved" (suspended)
    // 3. Cross-reference project paths with agent-deck session DB
    // 4. Read .vagrant dir mtime for suspend timestamp
    // 5. Read VagrantSettings for memory estimate
}

func DestroySuspendedVMs(vms []SuspendedVM) error {
    for _, vm := range vms {
        mgr := vagrant.NewManager(vm.ProjectPath)
        _ = mgr.Destroy()
    }
    return nil
}
```

Suspend age is approximated from the `.vagrant` directory's modification time (updated when `vagrant suspend` writes the state file).

### Disk Space Preflight

Before calling `vagrant up`, agent-deck checks available disk space and blocks session creation if insufficient. This prevents cryptic VirtualBox failures mid-provisioning.

#### Space Requirements

| Scenario | Estimated Space Needed |
|----------|----------------------|
| First boot (box not cached) | ~2GB (box) + ~4GB (VM disk) + ~1GB (provisioning) = **~7GB** |
| First boot (box cached) | ~4GB (VM disk) + ~1GB (provisioning) = **~5GB** |
| Resume from suspended | ~0GB (state file already on disk) |

#### Thresholds

```go
const (
    diskSpaceMinimumMB  = 5120  // 5GB — block session creation below this
    diskSpaceWarningMB  = 10240 // 10GB — warn but allow
)
```

#### Check Logic

`PreflightCheck()` is a combined preflight that validates all prerequisites before `vagrant up`: Vagrant installation, VirtualBox installation (and version), and disk space.

```go
func (m *Manager) CheckDiskSpace() (availableMB int64, err error) {
    // Uses syscall.Statfs on the project directory's filesystem
    // Returns available space in MB
}

func (m *Manager) CheckVBoxInstalled() (version string, err error) {
    // exec.LookPath("VBoxManage")
    // If found, run `VBoxManage --version` → parse "7.2.4r163906" → "7.2.4"
    // Returns version string and nil, or "" and error if not found
}

func (m *Manager) PreflightCheck() error {
    // 1. Check Vagrant installed
    if !m.IsInstalled() {
        return fmt.Errorf("Vagrant not found. Install from https://www.vagrantup.com/downloads")
    }

    // 2. Check VirtualBox installed and version
    vboxVersion, err := m.CheckVBoxInstalled()
    if err != nil {
        return fmt.Errorf("VirtualBox not found. Install from https://www.virtualbox.org/wiki/Downloads")
    }
    major, minor := parseVersion(vboxVersion)
    if major < 7 {
        return fmt.Errorf("VirtualBox %s is too old. Version 7.0+ required (7.2+ recommended for Apple Silicon)", vboxVersion)
    }
    if major == 7 && minor < 2 && runtime.GOARCH == "arm64" {
        // Non-blocking warning for Apple Silicon users on VB 7.0/7.1
        // (logged, not returned as error — 7.0+ works but 7.2+ is better)
    }

    // 3. Check disk space
    availableMB, err := m.CheckDiskSpace()
    if err != nil {
        return nil // non-blocking — don't prevent session creation on check failure
    }

    boxCached := m.IsBoxCached() // vagrant box list --machine-readable
    requiredMB := diskSpaceMinimumMB
    if !boxCached {
        requiredMB += 2048 // additional 2GB for box download
    }

    if availableMB < int64(diskSpaceMinimumMB) {
        return fmt.Errorf(
            "insufficient disk space: %dMB available, %dMB required. Free up space or reduce vagrant.memory_mb",
            availableMB, requiredMB,
        )
    }
    return nil
}
```

#### Integration

Called in `instance.Start()` before `EnsureRunning()`:

```go
if err := mgr.PreflightCheck(); err != nil {
    return err // session creation blocked, error shown in TUI
}
```

If available space is between the minimum (5GB) and warning (10GB) thresholds, a non-blocking toast is shown:

```go
if availableMB < int64(diskSpaceWarningMB) {
    app.ShowToast(fmt.Sprintf("Low disk space: %dMB available. VM may fail if space runs out.", availableMB))
}
```

The warning threshold accounts for runtime growth (npm installs, Docker images, build artifacts inside VM).

#### Box Cache Detection

```go
func (m *Manager) IsBoxCached() bool {
    // Run `vagrant box list --machine-readable`
    // Check if settings.Box appears in the output
    // Returns true if box is already downloaded
}
```

This adjusts the space estimate — first-time users need ~2GB more than returning users.

### Error Handling

| Scenario | Handling |
|----------|----------|
| Vagrant not installed | Preflight blocks session. Error: "Vagrant not found. Install from vagrantup.com/downloads". |
| VirtualBox not installed | Preflight blocks session. Error: "VirtualBox not found. Install from virtualbox.org/wiki/Downloads". |
| VirtualBox < 7.0 | Preflight blocks session. Error: "VirtualBox X.Y is too old. Version 7.0+ required." |
| VirtualBox < 7.2 on Apple Silicon | Non-blocking warning logged. Session proceeds. |
| VirtualBox kernel ext not approved (Apple Silicon) | `vagrant up` fails. Detected via stderr parsing. Error: "VirtualBox requires approval in System Settings → Privacy & Security. Approve, then press R to retry." |
| Insufficient disk space (<5GB) | Preflight blocks session. Error: "insufficient disk space: XMB available, YMB required." |
| Low disk space (5-10GB) | Warning toast shown, session proceeds. |
| First boot (box download) | Output visible in tmux session. |
| `vagrant up` fails | Error returned, session not created. |
| Existing Vagrantfile | Used as-is, no overwrite. |
| VM already running (new session) | Prompt: share existing VM or create separate VM. |
| VM already running (same session restart) | Detected via status, skip `vagrant up`. |
| Suspend fails | Warning logged, non-blocking. vmOpDone channel still signaled. |
| Destroy fails | Warning logged, session deleted anyway. vmOpDone channel still signaled. |
| Start races with in-flight suspend | Start waits on vmOpDone channel, shows "Waiting for VM to finish suspending..." spinner. 60s timeout. |
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
- File watchers (inotify) do NOT work on `/vagrant` with VirtualBox shared folders. Use polling mode instead: set `CHOKIDAR_USEPOLLING=1` for webpack/vite, `WATCHPACK_POLLING=true` for Next.js, or `--poll` flag for `tsc --watch`. Alternatively, suggest the user switch to NFS (`synced_folder_type = "nfs"`) for native inotify support.
- NEVER read, cat, print, log, or transmit the contents of credential files (`.env`, `.npmrc`, `credentials.json`, `*.pem`, `*.key`, `id_rsa`, `id_ed25519`, `.netrc`, `.docker/config.json`). Use environment variables for secrets — they are already forwarded via SSH. If you need a secret value, ask the user to add it to `[vagrant.env]` in config.toml.

### VagrantProvider Interface

`instance.go` interacts with Vagrant through a `VagrantProvider` interface rather than the concrete `Manager` struct. This enables unit testing of lifecycle methods (`Start`, `Stop`, `Restart`, health check integration) in CI without Vagrant installed.

```go
// internal/vagrant/provider.go

type VagrantProvider interface {
    IsInstalled() bool
    PreflightCheck() error
    EnsureRunning(onPhase func(BootPhase)) error
    Suspend() error
    Resume() error
    Destroy() error
    ForceRestart() error
    Reload() error
    Provision() error
    Status() (string, error)
    HealthCheck() (VMHealth, error)
    WrapCommand(cmd string, envVarNames []string, tunnelPorts []int) string
    EnsureVagrantfile() error
    EnsureSudoSkill() error
    SyncClaudeConfig() error
    GetMCPPackages() []string
    HasConfigDrift() bool
    WriteConfigHash() error
    RegisterSession(sessionID string)
    UnregisterSession(sessionID string)
    SessionCount() int
    IsLastSession(sessionID string) bool
    SetDotfilePath(sessionID string)
    CheckDiskSpace() (int64, error)
    IsBoxCached() bool
}
```

**Production:** `Manager` implements `VagrantProvider` (all existing methods already match).

**Tests:** `MockVagrantProvider` with configurable return values:

```go
// internal/vagrant/mock_provider.go (test file)

type MockVagrantProvider struct {
    installed       bool
    status          string
    health          VMHealth
    healthErr       error
    ensureRunErr    error
    suspendErr      error
    sessionCount    int
    configDrift     bool
    wrappedCmd      string
    // ... configurable per-method returns
}

func (m *MockVagrantProvider) IsInstalled() bool        { return m.installed }
func (m *MockVagrantProvider) Status() (string, error)  { return m.status, nil }
func (m *MockVagrantProvider) HealthCheck() (VMHealth, error) { return m.health, m.healthErr }
// ... etc
```

**Instance field updated:**

```go
type Instance struct {
    // ... existing fields ...
    vagrantProvider  vagrant.VagrantProvider `json:"-"` // was: vagrantManager *vagrant.Manager
}
```

All `instance.go` references to `vagrantManager` change to `vagrantProvider`. In production, `NewInstance()` assigns `vagrant.NewManager(projectPath)`. In tests, the mock is injected directly.

### Provision Drift Detection

When `[vagrant]` config changes (new packages, different box, updated provision script), the running VM may be out of sync with the desired state. agent-deck detects this and re-provisions automatically.

#### Config Hash

`EnsureVagrantfile()` computes a SHA-256 hash of the Vagrantfile template inputs (packages, npm packages, box, provision script content, port forwards) and writes it to `<projectPath>/.vagrant/agent-deck-config.sha256`. On subsequent calls, the hash is compared:

```go
func (m *Manager) configHash() string {
    h := sha256.New()
    h.Write([]byte(m.settings.Box))
    h.Write([]byte(strings.Join(m.resolvedPackages(), ",")))
    h.Write([]byte(strings.Join(m.settings.NpmPackages, ",")))
    if m.settings.ProvisionScript != "" {
        content, _ := os.ReadFile(m.settings.ProvisionScript)
        h.Write(content)
    }
    for _, pf := range m.settings.PortForwards {
        h.Write([]byte(fmt.Sprintf("%d:%d:%s", pf.Guest, pf.Host, pf.Protocol)))
    }
    return hex.EncodeToString(h.Sum(nil))
}

func (m *Manager) HasConfigDrift() bool {
    hashFile := filepath.Join(m.projectPath, ".vagrant", "agent-deck-config.sha256")
    stored, err := os.ReadFile(hashFile)
    if err != nil {
        return false // no hash file = first run, no drift
    }
    return strings.TrimSpace(string(stored)) != m.configHash()
}

func (m *Manager) WriteConfigHash() error {
    hashFile := filepath.Join(m.projectPath, ".vagrant", "agent-deck-config.sha256")
    return os.WriteFile(hashFile, []byte(m.configHash()), 0644)
}
```

#### Re-Provision Flow

In `instance.Start()`, after `EnsureRunning()` confirms the VM is running:

```go
if mgr.HasConfigDrift() {
    // Regenerate Vagrantfile with updated config
    mgr.EnsureVagrantfile()  // overwrites with new template
    // Re-provision without destroying VM
    if err := mgr.Provision(); err != nil {
        app.ShowToast("Re-provisioning failed. Delete and recreate session for clean state.")
    } else {
        mgr.WriteConfigHash()
        app.ShowToast("VM re-provisioned with updated config.")
    }
}
```

```go
func (m *Manager) Provision() error {
    // vagrant provision — runs provisioning scripts on running VM
    cmd := m.vagrantCmd("provision")
    return cmd.Run()
}
```

**Behavior:**
- Only triggers when config hash differs from stored hash
- Runs `vagrant provision` (not `vagrant destroy + up`) — preserves VM state and data
- Vagrantfile is regenerated before provisioning so new packages/ports are included
- Box changes require manual destroy+recreate (provisioning can't swap the base box). A warning is logged: "Box changed from X to Y. Delete and recreate session to apply."
- Hash written after successful provision to prevent re-triggering

#### Updated Manager API

```go
func (m *Manager) Provision() error        // vagrant provision on running VM
func (m *Manager) HasConfigDrift() bool     // compare stored hash to current config
func (m *Manager) WriteConfigHash() error   // persist current config hash
```

### Credential Guard Hook

A Claude Code `PreToolUse` hook is auto-injected into the session's `.claude/settings.local.json` when vagrant mode is active. The hook blocks Claude from reading known credential files, providing a hard guardrail beyond the skill's soft guidance.

**Hook definition (written by `EnsureSudoSkill()`):**

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Read|View|Cat",
        "hooks": [
          {
            "type": "command",
            "command": "bash -c 'FILE=\"$CLAUDE_TOOL_ARG_file_path\"; PATTERNS=\".env|.npmrc|credentials.json|.pem|.key|id_rsa|id_ed25519|.netrc|.docker/config.json|.aws/credentials|.gcloud/credentials\"; echo \"$FILE\" | grep -qiE \"($PATTERNS)\" && echo \"BLOCKED: Reading credential files is not allowed in vagrant mode. Use [vagrant.env] for secrets.\" && exit 1 || exit 0'"
          }
        ]
      }
    ]
  }
}
```

**Behavior:**
- Matches `Read`, `View`, and `Cat` tool calls
- Checks `file_path` argument against known credential file patterns
- Blocks with a clear message if matched, allowing the tool call otherwise
- Pattern list covers: `.env` (and `.env.*`), `.npmrc`, `credentials.json`, PEM/key files, SSH keys, `.netrc`, Docker config, AWS credentials, GCloud credentials
- The hook is written to the project's `.claude/settings.local.json` (merged, not overwritten)
- Only active for vagrant-mode sessions — non-vagrant sessions are unaffected

**Additional sync folder exclusion:** When `synced_folder_type = "rsync"`, the generated Vagrantfile includes an `rsync__exclude` list to prevent credential files from being synced into the VM entirely:

```ruby
config.vm.synced_folder ".", "/vagrant", type: "rsync",
  rsync__exclude: [
    ".env", ".env.*", ".npmrc", "credentials.json",
    "*.pem", "*.key", "id_rsa", "id_ed25519",
    ".netrc", ".docker/", ".aws/", ".gcloud/"
  ]
```

This exclusion only applies to rsync mode — VirtualBox and NFS shared folders don't support file-level exclusion. For those modes, the hook is the primary guardrail.

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

#### Solution: VM-Aware MCP Config Generation + SSH Reverse Tunnels

**New function: `WriteMCPJsonForVagrant()`** -- variant of `WriteMCPJsonFromConfig()` called when `UseVagrantMode == true`.

```go
func WriteMCPJsonForVagrant(projectPath string, enabledNames []string) error
```

Differences from the normal write path:

1. **STDIO MCPs**: Written as plain STDIO config (no pool socket references). Pool is always bypassed for vagrant sessions. The STDIO commands work because MCP npm packages are installed in the VM during provisioning.

2. **HTTP/SSE MCPs**: URLs left as `localhost` in `.mcp.json` — no rewriting needed. Connectivity is provided by SSH reverse tunnels (see below). This works regardless of whether the host service binds to `127.0.0.1` or `0.0.0.0`.

3. **Pool sockets**: Never referenced. Vagrant sessions always use STDIO fallback for MCPs that would normally use pool sockets.

4. **Env vars**: MCP-specific env vars from `MCPDef.Env` are passed through the `vagrant ssh` command as inline env vars alongside `ANTHROPIC_API_KEY`.

#### SSH Reverse Tunnels for HTTP/SSE MCPs

The design uses SSH reverse port forwarding (`-R`) to make host-side HTTP/SSE MCP servers reachable from inside the VM at `localhost`. This is strictly better than the NAT gateway IP rewrite approach because:

- Works when host services bind to `127.0.0.1` only (most MCP servers do)
- No config needed from the user (`host_gateway_ip` becomes unnecessary for HTTP MCPs)
- Standard SSH behavior, well-tested and reliable

**How it works:**

1. `CollectHTTPMCPPorts()` scans enabled MCPs for HTTP/SSE URLs referencing `localhost` or `127.0.0.1`
2. For each unique port found, a `-R PORT:localhost:PORT` flag is added to the `vagrant ssh` command
3. Inside the VM, `localhost:PORT` reaches the host service via the SSH tunnel

```go
func CollectHTTPMCPPorts(enabledNames []string) []int {
    // Scan enabled MCP definitions for HTTP/SSE URLs
    // Extract unique port numbers from localhost/127.0.0.1 URLs
    // Returns deduplicated sorted list of ports
}
```

**Example:** If the host has MCPs at `http://localhost:30000/mcp/` and `http://localhost:8080/sse`, the command becomes:

```
vagrant ssh -- -R 30000:localhost:30000 -R 8080:localhost:8080 -t 'cd /vagrant && ANTHROPIC_API_KEY=... claude ...'
```

**Fallback:** For non-localhost HTTP MCPs (e.g., `http://192.168.1.50:9000/mcp/`), URLs are written as-is — the VM can reach LAN addresses directly via NAT. The `host_gateway_ip` config is retained as an escape hatch for edge cases where reverse tunnels are insufficient (e.g., UDP-based transports), but is no longer the primary mechanism.

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

`VagrantManager.SyncClaudeConfig()` copies host-side Claude configs into the VM. No URL rewriting needed — SSH reverse tunnels handle localhost connectivity.

```go
func (m *Manager) SyncClaudeConfig() error
```

Steps:
1. Read host's `$CLAUDE_CONFIG_DIR/.claude.json` (global MCPs)
2. Read host's `~/.claude.json` (user MCPs)
3. Extract HTTP/SSE ports from global/user MCPs → add to reverse tunnel list
4. Write configs to VM via `vagrant ssh -c "mkdir -p ~/.claude && cat > ~/.claude/.claude.json << 'HEREDOC' ... HEREDOC"`
5. Run once during `EnsureRunning()`, after VM is up but before Claude launches

#### Command Wrapping (Updated)

The wrapped command uses `vagrant ssh --` with SSH flags for reverse tunnels, env forwarding, and `-t` for PTY allocation:

```
vagrant ssh -- \
  -R 30000:localhost:30000 -R 8080:localhost:8080 \
  -o SendEnv=ANTHROPIC_API_KEY -o SendEnv=EXA_API_KEY \
  -t 'cd /vagrant && claude --session-id UUID --dangerously-skip-permissions'
```

All env vars from enabled MCP definitions (`MCPDef.Env`) and user-defined `vagrant.env` are forwarded via SSH `SendEnv`/`AcceptEnv` (not inline). SSH reverse tunnel ports are auto-collected from enabled HTTP/SSE MCP URLs (both local `.mcp.json` and global/user configs).

#### Updated VagrantManager API

```go
func (m *Manager) WriteMCPJsonForVagrant(projectPath string, enabledNames []string) error
func (m *Manager) SyncClaudeConfig() error
func (m *Manager) GetMCPPackages() []string       // extract npm packages from config.toml MCPs
func CollectHTTPMCPPorts(enabledNames []string) []int  // extract ports from HTTP/SSE MCP URLs
func (m *Manager) WrapCommand(cmd string, envVarNames []string, tunnelPorts []int) string
```

#### Limitations (v1)

- STDIO MCPs using non-npm commands (python, custom binaries) must be manually installed in the VM by the user
- Env var forwarding requires VM sshd configured with `AcceptEnv *` (done during provisioning; custom Vagrantfile users must add this manually)
- Pool socket MCPs fall back to direct STDIO (higher memory usage in VM, one MCP process per session)
- If user provides their own Vagrantfile, MCP packages are not auto-provisioned
- SSH reverse tunnels only support TCP (HTTP/SSE). UDP-based transports would need `host_gateway_ip` fallback
- Reverse tunnels add one `-R` flag per unique MCP port; unlikely to hit SSH limits but many MCPs (20+) may add latency to connection setup

#### Error Handling (MCP-specific)

| Scenario | Handling |
|----------|----------|
| STDIO MCP command not found in VM | Claude Code reports tool unavailable. Non-blocking. |
| HTTP MCP unreachable from VM | Claude Code reports connection error. Non-blocking. Check host service is running. |
| SSH reverse tunnel port conflict inside VM | Tunnel setup logs warning, MCP may be unreachable. Non-blocking. |
| Host service not running on tunneled port | Claude Code reports connection refused. Non-blocking. |
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
| VM hangs (unresponsive) | Nothing (but frozen) | tmux pane (frozen), agent-deck | Liveness probe (`vagrant ssh -c 'echo pong'`) times out after 5s | StatusError: "VM unresponsive". Press R: `vagrant destroy -f && vagrant up` |
| Shared folder sync breaks | File I/O inside VM | VM, Claude (but erroring), agent-deck | Claude reports file errors in tmux output | Press R: `vagrant reload` (restarts VM, re-mounts folders) |

#### VM Health Check

Add a periodic VM health check to the existing `UpdateStatus()` polling loop. Only runs for vagrant-mode sessions.

```go
func (m *Manager) HealthCheck() (VMHealth, error)
```

**VMHealth struct:**
```go
type VMHealth struct {
    State       string // "running", "suspended", "not_created", "aborted", "poweroff"
    Healthy     bool   // true if running AND responsive to liveness probe
    Responsive  bool   // true if SSH liveness probe succeeded (only checked when State == "running")
    Message     string // human-readable status
}
```

**Two-phase health check:**

1. **Phase 1 — Hypervisor state:** `vagrant status --machine-readable` (~100-200ms). Detects crashed/stopped/destroyed VMs.
2. **Phase 2 — Guest liveness probe:** `vagrant ssh -c 'echo pong'` with 5s timeout. Only runs when Phase 1 reports "running". Detects in-VM hangs (systemd stuck, disk full, kernel deadlock) that the hypervisor can't see.

```go
func (m *Manager) HealthCheck() (VMHealth, error) {
    // Phase 1: hypervisor state
    state, err := m.Status()
    if err != nil {
        return VMHealth{}, err
    }
    if state != "running" {
        return VMHealth{State: state, Healthy: false, Responsive: false, Message: vmStateMessage(state)}, nil
    }

    // Phase 2: guest liveness probe (only when "running")
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    cmd := exec.CommandContext(ctx, "vagrant", "ssh", "-c", "echo pong")
    cmd.Dir = m.projectPath
    if err := cmd.Run(); err != nil {
        return VMHealth{State: "running", Healthy: false, Responsive: false, Message: "VM unresponsive — guest OS may be hung"}, nil
    }

    return VMHealth{State: "running", Healthy: true, Responsive: true, Message: "VM running"}, nil
}
```

**Integration with `UpdateStatus()`** (`instance.go`):
- For vagrant sessions, piggyback on the existing 500ms status polling
- VM health check runs at a lower frequency: every 30 seconds (configurable via `health_check_interval`)
- Phase 1 (vagrant status) cached in-memory with 30s TTL
- Phase 2 (liveness probe) only runs when Phase 1 says "running" — adds ~200ms when healthy, up to 5s when hung
- If VM state is "aborted"/"poweroff" or liveness probe fails → set `StatusError`

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

    switch {
    case health.State == "running" && health.Responsive:
        // VM is fine, just respawn Claude
    case health.State == "running" && !health.Responsive:
        // VM reports running but guest is hung — force restart
        _ = mgr.Destroy()
        if err := mgr.EnsureRunning(nil); err != nil {
            return fmt.Errorf("failed to restart hung VM: %w", err)
        }
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
    vagrantProvider     vagrant.VagrantProvider `json:"-"` // interface for testability
    lastVMHealthCheck   time.Time               `json:"-"`
    cleanShutdown       bool                    `json:"-"` // set to true on graceful Stop/Suspend
}
```

On startup, if `UseVagrantMode` is true and `cleanShutdown` is false (default for loaded instances), the first `UpdateStatus()` call triggers an immediate VM health check instead of waiting 30s. This catches cases where the host rebooted or agent-deck was killed.

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
| running (but unresponsive) | `VM unresponsive — press R to force restart` |
| running (but Claude exited) | `Claude exited — press R to resume` |
| unknown | `VM status unknown — press R to retry` |

These messages replace the generic error indicator for vagrant sessions and are stored in `Instance.lastError`.

#### Updated VagrantManager API (with recovery)

```go
func (m *Manager) HealthCheck() (VMHealth, error)    // two-phase: vagrant status + SSH liveness probe, cached 30s
func (m *Manager) Resume() error                     // vagrant resume (suspended → running)
func (m *Manager) ForceRestart() error               // vagrant destroy -f && vagrant up
func (m *Manager) Reload() error                     // vagrant reload (restarts VM, re-mounts sync)
```

#### Updated Acceptance Criteria (recovery)

- [ ] VM crash detected via health check within 30s, session shows "VM crashed" error
- [ ] VM hang (guest unresponsive) detected via SSH liveness probe, shows "VM unresponsive" error
- [ ] Press R on crashed vagrant session: destroys old VM, creates new, resumes Claude
- [ ] agent-deck crash + restart: automatically reconnects to running vagrant session
- [ ] Host reboot + restart: press R recreates VM and resumes Claude conversation
- [ ] Suspended VM correctly resumed (not full `vagrant up`) on restart
- [ ] Vagrant reload triggered when shared folder sync breaks
- [ ] New session detects existing running VM for same project and shows share/separate prompt
- [ ] "Share VM" option reuses running VM, skips `vagrant up`, registers session
- [ ] "Separate VM" option creates isolated VM via `VAGRANT_DOTFILE_PATH`
- [ ] Shared VM only suspends when last sharing session stops
- [ ] Shared VM only destroys when all sharing sessions are deleted
- [ ] Fork inherits parent's VM sharing choice
- [ ] Warning toast shown when 3+ suspended VMs detected on startup or after session stop
- [ ] `Shift+D` shows confirmation dialog listing suspended VMs with age and estimated size
- [ ] Selected VMs destroyed on Enter, sessions updated accordingly
- [ ] Session creation blocked when disk space < 5GB with clear error message
- [ ] Warning toast shown when disk space between 5-10GB, session proceeds
- [ ] Space estimate accounts for whether base box is already cached
- [ ] Session creation blocked with clear error when VirtualBox not installed
- [ ] Session creation blocked when VirtualBox version < 7.0
- [ ] Warning logged for Apple Silicon users on VirtualBox < 7.2
- [ ] Generated Vagrantfile includes `auto_correct: true` on all forwarded port lines
- [ ] `provision_packages` appends to base set, not replaces
- [ ] `provision_packages_exclude` removes specified packages from base set
- [ ] Base packages always include `nodejs`, `npm`, `git` unless explicitly excluded
- [ ] Polling env vars (`CHOKIDAR_USEPOLLING`, `WATCHPACK_POLLING`, `TSC_WATCHFILE`) auto-injected for VirtualBox sync
- [ ] Polling env vars NOT injected when `synced_folder_type` is `nfs` or `rsync`
- [ ] Sudo skill includes inotify/polling guidance for Claude
- [ ] Apple Silicon kernel extension failure detected and shown with System Settings guidance
- [ ] Generated Vagrantfile includes `config.vm.hostname` derived from project name
- [ ] Host proxy env vars auto-forwarded to VM provisioning and SSH sessions when set
- [ ] `forward_proxy_env = false` disables proxy auto-forwarding
- [ ] PreToolUse credential guard hook auto-injected for vagrant sessions
- [ ] Hook blocks Read/View/Cat of `.env`, `.key`, `credentials.json`, SSH keys, etc.
- [ ] Sudo skill warns Claude not to read or transmit credential file contents
- [ ] Rsync mode excludes credential files from sync via `rsync__exclude`
- [ ] Config drift detected when `[vagrant]` settings change between sessions
- [ ] `vagrant provision` runs automatically on config drift, toast shown
- [ ] Box change logs warning advising manual destroy+recreate
- [ ] `instance.go` uses `VagrantProvider` interface, not concrete `Manager`
- [ ] All lifecycle unit tests runnable in CI via `MockVagrantProvider`

### Testing Strategy

**Unit tests:**
- `TestWrapCommand` -- command wrapping with quote escaping
- `TestWrapCommandWithSendEnv` -- -o SendEnv flags for MCP env vars
- `TestWrapCommandWithVagrantEnvVars` -- vagrant.env var names included in SendEnv flags
- `TestEnsureVagrantfile` -- generation when missing, skip when exists
- `TestEnsureVagrantfileWithMCPs` -- npm MCP packages in provisioning script
- `TestEnsureVagrantfileWithCustomPackages` -- provision_packages and npm_packages in template
- `TestEnsureVagrantfileWithPortForwards` -- port forwarding lines in generated Vagrantfile
- `TestEnsureVagrantfileWithProvisionScript` -- custom script provisioner appended
- `TestEnsureVagrantfileCustomTemplate` -- vagrant.vagrantfile copies user's file
- `TestEnsureVagrantfileExistingRespected` -- existing Vagrantfile never overwritten
- `TestGetVagrantSudoSkill` -- skill content validation
- `TestWriteMCPJsonForVagrant` -- STDIO fallback, no pool references, URLs unchanged
- `TestCollectHTTPMCPPorts` -- port extraction from HTTP/SSE MCP URLs (localhost + 127.0.0.1)
- `TestCollectHTTPMCPPortsDedup` -- duplicate ports across MCPs deduplicated
- `TestWrapCommandWithTunnels` -- -R flags added for each tunnel port
- `TestGetMCPPackages` -- npm package extraction from config.toml MCPs
- `TestHealthCheckPhase1` -- VM state parsing from `vagrant status --machine-readable`
- `TestHealthCheckPhase2LivenessProbe` -- SSH echo probe detects hung guest
- `TestHealthCheckPhase2Timeout` -- 5s timeout on unresponsive guest
- `TestHealthCheckCaching` -- 30s TTL, no redundant subprocess calls
- `TestRestartVagrantSession` -- state-based recovery (running/suspended/aborted/not_created)
- `TestStartWaitsForInFlightSuspend` -- Start blocks on vmOpDone, proceeds after suspend completes
- `TestStartTimeoutOnHungSuspend` -- Start returns error after 60s timeout
- `TestVMHealthToErrorMessage` -- contextual error messages per VM state
- `TestBootPhaseParser` -- vagrant machine-readable output mapped to correct BootPhase
- `TestBootPhaseDownloadDetection` -- first-boot box download detected from output
- `TestGetRandomTip` -- returns valid tip with text and source
- `TestGetNextTipRotation` -- sequential tips don't repeat until full cycle
- `TestGetVagrantSettingsDefaults` -- default values for all VagrantSettings fields
- `TestGetVagrantSettingsOverrides` -- config.toml values override defaults
- `TestDetectRunningVMWithExistingSession` -- detects running VM owned by another session
- `TestDetectRunningVMOrphaned` -- orphaned VM (no session) reused silently without prompt
- `TestRegisterUnregisterSession` -- session tracking add/remove and count
- `TestSharedVMSuspendOnlyOnLastSession` -- suspend skipped while other sessions remain
- `TestSharedVMDestroyOnlyWhenAllDeleted` -- destroy runs only when session count reaches zero
- `TestSeparateVMDotfilePath` -- VAGRANT_DOTFILE_PATH set correctly per session ID
- `TestSeparateVMCleanup` -- .vagrant-<sessionID>/ removed after destroy
- `TestForkInheritsVMSharingChoice` -- forked session inherits parent's share/separate decision
- `TestStaleSuspendedVMWarning` -- warning shown when suspended count >= 3
- `TestStaleSuspendedVMWarningBelowThreshold` -- no warning when count < 3
- `TestListSuspendedAgentDeckVMs` -- only agent-deck VMs counted, other Vagrant VMs ignored
- `TestDestroySuspendedVMs` -- destroy called for each selected VM
- `TestPreflightCheckBlocksBelowMinimum` -- returns error when available < 5GB
- `TestPreflightCheckWarnsLowSpace` -- no error but warning range when 5-10GB
- `TestPreflightCheckPassesAboveThreshold` -- no error when > 10GB
- `TestPreflightCheckAccountsForBoxCache` -- higher requirement when box not cached
- `TestIsBoxCached` -- correctly parses `vagrant box list` output
- `TestPreflightCheckVBoxMissing` -- returns error with install URL when VBoxManage not on PATH
- `TestPreflightCheckVBoxTooOld` -- returns error when version < 7.0
- `TestPreflightCheckVBoxAppleSiliconWarning` -- warning logged for arm64 + VBox < 7.2
- `TestCheckVBoxInstalledParsesVersion` -- correctly parses "7.2.4r163906" → "7.2.4"
- `TestEnsureVagrantfilePortForwardsAutoCorrect` -- all forwarded_port lines include auto_correct: true
- `TestProvisionPackagesAppendToBase` -- user packages added after base set
- `TestProvisionPackagesExclude` -- excluded packages removed from base set
- `TestProvisionPackagesExcludeAndAppend` -- exclude + append work together correctly
- `TestProvisionPackagesEmptyUsesBaseOnly` -- empty config uses full base set
- `TestPollingEnvVarsInjectedForVirtualBox` -- CHOKIDAR_USEPOLLING, WATCHPACK_POLLING, TSC_WATCHFILE set when synced_folder_type = "virtualbox"
- `TestPollingEnvVarsNotInjectedForNFS` -- polling vars absent when synced_folder_type = "nfs"
- `TestPollingEnvVarsUserOverride` -- user-defined vagrant.env values take precedence over auto-injected
- `TestSudoSkillMentionsInotify` -- skill content includes inotify/polling guidance
- `TestWrapVagrantUpErrorAppleSiliconKext` -- stderr with "kernel driver" on darwin/arm64 → friendly message
- `TestWrapVagrantUpErrorNonAppleSilicon` -- same stderr on linux/amd64 → original error unchanged
- `TestWrapVagrantUpErrorUnrelatedFailure` -- non-kext errors passed through unchanged
- `TestHostnameFromProjectName` -- "my-project" → "agentdeck-my-project"
- `TestHostnameSanitization` -- "My Project!@#" → "agentdeck-my-project---"
- `TestHostnameTruncation` -- long names truncated to 63 chars
- `TestProxyEnvVarsForwardedWhenSet` -- HTTP_PROXY etc. included in provisioning and SendEnv when host has them
- `TestProxyEnvVarsNotForwardedWhenUnset` -- no proxy lines when host has no proxy vars
- `TestProxyEnvVarsDisabledByConfig` -- forward_proxy_env = false skips proxy forwarding
- `TestProxyEnvVarsUserOverride` -- vagrant.env proxy values take precedence over host
- `TestCredentialGuardHookInjected` -- PreToolUse hook written to .claude/settings.local.json for vagrant sessions
- `TestCredentialGuardHookBlocksEnvFile` -- hook blocks Read of .env file
- `TestCredentialGuardHookBlocksSSHKey` -- hook blocks Read of id_rsa, id_ed25519
- `TestCredentialGuardHookAllowsNormalFiles` -- hook allows Read of .ts, .go, .json (non-credential)
- `TestCredentialGuardHookNotInjectedForNonVagrant` -- non-vagrant sessions don't get the hook
- `TestRsyncExcludesCredentialFiles` -- rsync__exclude list includes credential patterns
- `TestSudoSkillMentionsCredentialWarning` -- skill content includes credential file warning
- `TestConfigHashDeterministic` -- same inputs produce same hash
- `TestConfigHashChangesOnPackageAdd` -- adding a package changes the hash
- `TestConfigHashChangesOnProvisionScript` -- modifying provision script changes hash
- `TestHasConfigDriftDetectsChange` -- returns true when stored hash differs from current
- `TestHasConfigDriftFalseOnFirstRun` -- no hash file = no drift (first run)
- `TestProvisionCalledOnDrift` -- vagrant provision triggered when drift detected
- `TestBoxChangeLogsWarning` -- box name change logged as warning, not auto-provisioned
- `TestManagerImplementsVagrantProvider` -- compile-time interface satisfaction check
- `TestMockProviderStartLifecycle` -- Start() with mock: EnsureRunning called, command wrapped
- `TestMockProviderStopSuspends` -- Stop() with mock: Suspend called on last session
- `TestMockProviderHealthCheckIntegration` -- UpdateStatus() with mock: health check triggers error state
- `TestMockProviderRestartRecovery` -- Restart() with mock: state-based recovery paths all work

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
- HTTP MCP reachable from VM via SSH reverse tunnel (even when host binds 127.0.0.1)
- STDIO MCP (npx-based) works inside VM after provisioning
- Global MCP config propagated to VM's ~/.claude/.claude.json
- Port forwarding: dev server in VM accessible on host via forwarded port
- vagrant.env vars available inside VM session
- VM crash recovery: force-kill VirtualBox → press R → VM recreated, Claude resumes
- agent-deck crash recovery: kill agent-deck → restart → session auto-reconnects
- Suspended VM resume: suspend → press R → `vagrant resume` (not full `vagrant up`)
- Shared folder recovery: corrupt sync → press R → `vagrant reload` re-mounts
- Multi-session share: create session A → create session B (share) → both run in same VM
- Multi-session share suspend: stop session A → VM stays running → stop session B → VM suspends
- Multi-session separate: create session A → create session B (separate) → two independent VMs
- Fork inherits sharing: fork from shared session → fork reuses same VM without prompt
- Stale VM cleanup: create 3 sessions → stop all → Shift+D → select all → VMs destroyed

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

- **macOS or Linux** — Windows is not currently supported (see note below)
- [Vagrant](https://www.vagrantup.com/downloads) installed and on PATH
- [VirtualBox](https://www.virtualbox.org/wiki/Downloads) 7.0+ (7.2+ recommended for Apple Silicon)
- Sufficient disk space for the VM base box (~2GB first download, cached afterward)

> **Windows:** Vagrant Mode is not supported on Windows for v1. Known blockers include: path separator handling in Vagrantfile generation, SSH client differences (`SendEnv`/`AcceptEnv` behavior with Windows OpenSSH vs PuTTY), `syscall.Statfs` unavailability for disk space checks, and untested VirtualBox shared folder behavior with Windows paths. Windows support may be added in a future version.

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
health_check_interval = 30          # Seconds between VM health checks (includes liveness probe)

# Network
host_gateway_ip = "10.0.2.2"       # VirtualBox NAT gateway (for MCP URL rewriting)
synced_folder_type = "virtualbox"   # Sync type: "virtualbox", "rsync", "nfs"

# Provisioning — what gets installed in the VM
# Base packages always included: docker.io, nodejs, npm, git, unzip, curl, build-essential
provision_packages = []             # Additional apt packages (appended to base set)
provision_packages_exclude = []     # Packages to remove from base set (use sparingly)
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
| `health_check_interval` | int | `30` | Seconds between VM health checks (hypervisor state + SSH liveness probe). Lower values detect crashes and hangs faster but add overhead. |

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

Each `[[vagrant.port_forwards]]` entry becomes a `config.vm.network "forwarded_port"` line in the generated Vagrantfile with `auto_correct: true`. If the requested host port is already in use (e.g., by another VM), Vagrant automatically assigns the next available port and logs the correction.

#### Provisioning

Control what software is installed in the VM. This is analogous to how `[mcps.*]` defines what MCP tools are available — `[vagrant]` defines what system packages and tools are available inside the VM.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `provision_packages` | array | `[]` | Additional apt packages installed during provisioning. **Appended** to the base set (see below). |
| `provision_packages_exclude` | array | `[]` | Packages to remove from the base set. Use sparingly — removing required packages may break Claude Code. |
| `npm_packages` | array | `[]` | Additional npm packages installed globally. `@anthropic-ai/claude-code` is always included regardless. MCP packages from `[mcps.*]` are also auto-included. |
| `provision_script` | string | `""` | Path to a custom shell script run after default provisioning. Use for project-specific setup (databases, language runtimes, config files). |

**Base packages (always installed unless excluded):**
`docker.io`, `nodejs`, `npm`, `git`, `unzip`, `curl`, `build-essential`

**Example: Add Python and Redis to the VM:**
```toml
[vagrant]
provision_packages = ["python3", "python3-pip", "redis-server"]
npm_packages = ["typescript", "tsx"]
provision_script = "scripts/vm-setup.sh"
# Result: base packages + python3 + python3-pip + redis-server
```

**Example: Remove Docker from the base set (rare):**
```toml
[vagrant]
provision_packages_exclude = ["docker.io"]
# Result: nodejs, npm, git, unzip, curl, build-essential (no docker)
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

These are forwarded to the VM via SSH `SendEnv`/`AcceptEnv` protocol — values never appear in the command string.

**Auto-injected proxy env vars:** When proxy env vars (`HTTP_PROXY`, `HTTPS_PROXY`, `NO_PROXY`, and their lowercase variants) are detected in the host environment, they are automatically forwarded to:
1. The VM provisioning script (via `export` lines in the Vagrantfile shell provisioner, before `apt-get` and `npm install`)
2. SSH sessions (via `SendEnv` alongside other env vars)

This ensures `vagrant up` provisioning and Claude's runtime both respect the host's proxy configuration. Disable with `vagrant.forward_proxy_env = false`.

```go
var proxyEnvVars = []string{
    "HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY",
    "http_proxy", "https_proxy", "no_proxy",
}

func collectProxyEnvVars() map[string]string {
    // Returns host proxy env vars that are set (non-empty)
    // Deduplicated: if both HTTP_PROXY and http_proxy are set, prefer uppercase
}
```

| Config Key | Type | Default | Description |
|-----------|------|---------|-------------|
| `forward_proxy_env` | bool | `true` | Auto-forward host proxy env vars to VM provisioning and sessions. Set `false` to disable. |

If the user also sets proxy vars in `[vagrant.env]`, those take precedence over auto-detected host values.

**Auto-injected polling env vars:** When `synced_folder_type = "virtualbox"` (default), the following env vars are automatically added to the session environment to enable polling-based file watchers (VirtualBox shared folders don't support inotify):

| Env Var | Value | Purpose |
|---------|-------|---------|
| `CHOKIDAR_USEPOLLING` | `1` | Polling mode for chokidar (webpack, vite, nodemon) |
| `WATCHPACK_POLLING` | `true` | Polling mode for watchpack (Next.js, webpack 5) |
| `TSC_WATCHFILE` | `UseFsEventsWithFallbackDynamicPolling` | Polling fallback for `tsc --watch` |

These are injected in `WrapCommand()` when `settings.SyncedFolderType == "virtualbox"`, merged with user-defined `vagrant.env`. User-defined values take precedence (allowing override). When using NFS or rsync, these are not injected since inotify works natively.

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
provision_packages = ["python3", "python3-pip", "python3-venv", "libpq-dev"]
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
| VirtualBox blocked on Apple Silicon | Kernel extension not approved in System Settings | Open System Settings → Privacy & Security → approve VirtualBox. Press R to retry. |
| First boot very slow | Downloading base box + provisioning | Normal. Subsequent boots use cached box (~30-60s). |
| MCP tool unavailable inside VM | npm package not installed in VM | Add to `npm_packages` or install in `provision_script`. |
| Can't reach host service from VM | Wrong gateway IP or service not listening on all interfaces | Check `host_gateway_ip`. Ensure host service binds to `0.0.0.0`, not just `127.0.0.1`. |
| File sync slow with node_modules | VirtualBox shared folders struggle with many small files | Use `synced_folder_type = "rsync"` or add `node_modules` to `.rsyncignore`. |
| Provisioning fails behind proxy | `apt-get` or `npm install` can't reach internet | Proxy vars auto-forwarded if set on host. Verify with `echo $HTTP_PROXY` inside VM. Override in `[vagrant.env]` or set `forward_proxy_env = false` to disable. |
| VM has outdated packages | Config changed but VM wasn't re-provisioned | Automatic: agent-deck detects config drift and runs `vagrant provision` on next start. Manual: delete session and recreate for clean slate. |
| Box changed but VM still uses old box | `vagrant provision` can't swap the base box | Delete the session and recreate. The new box will be downloaded on next `vagrant up`. |
| Port conflict on host | Another process or VM using the forwarded port | `auto_correct: true` assigns next available port automatically. Check `vagrant port` for actual mappings. |

### Config Reference Update (Implementation Note)

During implementation, add a `[vagrant]` section to `skills/agent-deck/references/config-reference.md` following the same format as existing sections (`[claude]`, `[mcps.*]`, etc.). Include the full key table, examples, and a link to this design document for architectural context.

### Loading Tips Content

100 tips compiled into `internal/vagrant/tips.go` — 50 Vagrant best practices and 50 world facts, displayed in shuffled order during VM boot. Each tip has text, source URL, and category.

#### Vagrant Best Practices (50)

1. "Use NFS for synced folders on macOS/Linux for significant I/O speed improvements over default VirtualBox shared folders." — vagrantup.com/docs/synced-folders/nfs
2. "Use rsync synced folders as a high-performance, one-way sync alternative that works on all platforms." — vagrantup.com/docs/synced-folders/rsync
3. "Allocate adequate memory with `vb.memory` inside the provider block to prevent swapping and sluggish performance." — vagrantup.com/docs/providers/virtualbox/configuration
4. "Assign multiple CPU cores with `vb.cpus` to improve performance for multi-threaded workloads like compilation." — vagrantup.com/docs/providers/virtualbox/configuration
5. "Enable KVM paravirtualization for Linux guests with `--paravirtprovider kvm` for better timer accuracy." — virtualbox.org/manual/ch03.html
6. "Use linked clones (`vb.linked_clone = true`) in multi-machine setups to save gigabytes of disk space per VM." — vagrantup.com/docs/providers/virtualbox/configuration
7. "Always use `Vagrant.configure(\"2\")` to lock your Vagrantfile to the current configuration format." — vagrantup.com/docs/vagrantfile
8. "Relocate the `.vagrant` metadata directory by setting `VAGRANT_DOTFILE_PATH` to keep source trees clean." — vagrantup.com/docs/other/environmental-variables
9. "Use Ruby loops in the Vagrantfile to define multiple similar machines without duplicating config." — vagrantup.com/docs/vagrantfile/tips
10. "Automate pre/post actions with Vagrant triggers to streamline your workflow." — vagrantup.com/docs/triggers
11. "Pass sensitive data into your Vagrantfile via `ENV['VAR_NAME']` instead of hardcoding." — vagrantup.com/docs/vagrantfile/tips
12. "Pin your box version with `config.vm.box_version` to prevent breakage from upstream updates." — vagrantup.com/docs/boxes/versioning
13. "Set `run: \"once\"` on shell provisioners so setup scripts only execute on first `vagrant up`." — vagrantup.com/docs/provisioning/shell
14. "Use `inline` shell provisioners for short commands to keep everything in the Vagrantfile." — vagrantup.com/docs/provisioning/shell
15. "Run provisioning scripts as non-root with `privileged: false` to practice least-privilege." — vagrantup.com/docs/provisioning/shell
16. "Use Ansible as your provisioner for complex environments to leverage idempotent configuration." — vagrantup.com/docs/provisioning/ansible
17. "Force re-provisioning on a running machine with `vagrant provision` to apply updated scripts." — vagrantup.com/docs/cli/provision
18. "Run `vagrant rsync-auto` to automatically watch for host file changes and sync in near real-time." — vagrantup.com/docs/cli/rsync-auto
19. "Use a private network for stable host-only communication that survives VM reboots." — vagrantup.com/docs/networking/private_network
20. "Forward specific guest ports to the host with `forwarded_port` for accessing web services." — vagrantup.com/docs/networking/forwarded_ports
21. "Use a public (bridged) network to give the VM its own IP on your LAN." — vagrantup.com/docs/networking/public_network
22. "Set a hostname with `config.vm.hostname` so the VM is identifiable in shell prompts and DNS." — vagrantup.com/docs/vagrantfile/machine_settings
23. "Set `auto_correct: true` on forwarded ports so Vagrant resolves host port conflicts automatically." — vagrantup.com/docs/networking/forwarded_ports
24. "Enable verbose logging with `VAGRANT_LOG=info vagrant up` to diagnose startup and provisioning issues." — vagrantup.com/docs/other/debugging
25. "Debug SSH issues with `vagrant ssh -- -vvv` for verbose SSH client output." — vagrantup.com/docs/cli/ssh
26. "Enable VirtualBox GUI with `vb.gui = true` to see console output during kernel panics or boot errors." — vagrantup.com/docs/providers/virtualbox/configuration
27. "Use `vagrant reload` to restart the guest and apply Vagrantfile changes that require a reboot." — vagrantup.com/docs/cli/reload
28. "When all else fails, `vagrant destroy -f && vagrant up` rebuilds from scratch." — vagrantup.com/docs/cli/destroy
29. "Create reusable boxes from running VMs with `vagrant package --output my-custom.box`." — vagrantup.com/docs/cli/package
30. "Only use boxes from trusted sources like the official HashiCorp Cloud catalog." — app.vagrantup.com/boxes/search
31. "Resolve DNS issues inside the guest with `--natdnshostresolver1 on` in VirtualBox customizations." — virtualbox.org/manual/ch08.html
32. "On Windows, use the `vagrant-winnfsd` plugin for NFS-like synced folder performance." — github.com/winnfsd/vagrant-winnfsd
33. "Install `vagrant-vbguest` to keep Guest Additions in sync with your VirtualBox version automatically." — github.com/dotless-de/vagrant-vbguest
34. "Use `vagrant-cachier` to cache package downloads across `vagrant destroy` cycles." — github.com/fgrehm/vagrant-cachier
35. "Resize VM disks without manual VBoxManage commands using the `vagrant-disksize` plugin." — github.com/sprotheroe/vagrant-disksize
36. "Manage host `/etc/hosts` entries for VMs automatically with `vagrant-hostsupdater`." — github.com/agiledivider/vagrant-hostsupdater
37. "Audit installed plugins with `vagrant plugin list` and remove unused ones to stay lean." — vagrantup.com/docs/cli/plugin
38. "Define multiple machines in one Vagrantfile with `config.vm.define` blocks." — vagrantup.com/docs/multi-machine
39. "Target commands to specific machines like `vagrant ssh web` or `vagrant halt db`." — vagrantup.com/docs/multi-machine
40. "Designate a primary machine with `primary: true` so bare commands default to it." — vagrantup.com/docs/multi-machine
41. "Connect multi-machine setups via a private network on the same subnet." — vagrantup.com/docs/multi-machine
42. "Check for outdated boxes across all environments with `vagrant box outdated --global`." — vagrantup.com/docs/cli/box
43. "Update boxes with `vagrant box update`; the running VM uses the new box on next destroy+up." — vagrantup.com/docs/cli/box
44. "Improve network throughput with virtio-net adapters: `--nictype1 virtio`." — vagrantup.com/docs/providers/virtualbox/configuration
45. "Reclaim disk space by pruning old box versions with `vagrant box prune`." — vagrantup.com/docs/cli/box
46. "Create custom base boxes with pre-installed toolchains to standardize team onboarding." — vagrantup.com/docs/boxes/base
47. "Use `vagrant suspend/resume` for the fastest start/stop cycle -- state saved and restored in seconds." — vagrantup.com/docs/cli/suspend
48. "Prefer `vagrant halt` for a graceful shutdown when you want to free all host resources." — vagrantup.com/docs/cli/halt
49. "Take named snapshots with `vagrant snapshot save` before risky changes for instant rollback." — vagrantup.com/docs/cli/snapshot
50. "Run `vagrant global-status --prune` to find and clean orphaned VMs consuming resources." — vagrantup.com/docs/cli/global-status

#### World Facts (50)

1. "About 8% of human DNA is made of ancient viral sequences — remnants of retroviral infections embedded over millions of years." — nature.com/scitable/topicpage/endogenous-retroviruses
2. "The placebo effect works even when people know they're taking a placebo; ritual and expectation alone produce measurable changes." — nature.com/articles/s41598-017-19185-1
3. "Aerogel is over 90% air by volume, looks like 'frozen smoke,' and is used by NASA for extreme insulation." — nasa.gov/mission_pages/stardust/multimedia/aerogel.html
4. "Ultra-thin gold films (nanometers thick) can transmit light, becoming transparent — a property bulk gold lacks." — nature.com/articles/nnano.2016.256
5. "The 'Banana Equivalent Dose' is a real radiation unit — bananas contain enough potassium-40 to serve as a baseline." — en.wikipedia.org/wiki/Banana_equivalent_dose
6. "The stoplight loosejaw dragonfish has rotatable red-light 'headlights' invisible to most deep-sea prey." — mbari.org/animal/stoplight-loosejaw-dragonfish/
7. "Researchers documented octopuses punching fish hunting partners with no obvious benefit — possibly spite." — cell.com/current-biology/fulltext/S0960-9822(22)00484-4
8. "Some sea slugs steal working chloroplasts from algae and keep them functioning — borrowing photosynthesis." — nationalgeographic.com/animals/invertebrates/facts/sea-slugs
9. "The bone-eating worm Osedax has no mouth or stomach — it dissolves whale bones using symbiotic bacteria." — mbari.org/animal/osedax/
10. "Certain caterpillar larvae stack prey corpses onto specialized back bristles as camouflage." — nationalgeographic.com/animals/article/trash-carrying-larvae-insects
11. "The fungus Ophiocordyceps hijacks ant nervous systems, forcing them to climb and clamp at a precise height before erupting spores." — nationalgeographic.com/animals/article/cordyceps-zombie-fungus-ants
12. "Leafcutter ants don't eat the leaves they harvest — they use them to cultivate underground fungus gardens." — britannica.com/animal/leafcutter-ant
13. "Wombat poop is cube-shaped, likely preventing it from rolling away on slopes for more effective territory marking." — nationalgeographic.com/animals/mammals/facts/wombat
14. "Some frogs survive being literally frozen solid by producing glucose cryoprotectants in their cells." — britannica.com/animal/wood-frog
15. "Plants can boost chemical defenses when exposed to vibrations matching caterpillar chewing — a form of 'hearing.'" — scientificamerican.com/article/can-plants-hear/
16. "On exoplanet WASP-76b, iron vaporizes on the dayside and rains down as liquid iron on the cooler nightside." — eso.org/public/news/eso1916/
17. "JWST identifies specific molecules in exoplanet atmospheres billions of miles away — turning starlight into chemistry." — nasa.gov/mission/webb/
18. "Stars oscillate like instruments; asteroseismology studies these 'rings' to determine internal structure." — esa.int/Science_Exploration/Space_Science/Asteroseismology
19. "A teaspoon of neutron star material would weigh roughly one billion tons on Earth." — nasa.gov/universe/neutron-stars/
20. "The Moon drifts away from Earth at 3.8 cm/year — measured by bouncing lasers off Apollo reflectors." — nasa.gov/moon-facts/
21. "A day on Venus (243 Earth days) is longer than its year (225 Earth days), and it rotates backwards." — solarsystem.nasa.gov/planets/venus/
22. "The ISS orbits Earth every 90 minutes and is the third-brightest object in the night sky." — spotthestation.nasa.gov/
23. "The Anglo-Zanzibar War of 1896 lasted 38-45 minutes — the shortest war in recorded history." — britannica.com/event/Anglo-Zanzibar-War
24. "The first documented 'computer bug' was a literal moth found in a Harvard Mark II relay in 1947." — computerhistory.org/tdih/september/9/
25. "Linear B was deciphered to reveal early Greek — overturning assumptions it was an unknown lost language." — britannica.com/topic/Linear-B
26. "Peru's 'Boiling River' reaches scalding temperatures from deep geothermal heat, not volcanic lava." — bbc.com/travel/article/20160518-the-amazon-rainforests-mysterious-boiling-river
27. "Death Valley's Racetrack Playa rocks slide on their own — explained by thin ice sheets pushed by wind." — nps.gov/deva/learn/nature/racetrack.htm
28. "Earth has measurable 'gravity holes' — Canada's Hudson Bay has weaker gravity from glacial rebound effects." — nasa.gov/mission_pages/GRACE/
29. "Lake Nyos in Cameroon 'burped' a lethal CO2 cloud in 1986, suffocating 1,700 people in minutes." — britannica.com/event/Lake-Nyos-disaster
30. "Desert sand dunes can 'sing' — producing deep resonant hums audible for miles when sand avalanches." — britannica.com/science/singing-sand
31. "New Zealand has the world's longest place name at 85 characters: Taumatawhakatangihanga..." — guinnessworldrecords.com/world-records/longest-place-name
32. "Earth's mantle holds water locked inside minerals — potentially more than all surface oceans combined." — nature.com/articles/nature13080
33. "The 'liking gap': after conversations, people consistently underestimate how much strangers liked them." — journals.sagepub.com/doi/10.1177/0956797618783714
34. "Most smartphone users experience 'phantom vibration syndrome' — feeling the phone buzz when it hasn't." — psychologytoday.com/us/blog/brain-myths/201307/phantom-vibration-syndrome
35. "The 'decoy effect': adding an inferior third option measurably shifts preference toward a target choice." — behavioraleconomics.com/resources/mini-encyclopedia-of-be/decoy-effect/
36. "The 'uncanny valley' makes near-perfect human replicas feel eerier than obviously fake ones." — britannica.com/science/uncanny-valley
37. "The 'Mozart effect' is overstated — any measured boost is short-lived and linked to arousal, not intelligence." — britannica.com/story/does-listening-to-mozart-really-make-you-smarter
38. "Candy banana flavor resembles the Gros Michel variety, nearly wiped out by fungus in the 1950s, not modern bananas." — smithsonianmag.com/arts-culture/why-dont-banana-candies-taste-like-real-bananas
39. "Bacteriophages (viruses attacking bacteria) can alter cheese flavor by disrupting starter cultures during fermentation." — asm.org/Articles/2019/December/Bacteriophages-and-the-Dairy-Industry
40. "Cold-brew coffee is chemically different from hot-brew — cold extraction shifts acidity and volatile compounds." — acs.org/pressroom/presspacs/2018/acs-presspac-march-28-2018.html
41. "You can't 'taste' spicy — capsaicin activates heat/pain receptors, not taste buds." — britannica.com/science/capsaicin
42. "The Netherlands has more bicycles than people and exports cycling infrastructure consulting worldwide." — government.nl/topics/bicycles
43. "Blue Zone longevity research found social connection and purpose matter as much as diet for long life." — nationalgeographic.com/magazine/article/secrets-of-long-life
44. "Some languages have no left/right — speakers use cardinal directions for everything and maintain remarkable orientation." — pnas.org/doi/10.1073/pnas.0702920104
45. "Crows recognize individual human faces for years and socially communicate this to other crows — a collective 'grudge list.'" — scientificamerican.com/article/crows-never-forget-your-face/
46. "The vagus nerve is the literal wiring behind 'gut feelings' — a highway between gut and brain regulating mood." — nature.com/articles/d41586-022-01043-4
47. "The longest tennis match lasted 11 hours 5 minutes across 3 days at Wimbledon 2010 (Isner vs Mahut, 70-68 final set)." — wimbledon.com/en_GB/news/articles/2019-06-24/the_longest_match.html
48. "The fastest red card in professional soccer was given within seconds of kickoff for violent conduct." — guinnessworldrecords.com/world-records/fastest-red-card
49. "'Extreme ironing' is a real sport — competitors iron clothes on mountaintops, underwater, and while skydiving." — bbc.com/news/uk-england-25823682
50. "In rare 'electrophonic' meteor events, observers hear crackling sounds simultaneously with seeing the meteor." — britannica.com/science/meteor-astronomy

## Next Steps

- [ ] Create implementation plan (use `agentic-ai-plan`)
- [ ] Set up worktree for implementation
- [ ] Execute plan with agent team
