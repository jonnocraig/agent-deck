package ui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/asheshgoplani/agent-deck/internal/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mock Types ---

// mockTransitionEngineForConductor implements TransitionEngine for conductor tests.
type mockTransitionEngineForConductor struct {
	mu sync.Mutex

	// RequestMove behavior
	moveResults  map[string]MoveResult // key: "from->to"
	moveCalls    []MoveRequest
	moveErr      error
	defaultMove  MoveResult

	// ResolveSkill behavior
	skillResults map[session.KanbanColumn]SkillMapping
	skillErr     error

	// IsValidMove behavior
	validMoveResults map[string]MoveValidation // key: "from->to"

	// Rollback behavior
	rollbackErr error
}

func newMockTransitionEngine() *mockTransitionEngineForConductor {
	return &mockTransitionEngineForConductor{
		moveResults:      make(map[string]MoveResult),
		skillResults:     make(map[session.KanbanColumn]SkillMapping),
		validMoveResults: make(map[string]MoveValidation),
		defaultMove: MoveResult{
			Success: true,
		},
	}
}

func (m *mockTransitionEngineForConductor) IsValidMove(from, to session.KanbanColumn) MoveValidation {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := fmt.Sprintf("%s->%s", from, to)
	if v, ok := m.validMoveResults[key]; ok {
		return v
	}
	return MoveValidation{Valid: true}
}

func (m *mockTransitionEngineForConductor) RequestMove(req MoveRequest) MoveResult {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.moveCalls = append(m.moveCalls, req)
	key := fmt.Sprintf("%s->%s", req.FromColumn, req.ToColumn)
	if r, ok := m.moveResults[key]; ok {
		return r
	}
	return MoveResult{
		Success:    m.defaultMove.Success,
		FromColumn: req.FromColumn,
		ToColumn:   req.ToColumn,
	}
}

func (m *mockTransitionEngineForConductor) ResolveSkill(groupPath string, column session.KanbanColumn) (SkillMapping, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.skillErr != nil {
		return SkillMapping{}, m.skillErr
	}
	if s, ok := m.skillResults[column]; ok {
		return s, nil
	}
	return SkillMapping{
		ColumnName: column,
		SkillName:  "skill-" + column.String(),
		AutoTrigger: true,
		TriggerOnEnter: true,
	}, nil
}

func (m *mockTransitionEngineForConductor) Rollback(sessionID string, originalColumn session.KanbanColumn, originalSortOrder int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.rollbackErr
}

func (m *mockTransitionEngineForConductor) getMoveCalls() []MoveRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	calls := make([]MoveRequest, len(m.moveCalls))
	copy(calls, m.moveCalls)
	return calls
}

// mockGateChecker implements GateChecker for conductor tests.
type mockGateChecker struct {
	mu      sync.Mutex
	results map[string]GateResult // key: "from->to"
	err     error
	calls   []gateCheckCall
}

type gateCheckCall struct {
	SessionID      string
	From           session.KanbanColumn
	To             session.KanbanColumn
	ContinuationID string
}

func newMockGateChecker() *mockGateChecker {
	return &mockGateChecker{
		results: make(map[string]GateResult),
	}
}

func (m *mockGateChecker) CheckGate(ctx context.Context, sessionID string, from, to session.KanbanColumn, continuationID string) (GateResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, gateCheckCall{
		SessionID:      sessionID,
		From:           from,
		To:             to,
		ContinuationID: continuationID,
	})
	if m.err != nil {
		return GateResult{}, m.err
	}
	key := fmt.Sprintf("%s->%s", from, to)
	if r, ok := m.results[key]; ok {
		return r, nil
	}
	return GateResult{Pass: true, Models: map[string]string{"default": "pass"}}, nil
}

func (m *mockGateChecker) getCalls() []gateCheckCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	calls := make([]gateCheckCall, len(m.calls))
	copy(calls, m.calls)
	return calls
}

// --- Constructor Tests ---

func TestConductor_NewConductor(t *testing.T) {
	t.Run("default values", func(t *testing.T) {
		c := NewConductor()

		assert.Equal(t, ConductorIdle, c.State())
		assert.Empty(t, c.SessionID)
		assert.Empty(t, c.GroupPath)
		assert.Zero(t, c.retryCount)
	})

	t.Run("with options", func(t *testing.T) {
		eng := newMockTransitionEngine()
		gate := newMockGateChecker()
		var statusMsgs []ConductorStatusMsg
		cb := func(msg ConductorStatusMsg) {
			statusMsgs = append(statusMsgs, msg)
		}

		cfg := session.YOLOConfig{
			ConsensusModels: []string{"model-a", "model-b"},
			PassThreshold:   0.8,
			PauseOnFail:     true,
			MaxRetries:      3,
			SkipColumns:     []string{"backlog"},
			ContinuationID:  "cont-123",
		}

		c := NewConductor(
			WithSessionID("sess-1"),
			WithGroupPath("/projects/app"),
			WithYOLOConfig(cfg),
			WithTransitionEngine(eng),
			WithGateChecker(gate),
			WithStatusCallback(cb),
			WithLogPath("/tmp/test-conductor.md"),
			WithCurrentColumn(session.KanbanDesign),
		)

		assert.Equal(t, "sess-1", c.SessionID)
		assert.Equal(t, "/projects/app", c.GroupPath)
		assert.Equal(t, cfg, c.config)
		assert.Equal(t, session.KanbanDesign, c.currentColumn)
		assert.Equal(t, "/tmp/test-conductor.md", c.logPath)
		assert.Equal(t, ConductorIdle, c.State())
		assert.NotNil(t, c.engine)
		assert.NotNil(t, c.gate)
	})
}

// --- State Transition Tests ---

func TestConductor_StateTransitions(t *testing.T) {
	t.Run("idle to running", func(t *testing.T) {
		c := NewConductor(WithSessionID("s1"))

		assert.Equal(t, ConductorIdle, c.State())
		c.Start()
		assert.Equal(t, ConductorRunning, c.State())
	})

	t.Run("running to paused", func(t *testing.T) {
		c := NewConductor(WithSessionID("s1"))
		c.Start()

		c.Pause()
		assert.Equal(t, ConductorPaused, c.State())
	})

	t.Run("paused to running via resume", func(t *testing.T) {
		c := NewConductor(WithSessionID("s1"))
		c.Start()
		c.Pause()

		c.Resume()
		assert.Equal(t, ConductorRunning, c.State())
	})

	t.Run("running to idle via stop", func(t *testing.T) {
		c := NewConductor(WithSessionID("s1"))
		c.Start()

		c.Stop()
		assert.Equal(t, ConductorIdle, c.State())
	})

	t.Run("paused to idle via stop", func(t *testing.T) {
		c := NewConductor(WithSessionID("s1"))
		c.Start()
		c.Pause()

		c.Stop()
		assert.Equal(t, ConductorIdle, c.State())
	})

	t.Run("concurrent state access is safe", func(t *testing.T) {
		c := NewConductor(WithSessionID("s1"))
		var wg sync.WaitGroup

		for range 100 {
			wg.Add(3)
			go func() {
				defer wg.Done()
				c.Start()
			}()
			go func() {
				defer wg.Done()
				_ = c.State()
			}()
			go func() {
				defer wg.Done()
				c.Stop()
			}()
		}
		wg.Wait()
		// If we reach here without data race, the test passes
	})
}

// --- NextColumn Tests ---

func TestConductor_NextColumn(t *testing.T) {
	t.Run("backlog to design", func(t *testing.T) {
		c := NewConductor()
		next, ok := c.NextColumn(session.KanbanBacklog)
		assert.True(t, ok)
		assert.Equal(t, session.KanbanDesign, next)
	})

	t.Run("review to done", func(t *testing.T) {
		c := NewConductor()
		next, ok := c.NextColumn(session.KanbanReview)
		assert.True(t, ok)
		assert.Equal(t, session.KanbanDone, next)
	})

	t.Run("done has no next", func(t *testing.T) {
		c := NewConductor()
		_, ok := c.NextColumn(session.KanbanDone)
		assert.False(t, ok)
	})

	t.Run("skip configured columns", func(t *testing.T) {
		c := NewConductor(WithYOLOConfig(session.YOLOConfig{
			SkipColumns: []string{"design", "plan"},
		}))
		next, ok := c.NextColumn(session.KanbanBacklog)
		assert.True(t, ok)
		assert.Equal(t, session.KanbanImplement, next)
	})

	t.Run("skip all remaining returns false", func(t *testing.T) {
		c := NewConductor(WithYOLOConfig(session.YOLOConfig{
			SkipColumns: []string{"done"},
		}))
		_, ok := c.NextColumn(session.KanbanReview)
		assert.False(t, ok)
	})

	t.Run("skip does not affect current column", func(t *testing.T) {
		c := NewConductor(WithYOLOConfig(session.YOLOConfig{
			SkipColumns: []string{"design"},
		}))
		// From backlog, design is skipped, so next is plan
		next, ok := c.NextColumn(session.KanbanBacklog)
		assert.True(t, ok)
		assert.Equal(t, session.KanbanPlan, next)
	})
}

// --- ShouldTerminate Tests ---

func TestConductor_ShouldTerminate(t *testing.T) {
	t.Run("terminates at done", func(t *testing.T) {
		c := NewConductor(
			WithCurrentColumn(session.KanbanDone),
		)
		assert.True(t, c.ShouldTerminate())
	})

	t.Run("terminates at max retries", func(t *testing.T) {
		c := NewConductor(
			WithCurrentColumn(session.KanbanImplement),
			WithYOLOConfig(session.YOLOConfig{MaxRetries: 3}),
		)
		c.retryCount = 3
		assert.True(t, c.ShouldTerminate())
	})

	t.Run("does not terminate below max retries", func(t *testing.T) {
		c := NewConductor(
			WithCurrentColumn(session.KanbanImplement),
			WithYOLOConfig(session.YOLOConfig{MaxRetries: 3}),
		)
		c.retryCount = 2
		assert.False(t, c.ShouldTerminate())
	})

	t.Run("does not terminate when idle at backlog", func(t *testing.T) {
		c := NewConductor(
			WithCurrentColumn(session.KanbanBacklog),
		)
		assert.False(t, c.ShouldTerminate())
	})

	t.Run("max retries zero means no limit", func(t *testing.T) {
		c := NewConductor(
			WithCurrentColumn(session.KanbanImplement),
			WithYOLOConfig(session.YOLOConfig{MaxRetries: 0}),
		)
		c.retryCount = 100
		assert.False(t, c.ShouldTerminate())
	})
}

// --- AdvanceColumn Tests ---

func TestConductor_AdvanceColumn(t *testing.T) {
	t.Run("advances column via transition engine", func(t *testing.T) {
		eng := newMockTransitionEngine()
		c := NewConductor(
			WithSessionID("s1"),
			WithGroupPath("/project"),
			WithCurrentColumn(session.KanbanBacklog),
			WithTransitionEngine(eng),
		)

		err := c.AdvanceColumn(session.KanbanDesign)
		require.NoError(t, err)
		assert.Equal(t, session.KanbanDesign, c.currentColumn)

		calls := eng.getMoveCalls()
		require.Len(t, calls, 1)
		assert.Equal(t, "s1", calls[0].SessionID)
		assert.Equal(t, session.KanbanBacklog, calls[0].FromColumn)
		assert.Equal(t, session.KanbanDesign, calls[0].ToColumn)
		assert.True(t, calls[0].ForceConfirm)
	})

	t.Run("does not update column on move failure", func(t *testing.T) {
		eng := newMockTransitionEngine()
		eng.moveResults["backlog->design"] = MoveResult{
			Success: false,
			Error:   errors.New("storage unavailable"),
		}

		c := NewConductor(
			WithSessionID("s1"),
			WithGroupPath("/project"),
			WithCurrentColumn(session.KanbanBacklog),
			WithTransitionEngine(eng),
		)

		err := c.AdvanceColumn(session.KanbanDesign)
		require.Error(t, err)
		assert.Equal(t, session.KanbanBacklog, c.currentColumn)
	})

	t.Run("resets retry count on success", func(t *testing.T) {
		eng := newMockTransitionEngine()
		c := NewConductor(
			WithSessionID("s1"),
			WithGroupPath("/project"),
			WithCurrentColumn(session.KanbanBacklog),
			WithTransitionEngine(eng),
		)
		c.retryCount = 2

		err := c.AdvanceColumn(session.KanbanDesign)
		require.NoError(t, err)
		assert.Equal(t, 0, c.retryCount)
	})
}

// --- HandleSkillCompletion Tests ---

func TestConductor_HandleSkillCompletion(t *testing.T) {
	t.Run("success increments no retries", func(t *testing.T) {
		c := NewConductor(
			WithCurrentColumn(session.KanbanBacklog),
		)

		action := c.HandleSkillCompletion(SkillCompletedMsg{
			SessionID: "s1",
			Success:   true,
		})

		assert.Equal(t, SkillActionAdvance, action)
		assert.Equal(t, 0, c.retryCount)
	})

	t.Run("failure with pause on fail", func(t *testing.T) {
		c := NewConductor(
			WithCurrentColumn(session.KanbanBacklog),
			WithYOLOConfig(session.YOLOConfig{PauseOnFail: true, MaxRetries: 3}),
		)

		action := c.HandleSkillCompletion(SkillCompletedMsg{
			SessionID: "s1",
			Success:   false,
			Error:     errors.New("skill failed"),
		})

		assert.Equal(t, SkillActionPause, action)
		assert.Equal(t, 1, c.retryCount)
	})

	t.Run("failure without pause retries", func(t *testing.T) {
		c := NewConductor(
			WithCurrentColumn(session.KanbanBacklog),
			WithYOLOConfig(session.YOLOConfig{PauseOnFail: false, MaxRetries: 3}),
		)

		action := c.HandleSkillCompletion(SkillCompletedMsg{
			SessionID: "s1",
			Success:   false,
		})

		assert.Equal(t, SkillActionRetry, action)
		assert.Equal(t, 1, c.retryCount)
	})

	t.Run("failure at max retries terminates", func(t *testing.T) {
		c := NewConductor(
			WithCurrentColumn(session.KanbanBacklog),
			WithYOLOConfig(session.YOLOConfig{PauseOnFail: false, MaxRetries: 2}),
		)
		c.retryCount = 1

		action := c.HandleSkillCompletion(SkillCompletedMsg{
			SessionID: "s1",
			Success:   false,
		})

		assert.Equal(t, SkillActionTerminate, action)
		assert.Equal(t, 2, c.retryCount)
	})
}

// --- HandleGateResult Tests ---

func TestConductor_HandleGateResult(t *testing.T) {
	t.Run("pass advances", func(t *testing.T) {
		c := NewConductor(
			WithCurrentColumn(session.KanbanBacklog),
		)

		result := GateResult{
			Pass:   true,
			Models: map[string]string{"model-a": "pass"},
		}

		action := c.HandleGateResult(result)
		assert.Equal(t, GateActionAdvance, action)
		assert.Equal(t, 0, c.retryCount)
	})

	t.Run("fail with pause on fail", func(t *testing.T) {
		c := NewConductor(
			WithCurrentColumn(session.KanbanBacklog),
			WithYOLOConfig(session.YOLOConfig{PauseOnFail: true, MaxRetries: 3}),
		)

		result := GateResult{
			Pass:   false,
			Models: map[string]string{"model-a": "fail"},
		}

		action := c.HandleGateResult(result)
		assert.Equal(t, GateActionPause, action)
		assert.Equal(t, 1, c.retryCount)
	})

	t.Run("fail without pause retries", func(t *testing.T) {
		c := NewConductor(
			WithCurrentColumn(session.KanbanBacklog),
			WithYOLOConfig(session.YOLOConfig{PauseOnFail: false, MaxRetries: 3}),
		)

		result := GateResult{
			Pass:   false,
			Models: map[string]string{"model-a": "fail"},
		}

		action := c.HandleGateResult(result)
		assert.Equal(t, GateActionRetry, action)
		assert.Equal(t, 1, c.retryCount)
	})

	t.Run("fail at max retries terminates", func(t *testing.T) {
		c := NewConductor(
			WithCurrentColumn(session.KanbanBacklog),
			WithYOLOConfig(session.YOLOConfig{PauseOnFail: false, MaxRetries: 2}),
		)
		c.retryCount = 1

		result := GateResult{Pass: false}

		action := c.HandleGateResult(result)
		assert.Equal(t, GateActionTerminate, action)
		assert.Equal(t, 2, c.retryCount)
	})
}

// --- MaxRetries Tests ---

func TestConductor_MaxRetries(t *testing.T) {
	t.Run("stops after MaxRetries exceeded", func(t *testing.T) {
		c := NewConductor(
			WithSessionID("s1"),
			WithCurrentColumn(session.KanbanImplement),
			WithYOLOConfig(session.YOLOConfig{MaxRetries: 2}),
		)
		c.retryCount = 2

		assert.True(t, c.ShouldTerminate())
	})

	t.Run("retry count tracks failures", func(t *testing.T) {
		c := NewConductor(
			WithCurrentColumn(session.KanbanBacklog),
			WithYOLOConfig(session.YOLOConfig{MaxRetries: 5}),
		)

		for i := range 5 {
			_ = c.HandleSkillCompletion(SkillCompletedMsg{
				SessionID: "s1",
				Success:   false,
			})
			assert.Equal(t, i+1, c.retryCount)
		}

		assert.True(t, c.ShouldTerminate())
	})

	t.Run("advance resets retry count", func(t *testing.T) {
		eng := newMockTransitionEngine()
		c := NewConductor(
			WithSessionID("s1"),
			WithGroupPath("/project"),
			WithCurrentColumn(session.KanbanBacklog),
			WithTransitionEngine(eng),
			WithYOLOConfig(session.YOLOConfig{MaxRetries: 3}),
		)
		c.retryCount = 2

		err := c.AdvanceColumn(session.KanbanDesign)
		require.NoError(t, err)
		assert.Equal(t, 0, c.retryCount)
	})
}

// --- SkipColumns Tests ---

func TestConductor_SkipColumns(t *testing.T) {
	t.Run("skips single column", func(t *testing.T) {
		c := NewConductor(WithYOLOConfig(session.YOLOConfig{
			SkipColumns: []string{"design"},
		}))

		next, ok := c.NextColumn(session.KanbanBacklog)
		assert.True(t, ok)
		assert.Equal(t, session.KanbanPlan, next)
	})

	t.Run("skips multiple consecutive columns", func(t *testing.T) {
		c := NewConductor(WithYOLOConfig(session.YOLOConfig{
			SkipColumns: []string{"design", "plan", "implement"},
		}))

		next, ok := c.NextColumn(session.KanbanBacklog)
		assert.True(t, ok)
		assert.Equal(t, session.KanbanReview, next)
	})

	t.Run("isSkippedColumn returns correct values", func(t *testing.T) {
		c := NewConductor(WithYOLOConfig(session.YOLOConfig{
			SkipColumns: []string{"design", "review"},
		}))

		assert.True(t, c.isSkippedColumn(session.KanbanDesign))
		assert.True(t, c.isSkippedColumn(session.KanbanReview))
		assert.False(t, c.isSkippedColumn(session.KanbanBacklog))
		assert.False(t, c.isSkippedColumn(session.KanbanPlan))
		assert.False(t, c.isSkippedColumn(session.KanbanImplement))
		assert.False(t, c.isSkippedColumn(session.KanbanDone))
	})
}

// --- StatusMessage Tests ---

func TestConductor_StatusMessage(t *testing.T) {
	t.Run("produces correct status message", func(t *testing.T) {
		c := NewConductor(
			WithSessionID("s1"),
			WithCurrentColumn(session.KanbanDesign),
		)

		msg := c.StatusMessage()
		assert.Equal(t, "s1", msg.SessionID)
		assert.Equal(t, session.KanbanDesign, msg.Column)
	})

	t.Run("status callback is invoked", func(t *testing.T) {
		var received []ConductorStatusMsg
		cb := func(msg ConductorStatusMsg) {
			received = append(received, msg)
		}

		c := NewConductor(
			WithSessionID("s1"),
			WithCurrentColumn(session.KanbanDesign),
			WithStatusCallback(cb),
		)

		c.SendStatus(nil)
		require.Len(t, received, 1)
		assert.Equal(t, "s1", received[0].SessionID)
		assert.Equal(t, session.KanbanDesign, received[0].Column)
	})

	t.Run("status callback with gate status", func(t *testing.T) {
		var received []ConductorStatusMsg
		cb := func(msg ConductorStatusMsg) {
			received = append(received, msg)
		}

		c := NewConductor(
			WithSessionID("s1"),
			WithCurrentColumn(session.KanbanImplement),
			WithStatusCallback(cb),
		)

		gateStatus := map[string]string{"model-a": "pass", "model-b": "fail"}
		c.SendStatus(gateStatus)
		require.Len(t, received, 1)
		assert.Equal(t, gateStatus, received[0].GateStatus)
	})
}

// --- LogEntry Tests ---

func TestConductor_LogEntry(t *testing.T) {
	t.Run("writes gate decision to log", func(t *testing.T) {
		dir := t.TempDir()
		logPath := filepath.Join(dir, "conductor-log.md")

		c := NewConductor(
			WithSessionID("s1"),
			WithLogPath(logPath),
		)

		entry := ConductorLogEntry{
			Timestamp:  time.Date(2026, 2, 28, 12, 0, 0, 0, time.UTC),
			SessionID:  "s1",
			FromColumn: session.KanbanDesign,
			ToColumn:   session.KanbanPlan,
			GateResult: "pass",
			Models:     map[string]string{"model-a": "pass", "model-b": "pass"},
			Notes:      "all models agreed",
		}

		err := c.appendLogEntry(entry)
		require.NoError(t, err)

		data, err := os.ReadFile(logPath)
		require.NoError(t, err)

		content := string(data)
		assert.Contains(t, content, "s1")
		assert.Contains(t, content, "design")
		assert.Contains(t, content, "plan")
		assert.Contains(t, content, "pass")
		assert.Contains(t, content, "all models agreed")
	})

	t.Run("appends multiple entries", func(t *testing.T) {
		dir := t.TempDir()
		logPath := filepath.Join(dir, "conductor-log.md")

		c := NewConductor(
			WithSessionID("s1"),
			WithLogPath(logPath),
		)

		for i := range 3 {
			entry := ConductorLogEntry{
				Timestamp:  time.Date(2026, 2, 28, 12, i, 0, 0, time.UTC),
				SessionID:  "s1",
				FromColumn: session.KanbanDesign,
				ToColumn:   session.KanbanPlan,
				GateResult: "pass",
			}
			err := c.appendLogEntry(entry)
			require.NoError(t, err)
		}

		data, err := os.ReadFile(logPath)
		require.NoError(t, err)

		// Count entry markers
		content := string(data)
		count := strings.Count(content, "## Gate Decision")
		assert.Equal(t, 3, count)
	})

	t.Run("creates parent directory if needed", func(t *testing.T) {
		dir := t.TempDir()
		logPath := filepath.Join(dir, "nested", "dir", "conductor-log.md")

		c := NewConductor(
			WithSessionID("s1"),
			WithLogPath(logPath),
		)

		entry := ConductorLogEntry{
			Timestamp:  time.Date(2026, 2, 28, 12, 0, 0, 0, time.UTC),
			SessionID:  "s1",
			FromColumn: session.KanbanBacklog,
			ToColumn:   session.KanbanDesign,
			GateResult: "pass",
		}

		err := c.appendLogEntry(entry)
		require.NoError(t, err)

		_, err = os.Stat(logPath)
		assert.NoError(t, err)
	})

	t.Run("noop when log path is empty", func(t *testing.T) {
		c := NewConductor(WithSessionID("s1"))

		entry := ConductorLogEntry{
			Timestamp:  time.Date(2026, 2, 28, 12, 0, 0, 0, time.UTC),
			SessionID:  "s1",
			FromColumn: session.KanbanBacklog,
			ToColumn:   session.KanbanDesign,
			GateResult: "pass",
		}

		err := c.appendLogEntry(entry)
		assert.NoError(t, err)
	})
}

// --- RunLoop Tests ---

func TestConductor_RunLoop(t *testing.T) {
	t.Run("advances from backlog to done", func(t *testing.T) {
		eng := newMockTransitionEngine()
		gate := newMockGateChecker()
		var statuses []ConductorStatusMsg
		var mu sync.Mutex
		cb := func(msg ConductorStatusMsg) {
			mu.Lock()
			statuses = append(statuses, msg)
			mu.Unlock()
		}

		dir := t.TempDir()
		logPath := filepath.Join(dir, "conductor-log.md")

		c := NewConductor(
			WithSessionID("s1"),
			WithGroupPath("/project"),
			WithCurrentColumn(session.KanbanBacklog),
			WithTransitionEngine(eng),
			WithGateChecker(gate),
			WithStatusCallback(cb),
			WithLogPath(logPath),
			WithYOLOConfig(session.YOLOConfig{MaxRetries: 3}),
		)

		// Supply skill completion results for each column
		completionCh := make(chan SkillCompletedMsg, 10)
		for _, col := range kanbanColumnsOrdered {
			if col == session.KanbanDone {
				continue
			}
			completionCh <- SkillCompletedMsg{
				SessionID: "s1",
				SkillName: "skill-" + col.String(),
				Success:   true,
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := c.RunLoop(ctx, completionCh)
		require.NoError(t, err)

		assert.Equal(t, session.KanbanDone, c.currentColumn)

		// Verify moves were requested for each transition
		calls := eng.getMoveCalls()
		assert.GreaterOrEqual(t, len(calls), 4) // backlog->design, design->plan, plan->impl, impl->review, review->done
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		eng := newMockTransitionEngine()
		gate := newMockGateChecker()

		c := NewConductor(
			WithSessionID("s1"),
			WithGroupPath("/project"),
			WithCurrentColumn(session.KanbanBacklog),
			WithTransitionEngine(eng),
			WithGateChecker(gate),
		)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		completionCh := make(chan SkillCompletedMsg, 1)
		err := c.RunLoop(ctx, completionCh)
		assert.ErrorIs(t, err, context.Canceled)
	})

	t.Run("handles skill failure with pause", func(t *testing.T) {
		eng := newMockTransitionEngine()
		gate := newMockGateChecker()

		c := NewConductor(
			WithSessionID("s1"),
			WithGroupPath("/project"),
			WithCurrentColumn(session.KanbanBacklog),
			WithTransitionEngine(eng),
			WithGateChecker(gate),
			WithYOLOConfig(session.YOLOConfig{
				PauseOnFail: true,
				MaxRetries:  3,
			}),
		)

		completionCh := make(chan SkillCompletedMsg, 1)
		completionCh <- SkillCompletedMsg{
			SessionID: "s1",
			Success:   false,
			Error:     errors.New("skill crashed"),
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		err := c.RunLoop(ctx, completionCh)
		// Should pause without error
		require.NoError(t, err)
		assert.Equal(t, ConductorPaused, c.State())
	})

	t.Run("handles gate failure with retry", func(t *testing.T) {
		eng := newMockTransitionEngine()
		gate := newMockGateChecker()
		gate.results["backlog->design"] = GateResult{
			Pass:   false,
			Models: map[string]string{"model-a": "fail"},
		}

		c := NewConductor(
			WithSessionID("s1"),
			WithGroupPath("/project"),
			WithCurrentColumn(session.KanbanBacklog),
			WithTransitionEngine(eng),
			WithGateChecker(gate),
			WithYOLOConfig(session.YOLOConfig{
				PauseOnFail: true,
				MaxRetries:  3,
			}),
		)

		completionCh := make(chan SkillCompletedMsg, 1)
		completionCh <- SkillCompletedMsg{
			SessionID: "s1",
			SkillName: "skill-backlog",
			Success:   true,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		err := c.RunLoop(ctx, completionCh)
		require.NoError(t, err)
		assert.Equal(t, ConductorPaused, c.State())
	})

	t.Run("skips configured columns", func(t *testing.T) {
		eng := newMockTransitionEngine()
		gate := newMockGateChecker()

		c := NewConductor(
			WithSessionID("s1"),
			WithGroupPath("/project"),
			WithCurrentColumn(session.KanbanBacklog),
			WithTransitionEngine(eng),
			WithGateChecker(gate),
			WithYOLOConfig(session.YOLOConfig{
				SkipColumns: []string{"design", "plan"},
				MaxRetries:  3,
			}),
		)

		// Only need completions for: backlog, implement, review (design and plan skipped)
		completionCh := make(chan SkillCompletedMsg, 10)
		completionCh <- SkillCompletedMsg{SessionID: "s1", SkillName: "skill-backlog", Success: true}
		completionCh <- SkillCompletedMsg{SessionID: "s1", SkillName: "skill-implement", Success: true}
		completionCh <- SkillCompletedMsg{SessionID: "s1", SkillName: "skill-review", Success: true}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := c.RunLoop(ctx, completionCh)
		require.NoError(t, err)
		assert.Equal(t, session.KanbanDone, c.currentColumn)

		// Should have moved: backlog->implement, implement->review, review->done
		calls := eng.getMoveCalls()
		require.Len(t, calls, 3)
		assert.Equal(t, session.KanbanImplement, calls[0].ToColumn) // skip design+plan
		assert.Equal(t, session.KanbanReview, calls[1].ToColumn)
		assert.Equal(t, session.KanbanDone, calls[2].ToColumn)
	})

	t.Run("terminates when max retries exceeded", func(t *testing.T) {
		eng := newMockTransitionEngine()
		gate := newMockGateChecker()

		c := NewConductor(
			WithSessionID("s1"),
			WithGroupPath("/project"),
			WithCurrentColumn(session.KanbanBacklog),
			WithTransitionEngine(eng),
			WithGateChecker(gate),
			WithYOLOConfig(session.YOLOConfig{
				PauseOnFail: false,
				MaxRetries:  2,
			}),
		)

		completionCh := make(chan SkillCompletedMsg, 5)
		// Send 2 failures (will hit max retries)
		completionCh <- SkillCompletedMsg{SessionID: "s1", Success: false}
		completionCh <- SkillCompletedMsg{SessionID: "s1", Success: false}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := c.RunLoop(ctx, completionCh)
		require.NoError(t, err)
		assert.Equal(t, ConductorIdle, c.State())
		assert.Equal(t, session.KanbanBacklog, c.currentColumn) // never advanced
	})
}

// --- ConductorLogEntry Format Tests ---

func TestConductorLogEntry_Format(t *testing.T) {
	entry := ConductorLogEntry{
		Timestamp:  time.Date(2026, 2, 28, 14, 30, 0, 0, time.UTC),
		SessionID:  "session-abc",
		FromColumn: session.KanbanPlan,
		ToColumn:   session.KanbanImplement,
		GateResult: "pass",
		Models:     map[string]string{"gemini": "pass", "gpt": "pass"},
		Notes:      "consensus reached",
	}

	formatted := formatLogEntry(entry)
	assert.Contains(t, formatted, "## Gate Decision")
	assert.Contains(t, formatted, "2026-02-28")
	assert.Contains(t, formatted, "session-abc")
	assert.Contains(t, formatted, "plan")
	assert.Contains(t, formatted, "implement")
	assert.Contains(t, formatted, "pass")
	assert.Contains(t, formatted, "consensus reached")
}

// --- Verify GateResult struct ---

func TestGateResult(t *testing.T) {
	t.Run("pass result", func(t *testing.T) {
		r := GateResult{
			Pass:           true,
			Models:         map[string]string{"a": "pass", "b": "pass"},
			ContinuationID: "cid-1",
		}
		assert.True(t, r.Pass)
		assert.Equal(t, "cid-1", r.ContinuationID)
		assert.Len(t, r.Models, 2)
	})

	t.Run("fail result", func(t *testing.T) {
		r := GateResult{
			Pass:   false,
			Models: map[string]string{"a": "pass", "b": "fail"},
		}
		assert.False(t, r.Pass)
	})
}

