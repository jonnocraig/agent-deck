package vagrant

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestVMStateMessage(t *testing.T) {
	tests := []struct {
		state   string
		wantMsg string
	}{
		{"running", "VM running and responsive"},
		{"saved", "VM is suspended"},
		{"not_created", "VM not created"},
		{"aborted", "VM crashed or was force-stopped"},
		{"poweroff", "VM is powered off"},
		{"unknown_state", "VM in unexpected state: unknown_state"},
	}

	for _, tt := range tests {
		t.Run(tt.state, func(t *testing.T) {
			got := vmStateMessage(tt.state)
			if got != tt.wantMsg {
				t.Errorf("vmStateMessage(%q) = %q, want %q", tt.state, got, tt.wantMsg)
			}
		})
	}
}

func TestParseVagrantState(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{
			name:   "running state",
			output: "1234567890,default,state,running\n",
			want:   "running",
		},
		{
			name:   "suspended state",
			output: "1234567890,default,state,saved\n",
			want:   "saved",
		},
		{
			name:   "not created",
			output: "1234567890,default,state,not_created\n",
			want:   "not_created",
		},
		{
			name:   "poweroff",
			output: "1234567890,default,state,poweroff\n",
			want:   "poweroff",
		},
		{
			name:   "aborted",
			output: "1234567890,default,state,aborted\n",
			want:   "aborted",
		},
		{
			name: "multiple lines",
			output: `1234567890,default,metadata,provider,virtualbox
1234567890,default,state,running
1234567890,default,metadata,state-human-short,running
`,
			want: "running",
		},
		{
			name:   "no state line",
			output: "1234567890,default,metadata,provider,virtualbox\n",
			want:   "",
		},
		{
			name:   "empty output",
			output: "",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseStateFromOutput(tt.output)
			if got != tt.want {
				t.Errorf("parseStateFromOutput() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHealthCheckNotCreated(t *testing.T) {
	// Mock vagrant status output for not_created state
	statusOutput := "1234567890,default,state,not_created\n"
	state := parseStateFromOutput(statusOutput)

	health := buildVMHealth(state, false, nil)

	if health.State != "not_created" {
		t.Errorf("State = %q, want %q", health.State, "not_created")
	}
	if health.Healthy {
		t.Errorf("Healthy = true, want false")
	}
	if health.Responsive {
		t.Errorf("Responsive = true, want false")
	}
	if health.Message != "VM not created" {
		t.Errorf("Message = %q, want %q", health.Message, "VM not created")
	}
}

func TestHealthCheckSuspendedSkipsPhase2(t *testing.T) {
	// Mock vagrant status output for suspended state
	statusOutput := "1234567890,default,state,saved\n"
	state := parseStateFromOutput(statusOutput)

	health := buildVMHealth(state, false, nil)

	if health.State != "saved" {
		t.Errorf("State = %q, want %q", health.State, "saved")
	}
	if health.Healthy {
		t.Errorf("Healthy = true, want false")
	}
	if health.Responsive {
		t.Errorf("Responsive = true, want false")
	}
	if health.Message != "VM is suspended" {
		t.Errorf("Message = %q, want %q", health.Message, "VM is suspended")
	}
}

func TestHealthCheckRunningPhase2Success(t *testing.T) {
	// When state is "running", we should check SSH probe
	state := "running"
	sshSuccess := true

	health := buildVMHealth(state, sshSuccess, nil)

	if health.State != "running" {
		t.Errorf("State = %q, want %q", health.State, "running")
	}
	if !health.Healthy {
		t.Errorf("Healthy = false, want true")
	}
	if !health.Responsive {
		t.Errorf("Responsive = false, want true")
	}
	if health.Message != "VM running and responsive" {
		t.Errorf("Message = %q, want %q", health.Message, "VM running and responsive")
	}
}

func TestHealthCheckRunningPhase2Failure(t *testing.T) {
	// When state is "running" but SSH probe fails
	state := "running"
	sshSuccess := false
	sshError := context.DeadlineExceeded

	health := buildVMHealth(state, sshSuccess, sshError)

	if health.State != "running" {
		t.Errorf("State = %q, want %q", health.State, "running")
	}
	if health.Healthy {
		t.Errorf("Healthy = true, want false")
	}
	if health.Responsive {
		t.Errorf("Responsive = true, want false")
	}
	if !strings.Contains(health.Message, "unresponsive") {
		t.Errorf("Message = %q, should contain 'unresponsive'", health.Message)
	}
}

func TestHealthCheckCacheTTL(t *testing.T) {
	m := &Manager{
		projectPath: "/test/path",
		cache: &healthCache{
			lastCheck: time.Now(),
			result: VMHealth{
				State:      "running",
				Healthy:    true,
				Responsive: true,
				Message:    "VM running and responsive",
			},
			ttl: 30 * time.Second,
		},
	}

	// Cache should be valid
	if !m.cache.isValid() {
		t.Error("Cache should be valid within TTL")
	}

	// Expire the cache
	m.cache.lastCheck = time.Now().Add(-31 * time.Second)

	if m.cache.isValid() {
		t.Error("Cache should be invalid after TTL")
	}
}

func TestHealthCheckCacheNil(t *testing.T) {
	m := &Manager{
		projectPath: "/test/path",
		cache:       nil,
	}

	// Should not panic with nil cache
	m.initCache()

	if m.cache == nil {
		t.Error("Cache should be initialized")
	}

	if m.cache.ttl != 30*time.Second {
		t.Errorf("Cache TTL = %v, want %v", m.cache.ttl, 30*time.Second)
	}
}

func TestBuildVMHealthAllStates(t *testing.T) {
	tests := []struct {
		name       string
		state      string
		sshSuccess bool
		sshErr     error
		wantHealth VMHealth
	}{
		{
			name:       "running and responsive",
			state:      "running",
			sshSuccess: true,
			sshErr:     nil,
			wantHealth: VMHealth{
				State:      "running",
				Healthy:    true,
				Responsive: true,
				Message:    "VM running and responsive",
			},
		},
		{
			name:       "running but unresponsive",
			state:      "running",
			sshSuccess: false,
			sshErr:     context.DeadlineExceeded,
			wantHealth: VMHealth{
				State:      "running",
				Healthy:    false,
				Responsive: false,
				Message:    "VM running but unresponsive (SSH probe failed)",
			},
		},
		{
			name:       "suspended",
			state:      "saved",
			sshSuccess: false,
			sshErr:     nil,
			wantHealth: VMHealth{
				State:      "saved",
				Healthy:    false,
				Responsive: false,
				Message:    "VM is suspended",
			},
		},
		{
			name:       "not created",
			state:      "not_created",
			sshSuccess: false,
			sshErr:     nil,
			wantHealth: VMHealth{
				State:      "not_created",
				Healthy:    false,
				Responsive: false,
				Message:    "VM not created",
			},
		},
		{
			name:       "aborted",
			state:      "aborted",
			sshSuccess: false,
			sshErr:     nil,
			wantHealth: VMHealth{
				State:      "aborted",
				Healthy:    false,
				Responsive: false,
				Message:    "VM crashed or was force-stopped",
			},
		},
		{
			name:       "poweroff",
			state:      "poweroff",
			sshSuccess: false,
			sshErr:     nil,
			wantHealth: VMHealth{
				State:      "poweroff",
				Healthy:    false,
				Responsive: false,
				Message:    "VM is powered off",
			},
		},
		{
			name:       "unexpected state",
			state:      "weird_state",
			sshSuccess: false,
			sshErr:     nil,
			wantHealth: VMHealth{
				State:      "weird_state",
				Healthy:    false,
				Responsive: false,
				Message:    "VM in unexpected state: weird_state",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildVMHealth(tt.state, tt.sshSuccess, tt.sshErr)

			if got.State != tt.wantHealth.State {
				t.Errorf("State = %q, want %q", got.State, tt.wantHealth.State)
			}
			if got.Healthy != tt.wantHealth.Healthy {
				t.Errorf("Healthy = %v, want %v", got.Healthy, tt.wantHealth.Healthy)
			}
			if got.Responsive != tt.wantHealth.Responsive {
				t.Errorf("Responsive = %v, want %v", got.Responsive, tt.wantHealth.Responsive)
			}
			if got.Message != tt.wantHealth.Message {
				t.Errorf("Message = %q, want %q", got.Message, tt.wantHealth.Message)
			}
		})
	}
}
