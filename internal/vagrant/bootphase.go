package vagrant

import (
	"fmt"
	"runtime"
	"strings"
)

// ParseBootPhase parses a single line of vagrant's machine-readable output
// and returns the corresponding BootPhase if matched, or ("", false) if no phase detected.
//
// Machine-readable format: timestamp,target,type,data
func ParseBootPhase(line string) (BootPhase, bool) {
	// Check for downloading phase
	if strings.Contains(line, "Downloading") {
		return BootPhaseDownloading, true
	}

	// Check for importing phase
	if strings.Contains(line, "Importing") {
		return BootPhaseImporting, true
	}

	// Check for npm install phase (must be before other checks to avoid false positives)
	if strings.Contains(line, "npm install -g @anthropic-ai/claude-code") {
		return BootPhaseNpmInstall, true
	}

	// Check for booting phase
	if strings.Contains(line, "Booting") || strings.Contains(line, "action,up,start") {
		return BootPhaseBooting, true
	}

	// Check for network configuration phase
	if strings.Contains(line, "action,up,configure_networks") {
		return BootPhaseNetwork, true
	}

	// Check for mounting/syncing folders phase
	if strings.Contains(line, "action,up,share_folders") || strings.Contains(line, "action,up,sync_folders") {
		return BootPhaseMounting, true
	}

	// Check for provisioning phase
	if strings.Contains(line, "action,up,provision") {
		return BootPhaseProvisioning, true
	}

	// Check for ready/complete phase
	if strings.Contains(line, "action,up,complete") {
		return BootPhaseReady, true
	}

	// No match found
	return "", false
}

// wrapVagrantUpError wraps vagrant up errors with user-friendly messages for known failure patterns.
//
// On darwin/arm64 (Apple Silicon), if stderr contains VirtualBox kernel driver errors,
// returns a helpful message about approving the kernel extension in System Settings.
// Otherwise, returns the original error unchanged.
func wrapVagrantUpError(err error, stderr string) error {
	// Only apply special handling on Apple Silicon
	if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		return err
	}

	// Check for VirtualBox kernel extension approval issues
	if strings.Contains(stderr, "kernel driver") ||
		strings.Contains(stderr, "vboxdrv") ||
		strings.Contains(stderr, "NS_ERROR_FAILURE") {
		return fmt.Errorf("VirtualBox requires approval in System Settings → Privacy & Security. Approve the VirtualBox kernel extension, then press R to retry. Original error: %w", err)
	}

	// Return original error for unrelated failures
	return err
}
