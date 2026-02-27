package ui

// kanban_conductor.go — Kanban conductor orchestration (Phase 0 stub)
//
// The Conductor manages autonomous session orchestration in kanban mode.
// It tracks continuation context and state for each managed session.
// The actual implementation will be added in Phase 1.

// ConductorState represents the current state of a conductor instance.
type ConductorState int

const (
	ConductorIdle    ConductorState = iota // Not actively orchestrating
	ConductorRunning                       // Actively managing a session
	ConductorPaused                        // Paused by user or system
)

// Conductor holds orchestration context for a single managed session.
type Conductor struct {
	SessionID      string
	ContinuationID string
}
