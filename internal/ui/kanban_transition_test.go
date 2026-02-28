package ui

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/asheshgoplani/agent-deck/internal/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- MoveValidation Tests ---

func TestIsValidMove_Forward(t *testing.T) {
	eng := NewTransitionEngine(&mockTransitionStorage{})

	tests := []struct {
		name string
		from session.KanbanColumn
		to   session.KanbanColumn
	}{
		{"Backlog to Design", session.KanbanBacklog, session.KanbanDesign},
		{"Design to Plan", session.KanbanDesign, session.KanbanPlan},
		{"Implement to Review", session.KanbanImplement, session.KanbanReview},
		{"Review to Done", session.KanbanReview, session.KanbanDone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := eng.IsValidMove(tt.from, tt.to)
			assert.True(t, v.Valid, "forward moves should be valid")
			assert.False(t, v.NeedsConfirm, "forward moves should not need confirmation")
			assert.Empty(t, v.Reason, "forward moves should have no reason")
		})
	}
}

func TestIsValidMove_Backward(t *testing.T) {
	eng := NewTransitionEngine(&mockTransitionStorage{})

	tests := []struct {
		name string
		from session.KanbanColumn
		to   session.KanbanColumn
	}{
		{"Design to Backlog", session.KanbanDesign, session.KanbanBacklog},
		{"Plan to Design", session.KanbanPlan, session.KanbanDesign},
		{"Review to Implement", session.KanbanReview, session.KanbanImplement},
		{"Done to Backlog", session.KanbanDone, session.KanbanBacklog},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := eng.IsValidMove(tt.from, tt.to)
			assert.True(t, v.Valid, "backward moves should be valid")
			assert.True(t, v.NeedsConfirm, "backward moves should need confirmation")
			assert.Empty(t, v.Reason, "backward moves should have no reason")
		})
	}
}

func TestIsValidMove_SameColumn(t *testing.T) {
	eng := NewTransitionEngine(&mockTransitionStorage{})

	columns := []session.KanbanColumn{
		session.KanbanBacklog, session.KanbanDesign, session.KanbanPlan,
		session.KanbanImplement, session.KanbanReview, session.KanbanDone,
	}

	for _, col := range columns {
		t.Run(col.String(), func(t *testing.T) {
			v := eng.IsValidMove(col, col)
			assert.False(t, v.Valid, "same column moves should be invalid")
			assert.False(t, v.NeedsConfirm, "same column moves should not need confirmation")
			assert.Equal(t, "same column", v.Reason)
		})
	}
}

func TestIsValidMove_AllForwardPairs(t *testing.T) {
	eng := NewTransitionEngine(&mockTransitionStorage{})

	columns := []session.KanbanColumn{
		session.KanbanBacklog, session.KanbanDesign, session.KanbanPlan,
		session.KanbanImplement, session.KanbanReview, session.KanbanDone,
	}

	count := 0
	for i := 0; i < len(columns); i++ {
		for j := i + 1; j < len(columns); j++ {
			from, to := columns[i], columns[j]
			t.Run(from.String()+"_to_"+to.String(), func(t *testing.T) {
				v := eng.IsValidMove(from, to)
				assert.True(t, v.Valid, "forward move %s->%s should be valid", from, to)
				assert.False(t, v.NeedsConfirm, "forward move %s->%s should not need confirmation", from, to)
			})
			count++
		}
	}
	assert.Equal(t, 15, count, "should test all 15 forward pairs")
}

func TestIsValidMove_AllBackwardPairs(t *testing.T) {
	eng := NewTransitionEngine(&mockTransitionStorage{})

	columns := []session.KanbanColumn{
		session.KanbanBacklog, session.KanbanDesign, session.KanbanPlan,
		session.KanbanImplement, session.KanbanReview, session.KanbanDone,
	}

	count := 0
	for i := 1; i < len(columns); i++ {
		for j := 0; j < i; j++ {
			from, to := columns[i], columns[j]
			t.Run(from.String()+"_to_"+to.String(), func(t *testing.T) {
				v := eng.IsValidMove(from, to)
				assert.True(t, v.Valid, "backward move %s->%s should be valid", from, to)
				assert.True(t, v.NeedsConfirm, "backward move %s->%s should need confirmation", from, to)
			})
			count++
		}
	}
	assert.Equal(t, 15, count, "should test all 15 backward pairs")
}

// --- RequestMove Tests ---

func TestRequestMove_Forward(t *testing.T) {
	store := &mockTransitionStorage{}
	eng := NewTransitionEngine(store)

	req := MoveRequest{
		SessionID:  "sess-1",
		FromColumn: session.KanbanBacklog,
		ToColumn:   session.KanbanDesign,
	}

	result := eng.RequestMove(req)

	require.True(t, result.Success, "forward move should succeed")
	assert.False(t, result.NeedsConfirm, "forward move should not need confirm")
	assert.Equal(t, session.KanbanBacklog, result.FromColumn)
	assert.Equal(t, session.KanbanDesign, result.ToColumn)
	assert.NoError(t, result.Error)

	// Verify storage was called
	assert.Equal(t, "sess-1", store.lastUpdatedSessionID)
	assert.NotNil(t, store.lastUpdatedColumn)
	assert.Equal(t, session.KanbanDesign, *store.lastUpdatedColumn)
	assert.True(t, store.sortOrderUpdated, "sort order should be updated")
}

func TestRequestMove_Backward_Confirmed(t *testing.T) {
	store := &mockTransitionStorage{}
	eng := NewTransitionEngine(store)

	req := MoveRequest{
		SessionID:    "sess-2",
		FromColumn:   session.KanbanReview,
		ToColumn:     session.KanbanImplement,
		ForceConfirm: true, // user confirmed the backward move
	}

	result := eng.RequestMove(req)

	require.True(t, result.Success, "confirmed backward move should succeed")
	assert.False(t, result.NeedsConfirm, "confirmed backward move should not need further confirm")
	assert.Equal(t, session.KanbanReview, result.FromColumn)
	assert.Equal(t, session.KanbanImplement, result.ToColumn)
	assert.NoError(t, result.Error)

	// Verify storage was called
	assert.Equal(t, "sess-2", store.lastUpdatedSessionID)
	assert.NotNil(t, store.lastUpdatedColumn)
	assert.Equal(t, session.KanbanImplement, *store.lastUpdatedColumn)
}

func TestRequestMove_Backward_Unconfirmed(t *testing.T) {
	store := &mockTransitionStorage{}
	eng := NewTransitionEngine(store)

	req := MoveRequest{
		SessionID:    "sess-3",
		FromColumn:   session.KanbanReview,
		ToColumn:     session.KanbanImplement,
		ForceConfirm: false, // user has NOT confirmed
	}

	result := eng.RequestMove(req)

	assert.False(t, result.Success, "unconfirmed backward move should not succeed")
	assert.True(t, result.NeedsConfirm, "unconfirmed backward move should request confirmation")
	assert.Equal(t, session.KanbanReview, result.FromColumn)
	assert.Equal(t, session.KanbanImplement, result.ToColumn)
	assert.NoError(t, result.Error)

	// Verify storage was NOT called
	assert.Empty(t, store.lastUpdatedSessionID, "storage should not be updated for unconfirmed moves")
}

func TestRequestMove_SameColumn(t *testing.T) {
	store := &mockTransitionStorage{}
	eng := NewTransitionEngine(store)

	req := MoveRequest{
		SessionID:  "sess-4",
		FromColumn: session.KanbanPlan,
		ToColumn:   session.KanbanPlan,
	}

	result := eng.RequestMove(req)

	assert.False(t, result.Success, "same-column move should not succeed")
	assert.False(t, result.NeedsConfirm)
	require.Error(t, result.Error, "same-column move should return an error")

	var transErr *TransitionError
	require.True(t, errors.As(result.Error, &transErr), "error should be TransitionError")
	assert.Equal(t, "sess-4", transErr.SessionID)
	assert.Equal(t, session.KanbanPlan, transErr.From)
	assert.Equal(t, session.KanbanPlan, transErr.To)

	// Verify storage was NOT called
	assert.Empty(t, store.lastUpdatedSessionID)
}

func TestRequestMove_StorageError(t *testing.T) {
	store := &mockTransitionStorage{
		updateColumnErr: errors.New("database locked"),
	}
	eng := NewTransitionEngine(store)

	req := MoveRequest{
		SessionID:  "sess-5",
		FromColumn: session.KanbanBacklog,
		ToColumn:   session.KanbanDesign,
	}

	result := eng.RequestMove(req)

	assert.False(t, result.Success, "should fail when storage errors")
	require.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "database locked")

	var transErr *TransitionError
	require.True(t, errors.As(result.Error, &transErr))
	assert.Equal(t, session.KanbanBacklog, transErr.From)
	assert.Equal(t, session.KanbanDesign, transErr.To)
}

func TestRequestMove_SortOrderError(t *testing.T) {
	store := &mockTransitionStorage{
		updateSortOrderErr: errors.New("sort order write failed"),
	}
	eng := NewTransitionEngine(store)

	req := MoveRequest{
		SessionID:  "sess-sort-err",
		FromColumn: session.KanbanBacklog,
		ToColumn:   session.KanbanDesign,
	}

	result := eng.RequestMove(req)

	assert.False(t, result.Success)
	require.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "update sort order failed")
}

// --- Skill Name Validation Tests ---

func TestValidateSkillName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"valid simple", "agentic-ai-review", true},
		{"valid with dots", "my.skill.name", true},
		{"valid with underscores", "my_skill_name", true},
		{"valid alphanumeric", "skill123", true},
		{"empty string", "", false},
		{"contains newline", "skill\nrm -rf /", false},
		{"contains semicolon", "skill;echo pwned", false},
		{"contains pipe", "skill|cat /etc/passwd", false},
		{"contains backtick", "skill`id`", false},
		{"contains space", "skill name", false},
		{"contains slash", "skill/path", false},
		{"starts with dot", ".hidden", false},
		{"starts with hyphen", "-flag", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ValidateSkillName(tt.input))
		})
	}
}

// --- Error Type Tests ---

func TestTransitionError_Error(t *testing.T) {
	err := &TransitionError{
		SessionID: "sess-1",
		From:      session.KanbanBacklog,
		To:        session.KanbanDesign,
		Reason:    "test failure",
	}

	assert.Contains(t, err.Error(), "sess-1")
	assert.Contains(t, err.Error(), "backlog")
	assert.Contains(t, err.Error(), "design")
	assert.Contains(t, err.Error(), "test failure")
}

func TestTransitionError_Unwrap(t *testing.T) {
	inner := errors.New("database error")
	err := &TransitionError{
		SessionID: "sess-1",
		From:      session.KanbanBacklog,
		To:        session.KanbanDesign,
		Reason:    "storage failed",
		Err:       inner,
	}

	assert.True(t, errors.Is(err, inner), "should unwrap to inner error")
}

func TestSkillFailedError_Error(t *testing.T) {
	err := &SkillFailedError{
		SkillName: "review-check",
		SessionID: "sess-1",
		Output:    "lint errors found",
	}

	assert.Contains(t, err.Error(), "review-check")
	assert.Contains(t, err.Error(), "sess-1")
}

func TestSkillFailedError_Unwrap(t *testing.T) {
	inner := errors.New("command failed")
	err := &SkillFailedError{
		SkillName: "test-runner",
		SessionID: "sess-1",
		Err:       inner,
	}

	assert.True(t, errors.Is(err, inner))
}

func TestRollbackError_Error(t *testing.T) {
	origErr := errors.New("skill failed")
	rollErr := errors.New("column update failed")
	err := &RollbackError{
		OriginalError: origErr,
		RollbackErr:   rollErr,
		SessionID:     "sess-1",
		FromColumn:    session.KanbanReview,
	}

	assert.Contains(t, err.Error(), "sess-1")
	assert.Contains(t, err.Error(), "review")
	assert.Contains(t, err.Error(), "rollback")
}

func TestRollbackError_Unwrap(t *testing.T) {
	origErr := errors.New("skill failed")
	rollErr := errors.New("column update failed")
	err := &RollbackError{
		OriginalError: origErr,
		RollbackErr:   rollErr,
		SessionID:     "sess-1",
		FromColumn:    session.KanbanReview,
	}

	assert.True(t, errors.Is(err, origErr), "should unwrap to original error")
}

// --- ResolveSkill Tests ---

func TestResolveSkill_DefaultsOnly(t *testing.T) {
	store := &mockTransitionStorage{}
	eng := NewTransitionEngine(store, WithConfigPath("/nonexistent/path/kanban.yaml"))

	columns := []session.KanbanColumn{
		session.KanbanBacklog, session.KanbanDesign, session.KanbanPlan,
		session.KanbanImplement, session.KanbanReview, session.KanbanDone,
	}

	for _, col := range columns {
		t.Run(col.String(), func(t *testing.T) {
			mapping, err := eng.ResolveSkill("test-group", col)
			require.NoError(t, err)
			assert.Equal(t, col, mapping.ColumnName)
			assert.NotEmpty(t, mapping.SkillName, "default mapping should have a skill name")
			assert.True(t, mapping.TriggerOnEnter, "defaults should have TriggerOnEnter=true")
			assert.False(t, mapping.AutoTrigger, "defaults should have AutoTrigger=false")
		})
	}
}

func TestResolveSkill_DefaultMappings(t *testing.T) {
	store := &mockTransitionStorage{}
	eng := NewTransitionEngine(store, WithConfigPath("/nonexistent/path/kanban.yaml"))

	tests := []struct {
		column    session.KanbanColumn
		wantSkill string
	}{
		{session.KanbanBacklog, "agentic-ai-backlog"},
		{session.KanbanDesign, "agentic-ai-brainstorm"},
		{session.KanbanPlan, "agentic-ai-plan"},
		{session.KanbanImplement, "agentic-ai-implement"},
		{session.KanbanReview, "agentic-ai-review"},
		{session.KanbanDone, "agentic-ai-done"},
	}

	for _, tt := range tests {
		t.Run(tt.column.String(), func(t *testing.T) {
			mapping, err := eng.ResolveSkill("any-group", tt.column)
			require.NoError(t, err)
			assert.Equal(t, tt.wantSkill, mapping.SkillName)
			assert.Equal(t, tt.column, mapping.ColumnName)
			assert.True(t, mapping.TriggerOnEnter)
			assert.False(t, mapping.AutoTrigger)
		})
	}
}

func TestResolveSkill_SQLiteOverride(t *testing.T) {
	store := &mockTransitionStorage{
		skillMappings: []*session.ColumnSkillMapping{
			{
				GroupPath:      "my-project",
				ColumnName:     session.KanbanImplement,
				SkillName:      "custom-sqlite-implement",
				AutoTrigger:    true,
				TriggerOnEnter: true,
			},
		},
	}
	eng := NewTransitionEngine(store, WithConfigPath("/nonexistent/path/kanban.yaml"))

	mapping, err := eng.ResolveSkill("my-project", session.KanbanImplement)
	require.NoError(t, err)
	assert.Equal(t, "custom-sqlite-implement", mapping.SkillName, "SQLite mapping should override defaults")
	assert.True(t, mapping.AutoTrigger, "SQLite auto_trigger should be preserved")
	assert.Equal(t, session.KanbanImplement, mapping.ColumnName)

	// A column NOT in SQLite should fall back to default
	mapping2, err := eng.ResolveSkill("my-project", session.KanbanBacklog)
	require.NoError(t, err)
	assert.Equal(t, "agentic-ai-backlog", mapping2.SkillName, "non-overridden column should use default")
}

func TestResolveSkill_YAMLOverride(t *testing.T) {
	// Create a temp YAML config file
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "kanban.yaml")
	yamlContent := `columns:
  design:
    skill: "custom-yaml-brainstorm"
    auto_trigger: true
  plan:
    skill: "custom-yaml-plan"
    auto_trigger: false
`
	require.NoError(t, os.WriteFile(yamlPath, []byte(yamlContent), 0644))

	store := &mockTransitionStorage{} // no SQLite overrides
	eng := NewTransitionEngine(store, WithConfigPath(yamlPath))

	// design should use YAML override
	mapping, err := eng.ResolveSkill("any-group", session.KanbanDesign)
	require.NoError(t, err)
	assert.Equal(t, "custom-yaml-brainstorm", mapping.SkillName, "YAML mapping should override defaults")
	assert.True(t, mapping.AutoTrigger, "YAML auto_trigger should be preserved")

	// plan should also use YAML override
	mapping2, err := eng.ResolveSkill("any-group", session.KanbanPlan)
	require.NoError(t, err)
	assert.Equal(t, "custom-yaml-plan", mapping2.SkillName)
	assert.False(t, mapping2.AutoTrigger)

	// backlog is NOT in YAML, should use default
	mapping3, err := eng.ResolveSkill("any-group", session.KanbanBacklog)
	require.NoError(t, err)
	assert.Equal(t, "agentic-ai-backlog", mapping3.SkillName, "non-overridden column should use default")
}

func TestResolveSkill_NoYAMLFile(t *testing.T) {
	store := &mockTransitionStorage{}
	eng := NewTransitionEngine(store, WithConfigPath("/this/path/does/not/exist.yaml"))

	// Should gracefully fall back to defaults, NOT return an error
	mapping, err := eng.ResolveSkill("any-group", session.KanbanReview)
	require.NoError(t, err, "missing YAML file should not produce an error")
	assert.Equal(t, "agentic-ai-review", mapping.SkillName)
}

func TestResolveSkill_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "kanban.yaml")
	require.NoError(t, os.WriteFile(yamlPath, []byte("{{invalid yaml[[["), 0644))

	store := &mockTransitionStorage{}
	eng := NewTransitionEngine(store, WithConfigPath(yamlPath))

	_, err := eng.ResolveSkill("any-group", session.KanbanPlan)
	require.Error(t, err, "malformed YAML should return an error")
	assert.Contains(t, err.Error(), "parse kanban config")
}

func TestResolveSkill_AutoTrigger(t *testing.T) {
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "kanban.yaml")
	yamlContent := `columns:
  implement:
    skill: "auto-impl"
    auto_trigger: true
`
	require.NoError(t, os.WriteFile(yamlPath, []byte(yamlContent), 0644))

	store := &mockTransitionStorage{}
	eng := NewTransitionEngine(store, WithConfigPath(yamlPath))

	mapping, err := eng.ResolveSkill("any-group", session.KanbanImplement)
	require.NoError(t, err)
	assert.True(t, mapping.AutoTrigger, "auto_trigger=true should be reflected in SkillMapping")
	assert.Equal(t, "auto-impl", mapping.SkillName)
}

func TestResolveSkill_ResolutionOrder(t *testing.T) {
	// All 3 tiers configured for the same column — SQLite should win

	// Tier 3: defaults exist for all columns (e.g., implement -> "agentic-ai-implement")

	// Tier 2: YAML override
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "kanban.yaml")
	yamlContent := `columns:
  implement:
    skill: "yaml-implement"
    auto_trigger: false
`
	require.NoError(t, os.WriteFile(yamlPath, []byte(yamlContent), 0644))

	// Tier 1: SQLite override (highest priority)
	store := &mockTransitionStorage{
		skillMappings: []*session.ColumnSkillMapping{
			{
				GroupPath:      "my-project",
				ColumnName:     session.KanbanImplement,
				SkillName:      "sqlite-implement",
				AutoTrigger:    true,
				TriggerOnEnter: true,
			},
		},
	}

	eng := NewTransitionEngine(store, WithConfigPath(yamlPath))

	// SQLite should win over YAML and defaults
	mapping, err := eng.ResolveSkill("my-project", session.KanbanImplement)
	require.NoError(t, err)
	assert.Equal(t, "sqlite-implement", mapping.SkillName, "SQLite (tier 1) should win over YAML (tier 2) and defaults (tier 3)")
	assert.True(t, mapping.AutoTrigger)

	// For a different group without SQLite overrides, YAML should win over defaults
	mapping2, err := eng.ResolveSkill("other-project", session.KanbanImplement)
	require.NoError(t, err)
	assert.Equal(t, "yaml-implement", mapping2.SkillName, "YAML (tier 2) should win over defaults (tier 3) when no SQLite override")
	assert.False(t, mapping2.AutoTrigger)

	// For a column not in YAML or SQLite, defaults should be used
	mapping3, err := eng.ResolveSkill("my-project", session.KanbanDone)
	require.NoError(t, err)
	assert.Equal(t, "agentic-ai-done", mapping3.SkillName, "defaults (tier 3) should be used when no overrides")
}

// --- Skill Trigger + Rollback Tests ---

func TestRequestMoveWithSkill_AutoTrigger(t *testing.T) {
	store := &mockTransitionStorage{
		skillMappings: []*session.ColumnSkillMapping{
			{
				GroupPath:      "my-project",
				ColumnName:     session.KanbanDesign,
				SkillName:      "custom-brainstorm",
				AutoTrigger:    true,
				TriggerOnEnter: true,
			},
		},
	}
	eng := NewTransitionEngine(store)

	req := MoveRequest{
		SessionID:  "sess-skill-1",
		FromColumn: session.KanbanBacklog,
		ToColumn:   session.KanbanDesign,
		GroupPath:  "my-project",
	}

	result := eng.RequestMove(req)

	require.True(t, result.Success, "forward move should succeed")
	assert.Equal(t, "custom-brainstorm", result.SkillName,
		"auto-trigger skill should be set on result")
	assert.NoError(t, result.Error)
}

func TestRequestMoveWithSkill_NoAutoTrigger(t *testing.T) {
	store := &mockTransitionStorage{
		skillMappings: []*session.ColumnSkillMapping{
			{
				GroupPath:      "my-project",
				ColumnName:     session.KanbanDesign,
				SkillName:      "manual-brainstorm",
				AutoTrigger:    false,
				TriggerOnEnter: true,
			},
		},
	}
	eng := NewTransitionEngine(store)

	req := MoveRequest{
		SessionID:  "sess-skill-2",
		FromColumn: session.KanbanBacklog,
		ToColumn:   session.KanbanDesign,
		GroupPath:  "my-project",
	}

	result := eng.RequestMove(req)

	require.True(t, result.Success, "move should succeed")
	assert.Empty(t, result.SkillName,
		"non-auto-trigger skill should NOT be set on result")
	assert.NoError(t, result.Error)
}

func TestRequestMoveWithSkill_SkillResolveFails(t *testing.T) {
	store := &mockTransitionStorage{
		skillMappingsErr: errors.New("skill DB corrupted"),
	}
	eng := NewTransitionEngine(store)

	req := MoveRequest{
		SessionID:  "sess-skill-3",
		FromColumn: session.KanbanBacklog,
		ToColumn:   session.KanbanDesign,
		GroupPath:  "my-project",
	}

	result := eng.RequestMove(req)

	require.True(t, result.Success, "move should succeed even when skill resolution fails")
	assert.Empty(t, result.SkillName,
		"skill name should be empty when resolution fails")
	assert.NoError(t, result.Error, "skill failure is non-fatal")
}

func TestRollbackMove(t *testing.T) {
	store := &mockTransitionStorage{}
	eng := NewTransitionEngine(store)

	err := eng.Rollback("sess-rb-1", session.KanbanBacklog, 5)

	require.NoError(t, err, "rollback should succeed")
	assert.Equal(t, "sess-rb-1", store.lastUpdatedSessionID)
	assert.NotNil(t, store.lastUpdatedColumn)
	assert.Equal(t, session.KanbanBacklog, *store.lastUpdatedColumn)
	assert.Equal(t, 5, store.lastUpdatedSortOrder)
	assert.True(t, store.sortOrderUpdated)
}

func TestRollbackMove_StorageError(t *testing.T) {
	store := &mockTransitionStorage{
		updateColumnErr: errors.New("disk full"),
	}
	eng := NewTransitionEngine(store)

	err := eng.Rollback("sess-rb-2", session.KanbanReview, 3)

	require.Error(t, err, "rollback should fail when storage errors")
	var rbErr *RollbackError
	require.True(t, errors.As(err, &rbErr), "error should be RollbackError")
	assert.Equal(t, "sess-rb-2", rbErr.SessionID)
	assert.Equal(t, session.KanbanReview, rbErr.FromColumn)
	assert.Contains(t, rbErr.RollbackErr.Error(), "disk full")
}

func TestBuildBackwardConfirmDialog(t *testing.T) {
	req := MoveRequest{
		SessionID:  "sess-cd-1",
		FromColumn: session.KanbanReview,
		ToColumn:   session.KanbanImplement,
		GroupPath:  "test-group",
	}

	dialog := BuildBackwardConfirmDialog(req)

	assert.Contains(t, dialog.Title, "Backward")
	assert.Contains(t, dialog.Body, session.KanbanReview.String())
	assert.Contains(t, dialog.Body, session.KanbanImplement.String())

	// OnConfirm should be an ExecuteTransitionMsg with ForceConfirm=true
	execMsg, ok := dialog.OnConfirm.(ExecuteTransitionMsg)
	require.True(t, ok, "OnConfirm should be ExecuteTransitionMsg")
	assert.Equal(t, "sess-cd-1", execMsg.Request.SessionID)
	assert.True(t, execMsg.Request.ForceConfirm, "confirmed dialog should set ForceConfirm=true")
}

func TestBuildSkillErrorActions(t *testing.T) {
	skillErr := errors.New("lint failed")
	errDisplay := BuildSkillErrorActions("sess-ea-1", "agentic-ai-review", skillErr)

	assert.Equal(t, "sess-ea-1", errDisplay.SessionID)
	assert.Error(t, errDisplay.Error)
	assert.Contains(t, errDisplay.Error.Error(), "lint failed")

	// Should have 3 actions: retry, view logs, ignore
	require.Len(t, errDisplay.Actions, 3, "should have retry, view logs, ignore")

	actionLabels := make([]string, len(errDisplay.Actions))
	for i, a := range errDisplay.Actions {
		actionLabels[i] = a.Label
	}
	assert.Contains(t, actionLabels, "Retry")
	assert.Contains(t, actionLabels, "View Logs")
	assert.Contains(t, actionLabels, "Ignore")
}

// --- Mock Storage ---

// mockTransitionStorage implements the TransitionStorage interface for testing.
type mockTransitionStorage struct {
	// Track calls
	lastUpdatedSessionID string
	lastUpdatedColumn    *session.KanbanColumn
	lastUpdatedSortOrder int
	sortOrderUpdated     bool

	// Error injection
	updateColumnErr    error
	updateSortOrderErr error

	// Existing cards for sort order calculation
	existingCards []*session.Instance

	// Skill mappings (for ResolveSkill tests)
	skillMappings    []*session.ColumnSkillMapping
	skillMappingsErr error
}

func (m *mockTransitionStorage) UpdateKanbanColumn(sessionID string, column *session.KanbanColumn) error {
	if m.updateColumnErr != nil {
		return m.updateColumnErr
	}
	m.lastUpdatedSessionID = sessionID
	m.lastUpdatedColumn = column
	return nil
}

func (m *mockTransitionStorage) UpdateKanbanSortOrder(sessionID string, sortOrder int) error {
	if m.updateSortOrderErr != nil {
		return m.updateSortOrderErr
	}
	m.lastUpdatedSortOrder = sortOrder
	m.sortOrderUpdated = true
	return nil
}

func (m *mockTransitionStorage) GetSessionsByColumn(column session.KanbanColumn) ([]*session.Instance, error) {
	if m.existingCards != nil {
		return m.existingCards, nil
	}
	return []*session.Instance{}, nil
}

func (m *mockTransitionStorage) GetColumnSkillMappings(groupPath string) ([]*session.ColumnSkillMapping, error) {
	if m.skillMappingsErr != nil {
		return nil, m.skillMappingsErr
	}
	// Filter mappings by groupPath to match real storage behavior
	var filtered []*session.ColumnSkillMapping
	for _, sm := range m.skillMappings {
		if sm.GroupPath == groupPath {
			filtered = append(filtered, sm)
		}
	}
	return filtered, nil
}
