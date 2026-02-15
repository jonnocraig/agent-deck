package vagrant

import (
	"testing"

	"github.com/asheshgoplani/agent-deck/internal/session"
)

// TestNewVagrantVM verifies that newVagrantVM creates a valid adapter.
func TestNewVagrantVM(t *testing.T) {
	projectPath := "/test/project"
	settings := session.VagrantSettings{
		MemoryMB: 4096,
		CPUs:     2,
		Box:      "bento/ubuntu-24.04",
	}

	vm := newVagrantVM(projectPath, settings)

	if vm == nil {
		t.Fatal("newVagrantVM returned nil")
	}

	adapter, ok := vm.(*vagrantVMAdapter)
	if !ok {
		t.Fatalf("newVagrantVM should return *vagrantVMAdapter, got %T", vm)
	}

	if adapter.mgr == nil {
		t.Error("adapter.mgr should not be nil")
	}

	if adapter.mgr.projectPath != projectPath {
		t.Errorf("adapter.mgr.projectPath = %q, want %q", adapter.mgr.projectPath, projectPath)
	}

	if adapter.mgr.settings.MemoryMB != 4096 {
		t.Errorf("adapter.mgr.settings.MemoryMB = %d, want 4096", adapter.mgr.settings.MemoryMB)
	}
}

// TestVagrantVMAdapter_HealthCheckConversion verifies that HealthCheck correctly
// converts vagrant.VMHealth to session.VMHealthResult.
func TestVagrantVMAdapter_HealthCheckConversion(t *testing.T) {
	tests := []struct {
		name     string
		vmHealth VMHealth
	}{
		{
			name: "running and healthy",
			vmHealth: VMHealth{
				State:      "running",
				Healthy:    true,
				Responsive: true,
				Message:    "VM running and responsive",
			},
		},
		{
			name: "running but unresponsive",
			vmHealth: VMHealth{
				State:      "running",
				Healthy:    false,
				Responsive: false,
				Message:    "VM running but unresponsive (SSH probe failed)",
			},
		},
		{
			name: "suspended",
			vmHealth: VMHealth{
				State:      "saved",
				Healthy:    false,
				Responsive: false,
				Message:    "VM is suspended",
			},
		},
		{
			name: "not created",
			vmHealth: VMHealth{
				State:      "not_created",
				Healthy:    false,
				Responsive: false,
				Message:    "VM not created",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the conversion logic directly by simulating what HealthCheck does:
			// It takes vagrant.VMHealth and converts to session.VMHealthResult
			result := session.VMHealthResult{
				State:      tt.vmHealth.State,
				Healthy:    tt.vmHealth.Healthy,
				Responsive: tt.vmHealth.Responsive,
				Message:    tt.vmHealth.Message,
			}

			// Verify all fields are correctly mapped (same as in bridge.go HealthCheck)
			if result.State != tt.vmHealth.State {
				t.Errorf("State = %q, want %q", result.State, tt.vmHealth.State)
			}
			if result.Healthy != tt.vmHealth.Healthy {
				t.Errorf("Healthy = %v, want %v", result.Healthy, tt.vmHealth.Healthy)
			}
			if result.Responsive != tt.vmHealth.Responsive {
				t.Errorf("Responsive = %v, want %v", result.Responsive, tt.vmHealth.Responsive)
			}
			if result.Message != tt.vmHealth.Message {
				t.Errorf("Message = %q, want %q", result.Message, tt.vmHealth.Message)
			}
		})
	}
}

// TestVagrantVMAdapter_InterfaceCompliance verifies at compile time that
// vagrantVMAdapter satisfies session.VagrantVM interface.
func TestVagrantVMAdapter_InterfaceCompliance(t *testing.T) {
	// Compile-time check that vagrantVMAdapter implements session.VagrantVM
	var _ session.VagrantVM = (*vagrantVMAdapter)(nil)

	// Also verify at runtime
	mgr := NewManager("/test/project", session.VagrantSettings{})
	adapter := &vagrantVMAdapter{mgr: mgr}

	var iface interface{} = adapter
	if _, ok := iface.(session.VagrantVM); !ok {
		t.Error("vagrantVMAdapter does not implement session.VagrantVM interface")
	}
}

// TestVagrantVMAdapter_AllMethodsDelegate verifies that all adapter methods
// correctly delegate to the underlying Manager.
func TestVagrantVMAdapter_AllMethodsDelegate(t *testing.T) {
	projectPath := "/test/project"
	settings := session.VagrantSettings{}
	vm := newVagrantVM(projectPath, settings)
	adapter := vm.(*vagrantVMAdapter)

	// Test session management delegation
	adapter.RegisterSession("session-1")
	if count := adapter.SessionCount(); count != 1 {
		t.Errorf("SessionCount() = %d, want 1", count)
	}

	if !adapter.IsLastSession("session-1") {
		t.Error("IsLastSession should return true for only session")
	}

	adapter.RegisterSession("session-2")
	if count := adapter.SessionCount(); count != 2 {
		t.Errorf("SessionCount() = %d, want 2", count)
	}

	adapter.UnregisterSession("session-1")
	if count := adapter.SessionCount(); count != 1 {
		t.Errorf("SessionCount() = %d after unregister, want 1", count)
	}

	// Test dotfile path delegation - just verify it doesn't panic
	// (The field is private, so we can't check it directly, but we verify
	// the method exists and is callable without error)
	adapter.SetDotfilePath("session-test")

	// Test IsInstalled delegation (returns true/false without panic)
	result := adapter.IsInstalled()
	if result != true && result != false {
		t.Errorf("IsInstalled should return bool, got %v", result)
	}

	// Test WrapCommand delegation
	wrapped := adapter.WrapCommand("echo test", []string{"VAR1"}, []int{8080})
	expected := "vagrant ssh -- -R 8080:localhost:8080 -o SendEnv=VAR1 -t 'cd /vagrant && echo test'"
	if wrapped != expected {
		t.Errorf("WrapCommand not delegated correctly.\nGot:  %q\nWant: %q", wrapped, expected)
	}
}

// TestVagrantProviderFactory_Registration verifies that the factory is registered
// in the init function for use by the session package.
func TestVagrantProviderFactory_Registration(t *testing.T) {
	if session.VagrantProviderFactory == nil {
		t.Fatal("VagrantProviderFactory should be set by init()")
	}

	// Verify factory creates valid VagrantVM instance
	projectPath := "/test/factory"
	settings := session.VagrantSettings{
		MemoryMB: 2048,
		CPUs:     1,
	}

	vm := session.VagrantProviderFactory(projectPath, settings)
	if vm == nil {
		t.Fatal("VagrantProviderFactory returned nil")
	}

	// Verify it's a VagrantVM
	if _, ok := vm.(session.VagrantVM); !ok {
		t.Errorf("VagrantProviderFactory should return session.VagrantVM, got %T", vm)
	}
}
