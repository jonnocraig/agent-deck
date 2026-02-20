package vagrant

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/asheshgoplani/agent-deck/internal/session"
)

// TestWrapCommandBasic tests wrapping a simple command with no tunnels or env vars
func TestWrapCommandBasic(t *testing.T) {
	m := &Manager{
		projectPath: "/project",
		settings:    session.VagrantSettings{},
	}

	result := m.WrapCommand("npm test", nil, nil)
	expected := "vagrant ssh -- -t 'cd /vagrant && npm test'"

	if result != expected {
		t.Errorf("Expected: %s\nGot: %s", expected, result)
	}
}

// TestWrapCommandWithTunnels tests wrapping with multiple tunnel ports
func TestWrapCommandWithTunnels(t *testing.T) {
	m := &Manager{
		projectPath: "/project",
		settings:    session.VagrantSettings{},
	}

	result := m.WrapCommand("npm start", nil, []int{8080, 3000})
	expected := "vagrant ssh -- -R 3000:localhost:3000 -R 8080:localhost:8080 -t 'cd /vagrant && npm start'"

	if result != expected {
		t.Errorf("Expected: %s\nGot: %s", expected, result)
	}
}

// TestWrapCommandWithSendEnv tests wrapping with multiple env var names
func TestWrapCommandWithSendEnv(t *testing.T) {
	m := &Manager{
		projectPath: "/project",
		settings:    session.VagrantSettings{},
	}

	result := m.WrapCommand("npm start", []string{"ANTHROPIC_API_KEY", "EXA_API_KEY"}, nil)
	expected := "vagrant ssh -- -o SendEnv=ANTHROPIC_API_KEY -o SendEnv=EXA_API_KEY -t 'cd /vagrant && npm start'"

	if result != expected {
		t.Errorf("Expected: %s\nGot: %s", expected, result)
	}
}

// TestWrapCommandWithBothTunnelsAndEnv tests wrapping with both tunnels and env vars
func TestWrapCommandWithBothTunnelsAndEnv(t *testing.T) {
	m := &Manager{
		projectPath: "/project",
		settings:    session.VagrantSettings{},
	}

	result := m.WrapCommand("npm start", []string{"API_KEY"}, []int{8080})
	expected := "vagrant ssh -- -R 8080:localhost:8080 -o SendEnv=API_KEY -t 'cd /vagrant && npm start'"

	if result != expected {
		t.Errorf("Expected: %s\nGot: %s", expected, result)
	}
}

// TestWrapCommandEmptyCommand tests wrapping an empty command
func TestWrapCommandEmptyCommand(t *testing.T) {
	m := &Manager{
		projectPath: "/project",
		settings:    session.VagrantSettings{},
	}

	result := m.WrapCommand("", nil, nil)
	expected := "vagrant ssh -- -t 'cd /vagrant && '"

	if result != expected {
		t.Errorf("Expected: %s\nGot: %s", expected, result)
	}
}

// TestWrapCommandTunnelDedup tests that duplicate tunnel ports are deduplicated
func TestWrapCommandTunnelDedup(t *testing.T) {
	m := &Manager{
		projectPath: "/project",
		settings:    session.VagrantSettings{},
	}

	result := m.WrapCommand("npm start", nil, []int{8080, 8080, 3000})
	expected := "vagrant ssh -- -R 3000:localhost:3000 -R 8080:localhost:8080 -t 'cd /vagrant && npm start'"

	if result != expected {
		t.Errorf("Expected: %s\nGot: %s", expected, result)
	}
}

// TestWrapCommandTunnelSorted tests that tunnel ports are sorted
func TestWrapCommandTunnelSorted(t *testing.T) {
	m := &Manager{
		projectPath: "/project",
		settings:    session.VagrantSettings{},
	}

	result := m.WrapCommand("npm start", nil, []int{9000, 8080, 3000})
	expected := "vagrant ssh -- -R 3000:localhost:3000 -R 8080:localhost:8080 -R 9000:localhost:9000 -t 'cd /vagrant && npm start'"

	if result != expected {
		t.Errorf("Expected: %s\nGot: %s", expected, result)
	}
}

// TestWrapCommandEnvSorted tests that env var names are sorted
func TestWrapCommandEnvSorted(t *testing.T) {
	m := &Manager{
		projectPath: "/project",
		settings:    session.VagrantSettings{},
	}

	result := m.WrapCommand("npm start", []string{"ZED_API_KEY", "ANTHROPIC_API_KEY", "EXA_API_KEY"}, nil)
	expected := "vagrant ssh -- -o SendEnv=ANTHROPIC_API_KEY -o SendEnv=EXA_API_KEY -o SendEnv=ZED_API_KEY -t 'cd /vagrant && npm start'"

	if result != expected {
		t.Errorf("Expected: %s\nGot: %s", expected, result)
	}
}

// TestPollingEnvVarsInjectedForVirtualBox tests that polling env vars are added for VirtualBox
func TestPollingEnvVarsInjectedForVirtualBox(t *testing.T) {
	m := &Manager{
		projectPath: "/project",
		settings: session.VagrantSettings{
			SyncedFolderType: "virtualbox",
		},
	}

	result := m.WrapCommand("npm start", []string{"API_KEY"}, nil)

	// Should contain the polling env vars
	if !strings.Contains(result, "CHOKIDAR_USEPOLLING") {
		t.Error("Expected CHOKIDAR_USEPOLLING to be included")
	}
	if !strings.Contains(result, "WATCHPACK_POLLING") {
		t.Error("Expected WATCHPACK_POLLING to be included")
	}
	if !strings.Contains(result, "TSC_WATCHFILE") {
		t.Error("Expected TSC_WATCHFILE to be included")
	}
	if !strings.Contains(result, "API_KEY") {
		t.Error("Expected API_KEY to be included")
	}
}

// TestPollingEnvVarsNotInjectedForNFS tests that polling env vars are NOT added for NFS
func TestPollingEnvVarsNotInjectedForNFS(t *testing.T) {
	m := &Manager{
		projectPath: "/project",
		settings: session.VagrantSettings{
			SyncedFolderType: "nfs",
		},
	}

	result := m.WrapCommand("npm start", []string{"API_KEY"}, nil)

	// Should NOT contain polling env vars
	if strings.Contains(result, "CHOKIDAR_USEPOLLING") {
		t.Error("CHOKIDAR_USEPOLLING should not be included for NFS")
	}
	if strings.Contains(result, "WATCHPACK_POLLING") {
		t.Error("WATCHPACK_POLLING should not be included for NFS")
	}
	if strings.Contains(result, "TSC_WATCHFILE") {
		t.Error("TSC_WATCHFILE should not be included for NFS")
	}
	// Should still contain the user-provided env var
	if !strings.Contains(result, "API_KEY") {
		t.Error("Expected API_KEY to be included")
	}
}

// TestProxyEnvVarsForwardedWhenSet tests that proxy env vars are forwarded when set
func TestProxyEnvVarsForwardedWhenSet(t *testing.T) {
	// Set proxy env vars for this test
	t.Setenv("HTTP_PROXY", "http://proxy.example.com:8080")
	t.Setenv("HTTPS_PROXY", "https://proxy.example.com:8443")
	t.Setenv("NO_PROXY", "localhost,127.0.0.1")

	forwardProxyEnv := true
	m := &Manager{
		projectPath: "/project",
		settings: session.VagrantSettings{
			ForwardProxyEnv: &forwardProxyEnv,
		},
	}

	result := m.WrapCommand("npm start", nil, nil)

	// Should contain the proxy env vars
	if !strings.Contains(result, "HTTP_PROXY") {
		t.Error("Expected HTTP_PROXY to be forwarded")
	}
	if !strings.Contains(result, "HTTPS_PROXY") {
		t.Error("Expected HTTPS_PROXY to be forwarded")
	}
	if !strings.Contains(result, "NO_PROXY") {
		t.Error("Expected NO_PROXY to be forwarded")
	}
}

// TestProxyEnvVarsNotForwardedWhenUnset tests that proxy env vars are NOT forwarded when not set
func TestProxyEnvVarsNotForwardedWhenUnset(t *testing.T) {
	// Ensure proxy env vars are NOT set
	for _, key := range []string{"HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY", "http_proxy", "https_proxy", "no_proxy"} {
		os.Unsetenv(key)
	}

	forwardProxyEnv := true
	m := &Manager{
		projectPath: "/project",
		settings: session.VagrantSettings{
			ForwardProxyEnv: &forwardProxyEnv,
		},
	}

	result := m.WrapCommand("npm start", nil, nil)

	// Should NOT contain any proxy env vars
	if strings.Contains(result, "HTTP_PROXY") {
		t.Error("HTTP_PROXY should not be forwarded when not set")
	}
	if strings.Contains(result, "HTTPS_PROXY") {
		t.Error("HTTPS_PROXY should not be forwarded when not set")
	}
	if strings.Contains(result, "NO_PROXY") {
		t.Error("NO_PROXY should not be forwarded when not set")
	}
}

// TestProxyEnvVarsDisabledByConfig tests that proxy env vars are NOT forwarded when disabled
func TestProxyEnvVarsDisabledByConfig(t *testing.T) {
	// Set proxy env vars
	t.Setenv("HTTP_PROXY", "http://proxy.example.com:8080")
	t.Setenv("HTTPS_PROXY", "https://proxy.example.com:8443")

	forwardProxyEnv := false
	m := &Manager{
		projectPath: "/project",
		settings: session.VagrantSettings{
			ForwardProxyEnv: &forwardProxyEnv,
		},
	}

	result := m.WrapCommand("npm start", nil, nil)

	// Should NOT contain proxy env vars even though they're set
	if strings.Contains(result, "HTTP_PROXY") {
		t.Error("HTTP_PROXY should not be forwarded when ForwardProxyEnv=false")
	}
	if strings.Contains(result, "HTTPS_PROXY") {
		t.Error("HTTPS_PROXY should not be forwarded when ForwardProxyEnv=false")
	}
}

// TestProxyEnvVarsBothCases tests that both uppercase and lowercase proxy vars are forwarded
func TestProxyEnvVarsBothCases(t *testing.T) {
	// Set both uppercase and lowercase
	t.Setenv("HTTP_PROXY", "http://proxy1.example.com:8080")
	t.Setenv("http_proxy", "http://proxy2.example.com:8080")

	forwardProxyEnv := true
	m := &Manager{
		projectPath: "/project",
		settings: session.VagrantSettings{
			ForwardProxyEnv: &forwardProxyEnv,
		},
	}

	result := m.WrapCommand("npm start", nil, nil)

	// Should contain both
	if !strings.Contains(result, "HTTP_PROXY") {
		t.Error("Expected HTTP_PROXY to be forwarded")
	}
	if !strings.Contains(result, "http_proxy") {
		t.Error("Expected http_proxy to be forwarded")
	}

	// Count occurrences to ensure both are present (not just one)
	httpProxyCount := strings.Count(result, "HTTP_PROXY")
	httpProxyLowerCount := strings.Count(result, "http_proxy")

	if httpProxyCount != 1 {
		t.Errorf("Expected HTTP_PROXY to appear exactly once, got %d", httpProxyCount)
	}
	if httpProxyLowerCount != 1 {
		t.Errorf("Expected http_proxy to appear exactly once, got %d", httpProxyLowerCount)
	}
}

// TestWrapCommandComplexScenario tests a complex scenario with all features
func TestWrapCommandComplexScenario(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://proxy.example.com:8080")
	t.Setenv("no_proxy", "localhost")

	forwardProxyEnv := true
	m := &Manager{
		projectPath: "/project",
		settings: session.VagrantSettings{
			SyncedFolderType: "virtualbox",
			ForwardProxyEnv:  &forwardProxyEnv,
		},
	}

	result := m.WrapCommand("npm test", []string{"API_KEY", "ZED_KEY"}, []int{8080, 3000})

	// Verify all components
	if !strings.HasPrefix(result, "vagrant ssh --") {
		t.Error("Expected command to start with 'vagrant ssh --'")
	}
	if !strings.Contains(result, "-R 3000:localhost:3000") {
		t.Error("Expected tunnel for port 3000")
	}
	if !strings.Contains(result, "-R 8080:localhost:8080") {
		t.Error("Expected tunnel for port 8080")
	}
	if !strings.Contains(result, "SendEnv=API_KEY") {
		t.Error("Expected SendEnv for API_KEY")
	}
	if !strings.Contains(result, "SendEnv=ZED_KEY") {
		t.Error("Expected SendEnv for ZED_KEY")
	}
	if !strings.Contains(result, "SendEnv=CHOKIDAR_USEPOLLING") {
		t.Error("Expected SendEnv for CHOKIDAR_USEPOLLING")
	}
	if !strings.Contains(result, "SendEnv=HTTP_PROXY") {
		t.Error("Expected SendEnv for HTTP_PROXY")
	}
	if !strings.Contains(result, "SendEnv=no_proxy") {
		t.Error("Expected SendEnv for no_proxy")
	}
	if !strings.Contains(result, "-t 'cd /vagrant && npm test'") {
		t.Error("Expected command to end with -t and cd /vagrant")
	}
}

// TestWrapCommandOrderOfFlags tests that flags appear in the correct order
func TestWrapCommandOrderOfFlags(t *testing.T) {
	m := &Manager{
		projectPath: "/project",
		settings:    session.VagrantSettings{},
	}

	result := m.WrapCommand("npm test", []string{"API_KEY"}, []int{8080})

	// Expected order: vagrant ssh -- <tunnels> <sendenv> -t '<command>'
	// Extract the part between "vagrant ssh --" and "-t"
	parts := strings.Split(result, "-t ")
	if len(parts) != 2 {
		t.Fatal("Expected result to contain exactly one -t flag")
	}

	flagsPart := parts[0]

	// Find positions of first tunnel and first SendEnv
	tunnelPos := strings.Index(flagsPart, "-R ")
	sendenvPos := strings.Index(flagsPart, "-o SendEnv=")

	if tunnelPos == -1 {
		t.Error("Expected to find -R flag")
	}
	if sendenvPos == -1 {
		t.Error("Expected to find -o SendEnv flag")
	}

	// Tunnels should come before SendEnv
	if tunnelPos >= sendenvPos {
		t.Errorf("Expected tunnels (-R) to come before SendEnv (-o), but got tunnelPos=%d, sendenvPos=%d", tunnelPos, sendenvPos)
	}
}

// TestWrapCommandQuoting tests that the command is properly quoted
func TestWrapCommandQuoting(t *testing.T) {
	m := &Manager{
		projectPath: "/project",
		settings:    session.VagrantSettings{},
	}

	// Command with special characters that need quoting
	result := m.WrapCommand("echo 'hello world'", nil, nil)

	// The entire command should be wrapped in single quotes
	if !strings.HasSuffix(result, "'cd /vagrant && echo 'hello world''") {
		t.Errorf("Expected command to be properly quoted, got: %s", result)
	}
}

// TestWrapCommandDefaultProxyForwarding tests that proxy forwarding defaults to true
func TestWrapCommandDefaultProxyForwarding(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://proxy.example.com:8080")

	// ForwardProxyEnv is nil (not set), should default to true
	m := &Manager{
		projectPath: "/project",
		settings:    session.VagrantSettings{},
	}

	result := m.WrapCommand("npm start", nil, nil)

	// Should contain HTTP_PROXY by default
	if !strings.Contains(result, "HTTP_PROXY") {
		t.Error("Expected HTTP_PROXY to be forwarded by default when ForwardProxyEnv is nil")
	}
}

// Benchmark for WrapCommand performance
func BenchmarkWrapCommand(b *testing.B) {
	forwardProxyEnv := true
	m := &Manager{
		projectPath: "/project",
		settings: session.VagrantSettings{
			SyncedFolderType: "virtualbox",
			ForwardProxyEnv:  &forwardProxyEnv,
		},
	}

	envVars := []string{"API_KEY", "ZED_KEY", "ANTHROPIC_API_KEY"}
	tunnels := []int{8080, 3000, 5000}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.WrapCommand("npm test", envVars, tunnels)
	}
}

// TestWrapCommandEnvVarDeduplication tests that duplicate env vars are deduplicated
func TestWrapCommandEnvVarDeduplication(t *testing.T) {
	m := &Manager{
		projectPath: "/project",
		settings:    session.VagrantSettings{},
	}

	// Pass duplicate env var names
	result := m.WrapCommand("npm start", []string{"API_KEY", "API_KEY", "ZED_KEY"}, nil)

	// Count occurrences of API_KEY
	count := strings.Count(result, "SendEnv=API_KEY")
	if count != 1 {
		t.Errorf("Expected API_KEY to appear exactly once, got %d occurrences", count)
	}

	// Should still have ZED_KEY
	if !strings.Contains(result, "SendEnv=ZED_KEY") {
		t.Error("Expected ZED_KEY to be included")
	}
}

// TestWrapCommandProxyVarDeduplication tests that proxy vars don't duplicate user-provided vars
func TestWrapCommandProxyVarDeduplication(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://proxy.example.com:8080")

	forwardProxyEnv := true
	m := &Manager{
		projectPath: "/project",
		settings: session.VagrantSettings{
			ForwardProxyEnv: &forwardProxyEnv,
		},
	}

	// User explicitly passes HTTP_PROXY
	result := m.WrapCommand("npm start", []string{"HTTP_PROXY"}, nil)

	// Should appear exactly once
	count := strings.Count(result, "SendEnv=HTTP_PROXY")
	if count != 1 {
		t.Errorf("Expected HTTP_PROXY to appear exactly once (no duplication), got %d", count)
	}
}

// TestWrapCommandPollingVarDeduplication tests that polling vars don't duplicate user-provided vars
func TestWrapCommandPollingVarDeduplication(t *testing.T) {
	m := &Manager{
		projectPath: "/project",
		settings: session.VagrantSettings{
			SyncedFolderType: "virtualbox",
		},
	}

	// User explicitly passes CHOKIDAR_USEPOLLING
	result := m.WrapCommand("npm start", []string{"CHOKIDAR_USEPOLLING"}, nil)

	// Should appear exactly once
	count := strings.Count(result, "SendEnv=CHOKIDAR_USEPOLLING")
	if count != 1 {
		t.Errorf("Expected CHOKIDAR_USEPOLLING to appear exactly once (no duplication), got %d", count)
	}
}

// Example demonstrating typical usage
func ExampleManager_WrapCommand() {
	forwardProxyEnv := true
	m := &Manager{
		projectPath: "/home/user/myproject",
		settings: session.VagrantSettings{
			SyncedFolderType: "virtualbox",
			ForwardProxyEnv:  &forwardProxyEnv,
		},
	}

	wrapped := m.WrapCommand(
		"claude code",
		[]string{"ANTHROPIC_API_KEY"},
		[]int{30000},
	)

	fmt.Println(wrapped)
	// Output will include vagrant ssh -- with tunnels, env vars, and the command
}
