package ui

import (
	"fmt"

	"github.com/asheshgoplani/agent-deck/internal/session"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ClaudeOptionsPanel is a UI panel for Claude-specific launch options
// Used in both ForkDialog and NewDialog
type ClaudeOptionsPanel struct {
	// Session mode: 0=new, 1=continue, 2=resume
	sessionMode int
	// Resume session ID input (only for mode=resume)
	resumeIDInput textinput.Model
	// Checkbox states
	skipPermissions      bool
	allowSkipPermissions bool
	useChrome            bool
	useTeammateMode      bool
	useVagrantMode       bool
	// Focus tracking
	focusIndex int
	focusCount int
	// Whether this panel is for fork dialog (fewer options)
	isForkMode bool
}

// NewClaudeOptionsPanel creates a new panel for NewDialog
func NewClaudeOptionsPanel() *ClaudeOptionsPanel {
	resumeInput := textinput.New()
	resumeInput.Placeholder = "session_id..."
	resumeInput.CharLimit = 64
	resumeInput.Width = 30

	return &ClaudeOptionsPanel{
		sessionMode:   0, // new
		resumeIDInput: resumeInput,
		isForkMode:    false,
	}
}

// NewClaudeOptionsPanelForFork creates a panel for ForkDialog (fewer options)
func NewClaudeOptionsPanelForFork() *ClaudeOptionsPanel {
	return &ClaudeOptionsPanel{
		sessionMode:   0,
		resumeIDInput: textinput.New(), // Not used in fork mode
		isForkMode:    true,
	}
}

// SetDefaults applies default values from config
func (p *ClaudeOptionsPanel) SetDefaults(config *session.UserConfig) {
	if config != nil {
		p.skipPermissions = config.Claude.GetDangerousMode()
		p.allowSkipPermissions = config.Claude.AllowDangerousMode
	}
}

// SetFromOptions applies persisted ClaudeOptions to the panel fields.
func (p *ClaudeOptionsPanel) SetFromOptions(opts *session.ClaudeOptions) {
	if opts == nil {
		return
	}
	switch opts.SessionMode {
	case "continue":
		p.sessionMode = 1
	case "resume":
		p.sessionMode = 2
		p.resumeIDInput.SetValue(opts.ResumeSessionID)
	default:
		p.sessionMode = 0
	}
	p.skipPermissions = opts.SkipPermissions
	p.allowSkipPermissions = opts.AllowSkipPermissions
	p.useChrome = opts.UseChrome
	p.useTeammateMode = opts.UseTeammateMode
	p.updateInputFocus()
	p.focusCount = p.getFocusCount()
}

// Focus sets focus to this panel
func (p *ClaudeOptionsPanel) Focus() {
	p.focusIndex = 0
	p.updateInputFocus()
}

// Blur removes focus from this panel
func (p *ClaudeOptionsPanel) Blur() {
	p.focusIndex = -1
	p.resumeIDInput.Blur()
}

// IsFocused returns true if any element in the panel has focus
func (p *ClaudeOptionsPanel) IsFocused() bool {
	return p.focusIndex >= 0
}

// AtTop returns true if focus is on the first element
func (p *ClaudeOptionsPanel) AtTop() bool {
	return p.focusIndex <= 0
}

// GetOptions returns current options as ClaudeOptions
func (p *ClaudeOptionsPanel) GetOptions() *session.ClaudeOptions {
	opts := &session.ClaudeOptions{
		SkipPermissions:      p.skipPermissions,
		AllowSkipPermissions: p.allowSkipPermissions,
		UseChrome:            p.useChrome,
		UseTeammateMode:      p.useTeammateMode,
		UseVagrantMode:       p.useVagrantMode,
	}

	if !p.isForkMode {
		switch p.sessionMode {
		case 0:
			opts.SessionMode = "new"
		case 1:
			opts.SessionMode = "continue"
		case 2:
			opts.SessionMode = "resume"
			opts.ResumeSessionID = p.resumeIDInput.Value()
		}
	}

	return opts
}

// buildFocusItems returns a dynamic slice of focus item names based on current state.
// The slice changes depending on whether skipPermissions is checked (vagrantMode appears)
// and whether resume mode is active (resumeInput appears).
func (p *ClaudeOptionsPanel) buildFocusItems() []string {
	if p.isForkMode {
		items := []string{"skipPermissions"}
		if p.skipPermissions {
			items = append(items, "vagrantMode")
		}
		items = append(items, "chrome", "teammateMode")
		return items
	}

	// NewDialog mode
	items := []string{"sessionMode"}
	if p.sessionMode == 2 {
		items = append(items, "resumeInput")
	}
	items = append(items, "skipPermissions")
	if p.skipPermissions {
		items = append(items, "vagrantMode")
	}
	items = append(items, "chrome", "teammateMode")
	return items
}

// getFocusType returns what type of element is currently focused
func (p *ClaudeOptionsPanel) getFocusType() string {
	items := p.buildFocusItems()
	if p.focusIndex >= 0 && p.focusIndex < len(items) {
		return items[p.focusIndex]
	}
	return ""
}

// getFocusCount returns the number of focusable elements
func (p *ClaudeOptionsPanel) getFocusCount() int {
	return len(p.buildFocusItems())
}

// Update handles key events
func (p *ClaudeOptionsPanel) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up":
			p.focusIndex--
			if p.focusIndex < 0 {
				p.focusIndex = p.getFocusCount() - 1
			}
			p.updateInputFocus()
			return nil

		case "down", "tab":
			p.focusIndex++
			if p.focusIndex >= p.getFocusCount() {
				p.focusIndex = 0
			}
			p.updateInputFocus()
			return nil

		case "shift+tab":
			p.focusIndex--
			if p.focusIndex < 0 {
				p.focusIndex = p.getFocusCount() - 1
			}
			p.updateInputFocus()
			return nil

		case " ":
			// Don't intercept space when focused on a text input
			if p.isResumeInputFocused() {
				break // Let it fall through to text input handling
			}
			// Toggle checkbox or radio at current focus
			p.handleSpaceKey()
			return nil

		case "left", "right":
			// For session mode radio buttons
			if !p.isForkMode && p.focusIndex == 0 {
				if msg.String() == "left" {
					p.sessionMode--
					if p.sessionMode < 0 {
						p.sessionMode = 2
					}
				} else {
					p.sessionMode = (p.sessionMode + 1) % 3
				}
				return nil
			}
		}
	}

	// Update text inputs if focused
	if p.isResumeInputFocused() {
		var cmd tea.Cmd
		p.resumeIDInput, cmd = p.resumeIDInput.Update(msg)
		return cmd
	}

	return nil
}

// handleSpaceKey handles space key for toggling checkboxes/radios
func (p *ClaudeOptionsPanel) handleSpaceKey() {
	switch p.getFocusType() {
	case "sessionMode":
		// Cycle through modes on space
		p.sessionMode = (p.sessionMode + 1) % 3
	case "skipPermissions":
		p.skipPermissions = !p.skipPermissions
		// When unchecking skip permissions, also clear vagrant mode
		if !p.skipPermissions {
			p.useVagrantMode = false
		}
		// Clamp focusIndex if it now exceeds available items
		if p.focusIndex >= p.getFocusCount() {
			p.focusIndex = p.getFocusCount() - 1
		}
	case "chrome":
		p.useChrome = !p.useChrome
	case "teammateMode":
		p.useTeammateMode = !p.useTeammateMode
	case "vagrantMode":
		p.useVagrantMode = !p.useVagrantMode
	}
}

// isResumeInputFocused returns true if resume input is focused
func (p *ClaudeOptionsPanel) isResumeInputFocused() bool {
	return !p.isForkMode && p.sessionMode == 2 && p.focusIndex == 1
}

// updateInputFocus updates which text input has focus
func (p *ClaudeOptionsPanel) updateInputFocus() {
	p.resumeIDInput.Blur()

	if p.isResumeInputFocused() {
		p.resumeIDInput.Focus()
	}
}

// View renders the options panel
func (p *ClaudeOptionsPanel) View() string {
	labelStyle := lipgloss.NewStyle().Foreground(ColorText)
	activeStyle := lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(ColorComment)
	headerStyle := lipgloss.NewStyle().Foreground(ColorComment)

	var content string

	if p.isForkMode {
		content = p.viewForkMode(labelStyle, activeStyle, dimStyle, headerStyle)
	} else {
		content = p.viewNewMode(labelStyle, activeStyle, dimStyle, headerStyle)
	}

	return content
}

// viewForkMode renders options for ForkDialog
func (p *ClaudeOptionsPanel) viewForkMode(_, _, _, headerStyle lipgloss.Style) string {
	var content string
	content += headerStyle.Render("─ Advanced Options ─") + "\n"

	focusIdx := 0

	// Skip permissions checkbox
	content += renderCheckboxLine("Skip permissions", p.skipPermissions, p.focusIndex == focusIdx)
	focusIdx++

	// YOLO vagrant mode (only when skip permissions is checked)
	if p.skipPermissions {
		content += renderIndentedCheckboxLine("YOLO (sudo perms inside Vagrant VM)", p.useVagrantMode, p.focusIndex == focusIdx)
		focusIdx++
		if p.useVagrantMode {
			content += renderVagrantSettingsSummary()
		}
	}

	// Chrome checkbox
	content += renderCheckboxLine("Chrome mode", p.useChrome, p.focusIndex == focusIdx)
	focusIdx++

	// Teammate mode checkbox
	content += renderCheckboxLine("Teammate mode", p.useTeammateMode, p.focusIndex == focusIdx)

	return content
}

// viewNewMode renders options for NewDialog
func (p *ClaudeOptionsPanel) viewNewMode(_, activeStyle, _, headerStyle lipgloss.Style) string {
	var content string
	content += headerStyle.Render("─ Claude Options ─") + "\n"

	// Session mode radio buttons
	focusIdx := 0
	radioLabel := "  Session: "
	if p.focusIndex == focusIdx {
		radioLabel = activeStyle.Render("▶ Session: ")
	}
	content += radioLabel
	content += p.renderRadio("New", p.sessionMode == 0, p.focusIndex == focusIdx) + "  "
	content += p.renderRadio("Continue", p.sessionMode == 1, p.focusIndex == focusIdx) + "  "
	content += p.renderRadio("Resume", p.sessionMode == 2, p.focusIndex == focusIdx) + "\n"
	focusIdx++

	// Resume ID input (only if resume mode)
	if p.sessionMode == 2 {
		if p.focusIndex == focusIdx {
			content += activeStyle.Render("    ▶ ID: ") + p.resumeIDInput.View() + "\n"
		} else {
			content += "      ID: " + p.resumeIDInput.View() + "\n"
		}
		focusIdx++
	}

	// Skip permissions checkbox
	content += renderCheckboxLine("Skip permissions", p.skipPermissions, p.focusIndex == focusIdx)
	focusIdx++

	// YOLO vagrant mode (only when skip permissions is checked)
	if p.skipPermissions {
		content += renderIndentedCheckboxLine("YOLO (sudo perms inside Vagrant VM)", p.useVagrantMode, p.focusIndex == focusIdx)
		focusIdx++
		if p.useVagrantMode {
			content += renderVagrantSettingsSummary()
		}
	}

	// Chrome checkbox
	content += renderCheckboxLine("Chrome mode", p.useChrome, p.focusIndex == focusIdx)
	focusIdx++

	// Teammate mode checkbox
	content += renderCheckboxLine("Teammate mode", p.useTeammateMode, p.focusIndex == focusIdx)

	return content
}

// renderCheckboxMark renders a checkbox mark [x] or [ ] with consistent styling.
// Shared across all tool option panels for visual consistency.
func renderCheckboxMark(checked, focused bool) string {
	style := lipgloss.NewStyle()
	if focused {
		style = style.Foreground(ColorAccent).Bold(true)
	}
	if checked {
		return style.Render("[x]")
	}
	return style.Render("[ ]")
}

// renderCheckboxLine renders a complete checkbox line with label, matching Claude options panel style.
// Used by Gemini and Codex options in NewDialog for visual consistency with Claude.
func renderCheckboxLine(label string, checked, focused bool) string {
	activeStyle := lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
	labelStyle := lipgloss.NewStyle().Foreground(ColorText)

	cb := renderCheckboxMark(checked, focused)
	if focused {
		return activeStyle.Render("▶ ") + cb + " " + label + "\n"
	}
	return "  " + cb + " " + labelStyle.Render(label) + "\n"
}

// renderIndentedCheckboxLine renders a checkbox line indented as a sub-option.
// Used for YOLO vagrant mode which is a sub-option of skip permissions.
func renderIndentedCheckboxLine(label string, checked, focused bool) string {
	activeStyle := lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
	labelStyle := lipgloss.NewStyle().Foreground(ColorText)

	cb := renderCheckboxMark(checked, focused)
	if focused {
		return "  " + activeStyle.Render("▶ ") + cb + " " + label + "\n"
	}
	return "    " + cb + " " + labelStyle.Render(label) + "\n"
}

// renderVagrantSettingsSummary renders a compact summary of active vagrant settings
// below the YOLO checkbox. Shows memory, CPUs, and box in dim text.
func renderVagrantSettingsSummary() string {
	settings := session.GetVagrantSettings()
	dimStyle := lipgloss.NewStyle().Foreground(ColorComment)

	memory := settings.MemoryMB
	if memory <= 0 {
		memory = 16384
	}
	cpus := settings.CPUs
	if cpus <= 0 {
		cpus = 2
	}
	box := settings.Box
	if box == "" {
		box = "bento/ubuntu-24.04"
	}

	summary := fmt.Sprintf("Memory: %d MB | CPUs: %d | Box: %s", memory, cpus, box)
	return "      " + dimStyle.Render(summary) + "\n"
}

// renderRadio renders a radio button (•) or ( )
func (p *ClaudeOptionsPanel) renderRadio(label string, selected, focused bool) string {
	style := lipgloss.NewStyle()
	if focused && selected {
		style = style.Foreground(ColorAccent).Bold(true)
	} else if selected {
		style = style.Foreground(ColorCyan)
	} else {
		style = style.Foreground(ColorComment)
	}

	if selected {
		return style.Render("(•) " + label)
	}
	return style.Render("( ) " + label)
}
