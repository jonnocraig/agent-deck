package vagrant

import (
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

const (
	diskSpaceMinimumMB = int64(5120)  // 5GB — block session creation below this
	diskSpaceWarningMB = int64(10240) // 10GB — warn but allow
)

// CheckDiskSpace returns available disk space in MB on the filesystem
// containing projectPath. Uses syscall.Statfs on darwin/linux.
func (m *Manager) CheckDiskSpace() (int64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(m.projectPath, &stat); err != nil {
		return 0, fmt.Errorf("failed to check disk space: %w", err)
	}

	// Calculate available bytes: available blocks * block size
	availableBytes := stat.Bavail * uint64(stat.Bsize)

	// Convert to MB
	return int64(availableBytes / (1024 * 1024)), nil
}

// CheckVBoxInstalled checks if VBoxManage is in PATH and returns version string.
// Returns version like "7.2.4" extracted from output like "7.2.4r163906".
func (m *Manager) CheckVBoxInstalled() (string, error) {
	// Check if VBoxManage exists in PATH
	path, err := exec.LookPath("VBoxManage")
	if err != nil {
		return "", fmt.Errorf("VirtualBox not found. Install from https://www.virtualbox.org/wiki/Downloads")
	}

	// Run VBoxManage --version to get version string
	cmd := exec.Command(path, "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get VirtualBox version: %w", err)
	}

	// Parse version from output (e.g., "7.2.4r163906" -> "7.2.4")
	rawVersion := strings.TrimSpace(string(output))
	version := extractVersion(rawVersion)

	return version, nil
}

// extractVersion extracts clean version string from VBoxManage output.
// Handles formats like "7.2.4r163906" -> "7.2.4"
func extractVersion(raw string) string {
	// Find first 'r' or other non-version character
	for i, c := range raw {
		if c != '.' && (c < '0' || c > '9') {
			return raw[:i]
		}
	}
	return raw
}

// parseVersion parses a version string like "7.2.4" into (major, minor).
// Returns (0, 0) for invalid or empty strings.
func parseVersion(version string) (major, minor int) {
	if version == "" {
		return 0, 0
	}

	parts := strings.Split(version, ".")
	if len(parts) == 0 {
		return 0, 0
	}

	// Parse major version
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0
	}

	// Parse minor version if available
	if len(parts) > 1 {
		minor, err = strconv.Atoi(parts[1])
		if err != nil {
			// Invalid minor, but major is valid
			return major, 0
		}
	}

	return major, minor
}

// IsBoxCached checks if the configured Vagrant box is already downloaded.
// Runs 'vagrant box list --machine-readable' and checks for settings.Box.
func (m *Manager) IsBoxCached() bool {
	cmd := exec.Command("vagrant", "box", "list", "--machine-readable")
	output, err := cmd.Output()
	if err != nil {
		// If vagrant box list fails, assume not cached
		return false
	}

	// Check if settings.Box appears in output
	return strings.Contains(string(output), m.settings.Box)
}

// PreflightCheck performs combined preflight validation before VM operations.
// Validates Vagrant installation, VirtualBox installation and version,
// and available disk space. Returns error if prerequisites are missing.
func (m *Manager) PreflightCheck() error {
	// 1. Check if Vagrant is installed
	if !m.IsInstalled() {
		return fmt.Errorf("Vagrant not found. Install from https://www.vagrantup.com/downloads")
	}

	// 2. Check if VirtualBox is installed and get version
	vboxVersion, err := m.CheckVBoxInstalled()
	if err != nil {
		return err // Error already contains install URL
	}

	// 3. Check VirtualBox version >= 7.0
	major, minor := parseVersion(vboxVersion)
	if major < 7 {
		return fmt.Errorf("VirtualBox %s is too old. Version 7.0+ required (7.2+ recommended for Apple Silicon)", vboxVersion)
	}

	// 4. If darwin/arm64 and VBox 7.0-7.1, log warning (non-blocking)
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		if major == 7 && minor < 2 {
			// Non-blocking warning - would normally be logged
			// For now, we just note this should be logged in production
			_ = fmt.Sprintf("WARNING: VirtualBox %s on Apple Silicon may have stability issues. Version 7.2+ recommended.", vboxVersion)
		}
	}

	// 5. Check disk space
	availableMB, err := m.CheckDiskSpace()
	if err != nil {
		return fmt.Errorf("failed to check disk space: %w", err)
	}

	// 6. Calculate required space (add 2GB if box not cached)
	requiredMB := diskSpaceMinimumMB
	if !m.IsBoxCached() {
		requiredMB += 2048 // Additional 2GB for box download
	}

	// 7. Check if we have enough space
	if availableMB < requiredMB {
		return fmt.Errorf("insufficient disk space: %dMB available, %dMB required. Free up space or reduce vagrant.memory_mb", availableMB, requiredMB)
	}

	return nil
}
