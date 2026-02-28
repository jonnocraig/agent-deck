package ui

// kanban_nav.go — Help bar and navigation rendering extracted from home.go
//
// Contains: renderHelpBar, renderHelpBarTiny, renderHelpBarMinimal,
// renderHelpBarCompact, helpKeyShort, previewModeShort, renderHelpBarFull, helpKey

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/asheshgoplani/agent-deck/internal/session"
)

// renderHelpBar renders context-aware keyboard shortcuts, adapting to terminal width
func (h *Home) renderHelpBar() string {
	// Route to appropriate tier based on width
	switch {
	case h.width < layoutBreakpointSingle:
		return h.renderHelpBarTiny()
	case h.width < 70:
		return h.renderHelpBarMinimal()
	case h.width < 100:
		return h.renderHelpBarCompact()
	default:
		return h.renderHelpBarFull()
	}
}

// renderHelpBarTiny renders minimal help for very narrow terminals (<50 cols)
func (h *Home) renderHelpBarTiny() string {
	borderStyle := lipgloss.NewStyle().Foreground(ColorBorder)
	border := borderStyle.Render(strings.Repeat("─", max(0, h.width)))

	hintStyle := lipgloss.NewStyle().Foreground(ColorComment)
	hint := hintStyle.Render("? for help")

	// Center the hint
	padding := (h.width - lipgloss.Width(hint)) / 2
	if padding < 0 {
		padding = 0
	}
	content := strings.Repeat(" ", padding) + hint

	raw := lipgloss.JoinVertical(lipgloss.Left, border, content)
	return lipgloss.NewStyle().MaxWidth(h.width).Render(raw)
}

// renderHelpBarMinimal renders keys-only help for narrow terminals (50-69 cols)
func (h *Home) renderHelpBarMinimal() string {
	borderStyle := lipgloss.NewStyle().Foreground(ColorBorder)
	border := borderStyle.Render(strings.Repeat("─", max(0, h.width)))

	keyStyle := lipgloss.NewStyle().
		Foreground(ColorBg).
		Background(ColorAccent).
		Bold(true)
	sepStyle := lipgloss.NewStyle().Foreground(ColorBorder)
	sep := sepStyle.Render(" │ ")

	// Context-specific keys (left side)
	var contextKeys string
	if len(h.flatItems) == 0 {
		contextKeys = keyStyle.Render(
			"n",
		) + " " + keyStyle.Render(
			"N",
		) + " " + keyStyle.Render(
			"i",
		) + " " + keyStyle.Render(
			"g",
		)
	} else if h.cursor < len(h.flatItems) {
		item := h.flatItems[h.cursor]
		if item.Type == session.ItemTypeGroup {
			contextKeys = keyStyle.Render("⏎") + " " + keyStyle.Render("n") + " " + keyStyle.Render("N") + " " + keyStyle.Render("g")
		} else {
			contextKeys = keyStyle.Render("⏎") + " " + keyStyle.Render("n") + " " + keyStyle.Render("N") + " " + keyStyle.Render("R")
			if item.Session != nil && item.Session.CanFork() {
				contextKeys += " " + keyStyle.Render("f")
			}
			if item.Session != nil && (item.Session.Tool == "claude" || item.Session.Tool == "gemini") {
				contextKeys += " " + keyStyle.Render("m")
			}
			if item.Session != nil && item.Session.Tool == "claude" {
				contextKeys += " " + keyStyle.Render("s")
			}
		}
	}

	// Global keys (right side)
	globalStyle := lipgloss.NewStyle().Foreground(ColorComment)
	globalKeys := globalStyle.Render("↑↓") + " " + globalStyle.Render("/") + " " +
		globalStyle.Render("S") + " " + globalStyle.Render("?") + " " + globalStyle.Render("q")

	// Calculate padding
	leftPart := contextKeys
	rightPart := globalKeys
	padding := h.width - lipgloss.Width(leftPart) - lipgloss.Width(rightPart) - 4
	if padding < 2 {
		// Content too wide for one line — drop right part to avoid overflow
		padding = 2
		rightPart = ""
	}

	content := leftPart + sep + strings.Repeat(" ", padding) + rightPart

	raw := lipgloss.JoinVertical(lipgloss.Left, border, content)
	return lipgloss.NewStyle().MaxWidth(h.width).Render(raw)
}

// renderHelpBarCompact renders abbreviated help for medium terminals (70-99 cols)
func (h *Home) renderHelpBarCompact() string {
	borderStyle := lipgloss.NewStyle().Foreground(ColorBorder)
	border := borderStyle.Render(strings.Repeat("─", max(0, h.width)))

	sepStyle := lipgloss.NewStyle().Foreground(ColorBorder)
	sep := sepStyle.Render(" │ ")

	// Abbreviated key+short desc
	var contextHints []string
	if len(h.flatItems) == 0 {
		contextHints = []string{
			h.helpKeyShort("n/N", "New"),
			h.helpKeyShort("i", "Import"),
		}
	} else if h.cursor < len(h.flatItems) {
		item := h.flatItems[h.cursor]
		if item.Type == session.ItemTypeGroup {
			contextHints = []string{
				h.helpKeyShort("⏎", "Toggle"),
				h.helpKeyShort("n/N", "New"),
			}
		} else {
			contextHints = []string{
				h.helpKeyShort("⏎", "Attach"),
				h.helpKeyShort("n/N", "New"),
				h.helpKeyShort("R", "Restart"),
			}
			if item.Session != nil && item.Session.CanFork() {
				contextHints = append(contextHints, h.helpKeyShort("f", "Fork"))
			}
			if item.Session != nil && (item.Session.Tool == "claude" || item.Session.Tool == "gemini") {
				contextHints = append(contextHints, h.helpKeyShort("m", "MCP"))
				contextHints = append(contextHints, h.helpKeyShort("v", h.previewModeShort()))
			}
			if item.Session != nil && item.Session.Tool == "claude" {
				contextHints = append(contextHints, h.helpKeyShort("s", "Skills"))
			}
			contextHints = append(contextHints, h.helpKeyShort("c", "Copy"))
			contextHints = append(contextHints, h.helpKeyShort("x", "Send"))
		}
	}

	// Show undo hint when undo stack is non-empty
	if len(h.undoStack) > 0 {
		contextHints = append(contextHints, h.helpKeyShort("^Z", "Undo"))
	}

	// Global hints (abbreviated)
	globalStyle := lipgloss.NewStyle().Foreground(ColorComment)
	globalHints := globalStyle.Render("↑↓ Nav") + " " +
		globalStyle.Render("/") + " " +
		globalStyle.Render("S") + " " +
		globalStyle.Render("?") + " " +
		globalStyle.Render("q")

	leftPart := strings.Join(contextHints, " ")
	rightPart := globalHints
	padding := h.width - lipgloss.Width(leftPart) - lipgloss.Width(rightPart) - 4
	if padding < 2 {
		// Content too wide for one line — drop right part to avoid overflow
		padding = 2
		rightPart = ""
	}

	content := leftPart + sep + strings.Repeat(" ", padding) + rightPart

	raw := lipgloss.JoinVertical(lipgloss.Left, border, content)
	return lipgloss.NewStyle().MaxWidth(h.width).Render(raw)
}

// helpKeyShort formats a compact keyboard shortcut (no padding)
func (h *Home) helpKeyShort(key, desc string) string {
	keyStyle := lipgloss.NewStyle().
		Foreground(ColorBg).
		Background(ColorAccent).
		Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(ColorText)
	return keyStyle.Render(key) + descStyle.Render(desc)
}

// previewModeShort returns a short description of current preview mode for help bar
func (h *Home) previewModeShort() string {
	switch h.previewMode {
	case PreviewModeOutput:
		return "Out"
	case PreviewModeAnalytics:
		return "Stats"
	default:
		return "Both"
	}
}

// renderHelpBarFull renders context-aware keyboard shortcuts with visual grouping (100+ cols)
func (h *Home) renderHelpBarFull() string {
	// Separator style for grouping related actions
	sepStyle := lipgloss.NewStyle().Foreground(ColorBorder)
	sep := sepStyle.Render(" │ ")

	// Determine context-specific hints grouped by action type
	var primaryHints []string   // Main actions (attach, toggle, etc.)
	var secondaryHints []string // Edit actions (rename, move, delete)
	var contextTitle string

	if len(h.flatItems) == 0 {
		contextTitle = "Empty"
		primaryHints = []string{
			h.helpKey("n/N", "New/Quick"),
			h.helpKey("i", "Import"),
			h.helpKey("g", "Group"),
		}
	} else if h.cursor < len(h.flatItems) {
		item := h.flatItems[h.cursor]
		if item.Type == session.ItemTypeGroup {
			contextTitle = "Group"
			primaryHints = []string{
				h.helpKey("Tab", "Toggle"),
				h.helpKey("n/N", "New/Quick"),
				h.helpKey("g", "Group"),
			}
			secondaryHints = []string{
				h.helpKey("r", "Rename"),
				h.helpKey("d", "Delete"),
			}
		} else {
			contextTitle = "Session"
			primaryHints = []string{
				h.helpKey("Enter", "Attach"),
				h.helpKey("n/N", "New/Quick"),
				h.helpKey("g", "Group"),
				h.helpKey("R", "Restart"),
			}
			// Only show fork hints if session has a valid Claude session ID
			if item.Session != nil && item.Session.CanFork() {
				primaryHints = append(primaryHints, h.helpKey("f/F", "Fork"))
			}
			// Show MCP Manager and preview mode toggle for Claude and Gemini sessions
			if item.Session != nil && (item.Session.Tool == "claude" || item.Session.Tool == "gemini") {
				primaryHints = append(primaryHints, h.helpKey("m", "MCP"))
				primaryHints = append(primaryHints, h.helpKey("v", h.previewModeShort()))
			}
			if item.Session != nil && item.Session.Tool == "claude" {
				primaryHints = append(primaryHints, h.helpKey("s", "Skills"))
			}
			if item.Session != nil && item.Session.IsSandboxed() {
				primaryHints = append(primaryHints, h.helpKey("E", "Exec"))
			}
			primaryHints = append(primaryHints, h.helpKey("c", "Copy"))
			primaryHints = append(primaryHints, h.helpKey("x", "Send"))
			secondaryHints = []string{
				h.helpKey("r", "Rename"),
				h.helpKey("M", "Move"),
				h.helpKey("d", "Delete"),
			}
		}
	}

	// Show undo hint when undo stack is non-empty
	if len(h.undoStack) > 0 {
		secondaryHints = append(secondaryHints, h.helpKey("^Z", "Undo"))
	}

	// Top border
	borderStyle := lipgloss.NewStyle().Foreground(ColorBorder)
	border := borderStyle.Render(strings.Repeat("─", max(0, h.width)))

	// Context indicator with subtle styling
	ctxStyle := lipgloss.NewStyle().
		Foreground(ColorPurple).
		Bold(true)
	contextLabel := ctxStyle.Render(contextTitle + ":")

	// Build shortcuts line with visual grouping
	var shortcutsLine string
	shortcutsLine = strings.Join(primaryHints, " ")
	if len(secondaryHints) > 0 {
		shortcutsLine += sep + strings.Join(secondaryHints, " ")
	}

	// Reload indicator
	var reloadIndicator string
	h.reloadMu.Lock()
	reloading := h.isReloading
	h.reloadMu.Unlock()
	if reloading {
		reloadStyle := lipgloss.NewStyle().
			Foreground(ColorYellow).
			Bold(true)
		reloadIndicator = reloadStyle.Render("⟳ Reloading...")
	}

	// Global shortcuts (right side) - more compact with separators
	globalStyle := lipgloss.NewStyle().Foreground(ColorComment)
	globalHints := globalStyle.Render("↑↓ Nav") + sep +
		globalStyle.Render("/ Search  G Global") + sep +
		globalStyle.Render("S Settings  ? Help  q Quit")

	// Calculate spacing between left (context) and right (global) portions
	leftPart := contextLabel + " " + shortcutsLine
	if reloadIndicator != "" {
		leftPart = reloadIndicator + sep + leftPart
	}
	rightPart := globalHints
	padding := h.width - lipgloss.Width(leftPart) - lipgloss.Width(rightPart) - spacingNormal
	if padding < spacingNormal {
		// Content too wide for one line — drop right part to avoid overflow
		padding = spacingNormal
		rightPart = ""
	}

	helpContent := leftPart + strings.Repeat(" ", padding) + rightPart

	raw := lipgloss.JoinVertical(lipgloss.Left, border, helpContent)
	return lipgloss.NewStyle().MaxWidth(h.width).Render(raw)
}

// helpKey formats a keyboard shortcut for the help bar
func (h *Home) helpKey(key, desc string) string {
	keyStyle := lipgloss.NewStyle().
		Foreground(ColorBg).
		Background(ColorAccent).
		Bold(true).
		Padding(0, 1)
	descStyle := lipgloss.NewStyle().Foreground(ColorText)
	return keyStyle.Render(key) + " " + descStyle.Render(desc)
}

// --- KanbanNav: 2D Cursor Navigation ---

// KanbanNav encapsulates the kanban board navigation state
type KanbanNav struct {
	Col              int        // Current column index (0-5)
	Row              int        // Current card index within column
	Focus            FocusPanel // Which panel has focus
	DetailOpen       bool       // Whether detail panel is visible
	ColumnCardCounts [6]int     // Number of cards in each column
	ScrollOffsets    [6]int     // Scroll offset for each column (first visible card index)
	MoveMode         bool       // Whether in move mode
	MoveTargetCol    int        // Target column for move
	MoveSourceCol    int        // Source column (where card was)
}

// MoveLeft moves the cursor to the previous column (h key).
// Clamps at column 0, skips empty columns, and clamps row to the new column's card count.
// Only works when Focus == PanelBoard.
func MoveLeft(nav KanbanNav) KanbanNav {
	if nav.Focus != PanelBoard {
		return nav
	}

	// Search left for a non-empty column
	newCol := nav.Col
	for newCol > 0 {
		newCol--
		if nav.ColumnCardCounts[newCol] > 0 {
			// Clamp row to the new column's card count
			newRow := nav.Row
			if newRow >= nav.ColumnCardCounts[newCol] {
				newRow = nav.ColumnCardCounts[newCol] - 1
			}
			return KanbanNav{
				Col:              newCol,
				Row:              newRow,
				Focus:            nav.Focus,
				DetailOpen:       nav.DetailOpen,
				ColumnCardCounts: nav.ColumnCardCounts,
				ScrollOffsets:    nav.ScrollOffsets,
				MoveMode:         nav.MoveMode,
				MoveTargetCol:    nav.MoveTargetCol,
				MoveSourceCol:    nav.MoveSourceCol,
			}
		}
	}

	// No non-empty column found to the left
	return nav
}

// MoveRight moves the cursor to the next column (l key).
// Clamps at column 5, skips empty columns, and clamps row to the new column's card count.
// Only works when Focus == PanelBoard.
func MoveRight(nav KanbanNav) KanbanNav {
	if nav.Focus != PanelBoard {
		return nav
	}

	// Search right for a non-empty column
	newCol := nav.Col
	for newCol < 5 {
		newCol++
		if nav.ColumnCardCounts[newCol] > 0 {
			// Clamp row to the new column's card count
			newRow := nav.Row
			if newRow >= nav.ColumnCardCounts[newCol] {
				newRow = nav.ColumnCardCounts[newCol] - 1
			}
			return KanbanNav{
				Col:              newCol,
				Row:              newRow,
				Focus:            nav.Focus,
				DetailOpen:       nav.DetailOpen,
				ColumnCardCounts: nav.ColumnCardCounts,
				ScrollOffsets:    nav.ScrollOffsets,
				MoveMode:         nav.MoveMode,
				MoveTargetCol:    nav.MoveTargetCol,
				MoveSourceCol:    nav.MoveSourceCol,
			}
		}
	}

	// No non-empty column found to the right
	return nav
}

// MoveUp moves the cursor to the previous card (k key).
// Clamps at row 0. Only works when Focus == PanelBoard.
func MoveUp(nav KanbanNav) KanbanNav {
	if nav.Focus != PanelBoard {
		return nav
	}

	if nav.Row > 0 {
		return KanbanNav{
			Col:              nav.Col,
			Row:              nav.Row - 1,
			Focus:            nav.Focus,
			DetailOpen:       nav.DetailOpen,
			ColumnCardCounts: nav.ColumnCardCounts,
			ScrollOffsets:    nav.ScrollOffsets,
			MoveMode:         nav.MoveMode,
			MoveTargetCol:    nav.MoveTargetCol,
			MoveSourceCol:    nav.MoveSourceCol,
		}
	}

	return nav
}

// MoveDown moves the cursor to the next card (j key).
// Clamps at ColumnCardCounts[Col]-1. Only works when Focus == PanelBoard.
func MoveDown(nav KanbanNav) KanbanNav {
	if nav.Focus != PanelBoard {
		return nav
	}

	maxRow := nav.ColumnCardCounts[nav.Col] - 1
	if nav.Row < maxRow {
		return KanbanNav{
			Col:              nav.Col,
			Row:              nav.Row + 1,
			Focus:            nav.Focus,
			DetailOpen:       nav.DetailOpen,
			ColumnCardCounts: nav.ColumnCardCounts,
			ScrollOffsets:    nav.ScrollOffsets,
			MoveMode:         nav.MoveMode,
			MoveTargetCol:    nav.MoveTargetCol,
			MoveSourceCol:    nav.MoveSourceCol,
		}
	}

	return nav
}

// CycleTabFocus cycles focus between panels with Tab key.
// Order: Sidebar -> Board -> Detail (if open) -> Sidebar
func CycleTabFocus(nav KanbanNav) KanbanNav {
	var newFocus FocusPanel

	switch nav.Focus {
	case PanelSidebar:
		newFocus = PanelBoard
	case PanelBoard:
		if nav.DetailOpen {
			newFocus = PanelDetail
		} else {
			newFocus = PanelSidebar
		}
	case PanelDetail:
		newFocus = PanelSidebar
	default:
		newFocus = PanelSidebar
	}

	return KanbanNav{
		Col:              nav.Col,
		Row:              nav.Row,
		Focus:            newFocus,
		DetailOpen:       nav.DetailOpen,
		ColumnCardCounts: nav.ColumnCardCounts,
		ScrollOffsets:    nav.ScrollOffsets,
		MoveMode:         nav.MoveMode,
		MoveTargetCol:    nav.MoveTargetCol,
		MoveSourceCol:    nav.MoveSourceCol,
	}
}

// JumpToColumn jumps to a specific column (1-6 keys).
// Sets Col to target (0-5), resets Row to 0, and sets Focus to PanelBoard.
// Only works when Focus == PanelBoard.
func JumpToColumn(nav KanbanNav, col int) KanbanNav {
	if nav.Focus != PanelBoard {
		return nav
	}

	// Validate column range
	if col < 0 || col > 5 {
		return nav
	}

	return KanbanNav{
		Col:              col,
		Row:              0,
		Focus:            PanelBoard,
		DetailOpen:       nav.DetailOpen,
		ColumnCardCounts: nav.ColumnCardCounts,
		ScrollOffsets:    nav.ScrollOffsets,
		MoveMode:         nav.MoveMode,
		MoveTargetCol:    nav.MoveTargetCol,
		MoveSourceCol:    nav.MoveSourceCol,
	}
}

// ToggleDetail toggles the detail panel (Space key).
// If closing and Focus == PanelDetail, moves focus to PanelBoard.
func ToggleDetail(nav KanbanNav) KanbanNav {
	newDetailOpen := !nav.DetailOpen
	newFocus := nav.Focus

	// If closing detail and focus is on detail, move focus to board
	if !newDetailOpen && nav.Focus == PanelDetail {
		newFocus = PanelBoard
	}

	return KanbanNav{
		Col:              nav.Col,
		Row:              nav.Row,
		Focus:            newFocus,
		DetailOpen:       newDetailOpen,
		ColumnCardCounts: nav.ColumnCardCounts,
		ScrollOffsets:    nav.ScrollOffsets,
		MoveMode:         nav.MoveMode,
		MoveTargetCol:    nav.MoveTargetCol,
		MoveSourceCol:    nav.MoveSourceCol,
	}
}

// --- Move Mode Functions ---

// EnterMoveMode enters move mode, setting the target to the current column.
// Only works when Focus == PanelBoard.
func EnterMoveMode(nav KanbanNav) KanbanNav {
	if nav.Focus != PanelBoard {
		return nav
	}

	return KanbanNav{
		Col:              nav.Col,
		Row:              nav.Row,
		Focus:            nav.Focus,
		DetailOpen:       nav.DetailOpen,
		ColumnCardCounts: nav.ColumnCardCounts,
		ScrollOffsets:    nav.ScrollOffsets,
		MoveMode:         true,
		MoveTargetCol:    nav.Col,
		MoveSourceCol:    nav.Col,
	}
}

// MoveModeLeft moves the target column left in move mode.
// Clamps at column 0.
func MoveModeLeft(nav KanbanNav) KanbanNav {
	if !nav.MoveMode {
		return nav
	}

	newTargetCol := nav.MoveTargetCol
	if newTargetCol > 0 {
		newTargetCol--
	}

	return KanbanNav{
		Col:              nav.Col,
		Row:              nav.Row,
		Focus:            nav.Focus,
		DetailOpen:       nav.DetailOpen,
		ColumnCardCounts: nav.ColumnCardCounts,
		ScrollOffsets:    nav.ScrollOffsets,
		MoveMode:         nav.MoveMode,
		MoveTargetCol:    newTargetCol,
		MoveSourceCol:    nav.MoveSourceCol,
	}
}

// MoveModeRight moves the target column right in move mode.
// Clamps at column 5.
func MoveModeRight(nav KanbanNav) KanbanNav {
	if !nav.MoveMode {
		return nav
	}

	newTargetCol := nav.MoveTargetCol
	if newTargetCol < 5 {
		newTargetCol++
	}

	return KanbanNav{
		Col:              nav.Col,
		Row:              nav.Row,
		Focus:            nav.Focus,
		DetailOpen:       nav.DetailOpen,
		ColumnCardCounts: nav.ColumnCardCounts,
		ScrollOffsets:    nav.ScrollOffsets,
		MoveMode:         nav.MoveMode,
		MoveTargetCol:    newTargetCol,
		MoveSourceCol:    nav.MoveSourceCol,
	}
}

// ExitMoveMode exits move mode without moving.
func ExitMoveMode(nav KanbanNav) KanbanNav {
	return KanbanNav{
		Col:              nav.Col,
		Row:              nav.Row,
		Focus:            nav.Focus,
		DetailOpen:       nav.DetailOpen,
		ColumnCardCounts: nav.ColumnCardCounts,
		ScrollOffsets:    nav.ScrollOffsets,
		MoveMode:         false,
		MoveTargetCol:    0,
		MoveSourceCol:    0,
	}
}

// ConfirmMove returns the confirmed move details (source, target) and exits move mode.
// Returns (nav, sourceCol, targetCol).
func ConfirmMove(nav KanbanNav) (KanbanNav, int, int) {
	sourceCol := nav.MoveSourceCol
	targetCol := nav.MoveTargetCol

	newNav := KanbanNav{
		Col:              nav.Col,
		Row:              nav.Row,
		Focus:            nav.Focus,
		DetailOpen:       nav.DetailOpen,
		ColumnCardCounts: nav.ColumnCardCounts,
		ScrollOffsets:    nav.ScrollOffsets,
		MoveMode:         false,
		MoveTargetCol:    0,
		MoveSourceCol:    0,
	}

	return newNav, sourceCol, targetCol
}

// --- Scroll Helpers ---

// CalculateVisibleCardRange returns the start/end indices of visible cards for a column.
// Parameters:
//   - totalCards: total number of cards in the column
//   - scrollOffset: index of the first card to display (scroll position)
//   - availableHeight: height in lines available for cards
//   - cardHeight: height of a single card in lines (typically 1 for compact cards)
//
// Returns (start, end) where start is inclusive and end is exclusive.
// If no cards can fit, returns (scrollOffset, scrollOffset).
func CalculateVisibleCardRange(totalCards, scrollOffset, availableHeight, cardHeight int) (start, end int) {
	if availableHeight <= 0 || cardHeight <= 0 {
		return scrollOffset, scrollOffset
	}

	// Calculate how many cards can fit in available height
	visibleCards := availableHeight / cardHeight
	if visibleCards <= 0 {
		return scrollOffset, scrollOffset
	}

	start = scrollOffset
	end = scrollOffset + visibleCards

	// Clamp to total cards
	if end > totalCards {
		end = totalCards
	}

	return start, end
}

// AutoScrollToSelection adjusts scroll offset to keep the selected row visible.
// Returns a new KanbanNav with updated ScrollOffsets for the selected column.
// Only modifies the scroll offset for the currently selected column.
func AutoScrollToSelection(nav KanbanNav, availableHeight, cardHeight int) KanbanNav {
	if availableHeight <= 0 || cardHeight <= 0 {
		return nav
	}

	col := nav.Col
	row := nav.Row
	scrollOffset := nav.ScrollOffsets[col]
	totalCards := nav.ColumnCardCounts[col]

	// Calculate visible range
	start, end := CalculateVisibleCardRange(totalCards, scrollOffset, availableHeight, cardHeight)

	// Check if selected row is visible
	if row < start {
		// Selected row is above viewport, scroll up to show it
		scrollOffset = row
	} else if row >= end {
		// Selected row is below viewport, scroll down to show it
		// Position selected row at the bottom of the viewport
		visibleCards := availableHeight / cardHeight
		scrollOffset = row - visibleCards + 1
		if scrollOffset < 0 {
			scrollOffset = 0
		}
	}
	// else: row is in view, no scroll needed

	// Create new ScrollOffsets array with updated value
	newScrollOffsets := nav.ScrollOffsets
	newScrollOffsets[col] = scrollOffset

	return KanbanNav{
		Col:              nav.Col,
		Row:              nav.Row,
		Focus:            nav.Focus,
		DetailOpen:       nav.DetailOpen,
		ColumnCardCounts: nav.ColumnCardCounts,
		ScrollOffsets:    newScrollOffsets,
		MoveMode:         nav.MoveMode,
		MoveTargetCol:    nav.MoveTargetCol,
		MoveSourceCol:    nav.MoveSourceCol,
	}
}
