package vagrant

import (
	"github.com/asheshgoplani/agent-deck/internal/session"
)

func init() {
	session.VagrantProviderFactory = newVagrantVM
}

// vagrantVMAdapter wraps a Manager to satisfy session.vagrantVM interface.
// This adapter exists because HealthCheck() needs to return session.VMHealthResult
// (instead of vagrant.VMHealth) to avoid the import cycle.
type vagrantVMAdapter struct {
	mgr *Manager
}

func newVagrantVM(projectPath string, settings session.VagrantSettings) session.VagrantVM {
	mgr := NewManager(projectPath, settings)
	return &vagrantVMAdapter{mgr: mgr}
}

func (a *vagrantVMAdapter) PreflightCheck() error             { return a.mgr.PreflightCheck() }
func (a *vagrantVMAdapter) EnsureVagrantfile() error          { return a.mgr.EnsureVagrantfile() }
func (a *vagrantVMAdapter) EnsureSudoSkill() error            { return a.mgr.EnsureSudoSkill() }
func (a *vagrantVMAdapter) Boot() error                       { return a.mgr.EnsureRunning(nil) }
func (a *vagrantVMAdapter) Suspend() error                    { return a.mgr.Suspend() }
func (a *vagrantVMAdapter) Resume() error                     { return a.mgr.Resume() }
func (a *vagrantVMAdapter) Destroy() error                    { return a.mgr.Destroy() }
func (a *vagrantVMAdapter) ForceRestart() error               { return a.mgr.ForceRestart() }
func (a *vagrantVMAdapter) Reload() error                     { return a.mgr.Reload() }
func (a *vagrantVMAdapter) Provision() error                  { return a.mgr.Provision() }
func (a *vagrantVMAdapter) Status() (string, error)           { return a.mgr.Status() }
func (a *vagrantVMAdapter) SyncClaudeConfig() error           { return a.mgr.SyncClaudeConfig() }
func (a *vagrantVMAdapter) HasConfigDrift() bool              { return a.mgr.HasConfigDrift() }
func (a *vagrantVMAdapter) WriteConfigHash() error            { return a.mgr.WriteConfigHash() }
func (a *vagrantVMAdapter) RegisterSession(id string)         { a.mgr.RegisterSession(id) }
func (a *vagrantVMAdapter) UnregisterSession(id string)       { a.mgr.UnregisterSession(id) }
func (a *vagrantVMAdapter) SessionCount() int                 { return a.mgr.SessionCount() }
func (a *vagrantVMAdapter) IsLastSession(id string) bool      { return a.mgr.IsLastSession(id) }
func (a *vagrantVMAdapter) SetDotfilePath(id string)           { a.mgr.SetDotfilePath(id) }
func (a *vagrantVMAdapter) IsInstalled() bool                 { return a.mgr.IsInstalled() }
func (a *vagrantVMAdapter) IsBoxCached() bool                 { return a.mgr.IsBoxCached() }

func (a *vagrantVMAdapter) WrapCommand(cmd string, envVarNames []string, tunnelPorts []int) string {
	return a.mgr.WrapCommand(cmd, envVarNames, tunnelPorts)
}

func (a *vagrantVMAdapter) HealthCheck() (session.VMHealthResult, error) {
	health, err := a.mgr.HealthCheck()
	return session.VMHealthResult{
		State:      health.State,
		Healthy:    health.Healthy,
		Responsive: health.Responsive,
		Message:    health.Message,
	}, err
}

func (a *vagrantVMAdapter) WriteMCPJson(projectPath string, enabledNames []string) error {
	return WriteMCPJsonForVagrant(projectPath, enabledNames)
}

func (a *vagrantVMAdapter) CollectEnvVarNames(enabledNames []string, vagrantEnv map[string]string) []string {
	return CollectEnvVarNames(enabledNames, vagrantEnv)
}

func (a *vagrantVMAdapter) CollectTunnelPorts(enabledNames []string) []int {
	return CollectHTTPMCPPorts(enabledNames)
}
