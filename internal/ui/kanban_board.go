package ui

// kanban_board.go — Board layout, responsive rendering, and empty states extracted from home.go
//
// Contains: layout constants, empty state constants, EmptyStateConfig,
// renderEmptyStateResponsive, ensureExactHeight, ensureExactWidth,
// renderDualColumnLayout, renderStackedLayout, renderSingleColumnLayout,
// renderPanelTitle, renderFilterBar, renderLoadingSplash, renderQuittingSplash

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/asheshgoplani/agent-deck/internal/session"
)

// Layout mode breakpoints for responsive design
const (
	layoutBreakpointSingle  = 50 // Below: single column, no preview
	layoutBreakpointStacked = 80 // Below: stacked layout (list above preview)
	// At or above 80: dual column (current side-by-side layout)
)

// Layout mode names
const (
	LayoutModeSingle  = "single"  // <50 cols: list only
	LayoutModeStacked = "stacked" // 50-79 cols: vertical stack
	LayoutModeDual    = "dual"    // 80+ cols: side-by-side
)

// Responsive breakpoints for empty state content tiers
// These define when to show full/compact/minimal content
const (
	// Width breakpoints (for left panel after 35% split)
	emptyStateWidthFull    = 45 // Full content with all hints
	emptyStateWidthCompact = 35 // Compact: fewer hints, shorter text
	// Below 35: minimal mode (icon + title + 1 hint)

	// Height breakpoints (for content area)
	emptyStateHeightFull    = 18 // Full content with generous spacing
	emptyStateHeightCompact = 12 // Compact: reduced spacing
	// Below 12: minimal mode
)

// EmptyStateConfig holds content for responsive empty state rendering
type EmptyStateConfig struct {
	Icon     string
	Title    string
	Subtitle string
	Hints    []string // Full list of hints (will be reduced based on space)
}

// renderEmptyStateResponsive creates a centered empty state that adapts to available space
// Uses progressive disclosure: full -> compact -> minimal based on width/height
func renderEmptyStateResponsive(config EmptyStateConfig, width, height int) string {
	// Determine content tier based on available space
	// Use the more restrictive of width or height constraints
	tier := "full"
	if width < emptyStateWidthCompact || height < emptyStateHeightCompact {
		tier = "minimal"
	} else if width < emptyStateWidthFull || height < emptyStateHeightFull {
		tier = "compact"
	}

	// Adaptive padding based on tier
	var vPad, hPad int
	switch tier {
	case "full":
		vPad, hPad = spacingNormal, spacingLarge
	case "compact":
		vPad, hPad = spacingTight, spacingNormal
	case "minimal":
		vPad, hPad = 0, spacingTight
	}

	// Styles
	iconStyle := lipgloss.NewStyle().
		Foreground(ColorAccent).
		Bold(true)
	titleStyle := lipgloss.NewStyle().
		Foreground(ColorText).
		Bold(true)
	subtitleStyle := lipgloss.NewStyle().
		Foreground(ColorText).
		Italic(true)
	hintStyle := lipgloss.NewStyle().
		Foreground(ColorComment)

	var content strings.Builder

	// Icon - always shown but with adaptive spacing
	content.WriteString(iconStyle.Render(config.Icon))
	if tier == "full" {
		content.WriteString("\n\n")
	} else {
		content.WriteString("\n")
	}

	// Title - always shown
	content.WriteString(titleStyle.Render(config.Title))

	// Subtitle - shown in full and compact modes
	if config.Subtitle != "" && tier != "minimal" {
		content.WriteString("\n")
		// Truncate subtitle if width is tight
		subtitle := config.Subtitle
		maxSubtitleWidth := width - hPad*2 - 4 // Account for padding and margins
		if maxSubtitleWidth > 0 && len(subtitle) > maxSubtitleWidth {
			subtitle = subtitle[:maxSubtitleWidth-3] + "..."
		}
		content.WriteString(subtitleStyle.Render(subtitle))
	}

	// Hints - progressive disclosure based on tier
	if len(config.Hints) > 0 {
		var hintsToShow []string
		switch tier {
		case "full":
			hintsToShow = config.Hints // Show all
		case "compact":
			// Show first 2 hints max
			if len(config.Hints) > 2 {
				hintsToShow = config.Hints[:2]
			} else {
				hintsToShow = config.Hints
			}
		case "minimal":
			// Show only the first (most important) hint
			hintsToShow = config.Hints[:1]
		}

		if tier == "full" {
			content.WriteString("\n\n")
		} else {
			content.WriteString("\n")
		}

		for i, hint := range hintsToShow {
			// Truncate hint if width is tight
			displayHint := hint
			maxHintWidth := width - hPad*2 - 6 // Account for "• " prefix and margins
			if maxHintWidth > 0 && len(displayHint) > maxHintWidth {
				displayHint = displayHint[:maxHintWidth-3] + "..."
			}
			content.WriteString(hintStyle.Render("• " + displayHint))
			if i < len(hintsToShow)-1 {
				content.WriteString("\n")
			}
		}
	}

	contentStyle := lipgloss.NewStyle().
		Foreground(ColorText).
		Align(lipgloss.Center).
		Padding(vPad, hPad).
		MaxWidth(width)

	rendered := contentStyle.Render(content.String())

	// Ensure exact height
	return ensureExactHeight(rendered, height)
}

// ensureExactHeight is a critical helper that ensures any content has EXACTLY n lines.
// This is essential for consistent TUI layout across all platforms and terminal sizes.
//
// Behavior:
//   - If content has fewer lines than n: pads with blank lines at the end
//   - If content has more lines than n: truncates from the end (keeps header/start)
//   - Returns content with exactly n lines (n-1 internal newlines, no trailing newline)
//
// This function handles ANSI-styled content correctly by counting \n characters
// rather than visual lines, which works reliably across all terminal emulators.
func ensureExactHeight(content string, n int) string {
	if n <= 0 {
		return ""
	}

	// Split into lines
	lines := strings.Split(content, "\n")

	// Truncate or pad to exactly n lines
	if len(lines) > n {
		// Keep first n lines (preserves header info)
		lines = lines[:n]
	} else if len(lines) < n {
		// Pad with blank lines
		for len(lines) < n {
			lines = append(lines, "")
		}
	}

	// Join back - this creates n-1 newlines for n lines
	return strings.Join(lines, "\n")
}

// ensureExactWidth ensures each line in content has exactly the specified visual width.
// This is essential for proper horizontal panel alignment in lipgloss.JoinHorizontal.
//
// CRITICAL: Uses lipgloss.Width() for measurement to stay consistent with
// lipgloss.JoinHorizontal's internal width calculation. Using a different
// measurement (e.g. runewidth.StringWidth after custom ANSI stripping) can
// disagree by even 1 character, causing JoinHorizontal to pad all lines to
// the wider measurement. This makes the joined output exceed terminal width,
// lines wrap, and Bubble Tea's renderer loses cursor tracking — producing
// duplicated/stacked content (the "scrolling artifact" bug).
//
// Behavior:
//   - Measures width using lipgloss.Width (same as JoinHorizontal)
//   - Pads short lines with spaces to reach target width
//   - Truncates long lines using lipgloss.MaxWidth (preserves ANSI where possible)
//   - Guarantees every line is exactly `width` visual characters
func ensureExactWidth(content string, width int) string {
	if width <= 0 {
		return content
	}

	lines := strings.Split(content, "\n")
	result := make([]string, len(lines))

	for i, line := range lines {
		// Measure visual width using lipgloss (same measurement as JoinHorizontal)
		displayWidth := lipgloss.Width(line)

		if displayWidth == width {
			result[i] = line
		} else if displayWidth < width {
			// Pad with spaces to reach target width
			result[i] = line + strings.Repeat(" ", width-displayWidth)
		} else {
			// Line too wide — truncate using lipgloss for consistent ANSI handling
			truncated := lipgloss.NewStyle().MaxWidth(width).Render(line)
			// Verify and pad if truncation left it short
			truncWidth := lipgloss.Width(truncated)
			if truncWidth < width {
				truncated += strings.Repeat(" ", width-truncWidth)
			}
			result[i] = truncated
		}
	}

	return strings.Join(result, "\n")
}

// renderDualColumnLayout renders side-by-side panels for wide terminals (80+ cols)
func (h *Home) renderDualColumnLayout(contentHeight int) string {
	var b strings.Builder

	// Calculate panel widths (35% left, 65% right for more preview space)
	leftWidth := int(float64(h.width) * 0.35)
	rightWidth := h.width - leftWidth - 3 // -3 for separator

	// Panel title is exactly 2 lines (title + underline)
	// Panel content gets the remaining space: contentHeight - 2
	panelTitleLines := 2
	panelContentHeight := contentHeight - panelTitleLines

	// Build left panel (session list) with styled title
	leftTitle := h.renderPanelTitle("SESSIONS", leftWidth)
	leftContent := h.renderSessionList(leftWidth, panelContentHeight)
	// CRITICAL: Ensure left content has exactly panelContentHeight lines
	leftContent = ensureExactHeight(leftContent, panelContentHeight)
	leftPanel := leftTitle + "\n" + leftContent

	// Build right panel (preview) with styled title
	rightTitle := h.renderPanelTitle("PREVIEW", rightWidth)
	rightContent := h.renderPreviewPane(rightWidth, panelContentHeight)
	// CRITICAL: Ensure right content has exactly panelContentHeight lines
	rightContent = ensureExactHeight(rightContent, panelContentHeight)
	rightPanel := rightTitle + "\n" + rightContent

	// Build separator - must be exactly contentHeight lines
	separatorStyle := lipgloss.NewStyle().Foreground(ColorBorder)
	separatorLines := make([]string, contentHeight)
	for i := range separatorLines {
		separatorLines[i] = separatorStyle.Render(" │ ")
	}
	separator := strings.Join(separatorLines, "\n")

	// CRITICAL: Ensure both panels have exactly contentHeight lines before joining
	leftPanel = ensureExactHeight(leftPanel, contentHeight)
	rightPanel = ensureExactHeight(rightPanel, contentHeight)

	// CRITICAL: Ensure both panels have exactly the correct width for proper alignment
	// Without this, variable-width lines cause JoinHorizontal to misalign content
	leftPanel = ensureExactWidth(leftPanel, leftWidth)
	rightPanel = ensureExactWidth(rightPanel, rightWidth)

	// Join panels horizontally - all components have exact heights AND widths now
	mainContent := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, separator, rightPanel)

	// Safety net: enforce per-line MaxWidth on the joined output.
	// Even with ensureExactWidth, JoinHorizontal can produce lines wider than
	// h.width due to separator ANSI codes or rounding. Any line that wraps in the
	// terminal adds a visual line, which shifts Bubble Tea's cursor tracking and
	// causes duplicated/stacked content on scroll.
	mainContent = lipgloss.NewStyle().MaxWidth(h.width).Render(mainContent)

	b.WriteString(mainContent)

	return b.String()
}

// renderStackedLayout renders list above preview for medium terminals (50-79 cols)
func (h *Home) renderStackedLayout(totalHeight int) string {
	var b strings.Builder

	// Split height: 60% list, 40% preview
	listHeight := (totalHeight * 60) / 100
	previewHeight := totalHeight - listHeight - 1 // -1 for separator

	if listHeight < 5 {
		listHeight = 5
	}
	if previewHeight < 3 {
		previewHeight = 3
	}

	// Session list (full width)
	listTitle := h.renderPanelTitle("SESSIONS", h.width)
	listContent := h.renderSessionList(h.width, listHeight-2) // -2 for title
	listContent = ensureExactHeight(listContent, listHeight-2)
	b.WriteString(listTitle)
	b.WriteString("\n")
	b.WriteString(listContent)
	b.WriteString("\n")

	// Separator
	sepStyle := lipgloss.NewStyle().Foreground(ColorBorder)
	b.WriteString(sepStyle.Render(strings.Repeat("─", max(0, h.width))))
	b.WriteString("\n")

	// Preview (full width)
	previewTitle := h.renderPanelTitle("PREVIEW", h.width)
	previewContent := h.renderPreviewPane(h.width, previewHeight-2) // -2 for title
	previewContent = ensureExactHeight(previewContent, previewHeight-2)
	b.WriteString(previewTitle)
	b.WriteString("\n")
	b.WriteString(previewContent)

	return b.String()
}

// renderSingleColumnLayout renders list only for narrow terminals (<50 cols)
func (h *Home) renderSingleColumnLayout(totalHeight int) string {
	var b strings.Builder

	// Full height for list
	listHeight := totalHeight - 2 // -2 for title

	listTitle := h.renderPanelTitle("SESSIONS", h.width)
	listContent := h.renderSessionList(h.width, listHeight)
	listContent = ensureExactHeight(listContent, listHeight)

	b.WriteString(listTitle)
	b.WriteString("\n")
	b.WriteString(listContent)

	return b.String()
}

// renderPanelTitle creates a styled section title with underline
func (h *Home) renderPanelTitle(title string, width int) string {
	// Truncate title if it exceeds width
	if len(title) > width {
		if width > 3 {
			title = title[:width-3] + "..."
		} else {
			title = title[:width]
		}
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(ColorCyan).
		Bold(true).
		Width(width)

	underlineStyle := lipgloss.NewStyle().
		Foreground(ColorBorder).
		Width(width)

	// Create underline that extends to panel width
	underlineLen := max(0, width)
	underline := underlineStyle.Render(strings.Repeat("─", underlineLen))

	return titleStyle.Render(title) + "\n" + underline
}

// renderFilterBar renders the quick filter pills
// Format: [All] [● Running 2] [◐ Waiting 1] [○ Idle 5] [✕ Error 1]
func (h *Home) renderFilterBar() string {
	running, waiting, idle, errored := h.countSessionStatuses()

	// Pill styling
	activePillStyle := lipgloss.NewStyle().
		Foreground(ColorBg).
		Background(ColorAccent).
		Bold(true).
		Padding(0, 1)

	inactivePillStyle := lipgloss.NewStyle().
		Foreground(ColorText).
		Background(ColorSurface).
		Padding(0, 1)

	dimPillStyle := lipgloss.NewStyle().
		Foreground(ColorText).
		Faint(true).
		Padding(0, 1)

	// Build pills
	var pills []string

	// "All" pill
	allLabel := "All"
	if h.statusFilter == "" {
		pills = append(pills, activePillStyle.Render(allLabel))
	} else {
		pills = append(pills, inactivePillStyle.Render(allLabel))
	}

	// Running pill (green when active, dim if 0)
	runningLabel := fmt.Sprintf("● %d", running)
	if h.statusFilter == session.StatusRunning {
		pills = append(pills, lipgloss.NewStyle().
			Foreground(ColorBg).
			Background(ColorGreen).
			Bold(true).
			Padding(0, 1).Render(runningLabel))
	} else if running > 0 {
		pills = append(pills, lipgloss.NewStyle().
			Foreground(ColorGreen).
			Background(ColorSurface).
			Padding(0, 1).Render(runningLabel))
	} else {
		pills = append(pills, dimPillStyle.Render(runningLabel))
	}

	// Waiting pill (yellow when active)
	waitingLabel := fmt.Sprintf("◐ %d", waiting)
	if h.statusFilter == session.StatusWaiting {
		pills = append(pills, lipgloss.NewStyle().
			Foreground(ColorBg).
			Background(ColorYellow).
			Bold(true).
			Padding(0, 1).Render(waitingLabel))
	} else if waiting > 0 {
		pills = append(pills, lipgloss.NewStyle().
			Foreground(ColorYellow).
			Background(ColorSurface).
			Padding(0, 1).Render(waitingLabel))
	} else {
		pills = append(pills, dimPillStyle.Render(waitingLabel))
	}

	// Idle pill (gray when active)
	idleLabel := fmt.Sprintf("○ %d", idle)
	if h.statusFilter == session.StatusIdle {
		pills = append(pills, lipgloss.NewStyle().
			Foreground(ColorBg).
			Background(ColorTextDim).
			Bold(true).
			Padding(0, 1).Render(idleLabel))
	} else if idle > 0 {
		pills = append(pills, lipgloss.NewStyle().
			Foreground(ColorText).
			Background(ColorSurface).
			Padding(0, 1).Render(idleLabel))
	} else {
		pills = append(pills, dimPillStyle.Render(idleLabel))
	}

	// Error pill (red when active)
	if errored > 0 || h.statusFilter == session.StatusError {
		errorLabel := fmt.Sprintf("✕ %d", errored)
		if h.statusFilter == session.StatusError {
			pills = append(pills, lipgloss.NewStyle().
				Foreground(ColorBg).
				Background(ColorRed).
				Bold(true).
				Padding(0, 1).Render(errorLabel))
		} else if errored > 0 {
			pills = append(pills, lipgloss.NewStyle().
				Foreground(ColorRed).
				Background(ColorSurface).
				Padding(0, 1).Render(errorLabel))
		}
	}

	// Hint for keyboard shortcuts (shift+number to filter, 0 to clear)
	hintStyle := lipgloss.NewStyle().Foreground(ColorComment).Faint(true)
	hint := hintStyle.Render("  !@#$ filter • 0 all")

	// Join pills with spaces (leading space replaces Padding)
	filterRow := " " + strings.Join(pills, " ") + hint

	return lipgloss.NewStyle().
		MaxWidth(h.width).
		Render(filterRow)
}

// renderLoadingSplash creates a simple centered loading splash screen
// Shows the three status indicators (running/waiting/idle) cycling
func renderLoadingSplash(width, height int, frame int) string {
	// Status indicator cycle: each status lights up in sequence
	// Frame 0-1: Running (green ●)
	// Frame 2-3: Waiting (yellow ◐)
	// Frame 4-5: Idle (gray ○)
	// Frame 6-7: All lit together

	phase := (frame / 2) % 4

	// Active status colors (match the actual TUI colors)
	greenStyle := lipgloss.NewStyle().Foreground(ColorGreen).Bold(true)
	yellowStyle := lipgloss.NewStyle().Foreground(ColorYellow).Bold(true)
	grayStyle := lipgloss.NewStyle().Foreground(ColorTextDim)

	// Dim style for inactive indicators
	dimStyle := lipgloss.NewStyle().Foreground(ColorComment)

	// Text styles
	titleStyle := lipgloss.NewStyle().Foreground(ColorText).Bold(true)
	subtitleStyle := lipgloss.NewStyle().Foreground(ColorTextDim)

	var content strings.Builder

	if width >= 40 && height >= 10 {
		// Full version - big status indicators in a row
		var running, waiting, idle string

		switch phase {
		case 0: // Running highlighted
			running = greenStyle.Render("●")
			waiting = dimStyle.Render("◐")
			idle = dimStyle.Render("○")
		case 1: // Waiting highlighted
			running = dimStyle.Render("●")
			waiting = yellowStyle.Render("◐")
			idle = dimStyle.Render("○")
		case 2: // Idle highlighted
			running = dimStyle.Render("●")
			waiting = dimStyle.Render("◐")
			idle = grayStyle.Render("○")
		case 3: // All lit
			running = greenStyle.Render("●")
			waiting = yellowStyle.Render("◐")
			idle = grayStyle.Render("○")
		}

		content.WriteString("\n")
		content.WriteString("      " + running + "   " + waiting + "   " + idle + "      \n")
		content.WriteString("\n")
		content.WriteString(titleStyle.Render("Agent Deck") + "\n")
		content.WriteString("\n")
		content.WriteString(subtitleStyle.Render("Loading sessions..."))
	} else if width >= 25 && height >= 6 {
		// Compact version
		var indicators string
		switch phase {
		case 0:
			indicators = greenStyle.Render("●") + " " + dimStyle.Render("◐") + " " + dimStyle.Render("○")
		case 1:
			indicators = dimStyle.Render("●") + " " + yellowStyle.Render("◐") + " " + dimStyle.Render("○")
		case 2:
			indicators = dimStyle.Render("●") + " " + dimStyle.Render("◐") + " " + grayStyle.Render("○")
		case 3:
			indicators = greenStyle.Render("●") + " " + yellowStyle.Render("◐") + " " + grayStyle.Render("○")
		}
		content.WriteString(indicators + "\n")
		content.WriteString("\n")
		content.WriteString(titleStyle.Render("Agent Deck") + "\n")
		content.WriteString(subtitleStyle.Render("Loading..."))
	} else {
		// Minimal
		content.WriteString(greenStyle.Render("●") + " " + titleStyle.Render("Agent Deck") + "\n")
		content.WriteString(subtitleStyle.Render("Loading..."))
	}

	// Center the content
	contentStyle := lipgloss.NewStyle().
		Align(lipgloss.Center).
		Width(width)

	rendered := lipgloss.Place(
		width, height,
		lipgloss.Center, lipgloss.Center,
		contentStyle.Render(content.String()),
	)

	return rendered
}

// renderQuittingSplash renders a splash screen during application shutdown
func renderQuittingSplash(width, height int, frame int) string {
	// Status indicator cycle (matches loading splash for consistency)
	phase := (frame / 2) % 4

	// Active status colors
	greenStyle := lipgloss.NewStyle().Foreground(ColorGreen).Bold(true)
	yellowStyle := lipgloss.NewStyle().Foreground(ColorYellow).Bold(true)
	grayStyle := lipgloss.NewStyle().Foreground(ColorTextDim)
	dimStyle := lipgloss.NewStyle().Foreground(ColorComment)

	// Text styles
	titleStyle := lipgloss.NewStyle().Foreground(ColorText).Bold(true)
	subtitleStyle := lipgloss.NewStyle().Foreground(ColorYellow)

	var content strings.Builder

	if width >= 40 && height >= 10 {
		var running, waiting, idle string
		switch phase {
		case 0:
			running = greenStyle.Render("●")
			waiting = dimStyle.Render("◐")
			idle = dimStyle.Render("○")
		case 1:
			running = dimStyle.Render("●")
			waiting = yellowStyle.Render("◐")
			idle = dimStyle.Render("○")
		case 2:
			running = dimStyle.Render("●")
			waiting = dimStyle.Render("◐")
			idle = grayStyle.Render("○")
		case 3:
			running = greenStyle.Render("●")
			waiting = yellowStyle.Render("◐")
			idle = grayStyle.Render("○")
		}

		content.WriteString("\n")
		content.WriteString("      " + running + "   " + waiting + "   " + idle + "      \n")
		content.WriteString("\n")
		content.WriteString(titleStyle.Render("Agent Deck") + "\n")
		content.WriteString("\n")
		content.WriteString(subtitleStyle.Render("Shutting down..."))
	} else {
		// Compact/Minimal
		content.WriteString(titleStyle.Render("Agent Deck") + "\n")
		content.WriteString(subtitleStyle.Render("Shutting down..."))
	}

	contentStyle := lipgloss.NewStyle().
		Align(lipgloss.Center).
		Width(width)

	return lipgloss.Place(
		width, height,
		lipgloss.Center, lipgloss.Center,
		contentStyle.Render(content.String()),
	)
}
