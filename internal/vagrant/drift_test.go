package vagrant

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/asheshgoplani/agent-deck/internal/session"
)

// TestConfigHashDeterministic verifies that identical settings produce identical hashes
func TestConfigHashDeterministic(t *testing.T) {
	settings := session.VagrantSettings{
		Box:             "bento/ubuntu-24.04",
		ProvisionPackages: []string{"curl", "git"},
		NpmPackages:     []string{"@modelcontextprotocol/server-memory"},
	}

	m1 := &Manager{
		projectPath: "/tmp/test1",
		settings:    settings,
	}

	m2 := &Manager{
		projectPath: "/tmp/test2",
		settings:    settings,
	}

	hash1 := m1.configHash()
	hash2 := m2.configHash()

	if hash1 != hash2 {
		t.Errorf("same settings produced different hashes: %s != %s", hash1, hash2)
	}
}

// TestConfigHashChangesOnPackageAdd verifies hash changes when packages are added
func TestConfigHashChangesOnPackageAdd(t *testing.T) {
	settings1 := session.VagrantSettings{
		Box:             "bento/ubuntu-24.04",
		ProvisionPackages: []string{"curl"},
		NpmPackages:     []string{},
	}

	settings2 := session.VagrantSettings{
		Box:             "bento/ubuntu-24.04",
		ProvisionPackages: []string{"curl", "git"},
		NpmPackages:     []string{},
	}

	m1 := &Manager{
		projectPath: "/tmp/test",
		settings:    settings1,
	}

	m2 := &Manager{
		projectPath: "/tmp/test",
		settings:    settings2,
	}

	hash1 := m1.configHash()
	hash2 := m2.configHash()

	if hash1 == hash2 {
		t.Errorf("different packages should produce different hashes, but both are %s", hash1)
	}
}

// TestConfigHashChangesOnBoxChange verifies hash changes when box is changed
func TestConfigHashChangesOnBoxChange(t *testing.T) {
	settings1 := session.VagrantSettings{
		Box:             "bento/ubuntu-24.04",
		ProvisionPackages: []string{},
		NpmPackages:     []string{},
	}

	settings2 := session.VagrantSettings{
		Box:             "bento/ubuntu-22.04",
		ProvisionPackages: []string{},
		NpmPackages:     []string{},
	}

	m1 := &Manager{
		projectPath: "/tmp/test",
		settings:    settings1,
	}

	m2 := &Manager{
		projectPath: "/tmp/test",
		settings:    settings2,
	}

	hash1 := m1.configHash()
	hash2 := m2.configHash()

	if hash1 == hash2 {
		t.Errorf("different boxes should produce different hashes, but both are %s", hash1)
	}
}

// TestConfigHashChangesOnProvisionScript verifies hash changes when provision script content changes
func TestConfigHashChangesOnProvisionScript(t *testing.T) {
	tmpDir := t.TempDir()

	// Create first script
	script1 := filepath.Join(tmpDir, "script1.sh")
	if err := os.WriteFile(script1, []byte("#!/bin/bash\necho hello"), 0o644); err != nil {
		t.Fatalf("failed to write script1: %v", err)
	}

	// Create second script
	script2 := filepath.Join(tmpDir, "script2.sh")
	if err := os.WriteFile(script2, []byte("#!/bin/bash\necho world"), 0o644); err != nil {
		t.Fatalf("failed to write script2: %v", err)
	}

	settings1 := session.VagrantSettings{
		Box:               "bento/ubuntu-24.04",
		ProvisionScript:   script1,
		ProvisionPackages: []string{},
		NpmPackages:       []string{},
	}

	settings2 := session.VagrantSettings{
		Box:               "bento/ubuntu-24.04",
		ProvisionScript:   script2,
		ProvisionPackages: []string{},
		NpmPackages:       []string{},
	}

	m1 := &Manager{
		projectPath: tmpDir,
		settings:    settings1,
	}

	m2 := &Manager{
		projectPath: tmpDir,
		settings:    settings2,
	}

	hash1 := m1.configHash()
	hash2 := m2.configHash()

	if hash1 == hash2 {
		t.Errorf("different provision scripts should produce different hashes, but both are %s", hash1)
	}
}

// TestConfigHashIncludesPortForwards verifies hash changes when port forwards change
func TestConfigHashIncludesPortForwards(t *testing.T) {
	settings1 := session.VagrantSettings{
		Box:               "bento/ubuntu-24.04",
		ProvisionPackages: []string{},
		NpmPackages:       []string{},
		PortForwards: []session.PortForward{
			{Guest: 8080, Host: 8080, Protocol: "tcp"},
		},
	}

	settings2 := session.VagrantSettings{
		Box:               "bento/ubuntu-24.04",
		ProvisionPackages: []string{},
		NpmPackages:       []string{},
		PortForwards: []session.PortForward{
			{Guest: 8080, Host: 8080, Protocol: "tcp"},
			{Guest: 3000, Host: 3000, Protocol: "tcp"},
		},
	}

	m1 := &Manager{
		projectPath: "/tmp/test",
		settings:    settings1,
	}

	m2 := &Manager{
		projectPath: "/tmp/test",
		settings:    settings2,
	}

	hash1 := m1.configHash()
	hash2 := m2.configHash()

	if hash1 == hash2 {
		t.Errorf("different port forwards should produce different hashes, but both are %s", hash1)
	}
}

// TestHasConfigDriftFalseOnFirstRun verifies no drift detected when no hash file exists
func TestHasConfigDriftFalseOnFirstRun(t *testing.T) {
	tmpDir := t.TempDir()

	settings := session.VagrantSettings{
		Box:               "bento/ubuntu-24.04",
		ProvisionPackages: []string{},
		NpmPackages:       []string{},
	}

	m := &Manager{
		projectPath: tmpDir,
		settings:    settings,
	}

	if m.HasConfigDrift() {
		t.Error("expected no drift on first run (no hash file), but drift was detected")
	}
}

// TestHasConfigDriftDetectsChange verifies drift is detected after hash is written and settings change
func TestHasConfigDriftDetectsChange(t *testing.T) {
	tmpDir := t.TempDir()

	settings1 := session.VagrantSettings{
		Box:               "bento/ubuntu-24.04",
		ProvisionPackages: []string{"curl"},
		NpmPackages:       []string{},
	}

	m := &Manager{
		projectPath: tmpDir,
		settings:    settings1,
	}

	// Write initial hash
	if err := m.WriteConfigHash(); err != nil {
		t.Fatalf("failed to write initial hash: %v", err)
	}

	// Verify no drift
	if m.HasConfigDrift() {
		t.Error("expected no drift after writing hash with same settings")
	}

	// Change settings
	m.settings.ProvisionPackages = []string{"curl", "git"}

	// Verify drift is detected
	if !m.HasConfigDrift() {
		t.Error("expected drift to be detected after changing settings")
	}
}

// TestWriteConfigHashCreatesFile verifies hash file is created
func TestWriteConfigHashCreatesFile(t *testing.T) {
	tmpDir := t.TempDir()

	settings := session.VagrantSettings{
		Box:               "bento/ubuntu-24.04",
		ProvisionPackages: []string{},
		NpmPackages:       []string{},
	}

	m := &Manager{
		projectPath: tmpDir,
		settings:    settings,
	}

	if err := m.WriteConfigHash(); err != nil {
		t.Fatalf("failed to write config hash: %v", err)
	}

	hashFile := filepath.Join(tmpDir, ".vagrant", "agent-deck-config.sha256")
	if _, err := os.Stat(hashFile); err != nil {
		t.Errorf("hash file not created at %s: %v", hashFile, err)
	}
}

// TestWriteConfigHashCreatesVagrantDir verifies .vagrant directory is created
func TestWriteConfigHashCreatesVagrantDir(t *testing.T) {
	tmpDir := t.TempDir()

	settings := session.VagrantSettings{
		Box:               "bento/ubuntu-24.04",
		ProvisionPackages: []string{},
		NpmPackages:       []string{},
	}

	m := &Manager{
		projectPath: tmpDir,
		settings:    settings,
	}

	if err := m.WriteConfigHash(); err != nil {
		t.Fatalf("failed to write config hash: %v", err)
	}

	vagrantDir := filepath.Join(tmpDir, ".vagrant")
	info, err := os.Stat(vagrantDir)
	if err != nil {
		t.Errorf("failed to stat .vagrant directory: %v", err)
	}
	if !info.IsDir() {
		t.Errorf(".vagrant is not a directory")
	}
}

// TestConfigHashIncludesNpmPackages verifies npm packages affect the hash
func TestConfigHashIncludesNpmPackages(t *testing.T) {
	settings1 := session.VagrantSettings{
		Box:               "bento/ubuntu-24.04",
		ProvisionPackages: []string{},
		NpmPackages:       []string{"@modelcontextprotocol/server-memory"},
	}

	settings2 := session.VagrantSettings{
		Box:               "bento/ubuntu-24.04",
		ProvisionPackages: []string{},
		NpmPackages:       []string{"@modelcontextprotocol/server-memory", "@modelcontextprotocol/server-filesystem"},
	}

	m1 := &Manager{
		projectPath: "/tmp/test",
		settings:    settings1,
	}

	m2 := &Manager{
		projectPath: "/tmp/test",
		settings:    settings2,
	}

	hash1 := m1.configHash()
	hash2 := m2.configHash()

	if hash1 == hash2 {
		t.Errorf("different npm packages should produce different hashes, but both are %s", hash1)
	}
}

// TestWriteConfigHashFileContent verifies written hash is valid
func TestWriteConfigHashFileContent(t *testing.T) {
	tmpDir := t.TempDir()

	settings := session.VagrantSettings{
		Box:               "bento/ubuntu-24.04",
		ProvisionPackages: []string{},
		NpmPackages:       []string{},
	}

	m := &Manager{
		projectPath: tmpDir,
		settings:    settings,
	}

	expectedHash := m.configHash()

	if err := m.WriteConfigHash(); err != nil {
		t.Fatalf("failed to write config hash: %v", err)
	}

	hashFile := filepath.Join(tmpDir, ".vagrant", "agent-deck-config.sha256")
	content, err := os.ReadFile(hashFile)
	if err != nil {
		t.Fatalf("failed to read hash file: %v", err)
	}

	writtenHash := string(content)
	if writtenHash != expectedHash {
		t.Errorf("written hash mismatch: expected %s, got %s", expectedHash, writtenHash)
	}
}

// TestConfigHashPortForwardProtocol verifies protocol is included in hash
func TestConfigHashPortForwardProtocol(t *testing.T) {
	settings1 := session.VagrantSettings{
		Box:               "bento/ubuntu-24.04",
		ProvisionPackages: []string{},
		NpmPackages:       []string{},
		PortForwards: []session.PortForward{
			{Guest: 8080, Host: 8080, Protocol: "tcp"},
		},
	}

	settings2 := session.VagrantSettings{
		Box:               "bento/ubuntu-24.04",
		ProvisionPackages: []string{},
		NpmPackages:       []string{},
		PortForwards: []session.PortForward{
			{Guest: 8080, Host: 8080, Protocol: "udp"},
		},
	}

	m1 := &Manager{
		projectPath: "/tmp/test",
		settings:    settings1,
	}

	m2 := &Manager{
		projectPath: "/tmp/test",
		settings:    settings2,
	}

	hash1 := m1.configHash()
	hash2 := m2.configHash()

	if hash1 == hash2 {
		t.Errorf("different port forward protocols should produce different hashes, but both are %s", hash1)
	}
}

// TestWriteConfigHashMkdirError tests WriteConfigHash when MkdirAll fails
func TestWriteConfigHashMkdirError(t *testing.T) {
	// Create a file where .vagrant directory should be (blocks MkdirAll)
	tmpDir := t.TempDir()
	blockingFile := filepath.Join(tmpDir, ".vagrant")
	if err := os.WriteFile(blockingFile, []byte("blocker"), 0o644); err != nil {
		t.Fatalf("failed to create blocking file: %v", err)
	}

	settings := session.VagrantSettings{
		Box:               "bento/ubuntu-24.04",
		ProvisionPackages: []string{},
		NpmPackages:       []string{},
	}

	m := &Manager{
		projectPath: tmpDir,
		settings:    settings,
	}

	err := m.WriteConfigHash()
	if err == nil {
		t.Error("expected error when .vagrant is a file, got nil")
	}
}

// TestWriteConfigHashReadonlyDir tests WriteConfigHash when directory is readonly
func TestWriteConfigHashReadonlyDir(t *testing.T) {
	tmpDir := t.TempDir()

	settings := session.VagrantSettings{
		Box:               "bento/ubuntu-24.04",
		ProvisionPackages: []string{},
		NpmPackages:       []string{},
	}

	m := &Manager{
		projectPath: tmpDir,
		settings:    settings,
	}

	// Create .vagrant directory but make it readonly
	vagrantDir := filepath.Join(tmpDir, ".vagrant")
	if err := os.MkdirAll(vagrantDir, 0o755); err != nil {
		t.Fatalf("failed to create .vagrant dir: %v", err)
	}
	if err := os.Chmod(vagrantDir, 0o444); err != nil {
		t.Fatalf("failed to chmod .vagrant dir: %v", err)
	}
	defer os.Chmod(vagrantDir, 0o755) // restore for cleanup

	err := m.WriteConfigHash()
	if err == nil {
		t.Error("expected error when .vagrant directory is readonly, got nil")
	}
}

// TestConfigHashProvisionScriptMissing tests behavior when provision script file is missing
func TestConfigHashProvisionScriptMissing(t *testing.T) {
	tmpDir := t.TempDir()

	settings := session.VagrantSettings{
		Box:               "bento/ubuntu-24.04",
		ProvisionScript:   "/nonexistent/script.sh",
		ProvisionPackages: []string{},
		NpmPackages:       []string{},
	}

	m := &Manager{
		projectPath: tmpDir,
		settings:    settings,
	}

	// Should not panic or error - just continue without hashing script content
	hash := m.configHash()
	if hash == "" {
		t.Error("expected hash to be computed even with missing provision script")
	}
}
