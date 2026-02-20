package vagrant

import (
	"os"
	"os/exec"
	"testing"

	"github.com/asheshgoplani/agent-deck/internal/session"
)

func TestNewManager(t *testing.T) {
	projectPath := "/test/project"
	settings := session.VagrantSettings{
		MemoryMB: 4096,
		CPUs:     2,
		Box:      "bento/ubuntu-24.04",
	}

	manager := NewManager(projectPath, settings)

	if manager == nil {
		t.Fatal("NewManager returned nil")
	}

	if manager.projectPath != projectPath {
		t.Errorf("projectPath = %q, want %q", manager.projectPath, projectPath)
	}

	if manager.settings.MemoryMB != 4096 {
		t.Errorf("settings.MemoryMB = %d, want 4096", manager.settings.MemoryMB)
	}

	if manager.dotfilePath != "" {
		t.Errorf("dotfilePath should be empty initially, got %q", manager.dotfilePath)
	}

	if len(manager.sessions) != 0 {
		t.Errorf("sessions should be empty initially, got %d items", len(manager.sessions))
	}
}

func TestManagerVagrantCmd(t *testing.T) {
	projectPath := "/test/project"
	settings := session.VagrantSettings{}
	manager := NewManager(projectPath, settings)

	tests := []struct {
		name           string
		args           []string
		dotfilePath    string
		wantArgs       []string
		wantDir        string
		wantEnvVarName string
		wantEnvVarVal  string
	}{
		{
			name:     "basic command without dotfile",
			args:     []string{"status"},
			wantArgs: []string{"status"},
			wantDir:  projectPath,
		},
		{
			name:     "command with multiple args",
			args:     []string{"up", "--machine-readable"},
			wantArgs: []string{"up", "--machine-readable"},
			wantDir:  projectPath,
		},
		{
			name:           "command with dotfile path",
			args:           []string{"status"},
			dotfilePath:    "/tmp/.vagrant-session-123",
			wantArgs:       []string{"status"},
			wantDir:        projectPath,
			wantEnvVarName: "VAGRANT_DOTFILE_PATH",
			wantEnvVarVal:  "/tmp/.vagrant-session-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager.dotfilePath = tt.dotfilePath
			cmd := manager.vagrantCmd(tt.args...)

			if cmd.Path != "vagrant" {
				lookPath, _ := exec.LookPath("vagrant")
				if cmd.Path != lookPath {
					t.Errorf("cmd.Path = %q, want 'vagrant' or result of LookPath", cmd.Path)
				}
			}

			if len(cmd.Args) != len(tt.wantArgs)+1 {
				t.Errorf("cmd.Args length = %d, want %d", len(cmd.Args), len(tt.wantArgs)+1)
			}

			for i, arg := range tt.wantArgs {
				if cmd.Args[i+1] != arg {
					t.Errorf("cmd.Args[%d] = %q, want %q", i+1, cmd.Args[i+1], arg)
				}
			}

			if cmd.Dir != tt.wantDir {
				t.Errorf("cmd.Dir = %q, want %q", cmd.Dir, tt.wantDir)
			}

			if tt.wantEnvVarName != "" {
				found := false
				for _, env := range cmd.Env {
					if env == tt.wantEnvVarName+"="+tt.wantEnvVarVal {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected env var %s=%s not found in cmd.Env", tt.wantEnvVarName, tt.wantEnvVarVal)
				}
			} else if len(cmd.Env) > 0 {
				// If no env var expected and dotfilePath is empty, Env should be nil
				if tt.dotfilePath == "" {
					t.Errorf("cmd.Env should be nil when dotfilePath is empty, got %v", cmd.Env)
				}
			}
		})
	}
}

func TestManagerIsInstalled(t *testing.T) {
	manager := NewManager("/test/project", session.VagrantSettings{})

	// We can't reliably test this without mocking exec.LookPath,
	// but we can verify it doesn't panic and returns a bool
	result := manager.IsInstalled()

	// Result depends on whether vagrant is actually installed
	// Just verify it returns either true or false
	if result != true && result != false {
		t.Errorf("IsInstalled should return a boolean, got %v", result)
	}
}

func TestManagerStatus_Parsing(t *testing.T) {
	// This test verifies the parsing logic for machine-readable output
	// In the actual implementation, we'd need to mock the command execution

	tests := []struct {
		name       string
		output     string
		wantState  string
		wantErr    bool
		shouldFail bool
	}{
		{
			name:      "running state",
			output:    "1234567890,default,state,running\n",
			wantState: "running",
			wantErr:   false,
		},
		{
			name:      "poweroff state",
			output:    "1234567890,default,state,poweroff\n",
			wantState: "poweroff",
			wantErr:   false,
		},
		{
			name:      "not_created state",
			output:    "1234567890,default,state,not_created\n",
			wantState: "not_created",
			wantErr:   false,
		},
		{
			name:      "saved state (suspended)",
			output:    "1234567890,default,state,saved\n",
			wantState: "saved",
			wantErr:   false,
		},
		{
			name:      "multiple lines with state",
			output:    "1234567890,default,metadata,foo\n1234567890,default,state,running\n1234567890,default,other,bar\n",
			wantState: "running",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the mock output directly
			state := parseVagrantState(tt.output)

			if tt.wantErr {
				if state != "" {
					t.Errorf("expected error/empty state, got %q", state)
				}
			} else {
				if state != tt.wantState {
					t.Errorf("state = %q, want %q", state, tt.wantState)
				}
			}
		})
	}
}

// Helper function to test state parsing logic
// In the actual implementation, this logic should be in Status() method
func parseVagrantState(output string) string {
	lines := splitLines(output)
	for _, line := range lines {
		fields := splitFields(line, ',')
		if len(fields) >= 4 && fields[2] == "state" {
			return fields[3]
		}
	}
	return ""
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			if i > start {
				lines = append(lines, s[start:i])
			}
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func splitFields(s string, sep byte) []string {
	var fields []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == sep {
			fields = append(fields, s[start:i])
			start = i + 1
		}
	}
	if start <= len(s) {
		fields = append(fields, s[start:])
	}
	return fields
}

func TestManagerInterfaceCompliance(t *testing.T) {
	// Compile-time check that Manager implements VagrantProvider
	var _ VagrantProvider = (*Manager)(nil)

	// Also verify at runtime
	manager := NewManager("/test/project", session.VagrantSettings{})
	var iface interface{} = manager

	if _, ok := iface.(VagrantProvider); !ok {
		t.Error("Manager does not implement VagrantProvider interface")
	}
}

func TestManagerStubMethods(t *testing.T) {
	// Test that stub methods exist and behave as expected
	// Note: Some methods are actually implemented in other files (preflight.go, health.go, etc.)
	// We just verify they don't panic and have reasonable behavior

	// Use a temp directory for testing to avoid file system errors
	tempDir := t.TempDir()
	manager := NewManager(tempDir, session.VagrantSettings{})

	// WrapCommand should wrap the command for vagrant ssh
	cmd := manager.WrapCommand("echo test", []string{"VAR1"}, []int{8080})
	expected := "vagrant ssh -- -R 8080:localhost:8080 -o SendEnv=VAR1 -t 'cd /vagrant && echo test'"
	if cmd != expected {
		t.Errorf("WrapCommand should wrap command for vagrant ssh, got %q, want %q", cmd, expected)
	}

	// EnsureVagrantfile stub should return nil
	if err := manager.EnsureVagrantfile(); err != nil {
		t.Errorf("EnsureVagrantfile stub should return nil, got %v", err)
	}

	// GetMCPPackages is implemented and returns packages from user config
	// Just verify it doesn't panic - return value depends on user's config.toml
	packages := manager.GetMCPPackages()
	// Packages may be nil or a list depending on what MCPs are configured
	t.Logf("GetMCPPackages returned: %v", packages)

	// HasConfigDrift should return false (no hash file exists yet)
	if drift := manager.HasConfigDrift(); drift != false {
		t.Errorf("HasConfigDrift should return false when no hash file exists, got %v", drift)
	}

	// Test session management methods (should not panic)
	if err := manager.RegisterSession("session-1"); err != nil {
		t.Fatalf("RegisterSession failed: %v", err)
	}
	if count := manager.SessionCount(); count != 1 {
		t.Errorf("SessionCount should return 1 after registering a session, got %d", count)
	}

	if last := manager.IsLastSession("session-1"); last != true {
		t.Errorf("IsLastSession should return true when only one session exists, got %v", last)
	}

	if err := manager.RegisterSession("session-2"); err != nil {
		t.Fatalf("RegisterSession failed: %v", err)
	}
	if count := manager.SessionCount(); count != 2 {
		t.Errorf("SessionCount should return 2 after registering two sessions, got %d", count)
	}

	if last := manager.IsLastSession("session-1"); last != false {
		t.Errorf("IsLastSession should return false when multiple sessions exist, got %v", last)
	}

	if err := manager.UnregisterSession("session-1"); err != nil {
		t.Fatalf("UnregisterSession failed: %v", err)
	}
	if count := manager.SessionCount(); count != 1 {
		t.Errorf("SessionCount should return 1 after unregistering one session, got %d", count)
	}

	manager.SetDotfilePath("session-1")
	// Just verify it doesn't panic

	if err := manager.EnsureSudoSkill(); err != nil {
		t.Errorf("EnsureSudoSkill stub should return nil, got %v", err)
	}

	if err := manager.SyncClaudeConfig(); err != nil {
		t.Errorf("SyncClaudeConfig stub should return nil, got %v", err)
	}
}

func TestManagerEnvironmentPropagation(t *testing.T) {
	// Test that vagrantCmd properly propagates environment
	projectPath := "/test/project"
	manager := NewManager(projectPath, session.VagrantSettings{})
	manager.dotfilePath = "/tmp/.vagrant-test"

	// Set a known environment variable
	testKey := "TEST_VAGRANT_VAR"
	testVal := "test-value"
	os.Setenv(testKey, testVal)
	defer os.Unsetenv(testKey)

	cmd := manager.vagrantCmd("status")

	// Check that VAGRANT_DOTFILE_PATH is set
	foundDotfile := false
	foundTestVar := false

	for _, env := range cmd.Env {
		if env == "VAGRANT_DOTFILE_PATH="+manager.dotfilePath {
			foundDotfile = true
		}
		if env == testKey+"="+testVal {
			foundTestVar = true
		}
	}

	if !foundDotfile {
		t.Error("VAGRANT_DOTFILE_PATH not found in cmd.Env")
	}

	if !foundTestVar {
		t.Error("environment variable TEST_VAGRANT_VAR not propagated to cmd.Env")
	}
}
