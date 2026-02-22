package vagrant

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractOAuthCredentials_EnvVarOverride(t *testing.T) {
	// Set env var — should take priority over all other sources
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "sk-ant-oat01-test-token")

	// Override the func to use the default implementation
	original := extractOAuthCredentialsFunc
	extractOAuthCredentialsFunc = extractOAuthCredentialsDefault
	defer func() { extractOAuthCredentialsFunc = original }()

	data, err := extractOAuthCredentialsFunc()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	var creds map[string]string
	if err := json.Unmarshal(data, &creds); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	if creds["accessToken"] != "sk-ant-oat01-test-token" {
		t.Errorf("accessToken = %q, want %q", creds["accessToken"], "sk-ant-oat01-test-token")
	}

	// Env var produces synthetic credentials — no refresh token
	if _, has := creds["refreshToken"]; has {
		t.Error("env var credentials should not have refreshToken")
	}
}

func TestBuildSyntheticCredentials(t *testing.T) {
	token := "sk-ant-oat01-abc123"
	data := buildSyntheticCredentials(token)

	// Must be valid JSON
	if !json.Valid(data) {
		t.Fatalf("buildSyntheticCredentials returned invalid JSON: %s", string(data))
	}

	var creds map[string]string
	if err := json.Unmarshal(data, &creds); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if creds["accessToken"] != token {
		t.Errorf("accessToken = %q, want %q", creds["accessToken"], token)
	}

	if len(creds) != 1 {
		t.Errorf("expected exactly 1 key (accessToken), got %d keys", len(creds))
	}
}

func TestExtractFromCredentialsFile_Exists(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	// Create ~/.claude/.credentials.json
	claudeDir := filepath.Join(tempHome, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("failed to create .claude dir: %v", err)
	}

	credContent := `{"accessToken":"sk-ant-oat01-file","refreshToken":"sk-ant-ort01-file","expiresAt":"2025-01-01T00:00:00Z"}`
	if err := os.WriteFile(filepath.Join(claudeDir, ".credentials.json"), []byte(credContent), 0600); err != nil {
		t.Fatalf("failed to write credentials file: %v", err)
	}

	data, err := extractFromCredentialsFile()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	var creds map[string]string
	if err := json.Unmarshal(data, &creds); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	if creds["accessToken"] != "sk-ant-oat01-file" {
		t.Errorf("accessToken = %q, want %q", creds["accessToken"], "sk-ant-oat01-file")
	}
	if creds["refreshToken"] != "sk-ant-ort01-file" {
		t.Errorf("refreshToken = %q, want %q", creds["refreshToken"], "sk-ant-ort01-file")
	}
}

func TestExtractFromCredentialsFile_Missing(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	_, err := extractFromCredentialsFile()
	if err == nil {
		t.Fatal("expected error for missing credentials file, got nil")
	}
}

func TestExtractFromCredentialsFile_InvalidJSON(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	claudeDir := filepath.Join(tempHome, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("failed to create .claude dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(claudeDir, ".credentials.json"), []byte("not valid json"), 0600); err != nil {
		t.Fatalf("failed to write credentials file: %v", err)
	}

	_, err := extractFromCredentialsFile()
	if err == nil {
		t.Fatal("expected error for invalid JSON credentials file, got nil")
	}
}

func TestExtractOAuthCredentials_NoCredentials(t *testing.T) {
	// Ensure no env var is set
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "")
	os.Unsetenv("CLAUDE_CODE_OAUTH_TOKEN")

	// Point HOME to empty temp dir (no credentials file)
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	// Use the real implementation
	original := extractOAuthCredentialsFunc
	extractOAuthCredentialsFunc = extractOAuthCredentialsDefault
	defer func() { extractOAuthCredentialsFunc = original }()

	_, err := extractOAuthCredentialsFunc()
	if err != ErrNoOAuthCredentials {
		t.Errorf("expected ErrNoOAuthCredentials, got: %v", err)
	}
}
