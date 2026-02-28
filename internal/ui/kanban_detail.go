package ui

// kanban_detail.go — Preview pane and detail rendering extracted from home.go
//
// Contains: renderPreviewPane, renderLaunchingState, renderMcpLoadingState,
// renderForkingState, renderSessionInfoCard, renderToolStatusLine,
// renderDetectedAtLine, renderForkHintLine, renderSimpleMCPLine,
// stripControlCharsPreserveANSI, truncatePath, formatRelativeTime,
// copySessionOutput, sendOutputToSession, getSessionContent, maxTransferSize

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/mattn/go-runewidth"

	"github.com/asheshgoplani/agent-deck/internal/clipboard"
	"github.com/asheshgoplani/agent-deck/internal/session"
	"github.com/asheshgoplani/agent-deck/internal/tmux"
)

// --- Copy & Send Output helpers ---

const maxTransferSize = 500 * 1024 // 500KB max for inter-session transfer

// renderToolStatusLine renders a Status + Session line for a tool section.
// sessionID is the detected session ID (empty = not connected).
// detectedAt is when detection ran (zero = still detecting, used only when threeState is true).
// threeState enables the "Detecting..." intermediate state (for tools like OpenCode/Codex).
func renderToolStatusLine(b *strings.Builder, sessionID string, detectedAt time.Time, threeState bool) {
	labelStyle := lipgloss.NewStyle().Foreground(ColorText)
	valueStyle := lipgloss.NewStyle().Foreground(ColorText)

	if sessionID != "" {
		statusStyle := lipgloss.NewStyle().Foreground(ColorGreen).Bold(true)
		b.WriteString(labelStyle.Render("Status:  "))
		b.WriteString(statusStyle.Render("● Connected"))
		b.WriteString("\n")

		b.WriteString(labelStyle.Render("Session: "))
		b.WriteString(valueStyle.Render(sessionID))
		b.WriteString("\n")
	} else if threeState && detectedAt.IsZero() {
		statusStyle := lipgloss.NewStyle().Foreground(ColorYellow)
		b.WriteString(labelStyle.Render("Status:  "))
		b.WriteString(statusStyle.Render("◐ Detecting session..."))
		b.WriteString("\n")
	} else {
		statusStyle := lipgloss.NewStyle().Foreground(ColorText)
		b.WriteString(labelStyle.Render("Status:  "))
		if threeState {
			b.WriteString(statusStyle.Render("○ No session found"))
		} else {
			b.WriteString(statusStyle.Render("○ Not connected"))
		}
		b.WriteString("\n")
	}
}

// renderDetectedAtLine renders a "Detected: X ago" line.
func renderDetectedAtLine(b *strings.Builder, detectedAt time.Time) {
	if detectedAt.IsZero() {
		return
	}
	labelStyle := lipgloss.NewStyle().Foreground(ColorText)
	dimStyle := lipgloss.NewStyle().Foreground(ColorText).Italic(true)
	b.WriteString(labelStyle.Render("Detected:"))
	b.WriteString(dimStyle.Render(" " + formatRelativeTime(detectedAt)))
	b.WriteString("\n")
}

// renderForkHintLine renders the fork keyboard hint line.
func renderForkHintLine(b *strings.Builder) {
	hintStyle := lipgloss.NewStyle().Foreground(ColorText).Italic(true)
	keyStyle := lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
	b.WriteString(hintStyle.Render("Fork:    "))
	b.WriteString(keyStyle.Render("f"))
	b.WriteString(hintStyle.Render(" quick fork, "))
	b.WriteString(keyStyle.Render("F"))
	b.WriteString(hintStyle.Render(" fork with options"))
	b.WriteString("\n")
}

// renderSimpleMCPLine renders MCPs without sync status (for Gemini and other tools).
// Width-aware truncation shows "(+N more)" when MCPs don't fit.
func renderSimpleMCPLine(b *strings.Builder, mcpInfo *session.MCPInfo, width int) {
	if mcpInfo == nil || !mcpInfo.HasAny() {
		return
	}

	labelStyle := lipgloss.NewStyle().Foreground(ColorText)
	valueStyle := lipgloss.NewStyle().Foreground(ColorText)

	var mcpParts []string
	for _, name := range mcpInfo.Global {
		mcpParts = append(mcpParts, valueStyle.Render(name+" (g)"))
	}
	for _, name := range mcpInfo.Project {
		mcpParts = append(mcpParts, valueStyle.Render(name+" (p)"))
	}
	for _, mcp := range mcpInfo.LocalMCPs {
		mcpParts = append(mcpParts, valueStyle.Render(mcp.Name+" (l)"))
	}

	if len(mcpParts) == 0 {
		return
	}

	b.WriteString(labelStyle.Render("MCPs:    "))

	mcpMaxWidth := width - 4 - 9
	if mcpMaxWidth < 20 {
		mcpMaxWidth = 20
	}

	var mcpResult strings.Builder
	mcpCount := 0
	currentWidth := 0

	for i, part := range mcpParts {
		plainPart := tmux.StripANSI(part)
		partWidth := runewidth.StringWidth(plainPart)

		addedWidth := partWidth
		if mcpCount > 0 {
			addedWidth += 2
		}

		remaining := len(mcpParts) - i
		isLast := remaining == 1

		var wouldExceed bool
		if isLast {
			wouldExceed = currentWidth+addedWidth > mcpMaxWidth
		} else {
			moreIndicator := fmt.Sprintf(" (+%d more)", remaining)
			moreWidth := runewidth.StringWidth(moreIndicator)
			wouldExceed = currentWidth+addedWidth+moreWidth > mcpMaxWidth
		}

		if wouldExceed {
			moreStyle := lipgloss.NewStyle().Foreground(ColorText).Italic(true)
			if mcpCount > 0 {
				mcpResult.WriteString(moreStyle.Render(fmt.Sprintf(" (+%d more)", remaining)))
			} else {
				mcpResult.WriteString(moreStyle.Render(fmt.Sprintf("(%d MCPs)", len(mcpParts))))
			}
			break
		}

		if mcpCount > 0 {
			mcpResult.WriteString(", ")
		}
		mcpResult.WriteString(part)
		currentWidth += addedWidth
		mcpCount++
	}

	b.WriteString(mcpResult.String())
	b.WriteString("\n")
}

// renderPreviewPane renders the right panel with live preview
func (h *Home) renderPreviewPane(width, height int) string {
	var b strings.Builder

	if len(h.flatItems) == 0 || h.cursor >= len(h.flatItems) {
		// Show different message when there are no sessions vs just no selection
		if len(h.flatItems) == 0 {
			return renderEmptyStateResponsive(EmptyStateConfig{
				Icon:     "✦",
				Title:    "Ready to Go",
				Subtitle: "Your workspace is set up",
				Hints: []string{
					"Press n to create your first session",
					"Press i to import tmux sessions",
				},
			}, width, height)
		}
		return renderEmptyStateResponsive(EmptyStateConfig{
			Icon:     "◇",
			Title:    "No Selection",
			Subtitle: "Select a session to preview",
			Hints:    nil,
		}, width, height)
	}

	item := h.flatItems[h.cursor]

	// If group is selected, show group info
	if item.Type == session.ItemTypeGroup {
		return h.renderGroupPreview(item.Group, width, height)
	}

	// Session preview
	selected := item.Session

	// Session info header box
	statusIcon := "○"
	statusColor := ColorTextDim
	switch selected.Status {
	case session.StatusRunning:
		statusIcon = "●"
		statusColor = ColorGreen
	case session.StatusWaiting:
		statusIcon = "◐"
		statusColor = ColorYellow
	case session.StatusError:
		statusIcon = "✕"
		statusColor = ColorRed
	}

	// Header with session name and status
	statusBadge := lipgloss.NewStyle().Foreground(statusColor).Render(statusIcon + " " + string(selected.Status))
	nameStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorAccent)
	b.WriteString(nameStyle.Render(selected.Title))
	b.WriteString("  ")
	b.WriteString(statusBadge)
	b.WriteString("\n")

	// Info lines: path and activity time
	infoStyle := lipgloss.NewStyle().Foreground(ColorText)
	pathStr := truncatePath(selected.ProjectPath, width-4)
	b.WriteString(infoStyle.Render("📁 " + pathStr))
	b.WriteString("\n")

	// Activity time - shows when session was last active
	activityTime := selected.GetLastActivityTime()
	activityStr := formatRelativeTime(activityTime)
	if selected.Status == session.StatusRunning {
		activityStr = "active now"
	}
	b.WriteString(infoStyle.Render("⏱ " + activityStr))
	b.WriteString("\n")

	toolBadge := lipgloss.NewStyle().
		Foreground(ColorBg).
		Background(ColorPurple).
		Padding(0, 1).
		Render(selected.Tool)
	groupBadge := lipgloss.NewStyle().
		Foreground(ColorBg).
		Background(ColorCyan).
		Padding(0, 1).
		Render(selected.GroupPath)
	b.WriteString(toolBadge)
	b.WriteString(" ")
	b.WriteString(groupBadge)
	if selected.IsVagrantMode() {
		vagrantBadge := lipgloss.NewStyle().
			Foreground(ColorBg).
			Background(ColorBlue).
			Padding(0, 1).
			Render("Vagrant")
		b.WriteString(" ")
		b.WriteString(vagrantBadge)
	}
	b.WriteString("\n")

	// Worktree info section (for sessions running in git worktrees)
	if selected.IsWorktree() {
		wtHeader := renderSectionDivider("Worktree", width-4)
		b.WriteString(wtHeader)
		b.WriteString("\n")

		wtLabelStyle := lipgloss.NewStyle().Foreground(ColorText)
		wtBranchStyle := lipgloss.NewStyle().Foreground(ColorCyan).Bold(true)
		wtValueStyle := lipgloss.NewStyle().Foreground(ColorText)
		wtHintStyle := lipgloss.NewStyle().Foreground(ColorText).Italic(true)
		wtKeyStyle := lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)

		// Branch
		if selected.WorktreeBranch != "" {
			b.WriteString(wtLabelStyle.Render("Branch:  "))
			b.WriteString(wtBranchStyle.Render(selected.WorktreeBranch))
			b.WriteString("\n")
		}

		// Repo root (truncated)
		if selected.WorktreeRepoRoot != "" {
			repoPath := truncatePath(selected.WorktreeRepoRoot, width-4-9)
			b.WriteString(wtLabelStyle.Render("Repo:    "))
			b.WriteString(wtValueStyle.Render(repoPath))
			b.WriteString("\n")
		}

		// Worktree path (truncated)
		if selected.WorktreePath != "" {
			wtPath := truncatePath(selected.WorktreePath, width-4-9)
			b.WriteString(wtLabelStyle.Render("Path:    "))
			b.WriteString(wtValueStyle.Render(wtPath))
			b.WriteString("\n")
		}

		// Dirty status (lazy-cached, fetched via previewDebounce handler with 10s TTL)
		h.worktreeDirtyMu.Lock()
		isDirty, hasCached := h.worktreeDirtyCache[selected.ID]
		h.worktreeDirtyMu.Unlock()

		dirtyLabel := "checking..."
		dirtyStyle := wtValueStyle
		if hasCached {
			if isDirty {
				dirtyLabel = "dirty (uncommitted changes)"
				dirtyStyle = lipgloss.NewStyle().Foreground(ColorYellow)
			} else {
				dirtyLabel = "clean"
				dirtyStyle = lipgloss.NewStyle().Foreground(ColorGreen)
			}
		}
		b.WriteString(wtLabelStyle.Render("Status:  "))
		b.WriteString(dirtyStyle.Render(dirtyLabel))
		b.WriteString("\n")

		// Finish hint
		b.WriteString(wtHintStyle.Render("Finish:  "))
		b.WriteString(wtKeyStyle.Render("W"))
		b.WriteString(wtHintStyle.Render(" merge + cleanup"))
		b.WriteString("\n")
	}

	// Claude-specific info (session ID and MCPs)
	if selected.Tool == "claude" {
		// Section divider for Claude info
		claudeHeader := renderSectionDivider("Claude", width-4)
		b.WriteString(claudeHeader)
		b.WriteString("\n")

		labelStyle := lipgloss.NewStyle().Foreground(ColorText)
		valueStyle := lipgloss.NewStyle().Foreground(ColorText)

		// Status line
		if selected.ClaudeSessionID != "" {
			statusStyle := lipgloss.NewStyle().Foreground(ColorGreen).Bold(true)
			b.WriteString(labelStyle.Render("Status:  "))
			b.WriteString(statusStyle.Render("● Connected"))
			b.WriteString("\n")

			// Full session ID on its own line
			b.WriteString(labelStyle.Render("Session: "))
			b.WriteString(valueStyle.Render(selected.ClaudeSessionID))
			b.WriteString("\n")
		} else {
			statusStyle := lipgloss.NewStyle().Foreground(ColorText)
			b.WriteString(labelStyle.Render("Status:  "))
			b.WriteString(statusStyle.Render("○ Not connected"))
			b.WriteString("\n")
		}

		// MCP servers - compact format with source indicators and sync status
		mcpInfo := selected.GetMCPInfo()
		hasLoadedMCPs := len(selected.LoadedMCPNames) > 0
		hasMCPs := mcpInfo != nil && mcpInfo.HasAny()

		if hasMCPs || hasLoadedMCPs {
			b.WriteString(labelStyle.Render("MCPs:    "))

			// Build set of loaded MCPs for comparison
			loadedSet := make(map[string]bool)
			for _, name := range selected.LoadedMCPNames {
				loadedSet[name] = true
			}

			// Build set of current MCPs (from config)
			currentSet := make(map[string]bool)
			if mcpInfo != nil {
				for _, name := range mcpInfo.Global {
					currentSet[name] = true
				}
				for _, name := range mcpInfo.Project {
					currentSet[name] = true
				}
				for _, mcp := range mcpInfo.LocalMCPs {
					currentSet[mcp.Name] = true
				}
			}

			// Styles for different MCP states
			pendingStyle := lipgloss.NewStyle().Foreground(ColorYellow)
			staleStyle := lipgloss.NewStyle().Foreground(ColorText)

			var mcpParts []string

			// Helper to add MCP with appropriate styling
			addMCP := func(name, source string) {
				label := name + " (" + source + ")"
				if !hasLoadedMCPs {
					// Old session without LoadedMCPNames - show all as normal (no sync info)
					mcpParts = append(mcpParts, valueStyle.Render(label))
				} else if loadedSet[name] {
					// In both loaded and current - active (normal style)
					mcpParts = append(mcpParts, valueStyle.Render(label))
				} else {
					// In current but not loaded - pending (needs restart)
					mcpParts = append(mcpParts, pendingStyle.Render(label+" ⟳"))
				}
			}

			// Add MCPs from current config with source indicators
			if mcpInfo != nil {
				for _, name := range mcpInfo.Global {
					addMCP(name, "g")
				}
				for _, name := range mcpInfo.Project {
					addMCP(name, "p")
				}
				for _, mcp := range mcpInfo.LocalMCPs {
					// Show source path if different from project path
					sourceIndicator := "l"
					if mcp.SourcePath != selected.ProjectPath {
						// Show abbreviated path (just directory name)
						sourceIndicator = "l:" + filepath.Base(mcp.SourcePath)
					}
					addMCP(mcp.Name, sourceIndicator)
				}
			}

			// Add stale MCPs (loaded but no longer in config)
			if hasLoadedMCPs {
				for _, name := range selected.LoadedMCPNames {
					if !currentSet[name] {
						// Still running but removed from config
						mcpParts = append(mcpParts, staleStyle.Render(name+" ✕"))
					}
				}
			}

			// Calculate available width for MCPs (width - 4 for panel padding - 9 for "MCPs:    " label)
			mcpMaxWidth := width - 4 - 9
			if mcpMaxWidth < 20 {
				mcpMaxWidth = 20 // Minimum sensible width
			}

			// Build MCPs progressively to fit within available width
			var mcpResult strings.Builder
			mcpCount := 0
			currentWidth := 0

			for i, part := range mcpParts {
				// Strip ANSI codes to measure actual display width
				plainPart := tmux.StripANSI(part)
				partWidth := runewidth.StringWidth(plainPart)

				// Calculate width including separator if not first
				addedWidth := partWidth
				if mcpCount > 0 {
					addedWidth += 2 // ", " separator
				}

				remaining := len(mcpParts) - i
				isLast := remaining == 1

				// For non-last MCPs: reserve space for "+N more" indicator
				// For last MCP: just check if it fits without indicator
				var wouldExceed bool
				if isLast {
					// Last MCP - just check if it fits
					wouldExceed = currentWidth+addedWidth > mcpMaxWidth
				} else {
					// Not last - check with indicator space reserved
					moreIndicator := fmt.Sprintf(" (+%d more)", remaining)
					moreWidth := runewidth.StringWidth(moreIndicator)
					wouldExceed = currentWidth+addedWidth+moreWidth > mcpMaxWidth
				}

				if wouldExceed {
					// Would exceed - show indicator for remaining
					moreStyle := lipgloss.NewStyle().Foreground(ColorText).Italic(true)
					if mcpCount > 0 {
						mcpResult.WriteString(moreStyle.Render(fmt.Sprintf(" (+%d more)", remaining)))
					} else {
						// No MCPs fit - just show count
						mcpResult.WriteString(moreStyle.Render(fmt.Sprintf("(%d MCPs)", len(mcpParts))))
					}
					break
				}

				// Add separator if not first
				if mcpCount > 0 {
					mcpResult.WriteString(", ")
				}
				mcpResult.WriteString(part)
				currentWidth += addedWidth
				mcpCount++
			}

			b.WriteString(mcpResult.String())
			b.WriteString("\n")
		}

		// Fork hint when session can be forked
		if selected.CanFork() {
			hintStyle := lipgloss.NewStyle().Foreground(ColorText).Italic(true)
			keyStyle := lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
			b.WriteString(hintStyle.Render("Fork:    "))
			b.WriteString(keyStyle.Render("f"))
			b.WriteString(hintStyle.Render(" quick fork, "))
			b.WriteString(keyStyle.Render("F"))
			b.WriteString(hintStyle.Render(" fork with options"))
			b.WriteString("\n")
		}
	}

	// Gemini-specific info (session ID)
	if selected.Tool == "gemini" {
		geminiHeader := renderSectionDivider("Gemini", width-4)
		b.WriteString(geminiHeader)
		b.WriteString("\n")

		labelStyle := lipgloss.NewStyle().Foreground(ColorText)
		valueStyle := lipgloss.NewStyle().Foreground(ColorText)

		if selected.GeminiSessionID != "" {
			statusStyle := lipgloss.NewStyle().Foreground(ColorGreen).Bold(true)
			b.WriteString(labelStyle.Render("Status:  "))
			b.WriteString(statusStyle.Render("● Connected"))
			b.WriteString("\n")

			b.WriteString(labelStyle.Render("Session: "))
			b.WriteString(valueStyle.Render(selected.GeminiSessionID))
			b.WriteString("\n")

			// Display active model
			modelDisplay := "auto"
			if selected.GeminiModel != "" {
				modelDisplay = selected.GeminiModel
			}
			accentStyle := lipgloss.NewStyle().Foreground(ColorAccent)
			b.WriteString(labelStyle.Render("Model:   "))
			b.WriteString(accentStyle.Render(modelDisplay))
			b.WriteString("\n")

			// MCPs for Gemini (global only)
			mcpInfo := selected.GetMCPInfo()
			renderSimpleMCPLine(&b, mcpInfo, width)
		} else {
			statusStyle := lipgloss.NewStyle().Foreground(ColorText)
			b.WriteString(labelStyle.Render("Status:  "))
			b.WriteString(statusStyle.Render("○ Not connected"))
			b.WriteString("\n")
		}
	}

	// OpenCode-specific info (session ID)
	if selected.Tool == "opencode" {
		opencodeHeader := renderSectionDivider("OpenCode", width-4)
		b.WriteString(opencodeHeader)
		b.WriteString("\n")

		labelStyle := lipgloss.NewStyle().Foreground(ColorText)
		valueStyle := lipgloss.NewStyle().Foreground(ColorText)

		// Debug: log what value we're seeing
		uiLog.Debug(
			"opencode_rendering_preview",
			slog.String("title", selected.Title),
			slog.String("session_id", selected.OpenCodeSessionID),
		)

		if selected.OpenCodeSessionID != "" {
			statusStyle := lipgloss.NewStyle().Foreground(ColorGreen).Bold(true)
			b.WriteString(labelStyle.Render("Status:  "))
			b.WriteString(statusStyle.Render("● Connected"))
			b.WriteString("\n")

			b.WriteString(labelStyle.Render("Session: "))
			b.WriteString(valueStyle.Render(selected.OpenCodeSessionID))
			b.WriteString("\n")

			// Show when session was detected
			if !selected.OpenCodeDetectedAt.IsZero() {
				detectedAgo := formatRelativeTime(selected.OpenCodeDetectedAt)
				dimStyle := lipgloss.NewStyle().Foreground(ColorText).Italic(true)
				b.WriteString(labelStyle.Render("Detected:"))
				b.WriteString(dimStyle.Render(" " + detectedAgo))
				b.WriteString("\n")
			}

			// Fork hint for OpenCode
			if selected.CanFork() {
				renderForkHintLine(&b)
			}
		} else {
			// Check if detection has completed (OpenCodeDetectedAt is set even when no session found)
			if selected.OpenCodeDetectedAt.IsZero() {
				// Detection not yet completed - show detecting state
				statusStyle := lipgloss.NewStyle().Foreground(ColorYellow)
				b.WriteString(labelStyle.Render("Status:  "))
				b.WriteString(statusStyle.Render("◐ Detecting session..."))
				b.WriteString("\n")
			} else {
				// Detection completed but no session found
				statusStyle := lipgloss.NewStyle().Foreground(ColorText)
				b.WriteString(labelStyle.Render("Status:  "))
				b.WriteString(statusStyle.Render("○ No session found"))
				b.WriteString("\n")
			}
		}
	}

	// Codex-specific info (session ID, detection)
	if selected.Tool == "codex" {
		codexHeader := renderSectionDivider("Codex", width-4)
		b.WriteString(codexHeader)
		b.WriteString("\n")

		renderToolStatusLine(&b, selected.CodexSessionID, selected.CodexDetectedAt, true)
		if selected.CodexSessionID != "" {
			renderDetectedAtLine(&b, selected.CodexDetectedAt)
		}
	}

	// Custom tool info (tools defined in config.toml that aren't built-in)
	if selected.Tool != "claude" && selected.Tool != "gemini" && selected.Tool != "opencode" &&
		selected.Tool != "codex" {
		if toolDef := session.GetToolDef(selected.Tool); toolDef != nil {
			toolName := selected.Tool
			if toolDef.Icon != "" {
				toolName = toolDef.Icon + " " + toolName
			}
			customHeader := renderSectionDivider(toolName, width-4)
			b.WriteString(customHeader)
			b.WriteString("\n")

			labelStyle := lipgloss.NewStyle().Foreground(ColorText)

			genericID := selected.GetGenericSessionID()
			if genericID != "" {
				statusStyle := lipgloss.NewStyle().Foreground(ColorGreen).Bold(true)
				valueStyle := lipgloss.NewStyle().Foreground(ColorText)
				b.WriteString(labelStyle.Render("Status:  "))
				b.WriteString(statusStyle.Render("● Connected"))
				b.WriteString("\n")

				b.WriteString(labelStyle.Render("Session: "))
				b.WriteString(valueStyle.Render(genericID))
				b.WriteString("\n")
			} else {
				statusStyle := lipgloss.NewStyle().Foreground(ColorText)
				b.WriteString(labelStyle.Render("Status:  "))
				b.WriteString(statusStyle.Render("○ Not connected"))
				b.WriteString("\n")
			}

			// Resume hint when tool supports restart with session resume
			if selected.CanRestartGeneric() {
				hintStyle := lipgloss.NewStyle().Foreground(ColorText).Italic(true)
				keyStyle := lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
				b.WriteString(hintStyle.Render("Resume:  "))
				b.WriteString(keyStyle.Render("r"))
				b.WriteString(hintStyle.Render(" restart with session resume"))
				b.WriteString("\n")
			}
		}
	}

	b.WriteString("\n")

	// Special handling for error state - show guidance instead of output
	if selected.Status == session.StatusError {
		errorHeader := renderSectionDivider("Session Inactive", width-4)
		b.WriteString(errorHeader)
		b.WriteString("\n\n")

		// Warning icon and message
		warnStyle := lipgloss.NewStyle().Foreground(ColorYellow)
		dimStyle := lipgloss.NewStyle().Foreground(ColorText)
		keyStyle := lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)

		b.WriteString(warnStyle.Render("⚠ No tmux session running"))
		b.WriteString("\n\n")
		b.WriteString(dimStyle.Render("This can happen if:"))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("  • Session was added but not yet started"))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("  • tmux server was restarted"))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("  • Terminal was closed or system rebooted"))
		b.WriteString("\n\n")
		b.WriteString(dimStyle.Render("Actions:"))
		b.WriteString("\n")
		b.WriteString("  ")
		b.WriteString(keyStyle.Render("R"))
		b.WriteString(dimStyle.Render(" Start   - create and start tmux session"))
		b.WriteString("\n")
		b.WriteString("  ")
		b.WriteString(keyStyle.Render("d"))
		b.WriteString(dimStyle.Render(" Delete  - remove from list"))
		b.WriteString("\n")
		b.WriteString("  ")
		b.WriteString(keyStyle.Render("Enter"))
		b.WriteString(dimStyle.Render(" - attach (will auto-start)"))
		b.WriteString("\n")

		// Pad output to exact height to prevent layout shifts
		content := b.String()
		lines := strings.Split(content, "\n")
		lineCount := len(lines)

		if lineCount < height {
			for i := lineCount; i < height; i++ {
				content += "\n"
			}
		}

		if len(content) > 0 && content[len(content)-1] == '\n' {
			content = content[:len(content)-1]
		}

		return content
	}

	// Check preview settings for what to show
	config, _ := session.LoadUserConfig()
	showAnalytics := config != nil && config.GetShowAnalytics() &&
		(selected.Tool == "claude" || selected.Tool == "gemini")
	showOutput := config == nil || config.GetShowOutput() // Default to true if config fails

	// Apply preview mode override (v key cycles through modes)
	switch h.previewMode {
	case PreviewModeOutput:
		showAnalytics = false
		showOutput = true
	case PreviewModeAnalytics:
		// showAnalytics keeps its default value (only available for Claude/Gemini)
		showOutput = false
		// PreviewModeBoth: use config settings (default)
	}

	// Check if session is launching/resuming (for animation priority)
	_, isSessionLaunching := h.launchingSessions[selected.ID]
	_, isSessionResuming := h.resumingSessions[selected.ID]
	_, isSessionForking := h.forkingSessions[selected.ID]
	isStartingUp := isSessionLaunching || isSessionResuming || isSessionForking

	// Analytics panel (for Claude/Gemini sessions with analytics enabled)
	// Skip showing "Loading analytics..." during startup - let the launch animation take focus
	if showAnalytics && !isStartingUp {
		analyticsHeader := renderSectionDivider("Analytics", width-4)
		b.WriteString(analyticsHeader)
		b.WriteString("\n")

		// Check if we have analytics for this session
		if h.analyticsSessionID == selected.ID && (h.currentAnalytics != nil || h.currentGeminiAnalytics != nil) {
			// Pass display settings from config
			if config != nil {
				h.analyticsPanel.SetDisplaySettings(config.Preview.GetAnalyticsSettings())
			}
			h.analyticsPanel.SetSize(width-4, height/2)
			b.WriteString(h.analyticsPanel.View())
			b.WriteString("\n")
		} else {
			// Analytics not yet loaded
			loadingStyle := lipgloss.NewStyle().
				Foreground(ColorText).
				Italic(true)
			b.WriteString(loadingStyle.Render("Loading analytics..."))
			b.WriteString("\n\n")
		}
	}

	// If output is disabled AND not starting up, return early
	// (We want to show the launch animation even if output is normally disabled)
	if !showOutput && !isStartingUp {
		// If analytics was also not shown, display session info card as fallback
		if !showAnalytics {
			infoCard := h.renderSessionInfoCard(selected, width, height)
			b.WriteString("\n")
			b.WriteString(infoCard)
		}

		// Pad output to exact height to prevent layout shifts
		content := b.String()
		lines := strings.Split(content, "\n")
		lineCount := len(lines)
		if lineCount < height {
			for i := lineCount; i < height; i++ {
				content += "\n"
			}
		}
		if len(content) > 0 && content[len(content)-1] == '\n' {
			content = content[:len(content)-1]
		}
		return content
	}

	// Terminal output header
	termHeader := renderSectionDivider("Output", width-4)
	b.WriteString(termHeader)
	b.WriteString("\n")

	// Check if this session is launching (newly created), resuming (restarted), or forking
	launchTime, isLaunching := h.launchingSessions[selected.ID]
	resumeTime, isResuming := h.resumingSessions[selected.ID]
	mcpLoadTime, isMcpLoading := h.mcpLoadingSessions[selected.ID]
	forkTime, isForking := h.forkingSessions[selected.ID]

	// Determine if we should show animation (launch, resume, MCP loading, or forking)
	// For Claude: show for minimum 6 seconds, then check for ready indicators
	// For others: show for first 3 seconds after creation
	showLaunchingAnimation := false
	showMcpLoadingAnimation := false
	showForkingAnimation := isForking // Show forking animation immediately
	var animationStartTime time.Time
	if isLaunching {
		animationStartTime = launchTime
	} else if isResuming {
		animationStartTime = resumeTime
	} else if isMcpLoading {
		animationStartTime = mcpLoadTime
	}

	// Apply STATUS-BASED animation logic (matches hasActiveAnimation exactly)
	// Animation shows until session is ready, detected via status or content
	if isLaunching || isResuming || isMcpLoading {
		timeSinceStart := time.Since(animationStartTime)

		// Brief minimum to prevent flicker
		if timeSinceStart < launchAnimationMinDuration(selected.Tool) {
			if isMcpLoading {
				showMcpLoadingAnimation = true
			} else {
				showLaunchingAnimation = true
			}
		} else if timeSinceStart < 15*time.Second {
			// STATUS-BASED CHECK: Session ready when Running/Waiting/Idle
			sessionReady := selected.Status == session.StatusRunning ||
				selected.Status == session.StatusWaiting ||
				selected.Status == session.StatusIdle

			if !sessionReady {
				// Also check content for faster detection
				h.previewCacheMu.RLock()
				previewContent := h.previewCache[selected.ID]
				h.previewCacheMu.RUnlock()

				// Strip ANSI for reliable pattern matching
				plainPreview := ansi.Strip(previewContent)

				if selected.Tool == "claude" || selected.Tool == "gemini" {
					// Claude/Gemini ready indicators
					agentReady := strings.Contains(plainPreview, "ctrl+c to interrupt") ||
						strings.Contains(plainPreview, "No, and tell Claude what to do differently") ||
						strings.Contains(plainPreview, "\n> ") ||
						strings.Contains(plainPreview, "> \n") ||
						strings.Contains(plainPreview, "esc to interrupt") ||
						strings.Contains(plainPreview, "⠋") || strings.Contains(plainPreview, "⠙") ||
						strings.Contains(plainPreview, "Thinking") ||
						strings.Contains(plainPreview, "╭─")

					if selected.Tool == "gemini" {
						agentReady = agentReady ||
							strings.Contains(plainPreview, "▸") ||
							strings.Contains(plainPreview, "gemini>")
					}

					if !agentReady {
						if isMcpLoading {
							showMcpLoadingAnimation = true
						} else {
							showLaunchingAnimation = true
						}
					}
				} else {
					// Non-Claude/Gemini: ready if substantial content
					if len(strings.TrimSpace(plainPreview)) <= 50 {
						if isMcpLoading {
							showMcpLoadingAnimation = true
						} else {
							showLaunchingAnimation = true
						}
					}
				}
			}
		}
		// After 15 seconds, animation stops regardless
	}

	// Terminal preview - use cached content (async fetching keeps View() pure)
	h.previewCacheMu.RLock()
	preview, hasCached := h.previewCache[selected.ID]
	h.previewCacheMu.RUnlock()

	// Show forking animation when fork is in progress (highest priority)
	if showForkingAnimation {
		b.WriteString("\n")
		b.WriteString(h.renderForkingState(selected, width, forkTime))
	} else if showMcpLoadingAnimation {
		// Show MCP loading animation when reloading MCPs
		b.WriteString("\n")
		b.WriteString(h.renderMcpLoadingState(selected, width, mcpLoadTime))
	} else if showLaunchingAnimation {
		// Show launching animation for new sessions
		b.WriteString("\n")
		b.WriteString(h.renderLaunchingState(selected, width, animationStartTime))
	} else if !hasCached {
		// Show loading indicator while waiting for async fetch
		loadingStyle := lipgloss.NewStyle().
			Foreground(ColorText).
			Italic(true)
		b.WriteString(loadingStyle.Render("Loading preview..."))
	} else if preview == "" {
		emptyTerm := lipgloss.NewStyle().
			Foreground(ColorText).
			Italic(true).
			Render("(terminal is empty)")
		b.WriteString(emptyTerm)
	} else {
		// Calculate maxLines dynamically based on how many header lines we've already written
		// This accounts for Claude sessions having more header lines than other sessions
		currentContent := b.String()
		headerLines := strings.Count(currentContent, "\n") + 1 // +1 for the current line
		lines := strings.Split(preview, "\n")

		// Strip trailing empty lines BEFORE truncation
		// This ensures we show actual content, not empty trailing lines when space is limited
		// (Terminal output often ends with empty lines at cursor position)
		for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
			lines = lines[:len(lines)-1]
		}

		// If all lines were empty, show empty indicator
		if len(lines) == 0 {
			emptyTerm := lipgloss.NewStyle().
				Foreground(ColorText).
				Italic(true).
				Render("(terminal is empty)")
			b.WriteString(emptyTerm)
			return b.String()
		}

		maxLines := height - headerLines - 1 // -1 for potential truncation indicator
		if maxLines < 1 {
			maxLines = 1
		}

		// Track if we're truncating from the top (for indicator)
		truncatedFromTop := len(lines) > maxLines
		truncatedCount := 0
		if truncatedFromTop {
			// Reserve one line for the truncation indicator
			maxLines--
			if maxLines < 1 {
				maxLines = 1
			}
			truncatedCount = len(lines) - maxLines
			lines = lines[len(lines)-maxLines:]
		}

		maxWidth := width - 4
		if maxWidth < 10 {
			maxWidth = 10
		}

		// Show truncation indicator if content was cut from top
		if truncatedFromTop {
			truncIndicator := lipgloss.NewStyle().
				Foreground(ColorText).
				Italic(true).
				Render(fmt.Sprintf("⋮ %d more lines above", truncatedCount))
			b.WriteString(truncIndicator)
			b.WriteString("\n")
		}

		// Track consecutive empty lines to preserve some spacing
		consecutiveEmpty := 0
		const maxConsecutiveEmpty = 2 // Allow up to 2 consecutive empty lines

		for _, line := range lines {
			// Strip dangerous control characters (\r, \b, etc.) but preserve
			// ANSI escape sequences (ESC = 0x1b) so colors and formatting
			// from the captured terminal output pass through to display.
			safeLine := stripControlCharsPreserveANSI(line)

			// Check if visually empty (strip ANSI for this check)
			stripped := ansi.Strip(safeLine)
			trimmed := strings.TrimSpace(stripped)
			if trimmed == "" {
				consecutiveEmpty++
				if consecutiveEmpty <= maxConsecutiveEmpty {
					b.WriteString("\n") // Preserve empty line
				}
				continue
			}
			consecutiveEmpty = 0 // Reset counter on non-empty line

			// Truncate based on display width using ANSI-aware measurement
			// ansi.StringWidth ignores escape sequences for accurate width
			displayWidth := ansi.StringWidth(safeLine)
			if displayWidth > maxWidth {
				// ansi.Truncate preserves ANSI codes while truncating visible content
				safeLine = ansi.Truncate(safeLine, maxWidth-3, "...")
			}

			b.WriteString(safeLine)
			b.WriteString("\n")
		}
	}

	// CRITICAL: Enforce width constraint on ALL lines to prevent overflow into left panel
	// When lipgloss.JoinHorizontal combines panels, any line exceeding rightWidth
	// will wrap and corrupt the layout
	maxWidth := width - 2 // Small margin for safety
	if maxWidth < 20 {
		maxWidth = 20
	}

	result := b.String()
	lines := strings.Split(result, "\n")
	var truncatedLines []string
	for _, line := range lines {
		// Use ANSI-aware width measurement to handle lines with escape codes
		displayWidth := ansi.StringWidth(line)
		if displayWidth > maxWidth {
			// ANSI-aware truncation preserves escape codes while trimming visible content
			truncated := ansi.Truncate(line, maxWidth-3, "...")
			truncatedLines = append(truncatedLines, truncated)
		} else {
			truncatedLines = append(truncatedLines, line)
		}
	}

	return strings.Join(truncatedLines, "\n")
}

// renderLaunchingState renders the animated launching/resuming indicator for sessions
func (h *Home) renderLaunchingState(inst *session.Instance, width int, startTime time.Time) string {
	var b strings.Builder

	// Check if this is a resume operation (vs new launch)
	_, isResuming := h.resumingSessions[inst.ID]

	// Braille spinner frames - creates smooth rotation effect
	spinnerFrames := []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}
	spinner := spinnerFrames[h.animationFrame]

	// Tool-specific messaging with emoji
	var toolName, toolDesc, emoji string
	if isResuming {
		emoji = "🔄"
	} else {
		emoji = "🚀"
	}

	switch inst.Tool {
	case "claude":
		toolName = "Claude Code"
		if isResuming {
			toolDesc = "Resuming Claude session..."
		} else {
			toolDesc = "Starting Claude session..."
		}
	case "gemini":
		toolName = "Gemini"
		if isResuming {
			toolDesc = "Resuming Gemini session..."
		} else {
			toolDesc = "Connecting to Gemini..."
		}
	case "aider":
		toolName = "Aider"
		if isResuming {
			toolDesc = "Resuming Aider session..."
		} else {
			toolDesc = "Starting Aider..."
		}
	case "codex":
		toolName = "Codex"
		if isResuming {
			toolDesc = "Resuming Codex session..."
		} else {
			toolDesc = "Starting Codex..."
		}
	case "opencode":
		toolName = "OpenCode"
		if isResuming {
			toolDesc = "Resuming OpenCode session..."
		} else {
			toolDesc = "Starting OpenCode..."
		}
	default:
		toolName = "Shell"
		if isResuming {
			toolDesc = "Resuming shell session..."
		} else {
			toolDesc = "Launching shell session..."
		}
	}

	// Centered layout
	centerStyle := lipgloss.NewStyle().
		Width(width - 4).
		Align(lipgloss.Center)

	// Spinner with tool color
	spinnerStyle := lipgloss.NewStyle().
		Foreground(ColorAccent).
		Bold(true)
	spinnerLine := spinnerStyle.Render(spinner + "  " + spinner + "  " + spinner)
	b.WriteString(centerStyle.Render(spinnerLine))
	b.WriteString("\n\n")

	// Title with emoji
	titleStyle := lipgloss.NewStyle().
		Foreground(ColorPurple).
		Bold(true)
	var actionVerb string
	if isResuming {
		actionVerb = "Resuming"
	} else {
		actionVerb = "Launching"
	}
	b.WriteString(centerStyle.Render(titleStyle.Render(emoji + " " + actionVerb + " " + toolName)))
	b.WriteString("\n\n")

	// Description
	descStyle := lipgloss.NewStyle().
		Foreground(ColorText).
		Italic(true)
	b.WriteString(centerStyle.Render(descStyle.Render(toolDesc)))
	b.WriteString("\n\n")

	// Progress dots animation
	dotsCount := (h.animationFrame % 4) + 1
	dots := strings.Repeat("●", dotsCount) + strings.Repeat("○", 4-dotsCount)
	dotsStyle := lipgloss.NewStyle().
		Foreground(ColorAccent)
	b.WriteString(centerStyle.Render(dotsStyle.Render(dots)))
	b.WriteString("\n\n")

	// Elapsed time (consistent with MCP and Fork animations)
	elapsed := time.Since(startTime).Round(time.Second)
	timeStyle := lipgloss.NewStyle().
		Foreground(ColorYellow).
		Italic(true)
	b.WriteString(centerStyle.Render(timeStyle.Render(fmt.Sprintf("Loading... %s", elapsed))))

	return b.String()
}

// renderMcpLoadingState renders the MCP loading animation in the preview pane
func (h *Home) renderMcpLoadingState(inst *session.Instance, width int, startTime time.Time) string {
	var b strings.Builder

	// Braille spinner frames - creates smooth rotation effect
	spinnerFrames := []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}
	spinner := spinnerFrames[h.animationFrame]

	// Centered layout
	centerStyle := lipgloss.NewStyle().
		Width(width - 4).
		Align(lipgloss.Center)

	// Spinner with cyan color (MCP-themed)
	spinnerStyle := lipgloss.NewStyle().
		Foreground(ColorCyan).
		Bold(true)
	spinnerLine := spinnerStyle.Render(spinner + "  " + spinner + "  " + spinner)
	b.WriteString(centerStyle.Render(spinnerLine))
	b.WriteString("\n\n")

	// MCP loading title
	titleStyle := lipgloss.NewStyle().
		Foreground(ColorCyan).
		Bold(true)
	b.WriteString(centerStyle.Render(titleStyle.Render("🔌 Reloading MCPs")))
	b.WriteString("\n\n")

	// Description
	descStyle := lipgloss.NewStyle().
		Foreground(ColorText).
		Italic(true)
	b.WriteString(centerStyle.Render(descStyle.Render("Restarting session with updated MCP configuration...")))
	b.WriteString("\n\n")

	// Progress dots animation
	dotsCount := (h.animationFrame % 4) + 1
	dots := strings.Repeat("●", dotsCount) + strings.Repeat("○", 4-dotsCount)
	dotsStyle := lipgloss.NewStyle().
		Foreground(ColorCyan)
	b.WriteString(centerStyle.Render(dotsStyle.Render(dots)))
	b.WriteString("\n\n")

	// Elapsed time
	elapsed := time.Since(startTime).Round(time.Second)
	timeStyle := lipgloss.NewStyle().
		Foreground(ColorYellow).
		Italic(true)
	b.WriteString(centerStyle.Render(timeStyle.Render(fmt.Sprintf("Loading... %s", elapsed))))

	return b.String()
}

// renderForkingState renders the forking animation when session is being forked
func (h *Home) renderForkingState(inst *session.Instance, width int, startTime time.Time) string {
	var b strings.Builder

	// Centered layout
	centerStyle := lipgloss.NewStyle().
		Width(width - 4).
		Align(lipgloss.Center)

	// Braille spinner frames
	spinnerFrames := []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}
	spinner := spinnerFrames[h.animationFrame]

	// Spinner with purple color (fork-themed)
	spinnerStyle := lipgloss.NewStyle().
		Foreground(ColorPurple).
		Bold(true)
	spinnerLine := spinnerStyle.Render(spinner + "  " + spinner + "  " + spinner)
	b.WriteString(centerStyle.Render(spinnerLine))
	b.WriteString("\n\n")

	// Forking title
	titleStyle := lipgloss.NewStyle().
		Foreground(ColorPurple).
		Bold(true)
	b.WriteString(centerStyle.Render(titleStyle.Render("🔀 Forking Session")))
	b.WriteString("\n\n")

	// Description
	descStyle := lipgloss.NewStyle().
		Foreground(ColorText).
		Italic(true)
	b.WriteString(centerStyle.Render(descStyle.Render("Creating a new Claude session from this conversation...")))
	b.WriteString("\n\n")

	// Progress dots animation
	dotsCount := (h.animationFrame % 4) + 1
	dots := strings.Repeat("●", dotsCount) + strings.Repeat("○", 4-dotsCount)
	dotsStyle := lipgloss.NewStyle().
		Foreground(ColorPurple)
	b.WriteString(centerStyle.Render(dotsStyle.Render(dots)))
	b.WriteString("\n\n")

	// Elapsed time (consistent with other animations)
	elapsed := time.Since(startTime).Round(time.Second)
	timeStyle := lipgloss.NewStyle().
		Foreground(ColorYellow).
		Italic(true)
	b.WriteString(centerStyle.Render(timeStyle.Render(fmt.Sprintf("Loading... %s", elapsed))))

	return b.String()
}

// renderSessionInfoCard renders a simple session info card as fallback view
// Used when both show_output and show_analytics are disabled
func (h *Home) renderSessionInfoCard(inst *session.Instance, width, height int) string {
	if inst == nil {
		dimStyle := lipgloss.NewStyle().Foreground(ColorTextDim).Italic(true)
		return dimStyle.Render("No session selected")
	}

	var b strings.Builder

	// Snapshot status/tool under read lock for thread safety
	cardStatus := inst.GetStatusThreadSafe()
	cardTool := inst.GetToolThreadSafe()

	// Header with tool icon
	icon := ToolIcon(cardTool)
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorAccent).
		Render(fmt.Sprintf("%s %s", icon, inst.Title))
	b.WriteString(header)
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", max(0, min(width-4, 40))))
	b.WriteString("\n\n")

	labelStyle := lipgloss.NewStyle().Foreground(ColorTextDim)
	valueStyle := lipgloss.NewStyle().Foreground(ColorText)

	// Path
	b.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Path:"), valueStyle.Render(inst.ProjectPath)))

	// Status with color
	var statusColor lipgloss.Color
	switch cardStatus {
	case session.StatusRunning:
		statusColor = ColorGreen
	case session.StatusWaiting:
		statusColor = ColorYellow
	case session.StatusError:
		statusColor = ColorRed
	default:
		statusColor = ColorTextDim
	}
	statusStyle := lipgloss.NewStyle().Foreground(statusColor)
	b.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Status:"), statusStyle.Render(string(cardStatus))))

	// Tool
	b.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Tool:"), valueStyle.Render(cardTool)))

	// Session ID (if available) - Claude, Gemini, or OpenCode
	sessionID := inst.ClaudeSessionID
	if sessionID == "" {
		sessionID = inst.GeminiSessionID
	}
	if sessionID == "" {
		sessionID = inst.OpenCodeSessionID
	}
	if sessionID != "" {
		shortID := sessionID
		if len(shortID) > 12 {
			shortID = shortID[:12] + "..."
		}
		b.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Session:"), valueStyle.Render(shortID)))
	}

	// Created date
	b.WriteString(
		fmt.Sprintf("%s %s\n", labelStyle.Render("Created:"), valueStyle.Render(inst.CreatedAt.Format("Jan 2 15:04"))),
	)

	return b.String()
}

// stripControlCharsPreserveANSI removes dangerous C0 control characters while
// preserving ANSI escape sequences (ESC = 0x1b). This allows terminal colors
// and formatting from capture-pane -e output to pass through to display, while
// still stripping \r, \b, and other control chars that corrupt TUI layout.
func stripControlCharsPreserveANSI(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 && r != '\n' && r != '\t' && r != '\x1b' {
			return -1 // Drop the character
		}
		return r
	}, s)
}

// truncatePath shortens a path to fit within maxLen display width
func truncatePath(path string, maxLen int) string {
	pathWidth := runewidth.StringWidth(path)
	if pathWidth <= maxLen {
		return path
	}
	if maxLen < 10 {
		maxLen = 10
	}
	// Show beginning and end: /Users/.../project
	// Use rune-based slicing for proper Unicode handling
	runes := []rune(path)
	startLen := maxLen / 3
	endLen := maxLen*2/3 - 3
	if startLen+endLen+3 > len(runes) {
		// Path is short in runes but wide in display - use simple truncation
		return runewidth.Truncate(path, maxLen-3, "...")
	}
	return string(runes[:startLen]) + "..." + string(runes[len(runes)-endLen:])
}

// formatRelativeTime formats a time as a human-readable relative string
// Examples: "just now", "2m ago", "1h ago", "3h ago", "1d ago"
func formatRelativeTime(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}

	d := time.Since(t)

	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		mins := int(d.Minutes())
		if mins == 1 {
			return "1m ago"
		}
		return fmt.Sprintf("%dm ago", mins)
	case d < 24*time.Hour:
		hours := int(d.Hours())
		if hours == 1 {
			return "1h ago"
		}
		return fmt.Sprintf("%dh ago", hours)
	default:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1d ago"
		}
		return fmt.Sprintf("%dd ago", days)
	}
}

// copySessionOutput returns a tea.Cmd that copies the session's last response to clipboard.
func (h *Home) copySessionOutput(inst *session.Instance) tea.Cmd {
	return func() tea.Msg {
		content, err := getSessionContent(inst)
		if err != nil {
			return copyResultMsg{err: err}
		}

		termInfo := tmux.GetTerminalInfo()
		result, err := clipboard.Copy(content, termInfo.SupportsOSC52)
		if err != nil {
			return copyResultMsg{err: fmt.Errorf("clipboard: %w", err)}
		}
		return copyResultMsg{
			sessionTitle: inst.Title,
			lineCount:    result.LineCount,
		}
	}
}

// sendOutputToSession returns a tea.Cmd that sends the source session's output to the target.
func (h *Home) sendOutputToSession(source, target *session.Instance) tea.Cmd {
	return func() tea.Msg {
		content, err := getSessionContent(source)
		if err != nil {
			return sendOutputResultMsg{
				targetTitle: target.Title,
				err:         err,
			}
		}

		// Truncate if too large
		if len(content) > maxTransferSize {
			content = content[:maxTransferSize] + "\n[Truncated at 500KB]"
		}

		// Wrap with header/footer
		wrapped := fmt.Sprintf("--- Output from [%s] ---\n%s\n--- End output from [%s] ---\n",
			source.Title, content, source.Title)

		tmuxSession := target.GetTmuxSession()
		if tmuxSession == nil {
			return sendOutputResultMsg{
				targetTitle: target.Title,
				err:         fmt.Errorf("target session has no tmux pane"),
			}
		}

		if err := tmuxSession.SendKeysChunked(wrapped); err != nil {
			return sendOutputResultMsg{
				targetTitle: target.Title,
				err:         fmt.Errorf("send failed: %w", err),
			}
		}

		lineCount := strings.Count(content, "\n")
		return sendOutputResultMsg{
			sourceTitle: source.Title,
			targetTitle: target.Title,
			lineCount:   lineCount,
		}
	}
}

// getSessionContent retrieves displayable content from a session.
// Tries GetLastResponse first, falls back to CaptureFullHistory.
func getSessionContent(inst *session.Instance) (string, error) {
	// Try AI response first
	resp, err := inst.GetLastResponse()
	if err == nil && resp.Content != "" {
		return resp.Content, nil
	}

	// Fall back to tmux pane capture
	tmuxSession := inst.GetTmuxSession()
	if tmuxSession == nil {
		return "", fmt.Errorf("no output available for this session")
	}

	content, err := tmuxSession.CaptureFullHistory()
	if err != nil {
		return "", fmt.Errorf("failed to capture output: %w", err)
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return "", fmt.Errorf("no output available for this session")
	}

	return content, nil
}

// --- Kanban Detail Panel (Task 4.1) ---

// KanbanDetailState represents the state for rendering the kanban detail panel.
// This is a value type for rendering, not a Bubble Tea component.
type KanbanDetailState struct {
	Instance     *session.Instance // Current session to display (nil = nothing selected)
	Visible      bool              // Whether the detail panel is shown
	Editing      bool              // Whether in edit mode (task 4.2 will implement this)
	FocusedField int               // Which field is focused in edit mode
	Width        int               // Available width for rendering
	Height       int               // Available height (40% of terminal)
}

// renderKanbanDetail renders the kanban detail panel.
// Returns empty string if not visible or no instance selected.
func renderKanbanDetail(state KanbanDetailState) string {
	if !state.Visible || state.Instance == nil {
		return ""
	}

	inst := state.Instance
	var lines []string

	// Calculate max width for content (account for border + padding)
	// Border = 2 chars per side, Padding = 1 char per side = 6 total
	maxContentWidth := state.Width - 6
	if maxContentWidth < 20 {
		maxContentWidth = 20
	}

	// Field index counter for edit mode
	fieldIdx := 0

	// Helper to add a field with optional edit marker, truncating to fit
	addField := func(label, value string, editable bool) {
		marker := ""
		focusIndicator := ""

		if editable {
			if state.Editing {
				// In edit mode: show focus indicator on the focused field
				if fieldIdx == state.FocusedField {
					focusIndicator = " ▶ "
				} else {
					focusIndicator = "   "
				}
			} else {
				// Not in edit mode: show [e] marker
				marker = " [e]"
			}
			fieldIdx++
		}

		line := fmt.Sprintf("%s%s:%s %s", focusIndicator, label, marker, value)

		// Truncate line if it exceeds max width
		if runewidth.StringWidth(line) > maxContentWidth {
			line = truncateToWidth(line, maxContentWidth)
		}
		lines = append(lines, line)
	}

	// Editable fields (0-5 in order)
	addField("Title", inst.Title, true)

	addField("Description", inst.Description, true)

	addField("Acceptance Criteria", inst.AcceptCriteria, true)

	column := ""
	if inst.KanbanColumn != nil {
		column = inst.KanbanColumn.String()
	}
	addField("Column", column, true)

	autoMode := "disabled"
	if inst.AutomationMode != "" {
		autoMode = string(inst.AutomationMode)
	}
	addField("Automation", autoMode, true)

	yoloMode := "disabled"
	if inst.AutomationMode == session.AutomationYOLO {
		yoloMode = "yolo"
	}
	addField("YOLO", yoloMode, true)

	// Read-only fields (no markers)
	addField("Status", string(inst.Status), false)

	tool := inst.Tool
	if tool == "" {
		tool = "shell"
	}
	addField("Tool", tool, false)

	worktree := inst.WorktreePath
	if worktree == "" {
		worktree = "(none)"
	}
	addField("Worktree Path", worktree, false)

	branch := inst.WorktreeBranch
	if branch == "" {
		branch = "(none)"
	}
	addField("Worktree Branch", branch, false)

	addField("Project Path", inst.ProjectPath, false)

	model := inst.GeminiModel
	if model == "" {
		model = "(default)"
	}
	addField("Model", model, false)

	mcpList := "(none)"
	if len(inst.LoadedMCPNames) > 0 {
		mcpList = strings.Join(inst.LoadedMCPNames, ", ")
	}
	addField("MCP Servers", mcpList, false)

	prompt := inst.LatestPrompt
	if prompt == "" {
		prompt = "(none)"
	}
	addField("Last Prompt", prompt, false)

	// Add edit mode help footer
	if state.Editing {
		lines = append(lines, "")
		helpStyle := lipgloss.NewStyle().Foreground(ColorTextDim).Italic(true)
		helpLine := helpStyle.Render("Tab: Next field | Shift+Tab: Prev | Esc: Cancel | Enter: Save")
		lines = append(lines, helpLine)
	}

	// Render content within panel
	content := strings.Join(lines, "\n")

	// Apply border and constraints
	borderColor := lipgloss.Color("12")
	if state.Editing {
		// Highlight border in edit mode
		borderColor = ColorAccent
	}

	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(1).
		MaxWidth(state.Width).
		Height(state.Height - 2)

	return panelStyle.Render(content)
}

// calculateDetailHeight returns 40% of terminal height, minimum 10 lines.
func calculateDetailHeight(termHeight int) int {
	height := termHeight * 40 / 100
	if height < 10 {
		return 10
	}
	return height
}

// truncateToWidth truncates a string to fit within maxWidth display width.
func truncateToWidth(s string, maxWidth int) string {
	if maxWidth <= 3 {
		return "..."
	}
	w := runewidth.StringWidth(s)
	if w <= maxWidth {
		return s
	}

	// Truncate and add ellipsis
	var result []rune
	currentWidth := 0
	for _, r := range s {
		rw := runewidth.RuneWidth(r)
		if currentWidth+rw+3 > maxWidth { // +3 for "..."
			break
		}
		result = append(result, r)
		currentWidth += rw
	}
	return string(result) + "..."
}

// --- Edit Mode Functions (Task 4.2) ---

// EnterEditMode enters edit mode for the detail panel.
// Only works when detail is visible and has an instance.
// Returns a new KanbanDetailState with Editing=true and FocusedField=0.
func EnterEditMode(state KanbanDetailState) KanbanDetailState {
	if !state.Visible || state.Instance == nil {
		return state
	}

	return KanbanDetailState{
		Instance:     state.Instance,
		Visible:      state.Visible,
		Editing:      true,
		FocusedField: 0,
		Width:        state.Width,
		Height:       state.Height,
	}
}

// ExitEditMode exits edit mode, discarding any unsaved changes.
// Returns a new KanbanDetailState with Editing=false.
func ExitEditMode(state KanbanDetailState) KanbanDetailState {
	return KanbanDetailState{
		Instance:     state.Instance,
		Visible:      state.Visible,
		Editing:      false,
		FocusedField: state.FocusedField,
		Width:        state.Width,
		Height:       state.Height,
	}
}

// NextEditField moves focus to the next editable field (Tab).
// Editable fields: Title(0), Description(1), AcceptCriteria(2), Column(3), AutoTrigger(4), YOLO(5).
// Wraps from field 5 to 0.
func NextEditField(state KanbanDetailState) KanbanDetailState {
	if !state.Editing {
		return state
	}

	nextField := (state.FocusedField + 1) % 6

	return KanbanDetailState{
		Instance:     state.Instance,
		Visible:      state.Visible,
		Editing:      state.Editing,
		FocusedField: nextField,
		Width:        state.Width,
		Height:       state.Height,
	}
}

// PrevEditField moves focus to the previous editable field (Shift+Tab).
// Wraps from field 0 to 5.
func PrevEditField(state KanbanDetailState) KanbanDetailState {
	if !state.Editing {
		return state
	}

	prevField := state.FocusedField - 1
	if prevField < 0 {
		prevField = 5
	}

	return KanbanDetailState{
		Instance:     state.Instance,
		Visible:      state.Visible,
		Editing:      state.Editing,
		FocusedField: prevField,
		Width:        state.Width,
		Height:       state.Height,
	}
}
