package session

import (
	"sync/atomic"
)

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
	RegisterSession(sessionID string) error
	UnregisterSession(sessionID string) error
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

// vagrantProviderFactoryValue holds the factory function in an atomic.Value
// for thread-safe access during parallel tests.
var vagrantProviderFactoryValue atomic.Value

// SetVagrantProviderFactory sets the factory function. Called by vagrant package init().
func SetVagrantProviderFactory(fn func(projectPath string, settings VagrantSettings) VagrantVM) {
	vagrantProviderFactoryValue.Store(fn)
}

// GetVagrantProviderFactory returns the registered factory function, or nil if not set.
func GetVagrantProviderFactory() func(projectPath string, settings VagrantSettings) VagrantVM {
	v := vagrantProviderFactoryValue.Load()
	if v == nil {
		return nil
	}
	fn, ok := v.(func(projectPath string, settings VagrantSettings) VagrantVM)
	if !ok {
		return nil
	}
	return fn
}
