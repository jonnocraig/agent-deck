package ui

// kanban_sidebar.go — Group rendering and sidebar helpers extracted from home.go
//
// Contains: renderGroupItem, renderGroupPreview, groupWorktreeBranch,
// groupWorktreeInfo, getGroupWorktreeInfo, renderSectionDivider

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"

	"github.com/asheshgoplani/agent-deck/internal/session"
	"github.com/asheshgoplani/agent-deck/internal/tmux"
)

// renderSectionDivider creates a modern section divider with optional centered label
// Format: ─────────── Label ─────────── (lines extend to fill width)
func renderSectionDivider(label string, width int) string {
	lineStyle := lipgloss.NewStyle().Foreground(ColorBorder)

	if label == "" {
		return lineStyle.Render(strings.Repeat("─", max(0, width)))
	}

	// Label with subtle background for better visibility
	labelStyle := lipgloss.NewStyle().
		Foreground(ColorText).
		Bold(true)

	// Calculate side widths
	labelWidth := len(label) + 2 // +2 for spacing on each side of label
	sideWidth := (width - labelWidth) / 2
	if sideWidth < 3 {
		sideWidth = 3
	}

	return lineStyle.Render(strings.Repeat("─", sideWidth)) +
		" " + labelStyle.Render(label) + " " +
		lineStyle.Render(strings.Repeat("─", sideWidth))
}

// renderGroupItem renders a group header
// PERFORMANCE: Uses cached styles from styles.go to avoid allocations
func (h *Home) renderGroupItem(b *strings.Builder, item session.Item, selected bool, itemIndex int) {
	group := item.Group

	// Calculate indentation based on nesting level (no tree lines, just spaces)
	// Uses spacingNormal (2 chars) per level for consistent hierarchy visualization
	indent := strings.Repeat(strings.Repeat(" ", spacingNormal), max(0, item.Level))

	// Expand/collapse indicator with filled triangles (using cached styles)
	var expandIcon string
	if selected {
		if group.Expanded {
			expandIcon = GroupExpandSelStyle.Render("▾")
		} else {
			expandIcon = GroupExpandSelStyle.Render("▸")
		}
	} else {
		if group.Expanded {
			expandIcon = GroupExpandStyle.Render("▾") // Filled triangle for expanded
		} else {
			expandIcon = GroupExpandStyle.Render("▸") // Filled triangle for collapsed
		}
	}

	// Hotkey indicator (subtle, only for root groups, hidden when selected)
	// Uses pre-computed RootGroupNum from rebuildFlatItems() - O(1) lookup instead of O(n) loop
	hotkeyStr := ""
	if item.Level == 0 && !selected {
		if item.RootGroupNum >= 1 && item.RootGroupNum <= 9 {
			hotkeyStr = GroupHotkeyStyle.Render(fmt.Sprintf("%d·", item.RootGroupNum))
		}
	}

	// Select appropriate cached styles based on selection state
	nameStyle := GroupNameStyle
	countStyle := GroupCountStyle
	if selected {
		nameStyle = GroupNameSelStyle
		countStyle = GroupCountSelStyle
	}

	// Use recursive count to include sessions in subgroups (Issue #48)
	sessionCount := h.groupTree.SessionCountForGroup(group.Path)
	countStr := countStyle.Render(fmt.Sprintf(" (%d)", sessionCount))

	// Status indicators (compact, on same line) using cached styles
	// Also count recursively for subgroups
	running := 0
	waiting := 0
	for path, g := range h.groupTree.Groups {
		if path == group.Path || strings.HasPrefix(path, group.Path+"/") {
			for _, sess := range g.Sessions {
				switch sess.Status {
				case session.StatusRunning:
					running++
				case session.StatusWaiting:
					waiting++
				}
			}
		}
	}

	statusStr := ""
	if running > 0 {
		statusStr += " " + GroupStatusRunning.Render(fmt.Sprintf("● %d", running))
	}
	if waiting > 0 {
		statusStr += " " + GroupStatusWaiting.Render(fmt.Sprintf("◐ %d", waiting))
	}

	// Build the row: [indent][hotkey][expand] [name](count) [status]
	row := fmt.Sprintf(
		"%s%s%s %s%s%s",
		indent,
		hotkeyStr,
		expandIcon,
		nameStyle.Render(group.Name),
		countStr,
		statusStr,
	)
	b.WriteString(row)
	b.WriteString("\n")
}

// renderGroupPreview renders the preview pane for a group
func (h *Home) renderGroupPreview(group *session.Group, width, height int) string {
	var b strings.Builder

	// Group header with folder icon
	headerStyle := lipgloss.NewStyle().
		Foreground(ColorCyan).
		Bold(true)
	b.WriteString(headerStyle.Render("📁 " + group.Name))
	b.WriteString("\n\n")

	// Session count
	countStyle := lipgloss.NewStyle().
		Foreground(ColorText).
		Bold(true)
	b.WriteString(countStyle.Render(fmt.Sprintf("%d sessions", len(group.Sessions))))
	b.WriteString("\n\n")

	// Status breakdown with inline badges
	running, waiting, idle, errored := 0, 0, 0, 0
	for _, sess := range group.Sessions {
		switch sess.Status {
		case session.StatusRunning:
			running++
		case session.StatusWaiting:
			waiting++
		case session.StatusIdle:
			idle++
		case session.StatusError:
			errored++
		}
	}

	// Compact status line (inline, not badges)
	var statuses []string
	if running > 0 {
		statuses = append(
			statuses,
			lipgloss.NewStyle().Foreground(ColorGreen).Render(fmt.Sprintf("● %d running", running)),
		)
	}
	if waiting > 0 {
		statuses = append(
			statuses,
			lipgloss.NewStyle().Foreground(ColorYellow).Render(fmt.Sprintf("◐ %d waiting", waiting)),
		)
	}
	if idle > 0 {
		statuses = append(statuses, lipgloss.NewStyle().Foreground(ColorText).Render(fmt.Sprintf("○ %d idle", idle)))
	}
	if errored > 0 {
		statuses = append(statuses, lipgloss.NewStyle().Foreground(ColorRed).Render(fmt.Sprintf("✕ %d error", errored)))
	}

	if len(statuses) > 0 {
		b.WriteString(strings.Join(statuses, "  "))
		b.WriteString("\n\n")
	}

	// Repository worktree summary (when all sessions share the same repo root)
	if repoInfo := h.getGroupWorktreeInfo(group); repoInfo != nil {
		b.WriteString(renderSectionDivider("Repository", width-4))
		b.WriteString("\n")

		repoLabelStyle := lipgloss.NewStyle().Foreground(ColorText)
		repoValueStyle := lipgloss.NewStyle().Foreground(ColorText)
		repoBranchStyle := lipgloss.NewStyle().Foreground(ColorCyan).Bold(true)

		b.WriteString(repoLabelStyle.Render("Repo:       "))
		b.WriteString(repoValueStyle.Render(truncatePath(repoInfo.repoRoot, width-4-12)))
		b.WriteString("\n")

		b.WriteString(repoLabelStyle.Render("Worktrees:  "))
		b.WriteString(repoValueStyle.Render(fmt.Sprintf("%d active", len(repoInfo.branches))))
		b.WriteString("\n")

		for _, br := range repoInfo.branches {
			dirtyMark := ""
			if br.dirtyChecked {
				if br.isDirty {
					dirtyMark = lipgloss.NewStyle().Foreground(ColorYellow).Render(" (dirty)")
				} else {
					dirtyMark = lipgloss.NewStyle().Foreground(ColorGreen).Render(" (clean)")
				}
			}
			b.WriteString("  ")
			b.WriteString(repoBranchStyle.Render("• " + br.branch))
			b.WriteString(dirtyMark)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	// Sessions divider
	b.WriteString(renderSectionDivider("Sessions", width-4))
	b.WriteString("\n")

	// Session list (compact)
	if len(group.Sessions) == 0 {
		emptyStyle := lipgloss.NewStyle().Foreground(ColorText).Italic(true)
		b.WriteString(emptyStyle.Render("  No sessions in this group"))
		b.WriteString("\n")
	} else {
		maxShow := height - 12
		if maxShow < 3 {
			maxShow = 3
		}
		for i, sess := range group.Sessions {
			if i >= maxShow {
				remaining := len(group.Sessions) - i
				b.WriteString(DimStyle.Render(fmt.Sprintf("  ... +%d more", remaining)))
				break
			}

			// Status icon
			statusIcon := "○"
			statusColor := ColorTextDim
			switch sess.Status {
			case session.StatusRunning:
				statusIcon, statusColor = "●", ColorGreen
			case session.StatusWaiting:
				statusIcon, statusColor = "◐", ColorYellow
			case session.StatusError:
				statusIcon, statusColor = "✕", ColorRed
			}
			status := lipgloss.NewStyle().Foreground(statusColor).Render(statusIcon)
			name := lipgloss.NewStyle().Foreground(ColorText).Render(sess.Title)
			tool := lipgloss.NewStyle().Foreground(ColorPurple).Faint(true).Render(sess.Tool)

			b.WriteString(fmt.Sprintf("  %s %s %s\n", status, name, tool))
		}
	}

	// Keyboard hints at bottom
	b.WriteString("\n")
	hintStyle := lipgloss.NewStyle().Foreground(ColorComment).Italic(true)
	b.WriteString(hintStyle.Render("Tab toggle • R rename • d delete • g subgroup"))

	// CRITICAL: Enforce width constraint on ALL lines to prevent overflow into left panel
	maxWidth := max(width-2, 20)

	result := b.String()
	lines := strings.Split(result, "\n")
	var truncatedLines []string
	for _, line := range lines {
		cleanLine := tmux.StripANSI(line)
		displayWidth := runewidth.StringWidth(cleanLine)
		if displayWidth > maxWidth {
			truncated := runewidth.Truncate(cleanLine, maxWidth-3, "...")
			truncatedLines = append(truncatedLines, truncated)
		} else {
			truncatedLines = append(truncatedLines, line)
		}
	}

	return strings.Join(truncatedLines, "\n")
}

// groupWorktreeBranch holds info about a single worktree branch in a group
type groupWorktreeBranch struct {
	branch       string
	isDirty      bool
	dirtyChecked bool
}

// groupWorktreeInfo holds aggregated worktree info for a group sharing a common repo
type groupWorktreeInfo struct {
	repoRoot string
	branches []groupWorktreeBranch
}

// getGroupWorktreeInfo returns worktree summary if all sessions in the group
// share the same repo root and at least one is a worktree. Returns nil otherwise.
func (h *Home) getGroupWorktreeInfo(group *session.Group) *groupWorktreeInfo {
	if len(group.Sessions) < 2 {
		return nil
	}

	// Check if all sessions share a common repo root and count worktrees
	var commonRepo string
	var branches []groupWorktreeBranch
	for _, sess := range group.Sessions {
		if !sess.IsWorktree() {
			continue
		}
		if commonRepo == "" {
			commonRepo = sess.WorktreeRepoRoot
		} else if sess.WorktreeRepoRoot != commonRepo {
			return nil // Different repos, skip
		}

		// Get dirty status from cache
		h.worktreeDirtyMu.Lock()
		isDirty, hasCached := h.worktreeDirtyCache[sess.ID]
		h.worktreeDirtyMu.Unlock()

		branches = append(branches, groupWorktreeBranch{
			branch:       sess.WorktreeBranch,
			isDirty:      isDirty,
			dirtyChecked: hasCached,
		})
	}

	if len(branches) == 0 {
		return nil
	}

	return &groupWorktreeInfo{
		repoRoot: commonRepo,
		branches: branches,
	}
}
