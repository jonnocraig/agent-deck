package vagrant

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

// WrapCommand wraps a command for execution inside the VM via 'vagrant ssh --'.
// It handles SSH environment variable forwarding, reverse tunnel setup, and PTY allocation.
//
// The output format:
//
//	vagrant ssh -- -R PORT:localhost:PORT ... -o SendEnv=VAR ... -t 'cd /vagrant && <cmd>'
//
// Components:
//  1. Base command: vagrant ssh --
//  2. Reverse tunnels: -R PORT:localhost:PORT (one flag per tunnel port)
//  3. Env var forwarding: -o SendEnv=VAR (one flag per env var name)
//  4. PTY allocation: -t (always included for interactive Claude sessions)
//  5. Command execution: 'cd /vagrant && <cmd>' (change to synced folder and run command)
//
// envVarNames: environment variables to forward via SendEnv
// tunnelPorts: local ports to reverse tunnel into the VM
func (m *Manager) WrapCommand(cmd string, envVarNames []string, tunnelPorts []int) string {
	var parts []string
	parts = append(parts, "vagrant ssh --")

	// 1. Add reverse tunnels (sorted and deduplicated)
	if len(tunnelPorts) > 0 {
		uniquePorts := deduplicateInts(tunnelPorts)
		sort.Ints(uniquePorts)
		for _, port := range uniquePorts {
			parts = append(parts, fmt.Sprintf("-R %d:localhost:%d", port, port))
		}
	}

	// 2. Collect all env var names to forward
	envVars := make([]string, 0, len(envVarNames))
	envVars = append(envVars, envVarNames...)

	// Auto-inject polling env vars for VirtualBox shared folders
	if m.settings.SyncedFolderType == "virtualbox" {
		envVars = append(envVars, "CHOKIDAR_USEPOLLING", "WATCHPACK_POLLING", "TSC_WATCHFILE")
	}

	// Auto-inject proxy env vars if enabled (default: true)
	forwardProxyEnv := m.settings.ForwardProxyEnv == nil || *m.settings.ForwardProxyEnv
	if forwardProxyEnv {
		proxyVars := detectProxyEnvVars()
		envVars = append(envVars, proxyVars...)
	}

	// Deduplicate and sort env var names
	if len(envVars) > 0 {
		uniqueEnvVars := deduplicateStrings(envVars)
		sort.Strings(uniqueEnvVars)
		for _, envVar := range uniqueEnvVars {
			parts = append(parts, fmt.Sprintf("-o SendEnv=%s", envVar))
		}
	}

	// 3. Add PTY allocation flag (always included)
	parts = append(parts, "-t")

	// 4. Add command wrapped with cd /vagrant
	wrappedCmd := fmt.Sprintf("'cd /vagrant && %s'", cmd)
	parts = append(parts, wrappedCmd)

	return strings.Join(parts, " ")
}

// detectProxyEnvVars checks for set proxy environment variables and returns their names.
// Only returns names of env vars that are actually set (non-empty).
// Checks both uppercase and lowercase variants.
func detectProxyEnvVars() []string {
	proxyVarNames := []string{
		"HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY",
		"http_proxy", "https_proxy", "no_proxy",
	}

	var found []string
	for _, name := range proxyVarNames {
		if os.Getenv(name) != "" {
			found = append(found, name)
		}
	}
	return found
}

// deduplicateInts removes duplicate integers from a slice
func deduplicateInts(slice []int) []int {
	seen := make(map[int]bool)
	result := make([]int, 0, len(slice))
	for _, val := range slice {
		if !seen[val] {
			seen[val] = true
			result = append(result, val)
		}
	}
	return result
}

// deduplicateStrings removes duplicate strings from a slice
func deduplicateStrings(slice []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(slice))
	for _, val := range slice {
		if !seen[val] {
			seen[val] = true
			result = append(result, val)
		}
	}
	return result
}
