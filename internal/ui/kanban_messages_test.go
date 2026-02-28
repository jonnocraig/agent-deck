package ui

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/asheshgoplani/agent-deck/internal/session"
)

// TestKanbanMessagesImplementMsg verifies all kanban message types satisfy tea.Msg interface
func TestKanbanMessagesImplementMsg(t *testing.T) {
	// Each message type must be assignable to tea.Msg (which is interface{})
	var _ tea.Msg = FocusChangedMsg{}
	var _ tea.Msg = ColumnChangedMsg{}
	var _ tea.Msg = CardChangedMsg{}
	var _ tea.Msg = ColumnJumpMsg{}
	var _ tea.Msg = DetailToggleMsg{}
	var _ tea.Msg = AttachSessionMsg{}
	var _ tea.Msg = EditModeMsg{}
	var _ tea.Msg = MoveInitiatedMsg{}
	var _ tea.Msg = MoveConfirmedMsg{}
	var _ tea.Msg = YOLOModeToggleMsg{}
	var _ tea.Msg = AutoTriggerToggleMsg{}
	var _ tea.Msg = CreateSessionMsg{}
	var _ tea.Msg = DeleteSessionMsg{}
	var _ tea.Msg = KanbanToggleMsg{}

	// System messages
	var _ tea.Msg = ShowConfirmDialogMsg{}
	var _ tea.Msg = ExecuteTransitionMsg{}
	var _ tea.Msg = SkillTriggeredMsg{}
	var _ tea.Msg = SkillCompletedMsg{}
	var _ tea.Msg = BoardRefreshMsg{}
	var _ tea.Msg = RollbackMsg{}
	var _ tea.Msg = ErrorDisplayMsg{}
	var _ tea.Msg = ConductorStatusMsg{}
}

func TestFocusPanelValues(t *testing.T) {
	// Verify all FocusPanel values are distinct
	panels := []FocusPanel{PanelSidebar, PanelBoard, PanelDetail}
	seen := make(map[FocusPanel]bool)
	for _, p := range panels {
		if seen[p] {
			t.Errorf("duplicate FocusPanel value: %d", p)
		}
		seen[p] = true
	}
}

func TestMoveConfirmedMsg_Fields(t *testing.T) {
	msg := MoveConfirmedMsg{
		SessionID:  "test-123",
		FromColumn: session.KanbanBacklog,
		ToColumn:   session.KanbanDesign,
	}
	if msg.SessionID != "test-123" {
		t.Errorf("expected SessionID test-123, got %s", msg.SessionID)
	}
	if msg.FromColumn != session.KanbanBacklog {
		t.Errorf("expected FromColumn backlog, got %s", msg.FromColumn)
	}
	if msg.ToColumn != session.KanbanDesign {
		t.Errorf("expected ToColumn design, got %s", msg.ToColumn)
	}
}

func TestErrorDisplayMsg_Actions(t *testing.T) {
	msg := ErrorDisplayMsg{
		SessionID: "test-123",
		Error:     fmt.Errorf("test error"),
		Actions: []ErrorAction{
			{Key: 'r', Label: "retry"},
			{Key: 'v', Label: "view logs"},
		},
	}
	if len(msg.Actions) != 2 {
		t.Errorf("expected 2 actions, got %d", len(msg.Actions))
	}
	if msg.Actions[0].Key != 'r' {
		t.Errorf("expected key 'r', got '%c'", msg.Actions[0].Key)
	}
}
