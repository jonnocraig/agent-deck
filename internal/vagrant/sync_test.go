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

	// Verify at least the 2 core configs were synced
	if len(writeCalls) < 2 {
		t.Errorf("expected at least 2 writeFileToVM calls, got %d", len(writeCalls))
	}
}

func TestSyncClaudeConfig_SettingsAndStatusline(t *testing.T) {
	tempHome := t.TempDir()

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
			name:        "strips installMethod and oauthAccount",
			input:       `{"installMethod":"native","oauthAccount":{"id":"abc"},"mcpServers":{"x":{}},"numStartups":5}`,
			wantAbsent:  []string{"installMethod", "oauthAccount", "mcpServers"},
			wantPresent: []string{"numStartups"},
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

func TestStripSettingsForVM(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantAbsent  []string
		wantPresent []string
	}{
		{
			name:        "strips plugins and hooks",
			input:       `{"enabledPlugins":{"a":true,"b":true},"hooks":{"preToolUse":[]},"statusLine":{"type":"command"},"permissions":{}}`,
			wantAbsent:  []string{"enabledPlugins", "hooks"},
			wantPresent: []string{"statusLine", "permissions"},
		},
		{
			name:        "only plugins present",
			input:       `{"enabledPlugins":{"a":true},"statusLine":{"type":"command"}}`,
			wantAbsent:  []string{"enabledPlugins"},
			wantPresent: []string{"statusLine"},
		},
		{
			name:        "no strippable fields leaves data unchanged",
			input:       `{"statusLine":{"type":"command"},"env":{}}`,
			wantAbsent:  []string{},
			wantPresent: []string{"statusLine", "env"},
		},
		{
			name:  "invalid JSON returns original",
			input: `{broken`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripSettingsForVM([]byte(tt.input))

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
			if strings.Contains(content, "oauthAccount") {
				t.Errorf("oauthAccount should be stripped from user config, got: %s", content)
			}
			if strings.Contains(content, "mcpServers") {
				t.Errorf("mcpServers should be stripped from user config, got: %s", content)
			}
			if !strings.Contains(content, "projects") {
				t.Errorf("projects should be preserved in user config, got: %s", content)
			}
		}
	}
}

func TestSyncClaudeConfig_SettingsPluginsStripped(t *testing.T) {
	tempHome := t.TempDir()

	claudeDir := filepath.Join(tempHome, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("failed to create .claude dir: %v", err)
	}

	settingsWithPlugins := []byte(`{"enabledPlugins":{"a":true,"b":true},"hooks":{"preToolUse":[]},"statusLine":{"type":"command"}}`)
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"), settingsWithPlugins, 0644); err != nil {
		t.Fatalf("failed to create settings.json: %v", err)
	}

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", originalHome)

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
		writeCalls = append(writeCalls, writeFileCall{remotePath: remotePath, content: content})
		return nil
	}

	if err := manager.SyncClaudeConfig(); err != nil {
		t.Fatalf("SyncClaudeConfig returned error: %v", err)
	}

	for _, call := range writeCalls {
		if call.remotePath == "~/.claude/settings.json" {
			content := string(call.content)
			if strings.Contains(content, "enabledPlugins") {
				t.Errorf("enabledPlugins should be stripped from settings.json, got: %s", content)
			}
			if strings.Contains(content, "hooks") {
				t.Errorf("hooks should be stripped from settings.json, got: %s", content)
			}
			if !strings.Contains(content, "statusLine") {
				t.Errorf("statusLine should be preserved in settings.json, got: %s", content)
			}
		}
	}
}

// Helper types for testing
type writeFileCall struct {
	remotePath string
	content    []byte
}
