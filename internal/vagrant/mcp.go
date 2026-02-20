package vagrant

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/asheshgoplani/agent-deck/internal/session"
)

// getAvailableMCPsFunc is a variable that allows tests to override GetAvailableMCPs
var getAvailableMCPsFunc = session.GetAvailableMCPs

// WriteMCPJsonForVagrant writes .mcp.json for vagrant mode sessions.
// Unlike session.WriteMCPJsonFromConfig, this ALWAYS uses stdio for STDIO MCPs
// (no pool socket references) and writes HTTP/SSE URLs as-is for SSH reverse tunnels.
func WriteMCPJsonForVagrant(projectPath string, enabledNames []string) error {
	mcpFile := filepath.Join(projectPath, ".mcp.json")
	availableMCPs := getAvailableMCPsFunc()

	// Read existing .mcp.json to preserve entries not managed by agent-deck
	existingServers := readExistingLocalMCPServers(mcpFile)

	// Build agent-deck managed MCP entries
	agentDeckServers := make(map[string]session.MCPServerConfig)

	for _, name := range enabledNames {
		def, ok := availableMCPs[name]
		if !ok {
			continue
		}

		// Check if this is an HTTP/SSE MCP (has URL configured)
		if def.URL != "" {
			transport := def.Transport
			if transport == "" {
				transport = "http" // default to http if URL is set
			}
			agentDeckServers[name] = session.MCPServerConfig{
				Type:    transport,
				URL:     def.URL,
				Headers: def.Headers,
			}
			continue
		}

		// STDIO MCP - always use plain stdio (no pool sockets for vagrant)
		args := def.Args
		if args == nil {
			args = []string{}
		}
		env := def.Env
		if env == nil {
			env = map[string]string{}
		}
		agentDeckServers[name] = session.MCPServerConfig{
			Type:    "stdio",
			Command: def.Command,
			Args:    args,
			Env:     env,
		}
	}

	// Merge: preserve non-agent-deck entries, then add agent-deck entries
	mergedServers := make(map[string]json.RawMessage)
	for name, raw := range existingServers {
		if _, managed := availableMCPs[name]; !managed {
			mergedServers[name] = raw
		}
	}
	for name, cfg := range agentDeckServers {
		raw, err := json.Marshal(cfg)
		if err != nil {
			continue
		}
		mergedServers[name] = raw
	}

	finalConfig := struct {
		MCPServers map[string]json.RawMessage `json:"mcpServers"`
	}{
		MCPServers: mergedServers,
	}

	data, err := json.MarshalIndent(finalConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal .mcp.json: %w", err)
	}

	// Atomic write
	tmpPath := mcpFile + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write .mcp.json: %w", err)
	}

	if err := os.Rename(tmpPath, mcpFile); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to save .mcp.json: %w", err)
	}

	return nil
}

// readExistingLocalMCPServers reads mcpServers from an existing .mcp.json file.
// Returns nil if the file doesn't exist or can't be parsed.
func readExistingLocalMCPServers(mcpFile string) map[string]json.RawMessage {
	data, err := os.ReadFile(mcpFile)
	if err != nil {
		return nil
	}
	var config struct {
		MCPServers map[string]json.RawMessage `json:"mcpServers"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil
	}
	return config.MCPServers
}

// CollectHTTPMCPPorts extracts unique port numbers from HTTP/SSE MCP URLs
// that reference localhost or 127.0.0.1. This is used to set up SSH reverse
// tunnels for vagrant mode.
func CollectHTTPMCPPorts(enabledNames []string) []int {
	availableMCPs := getAvailableMCPsFunc()
	portSet := make(map[int]bool)

	for _, name := range enabledNames {
		def, ok := availableMCPs[name]
		if !ok || def.URL == "" {
			continue
		}

		// Parse URL to extract host and port
		parsedURL, err := url.Parse(def.URL)
		if err != nil {
			continue
		}

		// Check if host is localhost or 127.0.0.1
		host := parsedURL.Hostname()
		if host != "localhost" && host != "127.0.0.1" {
			continue
		}

		// Extract port
		port := parsedURL.Port()
		if port == "" {
			// Default HTTP port
			if parsedURL.Scheme == "http" {
				portSet[80] = true
			} else if parsedURL.Scheme == "https" {
				portSet[443] = true
			}
		} else {
			// Parse port number
			var portNum int
			if _, err := fmt.Sscanf(port, "%d", &portNum); err == nil {
				portSet[portNum] = true
			}
		}
	}

	// Convert map to sorted slice
	ports := make([]int, 0, len(portSet))
	for port := range portSet {
		ports = append(ports, port)
	}
	sort.Ints(ports)

	return ports
}

// CollectEnvVarNames merges env var names from:
// 1. MCP definitions' Env maps (from enabled MCPs)
// 2. User's vagrant.env map
// 3. Always includes "ANTHROPIC_API_KEY"
// Returns sorted, deduplicated list of env var NAMES (not values).
func CollectEnvVarNames(enabledNames []string, vagrantEnv map[string]string) []string {
	availableMCPs := getAvailableMCPsFunc()
	nameSet := make(map[string]bool)

	// Always include ANTHROPIC_API_KEY
	nameSet["ANTHROPIC_API_KEY"] = true

	// Collect env var names from enabled MCPs
	for _, name := range enabledNames {
		def, ok := availableMCPs[name]
		if !ok {
			continue
		}

		// Add env var names from MCP definition
		for envName := range def.Env {
			nameSet[envName] = true
		}
	}

	// Collect env var names from vagrant.env
	for envName := range vagrantEnv {
		nameSet[envName] = true
	}

	// Convert map to sorted slice
	names := make([]string, 0, len(nameSet))
	for name := range nameSet {
		names = append(names, name)
	}
	sort.Strings(names)

	return names
}

// GetMCPPackages extracts npm package names from npx-based STDIO MCPs.
// For MCPs where Command == "npx" and Args contains "-y", the arg after "-y"
// is the package name. Returns sorted, deduplicated list.
func (m *Manager) GetMCPPackages() []string {
	availableMCPs := getAvailableMCPsFunc()
	packages := make(map[string]bool)

	for _, def := range availableMCPs {
		// Check if this is an npx MCP with -y flag
		if def.Command == "npx" && len(def.Args) >= 2 && def.Args[0] == "-y" {
			packages[def.Args[1]] = true
		}
	}

	if len(packages) == 0 {
		return nil
	}

	// Convert map to sorted slice
	result := make([]string, 0, len(packages))
	for pkg := range packages {
		result = append(result, pkg)
	}
	sort.Strings(result)

	return result
}

// extractNpxPackages is a helper function that extracts npx packages from
// a specific list of enabled MCPs. This is used for generating Vagrantfile
// provisioning scripts with only the packages needed for a specific session.
func extractNpxPackages(enabledNames []string, mcps map[string]session.MCPDef) []string {
	packages := make(map[string]bool)

	for _, name := range enabledNames {
		def, ok := mcps[name]
		if !ok {
			continue
		}

		// Check if this is an npx MCP with -y flag
		if def.Command == "npx" && len(def.Args) >= 2 && def.Args[0] == "-y" {
			pkg := def.Args[1]
			// Handle scoped packages and version specifiers
			// e.g., "@scope/package@version" -> "@scope/package"
			if idx := strings.LastIndex(pkg, "@"); idx > 0 && pkg[0] == '@' {
				// Scoped package with version: @scope/package@version
				pkg = pkg[:idx]
			} else if idx := strings.LastIndex(pkg, "@"); idx > 0 {
				// Unscoped package with version: package@version
				pkg = pkg[:idx]
			}
			packages[pkg] = true
		}
	}

	if len(packages) == 0 {
		return nil
	}

	// Convert map to sorted slice
	result := make([]string, 0, len(packages))
	for pkg := range packages {
		result = append(result, pkg)
	}
	sort.Strings(result)

	return result
}
