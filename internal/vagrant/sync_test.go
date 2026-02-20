package vagrant

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/asheshgoplani/agent-deck/internal/session"
)

func TestWriteFileToVM_CommandFormat(t *testing.T) {
	projectPath := "/test/project"
	manager := NewManager(projectPath, session.VagrantSettings{})

	tests := []struct {
		name       string
		remotePath string
		content    []byte
		wantSSH    bool
		wantBase64 bool
		wantMkdir  bool
	}{
		{
			name:       "simple file",
			remotePath: "~/.claude.json",
			content:    []byte(`{"test": "value"}`),
			wantSSH:    true,
			wantBase64: true,
			wantMkdir:  true,
		},
		{
			name:       "nested directory",
			remotePath: "~/.claude/.claude.json",
			content:    []byte(`{"mcpServers": {}}`),
			wantSSH:    true,
			wantBase64: true,
			wantMkdir:  true,
		},
		{
			name:       "special characters in content",
			remotePath: "~/test.json",
			content:    []byte(`{"key": "value with 'quotes' and \"escapes\""}`),
			wantSSH:    true,
			wantBase64: true,
			wantMkdir:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create the command (but don't execute it)
			cmd := manager.createWriteFileCmd(tt.remotePath, tt.content)

			// Verify it's a vagrant ssh command
			if tt.wantSSH {
				if len(cmd.Args) < 3 || cmd.Args[1] != "ssh" || cmd.Args[2] != "-c" {
					t.Errorf("expected vagrant ssh -c command, got: %v", cmd.Args)
				}
			}

			// Verify the command string contains expected parts
			if len(cmd.Args) >= 4 {
				cmdStr := cmd.Args[3]

				if tt.wantMkdir && !strings.Contains(cmdStr, "mkdir -p") {
					t.Errorf("command should contain 'mkdir -p', got: %s", cmdStr)
				}

				if tt.wantBase64 {
					if !strings.Contains(cmdStr, "base64 -d") {
						t.Errorf("command should contain 'base64 -d', got: %s", cmdStr)
					}

					// Verify content is base64 encoded in the command
					encoded := base64.StdEncoding.EncodeToString(tt.content)
					if !strings.Contains(cmdStr, encoded) {
						t.Errorf("command should contain base64 encoded content")
					}
				}

				// Verify remote path is in the command
				if !strings.Contains(cmdStr, tt.remotePath) {
					t.Errorf("command should contain remote path %s, got: %s", tt.remotePath, cmdStr)
				}
			}
		})
	}
}

func TestBase64EncodingRoundTrip(t *testing.T) {
	tests := []struct {
		name    string
		content []byte
	}{
		{
			name:    "simple JSON",
			content: []byte(`{"test": "value"}`),
		},
		{
			name:    "JSON with special chars",
			content: []byte(`{"key": "value with 'quotes' and \"escapes\" and $vars"}`),
		},
		{
			name:    "multi-line JSON",
			content: []byte("{\n  \"mcpServers\": {\n    \"test\": {}\n  }\n}"),
		},
		{
			name:    "empty object",
			content: []byte(`{}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encode
			encoded := base64.StdEncoding.EncodeToString(tt.content)

			// Decode
			decoded, err := base64.StdEncoding.DecodeString(encoded)
			if err != nil {
				t.Fatalf("failed to decode: %v", err)
			}

			// Verify round-trip
			if string(decoded) != string(tt.content) {
				t.Errorf("content mismatch after round-trip:\ngot:  %s\nwant: %s", string(decoded), string(tt.content))
			}
		})
	}
}

func TestSyncClaudeConfig_GlobalConfigExists(t *testing.T) {
	// Create a temp directory to simulate CLAUDE_CONFIG_DIR
	tempDir := t.TempDir()
	globalConfigPath := filepath.Join(tempDir, ".claude.json")
	globalContent := []byte(`{"mcpServers": {"test": {}}}`)

	// Write test config
	if err := os.WriteFile(globalConfigPath, globalContent, 0600); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}

	// Set CLAUDE_CONFIG_DIR to our temp directory
	originalEnv := os.Getenv("CLAUDE_CONFIG_DIR")
	os.Setenv("CLAUDE_CONFIG_DIR", tempDir)
	defer func() {
		if originalEnv != "" {
			os.Setenv("CLAUDE_CONFIG_DIR", originalEnv)
		} else {
			os.Unsetenv("CLAUDE_CONFIG_DIR")
		}
	}()

	// Create manager
	projectPath := "/test/project"
	manager := NewManager(projectPath, session.VagrantSettings{})

	// Track calls to writeFileToVM
	var writeCalls []writeFileCall
	manager.writeFileToVMFunc = func(remotePath string, content []byte) error {
		writeCalls = append(writeCalls, writeFileCall{
			remotePath: remotePath,
			content:    content,
		})
		return nil
	}

	// Run SyncClaudeConfig
	err := manager.SyncClaudeConfig()
	if err != nil {
		t.Errorf("SyncClaudeConfig returned error: %v", err)
	}

	// Verify writeFileToVM was called for global config
	found := false
	for _, call := range writeCalls {
		if call.remotePath == "~/.claude/.claude.json" {
			found = true
			if string(call.content) != string(globalContent) {
				t.Errorf("global config content mismatch:\ngot:  %s\nwant: %s", string(call.content), string(globalContent))
			}
		}
	}

	if !found {
		t.Error("writeFileToVM was not called for global config")
	}
}

func TestSyncClaudeConfig_UserConfigExists(t *testing.T) {
	// Create temp directory for user config
	tempHome := t.TempDir()
	userConfigPath := filepath.Join(tempHome, ".claude.json")
	userContent := []byte(`{"projects": {}}`)

	// Write test config
	if err := os.WriteFile(userConfigPath, userContent, 0600); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}

	// Set HOME to our temp directory
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", originalHome)

	// Create manager
	projectPath := "/test/project"
	manager := NewManager(projectPath, session.VagrantSettings{})

	// Track calls to writeFileToVM
	var writeCalls []writeFileCall
	manager.writeFileToVMFunc = func(remotePath string, content []byte) error {
		writeCalls = append(writeCalls, writeFileCall{
			remotePath: remotePath,
			content:    content,
		})
		return nil
	}

	// Run SyncClaudeConfig
	err := manager.SyncClaudeConfig()
	if err != nil {
		t.Errorf("SyncClaudeConfig returned error: %v", err)
	}

	// Verify writeFileToVM was called for user config
	found := false
	for _, call := range writeCalls {
		if call.remotePath == "~/.claude.json" {
			found = true
			if string(call.content) != string(userContent) {
				t.Errorf("user config content mismatch:\ngot:  %s\nwant: %s", string(call.content), string(userContent))
			}
		}
	}

	if !found {
		t.Error("writeFileToVM was not called for user config")
	}
}

func TestSyncClaudeConfig_NoGlobalConfig(t *testing.T) {
	// Create temp directory with no .claude.json
	tempDir := t.TempDir()

	// Set CLAUDE_CONFIG_DIR to our temp directory
	originalEnv := os.Getenv("CLAUDE_CONFIG_DIR")
	os.Setenv("CLAUDE_CONFIG_DIR", tempDir)
	defer func() {
		if originalEnv != "" {
			os.Setenv("CLAUDE_CONFIG_DIR", originalEnv)
		} else {
			os.Unsetenv("CLAUDE_CONFIG_DIR")
		}
	}()

	// Create manager
	projectPath := "/test/project"
	manager := NewManager(projectPath, session.VagrantSettings{})

	// Track calls to writeFileToVM
	var writeCalls []writeFileCall
	manager.writeFileToVMFunc = func(remotePath string, content []byte) error {
		writeCalls = append(writeCalls, writeFileCall{
			remotePath: remotePath,
			content:    content,
		})
		return nil
	}

	// Run SyncClaudeConfig
	err := manager.SyncClaudeConfig()
	if err != nil {
		t.Errorf("SyncClaudeConfig should not error when config doesn't exist: %v", err)
	}

	// Verify writeFileToVM was NOT called for global config
	for _, call := range writeCalls {
		if call.remotePath == "~/.claude/.claude.json" {
			t.Error("writeFileToVM should not be called when global config doesn't exist")
		}
	}
}

func TestSyncClaudeConfig_NoUserConfig(t *testing.T) {
	// Create temp directory with no .claude.json
	tempHome := t.TempDir()

	// Set HOME to our temp directory
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", originalHome)

	// Create manager
	projectPath := "/test/project"
	manager := NewManager(projectPath, session.VagrantSettings{})

	// Track calls to writeFileToVM
	var writeCalls []writeFileCall
	manager.writeFileToVMFunc = func(remotePath string, content []byte) error {
		writeCalls = append(writeCalls, writeFileCall{
			remotePath: remotePath,
			content:    content,
		})
		return nil
	}

	// Run SyncClaudeConfig
	err := manager.SyncClaudeConfig()
	if err != nil {
		t.Errorf("SyncClaudeConfig should not error when config doesn't exist: %v", err)
	}

	// Verify writeFileToVM was NOT called for user config
	for _, call := range writeCalls {
		if call.remotePath == "~/.claude.json" {
			t.Error("writeFileToVM should not be called when user config doesn't exist")
		}
	}
}

func TestSyncClaudeConfig_BothConfigs(t *testing.T) {
	// Create temp directory for global config
	tempConfigDir := t.TempDir()
	globalConfigPath := filepath.Join(tempConfigDir, ".claude.json")
	globalContent := []byte(`{"mcpServers": {"global": {}}}`)

	// Write global config
	if err := os.WriteFile(globalConfigPath, globalContent, 0600); err != nil {
		t.Fatalf("failed to create global config: %v", err)
	}

	// Create temp directory for user config
	tempHome := t.TempDir()
	userConfigPath := filepath.Join(tempHome, ".claude.json")
	userContent := []byte(`{"projects": {"user": {}}}`)

	// Write user config
	if err := os.WriteFile(userConfigPath, userContent, 0600); err != nil {
		t.Fatalf("failed to create user config: %v", err)
	}

	// Set environment variables
	originalEnv := os.Getenv("CLAUDE_CONFIG_DIR")
	originalHome := os.Getenv("HOME")
	os.Setenv("CLAUDE_CONFIG_DIR", tempConfigDir)
	os.Setenv("HOME", tempHome)
	defer func() {
		if originalEnv != "" {
			os.Setenv("CLAUDE_CONFIG_DIR", originalEnv)
		} else {
			os.Unsetenv("CLAUDE_CONFIG_DIR")
		}
		os.Setenv("HOME", originalHome)
	}()

	// Create manager
	projectPath := "/test/project"
	manager := NewManager(projectPath, session.VagrantSettings{})

	// Track calls to writeFileToVM
	var writeCalls []writeFileCall
	manager.writeFileToVMFunc = func(remotePath string, content []byte) error {
		writeCalls = append(writeCalls, writeFileCall{
			remotePath: remotePath,
			content:    content,
		})
		return nil
	}

	// Run SyncClaudeConfig
	err := manager.SyncClaudeConfig()
	if err != nil {
		t.Errorf("SyncClaudeConfig returned error: %v", err)
	}

	// Verify both configs were synced
	foundGlobal := false
	foundUser := false

	for _, call := range writeCalls {
		if call.remotePath == "~/.claude/.claude.json" {
			foundGlobal = true
			if string(call.content) != string(globalContent) {
				t.Errorf("global config content mismatch")
			}
		}
		if call.remotePath == "~/.claude.json" {
			foundUser = true
			if string(call.content) != string(userContent) {
				t.Errorf("user config content mismatch")
			}
		}
	}

	if !foundGlobal {
		t.Error("global config was not synced")
	}
	if !foundUser {
		t.Error("user config was not synced")
	}

	// Verify exactly 2 calls were made
	if len(writeCalls) != 2 {
		t.Errorf("expected 2 writeFileToVM calls, got %d", len(writeCalls))
	}
}

// Helper types for testing
type writeFileCall struct {
	remotePath string
	content    []byte
}
