package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/asheshgoplani/agent-deck/internal/session"
	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRenderKanbanBoard_120Cols tests 6 columns at ~16 chars each, all column headers visible
func TestRenderKanbanBoard_120Cols(t *testing.T) {
	instances := []*session.Instance{
		makeKanbanInstance("1", "Task 1", session.KanbanBacklog, 0),
		makeKanbanInstance("2", "Task 2", session.KanbanBacklog, 1),
		makeKanbanInstance("3", "Task 3", session.KanbanDesign, 0),
		makeKanbanInstance("4", "Task 4", session.KanbanImplement, 0),
		makeKanbanInstance("5", "Task 5", session.KanbanImplement, 1),
		makeKanbanInstance("6", "Task 6", session.KanbanImplement, 2),
		makeKanbanInstance("7", "Task 7", session.KanbanReview, 0),
	}

	width := 120
	height := 30
	selectedCol := 0
	selectedRow := 0
	sidebarWidth := 30

	result := renderKanbanBoard(instances, width, height, selectedCol, selectedRow, sidebarWidth)

	// Verify result has exactly height lines
	lines := strings.Split(result, "\n")
	assert.Equal(t, height, len(lines), "should have exactly %d lines", height)

	// Verify all 6 column headers are present with correct counts
	resultStr := lipgloss.NewStyle().Render(result) // Strip ANSI for checking
	assert.Contains(t, resultStr, "BL(2)", "should show Backlog with count 2")
	assert.Contains(t, resultStr, "DE(1)", "should show Design with count 1")
	assert.Contains(t, resultStr, "PL(0)", "should show Plan with count 0")
	assert.Contains(t, resultStr, "IM(3)", "should show Implement with count 3")
	assert.Contains(t, resultStr, "RE(1)", "should show Review with count 1")
	assert.Contains(t, resultStr, "DO(0)", "should show Done with count 0")
}

// TestRenderKanbanBoard_160Cols tests 6 columns at ~23 chars each
func TestRenderKanbanBoard_160Cols(t *testing.T) {
	instances := []*session.Instance{
		makeKanbanInstance("1", "Task 1", session.KanbanBacklog, 0),
	}

	width := 160
	height := 30
	selectedCol := 0
	selectedRow := 0
	sidebarWidth := 40

	result := renderKanbanBoard(instances, width, height, selectedCol, selectedRow, sidebarWidth)

	// Verify result has exactly height lines
	lines := strings.Split(result, "\n")
	assert.Equal(t, height, len(lines), "should have exactly %d lines", height)

	// Verify each line doesn't exceed width
	for i, line := range lines {
		visualWidth := lipgloss.Width(line)
		assert.LessOrEqual(t, visualWidth, width, "line %d should not exceed width", i)
	}
}

// TestRenderKanbanBoard_80Cols tests hiding sidebar and showing only 3 columns with scroll indicators
func TestRenderKanbanBoard_80Cols(t *testing.T) {
	instances := []*session.Instance{
		makeKanbanInstance("1", "Task 1", session.KanbanBacklog, 0),
		makeKanbanInstance("2", "Task 2", session.KanbanDesign, 0),
		makeKanbanInstance("3", "Task 3", session.KanbanPlan, 0),
		makeKanbanInstance("4", "Task 4", session.KanbanImplement, 0),
		makeKanbanInstance("5", "Task 5", session.KanbanReview, 0),
		makeKanbanInstance("6", "Task 6", session.KanbanDone, 0),
	}

	width := 80
	height := 30
	selectedCol := 2 // Plan column (middle)
	selectedRow := 0
	sidebarWidth := 0 // Sidebar hidden in narrow mode

	result := renderKanbanBoard(instances, width, height, selectedCol, selectedRow, sidebarWidth)

	// Verify result has exactly height lines
	lines := strings.Split(result, "\n")
	assert.Equal(t, height, len(lines), "should have exactly %d lines", height)

	// In narrow mode with selectedCol=2, should show columns 1,2,3 (Design, Plan, Implement)
	// Should show ◀ indicator (columns to the left exist)
	// Should show ▶ indicator (columns to the right exist)
	resultStr := result
	assert.Contains(t, resultStr, "◀", "should show left scroll indicator")
	assert.Contains(t, resultStr, "▶", "should show right scroll indicator")
}

// TestRenderKanbanBoard_ColumnHeaders tests abbreviated names with card counts
func TestRenderKanbanBoard_ColumnHeaders(t *testing.T) {
	instances := []*session.Instance{
		makeKanbanInstance("1", "Task 1", session.KanbanBacklog, 0),
		makeKanbanInstance("2", "Task 2", session.KanbanBacklog, 1),
		makeKanbanInstance("3", "Task 3", session.KanbanDesign, 0),
	}

	width := 120
	height := 10

	// Test with column 0 (Backlog) selected
	result := renderKanbanBoard(instances, width, height, 0, 0, 30)
	assert.Contains(t, result, "BL(2)", "should show Backlog count")
	assert.Contains(t, result, "DE(1)", "should show Design count")
	assert.Contains(t, result, "PL(0)", "should show Plan count")
}

// TestRenderKanbanBoard_EmptyBoard tests hint text when no cards in any column
func TestRenderKanbanBoard_EmptyBoard(t *testing.T) {
	instances := []*session.Instance{}

	width := 120
	height := 30
	selectedCol := 0
	selectedRow := 0
	sidebarWidth := 30

	result := renderKanbanBoard(instances, width, height, selectedCol, selectedRow, sidebarWidth)

	// Should show empty state hint
	resultStr := strings.ToLower(result)
	// Look for common empty state words
	hasEmptyHint := strings.Contains(resultStr, "empty") ||
		strings.Contains(resultStr, "no cards") ||
		strings.Contains(resultStr, "no sessions") ||
		strings.Contains(resultStr, "no tasks")
	assert.True(t, hasEmptyHint, "should show empty state hint")
}

// TestRenderKanbanBoard_CardsInColumns tests cards sorted by sort_order within each column
func TestRenderKanbanBoard_CardsInColumns(t *testing.T) {
	instances := []*session.Instance{
		makeKanbanInstance("1", "Task A", session.KanbanBacklog, 2), // Should be 3rd
		makeKanbanInstance("2", "Task B", session.KanbanBacklog, 0), // Should be 1st
		makeKanbanInstance("3", "Task C", session.KanbanBacklog, 1), // Should be 2nd
	}

	width := 120
	height := 30
	selectedCol := 0
	selectedRow := 0
	sidebarWidth := 30

	result := renderKanbanBoard(instances, width, height, selectedCol, selectedRow, sidebarWidth)

	// Find all occurrences of Task names
	lines := strings.Split(result, "\n")
	var taskOrder []string
	for _, line := range lines {
		if strings.Contains(line, "Task A") {
			taskOrder = append(taskOrder, "A")
		} else if strings.Contains(line, "Task B") {
			taskOrder = append(taskOrder, "B")
		} else if strings.Contains(line, "Task C") {
			taskOrder = append(taskOrder, "C")
		}
	}

	// Verify sort order: B (0), C (1), A (2)
	require.Len(t, taskOrder, 3, "should find all 3 tasks")
	assert.Equal(t, "B", taskOrder[0], "Task B should be first (sort_order=0)")
	assert.Equal(t, "C", taskOrder[1], "Task C should be second (sort_order=1)")
	assert.Equal(t, "A", taskOrder[2], "Task A should be third (sort_order=2)")
}

// TestRenderKanbanColumnHeader tests column header rendering with different states
func TestRenderKanbanColumnHeader(t *testing.T) {
	tests := []struct {
		name     string
		colName  string
		count    int
		width    int
		selected bool
		want     string
	}{
		{
			name:     "backlog with 2 cards",
			colName:  "BL",
			count:    2,
			width:    20,
			selected: false,
			want:     "BL(2)",
		},
		{
			name:     "design with 0 cards",
			colName:  "DE",
			count:    0,
			width:    20,
			selected: false,
			want:     "DE(0)",
		},
		{
			name:     "selected column",
			colName:  "PL",
			count:    5,
			width:    20,
			selected: true,
			want:     "PL(5)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := renderKanbanColumnHeader(tt.colName, tt.count, tt.width, tt.selected)
			// Strip ANSI codes for content checking
			stripped := lipgloss.NewStyle().Render(result)
			assert.Contains(t, stripped, tt.want, "should contain expected header text")

			// Verify height is exactly 1 line (header only, no newlines)
			lines := strings.Split(result, "\n")
			assert.Equal(t, 1, len(lines), "header should be exactly 1 line")
		})
	}
}

// Helper function to create kanban instances for testing
func makeKanbanInstance(id, title string, col session.KanbanColumn, sortOrder int) *session.Instance {
	now := time.Now()
	return &session.Instance{
		ID:              id,
		Title:           title,
		Tool:            "claude",
		Status:          session.StatusIdle,
		CreatedAt:       now,
		KanbanColumn:    &col,
		KanbanSortOrder: sortOrder,
	}
}
