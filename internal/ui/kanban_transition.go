package ui

// kanban_transition.go — Kanban column transition engine (Phase 0 stub)
//
// This file defines the interface and types for managing session transitions
// between kanban columns. The actual implementation will be added in Phase 1.

// TODO: Phase 1 will replace string column parameters with session.KanbanColumn
// once that type is defined in the session package.

// MoveRequest describes a request to move a session between kanban columns.
type MoveRequest struct {
	SessionID  string
	FromColumn string // TODO: Phase 1 → session.KanbanColumn
	ToColumn   string // TODO: Phase 1 → session.KanbanColumn
}

// MoveResult describes the outcome of a column transition.
type MoveResult struct {
	Success bool
	Error   error
}

// SkillMapping maps a skill name to its target column and any metadata.
type SkillMapping struct {
	SkillName    string
	TargetColumn string // TODO: Phase 1 → session.KanbanColumn
}

// TransitionEngine validates and executes session moves between kanban columns.
type TransitionEngine interface {
	// RequestMove attempts to move a session from one column to another.
	RequestMove(req MoveRequest) MoveResult

	// ResolveSkill maps a skill name and current column to a SkillMapping.
	// TODO: Phase 1 → second parameter becomes session.KanbanColumn
	ResolveSkill(skill string, currentColumn string) (SkillMapping, error)

	// IsValidMove checks whether a transition between two columns is allowed.
	// TODO: Phase 1 → parameters become session.KanbanColumn
	IsValidMove(from, to string) bool
}
