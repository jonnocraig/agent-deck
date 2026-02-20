package vagrant

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// healthCache stores the most recent health check result with TTL
type healthCache struct {
	mu        sync.RWMutex
	lastCheck time.Time
	result    VMHealth
	ttl       time.Duration
}

// getIfValid returns the cached result and true if still within TTL, or zero value and false if invalid.
// This method atomically checks validity and retrieves the result under a single lock to prevent TOCTOU races.
func (c *healthCache) getIfValid() (VMHealth, bool) {
	if c == nil {
		return VMHealth{}, false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()

	if time.Since(c.lastCheck) >= c.ttl {
		return VMHealth{}, false
	}

	// Return a copy to avoid data races on the returned value
	return c.result, true
}

// set updates the cache with a new result
func (c *healthCache) set(result VMHealth) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.result = result
	c.lastCheck = time.Now()
}

// initCache initializes the health cache exactly once using sync.Once for thread safety
func (m *Manager) initCache() {
	m.cacheOnce.Do(func() {
		m.cache = &healthCache{
			ttl: 30 * time.Second,
		}
	})
}

// HealthCheck performs comprehensive health check including SSH liveness probe.
// Phase 1: Check vagrant status (cached for 30s)
// Phase 2: If running, perform SSH liveness probe (always fresh)
func (m *Manager) HealthCheck() (VMHealth, error) {
	m.initCache()

	// Check cache for Phase 1 result
	if cached, valid := m.cache.getIfValid(); valid {
		// If cached state is "running", we still want to do Phase 2 (SSH probe)
		// But for non-running states, we can return cached result
		if cached.State != "running" {
			return cached, nil
		}
	}

	// Phase 1: Run vagrant status
	state, err := m.runVagrantStatus()
	if err != nil {
		return VMHealth{}, fmt.Errorf("vagrant status failed: %w", err)
	}

	// If state is not "running", we can skip Phase 2
	if state != "running" {
		health := buildVMHealth(state, false, nil)
		m.cache.set(health)
		return health, nil
	}

	// Phase 2: SSH liveness probe (only for running VMs)
	sshSuccess, sshErr := m.runSSHProbe()
	health := buildVMHealth(state, sshSuccess, sshErr)

	// Update cache
	m.cache.set(health)

	return health, nil
}

// runVagrantStatus executes vagrant status and parses the state
func (m *Manager) runVagrantStatus() (string, error) {
	cmd := m.vagrantCmd("status", "--machine-readable")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("vagrant status command failed: %w (output: %s)", err, string(output))
	}

	state := parseStateFromOutput(string(output))
	if state == "" {
		return "", fmt.Errorf("failed to parse vagrant state from output: %s", string(output))
	}

	return state, nil
}

// runSSHProbe runs vagrant ssh probe with 5 second timeout
func (m *Manager) runSSHProbe() (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := m.vagrantCmdContext(ctx, "ssh", "-c", "echo pong")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, err
	}

	// Check if we got "pong" back
	if strings.Contains(string(output), "pong") {
		return true, nil
	}

	return false, fmt.Errorf("unexpected SSH probe response: %s", string(output))
}

// parseStateFromOutput parses machine-readable vagrant status output
// Format: timestamp,target,state,VALUE
func parseStateFromOutput(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, ",")
		if len(parts) < 4 {
			continue
		}

		// Look for state line: timestamp,target,state,VALUE
		if parts[2] == "state" {
			return parts[3]
		}
	}
	return ""
}

// buildVMHealth constructs VMHealth from state and SSH probe result
func buildVMHealth(state string, sshSuccess bool, sshErr error) VMHealth {
	health := VMHealth{
		State: state,
	}

	switch state {
	case "running":
		if sshSuccess {
			health.Healthy = true
			health.Responsive = true
			health.Message = vmStateMessage(state)
		} else {
			health.Healthy = false
			health.Responsive = false
			health.Message = "VM running but unresponsive (SSH probe failed)"
		}
	case "saved":
		health.Healthy = false
		health.Responsive = false
		health.Message = vmStateMessage(state)
	case "not_created":
		health.Healthy = false
		health.Responsive = false
		health.Message = vmStateMessage(state)
	case "aborted":
		health.Healthy = false
		health.Responsive = false
		health.Message = vmStateMessage(state)
	case "poweroff":
		health.Healthy = false
		health.Responsive = false
		health.Message = vmStateMessage(state)
	default:
		health.Healthy = false
		health.Responsive = false
		health.Message = vmStateMessage(state)
	}

	return health
}

// vmStateMessage maps vagrant state strings to user-friendly messages
func vmStateMessage(state string) string {
	switch state {
	case "running":
		return "VM running and responsive"
	case "saved":
		return "VM is suspended"
	case "not_created":
		return "VM not created"
	case "aborted":
		return "VM crashed or was force-stopped"
	case "poweroff":
		return "VM is powered off"
	default:
		return fmt.Sprintf("VM in unexpected state: %s", state)
	}
}
