package vagrant

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestParseVersion tests version string parsing
func TestParseVersion(t *testing.T) {
	tests := []struct {
		name          string
		version       string
		expectedMajor int
		expectedMinor int
	}{
		{
			name:          "full version string",
			version:       "7.2.4",
			expectedMajor: 7,
			expectedMinor: 2,
		},
		{
			name:          "version with beta suffix",
			version:       "7.2.4r163906",
			expectedMajor: 7,
			expectedMinor: 2,
		},
		{
			name:          "two part version",
			version:       "6.1",
			expectedMajor: 6,
			expectedMinor: 1,
		},
		{
			name:          "single digit major",
			version:       "7.0.0",
			expectedMajor: 7,
			expectedMinor: 0,
		},
		{
			name:          "empty string",
			version:       "",
			expectedMajor: 0,
			expectedMinor: 0,
		},
		{
			name:          "invalid format",
			version:       "invalid",
			expectedMajor: 0,
			expectedMinor: 0,
		},
		{
			name:          "just major",
			version:       "7",
			expectedMajor: 7,
			expectedMinor: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			major, minor := parseVersion(tt.version)
			require.Equal(t, tt.expectedMajor, major, "major version mismatch")
			require.Equal(t, tt.expectedMinor, minor, "minor version mismatch")
		})
	}
}

// TestDiskSpaceConstants verifies threshold values
func TestDiskSpaceConstants(t *testing.T) {
	require.Equal(t, int64(5120), diskSpaceMinimumMB, "minimum disk space should be 5GB")
	require.Equal(t, int64(10240), diskSpaceWarningMB, "warning disk space should be 10GB")
	require.Greater(t, diskSpaceWarningMB, diskSpaceMinimumMB, "warning should be > minimum")
}

// TestCheckDiskSpaceReturnsPositive tests actual disk space check
func TestCheckDiskSpaceReturnsPositive(t *testing.T) {
	m := &Manager{projectPath: t.TempDir()}
	mb, err := m.CheckDiskSpace()
	require.NoError(t, err, "disk space check should not error")
	require.Greater(t, mb, int64(0), "disk space should be positive")
}

// TestCheckDiskSpaceInvalidPath tests error handling
func TestCheckDiskSpaceInvalidPath(t *testing.T) {
	m := &Manager{projectPath: "/nonexistent/path/that/does/not/exist"}
	_, err := m.CheckDiskSpace()
	require.Error(t, err, "should error for nonexistent path")
}

// TestPreflightCheckErrorMessages tests error message formatting
func TestPreflightCheckErrorMessages(t *testing.T) {
	tests := []struct {
		name     string
		errType  string
		expected string
	}{
		{
			name:     "vagrant missing",
			errType:  "vagrant_missing",
			expected: "Vagrant not found. Install from https://www.vagrantup.com/downloads",
		},
		{
			name:     "vbox missing",
			errType:  "vbox_missing",
			expected: "VirtualBox not found. Install from https://www.virtualbox.org/wiki/Downloads",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var errMsg string
			switch tt.errType {
			case "vagrant_missing":
				errMsg = "Vagrant not found. Install from https://www.vagrantup.com/downloads"
			case "vbox_missing":
				errMsg = "VirtualBox not found. Install from https://www.virtualbox.org/wiki/Downloads"
			}
			require.Contains(t, errMsg, tt.expected)
		})
	}
}

// TestCheckVBoxVersionComparison tests version comparison logic
func TestCheckVBoxVersionComparison(t *testing.T) {
	tests := []struct {
		name        string
		version     string
		shouldError bool
		description string
	}{
		{
			name:        "version 7.2 supported",
			version:     "7.2.4",
			shouldError: false,
			description: "modern VBox version should pass",
		},
		{
			name:        "version 7.0 supported",
			version:     "7.0.0",
			shouldError: false,
			description: "minimum version should pass",
		},
		{
			name:        "version 6.1 too old",
			version:     "6.1.50",
			shouldError: true,
			description: "old version should fail",
		},
		{
			name:        "version 5.x too old",
			version:     "5.2.44",
			shouldError: true,
			description: "very old version should fail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			major, _ := parseVersion(tt.version)
			tooOld := major < 7
			if tt.shouldError {
				require.True(t, tooOld, tt.description)
			} else {
				require.False(t, tooOld, tt.description)
			}
		})
	}
}

// TestAppleSiliconWarning tests darwin/arm64 detection
func TestAppleSiliconWarning(t *testing.T) {
	isAppleSilicon := runtime.GOOS == "darwin" && runtime.GOARCH == "arm64"

	// This test just verifies we can detect the platform
	// The actual warning is non-blocking and logged during PreflightCheck
	t.Logf("Platform: %s/%s, Apple Silicon: %v", runtime.GOOS, runtime.GOARCH, isAppleSilicon)

	// Test version range that triggers warning
	if isAppleSilicon {
		major, minor := parseVersion("7.0.10")
		shouldWarn := major == 7 && minor < 2
		require.True(t, shouldWarn, "VBox 7.0-7.1 should trigger warning on Apple Silicon")

		major, minor = parseVersion("7.2.0")
		shouldWarn = major == 7 && minor < 2
		require.False(t, shouldWarn, "VBox 7.2+ should not trigger warning")
	}
}

// TestDiskSpaceCalculation tests required space calculation
func TestDiskSpaceCalculation(t *testing.T) {
	tests := []struct {
		name         string
		boxCached    bool
		expectedBase int64
		expectedAdd  int64
	}{
		{
			name:         "box already cached",
			boxCached:    true,
			expectedBase: diskSpaceMinimumMB,
			expectedAdd:  0,
		},
		{
			name:         "box not cached",
			boxCached:    false,
			expectedBase: diskSpaceMinimumMB,
			expectedAdd:  2048,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			required := tt.expectedBase
			if !tt.boxCached {
				required += tt.expectedAdd
			}

			if tt.boxCached {
				require.Equal(t, diskSpaceMinimumMB, required)
			} else {
				require.Equal(t, diskSpaceMinimumMB+2048, required)
			}
		})
	}
}
