// Package vagrant provides interfaces and types for managing Vagrant VM lifecycle
// in support of the "Just Do It" mode that runs Claude Code inside a sandboxed VM
// with --dangerously-skip-permissions and sudo access.
package vagrant

// BootPhase represents a specific phase during VM boot/provisioning process.
// Used to provide user feedback during potentially long-running VM startup.
type BootPhase string

const (
	// BootPhaseDownloading indicates the base box is being downloaded from Vagrant Cloud
	BootPhaseDownloading BootPhase = "Downloading base box..."

	// BootPhaseImporting indicates the downloaded box is being imported into VirtualBox
	BootPhaseImporting BootPhase = "Importing base box..."

	// BootPhaseBooting indicates the VM is starting up
	BootPhaseBooting BootPhase = "Booting VM..."

	// BootPhaseNetwork indicates network interfaces are being configured
	BootPhaseNetwork BootPhase = "Configuring network..."

	// BootPhaseMounting indicates shared folders are being mounted
	BootPhaseMounting BootPhase = "Mounting shared folders..."

	// BootPhaseProvisioning indicates system packages are being installed
	BootPhaseProvisioning BootPhase = "Provisioning (installing packages)..."

	// BootPhaseNpmInstall indicates Claude Code and MCP tools are being installed
	BootPhaseNpmInstall BootPhase = "Installing Claude Code & MCP tools..."

	// BootPhaseReady indicates VM is fully provisioned and ready to run Claude
	BootPhaseReady BootPhase = "VM ready — starting Claude..."
)

// VMHealth represents the current health status of the Vagrant VM.
type VMHealth struct {
	// State is the Vagrant state string ("running", "suspended", "not_created", etc.)
	State string

	// Healthy is true if VM is running and responsive
	Healthy bool

	// Responsive is true if SSH liveness probe passed
	Responsive bool

	// Message is a human-readable status message
	Message string
}

// VagrantProvider defines the interface for all Vagrant VM lifecycle operations.
// This interface enables dependency injection and testing by allowing mock implementations.
type VagrantProvider interface {
	// IsInstalled checks if Vagrant CLI is available in PATH
	IsInstalled() bool

	// PreflightCheck validates that Vagrant and VirtualBox are properly installed
	// and configured. Returns error if prerequisites are missing.
	PreflightCheck() error

	// EnsureRunning ensures the VM is running and fully provisioned.
	// Calls onPhase callback with each boot phase for user feedback.
	// Idempotent - safe to call if VM is already running.
	EnsureRunning(onPhase func(BootPhase)) error

	// Suspend suspends the running VM to disk
	Suspend() error

	// Resume resumes a suspended VM
	Resume() error

	// Destroy completely destroys the VM and removes all data
	Destroy() error

	// ForceRestart performs a hard restart (destroy + recreate)
	ForceRestart() error

	// Reload reloads VM configuration (vagrant reload)
	Reload() error

	// Provision runs provisioning scripts without restarting VM
	Provision() error

	// Status returns the current Vagrant status string
	Status() (string, error)

	// HealthCheck performs comprehensive health check including SSH liveness probe
	HealthCheck() (VMHealth, error)

	// WrapCommand wraps a command for execution inside the VM via 'vagrant ssh --'.
	// Handles SSH environment variable forwarding and reverse tunnel setup.
	// envVarNames: environment variables to forward via SendEnv
	// tunnelPorts: local ports to reverse tunnel into the VM
	WrapCommand(cmd string, envVarNames []string, tunnelPorts []int) string

	// EnsureVagrantfile creates or updates the Vagrantfile in the project root
	EnsureVagrantfile() error

	// EnsureSudoSkill creates the claude-sudo skill inside the VM if not present
	EnsureSudoSkill() error

	// SyncClaudeConfig synchronizes ~/.claude config into the VM
	SyncClaudeConfig() error

	// GetMCPPackages returns the list of npx MCP packages to install in the VM
	GetMCPPackages() []string

	// HasConfigDrift checks if Vagrantfile or provisioning scripts have changed
	// since last VM creation
	HasConfigDrift() bool

	// WriteConfigHash writes current config hash to detect future drift
	WriteConfigHash() error

	// RegisterSession registers a new Claude Code session using the VM
	// Returns error if lockfile write fails.
	RegisterSession(sessionID string) error

	// UnregisterSession removes a session registration
	// Returns error if lockfile write fails.
	UnregisterSession(sessionID string) error

	// SessionCount returns the number of active sessions using the VM
	SessionCount() int

	// IsLastSession returns true if this is the last remaining session
	IsLastSession(sessionID string) bool

	// SetDotfilePath sets the path to sync .claude dotfiles from
	SetDotfilePath(sessionID string)

	// CheckDiskSpace returns available disk space in bytes on the VM
	CheckDiskSpace() (int64, error)

	// IsBoxCached returns true if the base box is already downloaded
	IsBoxCached() bool
}
