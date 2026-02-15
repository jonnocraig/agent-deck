package vagrant

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/asheshgoplani/agent-deck/internal/session"
)

// Manager implements the VagrantProvider interface for managing Vagrant VM lifecycle.
// This is the canonical Manager struct definition - DO NOT duplicate in other files.
type Manager struct {
	projectPath        string
	settings           session.VagrantSettings
	dotfilePath        string                 // for VAGRANT_DOTFILE_PATH (multi-session isolation)
	sessions           []string               // session IDs sharing this VM
	cache              *healthCache           // health check cache (defined in health.go)
	mu                 sync.Mutex
	writeFileToVMFunc  writeFileToVMFuncType  // injectable for testing (defined in sync.go)
}

// Compile-time interface compliance check
var _ VagrantProvider = (*Manager)(nil)

// NewManager creates a new Manager instance for managing a Vagrant VM.
func NewManager(projectPath string, settings session.VagrantSettings) *Manager {
	m := &Manager{
		projectPath: projectPath,
		settings:    settings,
		sessions:    []string{},
	}
	// Load existing sessions from lockfile if present
	m.loadLockfile()
	return m
}

// vagrantCmd creates an exec.Cmd for running vagrant with the given arguments.
// Sets the working directory to projectPath and adds VAGRANT_DOTFILE_PATH env if set.
func (m *Manager) vagrantCmd(args ...string) *exec.Cmd {
	cmd := exec.Command("vagrant", args...)
	cmd.Dir = m.projectPath

	if m.dotfilePath != "" {
		// Propagate current environment and add VAGRANT_DOTFILE_PATH
		cmd.Env = append(os.Environ(), "VAGRANT_DOTFILE_PATH="+m.dotfilePath)
	}

	return cmd
}

// IsInstalled checks if the Vagrant CLI is available in PATH.
func (m *Manager) IsInstalled() bool {
	_, err := exec.LookPath("vagrant")
	return err == nil
}

// Status returns the current Vagrant VM state by parsing machine-readable output.
func (m *Manager) Status() (string, error) {
	cmd := m.vagrantCmd("status", "--machine-readable")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("vagrant status failed: %w", err)
	}

	// Parse machine-readable output for state line
	// Format: timestamp,target,state,value
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Split(line, ",")
		if len(fields) >= 4 && fields[2] == "state" {
			return fields[3], nil
		}
	}

	return "", fmt.Errorf("could not parse vagrant state from output")
}

// EnsureRunning ensures the VM is running and fully provisioned.
// Calls onPhase callback with each boot phase for user feedback.
// Idempotent - safe to call if VM is already running.
func (m *Manager) EnsureRunning(onPhase func(BootPhase)) error {
	cmd := m.vagrantCmd("up", "--machine-readable")

	// Capture both stdout and stderr for error handling
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	// Stream stdout to parse boot phases
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return wrapVagrantUpError(err, stderrBuf.String())
	}

	// Parse output line by line for boot phases
	if onPhase != nil {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			// Use ParseBootPhase from bootphase.go
			if phase, ok := ParseBootPhase(line); ok {
				onPhase(phase)
			}
		}
	}

	if err := cmd.Wait(); err != nil {
		return wrapVagrantUpError(err, stderrBuf.String())
	}

	return nil
}

// Suspend suspends the running VM to disk.
func (m *Manager) Suspend() error {
	cmd := m.vagrantCmd("suspend")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("vagrant suspend failed: %w", err)
	}
	return nil
}

// Resume resumes a suspended VM.
func (m *Manager) Resume() error {
	cmd := m.vagrantCmd("resume")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("vagrant resume failed: %w", err)
	}
	return nil
}

// Destroy completely destroys the VM and removes all data.
func (m *Manager) Destroy() error {
	cmd := m.vagrantCmd("destroy", "-f")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("vagrant destroy failed: %w", err)
	}
	return nil
}

// ForceRestart performs a hard restart by destroying and recreating the VM.
func (m *Manager) ForceRestart() error {
	if err := m.Destroy(); err != nil {
		return err
	}
	return m.EnsureRunning(nil)
}

// Reload reloads VM configuration (vagrant reload).
func (m *Manager) Reload() error {
	cmd := m.vagrantCmd("reload")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("vagrant reload failed: %w", err)
	}
	return nil
}

// Provision runs provisioning scripts without restarting the VM.
func (m *Manager) Provision() error {
	cmd := m.vagrantCmd("provision")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("vagrant provision failed: %w", err)
	}
	return nil
}

// Note: The following methods are implemented in their respective files:
// - WrapCommand() - wrap.go
// - EnsureSudoSkill() - skill.go
// - SyncClaudeConfig() - sync.go
// - PreflightCheck() - preflight.go
// - HealthCheck() - health.go
// - CheckDiskSpace() - preflight.go
// - IsBoxCached() - preflight.go
// - HasConfigDrift() - drift.go
// - WriteConfigHash() - drift.go
// - RegisterSession() - sessions.go
// - UnregisterSession() - sessions.go
// - SessionCount() - sessions.go
// - IsLastSession() - sessions.go
// - SetDotfilePath() - sessions.go
// - loadLockfile() - sessions.go (private method)
// - GetMCPPackages() - mcp.go
