package ui

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// CleanupDialog shows a list of stale suspended VMs with checkboxes
// for selective destruction.
type CleanupDialog struct {
	vms             []StaleVM
	cursor          int
	selected        map[int]bool
	visible         bool
	destroying      bool
	destroyProgress string
	width           int
	height          int
	errorMsg        string
}

// StaleVM represents a suspended Vagrant VM that can be cleaned up.
type StaleVM struct {
	ProjectPath string
	VMState     string // e.g., "saved", "poweroff"
	SuspendAge  string // e.g., "3 days"
	VagrantID   string // ID from vagrant global-status
}

// NewCleanupDialog creates a new cleanup dialog.
func NewCleanupDialog() *CleanupDialog {
	return &CleanupDialog{
		selected: make(map[int]bool),
	}
}

// SetVMs populates the VM list.
func (d *CleanupDialog) SetVMs(vms []StaleVM) {
	d.vms = vms
	d.cursor = 0
	d.selected = make(map[int]bool)
	d.errorMsg = ""
}

// Show displays the dialog.
func (d *CleanupDialog) Show() {
	d.visible = true
	d.cursor = 0
	d.destroying = false
	d.destroyProgress = ""
	d.errorMsg = ""
}

// Hide hides the dialog.
func (d *CleanupDialog) Hide() {
	d.visible = false
	d.destroying = false
	d.destroyProgress = ""
	d.errorMsg = ""
}

// IsVisible returns whether the dialog is visible.
func (d *CleanupDialog) IsVisible() bool {
	return d.visible
}

// SetSize sets the dialog dimensions for centering.
func (d *CleanupDialog) SetSize(width, height int) {
	d.width = width
	d.height = height
}

// SetError sets an error message on the dialog.
func (d *CleanupDialog) SetError(msg string) {
	d.errorMsg = msg
	d.destroying = false
}

// SetDestroying sets the destroying state.
func (d *CleanupDialog) SetDestroying(destroying bool) {
	d.destroying = destroying
}

// SetDestroyProgress updates the progress message during destruction.
func (d *CleanupDialog) SetDestroyProgress(msg string) {
	d.destroyProgress = msg
}

// GetSelectedVMs returns the list of selected VMs.
func (d *CleanupDialog) GetSelectedVMs() []StaleVM {
	selected := []StaleVM{}
	for i, vm := range d.vms {
		if d.selected[i] {
			selected = append(selected, vm)
		}
	}
	return selected
}

// Update handles input events.
func (d *CleanupDialog) Update(msg tea.Msg) (*CleanupDialog, tea.Cmd) {
	if !d.visible || d.destroying {
		return d, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if d.cursor > 0 {
				d.cursor--
			}

		case "down", "j":
			if d.cursor < len(d.vms)-1 {
				d.cursor++
			}

		case " ":
			// Toggle selection
			d.selected[d.cursor] = !d.selected[d.cursor]

		case "a":
			// Select/deselect all
			allSelected := true
			for i := range d.vms {
				if !d.selected[i] {
					allSelected = false
					break
				}
			}
			// Toggle: if all selected, deselect all; otherwise select all
			for i := range d.vms {
				d.selected[i] = !allSelected
			}

		case "enter":
			// Confirm destruction of selected VMs
			selectedVMs := d.GetSelectedVMs()
			if len(selectedVMs) == 0 {
				d.errorMsg = "No VMs selected"
				return d, nil
			}
			// Signal to parent that destruction should begin
			return d, nil

		case "esc":
			d.Hide()
			return d, nil
		}
	}

	return d, nil
}

// View renders the dialog.
func (d *CleanupDialog) View() string {
	if !d.visible {
		return ""
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorCyan)
	labelStyle := lipgloss.NewStyle().Foreground(ColorText)
	checkboxStyle := lipgloss.NewStyle().Foreground(ColorText)
	checkboxActiveStyle := lipgloss.NewStyle().Foreground(ColorCyan).Bold(true)
	footerStyle := lipgloss.NewStyle().Foreground(ColorComment)
	errStyle := lipgloss.NewStyle().Foreground(ColorRed).Bold(true)

	// Responsive dialog width
	dialogWidth := 70
	if d.width > 0 && d.width < dialogWidth+10 {
		dialogWidth = d.width - 10
		if dialogWidth < 40 {
			dialogWidth = 40
		}
	}

	boxBorder := ColorAccent
	if d.errorMsg != "" {
		boxBorder = ColorRed
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(boxBorder).
		Padding(1, 2).
		Width(dialogWidth)

	// Show progress/loading state
	if d.destroying {
		var b strings.Builder
		b.WriteString(titleStyle.Render("Cleaning Up VMs..."))
		b.WriteString("\n\n")
		b.WriteString(labelStyle.Render("  " + d.destroyProgress))
		b.WriteString("\n\n")
		b.WriteString(footerStyle.Render("Please wait..."))
		dialog := boxStyle.Render(b.String())
		return lipgloss.Place(d.width, d.height, lipgloss.Center, lipgloss.Center, dialog)
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render("Stale VM Cleanup"))
	b.WriteString("\n\n")

	if len(d.vms) == 0 {
		b.WriteString(labelStyle.Render("  No stale VMs found."))
		b.WriteString("\n\n")
		b.WriteString(footerStyle.Render("Esc close"))
	} else {
		b.WriteString(labelStyle.Render("Select VMs to destroy:"))
		b.WriteString("\n\n")

		// List VMs with checkboxes
		for i, vm := range d.vms {
			checkbox := "[ ]"
			if d.selected[i] {
				checkbox = "[x]"
			}

			// Truncate project path for display
			displayPath := vm.ProjectPath
			if len(displayPath) > 50 {
				displayPath = "..." + displayPath[len(displayPath)-47:]
			}

			line := fmt.Sprintf("%s %s (%s, %s)", checkbox, displayPath, vm.VMState, vm.SuspendAge)

			if i == d.cursor {
				b.WriteString(checkboxActiveStyle.Render("▶ " + line))
			} else {
				b.WriteString(checkboxStyle.Render("  " + line))
			}
			b.WriteString("\n")
		}

		if d.errorMsg != "" {
			b.WriteString("\n")
			b.WriteString(errStyle.Render("  " + d.errorMsg))
			b.WriteString("\n")
		}

		b.WriteString("\n")
		selectedCount := 0
		for _, isSelected := range d.selected {
			if isSelected {
				selectedCount++
			}
		}
		if selectedCount > 0 {
			b.WriteString(footerStyle.Render(fmt.Sprintf("[Space] Toggle  [A] All  [Enter] Destroy (%d)  [Esc] Cancel", selectedCount)))
		} else {
			b.WriteString(footerStyle.Render("[Space] Toggle  [A] All  [Enter] Destroy  [Esc] Cancel"))
		}
	}

	dialog := boxStyle.Render(b.String())
	return lipgloss.Place(d.width, d.height, lipgloss.Center, lipgloss.Center, dialog)
}

// ListSuspendedAgentDeckVMs runs `vagrant global-status --machine-readable`
// and cross-references with agent-deck sessions to find stale VMs.
func ListSuspendedAgentDeckVMs() ([]StaleVM, error) {
	cmd := exec.Command("vagrant", "global-status", "--machine-readable")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("vagrant global-status failed: %w (output: %s)", err, string(output))
	}

	staleVMs := []StaleVM{}
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	// Parse machine-readable output
	// Format: timestamp,id,metadata-key,value
	// We're interested in lines with metadata-key = "state" where value = "saved" or "poweroff"
	vmData := make(map[string]map[string]string) // vagrantID -> {state, directory, etc}

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Split(line, ",")
		if len(fields) < 4 {
			continue
		}

		vagrantID := fields[1]
		metadataKey := fields[2]
		value := fields[3]

		if vmData[vagrantID] == nil {
			vmData[vagrantID] = make(map[string]string)
		}
		vmData[vagrantID][metadataKey] = value
	}

	// Filter for suspended VMs in agent-deck project directories
	for vagrantID, data := range vmData {
		state := data["state"]
		directory := data["directory"]

		// Only include suspended/powered-off VMs
		if state != "saved" && state != "poweroff" {
			continue
		}

		// Check if directory exists and looks like an agent-deck project
		// (has .agent-deck marker or is a known project directory)
		if directory == "" {
			continue
		}

		// Calculate suspend age (if possible from state-human-long-timestamp)
		suspendAge := "unknown"
		if timestamp := data["updated-at"]; timestamp != "" {
			if t, err := time.Parse(time.RFC3339, timestamp); err == nil {
				age := time.Since(t)
				suspendAge = formatVMAge(age)
			}
		}

		staleVMs = append(staleVMs, StaleVM{
			ProjectPath: directory,
			VMState:     state,
			SuspendAge:  suspendAge,
			VagrantID:   vagrantID,
		})
	}

	return staleVMs, nil
}

// DestroySuspendedVMs destroys the selected VMs sequentially.
// Returns errors for any VMs that failed to destroy.
func DestroySuspendedVMs(vms []StaleVM) []error {
	var errs []error

	for _, vm := range vms {
		// Verify Vagrantfile exists to prevent path traversal and ensure valid Vagrant project
		vagrantfile := filepath.Join(vm.ProjectPath, "Vagrantfile")
		if _, err := os.Stat(vagrantfile); err != nil {
			errs = append(errs, fmt.Errorf("skipping %s: Vagrantfile not found", vm.ProjectPath))
			continue
		}

		// cd to VM's project directory and run vagrant destroy -f
		cmd := exec.Command("vagrant", "destroy", "-f")
		cmd.Dir = vm.ProjectPath

		if err := cmd.Run(); err != nil {
			// If destroy fails, leave .vagrant directory intact so user can retry
			errs = append(errs, fmt.Errorf("failed to destroy VM at %s: %w", vm.ProjectPath, err))
			continue
		}

		// Only clean up .vagrant directory if destroy succeeded
		vagrantDir := filepath.Join(vm.ProjectPath, ".vagrant")
		if _, err := os.Stat(vagrantDir); err == nil {
			if err := os.RemoveAll(vagrantDir); err != nil {
				// Log warning but don't treat as fatal error
				errs = append(errs, fmt.Errorf("warning: destroyed VM but failed to clean .vagrant at %s: %w", vm.ProjectPath, err))
			}
		}
	}

	return errs
}

// formatVMAge formats a duration in a human-readable way for VM age display.
func formatVMAge(d time.Duration) string {
	days := int(d.Hours() / 24)
	if days > 0 {
		return fmt.Sprintf("%d days", days)
	}
	hours := int(d.Hours())
	if hours > 0 {
		return fmt.Sprintf("%d hours", hours)
	}
	minutes := int(d.Minutes())
	if minutes > 0 {
		return fmt.Sprintf("%d minutes", minutes)
	}
	return "just now"
}
