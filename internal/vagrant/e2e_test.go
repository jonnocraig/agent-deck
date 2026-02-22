package vagrant

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/asheshgoplani/agent-deck/internal/session"
)

// skipIfNoVagrant skips the test if Vagrant or VirtualBox are not available.
func skipIfNoVagrant(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("vagrant"); err != nil {
		t.Skip("vagrant not available")
	}
	if _, err := exec.LookPath("VBoxManage"); err != nil {
		t.Skip("VirtualBox not available")
	}
}

// TestE2EVagrantLifecycle performs a full end-to-end test of the vagrant VM lifecycle:
// preflight → vagrantfile → skill + hook injection → boot → SSH verify → destroy.
//
// Requires: Vagrant 2.4+, VirtualBox 7.0+, bento/ubuntu-24.04 box cached.
// Runtime: ~60s with cached box, ~5min without.
// Run: go test -v -run TestE2EVagrantLifecycle -timeout 5m ./internal/vagrant/
func TestE2EVagrantLifecycle(t *testing.T) {
	skipIfNoVagrant(t)

	if testing.Short() {
		t.Skip("skipping e2e vagrant test in short mode")
	}

	// Create an isolated temp directory as the project root
	projectDir := t.TempDir()

	settings := session.VagrantSettings{
		MemoryMB: 1024,
		CPUs:     1,
		Box:      "bento/ubuntu-24.04",
	}

	mgr := NewManager(projectDir, settings)

	// --- Phase 1: Preflight ---
	t.Run("preflight", func(t *testing.T) {
		if err := mgr.PreflightCheck(); err != nil {
			t.Fatalf("PreflightCheck failed: %v", err)
		}
	})

	// --- Phase 2: Vagrantfile generation ---
	t.Run("vagrantfile_generation", func(t *testing.T) {
		if err := mgr.EnsureVagrantfile(); err != nil {
			t.Fatalf("EnsureVagrantfile failed: %v", err)
		}

		vfPath := filepath.Join(projectDir, "Vagrantfile")
		data, err := os.ReadFile(vfPath)
		if err != nil {
			t.Fatalf("Vagrantfile not created: %v", err)
		}

		content := string(data)
		if !strings.Contains(content, "bento/ubuntu-24.04") {
			t.Error("Vagrantfile missing box reference")
		}
		if !strings.Contains(content, "vb.memory") {
			t.Error("Vagrantfile missing memory config")
		}
	})

	// --- Phase 3: Skill + credential guard hook injection ---
	t.Run("skill_and_hook_injection", func(t *testing.T) {
		if err := mgr.EnsureSudoSkill(); err != nil {
			t.Fatalf("EnsureSudoSkill failed: %v", err)
		}

		// Verify skill file with frontmatter
		skillPath := filepath.Join(projectDir, ".claude", "skills", "operating-in-vagrant.md")
		skillData, err := os.ReadFile(skillPath)
		if err != nil {
			t.Fatalf("Skill file not created: %v", err)
		}
		skillContent := string(skillData)

		if !strings.HasPrefix(skillContent, "---\n") {
			t.Error("Skill missing YAML frontmatter")
		}
		if !strings.Contains(skillContent, "name: operating-in-vagrant") {
			t.Error("Skill frontmatter missing name field")
		}
		if !strings.Contains(skillContent, "Ubuntu 24.04") {
			t.Error("Skill missing Ubuntu version reference")
		}
		if !strings.Contains(skillContent, "inotify") {
			t.Error("Skill missing file watcher warning")
		}

		// Verify credential guard hook in settings.local.json
		settingsPath := filepath.Join(projectDir, ".claude", "settings.local.json")
		settingsData, err := os.ReadFile(settingsPath)
		if err != nil {
			t.Fatalf("settings.local.json not created: %v", err)
		}

		var settingsJSON map[string]interface{}
		if err := json.Unmarshal(settingsData, &settingsJSON); err != nil {
			t.Fatalf("Failed to parse settings.local.json: %v", err)
		}

		hooks, ok := settingsJSON["hooks"].(map[string]interface{})
		if !ok {
			t.Fatal("settings.local.json missing 'hooks' key")
		}

		preToolUse, ok := hooks["PreToolUse"].([]interface{})
		if !ok || len(preToolUse) == 0 {
			t.Fatal("Missing PreToolUse hook array")
		}

		hookEntry, ok := preToolUse[0].(map[string]interface{})
		if !ok {
			t.Fatal("PreToolUse entry is not a map")
		}

		matcher := hookEntry["matcher"].(string)
		if matcher != "Read|View|Cat" {
			t.Errorf("Expected matcher 'Read|View|Cat', got %q", matcher)
		}

		innerHooks, ok := hookEntry["hooks"].([]interface{})
		if !ok || len(innerHooks) == 0 {
			t.Fatal("PreToolUse hook missing inner hooks array")
		}
		innerHook := innerHooks[0].(map[string]interface{})
		command := innerHook["command"].(string)
		if !strings.Contains(command, "BLOCKED") {
			t.Error("Credential guard hook command missing BLOCKED message")
		}
	})

	// --- Phase 4: Boot VM ---
	t.Run("boot_vm", func(t *testing.T) {
		var phases []BootPhase
		err := mgr.EnsureRunning(func(phase BootPhase) {
			phases = append(phases, phase)
			t.Logf("Boot phase: %s", phase)
		})
		if err != nil {
			// vagrant up may return non-zero exit from provisioning errors
			// (e.g., transient npm failures) while the VM itself boots fine.
			// Check if VM is actually running despite the error.
			status, statusErr := mgr.Status()
			if statusErr != nil || status != "running" {
				t.Fatalf("EnsureRunning failed and VM is not running (status=%q): %v", status, err)
			}
			t.Logf("EnsureRunning returned error but VM is running (provisioning issue): %v", err)
		}

		if len(phases) == 0 {
			t.Error("No boot phases reported")
		}
	})

	// --- Phase 5: Verify VM is running ---
	t.Run("verify_running", func(t *testing.T) {
		status, err := mgr.Status()
		if err != nil {
			t.Fatalf("Status check failed: %v", err)
		}
		if status != "running" {
			t.Errorf("Expected VM status 'running', got %q", status)
		}
	})

	// --- Phase 6: Verify SSH works ---
	t.Run("ssh_connectivity", func(t *testing.T) {
		cmd := exec.Command("vagrant", "ssh", "-c", "echo AGENTDECK_E2E_OK")
		cmd.Dir = projectDir
		output, err := cmd.Output()
		if err != nil {
			t.Fatalf("SSH into VM failed: %v", err)
		}
		if !strings.Contains(string(output), "AGENTDECK_E2E_OK") {
			t.Errorf("SSH command output unexpected: %s", output)
		}
	})

	// --- Phase 7: Verify project mount ---
	t.Run("project_mount", func(t *testing.T) {
		// Write a marker file in the project dir and check it's visible inside VM
		markerPath := filepath.Join(projectDir, ".e2e-marker")
		if err := os.WriteFile(markerPath, []byte("e2e-test"), 0644); err != nil {
			t.Fatalf("Failed to write marker file: %v", err)
		}

		cmd := exec.Command("vagrant", "ssh", "-c", "cat /vagrant/.e2e-marker")
		cmd.Dir = projectDir
		output, err := cmd.Output()
		if err != nil {
			t.Fatalf("Failed to read marker file in VM: %v", err)
		}
		if strings.TrimSpace(string(output)) != "e2e-test" {
			t.Errorf("Marker file content mismatch: %q", output)
		}
	})

	// --- Phase 8: Command wrapping ---
	t.Run("command_wrapping", func(t *testing.T) {
		wrapped := mgr.WrapCommand("claude --resume test123", []string{"ANTHROPIC_API_KEY"}, []int{8080})

		if !strings.Contains(wrapped, "vagrant ssh") {
			t.Error("Wrapped command missing 'vagrant ssh'")
		}
		if !strings.Contains(wrapped, "-R 8080:localhost:8080") {
			t.Error("Wrapped command missing reverse tunnel")
		}
		if !strings.Contains(wrapped, "SendEnv=ANTHROPIC_API_KEY") {
			t.Error("Wrapped command missing env forwarding")
		}
		if !strings.Contains(wrapped, "claude --resume test123") {
			t.Error("Wrapped command missing original command")
		}
	})

	// --- Phase 9: Destroy VM (cleanup) ---
	t.Run("destroy_vm", func(t *testing.T) {
		if err := mgr.Destroy(); err != nil {
			t.Fatalf("Destroy failed: %v", err)
		}

		status, err := mgr.Status()
		if err != nil {
			// After destroy, status may error or return "not_created"
			t.Logf("Status after destroy (expected): %v", err)
			return
		}
		if status == "running" {
			t.Error("VM still running after destroy")
		}
	})
}
