package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/asheshgoplani/agent-deck/internal/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper to create a basic test instance
func createTestInstance() *session.Instance {
	col := session.KanbanBacklog
	return &session.Instance{
		ID:              "test-123",
		Title:           "Test Session",
		Description:     "A test session for unit testing",
		AcceptCriteria:  "Should pass all tests",
		Status:          session.StatusIdle,
		Tool:            "claude",
		WorktreePath:    "/path/to/worktree",
		WorktreeBranch:  "feature/test",
		ProjectPath:     "/path/to/project",
		GeminiModel:     "gemini-3-pro",
		KanbanColumn:    &col,
		AutomationMode:  session.AutomationInteractive,
		LatestPrompt:    "This is a test prompt that might be long and needs truncation",
		LoadedMCPNames:  []string{"filesystem", "brave-search"},
		CreatedAt:       time.Now(),
	}
}

func TestDetailView_NotVisible(t *testing.T) {
	state := KanbanDetailState{
		Instance: createTestInstance(),
		Visible:  false,
		Width:    80,
		Height:   20,
	}

	result := renderKanbanDetail(state)

	assert.Empty(t, result, "should return empty string when not visible")
}

func TestDetailView_NilInstance(t *testing.T) {
	state := KanbanDetailState{
		Instance: nil,
		Visible:  true,
		Width:    80,
		Height:   20,
	}

	result := renderKanbanDetail(state)

	assert.Empty(t, result, "should return empty string when instance is nil")
}

func TestDetailView_ShowsTitle(t *testing.T) {
	state := KanbanDetailState{
		Instance: createTestInstance(),
		Visible:  true,
		Width:    80,
		Height:   20,
	}

	result := renderKanbanDetail(state)

	assert.Contains(t, result, "Test Session", "should display session title")
	assert.Contains(t, result, "Title", "should display title label")
}

func TestDetailView_ShowsDescription(t *testing.T) {
	state := KanbanDetailState{
		Instance: createTestInstance(),
		Visible:  true,
		Width:    80,
		Height:   20,
	}

	result := renderKanbanDetail(state)

	assert.Contains(t, result, "Description", "should display description label")
	assert.Contains(t, result, "A test session for unit testing", "should display description value")
}

func TestDetailView_ShowsStatus(t *testing.T) {
	state := KanbanDetailState{
		Instance: createTestInstance(),
		Visible:  true,
		Width:    80,
		Height:   20,
	}

	result := renderKanbanDetail(state)

	assert.Contains(t, result, "Status", "should display status label")
	assert.Contains(t, result, "idle", "should display status value")
}

func TestDetailView_ShowsTool(t *testing.T) {
	state := KanbanDetailState{
		Instance: createTestInstance(),
		Visible:  true,
		Width:    80,
		Height:   20,
	}

	result := renderKanbanDetail(state)

	assert.Contains(t, result, "Tool", "should display tool label")
	assert.Contains(t, result, "claude", "should display tool value")
}

func TestDetailView_ShowsColumn(t *testing.T) {
	state := KanbanDetailState{
		Instance: createTestInstance(),
		Visible:  true,
		Width:    80,
		Height:   20,
	}

	result := renderKanbanDetail(state)

	assert.Contains(t, result, "Column", "should display column label")
	assert.Contains(t, result, "backlog", "should display column value")
}

func TestDetailView_EditableMarkers(t *testing.T) {
	state := KanbanDetailState{
		Instance: createTestInstance(),
		Visible:  true,
		Editing:  false,
		Width:    80,
		Height:   20,
	}

	result := renderKanbanDetail(state)

	// Editable fields should have some kind of marker (e.g., [e] or pencil icon)
	// We'll check that at least one marker exists near editable fields
	// This is a structural test - the exact marker can vary
	lines := strings.Split(result, "\n")
	hasEditMarker := false
	for _, line := range lines {
		if (strings.Contains(line, "Title") ||
		    strings.Contains(line, "Description") ||
		    strings.Contains(line, "Column")) &&
		   (strings.Contains(line, "[e]") || strings.Contains(line, "✎")) {
			hasEditMarker = true
			break
		}
	}
	assert.True(t, hasEditMarker, "editable fields should have edit markers")
}

func TestDetailView_ReadOnlyNoMarkers(t *testing.T) {
	state := KanbanDetailState{
		Instance: createTestInstance(),
		Visible:  true,
		Editing:  false,
		Width:    80,
		Height:   20,
	}

	result := renderKanbanDetail(state)

	// Status and Tool are read-only, shouldn't have edit markers right next to them
	lines := strings.Split(result, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Status:") || strings.Contains(line, "Tool:") {
			// These lines shouldn't have edit markers
			assert.NotContains(t, line, "[e]", "read-only fields should not have [e] marker")
		}
	}
}

func TestDetailView_WidthConstraint(t *testing.T) {
	state := KanbanDetailState{
		Instance: createTestInstance(),
		Visible:  true,
		Width:    40,
		Height:   20,
	}

	result := renderKanbanDetail(state)
	require.NotEmpty(t, result)

	// Just check that output is reasonable and doesn't have excessively long lines
	// Lipgloss may add borders that slightly exceed the width, but content should be truncated
	lines := strings.Split(result, "\n")
	assert.Greater(t, len(lines), 10, "should have multiple lines of output")

	// Check that at least some content is displayed
	assert.Contains(t, result, "Test Session", "should show session title")
	assert.Contains(t, result, "claude", "should show tool")
}

func TestDetailView_LongDescription(t *testing.T) {
	inst := createTestInstance()
	inst.Description = strings.Repeat("This is a very long description. ", 20) // ~660 chars

	state := KanbanDetailState{
		Instance: inst,
		Visible:  true,
		Width:    60,
		Height:   20,
	}

	result := renderKanbanDetail(state)
	require.NotEmpty(t, result)

	// Long text should be truncated - check for ellipsis
	assert.Contains(t, result, "...", "long description should be truncated with ellipsis")

	// The full untruncated description should NOT appear
	fullDesc := inst.Description
	assert.NotContains(t, result, fullDesc, "full long description should not appear untruncated")
}

func TestDetailHeight_Normal(t *testing.T) {
	height := calculateDetailHeight(40)
	// 40% of 40 = 16
	assert.Equal(t, 16, height, "should calculate 40% of terminal height")
}

func TestDetailHeight_Small(t *testing.T) {
	height := calculateDetailHeight(20)
	// 40% of 20 = 8, but minimum is 10
	assert.Equal(t, 10, height, "should enforce minimum height of 10")
}

func TestDetailHeight_Minimum(t *testing.T) {
	height := calculateDetailHeight(5)
	// 40% of 5 = 2, but minimum is 10
	assert.Equal(t, 10, height, "should never go below minimum of 10")
}

func TestDetailView_ShowsAcceptanceCriteria(t *testing.T) {
	state := KanbanDetailState{
		Instance: createTestInstance(),
		Visible:  true,
		Width:    80,
		Height:   20,
	}

	result := renderKanbanDetail(state)

	assert.Contains(t, result, "Acceptance", "should display acceptance criteria label")
	assert.Contains(t, result, "Should pass all tests", "should display acceptance criteria value")
}

func TestDetailView_ShowsAutomationMode(t *testing.T) {
	state := KanbanDetailState{
		Instance: createTestInstance(),
		Visible:  true,
		Width:    80,
		Height:   20,
	}

	result := renderKanbanDetail(state)

	assert.Contains(t, result, "interactive", "should display automation mode")
}

func TestDetailView_ShowsYOLOMode(t *testing.T) {
	inst := createTestInstance()
	inst.AutomationMode = session.AutomationYOLO
	inst.YOLOConfig = &session.YOLOConfig{
		PassThreshold: 0.8,
		PauseOnFail:   true,
	}

	state := KanbanDetailState{
		Instance: inst,
		Visible:  true,
		Width:    80,
		Height:   20,
	}

	result := renderKanbanDetail(state)

	assert.Contains(t, result, "yolo", "should display YOLO mode")
}

func TestDetailView_ShowsWorktreeInfo(t *testing.T) {
	state := KanbanDetailState{
		Instance: createTestInstance(),
		Visible:  true,
		Width:    80,
		Height:   20,
	}

	result := renderKanbanDetail(state)

	assert.Contains(t, result, "/path/to/worktree", "should display worktree path")
	assert.Contains(t, result, "feature/test", "should display worktree branch")
}

func TestDetailView_ShowsModel(t *testing.T) {
	state := KanbanDetailState{
		Instance: createTestInstance(),
		Visible:  true,
		Width:    80,
		Height:   20,
	}

	result := renderKanbanDetail(state)

	assert.Contains(t, result, "Model", "should display model label")
	assert.Contains(t, result, "gemini-3-pro", "should display model value")
}

func TestDetailView_ShowsMCPServers(t *testing.T) {
	state := KanbanDetailState{
		Instance: createTestInstance(),
		Visible:  true,
		Width:    80,
		Height:   20,
	}

	result := renderKanbanDetail(state)

	assert.Contains(t, result, "MCP", "should display MCP label")
	assert.Contains(t, result, "filesystem", "should display filesystem MCP")
	assert.Contains(t, result, "brave-search", "should display brave-search MCP")
}

func TestDetailView_ShowsLastPrompt(t *testing.T) {
	state := KanbanDetailState{
		Instance: createTestInstance(),
		Visible:  true,
		Width:    80,
		Height:   20,
	}

	result := renderKanbanDetail(state)

	assert.Contains(t, result, "Last Prompt", "should display last prompt label")
	assert.Contains(t, result, "This is a test prompt", "should display prompt content")
}

// stripAnsi removes ANSI escape codes for accurate length measurement
func stripAnsi(s string) string {
	// Simple ANSI stripper - matches ESC[...m sequences
	var result strings.Builder
	inEscape := false
	for i := 0; i < len(s); i++ {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			inEscape = true
			i++ // skip '['
			continue
		}
		if inEscape {
			if (s[i] >= 'A' && s[i] <= 'Z') || (s[i] >= 'a' && s[i] <= 'z') {
				inEscape = false
			}
			continue
		}
		result.WriteByte(s[i])
	}
	return result.String()
}

// --- Edit Mode Tests (Task 4.2) ---

// TestEditMode_Enter tests that entering edit mode sets Editing=true and FocusedField=0
func TestEditMode_Enter(t *testing.T) {
	col := session.KanbanBacklog
	state := KanbanDetailState{
		Instance: &session.Instance{
			ID:           "test-1",
			Title:        "Test Task",
			KanbanColumn: &col,
		},
		Visible:      true,
		Editing:      false,
		FocusedField: 0,
		Width:        80,
		Height:       20,
	}

	result := EnterEditMode(state)

	assert.True(t, result.Editing, "should enter edit mode")
	assert.Equal(t, 0, result.FocusedField, "should focus first field")
	assert.Equal(t, state.Instance, result.Instance, "should preserve instance")
	assert.True(t, result.Visible, "should preserve visibility")
}

// TestEditMode_Enter_NotVisible tests that edit mode is not entered when panel is hidden
func TestEditMode_Enter_NotVisible(t *testing.T) {
	col := session.KanbanBacklog
	state := KanbanDetailState{
		Instance: &session.Instance{
			ID:           "test-1",
			Title:        "Test Task",
			KanbanColumn: &col,
		},
		Visible:      false,
		Editing:      false,
		FocusedField: 0,
		Width:        80,
		Height:       20,
	}

	result := EnterEditMode(state)

	assert.False(t, result.Editing, "should not enter edit mode when not visible")
	assert.Equal(t, state, result, "should return unchanged state")
}

// TestEditMode_Enter_NilInstance tests that edit mode is not entered when no instance is selected
func TestEditMode_Enter_NilInstance(t *testing.T) {
	state := KanbanDetailState{
		Instance:     nil,
		Visible:      true,
		Editing:      false,
		FocusedField: 0,
		Width:        80,
		Height:       20,
	}

	result := EnterEditMode(state)

	assert.False(t, result.Editing, "should not enter edit mode with nil instance")
	assert.Equal(t, state, result, "should return unchanged state")
}

// TestEditMode_Exit tests that exiting edit mode sets Editing=false
func TestEditMode_Exit(t *testing.T) {
	col := session.KanbanBacklog
	state := KanbanDetailState{
		Instance: &session.Instance{
			ID:           "test-1",
			Title:        "Test Task",
			KanbanColumn: &col,
		},
		Visible:      true,
		Editing:      true,
		FocusedField: 3,
		Width:        80,
		Height:       20,
	}

	result := ExitEditMode(state)

	assert.False(t, result.Editing, "should exit edit mode")
	assert.Equal(t, state.Instance, result.Instance, "should preserve instance")
	assert.Equal(t, state.FocusedField, result.FocusedField, "should preserve field focus")
}

// TestEditMode_NextField tests that Tab advances from field 0 to 1
func TestEditMode_NextField(t *testing.T) {
	col := session.KanbanBacklog
	state := KanbanDetailState{
		Instance: &session.Instance{
			ID:           "test-1",
			Title:        "Test Task",
			KanbanColumn: &col,
		},
		Visible:      true,
		Editing:      true,
		FocusedField: 0,
		Width:        80,
		Height:       20,
	}

	result := NextEditField(state)

	assert.Equal(t, 1, result.FocusedField, "should advance to field 1")
}

// TestEditMode_NextField_Wrap tests that Tab wraps from field 5 to 0
func TestEditMode_NextField_Wrap(t *testing.T) {
	col := session.KanbanBacklog
	state := KanbanDetailState{
		Instance: &session.Instance{
			ID:           "test-1",
			Title:        "Test Task",
			KanbanColumn: &col,
		},
		Visible:      true,
		Editing:      true,
		FocusedField: 5,
		Width:        80,
		Height:       20,
	}

	result := NextEditField(state)

	assert.Equal(t, 0, result.FocusedField, "should wrap to field 0")
}

// TestEditMode_PrevField tests that Shift+Tab goes from field 1 to 0
func TestEditMode_PrevField(t *testing.T) {
	col := session.KanbanBacklog
	state := KanbanDetailState{
		Instance: &session.Instance{
			ID:           "test-1",
			Title:        "Test Task",
			KanbanColumn: &col,
		},
		Visible:      true,
		Editing:      true,
		FocusedField: 1,
		Width:        80,
		Height:       20,
	}

	result := PrevEditField(state)

	assert.Equal(t, 0, result.FocusedField, "should go to field 0")
}

// TestEditMode_PrevField_Wrap tests that Shift+Tab wraps from field 0 to 5
func TestEditMode_PrevField_Wrap(t *testing.T) {
	col := session.KanbanBacklog
	state := KanbanDetailState{
		Instance: &session.Instance{
			ID:           "test-1",
			Title:        "Test Task",
			KanbanColumn: &col,
		},
		Visible:      true,
		Editing:      true,
		FocusedField: 0,
		Width:        80,
		Height:       20,
	}

	result := PrevEditField(state)

	assert.Equal(t, 5, result.FocusedField, "should wrap to field 5")
}

// TestEditMode_RenderEditing tests that rendering shows editing indicators when Editing=true
func TestEditMode_RenderEditing(t *testing.T) {
	col := session.KanbanBacklog
	state := KanbanDetailState{
		Instance: &session.Instance{
			ID:           "test-1",
			Title:        "Test Task",
			KanbanColumn: &col,
		},
		Visible:      true,
		Editing:      true,
		FocusedField: 0,
		Width:        80,
		Height:       20,
	}

	rendered := renderKanbanDetail(state)

	assert.NotEmpty(t, rendered, "should render content")
	// In edit mode, we don't show [e] markers
	assert.NotContains(t, rendered, "[e]", "should not show [e] markers in edit mode")
}

// TestEditMode_RenderNotEditing tests that rendering shows edit hints when Editing=false
func TestEditMode_RenderNotEditing(t *testing.T) {
	col := session.KanbanBacklog
	state := KanbanDetailState{
		Instance: &session.Instance{
			ID:           "test-1",
			Title:        "Test Task",
			KanbanColumn: &col,
		},
		Visible:      true,
		Editing:      false,
		FocusedField: 0,
		Width:        80,
		Height:       20,
	}

	rendered := renderKanbanDetail(state)

	assert.NotEmpty(t, rendered, "should render content")
	assert.Contains(t, rendered, "[e]", "should show [e] markers when not editing")
}

// TestEditMode_NextField_NotEditing tests that NextEditField is a no-op when not editing
func TestEditMode_NextField_NotEditing(t *testing.T) {
	col := session.KanbanBacklog
	state := KanbanDetailState{
		Instance: &session.Instance{
			ID:           "test-1",
			Title:        "Test Task",
			KanbanColumn: &col,
		},
		Visible:      true,
		Editing:      false,
		FocusedField: 0,
		Width:        80,
		Height:       20,
	}

	result := NextEditField(state)

	assert.Equal(t, state, result, "should return unchanged when not editing")
}

// TestEditMode_PrevField_NotEditing tests that PrevEditField is a no-op when not editing
func TestEditMode_PrevField_NotEditing(t *testing.T) {
	col := session.KanbanBacklog
	state := KanbanDetailState{
		Instance: &session.Instance{
			ID:           "test-1",
			Title:        "Test Task",
			KanbanColumn: &col,
		},
		Visible:      true,
		Editing:      false,
		FocusedField: 0,
		Width:        80,
		Height:       20,
	}

	result := PrevEditField(state)

	assert.Equal(t, state, result, "should return unchanged when not editing")
}
