package session

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/asheshgoplani/agent-deck/internal/statedb"
)

// newTestStorage creates a Storage backed by an in-memory-like temp dir SQLite database.
func newTestStorage(t *testing.T) *Storage {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "state.db")
	db, err := statedb.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.Migrate(); err != nil {
		t.Fatalf("failed to migrate test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return &Storage{db: db, dbPath: dbPath, profile: "_test"}
}

// TestStorageUpdatedAtTimestamp verifies that SaveWithGroups sets the UpdatedAt timestamp
// and GetUpdatedAt() returns it correctly.
func TestStorageUpdatedAtTimestamp(t *testing.T) {
	s := newTestStorage(t)

	instances := []*Instance{
		{
			ID:          "test-1",
			Title:       "Test Session",
			ProjectPath: "/tmp/test",
			GroupPath:   "test-group",
			Command:     "claude",
			Tool:        "claude",
			Status:      StatusIdle,
			CreatedAt:   time.Now(),
		},
	}

	// Save data
	beforeSave := time.Now()
	time.Sleep(10 * time.Millisecond)

	err := s.SaveWithGroups(instances, nil)
	if err != nil {
		t.Fatalf("SaveWithGroups failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond)
	afterSave := time.Now()

	// Get the updated timestamp
	updatedAt, err := s.GetUpdatedAt()
	if err != nil {
		t.Fatalf("GetUpdatedAt failed: %v", err)
	}

	// Verify timestamp is within expected range
	if updatedAt.Before(beforeSave) {
		t.Errorf("UpdatedAt %v is before save started %v", updatedAt, beforeSave)
	}
	if updatedAt.After(afterSave) {
		t.Errorf("UpdatedAt %v is after save completed %v", updatedAt, afterSave)
	}

	// Verify timestamp is not zero
	if updatedAt.IsZero() {
		t.Error("UpdatedAt is zero, expected a valid timestamp")
	}

	// Save again and verify timestamp updates
	time.Sleep(50 * time.Millisecond)
	firstUpdatedAt := updatedAt

	err = s.SaveWithGroups(instances, nil)
	if err != nil {
		t.Fatalf("Second SaveWithGroups failed: %v", err)
	}

	secondUpdatedAt, err := s.GetUpdatedAt()
	if err != nil {
		t.Fatalf("Second GetUpdatedAt failed: %v", err)
	}

	// Verify second timestamp is after first
	if !secondUpdatedAt.After(firstUpdatedAt) {
		t.Errorf("Second UpdatedAt %v should be after first %v", secondUpdatedAt, firstUpdatedAt)
	}
}

// TestGetUpdatedAtEmpty verifies behavior when no data has been saved
func TestGetUpdatedAtEmpty(t *testing.T) {
	s := newTestStorage(t)

	updatedAt, err := s.GetUpdatedAt()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !updatedAt.IsZero() {
		t.Errorf("Expected zero time for empty db, got %v", updatedAt)
	}
}

// TestLoadLite verifies that LoadLite returns raw InstanceData without tmux initialization
func TestLoadLite(t *testing.T) {
	s := newTestStorage(t)

	instances := []*Instance{
		{
			ID:          "test-1",
			Title:       "Test Session 1",
			ProjectPath: "/tmp/test1",
			GroupPath:   "test-group",
			Command:     "claude",
			Tool:        "claude",
			Status:      StatusWaiting,
			CreatedAt:   time.Now(),
		},
		{
			ID:          "test-2",
			Title:       "Test Session 2",
			ProjectPath: "/tmp/test2",
			GroupPath:   "other-group",
			Command:     "gemini",
			Tool:        "gemini",
			Status:      StatusIdle,
			CreatedAt:   time.Now(),
		},
	}

	err := s.SaveWithGroups(instances, nil)
	if err != nil {
		t.Fatalf("SaveWithGroups failed: %v", err)
	}

	instData, groupData, err := s.LoadLite()
	if err != nil {
		t.Fatalf("LoadLite failed: %v", err)
	}

	if len(instData) != 2 {
		t.Errorf("Expected 2 instances, got %d", len(instData))
	}

	if instData[0].ID != "test-1" {
		t.Errorf("Expected first instance ID 'test-1', got '%s'", instData[0].ID)
	}
	if instData[0].Title != "Test Session 1" {
		t.Errorf("Expected first instance title 'Test Session 1', got '%s'", instData[0].Title)
	}
	if instData[0].Status != StatusWaiting {
		t.Errorf("Expected first instance status 'waiting', got '%s'", instData[0].Status)
	}

	if instData[1].ID != "test-2" {
		t.Errorf("Expected second instance ID 'test-2', got '%s'", instData[1].ID)
	}
	if instData[1].Tool != "gemini" {
		t.Errorf("Expected second instance tool 'gemini', got '%s'", instData[1].Tool)
	}

	if len(groupData) != 0 {
		t.Errorf("Expected 0 groups, got %d", len(groupData))
	}
}

// TestLoadLiteEmptyDB verifies LoadLite returns empty slice when database is empty
func TestLoadLiteEmptyDB(t *testing.T) {
	s := newTestStorage(t)

	instData, groupData, err := s.LoadLite()
	if err != nil {
		t.Errorf("LoadLite should not return error for empty db, got: %v", err)
	}
	if len(instData) != 0 {
		t.Errorf("Expected empty instances, got %d", len(instData))
	}
	if len(groupData) != 0 {
		t.Errorf("Expected empty groups, got %d", len(groupData))
	}
}

func TestStorageSaveWithGroups_DedupsClaudeSessionIDs(t *testing.T) {
	s := newTestStorage(t)
	now := time.Now()

	older := &Instance{
		ID:               "old",
		Title:            "Older",
		ProjectPath:      "/tmp/one",
		GroupPath:        "grp",
		Command:          "claude",
		Tool:             "claude",
		Status:           StatusIdle,
		CreatedAt:        now.Add(-2 * time.Minute),
		ClaudeSessionID:  "shared-session-id",
		ClaudeDetectedAt: now.Add(-2 * time.Minute),
	}
	newer := &Instance{
		ID:               "new",
		Title:            "Newer",
		ProjectPath:      "/tmp/two",
		GroupPath:        "grp",
		Command:          "claude",
		Tool:             "claude",
		Status:           StatusIdle,
		CreatedAt:        now.Add(-1 * time.Minute),
		ClaudeSessionID:  "shared-session-id",
		ClaudeDetectedAt: now.Add(-1 * time.Minute),
	}
	otherTool := &Instance{
		ID:          "gem",
		Title:       "Gemini",
		ProjectPath: "/tmp/gem",
		GroupPath:   "grp",
		Command:     "gemini",
		Tool:        "gemini",
		Status:      StatusIdle,
		CreatedAt:   now,
	}

	// Intentionally unsorted to ensure dedup logic does not rely on caller order.
	instances := []*Instance{newer, otherTool, older}
	if err := s.SaveWithGroups(instances, nil); err != nil {
		t.Fatalf("SaveWithGroups failed: %v", err)
	}

	if older.ClaudeSessionID != "shared-session-id" {
		t.Fatalf("older session should keep shared ID, got %q", older.ClaudeSessionID)
	}
	if newer.ClaudeSessionID != "" {
		t.Fatalf("newer duplicate should be cleared, got %q", newer.ClaudeSessionID)
	}

	loaded, _, err := s.LoadLite()
	if err != nil {
		t.Fatalf("LoadLite failed: %v", err)
	}

	byID := make(map[string]*InstanceData, len(loaded))
	for _, inst := range loaded {
		byID[inst.ID] = inst
	}
	if byID["old"] == nil || byID["new"] == nil {
		t.Fatalf("expected old and new instances in DB, got keys: %#v", byID)
	}
	if byID["old"].ClaudeSessionID != "shared-session-id" {
		t.Fatalf("db old session ID = %q, want shared-session-id", byID["old"].ClaudeSessionID)
	}
	if byID["new"].ClaudeSessionID != "" {
		t.Fatalf("db newer session ID = %q, want empty", byID["new"].ClaudeSessionID)
	}
}

// TestStorageKanbanFieldRoundTrip verifies kanban fields are persisted and loaded correctly.
func TestStorageKanbanFieldRoundTrip(t *testing.T) {
	s := newTestStorage(t)
	now := time.Now()
	lastMoved := now.Add(-1 * time.Hour)

	backlogCol := KanbanBacklog
	instances := []*Instance{
		{
			ID:              "kanban-1",
			Title:           "Test Kanban Task",
			ProjectPath:     "/tmp/kanban",
			GroupPath:       "test-group",
			Command:         "claude",
			Tool:            "claude",
			Status:          StatusIdle,
			CreatedAt:       now,
			KanbanColumn:    &backlogCol,
			KanbanSortOrder: 5,
			KanbanLastMoved: &lastMoved,
			Description:     "Test description with\nmultiple lines",
			AcceptCriteria:  "- AC 1\n- AC 2\n- AC 3",
			AutomationMode:  AutomationYOLO,
			YOLOConfig: &YOLOConfig{
				ConsensusModels: []string{"gemini-3.1-pro", "gpt-5.2"},
				PassThreshold:   0.8,
				PauseOnFail:     true,
				MaxRetries:      3,
				SkipColumns:     []string{"review"},
				ContinuationID:  "cont-123",
			},
		},
	}

	err := s.SaveWithGroups(instances, nil)
	if err != nil {
		t.Fatalf("SaveWithGroups failed: %v", err)
	}

	loaded, _, err := s.LoadWithGroups()
	if err != nil {
		t.Fatalf("LoadWithGroups failed: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("Expected 1 instance, got %d", len(loaded))
	}

	inst := loaded[0]
	if inst.KanbanColumn == nil || *inst.KanbanColumn != KanbanBacklog {
		t.Errorf("KanbanColumn: expected backlog, got %v", inst.KanbanColumn)
	}
	if inst.KanbanSortOrder != 5 {
		t.Errorf("KanbanSortOrder: expected 5, got %d", inst.KanbanSortOrder)
	}
	if inst.KanbanLastMoved == nil {
		t.Error("KanbanLastMoved should not be nil")
	} else if !inst.KanbanLastMoved.Truncate(time.Second).Equal(lastMoved.Truncate(time.Second)) {
		t.Errorf("KanbanLastMoved: expected %v, got %v", lastMoved, *inst.KanbanLastMoved)
	}
	if inst.Description != "Test description with\nmultiple lines" {
		t.Errorf("Description: expected multi-line, got %q", inst.Description)
	}
	if inst.AcceptCriteria != "- AC 1\n- AC 2\n- AC 3" {
		t.Errorf("AcceptCriteria: expected multi-line, got %q", inst.AcceptCriteria)
	}
	if inst.AutomationMode != AutomationYOLO {
		t.Errorf("AutomationMode: expected yolo, got %s", inst.AutomationMode)
	}
	if inst.YOLOConfig == nil {
		t.Fatal("YOLOConfig should not be nil")
	}
	if len(inst.YOLOConfig.ConsensusModels) != 2 {
		t.Errorf("YOLOConfig.ConsensusModels: expected 2, got %d", len(inst.YOLOConfig.ConsensusModels))
	}
	if inst.YOLOConfig.PassThreshold != 0.8 {
		t.Errorf("YOLOConfig.PassThreshold: expected 0.8, got %f", inst.YOLOConfig.PassThreshold)
	}
	if !inst.YOLOConfig.PauseOnFail {
		t.Error("YOLOConfig.PauseOnFail: expected true")
	}
	if inst.YOLOConfig.MaxRetries != 3 {
		t.Errorf("YOLOConfig.MaxRetries: expected 3, got %d", inst.YOLOConfig.MaxRetries)
	}
	if inst.YOLOConfig.ContinuationID != "cont-123" {
		t.Errorf("YOLOConfig.ContinuationID: expected cont-123, got %s", inst.YOLOConfig.ContinuationID)
	}
}

// TestStorageKanbanFieldDefaults verifies default values for kanban fields.
func TestStorageKanbanFieldDefaults(t *testing.T) {
	s := newTestStorage(t)

	instances := []*Instance{
		{
			ID:          "default-1",
			Title:       "No Kanban Fields",
			ProjectPath: "/tmp/default",
			GroupPath:   "test-group",
			Command:     "shell",
			Tool:        "shell",
			Status:      StatusIdle,
			CreatedAt:   time.Now(),
		},
	}

	err := s.SaveWithGroups(instances, nil)
	if err != nil {
		t.Fatalf("SaveWithGroups failed: %v", err)
	}

	loaded, _, err := s.LoadWithGroups()
	if err != nil {
		t.Fatalf("LoadWithGroups failed: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("Expected 1 instance, got %d", len(loaded))
	}

	inst := loaded[0]
	if inst.KanbanColumn != nil {
		t.Errorf("KanbanColumn: expected nil, got %v", *inst.KanbanColumn)
	}
	if inst.KanbanSortOrder != 0 {
		t.Errorf("KanbanSortOrder: expected 0, got %d", inst.KanbanSortOrder)
	}
	if inst.KanbanLastMoved != nil {
		t.Errorf("KanbanLastMoved: expected nil, got %v", *inst.KanbanLastMoved)
	}
	if inst.Description != "" {
		t.Errorf("Description: expected empty, got %q", inst.Description)
	}
	if inst.AcceptCriteria != "" {
		t.Errorf("AcceptCriteria: expected empty, got %q", inst.AcceptCriteria)
	}
	if inst.AutomationMode != AutomationInteractive {
		t.Errorf("AutomationMode: expected interactive, got %s", inst.AutomationMode)
	}
	if inst.YOLOConfig != nil {
		t.Errorf("YOLOConfig: expected nil, got %v", inst.YOLOConfig)
	}
}

// TestUpdateKanbanColumn verifies updating kanban column field.
func TestUpdateKanbanColumn(t *testing.T) {
	s := newTestStorage(t)

	instances := []*Instance{
		{
			ID:          "test-1",
			Title:       "Test",
			ProjectPath: "/tmp/test",
			GroupPath:   "grp",
			Command:     "shell",
			Tool:        "shell",
			Status:      StatusIdle,
			CreatedAt:   time.Now(),
		},
	}

	if err := s.SaveWithGroups(instances, nil); err != nil {
		t.Fatalf("SaveWithGroups failed: %v", err)
	}

	// Set kanban column to design
	designCol := KanbanDesign
	if err := s.UpdateKanbanColumn("test-1", &designCol); err != nil {
		t.Fatalf("UpdateKanbanColumn failed: %v", err)
	}

	loaded, _, err := s.LoadWithGroups()
	if err != nil {
		t.Fatalf("LoadWithGroups failed: %v", err)
	}

	if loaded[0].KanbanColumn == nil || *loaded[0].KanbanColumn != KanbanDesign {
		t.Errorf("Expected KanbanColumn=design, got %v", loaded[0].KanbanColumn)
	}

	// Remove from kanban (set to nil)
	if err := s.UpdateKanbanColumn("test-1", nil); err != nil {
		t.Fatalf("UpdateKanbanColumn to nil failed: %v", err)
	}

	loaded, _, err = s.LoadWithGroups()
	if err != nil {
		t.Fatalf("LoadWithGroups failed: %v", err)
	}

	if loaded[0].KanbanColumn != nil {
		t.Errorf("Expected KanbanColumn=nil, got %v", *loaded[0].KanbanColumn)
	}
}

// TestUpdateKanbanColumn_InvalidColumn verifies validation of invalid columns.
func TestUpdateKanbanColumn_InvalidColumn(t *testing.T) {
	s := newTestStorage(t)

	instances := []*Instance{
		{
			ID:          "test-1",
			Title:       "Test",
			ProjectPath: "/tmp/test",
			GroupPath:   "grp",
			Command:     "shell",
			Tool:        "shell",
			Status:      StatusIdle,
			CreatedAt:   time.Now(),
		},
	}

	if err := s.SaveWithGroups(instances, nil); err != nil {
		t.Fatalf("SaveWithGroups failed: %v", err)
	}

	// Attempt to set invalid column
	invalidCol := KanbanColumn("invalid")
	err := s.UpdateKanbanColumn("test-1", &invalidCol)
	if err == nil {
		t.Error("Expected error for invalid column, got nil")
	}
}

// TestUpdateSortOrder verifies updating kanban sort order.
func TestUpdateSortOrder(t *testing.T) {
	s := newTestStorage(t)

	instances := []*Instance{
		{
			ID:          "test-1",
			Title:       "Test",
			ProjectPath: "/tmp/test",
			GroupPath:   "grp",
			Command:     "shell",
			Tool:        "shell",
			Status:      StatusIdle,
			CreatedAt:   time.Now(),
		},
	}

	if err := s.SaveWithGroups(instances, nil); err != nil {
		t.Fatalf("SaveWithGroups failed: %v", err)
	}

	if err := s.UpdateKanbanSortOrder("test-1", 42); err != nil {
		t.Fatalf("UpdateKanbanSortOrder failed: %v", err)
	}

	loaded, _, err := s.LoadWithGroups()
	if err != nil {
		t.Fatalf("LoadWithGroups failed: %v", err)
	}

	if loaded[0].KanbanSortOrder != 42 {
		t.Errorf("Expected KanbanSortOrder=42, got %d", loaded[0].KanbanSortOrder)
	}
}

// TestBatchUpdateSortOrders verifies batch updating sort orders.
func TestBatchUpdateSortOrders(t *testing.T) {
	s := newTestStorage(t)

	instances := []*Instance{
		{
			ID:          "test-1",
			Title:       "Test 1",
			ProjectPath: "/tmp/test1",
			GroupPath:   "grp",
			Command:     "shell",
			Tool:        "shell",
			Status:      StatusIdle,
			CreatedAt:   time.Now(),
		},
		{
			ID:          "test-2",
			Title:       "Test 2",
			ProjectPath: "/tmp/test2",
			GroupPath:   "grp",
			Command:     "shell",
			Tool:        "shell",
			Status:      StatusIdle,
			CreatedAt:   time.Now(),
		},
		{
			ID:          "test-3",
			Title:       "Test 3",
			ProjectPath: "/tmp/test3",
			GroupPath:   "grp",
			Command:     "shell",
			Tool:        "shell",
			Status:      StatusIdle,
			CreatedAt:   time.Now(),
		},
	}

	if err := s.SaveWithGroups(instances, nil); err != nil {
		t.Fatalf("SaveWithGroups failed: %v", err)
	}

	// Batch update sort orders
	updates := map[string]int{
		"test-1": 100,
		"test-2": 200,
		"test-3": 300,
	}

	if err := s.BatchUpdateSortOrders(updates); err != nil {
		t.Fatalf("BatchUpdateSortOrders failed: %v", err)
	}

	loaded, _, err := s.LoadWithGroups()
	if err != nil {
		t.Fatalf("LoadWithGroups failed: %v", err)
	}

	// Verify all sort orders were updated
	sortOrderMap := make(map[string]int)
	for _, inst := range loaded {
		sortOrderMap[inst.ID] = inst.KanbanSortOrder
	}

	if sortOrderMap["test-1"] != 100 {
		t.Errorf("Expected test-1 KanbanSortOrder=100, got %d", sortOrderMap["test-1"])
	}
	if sortOrderMap["test-2"] != 200 {
		t.Errorf("Expected test-2 KanbanSortOrder=200, got %d", sortOrderMap["test-2"])
	}
	if sortOrderMap["test-3"] != 300 {
		t.Errorf("Expected test-3 KanbanSortOrder=300, got %d", sortOrderMap["test-3"])
	}
}

// TestUpdateDescription verifies updating description field.
func TestUpdateDescription(t *testing.T) {
	s := newTestStorage(t)

	instances := []*Instance{
		{
			ID:          "test-1",
			Title:       "Test",
			ProjectPath: "/tmp/test",
			GroupPath:   "grp",
			Command:     "shell",
			Tool:        "shell",
			Status:      StatusIdle,
			CreatedAt:   time.Now(),
		},
	}

	if err := s.SaveWithGroups(instances, nil); err != nil {
		t.Fatalf("SaveWithGroups failed: %v", err)
	}

	desc := "Multi-line\ndescription\nhere"
	if err := s.UpdateDescription("test-1", desc); err != nil {
		t.Fatalf("UpdateDescription failed: %v", err)
	}

	loaded, _, err := s.LoadWithGroups()
	if err != nil {
		t.Fatalf("LoadWithGroups failed: %v", err)
	}

	if loaded[0].Description != desc {
		t.Errorf("Expected Description=%q, got %q", desc, loaded[0].Description)
	}

	// Empty string is allowed
	if err := s.UpdateDescription("test-1", ""); err != nil {
		t.Fatalf("UpdateDescription to empty failed: %v", err)
	}

	loaded, _, err = s.LoadWithGroups()
	if err != nil {
		t.Fatalf("LoadWithGroups failed: %v", err)
	}

	if loaded[0].Description != "" {
		t.Errorf("Expected empty Description, got %q", loaded[0].Description)
	}
}

// TestUpdateAcceptCriteria verifies updating acceptance criteria field.
func TestUpdateAcceptCriteria(t *testing.T) {
	s := newTestStorage(t)

	instances := []*Instance{
		{
			ID:          "test-1",
			Title:       "Test",
			ProjectPath: "/tmp/test",
			GroupPath:   "grp",
			Command:     "shell",
			Tool:        "shell",
			Status:      StatusIdle,
			CreatedAt:   time.Now(),
		},
	}

	if err := s.SaveWithGroups(instances, nil); err != nil {
		t.Fatalf("SaveWithGroups failed: %v", err)
	}

	ac := "- Criterion 1\n- Criterion 2\n- Criterion 3"
	if err := s.UpdateAcceptCriteria("test-1", ac); err != nil {
		t.Fatalf("UpdateAcceptCriteria failed: %v", err)
	}

	loaded, _, err := s.LoadWithGroups()
	if err != nil {
		t.Fatalf("LoadWithGroups failed: %v", err)
	}

	if loaded[0].AcceptCriteria != ac {
		t.Errorf("Expected AcceptCriteria=%q, got %q", ac, loaded[0].AcceptCriteria)
	}
}

// TestUpdateAutomationMode verifies updating automation mode field.
func TestUpdateAutomationMode(t *testing.T) {
	s := newTestStorage(t)

	instances := []*Instance{
		{
			ID:          "test-1",
			Title:       "Test",
			ProjectPath: "/tmp/test",
			GroupPath:   "grp",
			Command:     "shell",
			Tool:        "shell",
			Status:      StatusIdle,
			CreatedAt:   time.Now(),
		},
	}

	if err := s.SaveWithGroups(instances, nil); err != nil {
		t.Fatalf("SaveWithGroups failed: %v", err)
	}

	// Set to YOLO
	if err := s.UpdateAutomationMode("test-1", AutomationYOLO); err != nil {
		t.Fatalf("UpdateAutomationMode to yolo failed: %v", err)
	}

	loaded, _, err := s.LoadWithGroups()
	if err != nil {
		t.Fatalf("LoadWithGroups failed: %v", err)
	}

	if loaded[0].AutomationMode != AutomationYOLO {
		t.Errorf("Expected AutomationMode=yolo, got %s", loaded[0].AutomationMode)
	}

	// Set to interactive
	if err := s.UpdateAutomationMode("test-1", AutomationInteractive); err != nil {
		t.Fatalf("UpdateAutomationMode to interactive failed: %v", err)
	}

	loaded, _, err = s.LoadWithGroups()
	if err != nil {
		t.Fatalf("LoadWithGroups failed: %v", err)
	}

	if loaded[0].AutomationMode != AutomationInteractive {
		t.Errorf("Expected AutomationMode=interactive, got %s", loaded[0].AutomationMode)
	}

	// Invalid mode should error
	err = s.UpdateAutomationMode("test-1", AutomationMode("invalid"))
	if err == nil {
		t.Error("Expected error for invalid automation mode, got nil")
	}
}

// TestUpdateYOLOConfig verifies updating YOLO config.
func TestUpdateYOLOConfig(t *testing.T) {
	s := newTestStorage(t)

	instances := []*Instance{
		{
			ID:          "test-1",
			Title:       "Test",
			ProjectPath: "/tmp/test",
			GroupPath:   "grp",
			Command:     "shell",
			Tool:        "shell",
			Status:      StatusIdle,
			CreatedAt:   time.Now(),
		},
	}

	if err := s.SaveWithGroups(instances, nil); err != nil {
		t.Fatalf("SaveWithGroups failed: %v", err)
	}

	cfg := &YOLOConfig{
		ConsensusModels: []string{"model1", "model2"},
		PassThreshold:   0.75,
		PauseOnFail:     false,
		MaxRetries:      5,
		SkipColumns:     []string{"review", "done"},
		ContinuationID:  "cont-456",
	}

	if err := s.UpdateYOLOConfig("test-1", cfg); err != nil {
		t.Fatalf("UpdateYOLOConfig failed: %v", err)
	}

	loaded, _, err := s.LoadWithGroups()
	if err != nil {
		t.Fatalf("LoadWithGroups failed: %v", err)
	}

	if loaded[0].YOLOConfig == nil {
		t.Fatal("YOLOConfig is nil")
	}
	if len(loaded[0].YOLOConfig.ConsensusModels) != 2 {
		t.Errorf("Expected 2 consensus models, got %d", len(loaded[0].YOLOConfig.ConsensusModels))
	}
	if loaded[0].YOLOConfig.PassThreshold != 0.75 {
		t.Errorf("Expected PassThreshold=0.75, got %f", loaded[0].YOLOConfig.PassThreshold)
	}
	if loaded[0].YOLOConfig.PauseOnFail {
		t.Error("Expected PauseOnFail=false, got true")
	}
	if loaded[0].YOLOConfig.MaxRetries != 5 {
		t.Errorf("Expected MaxRetries=5, got %d", loaded[0].YOLOConfig.MaxRetries)
	}
	if loaded[0].YOLOConfig.ContinuationID != "cont-456" {
		t.Errorf("Expected ContinuationID=cont-456, got %s", loaded[0].YOLOConfig.ContinuationID)
	}

	// Set to nil
	if err := s.UpdateYOLOConfig("test-1", nil); err != nil {
		t.Fatalf("UpdateYOLOConfig to nil failed: %v", err)
	}

	loaded, _, err = s.LoadWithGroups()
	if err != nil {
		t.Fatalf("LoadWithGroups failed: %v", err)
	}

	if loaded[0].YOLOConfig != nil {
		t.Error("Expected YOLOConfig=nil")
	}
}

// TestGetSessionsByColumn verifies querying sessions by kanban column.
func TestGetSessionsByColumn(t *testing.T) {
	s := newTestStorage(t)

	backlogCol := KanbanBacklog
	designCol := KanbanDesign
	instances := []*Instance{
		{
			ID:              "backlog-1",
			Title:           "Backlog Task 1",
			ProjectPath:     "/tmp/b1",
			GroupPath:       "grp",
			Command:         "shell",
			Tool:            "shell",
			Status:          StatusIdle,
			CreatedAt:       time.Now(),
			KanbanColumn:    &backlogCol,
			KanbanSortOrder: 1,
		},
		{
			ID:              "backlog-2",
			Title:           "Backlog Task 2",
			ProjectPath:     "/tmp/b2",
			GroupPath:       "grp",
			Command:         "shell",
			Tool:            "shell",
			Status:          StatusIdle,
			CreatedAt:       time.Now(),
			KanbanColumn:    &backlogCol,
			KanbanSortOrder: 0,
		},
		{
			ID:              "backlog-3",
			Title:           "Backlog Task 3",
			ProjectPath:     "/tmp/b3",
			GroupPath:       "grp",
			Command:         "shell",
			Tool:            "shell",
			Status:          StatusIdle,
			CreatedAt:       time.Now(),
			KanbanColumn:    &backlogCol,
			KanbanSortOrder: 2,
		},
		{
			ID:              "design-1",
			Title:           "Design Task 1",
			ProjectPath:     "/tmp/d1",
			GroupPath:       "grp",
			Command:         "shell",
			Tool:            "shell",
			Status:          StatusIdle,
			CreatedAt:       time.Now(),
			KanbanColumn:    &designCol,
			KanbanSortOrder: 0,
		},
		{
			ID:              "design-2",
			Title:           "Design Task 2",
			ProjectPath:     "/tmp/d2",
			GroupPath:       "grp",
			Command:         "shell",
			Tool:            "shell",
			Status:          StatusIdle,
			CreatedAt:       time.Now(),
			KanbanColumn:    &designCol,
			KanbanSortOrder: 1,
		},
	}

	if err := s.SaveWithGroups(instances, nil); err != nil {
		t.Fatalf("SaveWithGroups failed: %v", err)
	}

	// Query backlog column
	backlogSessions, err := s.GetSessionsByColumn(KanbanBacklog)
	if err != nil {
		t.Fatalf("GetSessionsByColumn failed: %v", err)
	}

	if len(backlogSessions) != 3 {
		t.Errorf("Expected 3 backlog sessions, got %d", len(backlogSessions))
	}

	// Verify order (should be sorted by KanbanSortOrder)
	if backlogSessions[0].ID != "backlog-2" || backlogSessions[0].KanbanSortOrder != 0 {
		t.Errorf("Expected first session to be backlog-2 with order 0, got %s with order %d",
			backlogSessions[0].ID, backlogSessions[0].KanbanSortOrder)
	}
	if backlogSessions[1].ID != "backlog-1" || backlogSessions[1].KanbanSortOrder != 1 {
		t.Errorf("Expected second session to be backlog-1 with order 1, got %s with order %d",
			backlogSessions[1].ID, backlogSessions[1].KanbanSortOrder)
	}

	// Query design column
	designSessions, err := s.GetSessionsByColumn(KanbanDesign)
	if err != nil {
		t.Fatalf("GetSessionsByColumn failed: %v", err)
	}

	if len(designSessions) != 2 {
		t.Errorf("Expected 2 design sessions, got %d", len(designSessions))
	}
}

// TestSetGroupKanbanConfig verifies enabling kanban for a group.
func TestSetGroupKanbanConfig(t *testing.T) {
	s := newTestStorage(t)

	config := &GroupKanbanConfig{
		GroupPath:     "test-group",
		KanbanEnabled: true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := s.SetGroupKanbanConfig(config); err != nil {
		t.Fatalf("SetGroupKanbanConfig failed: %v", err)
	}

	loaded, err := s.GetGroupKanbanConfig("test-group")
	if err != nil {
		t.Fatalf("GetGroupKanbanConfig failed: %v", err)
	}

	if loaded == nil {
		t.Fatal("Expected config to be loaded, got nil")
	}
	if loaded.GroupPath != "test-group" {
		t.Errorf("Expected GroupPath=test-group, got %s", loaded.GroupPath)
	}
	if !loaded.KanbanEnabled {
		t.Error("Expected KanbanEnabled=true, got false")
	}
}

// TestSetGroupKanbanConfig_Toggle verifies toggling kanban enabled flag.
func TestSetGroupKanbanConfig_Toggle(t *testing.T) {
	s := newTestStorage(t)

	// Enable
	config := &GroupKanbanConfig{
		GroupPath:     "test-group",
		KanbanEnabled: true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := s.SetGroupKanbanConfig(config); err != nil {
		t.Fatalf("SetGroupKanbanConfig failed: %v", err)
	}

	loaded, err := s.GetGroupKanbanConfig("test-group")
	if err != nil {
		t.Fatalf("GetGroupKanbanConfig failed: %v", err)
	}
	if !loaded.KanbanEnabled {
		t.Error("Expected KanbanEnabled=true after first set")
	}

	// Disable
	config.KanbanEnabled = false
	config.UpdatedAt = time.Now()

	if err := s.SetGroupKanbanConfig(config); err != nil {
		t.Fatalf("Second SetGroupKanbanConfig failed: %v", err)
	}

	loaded, err = s.GetGroupKanbanConfig("test-group")
	if err != nil {
		t.Fatalf("Second GetGroupKanbanConfig failed: %v", err)
	}
	if loaded.KanbanEnabled {
		t.Error("Expected KanbanEnabled=false after toggle")
	}
}

// TestGetGroupKanbanConfig_NotFound verifies default behavior for missing config.
func TestGetGroupKanbanConfig_NotFound(t *testing.T) {
	s := newTestStorage(t)

	config, err := s.GetGroupKanbanConfig("nonexistent")
	if err != nil {
		t.Fatalf("GetGroupKanbanConfig should not error for missing config: %v", err)
	}
	if config != nil {
		t.Errorf("Expected nil for missing config, got %v", config)
	}
}

// TestSetColumnSkillMapping verifies setting a column skill mapping.
func TestSetColumnSkillMapping(t *testing.T) {
	s := newTestStorage(t)

	mapping := &ColumnSkillMapping{
		GroupPath:      "test-group",
		ColumnName:     KanbanBacklog,
		SkillName:      "agentic-ai-backlog",
		AutoTrigger:    false,
		TriggerOnEnter: true,
	}

	if err := s.SetColumnSkillMapping(mapping); err != nil {
		t.Fatalf("SetColumnSkillMapping failed: %v", err)
	}

	mappings, err := s.GetColumnSkillMappings("test-group")
	if err != nil {
		t.Fatalf("GetColumnSkillMappings failed: %v", err)
	}

	if len(mappings) != 1 {
		t.Fatalf("Expected 1 mapping, got %d", len(mappings))
	}

	m := mappings[0]
	if m.ColumnName != KanbanBacklog {
		t.Errorf("Expected ColumnName=backlog, got %s", m.ColumnName)
	}
	if m.SkillName != "agentic-ai-backlog" {
		t.Errorf("Expected SkillName=agentic-ai-backlog, got %s", m.SkillName)
	}
	if m.AutoTrigger {
		t.Error("Expected AutoTrigger=false, got true")
	}
	if !m.TriggerOnEnter {
		t.Error("Expected TriggerOnEnter=true, got false")
	}
}

// TestSetColumnSkillMapping_Override verifies overriding an existing mapping.
func TestSetColumnSkillMapping_Override(t *testing.T) {
	s := newTestStorage(t)

	// Set initial mapping
	mapping := &ColumnSkillMapping{
		GroupPath:      "test-group",
		ColumnName:     KanbanBacklog,
		SkillName:      "skill-v1",
		AutoTrigger:    false,
		TriggerOnEnter: true,
	}

	if err := s.SetColumnSkillMapping(mapping); err != nil {
		t.Fatalf("SetColumnSkillMapping failed: %v", err)
	}

	// Override with new skill
	mapping.SkillName = "skill-v2"
	mapping.AutoTrigger = true

	if err := s.SetColumnSkillMapping(mapping); err != nil {
		t.Fatalf("Second SetColumnSkillMapping failed: %v", err)
	}

	mappings, err := s.GetColumnSkillMappings("test-group")
	if err != nil {
		t.Fatalf("GetColumnSkillMappings failed: %v", err)
	}

	if len(mappings) != 1 {
		t.Fatalf("Expected 1 mapping (overridden), got %d", len(mappings))
	}

	if mappings[0].SkillName != "skill-v2" {
		t.Errorf("Expected SkillName=skill-v2, got %s", mappings[0].SkillName)
	}
	if !mappings[0].AutoTrigger {
		t.Error("Expected AutoTrigger=true after override")
	}
}

// TestGetColumnSkillMappings verifies retrieving multiple mappings.
func TestGetColumnSkillMappings(t *testing.T) {
	s := newTestStorage(t)

	mappings := []*ColumnSkillMapping{
		{
			GroupPath:      "test-group",
			ColumnName:     KanbanBacklog,
			SkillName:      "skill-backlog",
			AutoTrigger:    false,
			TriggerOnEnter: true,
		},
		{
			GroupPath:      "test-group",
			ColumnName:     KanbanDesign,
			SkillName:      "skill-design",
			AutoTrigger:    true,
			TriggerOnEnter: false,
		},
		{
			GroupPath:      "test-group",
			ColumnName:     KanbanPlan,
			SkillName:      "skill-plan",
			AutoTrigger:    true,
			TriggerOnEnter: true,
		},
	}

	for _, m := range mappings {
		if err := s.SetColumnSkillMapping(m); err != nil {
			t.Fatalf("SetColumnSkillMapping failed: %v", err)
		}
	}

	loaded, err := s.GetColumnSkillMappings("test-group")
	if err != nil {
		t.Fatalf("GetColumnSkillMappings failed: %v", err)
	}

	if len(loaded) != 3 {
		t.Fatalf("Expected 3 mappings, got %d", len(loaded))
	}

	// Verify each mapping exists
	byColumn := make(map[KanbanColumn]*ColumnSkillMapping)
	for _, m := range loaded {
		byColumn[m.ColumnName] = m
	}

	if byColumn[KanbanBacklog].SkillName != "skill-backlog" {
		t.Errorf("Backlog mapping incorrect: %v", byColumn[KanbanBacklog])
	}
	if byColumn[KanbanDesign].SkillName != "skill-design" {
		t.Errorf("Design mapping incorrect: %v", byColumn[KanbanDesign])
	}
	if byColumn[KanbanPlan].SkillName != "skill-plan" {
		t.Errorf("Plan mapping incorrect: %v", byColumn[KanbanPlan])
	}
}
