package statedb

import (
	"encoding/json"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func newTestDB(t *testing.T) *StateDB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "state.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := db.Migrate(); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestOpenClose(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "state.db")

	// Open and write
	db1, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := db1.Migrate(); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if err := db1.SaveInstance(&InstanceRow{
		ID:          "test-1",
		Title:       "Test",
		ProjectPath: "/tmp",
		GroupPath:   "group",
		Tool:        "shell",
		Status:      "idle",
		CreatedAt:   time.Now(),
		ToolData:    json.RawMessage("{}"),
	}); err != nil {
		t.Fatalf("SaveInstance: %v", err)
	}
	db1.Close()

	// Reopen and verify
	db2, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Reopen: %v", err)
	}
	defer db2.Close()
	if err := db2.Migrate(); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	rows, err := db2.LoadInstances()
	if err != nil {
		t.Fatalf("LoadInstances: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("Expected 1 instance, got %d", len(rows))
	}
	if rows[0].ID != "test-1" || rows[0].Title != "Test" {
		t.Errorf("Unexpected data: %+v", rows[0])
	}
}

func TestSaveLoadInstances(t *testing.T) {
	db := newTestDB(t)

	now := time.Now()
	instances := []*InstanceRow{
		{ID: "a", Title: "Alpha", ProjectPath: "/a", GroupPath: "grp", Order: 0, Tool: "claude", Status: "idle", CreatedAt: now, ToolData: json.RawMessage(`{"claude_session_id":"abc"}`)},
		{ID: "b", Title: "Beta", ProjectPath: "/b", GroupPath: "grp", Order: 1, Tool: "gemini", Status: "running", CreatedAt: now, ToolData: json.RawMessage("{}")},
	}

	if err := db.SaveInstances(instances); err != nil {
		t.Fatalf("SaveInstances: %v", err)
	}

	loaded, err := db.LoadInstances()
	if err != nil {
		t.Fatalf("LoadInstances: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("Expected 2 instances, got %d", len(loaded))
	}
	if loaded[0].ID != "a" || loaded[1].ID != "b" {
		t.Errorf("Wrong order: %s, %s", loaded[0].ID, loaded[1].ID)
	}
	if loaded[0].Tool != "claude" {
		t.Errorf("Expected tool 'claude', got %q", loaded[0].Tool)
	}

	// Verify tool_data round-trip
	if string(loaded[0].ToolData) != `{"claude_session_id":"abc"}` {
		t.Errorf("ToolData mismatch: %s", loaded[0].ToolData)
	}
}

func TestSaveLoadGroups(t *testing.T) {
	db := newTestDB(t)

	groups := []*GroupRow{
		{Path: "projects", Name: "Projects", Expanded: true, Order: 0},
		{Path: "personal", Name: "Personal", Expanded: false, Order: 1, DefaultPath: "/home"},
	}

	if err := db.SaveGroups(groups); err != nil {
		t.Fatalf("SaveGroups: %v", err)
	}

	loaded, err := db.LoadGroups()
	if err != nil {
		t.Fatalf("LoadGroups: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("Expected 2 groups, got %d", len(loaded))
	}
	if !loaded[0].Expanded || loaded[1].Expanded {
		t.Errorf("Expanded mismatch: %v, %v", loaded[0].Expanded, loaded[1].Expanded)
	}
	if loaded[1].DefaultPath != "/home" {
		t.Errorf("DefaultPath: %q", loaded[1].DefaultPath)
	}
}

func TestDeleteInstance(t *testing.T) {
	db := newTestDB(t)

	if err := db.SaveInstance(&InstanceRow{
		ID: "del-me", Title: "Delete Me", ProjectPath: "/tmp", GroupPath: "grp",
		Tool: "shell", Status: "idle", CreatedAt: time.Now(), ToolData: json.RawMessage("{}"),
	}); err != nil {
		t.Fatalf("SaveInstance: %v", err)
	}

	if err := db.DeleteInstance("del-me"); err != nil {
		t.Fatalf("DeleteInstance: %v", err)
	}

	rows, _ := db.LoadInstances()
	if len(rows) != 0 {
		t.Errorf("Expected 0 instances after delete, got %d", len(rows))
	}
}

func TestStatusReadWrite(t *testing.T) {
	db := newTestDB(t)

	// Insert instance first
	if err := db.SaveInstance(&InstanceRow{
		ID: "s1", Title: "S1", ProjectPath: "/tmp", GroupPath: "grp",
		Tool: "claude", Status: "idle", CreatedAt: time.Now(), ToolData: json.RawMessage("{}"),
	}); err != nil {
		t.Fatalf("SaveInstance: %v", err)
	}

	// Simulate previously acknowledged waiting/idle state.
	if err := db.SetAcknowledged("s1", true); err != nil {
		t.Fatalf("SetAcknowledged: %v", err)
	}

	// Write status
	if err := db.WriteStatus("s1", "running", "claude"); err != nil {
		t.Fatalf("WriteStatus: %v", err)
	}

	// Read back
	statuses, err := db.ReadAllStatuses()
	if err != nil {
		t.Fatalf("ReadAllStatuses: %v", err)
	}
	if s, ok := statuses["s1"]; !ok || s.Status != "running" || s.Tool != "claude" {
		t.Errorf("Unexpected status: %+v", statuses["s1"])
	}
	if statuses["s1"].Acknowledged {
		t.Error("running status should clear acknowledged flag")
	}
}

func TestAcknowledgedSync(t *testing.T) {
	db := newTestDB(t)

	if err := db.SaveInstance(&InstanceRow{
		ID: "ack1", Title: "Ack Test", ProjectPath: "/tmp", GroupPath: "grp",
		Tool: "shell", Status: "waiting", CreatedAt: time.Now(), ToolData: json.RawMessage("{}"),
	}); err != nil {
		t.Fatalf("SaveInstance: %v", err)
	}

	// Set acknowledged from "instance A"
	if err := db.SetAcknowledged("ack1", true); err != nil {
		t.Fatalf("SetAcknowledged: %v", err)
	}

	// Read from "instance B" - should see the ack
	statuses, err := db.ReadAllStatuses()
	if err != nil {
		t.Fatalf("ReadAllStatuses: %v", err)
	}
	if !statuses["ack1"].Acknowledged {
		t.Error("Expected acknowledged=true after SetAcknowledged")
	}

	// Clear ack
	if err := db.SetAcknowledged("ack1", false); err != nil {
		t.Fatalf("SetAcknowledged(false): %v", err)
	}
	statuses, _ = db.ReadAllStatuses()
	if statuses["ack1"].Acknowledged {
		t.Error("Expected acknowledged=false after clearing")
	}
}

func TestHeartbeat(t *testing.T) {
	db := newTestDB(t)

	// Register
	if err := db.RegisterInstance(true); err != nil {
		t.Fatalf("RegisterInstance: %v", err)
	}

	// Heartbeat
	if err := db.Heartbeat(); err != nil {
		t.Fatalf("Heartbeat: %v", err)
	}

	// Check alive count
	count, err := db.AliveInstanceCount()
	if err != nil {
		t.Fatalf("AliveInstanceCount: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 alive, got %d", count)
	}

	// Unregister
	if err := db.UnregisterInstance(); err != nil {
		t.Fatalf("UnregisterInstance: %v", err)
	}

	count, _ = db.AliveInstanceCount()
	if count != 0 {
		t.Errorf("Expected 0 alive after unregister, got %d", count)
	}
}

func TestHeartbeatCleanup(t *testing.T) {
	db := newTestDB(t)

	// Insert a fake stale heartbeat (pid=99999, heartbeat 2 minutes ago)
	stale := time.Now().Add(-2 * time.Minute).Unix()
	_, err := db.DB().Exec(
		"INSERT INTO instance_heartbeats (pid, started, heartbeat, is_primary) VALUES (?, ?, ?, ?)",
		99999, stale, stale, 0,
	)
	if err != nil {
		t.Fatalf("Insert stale: %v", err)
	}

	// Register our own (fresh)
	if err := db.RegisterInstance(false); err != nil {
		t.Fatalf("RegisterInstance: %v", err)
	}

	// Clean dead (30s timeout should remove the stale one)
	if err := db.CleanDeadInstances(30 * time.Second); err != nil {
		t.Fatalf("CleanDeadInstances: %v", err)
	}

	// Only our instance should remain
	count, _ := db.AliveInstanceCount()
	if count != 1 {
		t.Errorf("Expected 1 alive after cleanup, got %d", count)
	}
}

func TestTouchAndLastModified(t *testing.T) {
	db := newTestDB(t)

	// Initially no timestamp
	ts0, err := db.LastModified()
	if err != nil {
		t.Fatalf("LastModified: %v", err)
	}
	if ts0 != 0 {
		t.Errorf("Expected 0 before any touch, got %d", ts0)
	}

	// Touch
	if err := db.Touch(); err != nil {
		t.Fatalf("Touch: %v", err)
	}

	ts1, err := db.LastModified()
	if err != nil {
		t.Fatalf("LastModified: %v", err)
	}
	if ts1 == 0 {
		t.Error("Expected non-zero after touch")
	}

	// Touch again (should advance)
	time.Sleep(2 * time.Millisecond) // ensure different nanosecond
	if err := db.Touch(); err != nil {
		t.Fatalf("Touch: %v", err)
	}

	ts2, _ := db.LastModified()
	if ts2 <= ts1 {
		t.Errorf("Expected ts2 > ts1: %d <= %d", ts2, ts1)
	}
}

func TestToolDataJSON(t *testing.T) {
	db := newTestDB(t)

	toolData := json.RawMessage(`{
		"claude_session_id": "cls-abc123",
		"gemini_session_id": "gem-xyz789",
		"gemini_yolo_mode": true,
		"latest_prompt": "fix the auth bug",
		"loaded_mcp_names": ["github", "exa"]
	}`)

	if err := db.SaveInstance(&InstanceRow{
		ID: "json1", Title: "JSON Test", ProjectPath: "/tmp", GroupPath: "grp",
		Tool: "claude", Status: "idle", CreatedAt: time.Now(), ToolData: toolData,
	}); err != nil {
		t.Fatalf("SaveInstance: %v", err)
	}

	loaded, err := db.LoadInstances()
	if err != nil {
		t.Fatalf("LoadInstances: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("Expected 1, got %d", len(loaded))
	}

	// Parse the JSON to verify structure
	var parsed map[string]any
	if err := json.Unmarshal(loaded[0].ToolData, &parsed); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if parsed["claude_session_id"] != "cls-abc123" {
		t.Errorf("claude_session_id: %v", parsed["claude_session_id"])
	}
	if parsed["gemini_yolo_mode"] != true {
		t.Errorf("gemini_yolo_mode: %v", parsed["gemini_yolo_mode"])
	}
}

func TestConcurrentAccess(t *testing.T) {
	db := newTestDB(t)

	// Pre-insert instances
	for i := 0; i < 10; i++ {
		id := "concurrent-" + string(rune('a'+i))
		if err := db.SaveInstance(&InstanceRow{
			ID: id, Title: id, ProjectPath: "/tmp", GroupPath: "grp",
			Tool: "shell", Status: "idle", CreatedAt: time.Now(), ToolData: json.RawMessage("{}"),
		}); err != nil {
			t.Fatalf("SaveInstance: %v", err)
		}
	}

	// Concurrent readers and writers
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				_, _ = db.LoadInstances()
				_, _ = db.ReadAllStatuses()
			}
		}()
	}

	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				id := "concurrent-" + string(rune('a'+idx))
				_ = db.WriteStatus(id, "running", "shell")
				_ = db.Heartbeat()
				_ = db.Touch()
			}
		}(i)
	}

	wg.Wait()
}

func TestIsEmpty(t *testing.T) {
	db := newTestDB(t)

	empty, err := db.IsEmpty()
	if err != nil {
		t.Fatalf("IsEmpty: %v", err)
	}
	if !empty {
		t.Error("Expected empty db")
	}

	if err := db.SaveInstance(&InstanceRow{
		ID: "not-empty", Title: "X", ProjectPath: "/tmp", GroupPath: "grp",
		Tool: "shell", Status: "idle", CreatedAt: time.Now(), ToolData: json.RawMessage("{}"),
	}); err != nil {
		t.Fatalf("SaveInstance: %v", err)
	}

	empty, _ = db.IsEmpty()
	if empty {
		t.Error("Expected non-empty after insert")
	}
}

func TestMetadata(t *testing.T) {
	db := newTestDB(t)

	// Missing key returns empty
	val, err := db.GetMeta("nonexistent")
	if err != nil {
		t.Fatalf("GetMeta: %v", err)
	}
	if val != "" {
		t.Errorf("Expected empty, got %q", val)
	}

	// Set and get
	if err := db.SetMeta("test_key", "test_value"); err != nil {
		t.Fatalf("SetMeta: %v", err)
	}
	val, _ = db.GetMeta("test_key")
	if val != "test_value" {
		t.Errorf("Expected 'test_value', got %q", val)
	}

	// Overwrite
	if err := db.SetMeta("test_key", "new_value"); err != nil {
		t.Fatalf("SetMeta: %v", err)
	}
	val, _ = db.GetMeta("test_key")
	if val != "new_value" {
		t.Errorf("Expected 'new_value', got %q", val)
	}
}

func TestElectPrimary_FirstInstance(t *testing.T) {
	db := newTestDB(t)

	// Register and elect
	if err := db.RegisterInstance(false); err != nil {
		t.Fatalf("RegisterInstance: %v", err)
	}
	isPrimary, err := db.ElectPrimary(30 * time.Second)
	if err != nil {
		t.Fatalf("ElectPrimary: %v", err)
	}
	if !isPrimary {
		t.Error("First instance should become primary")
	}

	// Calling again should still return true (already primary)
	isPrimary, err = db.ElectPrimary(30 * time.Second)
	if err != nil {
		t.Fatalf("ElectPrimary (repeat): %v", err)
	}
	if !isPrimary {
		t.Error("Should still be primary on repeat call")
	}
}

func TestElectPrimary_SecondInstance(t *testing.T) {
	db := newTestDB(t)

	// Simulate first instance (PID 10001) as primary with fresh heartbeat
	now := time.Now().Unix()
	_, err := db.DB().Exec(
		"INSERT INTO instance_heartbeats (pid, started, heartbeat, is_primary) VALUES (?, ?, ?, ?)",
		10001, now, now, 1,
	)
	if err != nil {
		t.Fatalf("Insert primary: %v", err)
	}

	// Register our process (not primary yet)
	if err := db.RegisterInstance(false); err != nil {
		t.Fatalf("RegisterInstance: %v", err)
	}

	// Try to elect: should fail because PID 10001 is alive and primary
	isPrimary, err := db.ElectPrimary(30 * time.Second)
	if err != nil {
		t.Fatalf("ElectPrimary: %v", err)
	}
	if isPrimary {
		t.Error("Second instance should NOT become primary while first is alive")
	}
}

func TestElectPrimary_Failover(t *testing.T) {
	db := newTestDB(t)

	// Simulate a stale primary (heartbeat 2 minutes ago)
	stale := time.Now().Add(-2 * time.Minute).Unix()
	_, err := db.DB().Exec(
		"INSERT INTO instance_heartbeats (pid, started, heartbeat, is_primary) VALUES (?, ?, ?, ?)",
		10001, stale, stale, 1,
	)
	if err != nil {
		t.Fatalf("Insert stale primary: %v", err)
	}

	// Register our process
	if err := db.RegisterInstance(false); err != nil {
		t.Fatalf("RegisterInstance: %v", err)
	}

	// Elect: stale primary should be cleared, we should become primary
	isPrimary, err := db.ElectPrimary(30 * time.Second)
	if err != nil {
		t.Fatalf("ElectPrimary: %v", err)
	}
	if !isPrimary {
		t.Error("Should become primary after stale primary is cleared")
	}

	// Verify the stale PID is no longer primary
	var stalePrimary int
	err = db.DB().QueryRow(
		"SELECT is_primary FROM instance_heartbeats WHERE pid = 10001",
	).Scan(&stalePrimary)
	if err != nil {
		t.Fatalf("Query stale PID: %v", err)
	}
	if stalePrimary != 0 {
		t.Error("Stale PID should have is_primary=0")
	}
}

func TestResignPrimary(t *testing.T) {
	db := newTestDB(t)

	// Register and elect
	if err := db.RegisterInstance(false); err != nil {
		t.Fatalf("RegisterInstance: %v", err)
	}
	isPrimary, err := db.ElectPrimary(30 * time.Second)
	if err != nil {
		t.Fatalf("ElectPrimary: %v", err)
	}
	if !isPrimary {
		t.Fatal("Should be primary")
	}

	// Resign
	if err := db.ResignPrimary(); err != nil {
		t.Fatalf("ResignPrimary: %v", err)
	}

	// Verify we're no longer primary
	var isPrim int
	err = db.DB().QueryRow(
		"SELECT is_primary FROM instance_heartbeats WHERE pid = ?",
		db.pid,
	).Scan(&isPrim)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if isPrim != 0 {
		t.Error("Should not be primary after resign")
	}

	// Re-elect should work since no primary exists
	isPrimary, err = db.ElectPrimary(30 * time.Second)
	if err != nil {
		t.Fatalf("ElectPrimary after resign: %v", err)
	}
	if !isPrimary {
		t.Error("Should become primary again after resign")
	}
}

func TestGlobalSingleton(t *testing.T) {
	// Initially nil
	if GetGlobal() != nil {
		t.Error("Expected nil global initially")
	}

	db := newTestDB(t)
	SetGlobal(db)
	defer SetGlobal(nil) // cleanup

	if GetGlobal() != db {
		t.Error("Expected global to return the set db")
	}

	SetGlobal(nil)
	if GetGlobal() != nil {
		t.Error("Expected nil after clearing")
	}
}

// --- Kanban Migration Tests (Schema v2) ---

func TestMigrationV2_FreshDB(t *testing.T) {
	db := newTestDB(t)

	// Verify schema version is 2
	version, err := db.GetMeta("schema_version")
	if err != nil {
		t.Fatalf("GetMeta: %v", err)
	}
	if version != "2" {
		t.Errorf("Expected schema version 2, got %q", version)
	}

	// Verify new columns exist by inserting an instance with kanban fields
	now := time.Now()
	col := "todo"
	desc := "Fix login bug"
	ac := "User can log in successfully"
	mode := "yolo"
	yoloJSON := `{"auto_approve":true}`
	movedAt := now.Unix()

	inst := &InstanceRow{
		ID:              "kanban-test",
		Title:           "Kanban Test",
		ProjectPath:     "/tmp/kanban",
		GroupPath:       "grp",
		Tool:            "claude",
		Status:          "idle",
		CreatedAt:       now,
		ToolData:        json.RawMessage("{}"),
		KanbanColumn:    &col,
		KanbanSortOrder: 3,
		KanbanLastMoved: &movedAt,
		Description:     desc,
		AcceptCriteria:  ac,
		AutomationMode:  mode,
		YOLOConfigJSON:  &yoloJSON,
	}

	if err := db.SaveInstance(inst); err != nil {
		t.Fatalf("SaveInstance with kanban fields: %v", err)
	}

	// Load and verify
	loaded, err := db.LoadInstances()
	if err != nil {
		t.Fatalf("LoadInstances: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("Expected 1 instance, got %d", len(loaded))
	}

	l := loaded[0]
	if l.KanbanColumn == nil || *l.KanbanColumn != "todo" {
		t.Errorf("KanbanColumn: %v", l.KanbanColumn)
	}
	if l.KanbanSortOrder != 3 {
		t.Errorf("KanbanSortOrder: %d", l.KanbanSortOrder)
	}
	if l.KanbanLastMoved == nil || *l.KanbanLastMoved != movedAt {
		t.Errorf("KanbanLastMoved: %v", l.KanbanLastMoved)
	}
	if l.Description != desc {
		t.Errorf("Description: %q", l.Description)
	}
	if l.AcceptCriteria != ac {
		t.Errorf("AcceptCriteria: %q", l.AcceptCriteria)
	}
	if l.AutomationMode != mode {
		t.Errorf("AutomationMode: %q", l.AutomationMode)
	}
	if l.YOLOConfigJSON == nil || *l.YOLOConfigJSON != yoloJSON {
		t.Errorf("YOLOConfigJSON: %v", l.YOLOConfigJSON)
	}

	// Verify new tables exist
	var count int
	err = db.DB().QueryRow("SELECT COUNT(*) FROM group_kanban_configs").Scan(&count)
	if err != nil {
		t.Errorf("group_kanban_configs table doesn't exist: %v", err)
	}

	err = db.DB().QueryRow("SELECT COUNT(*) FROM column_skill_mappings").Scan(&count)
	if err != nil {
		t.Errorf("column_skill_mappings table doesn't exist: %v", err)
	}
}

func TestMigrationV2_ExistingData(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "state.db")

	// Create v1 database
	db1, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	// Manually create v1 schema (without kanban fields)
	tx, _ := db1.DB().Begin()
	tx.Exec(`CREATE TABLE metadata (key TEXT PRIMARY KEY, value TEXT NOT NULL)`)
	tx.Exec(`INSERT INTO metadata (key, value) VALUES ('schema_version', '1')`)
	tx.Exec(`
		CREATE TABLE instances (
			id              TEXT PRIMARY KEY,
			title           TEXT NOT NULL,
			project_path    TEXT NOT NULL,
			group_path      TEXT NOT NULL DEFAULT 'my-sessions',
			sort_order      INTEGER NOT NULL DEFAULT 0,
			command         TEXT NOT NULL DEFAULT '',
			wrapper         TEXT NOT NULL DEFAULT '',
			tool            TEXT NOT NULL DEFAULT 'shell',
			status          TEXT NOT NULL DEFAULT 'error',
			tmux_session    TEXT NOT NULL DEFAULT '',
			created_at      INTEGER NOT NULL,
			last_accessed   INTEGER NOT NULL DEFAULT 0,
			parent_session_id TEXT NOT NULL DEFAULT '',
			worktree_path     TEXT NOT NULL DEFAULT '',
			worktree_repo     TEXT NOT NULL DEFAULT '',
			worktree_branch   TEXT NOT NULL DEFAULT '',
			tool_data       TEXT NOT NULL DEFAULT '{}',
			acknowledged    INTEGER NOT NULL DEFAULT 0
		)
	`)

	// Insert existing session
	now := time.Now()
	tx.Exec(`
		INSERT INTO instances (id, title, project_path, group_path, sort_order, tool, status, created_at, tool_data)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "existing-1", "Existing Session", "/tmp/existing", "grp", 5, "claude", "idle", now.Unix(), "{}")
	tx.Commit()
	db1.Close()

	// Reopen and migrate to v2
	db2, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Reopen: %v", err)
	}
	defer db2.Close()

	if err := db2.Migrate(); err != nil {
		t.Fatalf("Migrate to v2: %v", err)
	}

	// Verify schema version updated
	version, _ := db2.GetMeta("schema_version")
	if version != "2" {
		t.Errorf("Expected schema version 2, got %q", version)
	}

	// Load existing instance and verify new columns have defaults
	loaded, err := db2.LoadInstances()
	if err != nil {
		t.Fatalf("LoadInstances: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("Expected 1 instance, got %d", len(loaded))
	}

	l := loaded[0]
	if l.ID != "existing-1" {
		t.Errorf("ID: %q", l.ID)
	}
	if l.KanbanColumn != nil {
		t.Errorf("KanbanColumn should be NULL for migrated data, got %v", l.KanbanColumn)
	}
	if l.KanbanSortOrder != 0 {
		t.Errorf("KanbanSortOrder should be 0, got %d", l.KanbanSortOrder)
	}
	if l.KanbanLastMoved != nil {
		t.Errorf("KanbanLastMoved should be NULL, got %v", l.KanbanLastMoved)
	}
	if l.Description != "" {
		t.Errorf("Description should be empty, got %q", l.Description)
	}
	if l.AcceptCriteria != "" {
		t.Errorf("AcceptCriteria should be empty, got %q", l.AcceptCriteria)
	}
	if l.AutomationMode != "interactive" {
		t.Errorf("AutomationMode should be 'interactive', got %q", l.AutomationMode)
	}
	if l.YOLOConfigJSON != nil {
		t.Errorf("YOLOConfigJSON should be NULL, got %v", l.YOLOConfigJSON)
	}
}

func TestMigrationV2_Idempotent(t *testing.T) {
	db := newTestDB(t)

	// Migrate again (should be no-op)
	if err := db.Migrate(); err != nil {
		t.Fatalf("Second Migrate: %v", err)
	}

	// Migrate a third time
	if err := db.Migrate(); err != nil {
		t.Fatalf("Third Migrate: %v", err)
	}

	version, _ := db.GetMeta("schema_version")
	if version != "2" {
		t.Errorf("Expected schema version 2, got %q", version)
	}
}

func TestGroupKanbanConfigsTable(t *testing.T) {
	db := newTestDB(t)

	// Verify table structure
	rows, err := db.DB().Query("PRAGMA table_info(group_kanban_configs)")
	if err != nil {
		t.Fatalf("PRAGMA table_info: %v", err)
	}
	defer rows.Close()

	columns := make(map[string]bool)
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull, pk int
		var dflt any
		rows.Scan(&cid, &name, &typ, &notNull, &dflt, &pk)
		columns[name] = true
	}

	required := []string{"group_path", "kanban_enabled", "created_at", "updated_at"}
	for _, col := range required {
		if !columns[col] {
			t.Errorf("Missing column: %s", col)
		}
	}
}

func TestColumnSkillMappingsTable(t *testing.T) {
	db := newTestDB(t)

	// Verify table structure
	rows, err := db.DB().Query("PRAGMA table_info(column_skill_mappings)")
	if err != nil {
		t.Fatalf("PRAGMA table_info: %v", err)
	}
	defer rows.Close()

	columns := make(map[string]bool)
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull, pk int
		var dflt any
		rows.Scan(&cid, &name, &typ, &notNull, &dflt, &pk)
		columns[name] = true
	}

	required := []string{"id", "group_path", "column_name", "skill_name", "auto_trigger", "trigger_on_enter"}
	for _, col := range required {
		if !columns[col] {
			t.Errorf("Missing column: %s", col)
		}
	}
}

// --- Kanban CRUD Tests ---

func TestSaveLoadGroupKanbanConfig(t *testing.T) {
	db := newTestDB(t)

	now := time.Now()
	config := &GroupKanbanConfigRow{
		GroupPath:     "my-project",
		KanbanEnabled: true,
		CreatedAt:     now.Unix(),
		UpdatedAt:     now.Unix(),
	}

	if err := db.SaveGroupKanbanConfig(config); err != nil {
		t.Fatalf("SaveGroupKanbanConfig: %v", err)
	}

	loaded, err := db.LoadGroupKanbanConfig("my-project")
	if err != nil {
		t.Fatalf("LoadGroupKanbanConfig: %v", err)
	}
	if loaded == nil {
		t.Fatal("Expected config, got nil")
	}
	if loaded.GroupPath != "my-project" {
		t.Errorf("GroupPath: %q", loaded.GroupPath)
	}
	if !loaded.KanbanEnabled {
		t.Error("KanbanEnabled should be true")
	}
	if loaded.CreatedAt != now.Unix() {
		t.Errorf("CreatedAt: %d", loaded.CreatedAt)
	}
}

func TestLoadGroupKanbanConfig_NotFound(t *testing.T) {
	db := newTestDB(t)

	loaded, err := db.LoadGroupKanbanConfig("nonexistent")
	if err != nil {
		t.Fatalf("LoadGroupKanbanConfig: %v", err)
	}
	if loaded != nil {
		t.Errorf("Expected nil, got %+v", loaded)
	}
}

func TestSaveColumnSkillMapping(t *testing.T) {
	db := newTestDB(t)

	mapping := &ColumnSkillMappingRow{
		GroupPath:      "my-project",
		ColumnName:     "todo",
		SkillName:      "planner",
		AutoTrigger:    false,
		TriggerOnEnter: true,
	}

	if err := db.SaveColumnSkillMapping(mapping); err != nil {
		t.Fatalf("SaveColumnSkillMapping: %v", err)
	}

	loaded, err := db.LoadColumnSkillMappings("my-project")
	if err != nil {
		t.Fatalf("LoadColumnSkillMappings: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("Expected 1 mapping, got %d", len(loaded))
	}

	m := loaded[0]
	if m.GroupPath != "my-project" {
		t.Errorf("GroupPath: %q", m.GroupPath)
	}
	if m.ColumnName != "todo" {
		t.Errorf("ColumnName: %q", m.ColumnName)
	}
	if m.SkillName != "planner" {
		t.Errorf("SkillName: %q", m.SkillName)
	}
	if m.AutoTrigger {
		t.Error("AutoTrigger should be false")
	}
	if !m.TriggerOnEnter {
		t.Error("TriggerOnEnter should be true")
	}
}

func TestSaveColumnSkillMapping_Upsert(t *testing.T) {
	db := newTestDB(t)

	// Insert first mapping
	mapping1 := &ColumnSkillMappingRow{
		GroupPath:      "my-project",
		ColumnName:     "todo",
		SkillName:      "planner",
		AutoTrigger:    false,
		TriggerOnEnter: true,
	}
	if err := db.SaveColumnSkillMapping(mapping1); err != nil {
		t.Fatalf("SaveColumnSkillMapping (1): %v", err)
	}

	// Update with different skill
	mapping2 := &ColumnSkillMappingRow{
		GroupPath:      "my-project",
		ColumnName:     "todo",
		SkillName:      "architect",
		AutoTrigger:    true,
		TriggerOnEnter: false,
	}
	if err := db.SaveColumnSkillMapping(mapping2); err != nil {
		t.Fatalf("SaveColumnSkillMapping (2): %v", err)
	}

	// Should only have one mapping (upsert replaced the first)
	loaded, err := db.LoadColumnSkillMappings("my-project")
	if err != nil {
		t.Fatalf("LoadColumnSkillMappings: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("Expected 1 mapping after upsert, got %d", len(loaded))
	}

	m := loaded[0]
	if m.SkillName != "architect" {
		t.Errorf("SkillName should be 'architect', got %q", m.SkillName)
	}
	if !m.AutoTrigger {
		t.Error("AutoTrigger should be true after upsert")
	}
	if m.TriggerOnEnter {
		t.Error("TriggerOnEnter should be false after upsert")
	}
}

func TestLoadColumnSkillMappings_Empty(t *testing.T) {
	db := newTestDB(t)

	loaded, err := db.LoadColumnSkillMappings("nonexistent")
	if err != nil {
		t.Fatalf("LoadColumnSkillMappings: %v", err)
	}
	if len(loaded) != 0 {
		t.Errorf("Expected empty slice, got %d mappings", len(loaded))
	}
}

func TestMigrationV2_CompleteWorkflow(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "state.db")

	// Step 1: Create v1 database with multiple sessions
	db1, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	tx, _ := db1.DB().Begin()
	tx.Exec(`CREATE TABLE metadata (key TEXT PRIMARY KEY, value TEXT NOT NULL)`)
	tx.Exec(`INSERT INTO metadata (key, value) VALUES ('schema_version', '1')`)
	tx.Exec(`
		CREATE TABLE instances (
			id              TEXT PRIMARY KEY,
			title           TEXT NOT NULL,
			project_path    TEXT NOT NULL,
			group_path      TEXT NOT NULL DEFAULT 'my-sessions',
			sort_order      INTEGER NOT NULL DEFAULT 0,
			command         TEXT NOT NULL DEFAULT '',
			wrapper         TEXT NOT NULL DEFAULT '',
			tool            TEXT NOT NULL DEFAULT 'shell',
			status          TEXT NOT NULL DEFAULT 'error',
			tmux_session    TEXT NOT NULL DEFAULT '',
			created_at      INTEGER NOT NULL,
			last_accessed   INTEGER NOT NULL DEFAULT 0,
			parent_session_id TEXT NOT NULL DEFAULT '',
			worktree_path     TEXT NOT NULL DEFAULT '',
			worktree_repo     TEXT NOT NULL DEFAULT '',
			worktree_branch   TEXT NOT NULL DEFAULT '',
			tool_data       TEXT NOT NULL DEFAULT '{}',
			acknowledged    INTEGER NOT NULL DEFAULT 0
		)
	`)
	tx.Exec(`CREATE TABLE groups (path TEXT PRIMARY KEY, name TEXT NOT NULL, expanded INTEGER NOT NULL DEFAULT 1, sort_order INTEGER NOT NULL DEFAULT 0, default_path TEXT NOT NULL DEFAULT '')`)
	tx.Exec(`CREATE TABLE instance_heartbeats (pid INTEGER PRIMARY KEY, started INTEGER NOT NULL, heartbeat INTEGER NOT NULL, is_primary INTEGER NOT NULL DEFAULT 0)`)

	now := time.Now()
	tx.Exec(`INSERT INTO instances (id, title, project_path, group_path, sort_order, tool, status, created_at, tool_data)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, "sess-1", "Session 1", "/proj1", "grp1", 0, "claude", "idle", now.Unix(), `{"claude_session_id":"cls-1"}`)
	tx.Exec(`INSERT INTO instances (id, title, project_path, group_path, sort_order, tool, status, created_at, tool_data)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, "sess-2", "Session 2", "/proj2", "grp1", 1, "gemini", "running", now.Unix(), "{}")
	tx.Exec(`INSERT INTO instances (id, title, project_path, group_path, sort_order, tool, status, created_at, tool_data)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, "sess-3", "Session 3", "/proj3", "grp2", 0, "shell", "idle", now.Unix(), "{}")

	tx.Commit()
	db1.Close()

	// Step 2: Reopen and migrate
	db2, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Reopen: %v", err)
	}
	defer db2.Close()

	if err := db2.Migrate(); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	// Step 3: Verify all sessions loaded with defaults
	sessions, err := db2.LoadInstances()
	if err != nil {
		t.Fatalf("LoadInstances: %v", err)
	}
	if len(sessions) != 3 {
		t.Fatalf("Expected 3 sessions, got %d", len(sessions))
	}

	for _, s := range sessions {
		if s.KanbanColumn != nil {
			t.Errorf("Session %s: KanbanColumn should be nil, got %v", s.ID, *s.KanbanColumn)
		}
		if s.AutomationMode != "interactive" {
			t.Errorf("Session %s: AutomationMode should be 'interactive', got %q", s.ID, s.AutomationMode)
		}
	}

	// Step 4: Update one session with kanban data
	col := "in-progress"
	movedAt := time.Now().Unix()
	sessions[0].KanbanColumn = &col
	sessions[0].KanbanSortOrder = 5
	sessions[0].KanbanLastMoved = &movedAt
	sessions[0].Description = "Implement kanban mode"
	sessions[0].AcceptCriteria = "Board displays correctly"
	sessions[0].AutomationMode = "yolo"
	yoloConfig := `{"auto_approve":true,"risk_level":"high"}`
	sessions[0].YOLOConfigJSON = &yoloConfig

	if err := db2.SaveInstance(sessions[0]); err != nil {
		t.Fatalf("SaveInstance: %v", err)
	}

	// Step 5: Reload and verify
	reloaded, err := db2.LoadInstances()
	if err != nil {
		t.Fatalf("LoadInstances (2): %v", err)
	}

	var updated *InstanceRow
	for _, s := range reloaded {
		if s.ID == "sess-1" {
			updated = s
			break
		}
	}
	if updated == nil {
		t.Fatal("Could not find sess-1")
	}

	if updated.KanbanColumn == nil || *updated.KanbanColumn != "in-progress" {
		t.Errorf("KanbanColumn: %v", updated.KanbanColumn)
	}
	if updated.KanbanSortOrder != 5 {
		t.Errorf("KanbanSortOrder: %d", updated.KanbanSortOrder)
	}
	if updated.Description != "Implement kanban mode" {
		t.Errorf("Description: %q", updated.Description)
	}
	if updated.AutomationMode != "yolo" {
		t.Errorf("AutomationMode: %q", updated.AutomationMode)
	}
	if updated.YOLOConfigJSON == nil || *updated.YOLOConfigJSON != yoloConfig {
		t.Errorf("YOLOConfigJSON: %v", updated.YOLOConfigJSON)
	}

	// Step 6: Add kanban config for a group
	config := &GroupKanbanConfigRow{
		GroupPath:     "grp1",
		KanbanEnabled: true,
		CreatedAt:     now.Unix(),
		UpdatedAt:     now.Unix(),
	}
	if err := db2.SaveGroupKanbanConfig(config); err != nil {
		t.Fatalf("SaveGroupKanbanConfig: %v", err)
	}

	loadedConfig, err := db2.LoadGroupKanbanConfig("grp1")
	if err != nil {
		t.Fatalf("LoadGroupKanbanConfig: %v", err)
	}
	if !loadedConfig.KanbanEnabled {
		t.Error("KanbanEnabled should be true")
	}

	// Step 7: Add column skill mappings
	mapping1 := &ColumnSkillMappingRow{
		GroupPath:      "grp1",
		ColumnName:     "todo",
		SkillName:      "planner",
		AutoTrigger:    false,
		TriggerOnEnter: true,
	}
	mapping2 := &ColumnSkillMappingRow{
		GroupPath:      "grp1",
		ColumnName:     "in-progress",
		SkillName:      "tdd-guide",
		AutoTrigger:    true,
		TriggerOnEnter: true,
	}

	if err := db2.SaveColumnSkillMapping(mapping1); err != nil {
		t.Fatalf("SaveColumnSkillMapping (1): %v", err)
	}
	if err := db2.SaveColumnSkillMapping(mapping2); err != nil {
		t.Fatalf("SaveColumnSkillMapping (2): %v", err)
	}

	mappings, err := db2.LoadColumnSkillMappings("grp1")
	if err != nil {
		t.Fatalf("LoadColumnSkillMappings: %v", err)
	}
	if len(mappings) != 2 {
		t.Fatalf("Expected 2 mappings, got %d", len(mappings))
	}

	// Step 8: Delete a mapping
	if err := db2.DeleteColumnSkillMapping("grp1", "todo"); err != nil {
		t.Fatalf("DeleteColumnSkillMapping: %v", err)
	}

	mappings, _ = db2.LoadColumnSkillMappings("grp1")
	if len(mappings) != 1 {
		t.Fatalf("Expected 1 mapping after delete, got %d", len(mappings))
	}
	if mappings[0].ColumnName != "in-progress" {
		t.Errorf("Expected 'in-progress', got %q", mappings[0].ColumnName)
	}
}
