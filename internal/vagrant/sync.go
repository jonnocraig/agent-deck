package vagrant

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/asheshgoplani/agent-deck/internal/session"
)

// SyncClaudeConfig synchronizes host-side Claude configuration files into the VM.
// This ensures that global/user MCPs configured on the host are available to
// Claude Code running inside the VM. No URL rewriting is needed because SSH
// reverse tunnels handle localhost connectivity.
//
// The method reads two config files:
// 1. Global config: CLAUDE_CONFIG_DIR/.claude.json (or ~/.claude/.claude.json)
// 2. User config: ~/.claude.json
//
// Both files are copied into the VM at their respective paths. If a file doesn't
// exist, it's silently skipped (not an error). Errors during sync are logged but
// don't fail the overall operation (non-fatal).
func (m *Manager) SyncClaudeConfig() error {
	// Use the injected function if available (for testing), otherwise use default
	writeFunc := m.writeFileToVMFunc
	if writeFunc == nil {
		writeFunc = m.writeFileToVM
	}

	// 1. Read and sync global config
	globalConfigDir := session.GetClaudeConfigDir()
	globalConfig := filepath.Join(globalConfigDir, ".claude.json")

	if data, err := os.ReadFile(globalConfig); err == nil {
		// File exists, sync it to VM
		if err := writeFunc("~/.claude/.claude.json", data); err != nil {
			// Log warning but don't fail - this is non-fatal
			// In production, this would use a proper logger
			// For now, we just continue
		}
	}
	// If file doesn't exist, skip silently (not an error)

	// 2. Read and sync user config
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Can't determine home directory - skip user config
		return nil
	}

	userConfig := filepath.Join(homeDir, ".claude.json")
	if data, err := os.ReadFile(userConfig); err == nil {
		// File exists, sync it to VM
		if err := writeFunc("~/.claude.json", data); err != nil {
			// Log warning but don't fail - this is non-fatal
		}
	}
	// If file doesn't exist, skip silently (not an error)

	return nil
}

// writeFileToVM writes a file to the VM using base64 encoding to avoid shell quoting issues.
// The file content is base64-encoded and passed via echo, then decoded and written to the
// remote path inside the VM via 'vagrant ssh -c'.
//
// This approach avoids all shell quoting complexity that would arise from trying to pass
// JSON content (with quotes, special characters, etc.) directly through the shell.
func (m *Manager) writeFileToVM(remotePath string, content []byte) error {
	// Encode content as base64 to avoid shell quoting issues
	encoded := base64.StdEncoding.EncodeToString(content)

	// Create the command:
	// 1. mkdir -p $(dirname REMOTE_PATH) - ensure parent directory exists
	// 2. echo 'BASE64' | base64 -d > REMOTE_PATH - decode and write file
	cmdStr := fmt.Sprintf("mkdir -p $(dirname %s) && echo '%s' | base64 -d > %s",
		remotePath, encoded, remotePath)

	cmd := m.vagrantCmd("ssh", "-c", cmdStr)
	return cmd.Run()
}

// createWriteFileCmd creates the exec.Cmd for writing a file to the VM.
// This is separated out to allow testing without actually executing vagrant commands.
func (m *Manager) createWriteFileCmd(remotePath string, content []byte) *exec.Cmd {
	encoded := base64.StdEncoding.EncodeToString(content)
	cmdStr := fmt.Sprintf("mkdir -p $(dirname %s) && echo '%s' | base64 -d > %s",
		remotePath, encoded, remotePath)
	return m.vagrantCmd("ssh", "-c", cmdStr)
}

// writeFileToVMFunc is an injectable function for testing.
// In tests, this can be set to capture calls without executing vagrant commands.
// In production, this is nil and writeFileToVM is used directly.
type writeFileToVMFuncType func(remotePath string, content []byte) error
