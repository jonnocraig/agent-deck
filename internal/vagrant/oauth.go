package vagrant

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// extractOAuthCredentialsFunc is a package-level injectable for testing.
// Follows the existing getAvailableMCPsFunc pattern.
var extractOAuthCredentialsFunc = extractOAuthCredentialsDefault

// ErrNoOAuthCredentials is returned when no OAuth credentials are found on the host.
var ErrNoOAuthCredentials = errors.New("no OAuth credentials found on host")

// extractOAuthCredentialsDefault extracts OAuth credentials from the host.
// Priority: CLAUDE_CODE_OAUTH_TOKEN env var > macOS Keychain > credentials file (~/.claude/.credentials.json)
func extractOAuthCredentialsDefault() ([]byte, error) {
	// 1. Check env var override (highest priority)
	if token := os.Getenv("CLAUDE_CODE_OAUTH_TOKEN"); token != "" {
		return buildSyntheticCredentials(token), nil
	}

	// 2. Try macOS Keychain (only on darwin)
	if runtime.GOOS == "darwin" {
		if data, err := extractFromMacOSKeychain(); err == nil {
			return data, nil
		}
	}

	// 3. Try credentials file (Linux path, also fallback on macOS)
	if data, err := extractFromCredentialsFile(); err == nil {
		return data, nil
	}

	return nil, ErrNoOAuthCredentials
}

// extractFromMacOSKeychain runs: security find-generic-password -s "Claude Code-credentials" -w
// Returns raw credential JSON bytes. Validates output is valid JSON.
func extractFromMacOSKeychain() ([]byte, error) {
	cmd := exec.Command("security", "find-generic-password", "-s", "Claude Code-credentials", "-w")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("keychain lookup failed: %w", err)
	}

	trimmed := []byte(strings.TrimSpace(string(out)))
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("keychain returned empty value")
	}

	// Validate that the output is valid JSON
	if !json.Valid(trimmed) {
		return nil, fmt.Errorf("keychain value is not valid JSON")
	}

	return trimmed, nil
}

// extractFromCredentialsFile reads ~/.claude/.credentials.json.
// This is the standard path on Linux and a fallback on macOS.
func extractFromCredentialsFile() ([]byte, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("cannot determine home directory: %w", err)
	}

	credPath := filepath.Join(homeDir, ".claude", ".credentials.json")
	data, err := os.ReadFile(credPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read credentials file: %w", err)
	}

	if !json.Valid(data) {
		return nil, fmt.Errorf("credentials file is not valid JSON")
	}

	return data, nil
}

// buildSyntheticCredentials creates minimal credentials JSON from a bare access token
// (from CLAUDE_CODE_OAUTH_TOKEN). No refresh token, so no auto-refresh.
func buildSyntheticCredentials(accessToken string) []byte {
	creds := map[string]string{
		"accessToken": accessToken,
	}
	data, _ := json.Marshal(creds)
	return data
}
