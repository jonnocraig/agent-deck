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

// ────────────────────────────────────────────────────────────────────────────
// YOLO Progress Indicator Tests
// ────────────────────────────────────────────────────────────────────────────

func TestRenderYOLOProgress_AllPending(t *testing.T) {
	InitTheme("dark") // ensure color vars are set

	// When current column is Backlog, the gear should be on Backlog
	// and everything after it is pending (dim).
	result := renderYOLOProgress(session.KanbanBacklog, 80)
	require.NotEmpty(t, result)

	// Backlog should show the gear icon (current)
	assert.Contains(t, result, "BL")
	assert.Contains(t, result, gearIcon)

	// No checkmark should appear since nothing is completed
	assert.NotContains(t, result, checkIcon)

	// Arrow separator should be present
	assert.Contains(t, result, arrowSep)

	// All column abbreviations should be present
	for _, abbr := range []string{"BL", "DE", "PL", "IM", "RE", "DO"} {
		assert.Contains(t, result, abbr)
	}
}

func TestRenderYOLOProgress_MidWay(t *testing.T) {
	InitTheme("dark")

	// Current column is Implement (index 3)
	// Backlog, Design, Plan should show checkmarks (completed)
	// Implement should show gear (current)
	// Review, Done should be pending (dim)
	result := renderYOLOProgress(session.KanbanImplement, 80)
	require.NotEmpty(t, result)

	assert.Contains(t, result, gearIcon)
	assert.Contains(t, result, checkIcon)
	assert.Contains(t, result, "IM")
}

func TestRenderYOLOProgress_Complete(t *testing.T) {
	InitTheme("dark")

	// When current column is Done, all columns should show checkmarks
	result := renderYOLOProgress(session.KanbanDone, 80)
	require.NotEmpty(t, result)

	// All should be completed: 6 checkmarks, no gear
	assert.NotContains(t, result, gearIcon)

	// Count checkmarks - should have one per column (6 total)
	checkCount := strings.Count(result, checkIcon)
	assert.Equal(t, 6, checkCount)
}

func TestRenderYOLOProgress_FirstColumn(t *testing.T) {
	InitTheme("dark")

	// First column (Backlog) shows gear, all others pending
	result := renderYOLOProgress(session.KanbanBacklog, 80)
	require.NotEmpty(t, result)

	// Should have exactly one gear icon
	gearCount := strings.Count(result, gearIcon)
	assert.Equal(t, 1, gearCount)

	// Should have zero check icons
	checkCount := strings.Count(result, checkIcon)
	assert.Equal(t, 0, checkCount)
}

func TestRenderYOLOProgress_NarrowWidth(t *testing.T) {
	InitTheme("dark")

	// Should still render without panicking on very narrow widths
	result := renderYOLOProgress(session.KanbanPlan, 20)
	require.NotEmpty(t, result)

	// Should contain at least one column abbreviation
	assert.Contains(t, result, "BL")
}

func TestRenderYOLOProgress_InvalidColumn(t *testing.T) {
	InitTheme("dark")

	// Invalid column should return a sensible result (all pending)
	result := renderYOLOProgress(session.KanbanColumn("invalid"), 80)
	require.NotEmpty(t, result)

	// No checkmarks and no gear for invalid column
	assert.NotContains(t, result, checkIcon)
	assert.NotContains(t, result, gearIcon)
}

// ────────────────────────────────────────────────────────────────────────────
// Gate Status Rendering Tests
// ────────────────────────────────────────────────────────────────────────────

func TestRenderGateStatus_InProgress(t *testing.T) {
	InitTheme("dark")

	gateStatus := map[string]string{
		"gemini-3-pro-preview": "passed",
		"gpt-5.2":             "pending",
		"o3":                  "pending",
	}

	result := renderGateStatus(gateStatus, 80)
	require.NotEmpty(t, result)

	// Should contain "Gate:" label
	assert.Contains(t, result, "Gate:")

	// Model names should appear (with -preview stripped)
	assert.Contains(t, result, "gemini-3-pro")
	assert.Contains(t, result, "gpt-5.2")
	assert.Contains(t, result, "o3")

	// Should contain passed and pending icons
	assert.Contains(t, result, gatePassedIcon)
	assert.Contains(t, result, gatePendingIcon)
}

func TestRenderGateStatus_AllPassed(t *testing.T) {
	InitTheme("dark")

	gateStatus := map[string]string{
		"gemini": "passed",
		"gpt":   "passed",
		"o3":    "passed",
	}

	result := renderGateStatus(gateStatus, 80)
	require.NotEmpty(t, result)

	// All should show passed icon
	passedCount := strings.Count(result, gatePassedIcon)
	assert.Equal(t, 3, passedCount)

	// No pending or failed icons
	assert.NotContains(t, result, gatePendingIcon)
	assert.NotContains(t, result, gateFailedIcon)
}

func TestRenderGateStatus_Failed(t *testing.T) {
	InitTheme("dark")

	gateStatus := map[string]string{
		"gemini": "passed",
		"gpt":   "failed",
		"o3":    "pending",
	}

	result := renderGateStatus(gateStatus, 80)
	require.NotEmpty(t, result)

	// Should contain all three icon types
	assert.Contains(t, result, gatePassedIcon)
	assert.Contains(t, result, gateFailedIcon)
	assert.Contains(t, result, gatePendingIcon)
}

func TestRenderGateStatus_NoStatus(t *testing.T) {
	InitTheme("dark")

	// Empty map should return empty string
	result := renderGateStatus(map[string]string{}, 80)
	assert.Empty(t, result)

	// Nil map should also return empty string
	result = renderGateStatus(nil, 80)
	assert.Empty(t, result)
}

func TestRenderGateStatus_UnknownModels(t *testing.T) {
	InitTheme("dark")

	gateStatus := map[string]string{
		"some-new-model-preview": "passed",
		"custom-model":          "unknown_status",
	}

	result := renderGateStatus(gateStatus, 80)
	require.NotEmpty(t, result)

	// -preview suffix should be stripped
	assert.Contains(t, result, "some-new-model")
	assert.NotContains(t, result, "some-new-model-preview")

	// Unknown status should show the unknown icon
	assert.Contains(t, result, gateUnknownIcon)
}

// ────────────────────────────────────────────────────────────────────────────
// Conductor ProgressColumns Tests
// ────────────────────────────────────────────────────────────────────────────

func TestConductor_ProgressColumns_MidWay(t *testing.T) {
	c := NewConductor(
		WithCurrentColumn(session.KanbanImplement),
	)

	completed, current, pending := c.ProgressColumns()

	// Backlog, Design, Plan should be completed
	assert.Equal(t, []session.KanbanColumn{
		session.KanbanBacklog,
		session.KanbanDesign,
		session.KanbanPlan,
	}, completed)

	// Implement is current
	assert.Equal(t, session.KanbanImplement, current)

	// Review, Done should be pending
	assert.Equal(t, []session.KanbanColumn{
		session.KanbanReview,
		session.KanbanDone,
	}, pending)
}

func TestConductor_ProgressColumns_AtStart(t *testing.T) {
	c := NewConductor(
		WithCurrentColumn(session.KanbanBacklog),
	)

	completed, current, pending := c.ProgressColumns()

	assert.Empty(t, completed)
	assert.Equal(t, session.KanbanBacklog, current)
	assert.Equal(t, []session.KanbanColumn{
		session.KanbanDesign,
		session.KanbanPlan,
		session.KanbanImplement,
		session.KanbanReview,
		session.KanbanDone,
	}, pending)
}

func TestConductor_ProgressColumns_AtDone(t *testing.T) {
	c := NewConductor(
		WithCurrentColumn(session.KanbanDone),
	)

	completed, current, pending := c.ProgressColumns()

	// All columns before Done are completed
	assert.Equal(t, []session.KanbanColumn{
		session.KanbanBacklog,
		session.KanbanDesign,
		session.KanbanPlan,
		session.KanbanImplement,
		session.KanbanReview,
	}, completed)

	assert.Equal(t, session.KanbanDone, current)
	assert.Empty(t, pending)
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
