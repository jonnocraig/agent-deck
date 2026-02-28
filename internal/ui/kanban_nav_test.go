package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNav_MoveLeft_Basic verifies h moves from col 3 to col 2
func TestNav_MoveLeft_Basic(t *testing.T) {
	nav := KanbanNav{
		Col:              3,
		Row:              0,
		Focus:            PanelBoard,
		DetailOpen:       false,
		ColumnCardCounts: [6]int{2, 3, 1, 4, 2, 1},
	}

	result := MoveLeft(nav)

	assert.Equal(t, 2, result.Col, "should move left from col 3 to col 2")
	assert.Equal(t, 0, result.Row, "row should remain 0")
}

// TestNav_MoveLeft_AtZero verifies h at col 0 stays at 0
func TestNav_MoveLeft_AtZero(t *testing.T) {
	nav := KanbanNav{
		Col:              0,
		Row:              0,
		Focus:            PanelBoard,
		DetailOpen:       false,
		ColumnCardCounts: [6]int{2, 3, 1, 4, 2, 1},
	}

	result := MoveLeft(nav)

	assert.Equal(t, 0, result.Col, "should stay at col 0")
}

// TestNav_MoveLeft_SkipEmpty verifies h skips columns with 0 cards
func TestNav_MoveLeft_SkipEmpty(t *testing.T) {
	nav := KanbanNav{
		Col:              4,
		Row:              0,
		Focus:            PanelBoard,
		DetailOpen:       false,
		ColumnCardCounts: [6]int{2, 0, 0, 4, 2, 1}, // cols 1,2 are empty
	}

	result := MoveLeft(nav)

	assert.Equal(t, 3, result.Col, "should skip empty columns and land on col 3")
}

// TestNav_MoveLeft_RowClamp verifies h to shorter column clamps row
func TestNav_MoveLeft_RowClamp(t *testing.T) {
	nav := KanbanNav{
		Col:              3,
		Row:              3,
		Focus:            PanelBoard,
		DetailOpen:       false,
		ColumnCardCounts: [6]int{2, 3, 1, 4, 2, 1}, // col 2 has 1 card
	}

	result := MoveLeft(nav)

	assert.Equal(t, 2, result.Col, "should move to col 2")
	assert.Equal(t, 0, result.Row, "row should clamp to 0 (max index for 1 card)")
}

// TestNav_MoveLeft_NotBoard verifies h with Focus=PanelSidebar is no-op
func TestNav_MoveLeft_NotBoard(t *testing.T) {
	nav := KanbanNav{
		Col:              3,
		Row:              2,
		Focus:            PanelSidebar,
		DetailOpen:       false,
		ColumnCardCounts: [6]int{2, 3, 1, 4, 2, 1},
	}

	result := MoveLeft(nav)

	assert.Equal(t, nav.Col, result.Col, "should not move when focus is not on board")
	assert.Equal(t, nav.Row, result.Row, "row should not change")
}

// TestNav_MoveRight_Basic verifies l moves from col 2 to col 3
func TestNav_MoveRight_Basic(t *testing.T) {
	nav := KanbanNav{
		Col:              2,
		Row:              0,
		Focus:            PanelBoard,
		DetailOpen:       false,
		ColumnCardCounts: [6]int{2, 3, 1, 4, 2, 1},
	}

	result := MoveRight(nav)

	assert.Equal(t, 3, result.Col, "should move right from col 2 to col 3")
	assert.Equal(t, 0, result.Row, "row should remain 0")
}

// TestNav_MoveRight_AtFive verifies l at col 5 stays at 5
func TestNav_MoveRight_AtFive(t *testing.T) {
	nav := KanbanNav{
		Col:              5,
		Row:              0,
		Focus:            PanelBoard,
		DetailOpen:       false,
		ColumnCardCounts: [6]int{2, 3, 1, 4, 2, 1},
	}

	result := MoveRight(nav)

	assert.Equal(t, 5, result.Col, "should stay at col 5")
}

// TestNav_MoveRight_SkipEmpty verifies l skips columns with 0 cards
func TestNav_MoveRight_SkipEmpty(t *testing.T) {
	nav := KanbanNav{
		Col:              0,
		Row:              0,
		Focus:            PanelBoard,
		DetailOpen:       false,
		ColumnCardCounts: [6]int{2, 0, 0, 4, 2, 1}, // cols 1,2 are empty
	}

	result := MoveRight(nav)

	assert.Equal(t, 3, result.Col, "should skip empty columns and land on col 3")
}

// TestNav_MoveUp_Basic verifies k moves from row 2 to row 1
func TestNav_MoveUp_Basic(t *testing.T) {
	nav := KanbanNav{
		Col:              3,
		Row:              2,
		Focus:            PanelBoard,
		DetailOpen:       false,
		ColumnCardCounts: [6]int{2, 3, 1, 4, 2, 1},
	}

	result := MoveUp(nav)

	assert.Equal(t, 1, result.Row, "should move up from row 2 to row 1")
	assert.Equal(t, 3, result.Col, "col should remain unchanged")
}

// TestNav_MoveUp_AtZero verifies k at row 0 stays at 0
func TestNav_MoveUp_AtZero(t *testing.T) {
	nav := KanbanNav{
		Col:              3,
		Row:              0,
		Focus:            PanelBoard,
		DetailOpen:       false,
		ColumnCardCounts: [6]int{2, 3, 1, 4, 2, 1},
	}

	result := MoveUp(nav)

	assert.Equal(t, 0, result.Row, "should stay at row 0")
}

// TestNav_MoveDown_Basic verifies j moves from row 0 to row 1
func TestNav_MoveDown_Basic(t *testing.T) {
	nav := KanbanNav{
		Col:              3,
		Row:              0,
		Focus:            PanelBoard,
		DetailOpen:       false,
		ColumnCardCounts: [6]int{2, 3, 1, 4, 2, 1},
	}

	result := MoveDown(nav)

	assert.Equal(t, 1, result.Row, "should move down from row 0 to row 1")
	assert.Equal(t, 3, result.Col, "col should remain unchanged")
}

// TestNav_MoveDown_AtMax verifies j at last card stays at last card
func TestNav_MoveDown_AtMax(t *testing.T) {
	nav := KanbanNav{
		Col:              3,
		Row:              3, // col 3 has 4 cards (max index 3)
		Focus:            PanelBoard,
		DetailOpen:       false,
		ColumnCardCounts: [6]int{2, 3, 1, 4, 2, 1},
	}

	result := MoveDown(nav)

	assert.Equal(t, 3, result.Row, "should stay at row 3 (last card)")
}

// TestNav_Tab_SidebarToBoard verifies Tab from Sidebar goes to Board
func TestNav_Tab_SidebarToBoard(t *testing.T) {
	nav := KanbanNav{
		Col:              2,
		Row:              1,
		Focus:            PanelSidebar,
		DetailOpen:       false,
		ColumnCardCounts: [6]int{2, 3, 1, 4, 2, 1},
	}

	result := CycleTabFocus(nav)

	assert.Equal(t, PanelBoard, result.Focus, "should move focus to Board")
}

// TestNav_Tab_BoardToSidebar_NoDetail verifies Tab from Board goes to Sidebar when detail closed
func TestNav_Tab_BoardToSidebar_NoDetail(t *testing.T) {
	nav := KanbanNav{
		Col:              2,
		Row:              1,
		Focus:            PanelBoard,
		DetailOpen:       false,
		ColumnCardCounts: [6]int{2, 3, 1, 4, 2, 1},
	}

	result := CycleTabFocus(nav)

	assert.Equal(t, PanelSidebar, result.Focus, "should move focus to Sidebar when detail closed")
}

// TestNav_Tab_BoardToDetail verifies Tab from Board goes to Detail when detail open
func TestNav_Tab_BoardToDetail(t *testing.T) {
	nav := KanbanNav{
		Col:              2,
		Row:              1,
		Focus:            PanelBoard,
		DetailOpen:       true,
		ColumnCardCounts: [6]int{2, 3, 1, 4, 2, 1},
	}

	result := CycleTabFocus(nav)

	assert.Equal(t, PanelDetail, result.Focus, "should move focus to Detail when open")
}

// TestNav_Tab_DetailToSidebar verifies Tab from Detail goes to Sidebar
func TestNav_Tab_DetailToSidebar(t *testing.T) {
	nav := KanbanNav{
		Col:              2,
		Row:              1,
		Focus:            PanelDetail,
		DetailOpen:       true,
		ColumnCardCounts: [6]int{2, 3, 1, 4, 2, 1},
	}

	result := CycleTabFocus(nav)

	assert.Equal(t, PanelSidebar, result.Focus, "should move focus to Sidebar")
}

// TestNav_JumpToColumn verifies 1-6 sets Col and resets Row to 0
func TestNav_JumpToColumn(t *testing.T) {
	nav := KanbanNav{
		Col:              0,
		Row:              2,
		Focus:            PanelBoard,
		DetailOpen:       false,
		ColumnCardCounts: [6]int{2, 3, 1, 4, 2, 1},
	}

	result := JumpToColumn(nav, 4) // Jump to 5th column (index 4)

	assert.Equal(t, 4, result.Col, "should jump to col 4")
	assert.Equal(t, 0, result.Row, "row should reset to 0")
	assert.Equal(t, PanelBoard, result.Focus, "focus should be set to Board")
}

// TestNav_JumpToColumn_NotBoard verifies jump when Focus != PanelBoard is no-op
func TestNav_JumpToColumn_NotBoard(t *testing.T) {
	nav := KanbanNav{
		Col:              0,
		Row:              2,
		Focus:            PanelSidebar,
		DetailOpen:       false,
		ColumnCardCounts: [6]int{2, 3, 1, 4, 2, 1},
	}

	result := JumpToColumn(nav, 4)

	assert.Equal(t, 0, result.Col, "col should not change when focus is not on board")
	assert.Equal(t, 2, result.Row, "row should not change")
}

// TestNav_ToggleDetail_Open verifies Space opens detail
func TestNav_ToggleDetail_Open(t *testing.T) {
	nav := KanbanNav{
		Col:              2,
		Row:              1,
		Focus:            PanelBoard,
		DetailOpen:       false,
		ColumnCardCounts: [6]int{2, 3, 1, 4, 2, 1},
	}

	result := ToggleDetail(nav)

	assert.True(t, result.DetailOpen, "detail should be opened")
}

// TestNav_ToggleDetail_Close verifies Space closes detail
func TestNav_ToggleDetail_Close(t *testing.T) {
	nav := KanbanNav{
		Col:              2,
		Row:              1,
		Focus:            PanelBoard,
		DetailOpen:       true,
		ColumnCardCounts: [6]int{2, 3, 1, 4, 2, 1},
	}

	result := ToggleDetail(nav)

	assert.False(t, result.DetailOpen, "detail should be closed")
}

// TestNav_ToggleDetail_CloseFocusShift verifies closing detail when Focus=PanelDetail moves focus to Board
func TestNav_ToggleDetail_CloseFocusShift(t *testing.T) {
	nav := KanbanNav{
		Col:              2,
		Row:              1,
		Focus:            PanelDetail,
		DetailOpen:       true,
		ColumnCardCounts: [6]int{2, 3, 1, 4, 2, 1},
	}

	result := ToggleDetail(nav)

	assert.False(t, result.DetailOpen, "detail should be closed")
	assert.Equal(t, PanelBoard, result.Focus, "focus should shift to Board when closing detail")
}

// --- Scroll Tests ---

// TestScroll_CalculateVisibleRange_NoOverflow verifies all cards fit, start=0 end=total
func TestScroll_CalculateVisibleRange_NoOverflow(t *testing.T) {
	start, end := CalculateVisibleCardRange(5, 0, 10, 1)

	assert.Equal(t, 0, start, "start should be 0")
	assert.Equal(t, 5, end, "end should be total cards (5)")
}

// TestScroll_CalculateVisibleRange_Overflow verifies only visible cards, start=offset end=offset+visible
func TestScroll_CalculateVisibleRange_Overflow(t *testing.T) {
	start, end := CalculateVisibleCardRange(20, 5, 10, 1)

	assert.Equal(t, 5, start, "start should be scroll offset (5)")
	assert.Equal(t, 15, end, "end should be start + available height (5 + 10 = 15)")
}

// TestScroll_CalculateVisibleRange_EndClamp verifies end never exceeds total
func TestScroll_CalculateVisibleRange_EndClamp(t *testing.T) {
	start, end := CalculateVisibleCardRange(12, 5, 10, 1)

	assert.Equal(t, 5, start, "start should be scroll offset (5)")
	assert.Equal(t, 12, end, "end should be clamped to total cards (12)")
}

// TestScroll_AutoScroll_SelectionBelow verifies scrolls down when selected below viewport
func TestScroll_AutoScroll_SelectionBelow(t *testing.T) {
	nav := KanbanNav{
		Col:              2,
		Row:              15,
		Focus:            PanelBoard,
		DetailOpen:       false,
		ColumnCardCounts: [6]int{5, 10, 20, 8, 3, 2},
		ScrollOffsets:    [6]int{0, 0, 0, 0, 0, 0},
	}

	result := AutoScrollToSelection(nav, 10, 1)

	assert.GreaterOrEqual(t, result.ScrollOffsets[2], 6, "should scroll down to show row 15")
	assert.Equal(t, nav.Col, result.Col, "col should not change")
	assert.Equal(t, nav.Row, result.Row, "row should not change")
}

// TestScroll_AutoScroll_SelectionAbove verifies scrolls up when selected above viewport
func TestScroll_AutoScroll_SelectionAbove(t *testing.T) {
	nav := KanbanNav{
		Col:              2,
		Row:              2,
		Focus:            PanelBoard,
		DetailOpen:       false,
		ColumnCardCounts: [6]int{5, 10, 20, 8, 3, 2},
		ScrollOffsets:    [6]int{0, 0, 10, 0, 0, 0},
	}

	result := AutoScrollToSelection(nav, 10, 1)

	assert.LessOrEqual(t, result.ScrollOffsets[2], 2, "should scroll up to show row 2")
	assert.Equal(t, nav.Col, result.Col, "col should not change")
	assert.Equal(t, nav.Row, result.Row, "row should not change")
}

// TestScroll_AutoScroll_InView verifies no change when selection is visible
func TestScroll_AutoScroll_InView(t *testing.T) {
	nav := KanbanNav{
		Col:              2,
		Row:              7,
		Focus:            PanelBoard,
		DetailOpen:       false,
		ColumnCardCounts: [6]int{5, 10, 20, 8, 3, 2},
		ScrollOffsets:    [6]int{0, 0, 5, 0, 0, 0},
	}

	result := AutoScrollToSelection(nav, 10, 1)

	assert.Equal(t, 5, result.ScrollOffsets[2], "scroll offset should not change")
}

// --- Move Mode Tests ---

// TestMove_EnterMoveMode verifies m enters move mode when Focus == PanelBoard
func TestMove_EnterMoveMode(t *testing.T) {
	nav := KanbanNav{
		Col:              2,
		Row:              1,
		Focus:            PanelBoard,
		DetailOpen:       false,
		ColumnCardCounts: [6]int{2, 3, 1, 4, 2, 1},
	}

	result := EnterMoveMode(nav)

	assert.True(t, result.MoveMode, "MoveMode should be true")
	assert.Equal(t, 2, result.MoveTargetCol, "MoveTargetCol should be set to current col")
	assert.Equal(t, 2, result.MoveSourceCol, "MoveSourceCol should be set to current col")
	assert.Equal(t, 2, result.Col, "Col should remain unchanged")
	assert.Equal(t, 1, result.Row, "Row should remain unchanged")
}

// TestMove_EnterMoveMode_NotBoard verifies m is no-op when Focus != PanelBoard
func TestMove_EnterMoveMode_NotBoard(t *testing.T) {
	nav := KanbanNav{
		Col:              2,
		Row:              1,
		Focus:            PanelSidebar,
		DetailOpen:       false,
		ColumnCardCounts: [6]int{2, 3, 1, 4, 2, 1},
	}

	result := EnterMoveMode(nav)

	assert.False(t, result.MoveMode, "MoveMode should remain false")
	assert.Equal(t, nav, result, "nav should be unchanged")
}

// TestMove_MoveLeft verifies h in move mode decrements MoveTargetCol
func TestMove_MoveLeft(t *testing.T) {
	nav := KanbanNav{
		Col:              2,
		Row:              1,
		Focus:            PanelBoard,
		DetailOpen:       false,
		ColumnCardCounts: [6]int{2, 3, 1, 4, 2, 1},
		MoveMode:         true,
		MoveTargetCol:    3,
		MoveSourceCol:    2,
	}

	result := MoveModeLeft(nav)

	assert.Equal(t, 2, result.MoveTargetCol, "MoveTargetCol should decrement from 3 to 2")
	assert.Equal(t, 2, result.Col, "Col should remain unchanged")
	assert.Equal(t, 2, result.MoveSourceCol, "MoveSourceCol should remain unchanged")
}

// TestMove_MoveLeft_Clamp verifies h in move mode clamps at 0
func TestMove_MoveLeft_Clamp(t *testing.T) {
	nav := KanbanNav{
		Col:              2,
		Row:              1,
		Focus:            PanelBoard,
		DetailOpen:       false,
		ColumnCardCounts: [6]int{2, 3, 1, 4, 2, 1},
		MoveMode:         true,
		MoveTargetCol:    0,
		MoveSourceCol:    2,
	}

	result := MoveModeLeft(nav)

	assert.Equal(t, 0, result.MoveTargetCol, "MoveTargetCol should clamp at 0")
}

// TestMove_MoveRight verifies l in move mode increments MoveTargetCol
func TestMove_MoveRight(t *testing.T) {
	nav := KanbanNav{
		Col:              2,
		Row:              1,
		Focus:            PanelBoard,
		DetailOpen:       false,
		ColumnCardCounts: [6]int{2, 3, 1, 4, 2, 1},
		MoveMode:         true,
		MoveTargetCol:    2,
		MoveSourceCol:    2,
	}

	result := MoveModeRight(nav)

	assert.Equal(t, 3, result.MoveTargetCol, "MoveTargetCol should increment from 2 to 3")
	assert.Equal(t, 2, result.Col, "Col should remain unchanged")
	assert.Equal(t, 2, result.MoveSourceCol, "MoveSourceCol should remain unchanged")
}

// TestMove_MoveRight_Clamp verifies l in move mode clamps at 5
func TestMove_MoveRight_Clamp(t *testing.T) {
	nav := KanbanNav{
		Col:              2,
		Row:              1,
		Focus:            PanelBoard,
		DetailOpen:       false,
		ColumnCardCounts: [6]int{2, 3, 1, 4, 2, 1},
		MoveMode:         true,
		MoveTargetCol:    5,
		MoveSourceCol:    2,
	}

	result := MoveModeRight(nav)

	assert.Equal(t, 5, result.MoveTargetCol, "MoveTargetCol should clamp at 5")
}

// TestMove_ExitMoveMode verifies Esc exits move mode
func TestMove_ExitMoveMode(t *testing.T) {
	nav := KanbanNav{
		Col:              2,
		Row:              1,
		Focus:            PanelBoard,
		DetailOpen:       false,
		ColumnCardCounts: [6]int{2, 3, 1, 4, 2, 1},
		MoveMode:         true,
		MoveTargetCol:    4,
		MoveSourceCol:    2,
	}

	result := ExitMoveMode(nav)

	assert.False(t, result.MoveMode, "MoveMode should be false")
	assert.Equal(t, 0, result.MoveTargetCol, "MoveTargetCol should be reset")
	assert.Equal(t, 0, result.MoveSourceCol, "MoveSourceCol should be reset")
	assert.Equal(t, 2, result.Col, "Col should remain unchanged")
	assert.Equal(t, 1, result.Row, "Row should remain unchanged")
}

// TestMove_ConfirmMove_Forward verifies Enter confirms move forward (target > source)
func TestMove_ConfirmMove_Forward(t *testing.T) {
	nav := KanbanNav{
		Col:              2,
		Row:              1,
		Focus:            PanelBoard,
		DetailOpen:       false,
		ColumnCardCounts: [6]int{2, 3, 1, 4, 2, 1},
		MoveMode:         true,
		MoveTargetCol:    4,
		MoveSourceCol:    2,
	}

	resultNav, sourceCol, targetCol := ConfirmMove(nav)

	assert.False(t, resultNav.MoveMode, "MoveMode should be false after confirm")
	assert.Equal(t, 2, sourceCol, "sourceCol should be 2")
	assert.Equal(t, 4, targetCol, "targetCol should be 4")
	assert.Equal(t, 0, resultNav.MoveTargetCol, "MoveTargetCol should be reset")
	assert.Equal(t, 0, resultNav.MoveSourceCol, "MoveSourceCol should be reset")
}

// TestMove_ConfirmMove_SameColumn verifies Enter when source == target
func TestMove_ConfirmMove_SameColumn(t *testing.T) {
	nav := KanbanNav{
		Col:              2,
		Row:              1,
		Focus:            PanelBoard,
		DetailOpen:       false,
		ColumnCardCounts: [6]int{2, 3, 1, 4, 2, 1},
		MoveMode:         true,
		MoveTargetCol:    2,
		MoveSourceCol:    2,
	}

	resultNav, sourceCol, targetCol := ConfirmMove(nav)

	assert.False(t, resultNav.MoveMode, "MoveMode should be false after confirm")
	assert.Equal(t, 2, sourceCol, "sourceCol should be 2")
	assert.Equal(t, 2, targetCol, "targetCol should be 2 (same as source)")
}
