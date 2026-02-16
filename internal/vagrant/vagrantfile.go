package vagrant

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// resolvedPackages computes the final apt package list.
// Base packages: docker.io, nodejs, npm, git, unzip, curl, build-essential
// Removes any in settings.ProvisionPkgExclude
// Appends settings.ProvisionPackages
// Returns deduplicated sorted list
func (m *Manager) resolvedPackages() []string {
	basePackages := []string{
		"docker.io",
		"nodejs",
		"npm",
		"git",
		"unzip",
		"curl",
		"build-essential",
	}

	// Create exclusion set for O(1) lookup
	excludeSet := make(map[string]bool)
	for _, pkg := range m.settings.ProvisionPkgExclude {
		excludeSet[pkg] = true
	}

	// Filter base packages (remove excluded)
	filtered := make([]string, 0, len(basePackages))
	for _, pkg := range basePackages {
		if !excludeSet[pkg] {
			filtered = append(filtered, pkg)
		}
	}

	// Append custom packages
	allPackages := append(filtered, m.settings.ProvisionPackages...)

	// Deduplicate using map
	deduped := make(map[string]bool)
	for _, pkg := range allPackages {
		deduped[pkg] = true
	}

	// Convert back to slice and sort
	result := make([]string, 0, len(deduped))
	for pkg := range deduped {
		result = append(result, pkg)
	}
	sort.Strings(result)

	return result
}

// sanitizeHostname converts a project name to an RFC 1123 compliant hostname.
// - Lowercase
// - Replace non-alphanumeric chars with -
// - Trim leading/trailing -
// - Truncate to 63 chars
// - Prefix with agentdeck-
func (m *Manager) sanitizeHostname(projectName string) string {
	// Lowercase
	s := strings.ToLower(projectName)

	// Replace non-alphanumeric chars with -
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	s = reg.ReplaceAllString(s, "-")

	// Trim leading/trailing -
	s = strings.Trim(s, "-")

	// Add prefix
	s = "agentdeck-" + s

	// Truncate to 63 chars (RFC 1123 limit)
	if len(s) > 63 {
		s = s[:63]
	}

	// Trim trailing - after truncation
	s = strings.TrimRight(s, "-")

	return s
}

// EnsureVagrantfile generates Vagrantfile in projectPath.
// - If Vagrantfile already exists → return nil (skip, don't overwrite)
// - If settings.Vagrantfile is set → copy that file to projectPath/Vagrantfile
// - Otherwise generate from template
func (m *Manager) EnsureVagrantfile() error {
	vagrantfilePath := filepath.Join(m.projectPath, "Vagrantfile")

	// Skip if Vagrantfile already exists
	if _, err := os.Stat(vagrantfilePath); err == nil {
		return nil
	}

	// If custom Vagrantfile is specified, copy it
	if m.settings.Vagrantfile != "" {
		content, err := os.ReadFile(m.settings.Vagrantfile)
		if err != nil {
			return fmt.Errorf("failed to read custom Vagrantfile: %w", err)
		}
		return os.WriteFile(vagrantfilePath, content, 0644)
	}

	// Generate from template
	content := m.generateVagrantfile()
	return os.WriteFile(vagrantfilePath, []byte(content), 0644)
}

// generateVagrantfile generates Vagrantfile content from template
func (m *Manager) generateVagrantfile() string {
	// Determine hostname from project path
	projectName := filepath.Base(m.projectPath)
	hostname := m.sanitizeHostname(projectName)

	// Get resolved packages
	packages := m.resolvedPackages()
	packagesStr := strings.Join(packages, " ")

	// Build port forwards section
	var portForwards strings.Builder
	for _, pf := range m.settings.PortForwards {
		protocol := pf.Protocol
		if protocol == "" {
			protocol = "tcp"
		}
		fmt.Fprintf(&portForwards, "  config.vm.network \"forwarded_port\", guest: %d, host: %d, protocol: %q, auto_correct: true\n",
			pf.Guest, pf.Host, protocol)
	}

	// Build npm packages section
	var npmSection strings.Builder
	allNpmPackages := append([]string{}, m.settings.NpmPackages...)

	// Add MCP packages (call GetMCPPackages if Manager implements VagrantProvider)
	// For now, we'll leave this empty as GetMCPPackages is part of the interface
	// and will be implemented in manager.go

	if len(allNpmPackages) > 0 {
		npmPackagesStr := strings.Join(allNpmPackages, " ")
		fmt.Fprintf(&npmSection, "    npm install -g %s --no-audit\n", npmPackagesStr)
	}

	// Build provision script section
	var provisionScriptSection string
	if m.settings.ProvisionScript != "" {
		provisionScriptSection = fmt.Sprintf("\n  config.vm.provision \"shell\", path: %q\n", m.settings.ProvisionScript)
	}

	// Memory and CPU defaults
	memory := m.settings.MemoryMB
	if memory <= 0 {
		memory = 4096
	}
	cpus := m.settings.CPUs
	if cpus <= 0 {
		cpus = 2
	}

	// Box default
	box := m.settings.Box
	if box == "" {
		box = "bento/ubuntu-24.04"
	}

	// SyncedFolderType default
	syncType := m.settings.SyncedFolderType
	if syncType == "" {
		syncType = "virtualbox"
	}

	template := fmt.Sprintf(`Vagrant.configure("2") do |config|
  config.vm.box = %q
  config.vm.hostname = %q
  config.vm.synced_folder ".", "/vagrant", type: %q
  config.ssh.forward_agent = true

%s
  config.vm.provider "virtualbox" do |vb|
    vb.memory = "%d"
    vb.cpus = %d
    vb.gui = false
    vb.customize ["modifyvm", :id, "--audio", "none"]
    vb.customize ["modifyvm", :id, "--usb", "off"]
    vb.customize ["modifyvm", :id, "--nested-hw-virt", "on"]
  end

  config.vm.provision "shell", inline: <<-SHELL
    export DEBIAN_FRONTEND=noninteractive
    apt-get update
    apt-get install -y %s
    npm install -g @anthropic-ai/claude-code --no-audit
%s    usermod -aG docker vagrant
    chown -R vagrant:vagrant /vagrant
    echo "AcceptEnv *" >> /etc/ssh/sshd_config
    systemctl restart sshd
  SHELL
%s
end
`,
		box,
		hostname,
		syncType,
		portForwards.String(),
		memory,
		cpus,
		packagesStr,
		npmSection.String(),
		provisionScriptSection,
	)

	return template
}
