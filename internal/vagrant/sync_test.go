package vagrant

import (
	"encoding/base64"
	"encoding/json"
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
	// Use content without mcpServers (they are stripped during sync)
	globalContent := []byte(`{"projects": {"test": {}}}`)

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

	// Disable OAuth extraction for this test
	originalOAuth := extractOAuthCredentialsFunc
	extractOAuthCredentialsFunc = func() ([]byte, error) { return nil, ErrNoOAuthCredentials }
	defer func() { extractOAuthCredentialsFunc = originalOAuth }()

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
			// Content should have hasCompletedOnboarding injected
			if !strings.Contains(string(call.content), "projects") {
				t.Errorf("global config should preserve projects key, got: %s", string(call.content))
			}
			if !strings.Contains(string(call.content), "hasCompletedOnboarding") {
				t.Errorf("global config should have hasCompletedOnboarding injected, got: %s", string(call.content))
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

	// Disable OAuth extraction for this test
	originalOAuth := extractOAuthCredentialsFunc
	extractOAuthCredentialsFunc = func() ([]byte, error) { return nil, ErrNoOAuthCredentials }
	defer func() { extractOAuthCredentialsFunc = originalOAuth }()

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
			if !strings.Contains(string(call.content), "projects") {
				t.Errorf("user config should preserve projects, got: %s", string(call.content))
			}
			if !strings.Contains(string(call.content), "hasCompletedOnboarding") {
				t.Errorf("user config should have hasCompletedOnboarding, got: %s", string(call.content))
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

	// Disable OAuth extraction for this test
	originalOAuth := extractOAuthCredentialsFunc
	extractOAuthCredentialsFunc = func() ([]byte, error) { return nil, ErrNoOAuthCredentials }
	defer func() { extractOAuthCredentialsFunc = originalOAuth }()

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

	// Disable OAuth extraction for this test
	originalOAuth := extractOAuthCredentialsFunc
	extractOAuthCredentialsFunc = func() ([]byte, error) { return nil, ErrNoOAuthCredentials }
	defer func() { extractOAuthCredentialsFunc = originalOAuth }()

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
	// Use content without mcpServers (they are stripped during sync)
	globalContent := []byte(`{"projects": {"global": {}}}`)

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

	// Disable OAuth extraction for this test
	originalOAuth := extractOAuthCredentialsFunc
	extractOAuthCredentialsFunc = func() ([]byte, error) { return nil, ErrNoOAuthCredentials }
	defer func() { extractOAuthCredentialsFunc = originalOAuth }()

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
			if !strings.Contains(string(call.content), "global") {
				t.Errorf("global config should preserve projects, got: %s", string(call.content))
			}
			if !strings.Contains(string(call.content), "hasCompletedOnboarding") {
				t.Errorf("global config should have hasCompletedOnboarding, got: %s", string(call.content))
			}
		}
		if call.remotePath == "~/.claude.json" {
			foundUser = true
			if !strings.Contains(string(call.content), "user") {
				t.Errorf("user config should preserve projects, got: %s", string(call.content))
			}
			if !strings.Contains(string(call.content), "hasCompletedOnboarding") {
				t.Errorf("user config should have hasCompletedOnboarding, got: %s", string(call.content))
			}
		}
	}

	if !foundGlobal {
		t.Error("global config was not synced")
	}
	if !foundUser {
		t.Error("user config was not synced")
	}

	// Verify at least the 2 core configs were synced
	if len(writeCalls) < 2 {
		t.Errorf("expected at least 2 writeFileToVM calls, got %d", len(writeCalls))
	}
}

func TestSyncClaudeConfig_SettingsAndStatusline(t *testing.T) {
	tempHome := t.TempDir()

	// Disable OAuth extraction for this test
	originalOAuth := extractOAuthCredentialsFunc
	extractOAuthCredentialsFunc = func() ([]byte, error) { return nil, ErrNoOAuthCredentials }
	defer func() { extractOAuthCredentialsFunc = originalOAuth }()

	// Create ~/.claude/settings.json
	claudeDir := filepath.Join(tempHome, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("failed to create .claude dir: %v", err)
	}
	settingsContent := []byte(`{"statusLine":{"type":"command","command":"~/.claude/statusline.sh"}}`)
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"), settingsContent, 0644); err != nil {
		t.Fatalf("failed to create settings.json: %v", err)
	}

	// Create ~/.claude/statusline.sh
	statuslineContent := []byte("#!/bin/bash\necho 'test'")
	if err := os.WriteFile(filepath.Join(claudeDir, "statusline.sh"), statuslineContent, 0755); err != nil {
		t.Fatalf("failed to create statusline.sh: %v", err)
	}

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", originalHome)

	// Redirect CLAUDE_CONFIG_DIR to avoid picking up real global config
	originalEnv := os.Getenv("CLAUDE_CONFIG_DIR")
	os.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())
	defer func() {
		if originalEnv != "" {
			os.Setenv("CLAUDE_CONFIG_DIR", originalEnv)
		} else {
			os.Unsetenv("CLAUDE_CONFIG_DIR")
		}
	}()

	manager := NewManager("/test/project", session.VagrantSettings{})

	var writeCalls []writeFileCall
	manager.writeFileToVMFunc = func(remotePath string, content []byte) error {
		writeCalls = append(writeCalls, writeFileCall{
			remotePath: remotePath,
			content:    content,
		})
		return nil
	}

	err := manager.SyncClaudeConfig()
	if err != nil {
		t.Errorf("SyncClaudeConfig returned error: %v", err)
	}

	foundSettings := false
	foundStatusline := false
	for _, call := range writeCalls {
		if call.remotePath == "~/.claude/settings.json" {
			foundSettings = true
			if string(call.content) != string(settingsContent) {
				t.Errorf("settings.json content mismatch")
			}
		}
		if call.remotePath == "~/.claude/statusline.sh" {
			foundStatusline = true
			if string(call.content) != string(statuslineContent) {
				t.Errorf("statusline.sh content mismatch")
			}
		}
	}

	if !foundSettings {
		t.Error("settings.json was not synced")
	}
	if !foundStatusline {
		t.Error("statusline.sh was not synced")
	}
}

func TestStripMCPServers(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantMCP  bool // should mcpServers be present in output
		wantKeys []string // other keys that should survive
	}{
		{
			name:     "strips mcpServers",
			input:    `{"mcpServers":{"test":{}},"projects":{"a":{}}}`,
			wantMCP:  false,
			wantKeys: []string{"projects"},
		},
		{
			name:     "no mcpServers unchanged",
			input:    `{"projects":{"a":{}}}`,
			wantMCP:  false,
			wantKeys: []string{"projects"},
		},
		{
			name:    "invalid JSON returns original",
			input:   `not json`,
			wantMCP: false,
		},
		{
			name:    "empty object",
			input:   `{}`,
			wantMCP: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripMCPServers([]byte(tt.input))

			// For invalid JSON, should return original
			if tt.name == "invalid JSON returns original" {
				if string(result) != tt.input {
					t.Errorf("expected original data for invalid JSON")
				}
				return
			}

			var parsed map[string]interface{}
			if err := json.Unmarshal(result, &parsed); err != nil {
				t.Fatalf("result is not valid JSON: %v", err)
			}

			_, hasMCP := parsed["mcpServers"]
			if hasMCP && !tt.wantMCP {
				t.Error("mcpServers should have been stripped")
			}

			for _, key := range tt.wantKeys {
				if _, ok := parsed[key]; !ok {
					t.Errorf("expected key %q to survive stripping", key)
				}
			}
		})
	}
}

func TestSyncClaudeConfig_MCPServersStripped(t *testing.T) {
	tempHome := t.TempDir()

	// Disable OAuth extraction for this test
	originalOAuth := extractOAuthCredentialsFunc
	extractOAuthCredentialsFunc = func() ([]byte, error) { return nil, ErrNoOAuthCredentials }
	defer func() { extractOAuthCredentialsFunc = originalOAuth }()

	// Create global config with mcpServers
	claudeDir := filepath.Join(tempHome, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("failed to create .claude dir: %v", err)
	}
	globalWithMCP := []byte(`{"mcpServers":{"blender":{"command":"npx"}},"other":"kept"}`)
	if err := os.WriteFile(filepath.Join(claudeDir, ".claude.json"), globalWithMCP, 0600); err != nil {
		t.Fatalf("failed to write global config: %v", err)
	}

	// Create user config with mcpServers
	userWithMCP := []byte(`{"mcpServers":{"context7":{"command":"npx"}},"projects":{}}`)
	if err := os.WriteFile(filepath.Join(tempHome, ".claude.json"), userWithMCP, 0600); err != nil {
		t.Fatalf("failed to write user config: %v", err)
	}

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", originalHome)

	originalEnv := os.Getenv("CLAUDE_CONFIG_DIR")
	os.Setenv("CLAUDE_CONFIG_DIR", claudeDir)
	defer func() {
		if originalEnv != "" {
			os.Setenv("CLAUDE_CONFIG_DIR", originalEnv)
		} else {
			os.Unsetenv("CLAUDE_CONFIG_DIR")
		}
	}()

	manager := NewManager("/test/project", session.VagrantSettings{})

	var writeCalls []writeFileCall
	manager.writeFileToVMFunc = func(remotePath string, content []byte) error {
		writeCalls = append(writeCalls, writeFileCall{remotePath: remotePath, content: content})
		return nil
	}

	if err := manager.SyncClaudeConfig(); err != nil {
		t.Fatalf("SyncClaudeConfig returned error: %v", err)
	}

	for _, call := range writeCalls {
		if call.remotePath == "~/.claude/.claude.json" || call.remotePath == "~/.claude.json" {
			if strings.Contains(string(call.content), "mcpServers") {
				t.Errorf("mcpServers should be stripped from %s, got: %s", call.remotePath, string(call.content))
			}
		}
	}
}

func TestStripHostOnlyFields(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantAbsent   []string // keys that should be removed
		wantPresent  []string // keys that should be preserved
	}{
		{
			name:        "strips installMethod and mcpServers but keeps oauthAccount",
			input:       `{"installMethod":"native","oauthAccount":{"id":"abc"},"mcpServers":{"x":{}},"numStartups":5}`,
			wantAbsent:  []string{"installMethod", "mcpServers"},
			wantPresent: []string{"numStartups", "oauthAccount"},
		},
		{
			name:        "only installMethod present",
			input:       `{"installMethod":"native","projects":{}}`,
			wantAbsent:  []string{"installMethod"},
			wantPresent: []string{"projects"},
		},
		{
			name:        "no host-only fields leaves data unchanged",
			input:       `{"numStartups":5,"projects":{}}`,
			wantAbsent:  []string{},
			wantPresent: []string{"numStartups", "projects"},
		},
		{
			name:  "invalid JSON returns original",
			input: `not valid json`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripHostOnlyFields([]byte(tt.input))

			if tt.name == "invalid JSON returns original" {
				if string(result) != tt.input {
					t.Errorf("expected original data for invalid JSON")
				}
				return
			}

			var parsed map[string]interface{}
			if err := json.Unmarshal(result, &parsed); err != nil {
				t.Fatalf("result is not valid JSON: %v\nresult: %s", err, string(result))
			}

			for _, key := range tt.wantAbsent {
				if _, ok := parsed[key]; ok {
					t.Errorf("key %q should have been stripped", key)
				}
			}

			for _, key := range tt.wantPresent {
				if _, ok := parsed[key]; !ok {
					t.Errorf("key %q should have been preserved", key)
				}
			}
		})
	}
}

func TestStripJSONKeys_NoChangeWhenNoKeysPresent(t *testing.T) {
	input := `{"a": 1, "b": 2}`
	result := stripJSONKeys([]byte(input), []string{"c", "d"})
	// Should return original data when no keys matched
	if string(result) != input {
		t.Errorf("expected original data when no keys match, got: %s", string(result))
	}
}

func TestSyncClaudeConfig_HostOnlyFieldsStripped(t *testing.T) {
	tempHome := t.TempDir()

	// Disable OAuth extraction for this test
	originalOAuth := extractOAuthCredentialsFunc
	extractOAuthCredentialsFunc = func() ([]byte, error) { return nil, ErrNoOAuthCredentials }
	defer func() { extractOAuthCredentialsFunc = originalOAuth }()

	// Create global config with host-only fields
	claudeDir := filepath.Join(tempHome, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("failed to create .claude dir: %v", err)
	}
	globalWithHostFields := []byte(`{"mcpServers":{"test":{}},"other":"kept"}`)
	if err := os.WriteFile(filepath.Join(claudeDir, ".claude.json"), globalWithHostFields, 0600); err != nil {
		t.Fatalf("failed to write global config: %v", err)
	}

	// Create user config with installMethod and oauthAccount
	userWithHostFields := []byte(`{"installMethod":"native","oauthAccount":{"id":"x"},"mcpServers":{"y":{}},"projects":{}}`)
	if err := os.WriteFile(filepath.Join(tempHome, ".claude.json"), userWithHostFields, 0600); err != nil {
		t.Fatalf("failed to write user config: %v", err)
	}

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", originalHome)

	originalEnv := os.Getenv("CLAUDE_CONFIG_DIR")
	os.Setenv("CLAUDE_CONFIG_DIR", claudeDir)
	defer func() {
		if originalEnv != "" {
			os.Setenv("CLAUDE_CONFIG_DIR", originalEnv)
		} else {
			os.Unsetenv("CLAUDE_CONFIG_DIR")
		}
	}()

	manager := NewManager("/test/project", session.VagrantSettings{})

	var writeCalls []writeFileCall
	manager.writeFileToVMFunc = func(remotePath string, content []byte) error {
		writeCalls = append(writeCalls, writeFileCall{remotePath: remotePath, content: content})
		return nil
	}

	if err := manager.SyncClaudeConfig(); err != nil {
		t.Fatalf("SyncClaudeConfig returned error: %v", err)
	}

	for _, call := range writeCalls {
		if call.remotePath == "~/.claude.json" {
			content := string(call.content)
			if strings.Contains(content, "installMethod") {
				t.Errorf("installMethod should be stripped from user config, got: %s", content)
			}
			if strings.Contains(content, "mcpServers") {
				t.Errorf("mcpServers should be stripped from user config, got: %s", content)
			}
			if !strings.Contains(content, "oauthAccount") {
				t.Errorf("oauthAccount should be preserved in user config for OAuth refresh, got: %s", content)
			}
			if !strings.Contains(content, "projects") {
				t.Errorf("projects should be preserved in user config, got: %s", content)
			}
		}
	}
}

func TestSyncClaudeConfig_SettingsPreserved(t *testing.T) {
	tempHome := t.TempDir()

	// Disable OAuth extraction for this test
	originalOAuth := extractOAuthCredentialsFunc
	extractOAuthCredentialsFunc = func() ([]byte, error) { return nil, ErrNoOAuthCredentials }
	defer func() { extractOAuthCredentialsFunc = originalOAuth }()

	claudeDir := filepath.Join(tempHome, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("failed to create .claude dir: %v", err)
	}

	settingsWithPlugins := []byte(`{"enabledPlugins":{"a":true,"b":true},"hooks":{"preToolUse":[]},"statusLine":{"type":"command"}}`)
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"), settingsWithPlugins, 0644); err != nil {
		t.Fatalf("failed to create settings.json: %v", err)
	}

	t.Setenv("HOME", tempHome)
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())

	manager := NewManager("/test/project", session.VagrantSettings{})

	var writeCalls []writeFileCall
	manager.writeFileToVMFunc = func(remotePath string, content []byte) error {
		writeCalls = append(writeCalls, writeFileCall{remotePath: remotePath, content: content})
		return nil
	}

	if err := manager.SyncClaudeConfig(); err != nil {
		t.Fatalf("SyncClaudeConfig returned error: %v", err)
	}

	for _, call := range writeCalls {
		if call.remotePath == "~/.claude/settings.json" {
			content := string(call.content)
			if !strings.Contains(content, "enabledPlugins") {
				t.Errorf("enabledPlugins should be preserved in settings.json, got: %s", content)
			}
			if !strings.Contains(content, "hooks") {
				t.Errorf("hooks should be preserved in settings.json, got: %s", content)
			}
			if !strings.Contains(content, "statusLine") {
				t.Errorf("statusLine should be preserved in settings.json, got: %s", content)
			}
		}
	}
}

func TestInjectVMFields_AddsOnboarding(t *testing.T) {
	input := `{"projects":{}}`
	result := injectVMFields([]byte(input))

	var parsed map[string]json.RawMessage
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	raw, ok := parsed["hasCompletedOnboarding"]
	if !ok {
		t.Fatal("hasCompletedOnboarding should be present")
	}
	if string(raw) != "true" {
		t.Errorf("hasCompletedOnboarding = %s, want true", string(raw))
	}
}

func TestInjectVMFields_PreservesExistingFields(t *testing.T) {
	input := `{"projects":{"test":{}},"numStartups":5}`
	result := injectVMFields([]byte(input))

	var parsed map[string]json.RawMessage
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	if _, ok := parsed["projects"]; !ok {
		t.Error("projects should be preserved")
	}
	if _, ok := parsed["numStartups"]; !ok {
		t.Error("numStartups should be preserved")
	}
	if _, ok := parsed["hasCompletedOnboarding"]; !ok {
		t.Error("hasCompletedOnboarding should be injected")
	}
}

func TestInjectVMFields_InvalidJSON(t *testing.T) {
	input := `not valid json`
	result := injectVMFields([]byte(input))
	if string(result) != input {
		t.Errorf("expected original data for invalid JSON, got: %s", string(result))
	}
}

func TestSyncClaudeConfig_OAuthCredentialsSynced(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	// Redirect CLAUDE_CONFIG_DIR to avoid picking up real global config
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())

	// Inject mock OAuth credentials
	original := extractOAuthCredentialsFunc
	extractOAuthCredentialsFunc = func() ([]byte, error) {
		return []byte(`{"accessToken":"sk-ant-oat01-test","refreshToken":"sk-ant-ort01-test"}`), nil
	}
	defer func() { extractOAuthCredentialsFunc = original }()

	manager := NewManager("/test/project", session.VagrantSettings{})

	var writeCalls []writeFileCall
	manager.writeFileToVMFunc = func(remotePath string, content []byte) error {
		writeCalls = append(writeCalls, writeFileCall{remotePath: remotePath, content: content})
		return nil
	}

	if err := manager.SyncClaudeConfig(); err != nil {
		t.Fatalf("SyncClaudeConfig returned error: %v", err)
	}

	found := false
	for _, call := range writeCalls {
		if call.remotePath == "~/.claude/.credentials.json" {
			found = true
			if !strings.Contains(string(call.content), "sk-ant-oat01-test") {
				t.Errorf("credentials content should contain access token, got: %s", string(call.content))
			}
		}
	}
	if !found {
		t.Error("OAuth credentials should be synced to ~/.claude/.credentials.json")
	}
}

func TestSyncClaudeConfig_NoOAuthGraceful(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())

	// Inject mock that returns no credentials
	original := extractOAuthCredentialsFunc
	extractOAuthCredentialsFunc = func() ([]byte, error) {
		return nil, ErrNoOAuthCredentials
	}
	defer func() { extractOAuthCredentialsFunc = original }()

	manager := NewManager("/test/project", session.VagrantSettings{})

	var writeCalls []writeFileCall
	manager.writeFileToVMFunc = func(remotePath string, content []byte) error {
		writeCalls = append(writeCalls, writeFileCall{remotePath: remotePath, content: content})
		return nil
	}

	if err := manager.SyncClaudeConfig(); err != nil {
		t.Fatalf("SyncClaudeConfig should not error when OAuth is absent: %v", err)
	}

	for _, call := range writeCalls {
		if call.remotePath == "~/.claude/.credentials.json" {
			t.Error("credentials should not be written when no OAuth found")
		}
	}
}

func TestSyncClaudeConfig_OnboardingInjected(t *testing.T) {
	tempHome := t.TempDir()

	// Create user config
	userContent := `{"projects":{}}`
	if err := os.WriteFile(filepath.Join(tempHome, ".claude.json"), []byte(userContent), 0600); err != nil {
		t.Fatalf("failed to create user config: %v", err)
	}

	t.Setenv("HOME", tempHome)
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())

	// Disable OAuth extraction for this test
	original := extractOAuthCredentialsFunc
	extractOAuthCredentialsFunc = func() ([]byte, error) { return nil, ErrNoOAuthCredentials }
	defer func() { extractOAuthCredentialsFunc = original }()

	manager := NewManager("/test/project", session.VagrantSettings{})

	var writeCalls []writeFileCall
	manager.writeFileToVMFunc = func(remotePath string, content []byte) error {
		writeCalls = append(writeCalls, writeFileCall{remotePath: remotePath, content: content})
		return nil
	}

	if err := manager.SyncClaudeConfig(); err != nil {
		t.Fatalf("SyncClaudeConfig returned error: %v", err)
	}

	for _, call := range writeCalls {
		if call.remotePath == "~/.claude.json" {
			if !strings.Contains(string(call.content), "hasCompletedOnboarding") {
				t.Errorf("user config should contain hasCompletedOnboarding, got: %s", string(call.content))
			}
		}
	}
}

// TestGetMergedVMClaudeMD verifies the CLAUDE.md content strips frontmatter and keeps the body
func TestGetMergedVMClaudeMD(t *testing.T) {
	md := getMergedVMClaudeMD()

	// Should NOT contain YAML frontmatter
	if strings.Contains(md, "---") && strings.HasPrefix(md, "---") {
		t.Error("CLAUDE.md should not contain YAML frontmatter delimiters")
	}
	if strings.Contains(md, "name: operating-in-vagrant") {
		t.Error("CLAUDE.md should not contain frontmatter name field")
	}

	// Should contain the skill body content
	for _, keyword := range []string{
		"Operating in a Vagrant VM",
		"sudo",
		"/vagrant",
		"10.0.2.2",
		"inotify",
		"disposable",
	} {
		if !strings.Contains(md, keyword) {
			t.Errorf("CLAUDE.md missing expected content: %q", keyword)
		}
	}
}

// TestSyncClaudeConfigWritesClaudeMD verifies that SyncClaudeConfig writes ~/.claude/CLAUDE.md
func TestSyncClaudeConfigWritesClaudeMD(t *testing.T) {
	tmpDir := t.TempDir()
	settings := session.VagrantSettings{}
	mgr := NewManager(tmpDir, settings)

	var writeCalls []writeFileCall
	mgr.writeFileToVMFunc = func(remotePath string, content []byte) error {
		writeCalls = append(writeCalls, writeFileCall{remotePath: remotePath, content: content})
		return nil
	}

	// Mock OAuth to avoid hitting real Keychain
	origOAuth := extractOAuthCredentialsFunc
	extractOAuthCredentialsFunc = func() ([]byte, error) { return nil, ErrNoOAuthCredentials }
	defer func() { extractOAuthCredentialsFunc = origOAuth }()

	if err := mgr.SyncClaudeConfig(); err != nil {
		t.Fatalf("SyncClaudeConfig failed: %v", err)
	}

	var claudeMDCall *writeFileCall
	for i, call := range writeCalls {
		if call.remotePath == "~/.claude/CLAUDE.md" {
			claudeMDCall = &writeCalls[i]
			break
		}
	}

	if claudeMDCall == nil {
		t.Fatal("SyncClaudeConfig did not write ~/.claude/CLAUDE.md")
	}

	content := string(claudeMDCall.content)
	if strings.Contains(content, "name: operating-in-vagrant") {
		t.Error("CLAUDE.md should not contain YAML frontmatter")
	}
	if !strings.Contains(content, "Operating in a Vagrant VM") {
		t.Error("CLAUDE.md should contain VM context instructions")
	}
}

// TestGetMergedVMClaudeMD_WithHostFile verifies that getMergedVMClaudeMD merges
// the VM context with the user's host ~/.claude/CLAUDE.md when it exists.
func TestGetMergedVMClaudeMD_WithHostFile(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	// Create ~/.claude/CLAUDE.md on the "host"
	claudeDir := filepath.Join(tempHome, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("failed to create .claude dir: %v", err)
	}
	hostContent := "# My Custom Rules\n\nAlways use tabs."
	if err := os.WriteFile(filepath.Join(claudeDir, "CLAUDE.md"), []byte(hostContent), 0644); err != nil {
		t.Fatalf("failed to write host CLAUDE.md: %v", err)
	}

	result := getMergedVMClaudeMD()

	// VM context must be present
	if !strings.Contains(result, "Operating in a Vagrant VM") {
		t.Error("merged CLAUDE.md should contain VM context ('Operating in a Vagrant VM')")
	}

	// User content must be present
	if !strings.Contains(result, "My Custom Rules") {
		t.Error("merged CLAUDE.md should contain user content ('My Custom Rules')")
	}
	if !strings.Contains(result, "Always use tabs") {
		t.Error("merged CLAUDE.md should contain user content ('Always use tabs')")
	}

	// Separator must be present between VM and user content
	if !strings.Contains(result, "\n\n---\n\n") {
		t.Error("merged CLAUDE.md should contain a '---' separator between VM and user content")
	}

	// VM content must appear BEFORE user content
	vmIdx := strings.Index(result, "Operating in a Vagrant VM")
	userIdx := strings.Index(result, "My Custom Rules")
	if vmIdx >= userIdx {
		t.Errorf("VM content (at %d) should appear before user content (at %d)", vmIdx, userIdx)
	}
}

// TestGetMergedVMClaudeMD_NoHostFile verifies that getMergedVMClaudeMD returns
// only VM context when the user has no ~/.claude/CLAUDE.md on the host.
func TestGetMergedVMClaudeMD_NoHostFile(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	// No ~/.claude/CLAUDE.md exists in tempHome

	result := getMergedVMClaudeMD()

	// VM context must be present
	if !strings.Contains(result, "Operating in a Vagrant VM") {
		t.Error("CLAUDE.md should contain VM context ('Operating in a Vagrant VM')")
	}

	// Must NOT contain the merge separator pattern
	if strings.Contains(result, "\n\n---\n\n") {
		t.Error("CLAUDE.md without host file should not contain the merge separator ('\\n\\n---\\n\\n')")
	}

	// Must NOT contain any user-specific content (sanity check)
	if strings.Contains(result, "My Custom Rules") {
		t.Error("CLAUDE.md should not contain user content when host file is absent")
	}
}

// TestSyncClaudeConfig_CallsProfileSync verifies that SyncClaudeConfig executes
// step 7 (syncProfileToVM) by checking that createProfileTarFunc is invoked.
func TestSyncClaudeConfig_CallsProfileSync(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())

	// Disable OAuth to avoid hitting real Keychain
	origOAuth := extractOAuthCredentialsFunc
	extractOAuthCredentialsFunc = func() ([]byte, error) { return nil, ErrNoOAuthCredentials }
	defer func() { extractOAuthCredentialsFunc = origOAuth }()

	// Track whether createProfileTar was called
	called := false
	origTar := createProfileTarFunc
	createProfileTarFunc = func(homeDir string) ([]byte, error) {
		called = true
		return []byte{}, nil // Empty tar = syncProfileToVM returns early before SSH
	}
	defer func() { createProfileTarFunc = origTar }()

	manager := NewManager("/test/project", session.VagrantSettings{})
	manager.writeFileToVMFunc = func(remotePath string, content []byte) error {
		return nil
	}

	if err := manager.SyncClaudeConfig(); err != nil {
		t.Fatalf("SyncClaudeConfig returned error: %v", err)
	}

	if !called {
		t.Error("SyncClaudeConfig should call createProfileTarFunc (step 7)")
	}
}

func TestSyncClaudeConfig_ShellRCFiles(t *testing.T) {
	tempHome := t.TempDir()

	origOAuth := extractOAuthCredentialsFunc
	extractOAuthCredentialsFunc = func() ([]byte, error) { return nil, ErrNoOAuthCredentials }
	defer func() { extractOAuthCredentialsFunc = origOAuth }()

	// Create shell RC files
	os.WriteFile(filepath.Join(tempHome, ".zshrc"), []byte("alias ll='ls -la'\nexport EDITOR=vim"), 0644)
	os.WriteFile(filepath.Join(tempHome, ".bashrc"), []byte("alias gs='git status'"), 0644)

	os.MkdirAll(filepath.Join(tempHome, ".claude"), 0755)

	t.Setenv("HOME", tempHome)
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())

	manager := NewManager("/test/project", session.VagrantSettings{})

	var writeCalls []writeFileCall
	manager.writeFileToVMFunc = func(remotePath string, content []byte) error {
		writeCalls = append(writeCalls, writeFileCall{remotePath: remotePath, content: content})
		return nil
	}

	if err := manager.SyncClaudeConfig(); err != nil {
		t.Fatalf("SyncClaudeConfig returned error: %v", err)
	}

	foundZshrc := false
	foundBashrc := false
	for _, call := range writeCalls {
		if call.remotePath == "~/.zshrc" {
			foundZshrc = true
			if !strings.Contains(string(call.content), "alias ll") {
				t.Errorf("~/.zshrc content mismatch: %s", string(call.content))
			}
		}
		if call.remotePath == "~/.bashrc" {
			foundBashrc = true
			if !strings.Contains(string(call.content), "alias gs") {
				t.Errorf("~/.bashrc content mismatch: %s", string(call.content))
			}
		}
	}

	if !foundZshrc {
		t.Error("expected ~/.zshrc to be synced")
	}
	if !foundBashrc {
		t.Error("expected ~/.bashrc to be synced")
	}
}

func TestSyncClaudeConfig_GitConfig(t *testing.T) {
	tempHome := t.TempDir()

	origOAuth := extractOAuthCredentialsFunc
	extractOAuthCredentialsFunc = func() ([]byte, error) { return nil, ErrNoOAuthCredentials }
	defer func() { extractOAuthCredentialsFunc = origOAuth }()

	gitConfig := "[user]\n\tname = Test User\n\temail = test@example.com\n"
	os.WriteFile(filepath.Join(tempHome, ".gitconfig"), []byte(gitConfig), 0644)

	os.MkdirAll(filepath.Join(tempHome, ".claude"), 0755)

	t.Setenv("HOME", tempHome)
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())

	manager := NewManager("/test/project", session.VagrantSettings{})

	var writeCalls []writeFileCall
	manager.writeFileToVMFunc = func(remotePath string, content []byte) error {
		writeCalls = append(writeCalls, writeFileCall{remotePath: remotePath, content: content})
		return nil
	}

	if err := manager.SyncClaudeConfig(); err != nil {
		t.Fatalf("SyncClaudeConfig returned error: %v", err)
	}

	found := false
	for _, call := range writeCalls {
		if call.remotePath == "~/.gitconfig" {
			found = true
			if !strings.Contains(string(call.content), "Test User") {
				t.Errorf("~/.gitconfig content mismatch: %s", string(call.content))
			}
		}
	}

	if !found {
		t.Error("expected ~/.gitconfig to be synced")
	}
}

// Helper types for testing
type writeFileCall struct {
	remotePath string
	content    []byte
}
