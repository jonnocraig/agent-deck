package vagrant

import (
	"archive/tar"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// readTarEntries extracts all entries from a tar archive into a map of name -> content.
// Regular files have their content as the value; directory entries have nil.
func readTarEntries(t *testing.T, data []byte) map[string][]byte {
	t.Helper()
	entries := make(map[string][]byte)
	tr := tar.NewReader(bytes.NewReader(data))
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("error reading tar: %v", err)
		}
		if hdr.Typeflag == tar.TypeReg {
			content, err := io.ReadAll(tr)
			if err != nil {
				t.Fatalf("error reading tar entry %s: %v", hdr.Name, err)
			}
			entries[hdr.Name] = content
		} else {
			entries[hdr.Name] = nil // directory entry
		}
	}
	return entries
}

// readTarHeaders extracts all headers from a tar archive for type-flag inspection.
func readTarHeaders(t *testing.T, data []byte) []*tar.Header {
	t.Helper()
	var headers []*tar.Header
	tr := tar.NewReader(bytes.NewReader(data))
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("error reading tar header: %v", err)
		}
		headers = append(headers, hdr)
	}
	return headers
}

// setupProfileDir creates a ~/.claude/ directory tree inside the given homeDir
// with the specified file map. Keys are relative paths from homeDir (e.g.
// ".claude/skills/my-skill/SKILL.md") and values are file content.
func setupProfileDir(t *testing.T, homeDir string, files map[string]string) {
	t.Helper()
	for relPath, content := range files {
		absPath := filepath.Join(homeDir, relPath)
		if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
			t.Fatalf("failed to create dir for %s: %v", relPath, err)
		}
		if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", relPath, err)
		}
	}
}

func TestCreateProfileTar_BasicStructure(t *testing.T) {
	homeDir := t.TempDir()

	files := map[string]string{
		".claude/skills/test-skill/SKILL.md":      "# Test Skill\nHello from test skill.",
		".claude/commands/test-cmd/command.md":     "# Test Command\nRun something.",
		".claude/agents/test-agent/agent.md":       "# Test Agent\nI am an agent.",
		".claude/rules/coding-style.md":            "# Coding Style\nUse gofmt.",
	}
	setupProfileDir(t, homeDir, files)

	data, err := createProfileTar(homeDir)
	if err != nil {
		t.Fatalf("createProfileTar returned error: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty tar data")
	}

	entries := readTarEntries(t, data)

	// Verify all 4 directories have entries with correct relative paths
	expected := map[string]string{
		"skills/test-skill/SKILL.md":  "# Test Skill\nHello from test skill.",
		"commands/test-cmd/command.md": "# Test Command\nRun something.",
		"agents/test-agent/agent.md":  "# Test Agent\nI am an agent.",
		"rules/coding-style.md":       "# Coding Style\nUse gofmt.",
	}

	for name, wantContent := range expected {
		got, ok := entries[name]
		if !ok {
			t.Errorf("tar missing expected entry %q", name)
			continue
		}
		if got != nil && string(got) != wantContent {
			t.Errorf("tar entry %q content mismatch:\ngot:  %s\nwant: %s", name, string(got), wantContent)
		}
	}
}

func TestCreateProfileTar_DereferencesFileSymlinks(t *testing.T) {
	// Resolve through EvalSymlinks to handle macOS /var -> /private/var
	rawHomeDir := t.TempDir()
	homeDir, err := filepath.EvalSymlinks(rawHomeDir)
	if err != nil {
		t.Fatalf("failed to resolve temp dir: %v", err)
	}

	// Create the real file in ~/.claude/ (a valid allowed path for symlink validation)
	realFileDir := filepath.Join(homeDir, ".claude", "agents", "shared")
	if err := os.MkdirAll(realFileDir, 0755); err != nil {
		t.Fatalf("failed to create shared dir: %v", err)
	}
	realContent := "# Shared Helpers\nShared content dereferenced via symlink."
	realFile := filepath.Join(realFileDir, "helpers.md")
	if err := os.WriteFile(realFile, []byte(realContent), 0644); err != nil {
		t.Fatalf("failed to write real file: %v", err)
	}

	// Create a skill directory with a file-level symlink pointing to the real file
	skillDir := filepath.Join(homeDir, ".claude", "skills", "my-skill")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("failed to create skill dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# My Skill"), 0644); err != nil {
		t.Fatalf("failed to write SKILL.md: %v", err)
	}
	symlinkPath := filepath.Join(skillDir, "helpers.md")
	if err := os.Symlink(realFile, symlinkPath); err != nil {
		t.Fatalf("failed to create file symlink: %v", err)
	}

	data, err := createProfileTar(homeDir)
	if err != nil {
		t.Fatalf("createProfileTar returned error: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty tar data for symlink test")
	}

	entries := readTarEntries(t, data)

	// Verify the symlinked file appears as a regular file with correct content
	got, ok := entries["skills/my-skill/helpers.md"]
	if !ok {
		t.Fatal("tar missing expected entry skills/my-skill/helpers.md")
	}
	if string(got) != realContent {
		t.Errorf("symlinked file content mismatch:\ngot:  %s\nwant: %s", string(got), realContent)
	}

	// Verify the tar entry type is regular file, not symlink
	headers := readTarHeaders(t, data)
	for _, hdr := range headers {
		if hdr.Name == "skills/my-skill/helpers.md" {
			if hdr.Typeflag == tar.TypeSymlink || hdr.Typeflag == tar.TypeLink {
				t.Errorf("entry should be a regular file, not a symlink (typeflag=%d)", hdr.Typeflag)
			}
			if hdr.Typeflag != tar.TypeReg {
				t.Errorf("entry should be TypeReg (%d), got typeflag=%d", tar.TypeReg, hdr.Typeflag)
			}
		}
	}
}

func TestCreateProfileTar_SkipsDSStore(t *testing.T) {
	homeDir := t.TempDir()

	files := map[string]string{
		".claude/skills/my-skill/SKILL.md":  "# My Skill",
		".claude/skills/my-skill/.DS_Store": "binary garbage",
		".claude/skills/.DS_Store":          "more garbage",
		".claude/rules/.DS_Store":           "even more garbage",
		".claude/rules/style.md":            "# Style Rules",
	}
	setupProfileDir(t, homeDir, files)

	data, err := createProfileTar(homeDir)
	if err != nil {
		t.Fatalf("createProfileTar returned error: %v", err)
	}

	entries := readTarEntries(t, data)

	// .DS_Store files should not appear in the tar
	for name := range entries {
		if filepath.Base(name) == ".DS_Store" {
			t.Errorf("tar should not contain .DS_Store entry, found: %s", name)
		}
	}

	// Valid files should still be present
	if _, ok := entries["skills/my-skill/SKILL.md"]; !ok {
		t.Error("tar missing expected entry skills/my-skill/SKILL.md")
	}
	if _, ok := entries["rules/style.md"]; !ok {
		t.Error("tar missing expected entry rules/style.md")
	}
}

func TestCreateProfileTar_ExcludesCLAUDEMD(t *testing.T) {
	homeDir := t.TempDir()

	files := map[string]string{
		".claude/CLAUDE.md":                    "# Root CLAUDE.md - should be excluded",
		".claude/skills/my-skill/SKILL.md":     "# My Skill",
		".claude/skills/my-skill/CLAUDE.md":    "# Nested CLAUDE.md - also excluded",
		".claude/rules/coding.md":              "# Coding rules",
	}
	setupProfileDir(t, homeDir, files)

	data, err := createProfileTar(homeDir)
	if err != nil {
		t.Fatalf("createProfileTar returned error: %v", err)
	}

	entries := readTarEntries(t, data)

	// Root CLAUDE.md should be excluded
	if _, ok := entries["CLAUDE.md"]; ok {
		t.Error("tar should not contain root CLAUDE.md")
	}

	// Nested CLAUDE.md should also be excluded per isExcludedProfilePath
	if _, ok := entries["skills/my-skill/CLAUDE.md"]; ok {
		t.Error("tar should not contain nested CLAUDE.md")
	}

	// Valid files should still be present
	if _, ok := entries["skills/my-skill/SKILL.md"]; !ok {
		t.Error("tar missing expected entry skills/my-skill/SKILL.md")
	}
	if _, ok := entries["rules/coding.md"]; !ok {
		t.Error("tar missing expected entry rules/coding.md")
	}
}

func TestCreateProfileTar_ExcludesOperatingInVagrant(t *testing.T) {
	homeDir := t.TempDir()

	files := map[string]string{
		".claude/skills/operating-in-vagrant/SKILL.md": "# Operating in Vagrant",
		".claude/skills/other-skill/SKILL.md":          "# Other Skill",
	}
	setupProfileDir(t, homeDir, files)

	data, err := createProfileTar(homeDir)
	if err != nil {
		t.Fatalf("createProfileTar returned error: %v", err)
	}

	entries := readTarEntries(t, data)

	// operating-in-vagrant should be excluded
	for name := range entries {
		if filepath.Base(filepath.Dir(name)) == "operating-in-vagrant" || name == "skills/operating-in-vagrant" {
			t.Errorf("tar should not contain operating-in-vagrant entry, found: %s", name)
		}
	}

	// Other skill should still be present
	if _, ok := entries["skills/other-skill/SKILL.md"]; !ok {
		t.Error("tar missing expected entry skills/other-skill/SKILL.md")
	}
}

func TestCreateProfileTar_MissingDirsGraceful(t *testing.T) {
	homeDir := t.TempDir()

	// Only create skills/ and rules/, skip commands/ and agents/
	files := map[string]string{
		".claude/skills/my-skill/SKILL.md": "# My Skill",
		".claude/rules/style.md":           "# Style",
	}
	setupProfileDir(t, homeDir, files)

	data, err := createProfileTar(homeDir)
	if err != nil {
		t.Fatalf("createProfileTar should not error with missing dirs: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty tar data when some dirs exist")
	}

	entries := readTarEntries(t, data)

	if _, ok := entries["skills/my-skill/SKILL.md"]; !ok {
		t.Error("tar missing expected entry skills/my-skill/SKILL.md")
	}
	if _, ok := entries["rules/style.md"]; !ok {
		t.Error("tar missing expected entry rules/style.md")
	}

	// Should not have entries from missing dirs
	for name := range entries {
		if strings.HasPrefix(name, "commands/") || strings.HasPrefix(name, "agents/") {
			t.Errorf("tar should not have entries from missing dirs, found: %s", name)
		}
	}
}

func TestCreateProfileTar_AllDirsMissing(t *testing.T) {
	homeDir := t.TempDir()

	// Create the .claude dir but none of the profile subdirs
	if err := os.MkdirAll(filepath.Join(homeDir, ".claude"), 0755); err != nil {
		t.Fatalf("failed to create .claude dir: %v", err)
	}

	data, err := createProfileTar(homeDir)
	if err != nil {
		t.Fatalf("createProfileTar should not error with all dirs missing: %v", err)
	}
	if len(data) != 0 {
		t.Errorf("expected empty tar data when all dirs are missing, got %d bytes", len(data))
	}
}

func TestCreateProfileTar_NoClaudeDirAtAll(t *testing.T) {
	homeDir := t.TempDir()
	// No .claude dir at all

	data, err := createProfileTar(homeDir)
	if err != nil {
		t.Fatalf("createProfileTar should not error with no .claude dir: %v", err)
	}
	if len(data) != 0 {
		t.Errorf("expected empty tar data with no .claude dir, got %d bytes", len(data))
	}
}

func TestCreateProfileTar_BrokenSymlinkSkipped(t *testing.T) {
	homeDir := t.TempDir()

	// Create a valid skill
	files := map[string]string{
		".claude/skills/good-skill/SKILL.md": "# Good Skill",
	}
	setupProfileDir(t, homeDir, files)

	// Create a broken symlink in skills/
	skillsDir := filepath.Join(homeDir, ".claude", "skills")
	brokenTarget := filepath.Join(homeDir, "nonexistent", "path")
	brokenLink := filepath.Join(skillsDir, "broken-skill")
	if err := os.Symlink(brokenTarget, brokenLink); err != nil {
		t.Fatalf("failed to create broken symlink: %v", err)
	}

	data, err := createProfileTar(homeDir)
	if err != nil {
		t.Fatalf("createProfileTar should not error with broken symlink: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty tar data (good skill should still be present)")
	}

	entries := readTarEntries(t, data)

	// Broken symlink should not appear
	for name := range entries {
		if filepath.Base(filepath.Dir(name)) == "broken-skill" || name == "skills/broken-skill" {
			t.Errorf("tar should not contain broken symlink entry, found: %s", name)
		}
	}

	// Good skill should still be present
	if _, ok := entries["skills/good-skill/SKILL.md"]; !ok {
		t.Error("tar missing expected entry skills/good-skill/SKILL.md")
	}
}

func TestCreateProfileTar_SymlinkOutsideAllowed(t *testing.T) {
	homeDir := t.TempDir()

	// Create a valid skill
	files := map[string]string{
		".claude/skills/good-skill/SKILL.md": "# Good Skill",
	}
	setupProfileDir(t, homeDir, files)

	// Create a file in /tmp that we'll symlink to
	tmpDir := t.TempDir() // separate temp dir to simulate "outside"
	outsideFile := filepath.Join(tmpDir, "outside-skill")
	if err := os.MkdirAll(outsideFile, 0755); err != nil {
		t.Fatalf("failed to create outside dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(outsideFile, "SKILL.md"), []byte("# Outside"), 0644); err != nil {
		t.Fatalf("failed to write outside file: %v", err)
	}

	// Create symlink pointing outside allowed dirs
	skillsDir := filepath.Join(homeDir, ".claude", "skills")
	outsideLink := filepath.Join(skillsDir, "outside-skill")
	if err := os.Symlink(outsideFile, outsideLink); err != nil {
		t.Fatalf("failed to create outside symlink: %v", err)
	}

	data, err := createProfileTar(homeDir)
	if err != nil {
		t.Fatalf("createProfileTar should not error with outside symlink: %v", err)
	}

	entries := readTarEntries(t, data)

	// Outside symlink should not appear
	for name := range entries {
		if filepath.Base(filepath.Dir(name)) == "outside-skill" || name == "skills/outside-skill" {
			t.Errorf("tar should not contain outside symlink entry, found: %s", name)
		}
	}

	// Good skill should still be present
	if _, ok := entries["skills/good-skill/SKILL.md"]; !ok {
		t.Error("tar missing expected entry skills/good-skill/SKILL.md")
	}
}

func TestCreateProfileTar_NormalSizeDoesNotError(t *testing.T) {
	homeDir := t.TempDir()

	// Create several files with moderate content to verify size tracking doesn't
	// produce false positives. Testing the 100MB limit directly would require
	// creating 100MB+ of test fixtures, which is impractical for unit tests.
	files := map[string]string{
		".claude/skills/skill-a/SKILL.md":  string(make([]byte, 1024)),    // 1KB
		".claude/skills/skill-b/SKILL.md":  string(make([]byte, 10*1024)), // 10KB
		".claude/rules/big-rule.md":        string(make([]byte, 50*1024)), // 50KB
		".claude/agents/agent-a/agent.md":  string(make([]byte, 5*1024)),  // 5KB
		".claude/commands/cmd-a/command.md": string(make([]byte, 1024)),   // 1KB
	}
	setupProfileDir(t, homeDir, files)

	data, err := createProfileTar(homeDir)
	if err != nil {
		t.Fatalf("createProfileTar should not error with normal-sized profile: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty tar data")
	}
}

func TestShouldSkipEntry(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"DS_Store", ".DS_Store", true},
		{"git dir", ".git", true},
		{"normal file", "SKILL.md", false},
		{"dotfile", ".hidden", false},
		{"gitignore", ".gitignore", false},
		{"empty string", "", false},
		{"regular go file", "main.go", false},
		{"git suffix", "not-.git", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldSkipEntry(tt.input)
			if got != tt.want {
				t.Errorf("shouldSkipEntry(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsExcludedProfilePath(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"root CLAUDE.md", "CLAUDE.md", true},
		{"nested CLAUDE.md", "skills/test/CLAUDE.md", true},
		{"operating-in-vagrant dir", "skills/operating-in-vagrant", true},
		{"operating-in-vagrant file", "skills/operating-in-vagrant/SKILL.md", true},
		{"settings.local.json", "settings.local.json", true},
		{"normal skill", "skills/my-skill/SKILL.md", false},
		{"normal rule", "rules/coding-style.md", false},
		{"normal command", "commands/my-cmd/command.md", false},
		{"normal agent", "agents/my-agent/agent.md", false},
		{"empty string", "", false},
		{"claude.md lowercase", "claude.md", false},
		{"CLAUDE.md deep nested", "rules/subdir/CLAUDE.md", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isExcludedProfilePath(tt.input)
			if got != tt.want {
				t.Errorf("isExcludedProfilePath(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidateSymlinkTarget(t *testing.T) {
	// Resolve through EvalSymlinks to handle macOS /var -> /private/var
	rawHomeDir := t.TempDir()
	homeDir, err := filepath.EvalSymlinks(rawHomeDir)
	if err != nil {
		t.Fatalf("failed to resolve temp dir: %v", err)
	}

	// Create the allowed directories
	claudeDir := filepath.Join(homeDir, ".claude")
	agentsDir := filepath.Join(homeDir, ".agents")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("failed to create .claude dir: %v", err)
	}
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatalf("failed to create .agents dir: %v", err)
	}

	// Create real files/dirs for valid symlink targets
	validClaudeTarget := filepath.Join(claudeDir, "skills", "valid-skill")
	if err := os.MkdirAll(validClaudeTarget, 0755); err != nil {
		t.Fatalf("failed to create valid claude target: %v", err)
	}
	if err := os.WriteFile(filepath.Join(validClaudeTarget, "SKILL.md"), []byte("valid"), 0644); err != nil {
		t.Fatalf("failed to write valid skill: %v", err)
	}

	validAgentsTarget := filepath.Join(agentsDir, "skills", "agent-skill")
	if err := os.MkdirAll(validAgentsTarget, 0755); err != nil {
		t.Fatalf("failed to create valid agents target: %v", err)
	}
	if err := os.WriteFile(filepath.Join(validAgentsTarget, "SKILL.md"), []byte("agent"), 0644); err != nil {
		t.Fatalf("failed to write agent skill: %v", err)
	}

	// Create symlinks to test
	symlinkInsideClaude := filepath.Join(claudeDir, "link-to-claude")
	if err := os.Symlink(validClaudeTarget, symlinkInsideClaude); err != nil {
		t.Fatalf("failed to create claude symlink: %v", err)
	}

	symlinkInsideAgents := filepath.Join(claudeDir, "link-to-agents")
	if err := os.Symlink(validAgentsTarget, symlinkInsideAgents); err != nil {
		t.Fatalf("failed to create agents symlink: %v", err)
	}

	// Create a symlink pointing outside allowed directories
	tmpOutside := t.TempDir()
	symlinkOutside := filepath.Join(claudeDir, "link-to-outside")
	if err := os.Symlink(tmpOutside, symlinkOutside); err != nil {
		t.Fatalf("failed to create outside symlink: %v", err)
	}

	// Create a broken symlink
	symlinkBroken := filepath.Join(claudeDir, "link-broken")
	if err := os.Symlink(filepath.Join(homeDir, "does-not-exist"), symlinkBroken); err != nil {
		t.Fatalf("failed to create broken symlink: %v", err)
	}

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "symlink within claude dir",
			path:    symlinkInsideClaude,
			wantErr: false,
		},
		{
			name:    "symlink within agents dir",
			path:    symlinkInsideAgents,
			wantErr: false,
		},
		{
			name:    "symlink outside allowed dirs",
			path:    symlinkOutside,
			wantErr: true,
		},
		{
			name:    "broken symlink",
			path:    symlinkBroken,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSymlinkTarget(tt.path, homeDir)
			if tt.wantErr && err == nil {
				t.Errorf("validateSymlinkTarget(%q) = nil, want error", tt.path)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("validateSymlinkTarget(%q) = %v, want nil", tt.path, err)
			}
		})
	}
}

func TestCreateProfileTar_ExcludesSettingsLocalJSON(t *testing.T) {
	homeDir := t.TempDir()

	files := map[string]string{
		".claude/settings.local.json":         `{"hooks":{}}`,
		".claude/skills/my-skill/SKILL.md":    "# My Skill",
		".claude/rules/style.md":              "# Style",
	}
	setupProfileDir(t, homeDir, files)

	data, err := createProfileTar(homeDir)
	if err != nil {
		t.Fatalf("createProfileTar returned error: %v", err)
	}

	entries := readTarEntries(t, data)

	// settings.local.json should be excluded
	if _, ok := entries["settings.local.json"]; ok {
		t.Error("tar should not contain settings.local.json")
	}

	// Valid files should still be present
	if _, ok := entries["skills/my-skill/SKILL.md"]; !ok {
		t.Error("tar missing expected entry skills/my-skill/SKILL.md")
	}
}

func TestCreateProfileTar_SkipsGitDirs(t *testing.T) {
	homeDir := t.TempDir()

	files := map[string]string{
		".claude/skills/my-skill/SKILL.md": "# My Skill",
		".claude/skills/my-skill/.git/HEAD": "ref: refs/heads/main",
		".claude/rules/style.md":            "# Style",
	}
	setupProfileDir(t, homeDir, files)

	data, err := createProfileTar(homeDir)
	if err != nil {
		t.Fatalf("createProfileTar returned error: %v", err)
	}

	entries := readTarEntries(t, data)

	// .git entries should not appear in the tar
	for name := range entries {
		if filepath.Base(name) == ".git" || filepath.Base(filepath.Dir(name)) == ".git" {
			t.Errorf("tar should not contain .git entry, found: %s", name)
		}
	}

	// Valid files should still be present
	if _, ok := entries["skills/my-skill/SKILL.md"]; !ok {
		t.Error("tar missing expected entry skills/my-skill/SKILL.md")
	}
}

func TestCreateProfileTar_MultipleFilesPerDir(t *testing.T) {
	homeDir := t.TempDir()

	files := map[string]string{
		".claude/skills/multi-skill/SKILL.md":    "# Multi Skill",
		".claude/skills/multi-skill/README.md":   "# README",
		".claude/skills/multi-skill/helpers.sh":  "#!/bin/bash\necho hi",
		".claude/rules/rule-a.md":                "# Rule A",
		".claude/rules/rule-b.md":                "# Rule B",
		".claude/rules/rule-c.md":                "# Rule C",
	}
	setupProfileDir(t, homeDir, files)

	data, err := createProfileTar(homeDir)
	if err != nil {
		t.Fatalf("createProfileTar returned error: %v", err)
	}

	entries := readTarEntries(t, data)

	expectedFiles := []string{
		"skills/multi-skill/SKILL.md",
		"skills/multi-skill/README.md",
		"skills/multi-skill/helpers.sh",
		"rules/rule-a.md",
		"rules/rule-b.md",
		"rules/rule-c.md",
	}

	for _, name := range expectedFiles {
		if _, ok := entries[name]; !ok {
			t.Errorf("tar missing expected entry %q", name)
		}
	}
}

func TestCreateProfileTar_EmptyDirsIgnored(t *testing.T) {
	homeDir := t.TempDir()

	// Create the 4 profile dirs but leave them empty
	for _, dir := range []string{"skills", "commands", "agents", "rules"} {
		dirPath := filepath.Join(homeDir, ".claude", dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			t.Fatalf("failed to create dir %s: %v", dir, err)
		}
	}

	data, err := createProfileTar(homeDir)
	if err != nil {
		t.Fatalf("createProfileTar should not error with empty dirs: %v", err)
	}

	// With empty directories, the tar should either be empty or contain only
	// directory entries (no regular file entries)
	if len(data) > 0 {
		entries := readTarEntries(t, data)
		for name, content := range entries {
			if content != nil {
				t.Errorf("expected no regular file entries in tar with empty dirs, found: %s", name)
			}
		}
	}
}
