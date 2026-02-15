package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestClaudeOptionsPanelCreation verifies NewClaudeOptionsPanel creates valid panel
func TestClaudeOptionsPanelCreation(t *testing.T) {
	panel := NewClaudeOptionsPanel()

	if panel == nil {
		t.Fatal("NewClaudeOptionsPanel returned nil")
	}

	if panel.sessionMode != 0 {
		t.Errorf("sessionMode = %d, want 0 (new)", panel.sessionMode)
	}

	if panel.isForkMode {
		t.Error("isForkMode should be false for NewClaudeOptionsPanel")
	}

	if panel.focusCount != 6 {
		t.Errorf("focusCount = %d, want 6", panel.focusCount)
	}

	if panel.skipPermissions {
		t.Error("skipPermissions should be false by default")
	}

	if panel.useVagrantMode {
		t.Error("useVagrantMode should be false by default")
	}
}

// TestVagrantModeCheckboxExists verifies the checkbox renders in the View
func TestVagrantModeCheckboxExists(t *testing.T) {
	panel := NewClaudeOptionsPanel()

	view := panel.View()

	if !strings.Contains(view, "Just do it (vagrant sudo)") {
		t.Error("View should contain vagrant mode checkbox label")
	}

	if !strings.Contains(view, "[ ]") {
		t.Error("View should contain unchecked checkbox mark")
	}
}

// TestVagrantModeForceSkipPermissions verifies toggling vagrant mode ON forces skip permissions
func TestVagrantModeForceSkipPermissions(t *testing.T) {
	panel := NewClaudeOptionsPanel()

	// Initially both should be false
	if panel.skipPermissions {
		t.Error("skipPermissions should be false initially")
	}
	if panel.useVagrantMode {
		t.Error("useVagrantMode should be false initially")
	}

	// Focus the vagrant mode checkbox (index 4 in NewDialog mode)
	panel.Focus()
	panel.focusIndex = 4

	// Toggle vagrant mode ON with space key
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
	panel.Update(msg)

	// Verify vagrant mode is enabled
	if !panel.useVagrantMode {
		t.Error("useVagrantMode should be true after toggle")
	}

	// Verify skip permissions was forced ON
	if !panel.skipPermissions {
		t.Error("skipPermissions should be forced true when vagrant mode is enabled")
	}

	// Toggle vagrant mode OFF
	panel.Update(msg)

	// Verify vagrant mode is disabled
	if panel.useVagrantMode {
		t.Error("useVagrantMode should be false after second toggle")
	}

	// Verify skip permissions was restored to previous state (false)
	if panel.skipPermissions {
		t.Error("skipPermissions should be restored to false when vagrant mode is disabled")
	}
}

// TestGetOptionsIncludesVagrantMode verifies GetOptions returns UseVagrantMode field
func TestGetOptionsIncludesVagrantMode(t *testing.T) {
	panel := NewClaudeOptionsPanel()

	// Get options when vagrant mode is disabled
	opts := panel.GetOptions()
	if opts == nil {
		t.Fatal("GetOptions returned nil")
	}
	if opts.UseVagrantMode {
		t.Error("UseVagrantMode should be false by default")
	}

	// Enable vagrant mode through proper toggle (which also sets skipPermissions)
	panel.Focus()
	panel.focusIndex = 4
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
	panel.Update(msg)

	// Get options when vagrant mode is enabled
	opts = panel.GetOptions()
	if !opts.UseVagrantMode {
		t.Error("UseVagrantMode should be true when vagrant mode is enabled")
	}

	// Verify skip permissions is also set
	if !opts.SkipPermissions {
		t.Error("SkipPermissions should be true when vagrant mode is enabled")
	}
}

// TestVagrantModePreservesUserSkipPermissionsChoice verifies restoration behavior
func TestVagrantModePreservesUserSkipPermissionsChoice(t *testing.T) {
	panel := NewClaudeOptionsPanel()

	// User enables skip permissions manually
	panel.skipPermissions = true
	panel.prevSkipPermissions = true

	// Focus vagrant mode checkbox
	panel.Focus()
	panel.focusIndex = 4

	// Toggle vagrant mode ON
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
	panel.Update(msg)

	if !panel.useVagrantMode {
		t.Error("useVagrantMode should be true after toggle")
	}
	if !panel.skipPermissions {
		t.Error("skipPermissions should remain true")
	}

	// Toggle vagrant mode OFF
	panel.Update(msg)

	// Verify skip permissions was restored to true (user's original choice)
	if !panel.skipPermissions {
		t.Error("skipPermissions should be restored to true (user's choice)")
	}
}

// TestVagrantModeInForkDialog verifies vagrant mode works in fork mode
func TestVagrantModeInForkDialog(t *testing.T) {
	panel := NewClaudeOptionsPanelForFork()

	if panel == nil {
		t.Fatal("NewClaudeOptionsPanelForFork returned nil")
	}

	if !panel.isForkMode {
		t.Error("isForkMode should be true for fork panel")
	}

	if panel.focusCount != 4 {
		t.Errorf("focusCount = %d, want 4 for fork mode", panel.focusCount)
	}

	// Focus vagrant mode checkbox (index 3 in fork mode)
	panel.Focus()
	panel.focusIndex = 3

	// Toggle vagrant mode ON
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
	panel.Update(msg)

	if !panel.useVagrantMode {
		t.Error("useVagrantMode should be true after toggle in fork mode")
	}

	if !panel.skipPermissions {
		t.Error("skipPermissions should be forced true in fork mode")
	}

	// Verify in options
	opts := panel.GetOptions()
	if !opts.UseVagrantMode {
		t.Error("GetOptions should include UseVagrantMode=true")
	}
}
