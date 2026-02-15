package vagrant

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/asheshgoplani/agent-deck/internal/session"
)

// Manager manages the Vagrant VM lifecycle and configuration for a project.
// It handles config drift detection to notify users when VM needs re-provisioning.
type Manager struct {
	projectPath string
	settings    session.VagrantSettings
}

// configHash computes a deterministic SHA-256 hash of the VM configuration inputs.
// This hash is used to detect when the VM configuration has changed (config drift).
// The hash includes:
// - Base box name
// - System packages (base + custom, excluding excluded packages)
// - NPM packages
// - Provision script content (if present)
// - Port forwarding rules
func (m *Manager) configHash() string {
	h := sha256.New()

	// Hash the box name
	h.Write([]byte(m.settings.Box))

	// Hash system packages: base packages + custom, excluding excluded packages
	basePackages := []string{
		"docker.io",
		"nodejs",
		"npm",
		"git",
		"unzip",
		"curl",
		"build-essential",
	}

	// Create a set of excluded packages for fast lookup
	excludeSet := make(map[string]bool)
	for _, pkg := range m.settings.ProvisionPkgExclude {
		excludeSet[pkg] = true
	}

	// Build final package list: base - excluded + custom
	var packages []string
	for _, pkg := range basePackages {
		if !excludeSet[pkg] {
			packages = append(packages, pkg)
		}
	}
	packages = append(packages, m.settings.ProvisionPackages...)

	// Sort for deterministic ordering
	sort.Strings(packages)
	h.Write([]byte(strings.Join(packages, ",")))

	// Hash NPM packages (sorted for determinism)
	npmPkgs := make([]string, len(m.settings.NpmPackages))
	copy(npmPkgs, m.settings.NpmPackages)
	sort.Strings(npmPkgs)
	h.Write([]byte(strings.Join(npmPkgs, ",")))

	// Hash provision script content if present
	if m.settings.ProvisionScript != "" {
		content, err := os.ReadFile(m.settings.ProvisionScript)
		if err == nil {
			h.Write(content)
		}
		// If script file cannot be read, we still continue (file might be deleted, etc.)
	}

	// Hash port forwarding rules
	for _, pf := range m.settings.PortForwards {
		portForwardStr := fmt.Sprintf("%d:%d:%s", pf.Guest, pf.Host, pf.Protocol)
		h.Write([]byte(portForwardStr))
	}

	return hex.EncodeToString(h.Sum(nil))
}

// HasConfigDrift checks if the VM configuration has drifted from the stored hash.
// Returns true if the stored hash exists and differs from the current config hash.
// Returns false if no hash file exists (first run) or if hashes match.
func (m *Manager) HasConfigDrift() bool {
	hashFile := filepath.Join(m.projectPath, ".vagrant", "agent-deck-config.sha256")
	stored, err := os.ReadFile(hashFile)
	if err != nil {
		// No hash file exists = first run, no drift
		return false
	}
	return strings.TrimSpace(string(stored)) != m.configHash()
}

// WriteConfigHash persists the current configuration hash to disk.
// Creates the .vagrant directory if it doesn't exist.
// The hash is written to .vagrant/agent-deck-config.sha256 for later comparison.
func (m *Manager) WriteConfigHash() error {
	vagrantDir := filepath.Join(m.projectPath, ".vagrant")
	if err := os.MkdirAll(vagrantDir, 0o755); err != nil {
		return fmt.Errorf("failed to create .vagrant directory: %w", err)
	}

	hashFile := filepath.Join(vagrantDir, "agent-deck-config.sha256")
	if err := os.WriteFile(hashFile, []byte(m.configHash()), 0o644); err != nil {
		return fmt.Errorf("failed to write config hash: %w", err)
	}

	return nil
}
