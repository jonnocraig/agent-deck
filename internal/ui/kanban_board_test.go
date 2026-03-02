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

	var scrollOffsets [6]int
	result := renderKanbanBoard(instances, width, height, selectedCol, selectedRow, sidebarWidth, false, 0, scrollOffsets)

	// Verify result has exactly height lines
	lines := strings.Split(result, "\n")
	assert.Equal(t, height, len(lines), "should have exactly %d lines", height)

	// Verify all 6 column headers are present with full names and correct counts
	// At width=120 sidebarWidth=30 → boardWidth=90, colWidth=15 → full names (>=14)
	resultStr := lipgloss.NewStyle().Render(result) // Strip ANSI for checking
	assert.Contains(t, resultStr, "Backlog(2)", "should show Backlog with count 2")
	assert.Contains(t, resultStr, "Design(1)", "should show Design with count 1")
	assert.Contains(t, resultStr, "Plan(0)", "should show Plan with count 0")
	assert.Contains(t, resultStr, "Implement(3)", "should show Implement with count 3")
	assert.Contains(t, resultStr, "Review(1)", "should show Review with count 1")
	assert.Contains(t, resultStr, "Done(0)", "should show Done with count 0")
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

	var scrollOffsets [6]int
	result := renderKanbanBoard(instances, width, height, selectedCol, selectedRow, sidebarWidth, false, 0, scrollOffsets)

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

	var scrollOffsets [6]int
	result := renderKanbanBoard(instances, width, height, selectedCol, selectedRow, sidebarWidth, false, 0, scrollOffsets)

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

// TestRenderKanbanBoard_ColumnHeaders tests full names with card counts (wide mode)
func TestRenderKanbanBoard_ColumnHeaders(t *testing.T) {
	instances := []*session.Instance{
		makeKanbanInstance("1", "Task 1", session.KanbanBacklog, 0),
		makeKanbanInstance("2", "Task 2", session.KanbanBacklog, 1),
		makeKanbanInstance("3", "Task 3", session.KanbanDesign, 0),
	}

	width := 120
	height := 10

	// Test with column 0 (Backlog) selected — wide mode shows full names
	var scrollOffsets [6]int
	result := renderKanbanBoard(instances, width, height, 0, 0, 30, false, 0, scrollOffsets)
	assert.Contains(t, result, "Backlog(2)", "should show Backlog count")
	assert.Contains(t, result, "Design(1)", "should show Design count")
	assert.Contains(t, result, "Plan(0)", "should show Plan count")
}

// TestRenderKanbanBoard_EmptyBoard tests hint text when no cards in any column
func TestRenderKanbanBoard_EmptyBoard(t *testing.T) {
	instances := []*session.Instance{}

	width := 120
	height := 30
	selectedCol := 0
	selectedRow := 0
	sidebarWidth := 30

	var scrollOffsets [6]int
	result := renderKanbanBoard(instances, width, height, selectedCol, selectedRow, sidebarWidth, false, 0, scrollOffsets)

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

	width := 180 // Increased width to accommodate bordered cards
	height := 30
	selectedCol := 0
	selectedRow := 0
	sidebarWidth := 30

	var scrollOffsets [6]int
	result := renderKanbanBoard(instances, width, height, selectedCol, selectedRow, sidebarWidth, false, 0, scrollOffsets)

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
			result := renderKanbanColumnHeader(tt.colName, tt.count, tt.width, tt.selected, false)
			// Strip ANSI codes for content checking
			stripped := lipgloss.NewStyle().Render(result)
			assert.Contains(t, stripped, tt.want, "should contain expected header text")

			// Verify height is exactly 1 line (header only, no newlines)
			lines := strings.Split(result, "\n")
			assert.Equal(t, 1, len(lines), "header should be exactly 1 line")
		})
	}
}

// TestRenderKanbanBoard_NilKanbanColumn tests that instances with nil KanbanColumn appear in Backlog
func TestRenderKanbanBoard_NilKanbanColumn(t *testing.T) {
	now := time.Now()
	instanceWithNilColumn := &session.Instance{
		ID:              "nil-col-1",
		Title:           "No Column Session",
		Tool:            "claude",
		Status:          session.StatusIdle,
		CreatedAt:       now,
		KanbanColumn:    nil, // Explicitly nil
		KanbanSortOrder: 0,
	}

	instances := []*session.Instance{instanceWithNilColumn}

	width := 120
	height := 30
	selectedCol := 0
	selectedRow := 0
	sidebarWidth := 30

	var scrollOffsets [6]int
	result := renderKanbanBoard(instances, width, height, selectedCol, selectedRow, sidebarWidth, false, 0, scrollOffsets)

	// Verify the result contains Backlog(1), proving the nil-column instance landed in Backlog
	resultStr := lipgloss.NewStyle().Render(result)
	assert.Contains(t, resultStr, "Backlog(1)", "instance with nil KanbanColumn should appear in Backlog column")
}

// TestRenderKanbanBoard_NarrowColumnAbbreviations tests abbreviated column names at narrow width
func TestRenderKanbanBoard_NarrowColumnAbbreviations(t *testing.T) {
	instances := []*session.Instance{
		makeKanbanInstance("1", "Task 1", session.KanbanBacklog, 0),
		makeKanbanInstance("2", "Task 2", session.KanbanDesign, 0),
	}

	// Use width=40, sidebarWidth=0 → boardWidth=40
	// In narrow mode (<= 80), shows 3 columns → colWidth = 40/3 = 13
	// colWidth < 14 triggers abbreviated names
	width := 40
	height := 30
	selectedCol := 0
	selectedRow := 0
	sidebarWidth := 0

	var scrollOffsets [6]int
	result := renderKanbanBoard(instances, width, height, selectedCol, selectedRow, sidebarWidth, false, 0, scrollOffsets)

	// Verify abbreviated names appear (BL, DE, etc.) instead of full names
	resultStr := lipgloss.NewStyle().Render(result)
	assert.Contains(t, resultStr, "BL(", "should use abbreviated name BL for Backlog at narrow width")
	assert.Contains(t, resultStr, "DE(", "should use abbreviated name DE for Design at narrow width")
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
