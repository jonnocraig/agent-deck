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

// TestGetVagrantSudoSkill verifies the skill content contains expected keywords
func TestGetVagrantSudoSkill(t *testing.T) {
	skill := GetVagrantSudoSkill()

	if skill == "" {
		t.Fatal("GetVagrantSudoSkill returned empty string")
	}

	// Check for key concepts from the enhanced "operating-in-vagrant" skill
	requiredKeywords := []string{
		"sudo",
		"/vagrant",
		"Ubuntu 24.04",
		"Docker",
		"isolated",
		"10.0.2.2",       // host access via NAT gateway
		"disposable",      // ephemeral mindset
		"Be bold",         // supercharged mindset
		"unrestricted",    // full capabilities
	}

	for _, keyword := range requiredKeywords {
		if !strings.Contains(skill, keyword) {
			t.Errorf("Skill content missing keyword: %q", keyword)
		}
	}
}

// TestSudoSkillMentionsInotify verifies the skill mentions file watcher issues and polling mode
func TestSudoSkillMentionsInotify(t *testing.T) {
	skill := GetVagrantSudoSkill()

	// Should mention inotify/file watcher issues
	if !strings.Contains(skill, "inotify") && !strings.Contains(skill, "file watcher") && !strings.Contains(skill, "CHOKIDAR_USEPOLLING") {
		t.Error("Skill should mention inotify/file watcher issues and polling mode workarounds")
	}

	// Should mention polling mode solutions
	pollingKeywords := []string{"CHOKIDAR_USEPOLLING", "WATCHPACK_POLLING", "fallbackPolling"}
	foundPolling := false
	for _, keyword := range pollingKeywords {
		if strings.Contains(skill, keyword) {
			foundPolling = true
			break
		}
	}
	if !foundPolling {
		t.Error("Skill should mention polling mode solutions (CHOKIDAR_USEPOLLING, WATCHPACK_POLLING, or fallbackPolling)")
	}
}

// TestSudoSkillMentionsCredentialWarning verifies the skill warns about credential files
func TestSudoSkillMentionsCredentialWarning(t *testing.T) {
	skill := GetVagrantSudoSkill()

	// Should mention credential files explicitly
	credentialKeywords := []string{".env", "credentials", "NEVER read", "NEVER cat", "NEVER print"}
	foundWarning := false
	for _, keyword := range credentialKeywords {
		if strings.Contains(skill, keyword) {
			foundWarning = true
			break
		}
	}

	if !foundWarning {
		t.Error("Skill should warn about never reading credential files")
	}
}

// TestEnsureSudoSkill verifies the skill file is written correctly
func TestEnsureSudoSkill(t *testing.T) {
	tmpDir := t.TempDir()
	settings := session.VagrantSettings{}
	mgr := NewManager(tmpDir, settings)

	err := mgr.EnsureSudoSkill()
	if err != nil {
		t.Fatalf("EnsureSudoSkill failed: %v", err)
	}

	// Verify file exists at correct path
	skillPath := filepath.Join(tmpDir, ".claude", "skills", "operating-in-vagrant.md")
	content, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("Failed to read skill file: %v", err)
	}

	// Verify content matches GetVagrantSudoSkill()
	expected := GetVagrantSudoSkill()
	if string(content) != expected {
		t.Errorf("Skill file content doesn't match GetVagrantSudoSkill()\nGot:\n%s\n\nExpected:\n%s", content, expected)
	}
}

// TestEnsureSudoSkillIdempotent verifies calling EnsureSudoSkill twice doesn't error
func TestEnsureSudoSkillIdempotent(t *testing.T) {
	tmpDir := t.TempDir()
	settings := session.VagrantSettings{}
	mgr := NewManager(tmpDir, settings)

	// First call
	if err := mgr.EnsureSudoSkill(); err != nil {
		t.Fatalf("First EnsureSudoSkill failed: %v", err)
	}

	// Second call should not error
	if err := mgr.EnsureSudoSkill(); err != nil {
		t.Fatalf("Second EnsureSudoSkill failed (not idempotent): %v", err)
	}

	// Verify content is still correct
	skillPath := filepath.Join(tmpDir, ".claude", "skills", "operating-in-vagrant.md")
	content, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("Failed to read skill file: %v", err)
	}

	expected := GetVagrantSudoSkill()
	if string(content) != expected {
		t.Error("Skill file content changed after second call")
	}
}

// TestEnsureSudoSkillInjectsCredentialGuard verifies that EnsureSudoSkill
// also creates the credential guard hook in settings.local.json.
func TestEnsureSudoSkillInjectsCredentialGuard(t *testing.T) {
	tmpDir := t.TempDir()
	settings := session.VagrantSettings{}
	mgr := NewManager(tmpDir, settings)

	if err := mgr.EnsureSudoSkill(); err != nil {
		t.Fatalf("EnsureSudoSkill failed: %v", err)
	}

	// Verify skill file exists
	skillPath := filepath.Join(tmpDir, ".claude", "skills", "operating-in-vagrant.md")
	if _, err := os.Stat(skillPath); os.IsNotExist(err) {
		t.Fatal("Skill file was not created")
	}

	// Verify settings.local.json exists with credential guard hook
	settingsPath := filepath.Join(tmpDir, ".claude", "settings.local.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("settings.local.json was not created: %v", err)
	}

	var settings2 map[string]interface{}
	if err := json.Unmarshal(data, &settings2); err != nil {
		t.Fatalf("Failed to parse settings.local.json: %v", err)
	}

	hooks, ok := settings2["hooks"].(map[string]interface{})
	if !ok {
		t.Fatal("settings.local.json missing 'hooks' key")
	}

	preToolUse, ok := hooks["PreToolUse"].([]interface{})
	if !ok || len(preToolUse) == 0 {
		t.Fatal("settings.local.json missing 'PreToolUse' hook array")
	}

	// Verify the hook matches Read|View|Cat
	hookEntry, ok := preToolUse[0].(map[string]interface{})
	if !ok {
		t.Fatal("PreToolUse hook entry is not a map")
	}
	matcher, ok := hookEntry["matcher"].(string)
	if !ok || matcher != "Read|View|Cat" {
		t.Errorf("Expected matcher 'Read|View|Cat', got %q", matcher)
	}
}

// TestEnsureSudoSkillHasFrontmatter verifies the skill includes YAML frontmatter
// so Claude Code recognizes it as a proper skill.
func TestEnsureSudoSkillHasFrontmatter(t *testing.T) {
	skill := GetVagrantSudoSkill()

	if !strings.HasPrefix(skill, "---\n") {
		t.Fatal("Skill should start with YAML frontmatter delimiter '---'")
	}

	if !strings.Contains(skill, "name: operating-in-vagrant") {
		t.Error("Skill frontmatter should contain 'name: operating-in-vagrant'")
	}

	if !strings.Contains(skill, "description:") {
		t.Error("Skill frontmatter should contain a 'description:' field")
	}

	// Verify closing frontmatter delimiter
	parts := strings.SplitN(skill, "---", 3)
	if len(parts) < 3 {
		t.Fatal("Skill should have opening and closing '---' frontmatter delimiters")
	}
}

// TestGetCredentialGuardHook verifies the hook structure
func TestGetCredentialGuardHook(t *testing.T) {
	hook := GetCredentialGuardHook()

	// Marshal to JSON to verify structure
	jsonData, err := json.Marshal(hook)
	if err != nil {
		t.Fatalf("Failed to marshal hook to JSON: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(jsonData, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal hook JSON: %v", err)
	}

	// Verify hooks key exists
	hooks, ok := parsed["hooks"].(map[string]interface{})
	if !ok {
		t.Fatal("Hook structure missing 'hooks' key")
	}

	// Verify PreToolUse exists
	preToolUse, ok := hooks["PreToolUse"].([]interface{})
	if !ok || len(preToolUse) == 0 {
		t.Fatal("Hook structure missing 'PreToolUse' array")
	}

	// Verify matcher field exists
	firstHook, ok := preToolUse[0].(map[string]interface{})
	if !ok {
		t.Fatal("PreToolUse hook is not an object")
	}

	matcher, ok := firstHook["matcher"].(string)
	if !ok {
		t.Fatal("PreToolUse hook missing 'matcher' field")
	}

	// Matcher should match Read, View, or Cat tools
	if !strings.Contains(matcher, "Read") {
		t.Error("Matcher should include 'Read' tool")
	}
}

// TestCredentialGuardHookBlocksEnvFile tests the grep pattern matches .env files
func TestCredentialGuardHookBlocksEnvFile(t *testing.T) {
	testPaths := []string{
		"/vagrant/.env",
		"/vagrant/backend/.env.local",
		"/home/vagrant/.env",
		".env",
	}

	for _, path := range testPaths {
		t.Run(path, func(t *testing.T) {
			// Extract the command from the hook
			hook := GetCredentialGuardHook()
			cmd := extractHookCommand(t, hook)

			// Test if the pattern matches
			if !testCredentialPattern(t, cmd, path) {
				t.Errorf("Pattern should block credential file: %s", path)
			}
		})
	}
}

// TestCredentialGuardHookBlocksSSHKey tests the pattern matches SSH keys
func TestCredentialGuardHookBlocksSSHKey(t *testing.T) {
	testPaths := []string{
		"/home/vagrant/.ssh/id_rsa",
		"/home/vagrant/.ssh/id_ed25519",
		"/vagrant/keys/server.key",
		"/vagrant/certs/private.pem",
	}

	for _, path := range testPaths {
		t.Run(path, func(t *testing.T) {
			hook := GetCredentialGuardHook()
			cmd := extractHookCommand(t, hook)

			if !testCredentialPattern(t, cmd, path) {
				t.Errorf("Pattern should block SSH key/cert file: %s", path)
			}
		})
	}
}

// TestCredentialGuardHookAllowsNormalFiles tests that normal files are not blocked
func TestCredentialGuardHookAllowsNormalFiles(t *testing.T) {
	testPaths := []string{
		"/vagrant/main.go",
		"/vagrant/README.md",
		"/vagrant/config.toml",
		"/vagrant/package.json",
	}

	for _, path := range testPaths {
		t.Run(path, func(t *testing.T) {
			hook := GetCredentialGuardHook()
			cmd := extractHookCommand(t, hook)

			if testCredentialPattern(t, cmd, path) {
				t.Errorf("Pattern should NOT block normal file: %s", path)
			}
		})
	}
}

// TestInjectCredentialGuardHook tests creating new settings.local.json
func TestInjectCredentialGuardHook(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, ".claude", "settings.local.json")

	err := InjectCredentialGuardHook(tmpDir)
	if err != nil {
		t.Fatalf("InjectCredentialGuardHook failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		t.Fatal("settings.local.json was not created")
	}

	// Read and parse the file
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("Failed to read settings.local.json: %v", err)
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("Failed to parse settings.local.json: %v", err)
	}

	// Verify hooks exist
	hooks, ok := settings["hooks"].(map[string]interface{})
	if !ok {
		t.Fatal("settings.local.json missing 'hooks' key")
	}

	preToolUse, ok := hooks["PreToolUse"].([]interface{})
	if !ok || len(preToolUse) == 0 {
		t.Fatal("settings.local.json missing 'PreToolUse' hooks")
	}
}

// TestInjectCredentialGuardHookMerge tests merging with existing settings
func TestInjectCredentialGuardHookMerge(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	settingsPath := filepath.Join(claudeDir, "settings.local.json")

	// Create directory
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("Failed to create .claude dir: %v", err)
	}

	// Write existing settings with other config
	existing := map[string]interface{}{
		"apiKey": "test-key",
		"theme":  "dark",
		"hooks": map[string]interface{}{
			"PostToolUse": []interface{}{
				map[string]interface{}{
					"matcher": "Edit",
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": "prettier --write",
						},
					},
				},
			},
		},
	}

	data, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal existing settings: %v", err)
	}

	if err := os.WriteFile(settingsPath, data, 0644); err != nil {
		t.Fatalf("Failed to write existing settings: %v", err)
	}

	// Inject credential guard hook
	if err := InjectCredentialGuardHook(tmpDir); err != nil {
		t.Fatalf("InjectCredentialGuardHook failed: %v", err)
	}

	// Read and verify merged settings
	data, err = os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("Failed to read merged settings: %v", err)
	}

	var merged map[string]interface{}
	if err := json.Unmarshal(data, &merged); err != nil {
		t.Fatalf("Failed to parse merged settings: %v", err)
	}

	// Verify existing settings are preserved
	if merged["apiKey"] != "test-key" {
		t.Error("Existing apiKey was not preserved")
	}
	if merged["theme"] != "dark" {
		t.Error("Existing theme was not preserved")
	}

	// Verify hooks were merged
	hooks, ok := merged["hooks"].(map[string]interface{})
	if !ok {
		t.Fatal("Merged settings missing 'hooks' key")
	}

	// Verify PostToolUse hook is preserved
	postToolUse, ok := hooks["PostToolUse"].([]interface{})
	if !ok || len(postToolUse) == 0 {
		t.Error("Existing PostToolUse hook was not preserved")
	}

	// Verify PreToolUse hook was added
	preToolUse, ok := hooks["PreToolUse"].([]interface{})
	if !ok || len(preToolUse) == 0 {
		t.Error("PreToolUse hook was not added")
	}
}

// Helper function to extract the command string from the hook structure
func extractHookCommand(t *testing.T, hook map[string]interface{}) string {
	t.Helper()

	hooks, ok := hook["hooks"].(map[string]interface{})
	if !ok {
		t.Fatal("Invalid hook structure: missing 'hooks' key")
	}

	preToolUse, ok := hooks["PreToolUse"].([]interface{})
	if !ok || len(preToolUse) == 0 {
		t.Fatal("Invalid hook structure: missing 'PreToolUse' array")
	}

	firstMatcher, ok := preToolUse[0].(map[string]interface{})
	if !ok {
		t.Fatal("Invalid hook structure: PreToolUse[0] is not an object")
	}

	hooksList, ok := firstMatcher["hooks"].([]interface{})
	if !ok || len(hooksList) == 0 {
		t.Fatal("Invalid hook structure: missing nested 'hooks' array")
	}

	hookDef, ok := hooksList[0].(map[string]interface{})
	if !ok {
		t.Fatal("Invalid hook structure: hook definition is not an object")
	}

	command, ok := hookDef["command"].(string)
	if !ok {
		t.Fatal("Invalid hook structure: missing 'command' string")
	}

	return command
}

// Helper function to test if the credential pattern matches a given file path
func testCredentialPattern(t *testing.T, hookCommand string, filePath string) bool {
	t.Helper()

	// The hook command uses CLAUDE_TOOL_ARG_file_path environment variable
	// We'll execute the bash command with this env var set
	cmd := exec.Command("bash", "-c", hookCommand)
	cmd.Env = append(os.Environ(), "CLAUDE_TOOL_ARG_file_path="+filePath)

	err := cmd.Run()
	// If exit code is 1, the pattern matched (file is blocked)
	// If exit code is 0, the pattern didn't match (file is allowed)
	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode() == 1
	}

	// No error means exit code 0 (file is allowed)
	return false
}
