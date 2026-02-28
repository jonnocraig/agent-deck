package ui

// kanban_conductor.go — Kanban conductor orchestration for YOLO autonomous mode.
//
// The Conductor manages a single session's lifecycle through the kanban board
// autonomously. It resolves skills, triggers them, waits for completion, runs
// consensus gates, and advances columns.
//
// State machine: Idle -> Running -> Paused (or back to Idle on Done/disable/max retries)
//
// The run loop is a blocking method; the caller decides whether to run it in a
// goroutine. The GateChecker interface is the seam for consensus gate logic
// (implemented separately in task 6.2).

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/asheshgoplani/agent-deck/internal/logging"
	"github.com/asheshgoplani/agent-deck/internal/session"
)

var conductorLog = logging.ForComponent("conductor")

// ConductorState represents the current state of a conductor instance.
type ConductorState int

const (
	ConductorIdle    ConductorState = iota // Not actively orchestrating
	ConductorRunning                       // Actively managing a session
	ConductorPaused                        // Paused by user or system
)

// String returns a human-readable label for the conductor state.
func (s ConductorState) String() string {
	switch s {
	case ConductorIdle:
		return "idle"
	case ConductorRunning:
		return "running"
	case ConductorPaused:
		return "paused"
	default:
		return "unknown"
	}
}

// --- Action Types ---

// SkillAction indicates what the conductor should do after a skill completion.
type SkillAction int

const (
	SkillActionAdvance   SkillAction = iota // Skill succeeded, proceed to gate
	SkillActionRetry                        // Skill failed, retry
	SkillActionPause                        // Skill failed, pause conductor
	SkillActionTerminate                    // Max retries exceeded, stop
)

// GateAction indicates what the conductor should do after a gate check.
type GateAction int

const (
	GateActionAdvance   GateAction = iota // Gate passed, advance column
	GateActionRetry                       // Gate failed, retry skill
	GateActionPause                       // Gate failed, pause conductor
	GateActionTerminate                   // Max retries exceeded, stop
)

// --- GateChecker Interface ---

// GateChecker validates whether a session should advance from one column to the next.
// The real implementation uses zen MCP consensus (task 6.2).
// A pass-through implementation is used by default.
type GateChecker interface {
	CheckGate(ctx context.Context, sessionID string, from, to session.KanbanColumn, continuationID string) (GateResult, error)
}

// GateResult holds the outcome of a consensus gate check.
type GateResult struct {
	Pass             bool
	Models           map[string]string  // model name -> verdict ("pass"/"fail")
	ContinuationID   string             // updated continuation ID for context revival
	ConsensusRequest *ConsensusParams   // populated by ZenGateChecker for UI to execute
}

// --- Log Entry ---

// ConductorLogEntry represents a single gate decision log entry.
type ConductorLogEntry struct {
	Timestamp  time.Time
	SessionID  string
	FromColumn session.KanbanColumn
	ToColumn   session.KanbanColumn
	GateResult string
	Models     map[string]string
	Notes      string
}

// --- Conductor ---

// Conductor holds orchestration context for a single managed session.
// It drives a session through the kanban board autonomously in YOLO mode.
type Conductor struct {
	// SessionID is the unique identifier of the managed session.
	SessionID string

	// GroupPath is the group the session belongs to (for skill resolution).
	GroupPath string

	mu            sync.Mutex
	state         ConductorState
	currentColumn session.KanbanColumn
	retryCount    int
	config        session.YOLOConfig
	engine        TransitionEngine
	gate          GateChecker
	statusCb      func(ConductorStatusMsg)
	logPath       string
}

// --- Functional Options ---

// ConductorOption configures a Conductor.
type ConductorOption func(*Conductor)

// WithSessionID sets the session ID for the conductor.
func WithSessionID(id string) ConductorOption {
	return func(c *Conductor) { c.SessionID = id }
}

// WithGroupPath sets the group path for skill resolution.
func WithGroupPath(path string) ConductorOption {
	return func(c *Conductor) { c.GroupPath = path }
}

// WithYOLOConfig sets the YOLO automation configuration.
func WithYOLOConfig(cfg session.YOLOConfig) ConductorOption {
	return func(c *Conductor) { c.config = cfg }
}

// WithTransitionEngine sets the transition engine for column moves.
func WithTransitionEngine(eng TransitionEngine) ConductorOption {
	return func(c *Conductor) { c.engine = eng }
}

// WithGateChecker sets the gate checker for consensus validation.
func WithGateChecker(gate GateChecker) ConductorOption {
	return func(c *Conductor) { c.gate = gate }
}

// WithStatusCallback sets the callback for status updates to the UI.
func WithStatusCallback(cb func(ConductorStatusMsg)) ConductorOption {
	return func(c *Conductor) { c.statusCb = cb }
}

// WithLogPath sets the file path for conductor log entries.
func WithLogPath(path string) ConductorOption {
	return func(c *Conductor) { c.logPath = path }
}

// WithCurrentColumn sets the starting column for the conductor.
func WithCurrentColumn(col session.KanbanColumn) ConductorOption {
	return func(c *Conductor) { c.currentColumn = col }
}

// NewConductor creates a new Conductor with the given options.
// The conductor starts in the Idle state.
func NewConductor(opts ...ConductorOption) *Conductor {
	c := &Conductor{
		state: ConductorIdle,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// --- State Machine ---

// State returns the current conductor state (thread-safe).
func (c *Conductor) State() ConductorState {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state
}

// Start transitions the conductor to the Running state.
func (c *Conductor) Start() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state = ConductorRunning
	conductorLog.Info("conductor_started",
		slog.String("session", c.SessionID),
		slog.String("column", string(c.currentColumn)),
	)
}

// Pause transitions the conductor to the Paused state.
func (c *Conductor) Pause() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state = ConductorPaused
	conductorLog.Info("conductor_paused",
		slog.String("session", c.SessionID),
		slog.String("column", string(c.currentColumn)),
	)
}

// Resume transitions the conductor from Paused to Running.
func (c *Conductor) Resume() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state = ConductorRunning
	conductorLog.Info("conductor_resumed",
		slog.String("session", c.SessionID),
		slog.String("column", string(c.currentColumn)),
	)
}

// Stop transitions the conductor to the Idle state and resets retry count.
func (c *Conductor) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state = ConductorIdle
	c.retryCount = 0
	conductorLog.Info("conductor_stopped",
		slog.String("session", c.SessionID),
	)
}

// --- Column Navigation ---

// NextColumn returns the next column after current, skipping any columns in
// the SkipColumns config. Returns false if Done is reached or all remaining
// columns are skipped.
func (c *Conductor) NextColumn(current session.KanbanColumn) (session.KanbanColumn, bool) {
	idx := current.Index()
	if idx < 0 {
		return "", false
	}

	for next := idx + 1; next < len(kanbanColumnsOrdered); next++ {
		col := kanbanColumnsOrdered[next]
		if !c.isSkippedColumn(col) {
			return col, true
		}
	}
	return "", false
}

// isSkippedColumn checks whether a column is in the SkipColumns configuration.
func (c *Conductor) isSkippedColumn(col session.KanbanColumn) bool {
	for _, skip := range c.config.SkipColumns {
		if skip == col.String() {
			return true
		}
	}
	return false
}

// --- Thread-Safe Getters ---

// CurrentColumn returns the current column (thread-safe).
func (c *Conductor) CurrentColumn() session.KanbanColumn {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.currentColumn
}

// --- Termination Check ---

// ShouldTerminate returns true if the conductor should stop running.
// This happens when Done is reached or max retries are exceeded. Thread-safe.
func (c *Conductor) ShouldTerminate() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.currentColumn == session.KanbanDone {
		return true
	}
	if c.config.MaxRetries > 0 && c.retryCount >= c.config.MaxRetries {
		return true
	}
	return false
}

// --- Column Advancement ---

// AdvanceColumn moves the session to the target column via the transition engine.
// On success, updates currentColumn and resets retryCount.
// ForceConfirm is set to true since the conductor is autonomous.
func (c *Conductor) AdvanceColumn(target session.KanbanColumn) error {
	if c.engine == nil {
		return fmt.Errorf("advance column: transition engine not configured")
	}

	from := c.currentColumn
	result := c.engine.RequestMove(MoveRequest{
		SessionID:    c.SessionID,
		FromColumn:   from,
		ToColumn:     target,
		GroupPath:    c.GroupPath,
		ForceConfirm: true,
	})

	if !result.Success {
		err := result.Error
		if err == nil {
			err = fmt.Errorf("move from %s to %s failed", from, target)
		}
		return fmt.Errorf("advance column %s->%s: %w", from, target, err)
	}

	c.mu.Lock()
	c.currentColumn = target
	c.retryCount = 0
	c.mu.Unlock()

	conductorLog.Info("column_advanced",
		slog.String("session", c.SessionID),
		slog.String("from", from.String()),
		slog.String("to", target.String()),
	)
	return nil
}

// --- Skill Completion Handling ---

// HandleSkillCompletion evaluates a skill completion message and returns
// the appropriate action for the conductor to take.
func (c *Conductor) HandleSkillCompletion(msg SkillCompletedMsg) SkillAction {
	if msg.Success {
		return SkillActionAdvance
	}

	c.mu.Lock()
	c.retryCount++
	count := c.retryCount
	maxRetries := c.config.MaxRetries
	pauseOnFail := c.config.PauseOnFail
	c.mu.Unlock()

	conductorLog.Warn("skill_failed_conductor",
		slog.String("session", msg.SessionID),
		slog.String("skill", msg.SkillName),
		slog.Int("retry_count", count),
		slog.Int("max_retries", maxRetries),
	)

	if maxRetries > 0 && count >= maxRetries {
		return SkillActionTerminate
	}
	if pauseOnFail {
		return SkillActionPause
	}
	return SkillActionRetry
}

// --- Gate Result Handling ---

// HandleGateResult evaluates a gate check result and returns the appropriate
// action for the conductor to take.
func (c *Conductor) HandleGateResult(result GateResult) GateAction {
	if result.Pass {
		return GateActionAdvance
	}

	c.mu.Lock()
	c.retryCount++
	count := c.retryCount
	maxRetries := c.config.MaxRetries
	pauseOnFail := c.config.PauseOnFail
	c.mu.Unlock()

	conductorLog.Warn("gate_failed",
		slog.String("session", c.SessionID),
		slog.Int("retry_count", count),
		slog.Int("max_retries", maxRetries),
	)

	if maxRetries > 0 && count >= maxRetries {
		return GateActionTerminate
	}
	if pauseOnFail {
		return GateActionPause
	}
	return GateActionRetry
}

// --- Status Reporting ---

// StatusMessage creates a ConductorStatusMsg for the current state. Thread-safe.
func (c *Conductor) StatusMessage() ConductorStatusMsg {
	c.mu.Lock()
	col := c.currentColumn
	c.mu.Unlock()
	return ConductorStatusMsg{
		SessionID: c.SessionID,
		Column:    col,
	}
}

// SendStatus sends a status update via the callback, if configured. Thread-safe.
func (c *Conductor) SendStatus(gateStatus map[string]string) {
	if c.statusCb == nil {
		return
	}
	c.mu.Lock()
	col := c.currentColumn
	c.mu.Unlock()
	msg := ConductorStatusMsg{
		SessionID:  c.SessionID,
		Column:     col,
		GateStatus: gateStatus,
	}
	c.statusCb(msg)
}

// ProgressColumns returns which columns are completed, current, and pending
// for rendering the YOLO progress indicator. Thread-safe.
func (c *Conductor) ProgressColumns() (completed []session.KanbanColumn, current session.KanbanColumn, pending []session.KanbanColumn) {
	c.mu.Lock()
	col := c.currentColumn
	c.mu.Unlock()

	currentIdx := col.Index()
	if currentIdx < 0 {
		// Invalid column: treat everything as pending
		return nil, col, append([]session.KanbanColumn(nil), kanbanColumnsOrdered...)
	}

	for i, col := range kanbanColumnsOrdered {
		switch {
		case i < currentIdx:
			completed = append(completed, col)
		case i == currentIdx:
			current = col
		default:
			pending = append(pending, col)
		}
	}
	return completed, current, pending
}

// --- Logging ---

// appendLogEntry writes a gate decision entry to the conductor log file.
// Creates the parent directory if it does not exist.
// Returns nil if logPath is empty (noop).
func (c *Conductor) appendLogEntry(entry ConductorLogEntry) error {
	if c.logPath == "" {
		return nil
	}

	dir := filepath.Dir(c.logPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create log directory %s: %w", dir, err)
	}

	formatted := formatLogEntry(entry)

	f, err := os.OpenFile(c.logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open conductor log %s: %w", c.logPath, err)
	}
	defer f.Close()

	if _, err := f.WriteString(formatted); err != nil {
		return fmt.Errorf("write conductor log entry: %w", err)
	}
	return nil
}

// formatLogEntry formats a ConductorLogEntry as a markdown section.
func formatLogEntry(entry ConductorLogEntry) string {
	var b strings.Builder
	b.WriteString("\n## Gate Decision\n\n")
	b.WriteString(fmt.Sprintf("- **Timestamp:** %s\n", entry.Timestamp.Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("- **Session:** %s\n", sanitizeLogField(entry.SessionID)))
	b.WriteString(fmt.Sprintf("- **Transition:** %s -> %s\n", entry.FromColumn, entry.ToColumn))
	b.WriteString(fmt.Sprintf("- **Result:** %s\n", sanitizeLogField(entry.GateResult)))

	if len(entry.Models) > 0 {
		b.WriteString("- **Models:**\n")
		for model, verdict := range entry.Models {
			b.WriteString(fmt.Sprintf("  - %s: %s\n", sanitizeLogField(model), sanitizeLogField(verdict)))
		}
	}

	if entry.Notes != "" {
		b.WriteString(fmt.Sprintf("- **Notes:** %s\n", sanitizeLogField(entry.Notes)))
	}
	b.WriteString("\n")
	return b.String()
}

// --- Run Loop ---

// RunLoop is the core conductor loop that drives a session through columns.
// It blocks until the session reaches Done, the context is cancelled, or the
// conductor is paused/terminated due to failures.
//
// The completionCh receives SkillCompletedMsg values from the UI layer after
// a skill finishes executing in tmux.
//
// This method sets the conductor to Running on entry and updates state as
// the loop progresses.
func (c *Conductor) RunLoop(ctx context.Context, completionCh <-chan SkillCompletedMsg) error {
	c.Start()
	defer func() {
		if c.State() == ConductorRunning {
			c.Stop()
		}
	}()

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		if c.ShouldTerminate() {
			c.Stop()
			return nil
		}

		if c.State() != ConductorRunning {
			return nil
		}

		// Snapshot current column under lock for use in this iteration
		curCol := c.CurrentColumn()

		// Resolve the next column to target
		nextCol, hasNext := c.NextColumn(curCol)
		if !hasNext {
			// At Done or all remaining columns skipped
			c.Stop()
			return nil
		}

		// Send status update
		c.SendStatus(nil)

		// Step 1: Resolve and report skill for current column
		skill, err := c.resolveCurrentSkill()
		if err != nil {
			conductorLog.Warn("skill_resolve_failed_conductor",
				slog.String("session", c.SessionID),
				slog.String("column", curCol.String()),
				slog.String("error", err.Error()),
			)
			// Continue without skill — just advance
		}

		if skill.SkillName != "" {
			// Wait for skill completion
			completed, err := c.waitForCompletion(ctx, completionCh)
			if err != nil {
				return err
			}

			action := c.HandleSkillCompletion(completed)
			switch action {
			case SkillActionAdvance:
				// Continue to gate check
			case SkillActionRetry:
				c.backoffDelay(ctx)
				continue // retry from top of loop
			case SkillActionPause:
				c.Pause()
				return nil
			case SkillActionTerminate:
				c.Stop()
				return nil
			}
		}

		// Step 2: Run consensus gate
		gateResult, err := c.runGateCheck(ctx, nextCol)
		if err != nil {
			conductorLog.Warn("gate_check_error",
				slog.String("session", c.SessionID),
				slog.String("error", err.Error()),
			)
			c.Pause()
			return nil
		}

		// Log the gate decision
		c.logGateDecision(nextCol, gateResult)

		// Send status with gate results
		c.SendStatus(gateResult.Models)

		gateAction := c.HandleGateResult(gateResult)
		switch gateAction {
		case GateActionAdvance:
			// Advance to next column
		case GateActionRetry:
			c.backoffDelay(ctx)
			continue
		case GateActionPause:
			c.Pause()
			return nil
		case GateActionTerminate:
			c.Stop()
			return nil
		}

		// Step 3: Advance column
		if err := c.AdvanceColumn(nextCol); err != nil {
			conductorLog.Error("advance_column_failed",
				slog.String("session", c.SessionID),
				slog.String("from", curCol.String()),
				slog.String("to", nextCol.String()),
				slog.String("error", err.Error()),
			)
			c.Pause()
			return nil
		}

		// Update continuation ID if gate provided one
		if gateResult.ContinuationID != "" {
			c.mu.Lock()
			c.config.ContinuationID = gateResult.ContinuationID
			c.mu.Unlock()
		}
	}
}

// resolveCurrentSkill looks up the skill mapping for the current column.
func (c *Conductor) resolveCurrentSkill() (SkillMapping, error) {
	if c.engine == nil {
		return SkillMapping{}, fmt.Errorf("transition engine not configured")
	}
	return c.engine.ResolveSkill(c.GroupPath, c.CurrentColumn())
}

// waitForCompletion blocks until a SkillCompletedMsg is received or the
// context is cancelled.
func (c *Conductor) waitForCompletion(ctx context.Context, ch <-chan SkillCompletedMsg) (SkillCompletedMsg, error) {
	select {
	case msg := <-ch:
		return msg, nil
	case <-ctx.Done():
		return SkillCompletedMsg{}, ctx.Err()
	}
}

// runGateCheck executes the consensus gate for a column transition.
// Uses the configured GateChecker, or a pass-through if none is set.
func (c *Conductor) runGateCheck(ctx context.Context, targetCol session.KanbanColumn) (GateResult, error) {
	if c.gate == nil {
		// Default pass-through: always pass
		return GateResult{Pass: true, Models: map[string]string{"default": "pass"}}, nil
	}
	c.mu.Lock()
	curCol := c.currentColumn
	contID := c.config.ContinuationID
	c.mu.Unlock()
	return c.gate.CheckGate(ctx, c.SessionID, curCol, targetCol, contID)
}

// logGateDecision writes a gate decision to the conductor log.
func (c *Conductor) logGateDecision(targetCol session.KanbanColumn, result GateResult) {
	verdict := "fail"
	if result.Pass {
		verdict = "pass"
	}

	entry := ConductorLogEntry{
		Timestamp:  time.Now(),
		SessionID:  c.SessionID,
		FromColumn: c.CurrentColumn(),
		ToColumn:   targetCol,
		GateResult: verdict,
		Models:     result.Models,
	}

	if err := c.appendLogEntry(entry); err != nil {
		conductorLog.Warn("log_entry_failed",
			slog.String("session", c.SessionID),
			slog.String("error", err.Error()),
		)
	}
}

// backoffDelay sleeps with linear backoff based on retry count, respecting context cancellation.
// Caps at 30 seconds per retry to avoid excessive waits.
func (c *Conductor) backoffDelay(ctx context.Context) {
	c.mu.Lock()
	count := c.retryCount
	c.mu.Unlock()

	delay := time.Duration(count) * time.Second
	const maxDelay = 30 * time.Second
	if delay > maxDelay {
		delay = maxDelay
	}
	if delay < time.Second {
		delay = time.Second
	}

	select {
	case <-time.After(delay):
	case <-ctx.Done():
	}
}

// sanitizeLogField strips newlines and carriage returns to prevent log injection.
func sanitizeLogField(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\n", " "), "\r", " ")
}
