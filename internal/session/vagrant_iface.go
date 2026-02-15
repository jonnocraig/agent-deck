package session

// VagrantVM defines the interface for Vagrant VM lifecycle operations
// needed by Instance. This is defined in the session package to avoid
// an import cycle (vagrant imports session for types).
// The vagrant package provides an adapter via VagrantProviderFactory.
type VagrantVM interface {
	PreflightCheck() error
	EnsureVagrantfile() error
	EnsureSudoSkill() error
	Boot() error
	Suspend() error
	Resume() error
	Destroy() error
	ForceRestart() error
	Reload() error
	Provision() error
	Status() (string, error)
	HealthCheck() (VMHealthResult, error)
	WrapCommand(cmd string, envVarNames []string, tunnelPorts []int) string
	SyncClaudeConfig() error
	WriteMCPJson(projectPath string, enabledNames []string) error
	CollectEnvVarNames(enabledNames []string, vagrantEnv map[string]string) []string
	CollectTunnelPorts(enabledNames []string) []int
	HasConfigDrift() bool
	WriteConfigHash() error
	RegisterSession(sessionID string)
	UnregisterSession(sessionID string)
	SessionCount() int
	IsLastSession(sessionID string) bool
	SetDotfilePath(sessionID string)
	IsInstalled() bool
	IsBoxCached() bool
}

// VMHealthResult represents VM health status (mirrors vagrant.VMHealth
// but defined here to avoid import cycle).
type VMHealthResult struct {
	State      string
	Healthy    bool
	Responsive bool
	Message    string
}

// VagrantProviderFactory creates a VagrantVM for the given project path and settings.
// Set by the vagrant package at init time to break the import cycle.
var VagrantProviderFactory func(projectPath string, settings VagrantSettings) VagrantVM
