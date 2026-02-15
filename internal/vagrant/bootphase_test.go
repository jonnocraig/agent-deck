package vagrant

import (
	"fmt"
	"runtime"
	"testing"
)

func TestParseBootPhaseDownloading(t *testing.T) {
	tests := []struct {
		line     string
		expected BootPhase
	}{
		{"1234567890,target,ui,info,Downloading box", BootPhaseDownloading},
		{"1234567890,target,ui,info,Downloading ubuntu/jammy64", BootPhaseDownloading},
		{"Some line with Downloading in it", BootPhaseDownloading},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			phase, ok := ParseBootPhase(tt.line)
			if !ok {
				t.Errorf("Expected match, got no match for line: %s", tt.line)
			}
			if phase != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, phase)
			}
		})
	}
}

func TestParseBootPhaseImporting(t *testing.T) {
	tests := []struct {
		line     string
		expected BootPhase
	}{
		{"1234567890,target,ui,info,Importing base box", BootPhaseImporting},
		{"1234567890,target,ui,info,Importing box as base", BootPhaseImporting},
		{"Random line with Importing keyword", BootPhaseImporting},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			phase, ok := ParseBootPhase(tt.line)
			if !ok {
				t.Errorf("Expected match, got no match for line: %s", tt.line)
			}
			if phase != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, phase)
			}
		})
	}
}

func TestParseBootPhaseBooting(t *testing.T) {
	tests := []struct {
		line     string
		expected BootPhase
	}{
		{"1234567890,target,ui,info,Booting VM", BootPhaseBooting},
		{"1234567890,target,action,up,start", BootPhaseBooting},
		{"1234567890,target,ui,info,Booting the VM now", BootPhaseBooting},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			phase, ok := ParseBootPhase(tt.line)
			if !ok {
				t.Errorf("Expected match, got no match for line: %s", tt.line)
			}
			if phase != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, phase)
			}
		})
	}
}

func TestParseBootPhaseNetwork(t *testing.T) {
	tests := []struct {
		line     string
		expected BootPhase
	}{
		{"1234567890,target,action,up,configure_networks", BootPhaseNetwork},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			phase, ok := ParseBootPhase(tt.line)
			if !ok {
				t.Errorf("Expected match, got no match for line: %s", tt.line)
			}
			if phase != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, phase)
			}
		})
	}
}

func TestParseBootPhaseMounting(t *testing.T) {
	tests := []struct {
		line     string
		expected BootPhase
	}{
		{"1234567890,target,action,up,share_folders", BootPhaseMounting},
		{"1234567890,target,action,up,sync_folders", BootPhaseMounting},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			phase, ok := ParseBootPhase(tt.line)
			if !ok {
				t.Errorf("Expected match, got no match for line: %s", tt.line)
			}
			if phase != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, phase)
			}
		})
	}
}

func TestParseBootPhaseProvisioning(t *testing.T) {
	tests := []struct {
		line     string
		expected BootPhase
	}{
		{"1234567890,target,action,up,provision", BootPhaseProvisioning},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			phase, ok := ParseBootPhase(tt.line)
			if !ok {
				t.Errorf("Expected match, got no match for line: %s", tt.line)
			}
			if phase != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, phase)
			}
		})
	}
}

func TestParseBootPhaseNpmInstall(t *testing.T) {
	tests := []struct {
		line     string
		expected BootPhase
	}{
		{"1234567890,target,ui,info,Running: npm install -g @anthropic-ai/claude-code", BootPhaseNpmInstall},
		{"npm install -g @anthropic-ai/claude-code", BootPhaseNpmInstall},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			phase, ok := ParseBootPhase(tt.line)
			if !ok {
				t.Errorf("Expected match, got no match for line: %s", tt.line)
			}
			if phase != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, phase)
			}
		})
	}
}

func TestParseBootPhaseReady(t *testing.T) {
	tests := []struct {
		line     string
		expected BootPhase
	}{
		{"1234567890,target,action,up,complete", BootPhaseReady},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			phase, ok := ParseBootPhase(tt.line)
			if !ok {
				t.Errorf("Expected match, got no match for line: %s", tt.line)
			}
			if phase != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, phase)
			}
		})
	}
}

func TestParseBootPhaseNoMatch(t *testing.T) {
	tests := []string{
		"1234567890,target,ui,info,Some random message",
		"1234567890,target,action,halt,stop",
		"Unrelated log line",
		"",
	}

	for _, line := range tests {
		t.Run(line, func(t *testing.T) {
			_, ok := ParseBootPhase(line)
			if ok {
				t.Errorf("Expected no match for line: %s", line)
			}
		})
	}
}

func TestWrapVagrantUpErrorAppleSilicon(t *testing.T) {
	// Only test the wrapping logic on darwin/arm64, or test the function directly
	if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		t.Skip("Skipping Apple Silicon specific test on non-darwin/arm64 platform")
	}

	tests := []struct {
		name          string
		stderr        string
		shouldWrap    bool
		expectedMsg   string
	}{
		{
			name:          "kernel driver error",
			stderr:        "VirtualBox failed to start. Error: kernel driver not installed",
			shouldWrap:    true,
			expectedMsg:   "VirtualBox requires approval",
		},
		{
			name:          "vboxdrv error",
			stderr:        "The vboxdrv kernel module is not loaded",
			shouldWrap:    true,
			expectedMsg:   "VirtualBox requires approval",
		},
		{
			name:          "NS_ERROR_FAILURE",
			stderr:        "NS_ERROR_FAILURE (0x80004005)",
			shouldWrap:    true,
			expectedMsg:   "VirtualBox requires approval",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalErr := fmt.Errorf("vagrant up failed")
			wrappedErr := wrapVagrantUpError(originalErr, tt.stderr)

			if tt.shouldWrap {
				errMsg := wrappedErr.Error()
				if !contains(errMsg, tt.expectedMsg) {
					t.Errorf("Expected error message to contain %q, got: %s", tt.expectedMsg, errMsg)
				}
				if !contains(errMsg, "System Settings → Privacy & Security") {
					t.Errorf("Expected error message to contain guidance about System Settings")
				}
			}
		})
	}
}

func TestWrapVagrantUpErrorUnrelatedFailure(t *testing.T) {
	tests := []struct {
		name   string
		stderr string
	}{
		{
			name:   "network timeout",
			stderr: "Failed to connect to vagrantcloud.com",
		},
		{
			name:   "disk space",
			stderr: "No space left on device",
		},
		{
			name:   "empty stderr",
			stderr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalErr := fmt.Errorf("vagrant up failed")
			wrappedErr := wrapVagrantUpError(originalErr, tt.stderr)

			// On non-darwin/arm64, should always return original error
			// On darwin/arm64 with unrelated failures, should also return original
			if wrappedErr != originalErr {
				errMsg := wrappedErr.Error()
				// If wrapped, it should NOT be the kext message
				if contains(errMsg, "VirtualBox requires approval") {
					t.Errorf("Should not wrap unrelated errors with kext message, got: %s", errMsg)
				}
			}
		})
	}
}

// Helper function for string containment check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && hasSubstring(s, substr)))
}

func hasSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
