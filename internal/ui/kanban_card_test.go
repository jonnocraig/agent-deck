package ui

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/asheshgoplani/agent-deck/internal/session"
)

func TestRenderKanbanCard_Compact(t *testing.T) {
	inst := &session.Instance{
		ID:             "test-1",
		Title:          "Fix login bug",
		Status:         session.StatusRunning,
		Tool:           "claude",
		AutomationMode: session.AutomationInteractive,
	}

	card := renderKanbanCard(inst, 20, false)
	require.NotEmpty(t, card)

	// Should contain status icon and title
	assert.Contains(t, card, "Fix login bug")
	// Should be compact (single line, no multi-line)
	assert.Equal(t, 0, strings.Count(card, "\n"))
}

func TestRenderKanbanCard_StatusIcons(t *testing.T) {
	tests := []struct {
		name           string
		status         session.Status
		expectedIcon   string
		expectedInCard bool
	}{
		{"running", session.StatusRunning, "●", true},
		{"waiting", session.StatusWaiting, "◐", true},
		{"idle", session.StatusIdle, "○", true},
		{"error", session.StatusError, "✕", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inst := &session.Instance{
				ID:     "test-1",
				Title:  "Test Session",
				Status: tt.status,
				Tool:   "claude",
			}

			card := renderKanbanCard(inst, 30, false)
			if tt.expectedInCard {
				// Icon should appear somewhere in the card
				// We can't test exact ANSI rendering, but we can test the icon is present
				assert.Contains(t, card, tt.expectedIcon)
			}
		})
	}
}

func TestRenderKanbanCard_Selected(t *testing.T) {
	inst := &session.Instance{
		ID:     "test-1",
		Title:  "Test Session",
		Status: session.StatusRunning,
		Tool:   "claude",
	}

	selected := renderKanbanCard(inst, 30, true)
	unselected := renderKanbanCard(inst, 30, false)

	// Selected and unselected should render differently
	assert.NotEqual(t, selected, unselected)

	// Both should contain the title
	assert.Contains(t, selected, "Test Session")
	assert.Contains(t, unselected, "Test Session")
}

func TestRenderKanbanCard_Unselected(t *testing.T) {
	inst := &session.Instance{
		ID:     "test-1",
		Title:  "Normal Session",
		Status: session.StatusIdle,
		Tool:   "gemini",
	}

	card := renderKanbanCard(inst, 25, false)

	// Should render the title
	assert.Contains(t, card, "Normal Session")
	// Should contain status icon
	assert.Contains(t, card, "○")
}

func TestRenderKanbanCard_TitleTruncation(t *testing.T) {
	inst := &session.Instance{
		ID:     "test-1",
		Title:  "This is a very long session title that should be truncated to fit the column width",
		Status: session.StatusRunning,
		Tool:   "claude",
	}

	// Narrow width should trigger truncation
	card := renderKanbanCard(inst, 20, false)

	// Should contain ellipsis for truncation
	assert.Contains(t, card, "...")
	// Should not contain the full title
	assert.NotContains(t, card, "that should be truncated")
}

func TestRenderKanbanCard_ShortTitle(t *testing.T) {
	inst := &session.Instance{
		ID:     "test-1",
		Title:  "Short",
		Status: session.StatusIdle,
		Tool:   "claude",
	}

	card := renderKanbanCard(inst, 30, false)

	// Should contain the full title without truncation
	assert.Contains(t, card, "Short")
	// Should NOT contain ellipsis
	assert.NotContains(t, card, "...")
}

func TestRenderKanbanCard_YOLOIcon(t *testing.T) {
	inst := &session.Instance{
		ID:             "test-1",
		Title:          "Autonomous Task",
		Status:         session.StatusRunning,
		Tool:           "claude",
		AutomationMode: session.AutomationYOLO,
	}

	card := renderKanbanCard(inst, 30, false)

	// Should contain the robot emoji for YOLO mode
	assert.Contains(t, card, "🤖")
	assert.Contains(t, card, "Autonomous Task")
}

func TestRenderKanbanCard_NilInstance(t *testing.T) {
	card := renderKanbanCard(nil, 20, false)

	// Should return empty string for nil instance
	assert.Equal(t, "", card)
}

func TestKanbanStatusIcon(t *testing.T) {
	tests := []struct {
		name         string
		status       session.Status
		expectedIcon string
	}{
		{"running", session.StatusRunning, "●"},
		{"waiting", session.StatusWaiting, "◐"},
		{"idle", session.StatusIdle, "○"},
		{"error", session.StatusError, "✕"},
		{"unknown defaults to idle", session.Status("unknown"), "○"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			icon, style := kanbanStatusIcon(tt.status)
			assert.Equal(t, tt.expectedIcon, icon)
			assert.NotNil(t, style)
		})
	}
}

func TestTruncateTitle(t *testing.T) {
	tests := []struct {
		name     string
		title    string
		maxWidth int
		expected string
	}{
		{
			name:     "short title no truncation",
			title:    "Short",
			maxWidth: 20,
			expected: "Short",
		},
		{
			name:     "exact fit",
			title:    "ExactlyTen",
			maxWidth: 10,
			expected: "ExactlyTen",
		},
		{
			name:     "long title truncated",
			title:    "This is a very long title",
			maxWidth: 10,
			expected: "This is...",
		},
		{
			name:     "very short max width",
			title:    "Title",
			maxWidth: 3,
			expected: "Tit",
		},
		{
			name:     "empty title",
			title:    "",
			maxWidth: 10,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateTitle(tt.title, tt.maxWidth)
			assert.Equal(t, tt.expected, result)
			assert.LessOrEqual(t, len(result), tt.maxWidth)
		})
	}
}
