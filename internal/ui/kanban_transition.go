package ui

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"regexp"

	"github.com/asheshgoplani/agent-deck/internal/session"
	"gopkg.in/yaml.v3"
)

// validSkillName restricts skill names to safe characters to prevent
// command injection when sending skill commands to tmux sessions.
var validSkillName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

// kanban_transition.go — Kanban column transition engine
//
// Validates and executes session transitions between kanban columns.
// Forward moves (lower index -> higher index) proceed automatically.
// Backward moves (higher index -> lower index) require user confirmation.
// Same-column moves are rejected.

// --- Validation Types ---

// MoveValidation describes whether a move between columns is allowed.
type MoveValidation struct {
	Valid        bool
	NeedsConfirm bool
	Reason       string
}

// MoveResult describes the outcome of an executed column transition.
type MoveResult struct {
	Success      bool
	NeedsConfirm bool
	SkillName    string // skill triggered on entry (empty if none)
	Error        error
	FromColumn   session.KanbanColumn
	ToColumn     session.KanbanColumn
}

// SkillMapping maps a kanban column to its associated skill.
type SkillMapping struct {
	ColumnName     session.KanbanColumn
	SkillName      string
	AutoTrigger    bool
	TriggerOnEnter bool
}

// --- Error Types ---

// TransitionError wraps transition-specific errors with context.
type TransitionError struct {
	SessionID string
	From      session.KanbanColumn
	To        session.KanbanColumn
	Reason    string
	Err       error
}

func (e *TransitionError) Error() string {
	msg := fmt.Sprintf("transition %s->%s for session %s: %s", e.From, e.To, e.SessionID, e.Reason)
	if e.Err != nil {
		msg += ": " + e.Err.Error()
	}
	return msg
}

func (e *TransitionError) Unwrap() error {
	return e.Err
}

// SkillFailedError wraps skill execution failures.
type SkillFailedError struct {
	SkillName string
	SessionID string
	Output    string
	Err       error
}

func (e *SkillFailedError) Error() string {
	msg := fmt.Sprintf("skill %q failed for session %s", e.SkillName, e.SessionID)
	if e.Err != nil {
		msg += ": " + e.Err.Error()
	}
	return msg
}

func (e *SkillFailedError) Unwrap() error {
	return e.Err
}

// RollbackError wraps rollback failures after a failed transition.
type RollbackError struct {
	OriginalError error
	RollbackErr   error
	SessionID     string
	FromColumn    session.KanbanColumn
}

func (e *RollbackError) Error() string {
	return fmt.Sprintf(
		"rollback to %s failed for session %s: original=%v, rollback=%v",
		e.FromColumn, e.SessionID, e.OriginalError, e.RollbackErr,
	)
}

func (e *RollbackError) Unwrap() error {
	return e.OriginalError
}

// --- Storage Interface ---

// TransitionStorage abstracts the storage operations needed by the transition engine.
// Defined at the consumer site for testability.
type TransitionStorage interface {
	UpdateKanbanColumn(sessionID string, column *session.KanbanColumn) error
	UpdateKanbanSortOrder(sessionID string, sortOrder int) error
	GetSessionsByColumn(column session.KanbanColumn) ([]*session.Instance, error)
	GetColumnSkillMappings(groupPath string) ([]*session.ColumnSkillMapping, error)
}

// --- TransitionEngine Interface ---

// TransitionEngine validates and executes session moves between kanban columns.
type TransitionEngine interface {
	// IsValidMove checks whether a transition between two columns is allowed.
	IsValidMove(from, to session.KanbanColumn) MoveValidation

	// RequestMove attempts to move a session from one column to another.
	RequestMove(req MoveRequest) MoveResult

	// ResolveSkill looks up the skill mapping for a column within a group.
	ResolveSkill(groupPath string, column session.KanbanColumn) (SkillMapping, error)

	// Rollback reverts a failed column transition to its original state.
	Rollback(sessionID string, originalColumn session.KanbanColumn, originalSortOrder int) error
}

// --- Default Skill Mappings ---

// defaultSkillMappings maps each kanban column to its default skill.
var defaultSkillMappings = map[session.KanbanColumn]SkillMapping{
	session.KanbanBacklog:   {ColumnName: session.KanbanBacklog, SkillName: "agentic-ai-backlog", AutoTrigger: false, TriggerOnEnter: true},
	session.KanbanDesign:    {ColumnName: session.KanbanDesign, SkillName: "agentic-ai-brainstorm", AutoTrigger: false, TriggerOnEnter: true},
	session.KanbanPlan:      {ColumnName: session.KanbanPlan, SkillName: "agentic-ai-plan", AutoTrigger: false, TriggerOnEnter: true},
	session.KanbanImplement: {ColumnName: session.KanbanImplement, SkillName: "agentic-ai-implement", AutoTrigger: false, TriggerOnEnter: true},
	session.KanbanReview:    {ColumnName: session.KanbanReview, SkillName: "agentic-ai-review", AutoTrigger: false, TriggerOnEnter: true},
	session.KanbanDone:      {ColumnName: session.KanbanDone, SkillName: "agentic-ai-done", AutoTrigger: false, TriggerOnEnter: true},
}

// --- YAML Config Types ---

// KanbanYAMLConfig represents the YAML config file structure.
type KanbanYAMLConfig struct {
	Columns map[string]KanbanColumnYAML `yaml:"columns"`
}

// KanbanColumnYAML represents a single column's config in YAML.
type KanbanColumnYAML struct {
	Skill       string `yaml:"skill"`
	AutoTrigger bool   `yaml:"auto_trigger"`
}

// --- DefaultTransitionEngine ---

// TransitionOption configures a DefaultTransitionEngine.
type TransitionOption func(*DefaultTransitionEngine)

// WithConfigPath sets the path to the kanban YAML config file.
func WithConfigPath(path string) TransitionOption {
	return func(e *DefaultTransitionEngine) {
		e.configPath = path
	}
}

// DefaultTransitionEngine is the standard implementation of TransitionEngine.
type DefaultTransitionEngine struct {
	storage    TransitionStorage
	configPath string // path to kanban.yaml, injectable for testing
}

// NewTransitionEngine creates a new DefaultTransitionEngine with the given storage and options.
func NewTransitionEngine(storage TransitionStorage, opts ...TransitionOption) *DefaultTransitionEngine {
	e := &DefaultTransitionEngine{storage: storage}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// IsValidMove checks whether a transition between two columns is allowed.
// Forward moves (from.Index() < to.Index()) are valid without confirmation.
// Backward moves (from.Index() > to.Index()) are valid but need confirmation.
// Same-column moves are invalid.
func (e *DefaultTransitionEngine) IsValidMove(from, to session.KanbanColumn) MoveValidation {
	fromIdx := from.Index()
	toIdx := to.Index()

	switch {
	case fromIdx == toIdx:
		return MoveValidation{
			Valid:  false,
			Reason: "same column",
		}
	case fromIdx < toIdx:
		return MoveValidation{
			Valid: true,
		}
	default:
		return MoveValidation{
			Valid:        true,
			NeedsConfirm: true,
		}
	}
}

// RequestMove attempts to move a session from one column to another.
// It validates the move, checks confirmation for backward moves, then persists
// the column and sort order changes via storage.
func (e *DefaultTransitionEngine) RequestMove(req MoveRequest) MoveResult {
	validation := e.IsValidMove(req.FromColumn, req.ToColumn)

	if !validation.Valid {
		return e.moveError(req, validation.Reason, nil)
	}

	if validation.NeedsConfirm && !req.ForceConfirm {
		return MoveResult{
			NeedsConfirm: true,
			FromColumn:   req.FromColumn,
			ToColumn:     req.ToColumn,
		}
	}

	if err := e.executeStorageMove(req); err != nil {
		return e.moveError(req, "", err)
	}

	// Resolve skill for the target column (non-fatal — move already succeeded)
	skillName := e.resolveAutoTriggerSkill(req.GroupPath, req.ToColumn)

	return MoveResult{
		Success:    true,
		SkillName:  skillName,
		FromColumn: req.FromColumn,
		ToColumn:   req.ToColumn,
	}
}

// executeStorageMove performs the storage writes for a column transition.
// Updates the column first, then calculates and sets the sort order.
// If the sort order step fails, it attempts to rollback the column change.
func (e *DefaultTransitionEngine) executeStorageMove(req MoveRequest) error {
	col := req.ToColumn
	if err := e.storage.UpdateKanbanColumn(req.SessionID, &col); err != nil {
		return fmt.Errorf("update column failed: %w", err)
	}

	sortOrder, err := e.calculateTargetSortOrder(req.ToColumn)
	if err != nil {
		// Rollback column change (best-effort)
		_ = e.Rollback(req.SessionID, req.FromColumn, 0)
		return fmt.Errorf("calculate sort order failed: %w", err)
	}

	if err := e.storage.UpdateKanbanSortOrder(req.SessionID, sortOrder); err != nil {
		// Rollback column change (best-effort)
		_ = e.Rollback(req.SessionID, req.FromColumn, 0)
		return fmt.Errorf("update sort order failed: %w", err)
	}

	return nil
}

// moveError creates a MoveResult with a TransitionError.
func (e *DefaultTransitionEngine) moveError(req MoveRequest, reason string, err error) MoveResult {
	transErr := &TransitionError{
		SessionID: req.SessionID,
		From:      req.FromColumn,
		To:        req.ToColumn,
		Reason:    reason,
		Err:       err,
	}
	if reason == "" && err != nil {
		transErr.Reason = err.Error()
	}
	return MoveResult{
		Error:      transErr,
		FromColumn: req.FromColumn,
		ToColumn:   req.ToColumn,
	}
}

// ResolveSkill looks up the skill mapping for a column within a group.
// Resolution order (highest priority first):
//  1. SQLite — per-group overrides from storage
//  2. YAML — global overrides from config file
//  3. Defaults — hardcoded default mappings
func (e *DefaultTransitionEngine) ResolveSkill(groupPath string, column session.KanbanColumn) (SkillMapping, error) {
	// Tier 1: Check SQLite (per-group overrides)
	mapping, found, err := e.resolveFromStorage(groupPath, column)
	if err != nil {
		return SkillMapping{}, fmt.Errorf("resolve skill from storage: %w", err)
	}
	if found {
		return mapping, nil
	}

	// Tier 2: Check YAML file (global overrides)
	mapping, found, err = e.resolveFromYAML(column)
	if err != nil {
		return SkillMapping{}, err
	}
	if found {
		return mapping, nil
	}

	// Tier 3: Return defaults
	return e.resolveFromDefaults(column), nil
}

// resolveFromStorage checks SQLite for a per-group skill mapping.
func (e *DefaultTransitionEngine) resolveFromStorage(groupPath string, column session.KanbanColumn) (SkillMapping, bool, error) {
	mappings, err := e.storage.GetColumnSkillMappings(groupPath)
	if err != nil {
		return SkillMapping{}, false, fmt.Errorf("get column skill mappings: %w", err)
	}

	for _, m := range mappings {
		if m.ColumnName == column {
			return SkillMapping{
				ColumnName:     m.ColumnName,
				SkillName:      m.SkillName,
				AutoTrigger:    m.AutoTrigger,
				TriggerOnEnter: m.TriggerOnEnter,
			}, true, nil
		}
	}

	return SkillMapping{}, false, nil
}

// resolveFromYAML checks the YAML config file for a global skill mapping.
func (e *DefaultTransitionEngine) resolveFromYAML(column session.KanbanColumn) (SkillMapping, bool, error) {
	cfg, err := loadYAMLConfig(e.configPath)
	if err != nil {
		return SkillMapping{}, false, err
	}
	if cfg == nil {
		return SkillMapping{}, false, nil
	}

	colYAML, ok := cfg.Columns[column.String()]
	if !ok || colYAML.Skill == "" {
		return SkillMapping{}, false, nil
	}

	return SkillMapping{
		ColumnName:     column,
		SkillName:      colYAML.Skill,
		AutoTrigger:    colYAML.AutoTrigger,
		TriggerOnEnter: true,
	}, true, nil
}

// resolveFromDefaults returns the hardcoded default skill mapping for a column.
func (e *DefaultTransitionEngine) resolveFromDefaults(column session.KanbanColumn) SkillMapping {
	if m, ok := defaultSkillMappings[column]; ok {
		return m
	}
	return SkillMapping{ColumnName: column}
}

// maxKanbanConfigSize limits the YAML config file size to prevent resource exhaustion.
const maxKanbanConfigSize = 1 << 20 // 1 MB

// loadYAMLConfig reads and parses the kanban YAML config file.
// Returns nil config (not error) if the file does not exist.
// Returns an error if the file exists but cannot be parsed or exceeds size limit.
func loadYAMLConfig(path string) (*KanbanYAMLConfig, error) {
	if path == "" {
		return nil, nil
	}

	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("stat kanban config %s: %w", path, err)
	}
	if info.Size() > maxKanbanConfigSize {
		return nil, fmt.Errorf("kanban config %s exceeds max size (%d bytes)", path, maxKanbanConfigSize)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read kanban config %s: %w", path, err)
	}

	var cfg KanbanYAMLConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse kanban config %s: %w", path, err)
	}

	return &cfg, nil
}

// resolveAutoTriggerSkill resolves the skill for a column and returns its name
// only if AutoTrigger is true. Returns empty string on failure, when the skill
// does not auto-trigger, or when the skill name fails validation.
func (e *DefaultTransitionEngine) resolveAutoTriggerSkill(groupPath string, column session.KanbanColumn) string {
	mapping, err := e.ResolveSkill(groupPath, column)
	if err != nil {
		uiLog.Warn("skill_resolve_failed",
			slog.String("column", column.String()),
			slog.String("group", groupPath),
			slog.String("error", err.Error()),
		)
		return ""
	}
	if !mapping.AutoTrigger {
		return ""
	}
	if !ValidateSkillName(mapping.SkillName) {
		uiLog.Error("skill_name_invalid",
			slog.String("skill", mapping.SkillName),
			slog.String("column", column.String()),
		)
		return ""
	}
	return mapping.SkillName
}

// ValidateSkillName checks that a skill name contains only safe characters
// (alphanumeric, dots, hyphens, underscores) to prevent command injection.
func ValidateSkillName(name string) bool {
	return name != "" && validSkillName.MatchString(name)
}

// Rollback reverts a failed column transition by restoring the original column
// and sort order. Returns a RollbackError if storage operations fail.
func (e *DefaultTransitionEngine) Rollback(sessionID string, originalColumn session.KanbanColumn, originalSortOrder int) error {
	col := originalColumn
	if err := e.storage.UpdateKanbanColumn(sessionID, &col); err != nil {
		return &RollbackError{
			SessionID:   sessionID,
			FromColumn:  originalColumn,
			RollbackErr: fmt.Errorf("restore column: %w", err),
		}
	}

	if err := e.storage.UpdateKanbanSortOrder(sessionID, originalSortOrder); err != nil {
		return &RollbackError{
			SessionID:   sessionID,
			FromColumn:  originalColumn,
			RollbackErr: fmt.Errorf("restore sort order: %w", err),
		}
	}

	return nil
}

// BuildBackwardConfirmDialog creates a ShowConfirmDialogMsg for backward moves.
// The OnConfirm payload is an ExecuteTransitionMsg with ForceConfirm set to true.
func BuildBackwardConfirmDialog(req MoveRequest) ShowConfirmDialogMsg {
	return ShowConfirmDialogMsg{
		Title: "Confirm Move Backward",
		Body:  "Move session from " + req.FromColumn.String() + " to " + req.ToColumn.String() + "?",
		OnConfirm: ExecuteTransitionMsg{
			Request: MoveRequest{
				SessionID:    req.SessionID,
				FromColumn:   req.FromColumn,
				ToColumn:     req.ToColumn,
				GroupPath:    req.GroupPath,
				ForceConfirm: true,
			},
		},
	}
}

// BuildSkillErrorActions creates an ErrorDisplayMsg with retry, view logs, and
// ignore actions for a failed skill execution.
func BuildSkillErrorActions(sessionID string, skillName string, err error) ErrorDisplayMsg {
	return ErrorDisplayMsg{
		SessionID: sessionID,
		Error:     err,
		Actions: []ErrorAction{
			{
				Key:   'r',
				Label: "Retry",
				Msg: SkillTriggeredMsg{
					SessionID: sessionID,
					SkillName: skillName,
				},
			},
			{
				Key:   'v',
				Label: "View Logs",
				Msg: AttachSessionMsg{
					SessionID: sessionID,
				},
			},
			{
				Key:   'i',
				Label: "Ignore",
				Msg:   BoardRefreshMsg{},
			},
		},
	}
}

// calculateTargetSortOrder determines the sort order for a card entering a column.
func (e *DefaultTransitionEngine) calculateTargetSortOrder(column session.KanbanColumn) (int, error) {
	existing, err := e.storage.GetSessionsByColumn(column)
	if err != nil {
		return 0, fmt.Errorf("get sessions by column %s: %w", column, err)
	}
	return session.CalculateMoveSortOrder(column, existing), nil
}
