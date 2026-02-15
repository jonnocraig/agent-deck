package ui

import (
	"testing"
	"time"
)

func TestNewCleanupDialog(t *testing.T) {
	dialog := NewCleanupDialog()

	if dialog == nil {
		t.Fatal("NewCleanupDialog returned nil")
	}

	if dialog.visible {
		t.Error("New dialog should not be visible")
	}

	if len(dialog.vms) != 0 {
		t.Errorf("New dialog should have 0 VMs, got %d", len(dialog.vms))
	}

	if dialog.selected == nil {
		t.Error("selected map should be initialized")
	}
}

func TestCleanupDialog_ShowHide(t *testing.T) {
	dialog := NewCleanupDialog()

	if dialog.IsVisible() {
		t.Error("Dialog should not be visible initially")
	}

	dialog.Show()
	if !dialog.IsVisible() {
		t.Error("Dialog should be visible after Show()")
	}

	dialog.Hide()
	if dialog.IsVisible() {
		t.Error("Dialog should not be visible after Hide()")
	}
}

func TestCleanupDialog_SetVMs(t *testing.T) {
	dialog := NewCleanupDialog()

	vms := []StaleVM{
		{
			ProjectPath: "/path/to/project1",
			VMState:     "saved",
			SuspendAge:  "3 days",
			VagrantID:   "abc123",
		},
		{
			ProjectPath: "/path/to/project2",
			VMState:     "poweroff",
			SuspendAge:  "1 hour",
			VagrantID:   "def456",
		},
	}

	dialog.SetVMs(vms)

	if len(dialog.vms) != 2 {
		t.Errorf("Expected 2 VMs, got %d", len(dialog.vms))
	}

	if dialog.cursor != 0 {
		t.Errorf("Cursor should be reset to 0, got %d", dialog.cursor)
	}

	if len(dialog.selected) != 0 {
		t.Errorf("Selected map should be cleared, has %d entries", len(dialog.selected))
	}
}

func TestCleanupDialog_GetSelectedVMs(t *testing.T) {
	dialog := NewCleanupDialog()

	vms := []StaleVM{
		{ProjectPath: "/path/1", VMState: "saved", VagrantID: "a"},
		{ProjectPath: "/path/2", VMState: "saved", VagrantID: "b"},
		{ProjectPath: "/path/3", VMState: "saved", VagrantID: "c"},
	}

	dialog.SetVMs(vms)
	dialog.selected[0] = true
	dialog.selected[2] = true

	selected := dialog.GetSelectedVMs()

	if len(selected) != 2 {
		t.Errorf("Expected 2 selected VMs, got %d", len(selected))
	}

	if selected[0].VagrantID != "a" || selected[1].VagrantID != "c" {
		t.Error("Wrong VMs selected")
	}
}

func TestCleanupDialog_SetSize(t *testing.T) {
	dialog := NewCleanupDialog()

	dialog.SetSize(100, 50)

	if dialog.width != 100 || dialog.height != 50 {
		t.Errorf("Size not set correctly: got %dx%d, want 100x50", dialog.width, dialog.height)
	}
}

func TestCleanupDialog_SetError(t *testing.T) {
	dialog := NewCleanupDialog()
	dialog.destroying = true

	dialog.SetError("test error")

	if dialog.errorMsg != "test error" {
		t.Errorf("Error message not set: got %q", dialog.errorMsg)
	}

	if dialog.destroying {
		t.Error("SetError should clear destroying state")
	}
}

func TestFormatVMAge(t *testing.T) {
	tests := []struct {
		name     string
		duration string
		want     string
	}{
		{"just now", "0s", "just now"},
		{"minutes", "5m", "5 minutes"},
		{"hours", "3h", "3 hours"},
		{"days", "48h", "2 days"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse duration string
			d, err := parseDuration(tt.duration)
			if err != nil {
				t.Fatalf("Failed to parse duration %s: %v", tt.duration, err)
			}

			got := formatVMAge(d)
			if got != tt.want {
				t.Errorf("formatVMAge(%s) = %q, want %q", tt.duration, got, tt.want)
			}
		})
	}
}

// Helper to parse duration strings
func parseDuration(s string) (time.Duration, error) {
	// Simple duration parser for tests
	if s == "0s" {
		return 0, nil
	}
	if s == "5m" {
		return 5 * time.Minute, nil
	}
	if s == "3h" {
		return 3 * time.Hour, nil
	}
	if s == "48h" {
		return 48 * time.Hour, nil
	}
	return time.ParseDuration(s)
}
