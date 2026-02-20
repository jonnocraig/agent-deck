package vagrant

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/asheshgoplani/agent-deck/internal/session"
)

// mockMCPs returns a test MCP configuration map
func mockMCPs() map[string]session.MCPDef {
	return map[string]session.MCPDef{
		// STDIO MCPs
		"memory": {
			Command: "npx",
			Args:    []string{"-y", "@modelcontextprotocol/server-memory"},
			Env:     map[string]string{"DEBUG": "true"},
		},
		"filesystem": {
			Command: "npx",
			Args:    []string{"-y", "@modelcontextprotocol/server-filesystem", "/tmp"},
			Env:     map[string]string{"MAX_SIZE": "10MB"},
		},
		// HTTP MCP - localhost
		"exa": {
			URL:       "http://localhost:8001/mcp",
			Transport: "http",
			Headers:   map[string]string{"Authorization": "Bearer token123"},
		},
		// HTTP MCP - 127.0.0.1
		"github": {
			URL:       "http://127.0.0.1:8002/api",
			Transport: "http",
		},
		// HTTP MCP - remote (should be skipped by port collection)
		"remote-api": {
			URL:       "https://api.example.com/mcp",
			Transport: "http",
		},
		// STDIO with non-npx command
		"custom": {
			Command: "python",
			Args:    []string{"-m", "my_mcp_server"},
			Env:     map[string]string{"PYTHON_ENV": "production"},
		},
		// HTTP with no port (default 80)
		"http-no-port": {
			URL:       "http://localhost/mcp",
			Transport: "http",
		},
	}
}

func TestCollectHTTPMCPPorts(t *testing.T) {
	tests := []struct {
		name         string
		enabledNames []string
		want         []int
	}{
		{
			name:         "localhost URLs",
			enabledNames: []string{"exa"},
			want:         []int{8001},
		},
		{
			name:         "127.0.0.1 URLs",
			enabledNames: []string{"github"},
			want:         []int{8002},
		},
		{
			name:         "mixed localhost and 127.0.0.1",
			enabledNames: []string{"exa", "github"},
			want:         []int{8001, 8002},
		},
		{
			name:         "remote URLs skipped",
			enabledNames: []string{"remote-api"},
			want:         []int{},
		},
		{
			name:         "no port uses default 80",
			enabledNames: []string{"http-no-port"},
			want:         []int{80},
		},
		{
			name:         "stdio MCPs skipped",
			enabledNames: []string{"memory", "filesystem"},
			want:         []int{},
		},
		{
			name:         "mixed stdio and HTTP",
			enabledNames: []string{"memory", "exa", "github"},
			want:         []int{8001, 8002},
		},
		{
			name:         "empty enabled list",
			enabledNames: []string{},
			want:         []int{},
		},
		{
			name:         "non-existent MCP names",
			enabledNames: []string{"nonexistent"},
			want:         []int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Override GetAvailableMCPs for testing
			originalFunc := getAvailableMCPsFunc
			getAvailableMCPsFunc = mockMCPs
			defer func() { getAvailableMCPsFunc = originalFunc }()

			got := CollectHTTPMCPPorts(tt.enabledNames)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CollectHTTPMCPPorts() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCollectHTTPMCPPortsDedup(t *testing.T) {
	// Create MCPs with duplicate ports
	duplicateMCPs := map[string]session.MCPDef{
		"server1": {URL: "http://localhost:8001/mcp", Transport: "http"},
		"server2": {URL: "http://127.0.0.1:8001/api", Transport: "http"},
		"server3": {URL: "http://localhost:8002/v1", Transport: "http"},
	}

	originalFunc := getAvailableMCPsFunc
	getAvailableMCPsFunc = func() map[string]session.MCPDef { return duplicateMCPs }
	defer func() { getAvailableMCPsFunc = originalFunc }()

	got := CollectHTTPMCPPorts([]string{"server1", "server2", "server3"})
	want := []int{8001, 8002}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("CollectHTTPMCPPorts() with duplicates = %v, want %v", got, want)
	}
}

func TestCollectEnvVarNames(t *testing.T) {
	tests := []struct {
		name         string
		enabledNames []string
		vagrantEnv   map[string]string
		want         []string
	}{
		{
			name:         "MCP env only",
			enabledNames: []string{"memory"},
			vagrantEnv:   nil,
			want:         []string{"ANTHROPIC_API_KEY", "DEBUG"},
		},
		{
			name:         "vagrant env only",
			enabledNames: []string{},
			vagrantEnv:   map[string]string{"CUSTOM_VAR": "value"},
			want:         []string{"ANTHROPIC_API_KEY", "CUSTOM_VAR"},
		},
		{
			name:         "merged MCP and vagrant env",
			enabledNames: []string{"memory", "filesystem"},
			vagrantEnv:   map[string]string{"CUSTOM_VAR": "value"},
			want:         []string{"ANTHROPIC_API_KEY", "CUSTOM_VAR", "DEBUG", "MAX_SIZE"},
		},
		{
			name:         "duplicate env var names",
			enabledNames: []string{"memory"},
			vagrantEnv:   map[string]string{"DEBUG": "false"},
			want:         []string{"ANTHROPIC_API_KEY", "DEBUG"},
		},
		{
			name:         "ANTHROPIC_API_KEY always included",
			enabledNames: []string{},
			vagrantEnv:   nil,
			want:         []string{"ANTHROPIC_API_KEY"},
		},
		{
			name:         "HTTP MCPs env included",
			enabledNames: []string{"exa"},
			vagrantEnv:   nil,
			want:         []string{"ANTHROPIC_API_KEY"},
		},
		{
			name:         "non-npx STDIO MCPs env included",
			enabledNames: []string{"custom"},
			vagrantEnv:   nil,
			want:         []string{"ANTHROPIC_API_KEY", "PYTHON_ENV"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalFunc := getAvailableMCPsFunc
			getAvailableMCPsFunc = mockMCPs
			defer func() { getAvailableMCPsFunc = originalFunc }()

			got := CollectEnvVarNames(tt.enabledNames, tt.vagrantEnv)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CollectEnvVarNames() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetMCPPackages(t *testing.T) {
	tests := []struct {
		name         string
		enabledNames []string
		want         []string
	}{
		{
			name:         "npx -y MCPs",
			enabledNames: []string{"memory", "filesystem"},
			want:         []string{"@modelcontextprotocol/server-filesystem", "@modelcontextprotocol/server-memory"},
		},
		{
			name:         "non-npx MCPs excluded",
			enabledNames: []string{"custom"},
			want:         nil,
		},
		{
			name:         "HTTP MCPs excluded",
			enabledNames: []string{"exa", "github"},
			want:         nil,
		},
		{
			name:         "mixed npx and non-npx",
			enabledNames: []string{"memory", "custom", "exa"},
			want:         []string{"@modelcontextprotocol/server-memory"},
		},
		{
			name:         "empty enabled list",
			enabledNames: []string{},
			want:         nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Override getAvailableMCPsFunc
			originalFunc := getAvailableMCPsFunc
			getAvailableMCPsFunc = mockMCPs
			defer func() { getAvailableMCPsFunc = originalFunc }()

			// Test the package-level helper function
			got := extractNpxPackages(tt.enabledNames, mockMCPs())

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("extractNpxPackages() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWriteMCPJsonForVagrant(t *testing.T) {
	tests := []struct {
		name         string
		enabledNames []string
		existing     string // existing .mcp.json content
		wantErr      bool
		validate     func(t *testing.T, mcpFile string)
	}{
		{
			name:         "STDIO MCPs use stdio transport",
			enabledNames: []string{"memory"},
			wantErr:      false,
			validate: func(t *testing.T, mcpFile string) {
				var config struct {
					MCPServers map[string]session.MCPServerConfig `json:"mcpServers"`
				}
				data, err := os.ReadFile(mcpFile)
				if err != nil {
					t.Fatalf("Failed to read .mcp.json: %v", err)
				}
				if err := json.Unmarshal(data, &config); err != nil {
					t.Fatalf("Failed to parse .mcp.json: %v", err)
				}

				memory, ok := config.MCPServers["memory"]
				if !ok {
					t.Fatal("memory MCP not found")
				}
				if memory.Type != "stdio" {
					t.Errorf("memory.Type = %q, want %q", memory.Type, "stdio")
				}
				if memory.Command != "npx" {
					t.Errorf("memory.Command = %q, want %q", memory.Command, "npx")
				}
				if len(memory.Args) != 2 || memory.Args[0] != "-y" {
					t.Errorf("memory.Args = %v, want [-y ...]", memory.Args)
				}
				if memory.Env["DEBUG"] != "true" {
					t.Errorf("memory.Env[DEBUG] = %q, want %q", memory.Env["DEBUG"], "true")
				}
			},
		},
		{
			name:         "HTTP MCPs use URL transport",
			enabledNames: []string{"exa"},
			wantErr:      false,
			validate: func(t *testing.T, mcpFile string) {
				var config struct {
					MCPServers map[string]session.MCPServerConfig `json:"mcpServers"`
				}
				data, err := os.ReadFile(mcpFile)
				if err != nil {
					t.Fatalf("Failed to read .mcp.json: %v", err)
				}
				if err := json.Unmarshal(data, &config); err != nil {
					t.Fatalf("Failed to parse .mcp.json: %v", err)
				}

				exa, ok := config.MCPServers["exa"]
				if !ok {
					t.Fatal("exa MCP not found")
				}
				if exa.Type != "http" {
					t.Errorf("exa.Type = %q, want %q", exa.Type, "http")
				}
				if exa.URL != "http://localhost:8001/mcp" {
					t.Errorf("exa.URL = %q, want %q", exa.URL, "http://localhost:8001/mcp")
				}
				if exa.Headers["Authorization"] != "Bearer token123" {
					t.Errorf("exa.Headers[Authorization] = %q, want %q", exa.Headers["Authorization"], "Bearer token123")
				}
			},
		},
		{
			name:         "preserves non-agent-deck entries",
			enabledNames: []string{"memory"},
			existing: `{
				"mcpServers": {
					"custom-mcp": {
						"command": "custom-command",
						"args": ["--flag"]
					}
				}
			}`,
			wantErr: false,
			validate: func(t *testing.T, mcpFile string) {
				var config struct {
					MCPServers map[string]session.MCPServerConfig `json:"mcpServers"`
				}
				data, err := os.ReadFile(mcpFile)
				if err != nil {
					t.Fatalf("Failed to read .mcp.json: %v", err)
				}
				if err := json.Unmarshal(data, &config); err != nil {
					t.Fatalf("Failed to parse .mcp.json: %v", err)
				}

				// Check that custom-mcp is preserved
				custom, ok := config.MCPServers["custom-mcp"]
				if !ok {
					t.Fatal("custom-mcp not preserved")
				}
				if custom.Command != "custom-command" {
					t.Errorf("custom.Command = %q, want %q", custom.Command, "custom-command")
				}

				// Check that memory was added
				if _, ok := config.MCPServers["memory"]; !ok {
					t.Fatal("memory MCP not added")
				}
			},
		},
		{
			name:         "overwrites existing agent-deck entries",
			enabledNames: []string{"memory"},
			existing: `{
				"mcpServers": {
					"memory": {
						"command": "old-command",
						"args": []
					}
				}
			}`,
			wantErr: false,
			validate: func(t *testing.T, mcpFile string) {
				var config struct {
					MCPServers map[string]session.MCPServerConfig `json:"mcpServers"`
				}
				data, err := os.ReadFile(mcpFile)
				if err != nil {
					t.Fatalf("Failed to read .mcp.json: %v", err)
				}
				if err := json.Unmarshal(data, &config); err != nil {
					t.Fatalf("Failed to parse .mcp.json: %v", err)
				}

				memory, ok := config.MCPServers["memory"]
				if !ok {
					t.Fatal("memory MCP not found")
				}
				if memory.Command != "npx" {
					t.Errorf("memory.Command = %q, want %q (should be overwritten)", memory.Command, "npx")
				}
			},
		},
		{
			name:         "empty enabled list creates empty servers",
			enabledNames: []string{},
			wantErr:      false,
			validate: func(t *testing.T, mcpFile string) {
				var config struct {
					MCPServers map[string]session.MCPServerConfig `json:"mcpServers"`
				}
				data, err := os.ReadFile(mcpFile)
				if err != nil {
					t.Fatalf("Failed to read .mcp.json: %v", err)
				}
				if err := json.Unmarshal(data, &config); err != nil {
					t.Fatalf("Failed to parse .mcp.json: %v", err)
				}

				if len(config.MCPServers) != 0 {
					t.Errorf("len(MCPServers) = %d, want 0", len(config.MCPServers))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// Write existing .mcp.json if provided
			if tt.existing != "" {
				mcpFile := filepath.Join(tmpDir, ".mcp.json")
				if err := os.WriteFile(mcpFile, []byte(tt.existing), 0644); err != nil {
					t.Fatalf("Failed to create test .mcp.json: %v", err)
				}
			}

			// Override getAvailableMCPsFunc
			originalFunc := getAvailableMCPsFunc
			getAvailableMCPsFunc = mockMCPs
			defer func() { getAvailableMCPsFunc = originalFunc }()

			err := WriteMCPJsonForVagrant(tmpDir, tt.enabledNames)
			if (err != nil) != tt.wantErr {
				t.Errorf("WriteMCPJsonForVagrant() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.validate != nil {
				mcpFile := filepath.Join(tmpDir, ".mcp.json")
				tt.validate(t, mcpFile)
			}
		})
	}
}
