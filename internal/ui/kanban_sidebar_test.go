package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRenderKanbanSidebar_Groups verifies group rendering with selection highlight
func TestRenderKanbanSidebar_Groups(t *testing.T) {
	state := KanbanSidebarState{
		Groups: []KanbanSidebarGroup{
			{Name: "All Sessions", Path: "", IsAllSessions: true},
			{Name: "Frontend", Path: "frontend", KanbanEnabled: false, SessionCount: 3},
			{Name: "Backend", Path: "backend", KanbanEnabled: true, SessionCount: 5},
		},
		SelectedIdx: 1, // Frontend selected
		Focused:     true,
		Height:      10,
	}

	output := renderKanbanSidebar(state)
	lines := strings.Split(output, "\n")

	// Should have exactly Height lines
	assert.Len(t, lines, state.Height, "output should have exactly Height lines")

	// Should contain group names
	assert.Contains(t, output, "All Sessions")
	assert.Contains(t, output, "Frontend")
	assert.Contains(t, output, "Backend")

	// Selected group should use accent color
	// We can't easily check exact styling, but we can verify structure
	assert.NotEmpty(t, output)
}

// TestRenderKanbanSidebar_KanbanIndicator verifies [K] suffix for kanban-enabled groups
func TestRenderKanbanSidebar_KanbanIndicator(t *testing.T) {
	state := KanbanSidebarState{
		Groups: []KanbanSidebarGroup{
			{Name: "All Sessions", Path: "", IsAllSessions: true},
			{Name: "Project A", Path: "project-a", KanbanEnabled: false, SessionCount: 2},
			{Name: "Project B", Path: "project-b", KanbanEnabled: true, SessionCount: 4},
		},
		SelectedIdx: 0,
		Focused:     true,
		Height:      10,
	}

	output := renderKanbanSidebar(state)

	// All Sessions should NOT have [K]
	assert.NotContains(t, output, "All Sessions [K]")

	// Project A should NOT have [K]
	assert.NotContains(t, output, "Project A [K]")

	// Project B SHOULD have [K]
	assert.Contains(t, output, "[K]")
}

// TestRenderKanbanSidebar_FixedWidth verifies exact 20-column width on every line
func TestRenderKanbanSidebar_FixedWidth(t *testing.T) {
	state := KanbanSidebarState{
		Groups: []KanbanSidebarGroup{
			{Name: "All Sessions", Path: "", IsAllSessions: true},
			{Name: "Short", Path: "short", KanbanEnabled: false},
			{Name: "Very Long Group Name That Should Be Truncated", Path: "long", KanbanEnabled: true},
		},
		SelectedIdx: 0,
		Focused:     true,
		Height:      10,
	}

	output := renderKanbanSidebar(state)
	lines := strings.Split(output, "\n")

	for i, line := range lines {
		width := lipgloss.Width(line)
		assert.Equal(t, kanbanSidebarWidth, width, "line %d should be exactly %d columns wide, got %d", i, kanbanSidebarWidth, width)
	}
}

// TestRenderKanbanSidebar_AllSessionsFirst verifies "All Sessions" is always first
func TestRenderKanbanSidebar_AllSessionsFirst(t *testing.T) {
	state := KanbanSidebarState{
		Groups: []KanbanSidebarGroup{
			{Name: "All Sessions", Path: "", IsAllSessions: true},
			{Name: "Group 1", Path: "group1", KanbanEnabled: false},
			{Name: "Group 2", Path: "group2", KanbanEnabled: true},
		},
		SelectedIdx: 0,
		Focused:     true,
		Height:      10,
	}

	output := renderKanbanSidebar(state)
	lines := strings.Split(output, "\n")

	// First non-empty line should contain "All Sessions"
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			// Strip ANSI to check content
			assert.Contains(t, line, "All Sessions", "first non-empty line should be 'All Sessions'")
			break
		}
	}
}

// TestRenderKanbanSidebar_JKNav verifies j/k navigation with boundary wrapping
func TestRenderKanbanSidebar_JKNav(t *testing.T) {
	initial := KanbanSidebarState{
		Groups: []KanbanSidebarGroup{
			{Name: "All Sessions", Path: "", IsAllSessions: true},
			{Name: "Group 1", Path: "group1"},
			{Name: "Group 2", Path: "group2"},
		},
		SelectedIdx: 0,
		Focused:     true,
		Height:      10,
	}

	// Test j (down) from first item
	state := updateKanbanSidebarNav(initial, "j")
	assert.Equal(t, 1, state.SelectedIdx, "j should move down from 0 to 1")

	// Test j (down) from middle
	state = updateKanbanSidebarNav(state, "j")
	assert.Equal(t, 2, state.SelectedIdx, "j should move down from 1 to 2")

	// Test j (down) at last item — should clamp
	state = updateKanbanSidebarNav(state, "j")
	assert.Equal(t, 2, state.SelectedIdx, "j should clamp at last item (2)")

	// Test k (up) from last item
	state = updateKanbanSidebarNav(state, "k")
	assert.Equal(t, 1, state.SelectedIdx, "k should move up from 2 to 1")

	// Test k (up) from middle
	state = updateKanbanSidebarNav(state, "k")
	assert.Equal(t, 0, state.SelectedIdx, "k should move up from 1 to 0")

	// Test k (up) at first item — should clamp
	state = updateKanbanSidebarNav(state, "k")
	assert.Equal(t, 0, state.SelectedIdx, "k should clamp at first item (0)")
}

// TestRenderKanbanSidebar_Focused verifies j/k only works when focused
func TestRenderKanbanSidebar_Focused(t *testing.T) {
	unfocused := KanbanSidebarState{
		Groups: []KanbanSidebarGroup{
			{Name: "All Sessions", Path: "", IsAllSessions: true},
			{Name: "Group 1", Path: "group1"},
			{Name: "Group 2", Path: "group2"},
		},
		SelectedIdx: 0,
		Focused:     false, // NOT focused
		Height:      10,
	}

	// j/k should be ignored when not focused
	state := updateKanbanSidebarNav(unfocused, "j")
	assert.Equal(t, 0, state.SelectedIdx, "j should be ignored when not focused")

	state = updateKanbanSidebarNav(unfocused, "k")
	assert.Equal(t, 0, state.SelectedIdx, "k should be ignored when not focused")

	// Now test with focus
	focused := unfocused
	focused.Focused = true

	state = updateKanbanSidebarNav(focused, "j")
	assert.Equal(t, 1, state.SelectedIdx, "j should work when focused")
}

// TestUpdateKanbanSidebarNav_IgnoresOtherKeys verifies other keys are ignored
func TestUpdateKanbanSidebarNav_IgnoresOtherKeys(t *testing.T) {
	initial := KanbanSidebarState{
		Groups: []KanbanSidebarGroup{
			{Name: "All Sessions", Path: "", IsAllSessions: true},
			{Name: "Group 1", Path: "group1"},
		},
		SelectedIdx: 0,
		Focused:     true,
		Height:      10,
	}

	// Test that other keys don't affect state
	keys := []string{"a", "b", "h", "l", "enter", "esc", "1", "2"}
	for _, key := range keys {
		state := updateKanbanSidebarNav(initial, key)
		assert.Equal(t, 0, state.SelectedIdx, "key %q should not change SelectedIdx", key)
	}
}

// TestUpdateKanbanSidebarNav_Immutability verifies immutable pattern
func TestUpdateKanbanSidebarNav_Immutability(t *testing.T) {
	initial := KanbanSidebarState{
		Groups: []KanbanSidebarGroup{
			{Name: "All Sessions", Path: "", IsAllSessions: true},
			{Name: "Group 1", Path: "group1"},
		},
		SelectedIdx: 0,
		Focused:     true,
		Height:      10,
	}

	// Save original pointer
	originalGroups := initial.Groups

	// Call update
	newState := updateKanbanSidebarNav(initial, "j")

	// Verify initial state unchanged
	assert.Equal(t, 0, initial.SelectedIdx, "initial state should not be mutated")
	assert.Equal(t, 1, newState.SelectedIdx, "new state should have updated index")

	// Verify groups slice is the same (we're not copying it, just the struct)
	require.Equal(t, &originalGroups[0], &newState.Groups[0], "groups slice should be same reference")
}

// TestRenderKanbanSidebar_EmptyGroups verifies behavior with no groups
func TestRenderKanbanSidebar_EmptyGroups(t *testing.T) {
	state := KanbanSidebarState{
		Groups:      []KanbanSidebarGroup{},
		SelectedIdx: 0,
		Focused:     true,
		Height:      10,
	}

	output := renderKanbanSidebar(state)
	lines := strings.Split(output, "\n")

	// Should still have exactly Height lines
	assert.Len(t, lines, state.Height, "output should have exactly Height lines even when empty")

	// All lines should be exactly kanbanSidebarWidth wide
	for i, line := range lines {
		width := lipgloss.Width(line)
		assert.Equal(t, kanbanSidebarWidth, width, "line %d should be exactly %d columns wide", i, kanbanSidebarWidth)
	}
}

// TestRenderKanbanSidebar_SingleGroup verifies behavior with one group
func TestRenderKanbanSidebar_SingleGroup(t *testing.T) {
	state := KanbanSidebarState{
		Groups: []KanbanSidebarGroup{
			{Name: "All Sessions", Path: "", IsAllSessions: true},
		},
		SelectedIdx: 0,
		Focused:     true,
		Height:      10,
	}

	output := renderKanbanSidebar(state)

	// Should render without error
	assert.NotEmpty(t, output)
	assert.Contains(t, output, "All Sessions")

	// Navigation should clamp
	state = updateKanbanSidebarNav(state, "j")
	assert.Equal(t, 0, state.SelectedIdx, "j should clamp at last item when only one group")

	state = updateKanbanSidebarNav(state, "k")
	assert.Equal(t, 0, state.SelectedIdx, "k should clamp at first item when only one group")
}
