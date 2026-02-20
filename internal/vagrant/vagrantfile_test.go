package vagrant

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/asheshgoplani/agent-deck/internal/session"
)

// TestResolvedPackagesBase tests base package list computation
func TestResolvedPackagesBase(t *testing.T) {
	m := &Manager{
		projectPath: "/tmp/test",
		settings:    session.VagrantSettings{},
	}

	packages := m.resolvedPackages()

	expectedBase := []string{
		"build-essential",
		"curl",
		"docker.io",
		"git",
		"nodejs",
		"npm",
		"unzip",
	}

	if len(packages) != len(expectedBase) {
		t.Fatalf("Expected %d base packages, got %d", len(expectedBase), len(packages))
	}

	for i, pkg := range expectedBase {
		if packages[i] != pkg {
			t.Errorf("Expected package[%d] = %s, got %s", i, pkg, packages[i])
		}
	}
}

// TestResolvedPackagesExclude tests package exclusion
func TestResolvedPackagesExclude(t *testing.T) {
	m := &Manager{
		projectPath: "/tmp/test",
		settings: session.VagrantSettings{
			ProvisionPkgExclude: []string{"nodejs", "npm"},
		},
	}

	packages := m.resolvedPackages()

	// nodejs and npm should be removed
	for _, pkg := range packages {
		if pkg == "nodejs" || pkg == "npm" {
			t.Errorf("Package %s should have been excluded", pkg)
		}
	}

	// docker.io should still be present
	found := false
	for _, pkg := range packages {
		if pkg == "docker.io" {
			found = true
			break
		}
	}
	if !found {
		t.Error("docker.io should still be present after exclusion")
	}
}

// TestResolvedPackagesAppend tests appending custom packages
func TestResolvedPackagesAppend(t *testing.T) {
	m := &Manager{
		projectPath: "/tmp/test",
		settings: session.VagrantSettings{
			ProvisionPackages: []string{"vim", "htop", "jq"},
		},
	}

	packages := m.resolvedPackages()

	// Check that custom packages are present
	expectedCustom := []string{"htop", "jq", "vim"}
	for _, custom := range expectedCustom {
		found := false
		for _, pkg := range packages {
			if pkg == custom {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Custom package %s not found in resolved packages", custom)
		}
	}
}

// TestResolvedPackagesDedupe tests deduplication
func TestResolvedPackagesDedupe(t *testing.T) {
	m := &Manager{
		projectPath: "/tmp/test",
		settings: session.VagrantSettings{
			ProvisionPackages: []string{"git", "curl", "git"}, // git appears in base + twice in custom
		},
	}

	packages := m.resolvedPackages()

	// Count occurrences of git
	count := 0
	for _, pkg := range packages {
		if pkg == "git" {
			count++
		}
	}

	if count != 1 {
		t.Errorf("Expected git to appear once, appeared %d times", count)
	}
}

// TestSanitizeHostnameBasic tests basic hostname sanitization
func TestSanitizeHostnameBasic(t *testing.T) {
	m := &Manager{projectPath: "/tmp/test"}

	tests := []struct {
		input    string
		expected string
	}{
		{"my-project", "agentdeck-my-project"},
		{"MyProject", "agentdeck-myproject"},
		{"my_project", "agentdeck-my-project"},
		{"my project", "agentdeck-my-project"},
		{"123project", "agentdeck-123project"},
	}

	for _, tt := range tests {
		result := m.sanitizeHostname(tt.input)
		if result != tt.expected {
			t.Errorf("sanitizeHostname(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

// TestSanitizeHostnameSpecialChars tests special character handling
func TestSanitizeHostnameSpecialChars(t *testing.T) {
	m := &Manager{projectPath: "/tmp/test"}

	tests := []struct {
		input    string
		expected string
	}{
		{"my@project", "agentdeck-my-project"},
		{"my#project!", "agentdeck-my-project"},
		{"my...project", "agentdeck-my-project"},
		{"---project---", "agentdeck-project"},
		{"-project-", "agentdeck-project"},
	}

	for _, tt := range tests {
		result := m.sanitizeHostname(tt.input)
		if result != tt.expected {
			t.Errorf("sanitizeHostname(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

// TestSanitizeHostnameTruncation tests hostname truncation to 63 chars
func TestSanitizeHostnameTruncation(t *testing.T) {
	m := &Manager{projectPath: "/tmp/test"}

	// Create a very long project name
	longName := strings.Repeat("abcd", 30) // 120 chars
	result := m.sanitizeHostname(longName)

	if len(result) > 63 {
		t.Errorf("Hostname length %d exceeds RFC 1123 limit of 63 chars", len(result))
	}

	if !strings.HasPrefix(result, "agentdeck-") {
		t.Errorf("Hostname should start with 'agentdeck-', got %q", result)
	}
}

// TestEnsureVagrantfileGeneration tests Vagrantfile creation from template
func TestEnsureVagrantfileGeneration(t *testing.T) {
	tmpDir := t.TempDir()

	m := &Manager{
		projectPath: tmpDir,
		settings: session.VagrantSettings{
			Box:              "bento/ubuntu-24.04",
			MemoryMB:         4096,
			CPUs:             2,
			SyncedFolderType: "virtualbox",
		},
	}

	err := m.EnsureVagrantfile()
	if err != nil {
		t.Fatalf("EnsureVagrantfile() failed: %v", err)
	}

	// Check that Vagrantfile was created
	vagrantfilePath := filepath.Join(tmpDir, "Vagrantfile")
	content, err := os.ReadFile(vagrantfilePath)
	if err != nil {
		t.Fatalf("Failed to read generated Vagrantfile: %v", err)
	}

	contentStr := string(content)

	// Verify key content is present
	expectedStrings := []string{
		`config.vm.box = "bento/ubuntu-24.04"`,
		`vb.memory = "4096"`,
		`vb.cpus = 2`,
		`type: "virtualbox"`,
		`docker.io`,
		`git`,
		`@anthropic-ai/claude-code`,
		`AcceptEnv *`,
		`config.ssh.forward_agent = true`,
		`vb.gui = false`,
		`--audio`, `none`,
		`--usb`, `off`,
	}

	// Nested hardware virtualization is only enabled on x86_64
	if runtime.GOARCH != "arm64" {
		expectedStrings = append(expectedStrings, `--nested-hw-virt`, `on`)
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(contentStr, expected) {
			t.Errorf("Generated Vagrantfile missing expected content: %q", expected)
		}
	}
}

// TestEnsureVagrantfileExistingNotOverwritten tests that existing Vagrantfile is not overwritten
func TestEnsureVagrantfileExistingNotOverwritten(t *testing.T) {
	tmpDir := t.TempDir()

	existingContent := "# My custom Vagrantfile\n"
	vagrantfilePath := filepath.Join(tmpDir, "Vagrantfile")
	err := os.WriteFile(vagrantfilePath, []byte(existingContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create existing Vagrantfile: %v", err)
	}

	m := &Manager{
		projectPath: tmpDir,
		settings:    session.VagrantSettings{},
	}

	err = m.EnsureVagrantfile()
	if err != nil {
		t.Fatalf("EnsureVagrantfile() failed: %v", err)
	}

	// Verify content was not overwritten
	content, err := os.ReadFile(vagrantfilePath)
	if err != nil {
		t.Fatalf("Failed to read Vagrantfile: %v", err)
	}

	if string(content) != existingContent {
		t.Error("Existing Vagrantfile was overwritten, should have been skipped")
	}
}

// TestEnsureVagrantfileCustomTemplate tests copying custom Vagrantfile template
func TestEnsureVagrantfileCustomTemplate(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a custom Vagrantfile template
	customContent := "# Custom Vagrantfile Template\nVagrant.configure(\"2\") do |config|\nend\n"
	customPath := filepath.Join(tmpDir, "custom.Vagrantfile")
	err := os.WriteFile(customPath, []byte(customContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create custom Vagrantfile: %v", err)
	}

	projectDir := filepath.Join(tmpDir, "project")
	err = os.Mkdir(projectDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}

	m := &Manager{
		projectPath: projectDir,
		settings: session.VagrantSettings{
			Vagrantfile: customPath,
		},
	}

	err = m.EnsureVagrantfile()
	if err != nil {
		t.Fatalf("EnsureVagrantfile() failed: %v", err)
	}

	// Verify custom content was copied
	vagrantfilePath := filepath.Join(projectDir, "Vagrantfile")
	content, err := os.ReadFile(vagrantfilePath)
	if err != nil {
		t.Fatalf("Failed to read Vagrantfile: %v", err)
	}

	if string(content) != customContent {
		t.Errorf("Custom Vagrantfile content not copied correctly.\nExpected: %s\nGot: %s", customContent, string(content))
	}
}

// TestEnsureVagrantfileWithPortForwards tests port forward generation
func TestEnsureVagrantfileWithPortForwards(t *testing.T) {
	tmpDir := t.TempDir()

	m := &Manager{
		projectPath: tmpDir,
		settings: session.VagrantSettings{
			Box:              "bento/ubuntu-24.04",
			MemoryMB:         4096,
			CPUs:             2,
			SyncedFolderType: "virtualbox",
			PortForwards: []session.PortForward{
				{Guest: 3000, Host: 3000, Protocol: "tcp"},
				{Guest: 8080, Host: 8080, Protocol: "tcp"},
			},
		},
	}

	err := m.EnsureVagrantfile()
	if err != nil {
		t.Fatalf("EnsureVagrantfile() failed: %v", err)
	}

	vagrantfilePath := filepath.Join(tmpDir, "Vagrantfile")
	content, err := os.ReadFile(vagrantfilePath)
	if err != nil {
		t.Fatalf("Failed to read Vagrantfile: %v", err)
	}

	contentStr := string(content)

	// Verify port forwards are present
	expectedLines := []string{
		`config.vm.network "forwarded_port", guest: 3000, host: 3000, protocol: "tcp", auto_correct: true`,
		`config.vm.network "forwarded_port", guest: 8080, host: 8080, protocol: "tcp", auto_correct: true`,
	}

	for _, expected := range expectedLines {
		if !strings.Contains(contentStr, expected) {
			t.Errorf("Port forward line missing: %q", expected)
		}
	}
}

// TestEnsureVagrantfileWithProvisionScript tests custom provision script inclusion
func TestEnsureVagrantfileWithProvisionScript(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a custom provision script
	provisionScript := filepath.Join(tmpDir, "provision.sh")
	err := os.WriteFile(provisionScript, []byte("#!/bin/bash\necho 'custom provision'\n"), 0755)
	if err != nil {
		t.Fatalf("Failed to create provision script: %v", err)
	}

	m := &Manager{
		projectPath: tmpDir,
		settings: session.VagrantSettings{
			Box:              "bento/ubuntu-24.04",
			MemoryMB:         4096,
			CPUs:             2,
			SyncedFolderType: "virtualbox",
			ProvisionScript:  provisionScript,
		},
	}

	err = m.EnsureVagrantfile()
	if err != nil {
		t.Fatalf("EnsureVagrantfile() failed: %v", err)
	}

	vagrantfilePath := filepath.Join(tmpDir, "Vagrantfile")
	content, err := os.ReadFile(vagrantfilePath)
	if err != nil {
		t.Fatalf("Failed to read Vagrantfile: %v", err)
	}

	contentStr := string(content)

	// Verify provision script line is present
	expectedLine := `config.vm.provision "shell", path: "` + provisionScript + `"`
	if !strings.Contains(contentStr, expectedLine) {
		t.Errorf("Provision script line missing: %q", expectedLine)
	}
}

// TestEnsureVagrantfileWithNpmPackages tests npm package installation
func TestEnsureVagrantfileWithNpmPackages(t *testing.T) {
	tmpDir := t.TempDir()

	m := &Manager{
		projectPath: tmpDir,
		settings: session.VagrantSettings{
			Box:              "bento/ubuntu-24.04",
			MemoryMB:         4096,
			CPUs:             2,
			SyncedFolderType: "virtualbox",
			NpmPackages:      []string{"typescript", "eslint"},
		},
	}

	err := m.EnsureVagrantfile()
	if err != nil {
		t.Fatalf("EnsureVagrantfile() failed: %v", err)
	}

	vagrantfilePath := filepath.Join(tmpDir, "Vagrantfile")
	content, err := os.ReadFile(vagrantfilePath)
	if err != nil {
		t.Fatalf("Failed to read Vagrantfile: %v", err)
	}

	contentStr := string(content)

	// Verify npm install line for custom packages
	if !strings.Contains(contentStr, "npm install -g") {
		t.Error("npm install line missing for custom packages")
	}

	// Check for specific packages
	if !strings.Contains(contentStr, "typescript") {
		t.Error("typescript package missing from npm install")
	}
	if !strings.Contains(contentStr, "eslint") {
		t.Error("eslint package missing from npm install")
	}
}
