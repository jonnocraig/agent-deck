package vagrant

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// TestRegisterSession verifies that sessions are added to the list
func TestRegisterSession(t *testing.T) {
	tmpDir := t.TempDir()
	m := &Manager{
		projectPath: tmpDir,
		sessions:    []string{},
	}

	if err := m.RegisterSession("session-1"); err != nil {
		t.Fatalf("RegisterSession failed: %v", err)
	}

	if len(m.sessions) != 1 {
		t.Errorf("expected 1 session, got %d", len(m.sessions))
	}

	if m.sessions[0] != "session-1" {
		t.Errorf("expected session-1, got %s", m.sessions[0])
	}
}

// TestRegisterSessionDuplicate verifies that duplicate sessions are not added
func TestRegisterSessionDuplicate(t *testing.T) {
	tmpDir := t.TempDir()
	m := &Manager{
		projectPath: tmpDir,
		sessions:    []string{},
	}

	if err := m.RegisterSession("session-1"); err != nil {
		t.Fatalf("RegisterSession failed: %v", err)
	}
	if err := m.RegisterSession("session-1"); err != nil {
		t.Fatalf("RegisterSession duplicate failed: %v", err)
	}

	if len(m.sessions) != 1 {
		t.Errorf("expected 1 session after duplicate add, got %d", len(m.sessions))
	}
}

// TestUnregisterSession verifies that sessions are removed from the list
func TestUnregisterSession(t *testing.T) {
	tmpDir := t.TempDir()
	m := &Manager{
		projectPath: tmpDir,
		sessions:    []string{"session-1", "session-2"},
	}

	if err := m.UnregisterSession("session-1"); err != nil {
		t.Fatalf("UnregisterSession failed: %v", err)
	}

	if len(m.sessions) != 1 {
		t.Errorf("expected 1 session after unregister, got %d", len(m.sessions))
	}

	if m.sessions[0] != "session-2" {
		t.Errorf("expected session-2, got %s", m.sessions[0])
	}
}

// TestUnregisterSessionNotFound verifies that unregistering unknown session is a no-op
func TestUnregisterSessionNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	m := &Manager{
		projectPath: tmpDir,
		sessions:    []string{"session-1"},
	}

	if err := m.UnregisterSession("session-999"); err != nil {
		t.Fatalf("UnregisterSession failed: %v", err)
	}

	if len(m.sessions) != 1 {
		t.Errorf("expected 1 session after no-op unregister, got %d", len(m.sessions))
	}

	if m.sessions[0] != "session-1" {
		t.Errorf("expected session-1, got %s", m.sessions[0])
	}
}

// TestSessionCount verifies that the count is accurate
func TestSessionCount(t *testing.T) {
	tmpDir := t.TempDir()
	m := &Manager{
		projectPath: tmpDir,
		sessions:    []string{},
	}

	if m.SessionCount() != 0 {
		t.Errorf("expected 0 sessions, got %d", m.SessionCount())
	}

	if err := m.RegisterSession("session-1"); err != nil {
		t.Fatalf("RegisterSession failed: %v", err)
	}
	if err := m.RegisterSession("session-2"); err != nil {
		t.Fatalf("RegisterSession failed: %v", err)
	}

	if m.SessionCount() != 2 {
		t.Errorf("expected 2 sessions, got %d", m.SessionCount())
	}
}

// TestIsLastSession verifies that IsLastSession returns true only for the last session
func TestIsLastSession(t *testing.T) {
	tmpDir := t.TempDir()
	m := &Manager{
		projectPath: tmpDir,
		sessions:    []string{"session-1"},
	}

	if !m.IsLastSession("session-1") {
		t.Error("expected IsLastSession to return true for only session")
	}
}

// TestIsLastSessionFalse verifies that IsLastSession returns false when multiple sessions exist
func TestIsLastSessionFalse(t *testing.T) {
	tmpDir := t.TempDir()
	m := &Manager{
		projectPath: tmpDir,
		sessions:    []string{"session-1", "session-2"},
	}

	if m.IsLastSession("session-1") {
		t.Error("expected IsLastSession to return false when multiple sessions exist")
	}

	if m.IsLastSession("session-2") {
		t.Error("expected IsLastSession to return false when multiple sessions exist")
	}
}

// TestSetDotfilePath verifies that the dotfile path is set correctly
func TestSetDotfilePath(t *testing.T) {
	tmpDir := t.TempDir()
	m := &Manager{
		projectPath: tmpDir,
		sessions:    []string{},
	}

	m.SetDotfilePath("session-1")

	expected := filepath.Join(tmpDir, ".vagrant-session-1")
	if m.dotfilePath != expected {
		t.Errorf("expected dotfilePath %s, got %s", expected, m.dotfilePath)
	}
}

// TestWriteAndLoadLockfile verifies that session persistence works correctly
func TestWriteAndLoadLockfile(t *testing.T) {
	tmpDir := t.TempDir()
	m := &Manager{
		projectPath: tmpDir,
		sessions:    []string{"session-1", "session-2"},
	}

	// Write lockfile
	if err := m.writeLockfile(); err != nil {
		t.Fatalf("writeLockfile failed: %v", err)
	}

	// Verify file exists
	lockPath := filepath.Join(tmpDir, ".vagrant", "agent-deck.lock")
	if _, err := os.Stat(lockPath); os.IsNotExist(err) {
		t.Fatal("lockfile was not created")
	}

	// Verify content
	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("failed to read lockfile: %v", err)
	}

	var lf lockfileData
	if err := json.Unmarshal(data, &lf); err != nil {
		t.Fatalf("failed to unmarshal lockfile: %v", err)
	}

	if len(lf.Sessions) != 2 {
		t.Errorf("expected 2 sessions in lockfile, got %d", len(lf.Sessions))
	}

	// Load into new manager
	m2 := &Manager{
		projectPath: tmpDir,
		sessions:    []string{},
	}
	m2.loadLockfile()

	if len(m2.sessions) != 2 {
		t.Errorf("expected 2 sessions after load, got %d", len(m2.sessions))
	}

	if m2.sessions[0] != "session-1" || m2.sessions[1] != "session-2" {
		t.Errorf("sessions not loaded correctly: %v", m2.sessions)
	}
}

// TestConcurrentSessionAccess verifies thread safety with goroutines
func TestConcurrentSessionAccess(t *testing.T) {
	tmpDir := t.TempDir()
	m := &Manager{
		projectPath: tmpDir,
		sessions:    []string{},
	}

	var wg sync.WaitGroup
	numGoroutines := 10

	// Concurrently register sessions
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			if err := m.RegisterSession("session-" + string(rune('0'+id))); err != nil {
				t.Errorf("RegisterSession failed: %v", err)
			}
		}(i)
	}

	wg.Wait()

	// Verify all sessions were registered
	count := m.SessionCount()
	if count != numGoroutines {
		t.Errorf("expected %d sessions after concurrent registration, got %d", numGoroutines, count)
	}

	// Concurrently unregister half
	for i := 0; i < numGoroutines/2; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			if err := m.UnregisterSession("session-" + string(rune('0'+id))); err != nil {
				t.Errorf("UnregisterSession failed: %v", err)
			}
		}(i)
	}

	wg.Wait()

	// Verify correct number remain
	finalCount := m.SessionCount()
	if finalCount != numGoroutines/2 {
		t.Errorf("expected %d sessions after concurrent unregister, got %d", numGoroutines/2, finalCount)
	}
}
