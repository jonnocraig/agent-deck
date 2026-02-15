package session

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
)

func TestUserConfig_ClaudeConfigDir(t *testing.T) {
	// Create temp config file
	tmpDir := t.TempDir()
	configContent := `
[claude]
config_dir = "~/.claude-work"

[tools.test]
command = "test"
`
	configPath := filepath.Join(tmpDir, "config.toml")
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Test parsing
	var config UserConfig
	_, err := toml.DecodeFile(configPath, &config)
	if err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}

	if config.Claude.ConfigDir != "~/.claude-work" {
		t.Errorf("Claude.ConfigDir = %s, want ~/.claude-work", config.Claude.ConfigDir)
	}
}

func TestUserConfig_ClaudeConfigDirEmpty(t *testing.T) {
	// Test with no Claude section
	tmpDir := t.TempDir()
	configContent := `
[tools.test]
command = "test"
`
	configPath := filepath.Join(tmpDir, "config.toml")
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	var config UserConfig
	_, err := toml.DecodeFile(configPath, &config)
	if err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}

	if config.Claude.ConfigDir != "" {
		t.Errorf("Claude.ConfigDir = %s, want empty string", config.Claude.ConfigDir)
	}
}

func TestGlobalSearchConfig(t *testing.T) {
	// Create temp config with global search settings
	tmpDir := t.TempDir()
	configContent := `
[global_search]
enabled = true
tier = "auto"
memory_limit_mb = 150
recent_days = 60
index_rate_limit = 30
`
	configPath := filepath.Join(tmpDir, "config.toml")
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Test parsing
	var config UserConfig
	_, err := toml.DecodeFile(configPath, &config)
	if err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}

	if !config.GlobalSearch.Enabled {
		t.Error("Expected GlobalSearch.Enabled to be true")
	}
	if config.GlobalSearch.Tier != "auto" {
		t.Errorf("Expected tier 'auto', got %q", config.GlobalSearch.Tier)
	}
	if config.GlobalSearch.MemoryLimitMB != 150 {
		t.Errorf("Expected MemoryLimitMB 150, got %d", config.GlobalSearch.MemoryLimitMB)
	}
	if config.GlobalSearch.RecentDays != 60 {
		t.Errorf("Expected RecentDays 60, got %d", config.GlobalSearch.RecentDays)
	}
	if config.GlobalSearch.IndexRateLimit != 30 {
		t.Errorf("Expected IndexRateLimit 30, got %d", config.GlobalSearch.IndexRateLimit)
	}
}

func TestGlobalSearchConfigDefaults(t *testing.T) {
	// Config without global_search section should parse with zero values
	// (defaults are applied by LoadUserConfig, not parsing)
	tmpDir := t.TempDir()
	configContent := `default_tool = "claude"`
	configPath := filepath.Join(tmpDir, "config.toml")
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	var config UserConfig
	_, err := toml.DecodeFile(configPath, &config)
	if err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}

	// When parsing directly without LoadUserConfig, values should be zero
	if config.GlobalSearch.Enabled {
		t.Error("GlobalSearch.Enabled should be false when not specified (zero value)")
	}
	if config.GlobalSearch.MemoryLimitMB != 0 {
		t.Errorf("Expected default MemoryLimitMB 0 (zero value), got %d", config.GlobalSearch.MemoryLimitMB)
	}
}

func TestGlobalSearchConfigDisabled(t *testing.T) {
	// Test explicitly disabling global search
	tmpDir := t.TempDir()
	configContent := `
[global_search]
enabled = false
tier = "disabled"
`
	configPath := filepath.Join(tmpDir, "config.toml")
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	var config UserConfig
	_, err := toml.DecodeFile(configPath, &config)
	if err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}

	if config.GlobalSearch.Enabled {
		t.Error("Expected GlobalSearch.Enabled to be false")
	}
	if config.GlobalSearch.Tier != "disabled" {
		t.Errorf("Expected tier 'disabled', got %q", config.GlobalSearch.Tier)
	}
}

func TestSaveUserConfig(t *testing.T) {
	// Setup: use temp directory
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Clear cache
	ClearUserConfigCache()

	// Create agent-deck directory
	agentDeckDir := filepath.Join(tempDir, ".agent-deck")
	_ = os.MkdirAll(agentDeckDir, 0700)

	// Create config to save
	dangerousModeBool := true
	config := &UserConfig{
		DefaultTool: "claude",
		Claude: ClaudeSettings{
			DangerousMode: &dangerousModeBool,
			ConfigDir:     "~/.claude-work",
		},
		Logs: LogSettings{
			MaxSizeMB:     20,
			MaxLines:      5000,
			RemoveOrphans: true,
		},
	}

	// Save it
	err := SaveUserConfig(config)
	if err != nil {
		t.Fatalf("SaveUserConfig failed: %v", err)
	}

	// Clear cache and reload
	ClearUserConfigCache()
	loaded, err := LoadUserConfig()
	if err != nil {
		t.Fatalf("LoadUserConfig failed: %v", err)
	}

	// Verify values
	if loaded.DefaultTool != "claude" {
		t.Errorf("DefaultTool: got %q, want %q", loaded.DefaultTool, "claude")
	}
	if !loaded.Claude.GetDangerousMode() {
		t.Error("DangerousMode should be true")
	}
	if loaded.Claude.ConfigDir != "~/.claude-work" {
		t.Errorf("ConfigDir: got %q, want %q", loaded.Claude.ConfigDir, "~/.claude-work")
	}
	if loaded.Logs.MaxSizeMB != 20 {
		t.Errorf("MaxSizeMB: got %d, want %d", loaded.Logs.MaxSizeMB, 20)
	}
}

func TestGetTheme_Default(t *testing.T) {
	// Setup: use temp directory with no config
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)
	ClearUserConfigCache()

	theme := GetTheme()
	if theme != "dark" {
		t.Errorf("GetTheme: got %q, want %q", theme, "dark")
	}
}

func TestGetTheme_Light(t *testing.T) {
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)
	ClearUserConfigCache()

	// Create config with light theme
	agentDeckDir := filepath.Join(tempDir, ".agent-deck")
	_ = os.MkdirAll(agentDeckDir, 0700)
	config := &UserConfig{Theme: "light"}
	_ = SaveUserConfig(config)
	ClearUserConfigCache()

	theme := GetTheme()
	if theme != "light" {
		t.Errorf("GetTheme: got %q, want %q", theme, "light")
	}
}

func TestWorktreeConfig(t *testing.T) {
	// Create temp config with worktree settings
	tmpDir := t.TempDir()
	configContent := `
[worktree]
default_location = "subdirectory"
auto_cleanup = false
`
	configPath := filepath.Join(tmpDir, "config.toml")
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Test parsing
	var config UserConfig
	_, err := toml.DecodeFile(configPath, &config)
	if err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}

	if config.Worktree.DefaultLocation != "subdirectory" {
		t.Errorf("Expected DefaultLocation 'subdirectory', got %q", config.Worktree.DefaultLocation)
	}
	if config.Worktree.AutoCleanup {
		t.Error("Expected AutoCleanup to be false")
	}
}

func TestWorktreeConfigDefaults(t *testing.T) {
	// Config without worktree section should parse with zero values
	// (defaults are applied by GetWorktreeSettings, not parsing)
	tmpDir := t.TempDir()
	configContent := `default_tool = "claude"`
	configPath := filepath.Join(tmpDir, "config.toml")
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	var config UserConfig
	_, err := toml.DecodeFile(configPath, &config)
	if err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}

	// When parsing directly without GetWorktreeSettings, values should be zero
	if config.Worktree.DefaultLocation != "" {
		t.Errorf("Expected empty DefaultLocation (zero value), got %q", config.Worktree.DefaultLocation)
	}
	if config.Worktree.AutoCleanup {
		t.Error("AutoCleanup should be false when not specified (zero value)")
	}
}

func TestGetWorktreeSettings(t *testing.T) {
	// Setup: use temp directory with no config
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)
	ClearUserConfigCache()

	settings := GetWorktreeSettings()
	if settings.DefaultLocation != "subdirectory" {
		t.Errorf("GetWorktreeSettings DefaultLocation: got %q, want %q", settings.DefaultLocation, "subdirectory")
	}
	if !settings.AutoCleanup {
		t.Error("GetWorktreeSettings AutoCleanup: should default to true")
	}
}

func TestGetWorktreeSettings_FromConfig(t *testing.T) {
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)
	ClearUserConfigCache()

	// Create config with custom worktree settings
	agentDeckDir := filepath.Join(tempDir, ".agent-deck")
	_ = os.MkdirAll(agentDeckDir, 0700)
	config := &UserConfig{
		Worktree: WorktreeSettings{
			DefaultLocation: "subdirectory",
			AutoCleanup:     false,
		},
	}
	_ = SaveUserConfig(config)
	ClearUserConfigCache()

	settings := GetWorktreeSettings()
	if settings.DefaultLocation != "subdirectory" {
		t.Errorf("GetWorktreeSettings DefaultLocation: got %q, want %q", settings.DefaultLocation, "subdirectory")
	}
	if settings.AutoCleanup {
		t.Error("GetWorktreeSettings AutoCleanup: should be false from config")
	}
}

// ============================================================================
// Preview Settings Tests
// ============================================================================

func TestPreviewSettings(t *testing.T) {
	// Create temp config
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	// Write config with preview settings
	content := `
[preview]
show_output = true
show_analytics = false
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	var config UserConfig
	_, err := toml.DecodeFile(configPath, &config)
	if err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}

	if config.Preview.ShowOutput == nil || !*config.Preview.ShowOutput {
		t.Error("Expected Preview.ShowOutput to be true")
	}
	if config.Preview.ShowAnalytics == nil {
		t.Error("Expected Preview.ShowAnalytics to be set")
	} else if *config.Preview.ShowAnalytics {
		t.Error("Expected Preview.ShowAnalytics to be false")
	}
}

func TestPreviewSettingsDefaults(t *testing.T) {
	cfg := &UserConfig{}

	// Default: output ON, analytics OFF
	if !cfg.GetShowOutput() {
		t.Error("GetShowOutput should default to true")
	}
	if cfg.GetShowAnalytics() {
		t.Error("GetShowAnalytics should default to false")
	}
}

func TestPreviewSettingsExplicitTrue(t *testing.T) {
	// Test when analytics is explicitly set to true
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	content := `
[preview]
show_output = false
show_analytics = true
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	var config UserConfig
	_, err := toml.DecodeFile(configPath, &config)
	if err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}

	if config.GetShowOutput() {
		t.Error("GetShowOutput should be false")
	}
	if !config.GetShowAnalytics() {
		t.Error("GetShowAnalytics should be true when explicitly set")
	}
}

func TestPreviewSettingsNotSet(t *testing.T) {
	// Test when preview section exists but analytics is not set
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	content := `
[preview]
show_output = true
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	var config UserConfig
	_, err := toml.DecodeFile(configPath, &config)
	if err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}

	if !config.GetShowOutput() {
		t.Error("GetShowOutput should be true")
	}
	// When not set, ShowAnalytics should default to false
	if config.GetShowAnalytics() {
		t.Error("GetShowAnalytics should default to false when not set")
	}
}

func TestGetPreviewSettings(t *testing.T) {
	// Setup: use temp directory with no config
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)
	ClearUserConfigCache()

	// With no config, should return defaults (output true, analytics false)
	settings := GetPreviewSettings()
	if !settings.GetShowOutput() {
		t.Error("GetPreviewSettings ShowOutput: should default to true")
	}
	if settings.GetShowAnalytics() {
		t.Error("GetPreviewSettings ShowAnalytics: should default to false")
	}
}

func TestGetPreviewSettings_FromConfig(t *testing.T) {
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)
	ClearUserConfigCache()

	// Create config with custom preview settings
	agentDeckDir := filepath.Join(tempDir, ".agent-deck")
	_ = os.MkdirAll(agentDeckDir, 0700)

	// Write config directly to test explicit false
	configPath := filepath.Join(agentDeckDir, "config.toml")
	content := `
[preview]
show_output = true
show_analytics = false
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	ClearUserConfigCache()

	settings := GetPreviewSettings()
	if !settings.GetShowOutput() {
		t.Error("GetPreviewSettings ShowOutput: should be true from config")
	}
	if settings.GetShowAnalytics() {
		t.Error("GetPreviewSettings ShowAnalytics: should be false from config")
	}
}

// ============================================================================
// Notifications Settings Tests
// ============================================================================

func TestNotificationsConfig_Defaults(t *testing.T) {
	// Test that default values are applied when section not present
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)
	ClearUserConfigCache()

	// With no config file, GetNotificationsSettings should return defaults
	settings := GetNotificationsSettings()
	if !settings.Enabled {
		t.Error("notifications should be enabled by default")
	}
	if settings.MaxShown != 6 {
		t.Errorf("max_shown should default to 6, got %d", settings.MaxShown)
	}
}

func TestNotificationsConfig_FromTOML(t *testing.T) {
	// Test parsing explicit TOML config
	tmpDir := t.TempDir()
	configContent := `
[notifications]
enabled = true
max_shown = 4
`
	configPath := filepath.Join(tmpDir, "config.toml")
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	var config UserConfig
	_, err := toml.DecodeFile(configPath, &config)
	if err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}

	if !config.Notifications.Enabled {
		t.Error("Expected Notifications.Enabled to be true")
	}
	if config.Notifications.MaxShown != 4 {
		t.Errorf("Expected MaxShown 4, got %d", config.Notifications.MaxShown)
	}
}

func TestGetNotificationsSettings(t *testing.T) {
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)
	ClearUserConfigCache()

	// Create config with custom notification settings
	agentDeckDir := filepath.Join(tempDir, ".agent-deck")
	_ = os.MkdirAll(agentDeckDir, 0700)

	configPath := filepath.Join(agentDeckDir, "config.toml")
	content := `
[notifications]
enabled = true
max_shown = 8
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	ClearUserConfigCache()

	settings := GetNotificationsSettings()
	if !settings.Enabled {
		t.Error("GetNotificationsSettings Enabled: should be true from config")
	}
	if settings.MaxShown != 8 {
		t.Errorf("GetNotificationsSettings MaxShown: got %d, want 8", settings.MaxShown)
	}
}

func TestClaudeSettings_AllowDangerousMode_TOML(t *testing.T) {
	tmpDir := t.TempDir()
	configContent := `
[claude]
dangerous_mode = false
allow_dangerous_mode = true
`
	configPath := filepath.Join(tmpDir, "config.toml")
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	var config UserConfig
	_, err := toml.DecodeFile(configPath, &config)
	if err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}

	if config.Claude.GetDangerousMode() {
		t.Error("Expected dangerous_mode false")
	}
	if !config.Claude.AllowDangerousMode {
		t.Error("Expected allow_dangerous_mode true")
	}
}

func TestClaudeSettings_AllowDangerousMode_Default(t *testing.T) {
	var config UserConfig
	if config.Claude.AllowDangerousMode {
		t.Error("allow_dangerous_mode should default to false")
	}
}

func TestGetNotificationsSettings_PartialConfig(t *testing.T) {
	// Test that missing fields get defaults
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)
	ClearUserConfigCache()

	agentDeckDir := filepath.Join(tempDir, ".agent-deck")
	_ = os.MkdirAll(agentDeckDir, 0700)

	// Config with only enabled set, max_shown should get default
	configPath := filepath.Join(agentDeckDir, "config.toml")
	content := `
[notifications]
enabled = true
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	ClearUserConfigCache()

	settings := GetNotificationsSettings()
	if !settings.Enabled {
		t.Error("GetNotificationsSettings Enabled: should be true")
	}
	if settings.MaxShown != 6 {
		t.Errorf("GetNotificationsSettings MaxShown: should default to 6, got %d", settings.MaxShown)
	}
}

func TestGetTmuxSettings_InjectStatusLine_Default(t *testing.T) {
	// Default (no config) should return true
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)
	ClearUserConfigCache()

	agentDeckDir := filepath.Join(tempDir, ".agent-deck")
	_ = os.MkdirAll(agentDeckDir, 0700)

	// Empty config file
	configPath := filepath.Join(agentDeckDir, "config.toml")
	if err := os.WriteFile(configPath, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	ClearUserConfigCache()

	settings := GetTmuxSettings()
	if !settings.GetInjectStatusLine() {
		t.Error("GetInjectStatusLine should default to true when not set")
	}
}

func TestGetTmuxSettings_InjectStatusLine_False(t *testing.T) {
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)
	ClearUserConfigCache()

	agentDeckDir := filepath.Join(tempDir, ".agent-deck")
	_ = os.MkdirAll(agentDeckDir, 0700)

	configPath := filepath.Join(agentDeckDir, "config.toml")
	configContent := `
[tmux]
inject_status_line = false
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	ClearUserConfigCache()

	settings := GetTmuxSettings()
	if settings.GetInjectStatusLine() {
		t.Error("GetInjectStatusLine should be false when set to false")
	}
}

func TestGetTmuxSettings_InjectStatusLine_True(t *testing.T) {
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)
	ClearUserConfigCache()

	agentDeckDir := filepath.Join(tempDir, ".agent-deck")
	_ = os.MkdirAll(agentDeckDir, 0700)

	configPath := filepath.Join(agentDeckDir, "config.toml")
	configContent := `
[tmux]
inject_status_line = true
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	ClearUserConfigCache()

	settings := GetTmuxSettings()
	if !settings.GetInjectStatusLine() {
		t.Error("GetInjectStatusLine should be true when set to true")
	}
}

// ============================================================================
// Vagrant Settings Tests
// ============================================================================

func TestGetVagrantSettingsDefaults(t *testing.T) {
	// Test that default values are applied when section not present
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)
	ClearUserConfigCache()

	// With no config file, GetVagrantSettings should return defaults
	settings := GetVagrantSettings()
	if settings.MemoryMB != 4096 {
		t.Errorf("MemoryMB should default to 4096, got %d", settings.MemoryMB)
	}
	if settings.CPUs != 2 {
		t.Errorf("CPUs should default to 2, got %d", settings.CPUs)
	}
	if settings.Box != "bento/ubuntu-24.04" {
		t.Errorf("Box should default to bento/ubuntu-24.04, got %s", settings.Box)
	}
	if settings.AutoSuspend == nil || !*settings.AutoSuspend {
		t.Error("AutoSuspend should default to true")
	}
	if settings.AutoDestroy {
		t.Error("AutoDestroy should default to false")
	}
	if settings.HostGatewayIP != "10.0.2.2" {
		t.Errorf("HostGatewayIP should default to 10.0.2.2, got %s", settings.HostGatewayIP)
	}
	if settings.SyncedFolderType != "virtualbox" {
		t.Errorf("SyncedFolderType should default to virtualbox, got %s", settings.SyncedFolderType)
	}
	if settings.HealthCheckInterval != 30 {
		t.Errorf("HealthCheckInterval should default to 30, got %d", settings.HealthCheckInterval)
	}
	if settings.ForwardProxyEnv == nil || !*settings.ForwardProxyEnv {
		t.Error("ForwardProxyEnv should default to true")
	}
	if settings.ProvisionPackages != nil {
		t.Errorf("ProvisionPackages should be nil, got %v", settings.ProvisionPackages)
	}
	if settings.ProvisionPkgExclude != nil {
		t.Errorf("ProvisionPkgExclude should be nil, got %v", settings.ProvisionPkgExclude)
	}
	if settings.NpmPackages != nil {
		t.Errorf("NpmPackages should be nil, got %v", settings.NpmPackages)
	}
	if settings.ProvisionScript != "" {
		t.Errorf("ProvisionScript should be empty, got %s", settings.ProvisionScript)
	}
	if settings.Vagrantfile != "" {
		t.Errorf("Vagrantfile should be empty, got %s", settings.Vagrantfile)
	}
	if settings.PortForwards != nil {
		t.Errorf("PortForwards should be nil, got %v", settings.PortForwards)
	}
	if settings.Env != nil {
		t.Errorf("Env should be nil, got %v", settings.Env)
	}
}

func TestGetVagrantSettingsOverrides(t *testing.T) {
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)
	ClearUserConfigCache()

	// Create config with custom vagrant settings
	agentDeckDir := filepath.Join(tempDir, ".agent-deck")
	_ = os.MkdirAll(agentDeckDir, 0700)

	configPath := filepath.Join(agentDeckDir, "config.toml")
	configContent := `
[vagrant]
memory_mb = 8192
cpus = 4
box = "ubuntu/jammy64"
auto_suspend = false
auto_destroy = true
host_gateway_ip = "192.168.1.1"
synced_folder_type = "nfs"
provision_packages = ["vim", "htop"]
provision_packages_exclude = ["nano"]
npm_packages = ["typescript", "eslint"]
provision_script = "/path/to/script.sh"
vagrantfile = "/path/to/Vagrantfile"
health_check_interval = 60
forward_proxy_env = false

[[vagrant.port_forwards]]
guest = 3000
host = 3000
protocol = "tcp"

[[vagrant.port_forwards]]
guest = 8080
host = 8080
protocol = "tcp"

[vagrant.env]
FOO = "bar"
BAZ = "qux"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	ClearUserConfigCache()

	settings := GetVagrantSettings()
	if settings.MemoryMB != 8192 {
		t.Errorf("MemoryMB: got %d, want 8192", settings.MemoryMB)
	}
	if settings.CPUs != 4 {
		t.Errorf("CPUs: got %d, want 4", settings.CPUs)
	}
	if settings.Box != "ubuntu/jammy64" {
		t.Errorf("Box: got %s, want ubuntu/jammy64", settings.Box)
	}
	if settings.AutoSuspend == nil || *settings.AutoSuspend {
		t.Error("AutoSuspend should be false")
	}
	if !settings.AutoDestroy {
		t.Error("AutoDestroy should be true")
	}
	if settings.HostGatewayIP != "192.168.1.1" {
		t.Errorf("HostGatewayIP: got %s, want 192.168.1.1", settings.HostGatewayIP)
	}
	if settings.SyncedFolderType != "nfs" {
		t.Errorf("SyncedFolderType: got %s, want nfs", settings.SyncedFolderType)
	}
	if len(settings.ProvisionPackages) != 2 || settings.ProvisionPackages[0] != "vim" || settings.ProvisionPackages[1] != "htop" {
		t.Errorf("ProvisionPackages: got %v, want [vim, htop]", settings.ProvisionPackages)
	}
	if len(settings.ProvisionPkgExclude) != 1 || settings.ProvisionPkgExclude[0] != "nano" {
		t.Errorf("ProvisionPkgExclude: got %v, want [nano]", settings.ProvisionPkgExclude)
	}
	if len(settings.NpmPackages) != 2 || settings.NpmPackages[0] != "typescript" || settings.NpmPackages[1] != "eslint" {
		t.Errorf("NpmPackages: got %v, want [typescript, eslint]", settings.NpmPackages)
	}
	if settings.ProvisionScript != "/path/to/script.sh" {
		t.Errorf("ProvisionScript: got %s, want /path/to/script.sh", settings.ProvisionScript)
	}
	if settings.Vagrantfile != "/path/to/Vagrantfile" {
		t.Errorf("Vagrantfile: got %s, want /path/to/Vagrantfile", settings.Vagrantfile)
	}
	if settings.HealthCheckInterval != 60 {
		t.Errorf("HealthCheckInterval: got %d, want 60", settings.HealthCheckInterval)
	}
	if settings.ForwardProxyEnv == nil || *settings.ForwardProxyEnv {
		t.Error("ForwardProxyEnv should be false")
	}
	if len(settings.PortForwards) != 2 {
		t.Errorf("PortForwards length: got %d, want 2", len(settings.PortForwards))
	} else {
		if settings.PortForwards[0].Guest != 3000 || settings.PortForwards[0].Host != 3000 || settings.PortForwards[0].Protocol != "tcp" {
			t.Errorf("PortForwards[0]: got %+v, want {Guest:3000 Host:3000 Protocol:tcp}", settings.PortForwards[0])
		}
		if settings.PortForwards[1].Guest != 8080 || settings.PortForwards[1].Host != 8080 || settings.PortForwards[1].Protocol != "tcp" {
			t.Errorf("PortForwards[1]: got %+v, want {Guest:8080 Host:8080 Protocol:tcp}", settings.PortForwards[1])
		}
	}
	if len(settings.Env) != 2 || settings.Env["FOO"] != "bar" || settings.Env["BAZ"] != "qux" {
		t.Errorf("Env: got %v, want map[FOO:bar BAZ:qux]", settings.Env)
	}
}

func TestVagrantConfig_TOML(t *testing.T) {
	// Test parsing explicit TOML config
	tmpDir := t.TempDir()
	configContent := `
[vagrant]
memory_mb = 2048
cpus = 1
box = "debian/bullseye64"
`
	configPath := filepath.Join(tmpDir, "config.toml")
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	var config UserConfig
	_, err := toml.DecodeFile(configPath, &config)
	if err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}

	if config.Vagrant.MemoryMB != 2048 {
		t.Errorf("Expected MemoryMB 2048, got %d", config.Vagrant.MemoryMB)
	}
	if config.Vagrant.CPUs != 1 {
		t.Errorf("Expected CPUs 1, got %d", config.Vagrant.CPUs)
	}
	if config.Vagrant.Box != "debian/bullseye64" {
		t.Errorf("Expected Box debian/bullseye64, got %s", config.Vagrant.Box)
	}
}
