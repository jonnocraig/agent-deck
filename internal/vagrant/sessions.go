package vagrant

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Manager struct is defined in manager.go

// lockfileData represents the JSON structure persisted to disk
type lockfileData struct {
	Sessions []string `json:"sessions"`
}

// RegisterSession adds a session to the active sessions list.
// Thread-safe and idempotent - won't add duplicates.
// Persists the session list to a lockfile.
// Returns error if lockfile write fails.
func (m *Manager) RegisterSession(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check for duplicates
	for _, id := range m.sessions {
		if id == sessionID {
			return nil
		}
	}

	m.sessions = append(m.sessions, sessionID)
	if err := m.writeLockfile(); err != nil {
		return fmt.Errorf("register session %s: %w", sessionID, err)
	}
	return nil
}

// UnregisterSession removes a session from the active sessions list.
// Thread-safe and uses immutable pattern (creates new slice).
// Persists the updated session list to a lockfile.
// Returns error if lockfile write fails.
func (m *Manager) UnregisterSession(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Create new slice without the specified session (immutable pattern)
	filtered := make([]string, 0, len(m.sessions))
	for _, id := range m.sessions {
		if id != sessionID {
			filtered = append(filtered, id)
		}
	}

	m.sessions = filtered
	if err := m.writeLockfile(); err != nil {
		return fmt.Errorf("unregister session %s: %w", sessionID, err)
	}
	return nil
}

// SessionCount returns the number of active sessions.
// Thread-safe.
func (m *Manager) SessionCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.sessions)
}

// IsLastSession returns true if the given session is the only remaining session.
// Thread-safe.
func (m *Manager) IsLastSession(sessionID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.sessions) == 1 && m.sessions[0] == sessionID
}

// SetDotfilePath sets the VAGRANT_DOTFILE_PATH for isolated VM state per session.
// This allows multiple sessions to share the same VM without conflicts.
// Thread-safe.
func (m *Manager) SetDotfilePath(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.dotfilePath = filepath.Join(m.projectPath, ".vagrant-"+sessionID)
}

// writeLockfile persists the current session list to a JSON lockfile.
// MUST be called under mutex lock.
// Creates .vagrant directory if it doesn't exist.
// Returns error if directory creation, JSON marshaling, or file write fails.
func (m *Manager) writeLockfile() error {
	lockPath := filepath.Join(m.projectPath, ".vagrant", "agent-deck.lock")

	// Ensure .vagrant directory exists
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		return fmt.Errorf("failed to create .vagrant directory: %w", err)
	}

	data, err := json.Marshal(lockfileData{Sessions: m.sessions})
	if err != nil {
		return fmt.Errorf("failed to marshal lockfile data: %w", err)
	}

	// Write lockfile with restrictive permissions
	if err := os.WriteFile(lockPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write lockfile: %w", err)
	}

	return nil
}

// loadLockfile reads the session list from the JSON lockfile on startup.
// Silently ignores errors (e.g., file doesn't exist on first run).
func (m *Manager) loadLockfile() {
	lockPath := filepath.Join(m.projectPath, ".vagrant", "agent-deck.lock")

	data, err := os.ReadFile(lockPath)
	if err != nil {
		return
	}

	var lf lockfileData
	if err := json.Unmarshal(data, &lf); err != nil {
		return
	}

	m.sessions = lf.Sessions
}
