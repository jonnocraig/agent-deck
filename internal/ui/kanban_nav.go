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
