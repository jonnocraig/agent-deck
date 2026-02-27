package ui

import (
	"github.com/asheshgoplani/agent-deck/internal/session"
)

// kanban_transition.go — Kanban column transition engine (Phase 0 stub)
//
// This file defines the interface and types for managing session transitions
// between kanban columns. The actual implementation will be added in Phase 1.

// MoveResult describes the outcome of a column transition.
type MoveResult struct {
	Success bool
	Error   error
}

// SkillMapping maps a skill name to its target column and any metadata.
type SkillMapping struct {
	SkillName    string
	TargetColumn session.KanbanColumn
}

// TransitionEngine validates and executes session moves between kanban columns.
type TransitionEngine interface {
	// RequestMove attempts to move a session from one column to another.
	RequestMove(req MoveRequest) MoveResult

	// ResolveSkill maps a skill name and current column to a SkillMapping.
	ResolveSkill(skill string, currentColumn session.KanbanColumn) (SkillMapping, error)

	// IsValidMove checks whether a transition between two columns is allowed.
	IsValidMove(from, to session.KanbanColumn) bool
}
