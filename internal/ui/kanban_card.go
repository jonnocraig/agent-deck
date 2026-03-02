package ui

// kanban_card.go — Session card and list rendering extracted from home.go
//
// Contains: tree drawing constants, renderItem, renderSessionItem, renderSessionList

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/asheshgoplani/agent-deck/internal/session"
)

// Tree drawing characters for visual hierarchy
const (
	treeBranch = "├─" // Mid-level item (has siblings below)
	treeLast   = "└─" // Last item in group (no siblings below)
	treeLine   = "│ " // Continuation line
	treeEmpty  = "  " // Empty space (for alignment)
	// Sub-session connectors (nested under parent)
	subBranch = "├─" // Sub-session with siblings below
	subLast   = "└─" // Last sub-session
)

// renderItem renders a single item (group or session) for the left panel
func (h *Home) renderItem(b *strings.Builder, item session.Item, selected bool, itemIndex int) {
	if item.Type == session.ItemTypeGroup {
		h.renderGroupItem(b, item, selected, itemIndex)
	} else {
		h.renderSessionItem(b, item, selected)
	}
}

// renderSessionItem renders a single session item for the left panel
// PERFORMANCE: Uses cached styles from styles.go to avoid allocations
func (h *Home) renderSessionItem(b *strings.Builder, item session.Item, selected bool) {
	inst := item.Session

	// Snapshot status and tool under read lock to avoid races with background worker
	instStatus := inst.GetStatusThreadSafe()
	instTool := inst.GetToolThreadSafe()

	// Tree style for connectors - Use ColorText for clear visibility of box-drawing characters
	treeStyle := TreeConnectorStyle

	// Calculate base indentation for parent levels
	// Level 1 means direct child of root group, Level 2 means child of nested group, etc.
	baseIndent := ""
	if item.Level > 1 {
		// For deeply nested items, add spacing for parent levels
		// Sub-sessions get extra indentation (they're at Level = groupLevel + 2)
		if item.IsSubSession {
			// Sub-session: indent for group level, then continuation line for parent
			// Add leading space so │ aligns with ├ in regular items (both at position 1)
			groupIndent := strings.Repeat(treeEmpty, item.Level-2)
			if item.ParentIsLastInGroup {
				baseIndent = groupIndent + "  " // 2 spaces - parent is last, no continuation needed
			} else {
				// Style the │ character - leading space aligns │ with ├ above
				baseIndent = groupIndent + " " + treeStyle.Render("│")
			}
		} else {
			baseIndent = strings.Repeat(treeEmpty, item.Level-1)
		}
	}

	// Tree connector: └─ for last item, ├─ for others
	treeConnector := treeBranch
	if item.IsSubSession {
		// Sub-session uses its own last-in-group logic
		if item.IsLastSubSession {
			treeConnector = subLast
		} else {
			treeConnector = subBranch
		}
	} else if item.IsLastInGroup {
		treeConnector = treeLast
	}

	// Status indicator with consistent sizing
	var statusIcon string
	var statusStyle lipgloss.Style
	switch instStatus {
	case session.StatusRunning:
		statusIcon = "●"
		statusStyle = SessionStatusRunning
	case session.StatusWaiting:
		statusIcon = "◐"
		statusStyle = SessionStatusWaiting
	case session.StatusIdle:
		statusIcon = "○"
		statusStyle = SessionStatusIdle
	case session.StatusError:
		statusIcon = "✕"
		statusStyle = SessionStatusError
	default:
		statusIcon = "○"
		statusStyle = SessionStatusIdle
	}

	status := statusStyle.Render(statusIcon)

	// Title styling - add bold/underline for accessibility (colorblind users)
	var titleStyle lipgloss.Style
	switch instStatus {
	case session.StatusRunning, session.StatusWaiting:
		// Bold for active states (distinguishable without color)
		titleStyle = SessionTitleActive
	case session.StatusError:
		// Underline for error (distinguishable without color)
		titleStyle = SessionTitleError
	default:
		titleStyle = SessionTitleDefault
	}

	// Tool badge with brand-specific color
	// Claude=orange, Gemini=purple, Codex=cyan, Aider=red
	toolStyle := GetToolStyle(instTool)

	// Selection indicator
	selectionPrefix := " "
	if selected {
		selectionPrefix = SessionSelectionPrefix.Render("▶")
		titleStyle = SessionTitleSelStyle
		toolStyle = SessionStatusSelStyle
		statusStyle = SessionStatusSelStyle
		status = statusStyle.Render(statusIcon)
		// Tree connector also gets selection styling
		treeStyle = TreeConnectorSelStyle
		// Rebuild baseIndent with selection styling for sub-sessions
		if item.IsSubSession && !item.ParentIsLastInGroup {
			groupIndent := strings.Repeat(treeEmpty, max(0, item.Level-2))
			baseIndent = groupIndent + " " + treeStyle.Render("│")
		}
	}

	title := titleStyle.Render(inst.Title)
	tool := toolStyle.Render(" " + instTool)

	// YOLO badge for Gemini sessions with YOLO mode enabled
	yoloBadge := ""
	if instTool == "gemini" && inst.GeminiYoloMode != nil && *inst.GeminiYoloMode {
		yoloStyle := lipgloss.NewStyle().Foreground(ColorYellow).Bold(true)
		if selected {
			yoloStyle = SessionStatusSelStyle
		}
		yoloBadge = yoloStyle.Render(" [YOLO]")
	}

	// Worktree branch badge for sessions running in git worktrees.
	worktreeBadge := ""
	if inst.IsWorktree() && inst.WorktreeBranch != "" {
		branch := inst.WorktreeBranch
		if len(branch) > 15 {
			branch = branch[:12] + "..."
		}
		wtStyle := lipgloss.NewStyle().Foreground(ColorCyan)
		if selected {
			wtStyle = SessionStatusSelStyle
		}
		worktreeBadge = wtStyle.Render(" [" + branch + "]")
	}

	// Sandbox badge for containerized sessions.
	sandboxBadge := ""
	if inst.IsSandboxed() {
		sbStyle := lipgloss.NewStyle().Foreground(ColorCyan)
		if selected {
			sbStyle = SessionStatusSelStyle
		}
		sandboxBadge = sbStyle.Render(" [sandbox]")
	}

	// Vagrant mode badge
	vagrantBadge := ""
	if inst.IsVagrantMode() {
		vStyle := lipgloss.NewStyle().Foreground(ColorBlue).Bold(true)
		if selected {
			vStyle = SessionStatusSelStyle
		}
		vagrantBadge = vStyle.Render(" [Vagrant]")
	}

	// Build row: [baseIndent][selection][tree][status] [title] [tool] [yolo] [worktree] [sandbox] [vagrant]
	// Format: " ├─ ● session-name tool" or "▶└─ ● session-name tool"
	// Sub-sessions get extra indent: "   ├─◐ sub-session tool"
	row := fmt.Sprintf(
		"%s%s%s %s %s%s%s%s%s%s",
		baseIndent,
		selectionPrefix,
		treeStyle.Render(treeConnector),
		status,
		title,
		tool,
		yoloBadge,
		worktreeBadge,
		sandboxBadge,
		vagrantBadge,
	)
	b.WriteString(row)
	b.WriteString("\n")
}

// renderSessionList renders the left panel with hierarchical session list
func (h *Home) renderSessionList(width, height int) string {
	var b strings.Builder

	if len(h.flatItems) == 0 {
		// Responsive empty state - adapts to available space
		// Account for border (2 chars each side) when calculating content area
		contentWidth := width - 4
		contentHeight := height - 2
		if contentWidth < 10 {
			contentWidth = 10
		}
		if contentHeight < 5 {
			contentHeight = 5
		}

		emptyContent := renderEmptyStateResponsive(EmptyStateConfig{
			Icon:     "⬡",
			Title:    "No Sessions Yet",
			Subtitle: "Get started by creating your first session",
			Hints: []string{
				"Press n to create a new session",
				"Press i to import existing tmux sessions",
				"Press g to create a group",
			},
		}, contentWidth, contentHeight)

		return lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Render(emptyContent)
	}

	// Render items starting from viewOffset
	visibleCount := 0
	maxVisible := height - 1 // Leave room for scrolling indicator
	if maxVisible < 1 {
		maxVisible = 1
	}

	// Show "more above" indicator if scrolled down
	if h.viewOffset > 0 {
		b.WriteString(DimStyle.Render(fmt.Sprintf("  ⋮ +%d above", h.viewOffset)))
		b.WriteString("\n")
		maxVisible-- // Account for the indicator line
	}

	for i := h.viewOffset; i < len(h.flatItems) && visibleCount < maxVisible; i++ {
		item := h.flatItems[i]
		h.renderItem(&b, item, i == h.cursor, i)
		visibleCount++
	}

	// Show "more below" indicator if there are more items
	remaining := len(h.flatItems) - (h.viewOffset + visibleCount)
	if remaining > 0 {
		b.WriteString(DimStyle.Render(fmt.Sprintf("  ⋮ +%d below", remaining)))
	}

	// Height padding is handled by ensureExactHeight() in View() for consistency
	return b.String()
}

// ────────────────────────────────────────────────────────────────────────────
// YOLO Progress Indicator Rendering
// ────────────────────────────────────────────────────────────────────────────

// Icons for YOLO progress and gate status rendering.
const (
	checkIcon       = "\u2713" // ✓
	gearIcon        = "\u2699" // ⚙
	arrowSep        = " \u2192 " // →
	gatePassedIcon  = "\u2713" // ✓
	gatePendingIcon = "\u23f3" // ⏳
	gateFailedIcon  = "\u2715" // ✕
	gateUnknownIcon = "?"
)

// columnAbbreviation maps kanban columns to 2-letter abbreviations.
var columnAbbreviation = map[session.KanbanColumn]string{
	session.KanbanBacklog:   "BL",
	session.KanbanDesign:    "DE",
	session.KanbanPlan:      "PL",
	session.KanbanImplement: "IM",
	session.KanbanReview:    "RE",
	session.KanbanDone:      "DO",
}

// renderYOLOProgress renders a column progress indicator for YOLO mode.
// Shows: BL ✓ → DE ✓ → PL ⚙ → IM → RE → DO
//   - checkIcon (green) for completed columns
//   - gearIcon (yellow) for current column
//   - (dim) for pending columns
//
// Standalone function for testability.
func renderYOLOProgress(currentColumn session.KanbanColumn) string {
	currentIdx := currentColumn.Index()

	greenStyle := lipgloss.NewStyle().Foreground(ColorGreen).Bold(true)
	yellowStyle := lipgloss.NewStyle().Foreground(ColorYellow).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(ColorTextDim)

	var parts []string
	for i, col := range kanbanColumnsOrdered {
		abbr := columnAbbreviation[col]
		switch {
		case currentIdx < 0:
			// Invalid column: render all as dim
			parts = append(parts, dimStyle.Render(abbr))
		case i < currentIdx:
			// Completed
			parts = append(parts, greenStyle.Render(checkIcon+" "+abbr))
		case i == currentIdx && col == session.KanbanDone:
			// Done column as current means everything is complete
			parts = append(parts, greenStyle.Render(checkIcon+" "+abbr))
		case i == currentIdx:
			// Current
			parts = append(parts, yellowStyle.Render(gearIcon+" "+abbr))
		default:
			// Pending
			parts = append(parts, dimStyle.Render(abbr))
		}
	}

	return strings.Join(parts, arrowSep)
}

// abbreviateModelName strips the "-preview" suffix from model names.
// For example: "gemini-3-pro-preview" becomes "gemini-3-pro".
func abbreviateModelName(name string) string {
	return strings.TrimSuffix(name, "-preview")
}

// renderGateStatus renders per-model gate status for YOLO mode.
// Shows: Gate: gemini ✓ gpt-5.2 ⏳ o3 ⏳
// Status values: "passed" -> ✓ (green), "pending" -> ⏳ (yellow), "failed" -> ✕ (red)
//
// Returns empty string if gateStatus is nil or empty.
// Standalone function for testability.
func renderGateStatus(gateStatus map[string]string) string {
	if len(gateStatus) == 0 {
		return ""
	}

	labelStyle := lipgloss.NewStyle().Foreground(ColorText)
	greenStyle := lipgloss.NewStyle().Foreground(ColorGreen).Bold(true)
	yellowStyle := lipgloss.NewStyle().Foreground(ColorYellow)
	redStyle := lipgloss.NewStyle().Foreground(ColorRed).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(ColorTextDim)

	var parts []string
	// Iterate in sorted order for deterministic output
	for _, model := range sortedKeys(gateStatus) {
		status := gateStatus[model]
		abbr := abbreviateModelName(model)

		var icon string
		switch status {
		case "passed", "pass":
			icon = greenStyle.Render(gatePassedIcon)
		case "pending":
			icon = yellowStyle.Render(gatePendingIcon)
		case "failed", "fail":
			icon = redStyle.Render(gateFailedIcon)
		default:
			icon = dimStyle.Render(gateUnknownIcon)
		}
		parts = append(parts, abbr+" "+icon)
	}

	return labelStyle.Render("Gate: ") + strings.Join(parts, " ")
}

// sortedKeys returns map keys in sorted order for deterministic rendering.
func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// Simple insertion sort for small maps (typically 2-4 models)
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return keys
}

// ────────────────────────────────────────────────────────────────────────────
// Kanban Board Card Rendering
// ────────────────────────────────────────────────────────────────────────────

// truncateTitle truncates a title to maxWidth runes with ellipsis.
// Uses rune-based truncation to handle multi-byte Unicode characters safely.
func truncateTitle(title string, maxWidth int) string {
	runes := []rune(title)
	if len(runes) <= maxWidth {
		return title
	}
	if maxWidth <= 3 {
		return string(runes[:maxWidth])
	}
	return string(runes[:maxWidth-3]) + "..."
}

// kanbanStatusIcon returns the icon and style for a given status.
// Used for compact kanban card rendering.
func kanbanStatusIcon(status session.Status) (icon string, style lipgloss.Style) {
	switch status {
	case session.StatusRunning:
		return "●", SessionStatusRunning
	case session.StatusWaiting:
		return "◐", SessionStatusWaiting
	case session.StatusIdle:
		return "○", SessionStatusIdle
	case session.StatusError:
		return "✕", SessionStatusError
	default:
		return "○", SessionStatusIdle
	}
}

// renderKanbanCard renders a bordered kanban card with 3-line height.
// Format (3 lines with border):
//   ╭──────────────╮
//   │ ● title  cl │
//   ╰──────────────╯
//
// Content line: [selection_icon][status_icon] [title] [tool_badge]
// - width: total card width in characters
// - selected: whether this card is currently selected
// Returns empty string if inst is nil.
func renderKanbanCard(inst *session.Instance, width int, selected bool) string {
	if inst == nil {
		return ""
	}

	// Get status icon and style
	instStatus := inst.GetStatusThreadSafe()
	instTool := inst.GetToolThreadSafe()
	icon, iconStyle := kanbanStatusIcon(instStatus)

	// Title style based on status and selection
	var titleStyle lipgloss.Style
	var toolStyle lipgloss.Style
	if selected {
		titleStyle = SessionTitleSelStyle
		iconStyle = SessionStatusSelStyle
		toolStyle = SessionStatusSelStyle
	} else {
		switch instStatus {
		case session.StatusRunning, session.StatusWaiting:
			titleStyle = SessionTitleActive
		case session.StatusError:
			titleStyle = SessionTitleError
		default:
			titleStyle = SessionTitleDefault
		}
		toolStyle = GetToolStyle(instTool)
	}

	// Tool badge abbreviation (2 letters)
	// claude -> cl, gemini -> ge, codex -> cx, aider -> ai, cursor -> cu, shell -> sh
	toolBadge := ""
	switch instTool {
	case "claude":
		toolBadge = "cl"
	case "gemini":
		toolBadge = "ge"
	case "codex":
		toolBadge = "cx"
	case "aider":
		toolBadge = "ai"
	case "cursor":
		toolBadge = "cu"
	case "shell":
		toolBadge = "sh"
	default:
		// Truncate to first 2 chars for unknown tools
		if len(instTool) >= 2 {
			toolBadge = instTool[:2]
		} else {
			toolBadge = instTool
		}
	}

	// Selection icon (for selected cards)
	selectionIcon := ""
	if selected {
		selectionIcon = "▶"
	}

	// YOLO icon for autonomous sessions
	yoloIcon := ""
	if inst.AutomationMode == session.AutomationYOLO {
		yoloIcon = "🤖"
	}

	// Calculate available width for title
	// Border takes 2 chars (left/right), padding takes 2 chars (left/right)
	// Content format: "[sel][yolo][icon] [title] [tool]"
	// Components: selection(1 or 0) + yolo(2 or 0) + icon(1) + space(1) + title(?) + space(1) + tool(2)
	borderAndPadding := 4 // 2 for border, 2 for padding
	selWidth := 0
	if selected {
		selWidth = 1
	}
	yoloWidth := 0
	if yoloIcon != "" {
		yoloWidth = 2 // emoji takes 2 visual chars
	}
	iconWidth := 1
	toolWidth := len(toolBadge)
	spaces := 2 // one before title, one before tool

	availWidth := width - borderAndPadding - selWidth - yoloWidth - iconWidth - spaces - toolWidth
	if availWidth < 3 {
		availWidth = 3
	}

	// Truncate title to fit
	title := truncateTitle(inst.Title, availWidth)

	// Render components
	styledSel := ""
	if selectionIcon != "" {
		styledSel = iconStyle.Render(selectionIcon)
	}
	styledYolo := ""
	if yoloIcon != "" {
		if selected {
			styledYolo = iconStyle.Render(yoloIcon)
		} else {
			styledYolo = lipgloss.NewStyle().Foreground(ColorYellow).Render(yoloIcon)
		}
	}
	styledIcon := iconStyle.Render(icon)
	styledTitle := titleStyle.Render(title)
	styledTool := toolStyle.Render(toolBadge)

	// Build content line: [sel][yolo][icon] [title] [tool]
	content := fmt.Sprintf("%s%s%s %s %s", styledSel, styledYolo, styledIcon, styledTitle, styledTool)

	// Create bordered card with RoundedBorder
	var cardStyle lipgloss.Style
	if selected {
		// Selected: accent color border + background highlight
		cardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorAccent).
			Background(ColorSurface).
			Width(width - 2). // -2 for border chars
			Padding(0, 1)
	} else {
		// Unselected: subtle border
		cardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Width(width - 2). // -2 for border chars
			Padding(0, 1)
	}

	return cardStyle.Render(content)
}
