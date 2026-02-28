package ui

import (
	"github.com/asheshgoplani/agent-deck/internal/session"
)

// FocusPanel identifies which panel has keyboard focus
type FocusPanel int

const (
	PanelSidebar FocusPanel = iota
	PanelBoard
	PanelDetail
)

// --- User Input Messages ---

// FocusChangedMsg is sent when Tab switches focus between panels
type FocusChangedMsg struct {
	Panel FocusPanel
}

// ColumnChangedMsg is sent when h/l navigates between columns
type ColumnChangedMsg struct {
	Column int
}

// CardChangedMsg is sent when j/k navigates between cards within a column
type CardChangedMsg struct {
	Row int
}

// ColumnJumpMsg is sent when 1-6 keys jump directly to a column
type ColumnJumpMsg struct {
	Column int
}

// DetailToggleMsg is sent when Space toggles the detail panel
type DetailToggleMsg struct{}

// AttachSessionMsg is sent when Enter attaches to the tmux session
type AttachSessionMsg struct {
	SessionID string
}

// EditModeMsg is sent when e enters edit mode or Esc exits it
type EditModeMsg struct {
	Enabled bool
}

// MoveInitiatedMsg is sent when m starts move mode for a card
type MoveInitiatedMsg struct {
	SessionID string
}

// MoveConfirmedMsg is sent when Enter confirms a move target in move mode
type MoveConfirmedMsg struct {
	SessionID  string
	FromColumn session.KanbanColumn
	ToColumn   session.KanbanColumn
}

// YOLOModeToggleMsg is sent when Shift+Y toggles YOLO mode for a session
type YOLOModeToggleMsg struct {
	SessionID string
}

// AutoTriggerToggleMsg is sent when Shift+A toggles auto-trigger for a session
type AutoTriggerToggleMsg struct {
	SessionID string
}

// CreateSessionMsg is sent when n creates a new session in the focused column
type CreateSessionMsg struct {
	Column session.KanbanColumn
}

// DeleteSessionMsg is sent when d deletes a session
type DeleteSessionMsg struct {
	SessionID string
}

// KanbanToggleMsg is sent when K toggles kanban mode for a group
type KanbanToggleMsg struct {
	GroupPath string
}

// --- System/Response Messages ---

// MoveRequest represents a validated move request for the TransitionEngine
type MoveRequest struct {
	SessionID    string
	FromColumn   session.KanbanColumn
	ToColumn     session.KanbanColumn
	GroupPath    string
	ForceConfirm bool
}

// ShowConfirmDialogMsg requests a confirmation dialog before an action
type ShowConfirmDialogMsg struct {
	Title     string
	Body      string
	OnConfirm interface{} // tea.Msg to send on confirmation
}

// ExecuteTransitionMsg is sent after confirmation to execute a column transition
type ExecuteTransitionMsg struct {
	Request MoveRequest
}

// SkillTriggeredMsg indicates a skill has started executing
type SkillTriggeredMsg struct {
	SessionID string
	SkillName string
	Column    session.KanbanColumn
}

// SkillCompletedMsg indicates a skill has finished executing
type SkillCompletedMsg struct {
	SessionID string
	SkillName string
	Success   bool
	Error     error
	Output    string
}

// BoardRefreshMsg triggers a full refresh of the board data from storage
type BoardRefreshMsg struct{}

// RollbackMsg indicates a move failed and the card is being rolled back
type RollbackMsg struct {
	SessionID  string
	FromColumn session.KanbanColumn
	ToColumn   session.KanbanColumn
	Error      error
}

// ErrorAction represents a user-actionable response to an error
type ErrorAction struct {
	Key   rune
	Label string
	Msg   interface{} // tea.Msg to send when this action is chosen
}

// ErrorDisplayMsg shows an error with actionable options
type ErrorDisplayMsg struct {
	SessionID string
	Error     error
	Actions   []ErrorAction
}

// ConductorStatusMsg reports YOLO conductor progress for a session
type ConductorStatusMsg struct {
	SessionID  string
	Column     session.KanbanColumn
	GateStatus map[string]string
}
