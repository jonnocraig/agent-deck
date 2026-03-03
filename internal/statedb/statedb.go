package statedb

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// SchemaVersion tracks the current database schema version.
// Bump this when adding migrations.
const SchemaVersion = 2

// StateDB wraps a SQLite database for session/group persistence.
// Thread-safe for concurrent use from multiple goroutines within one process.
// Multiple OS processes can safely read/write via WAL mode + busy timeout.
type StateDB struct {
	db  *sql.DB
	pid int
}

// InstanceRow represents a session row in the database.
type InstanceRow struct {
	ID              string
	Title           string
	ProjectPath     string
	GroupPath       string
	Order           int
	Command         string
	Wrapper         string
	Tool            string
	Status          string
	TmuxSession     string
	CreatedAt       time.Time
	LastAccessed    time.Time
	ParentSessionID string
	WorktreePath    string
	WorktreeRepo    string
	WorktreeBranch  string
	ToolData        json.RawMessage // JSON blob for tool-specific data

	// Kanban board fields (v2 schema)
	KanbanColumn    *string `json:"kanban_column,omitempty"`    // nullable - nil means not on kanban
	KanbanSortOrder int     `json:"kanban_sort_order"`
	KanbanLastMoved *int64  `json:"kanban_last_moved,omitempty"` // Unix timestamp, nullable
	Description     string  `json:"description"`
	AcceptCriteria  string  `json:"accept_criteria"`
	AutomationMode  string  `json:"automation_mode"`     // "interactive" or "yolo"
	YOLOConfigJSON  *string `json:"yolo_config,omitempty"` // JSON string, nullable
}

// GroupRow represents a group row in the database.
type GroupRow struct {
	Path        string
	Name        string
	Expanded    bool
	Order       int
	DefaultPath string
}

// StatusRow holds status + acknowledgment for a session.
type StatusRow struct {
	Status       string
	Tool         string
	Acknowledged bool
}

// GroupKanbanConfigRow represents kanban config for a group.
type GroupKanbanConfigRow struct {
	GroupPath     string
	KanbanEnabled bool
	CreatedAt     int64
	UpdatedAt     int64
}

// ColumnSkillMappingRow represents a skill mapping for a kanban column.
type ColumnSkillMappingRow struct {
	ID             int
	GroupPath      string
	ColumnName     string
	SkillName      string
	AutoTrigger    bool
	TriggerOnEnter bool
}

// RecentSessionRow captures the config of a deleted session for quick re-creation.
type RecentSessionRow struct {
	ID             string // SHA-256 dedup key (title+path+tool+group)
	Title          string
	ProjectPath    string
	GroupPath      string
	Command        string
	Wrapper        string
	Tool           string
	ToolOptions    json.RawMessage // serialized ToolOptionsWrapper
	SandboxEnabled bool
	GeminiYoloMode *bool
	DeletedAt      time.Time
}

// global singleton for cross-package access (status writes from background worker)
var (
	globalDB   *StateDB
	globalDBMu sync.RWMutex
)

// SetGlobal sets the global StateDB instance.
func SetGlobal(db *StateDB) {
	globalDBMu.Lock()
	globalDB = db
	globalDBMu.Unlock()
}

// GetGlobal returns the global StateDB instance (may be nil).
func GetGlobal() *StateDB {
	globalDBMu.RLock()
	defer globalDBMu.RUnlock()
	return globalDB
}

// Open creates or opens a SQLite database at dbPath with WAL mode and busy timeout.
func Open(dbPath string) (*StateDB, error) {
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0700); err != nil {
		return nil, fmt.Errorf("statedb: mkdir: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("statedb: open: %w", err)
	}

	// WAL mode: allows concurrent readers while writing
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("statedb: wal mode: %w", err)
	}

	// Busy timeout: wait up to 5s if another process holds a lock
	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		db.Close()
		return nil, fmt.Errorf("statedb: busy timeout: %w", err)
	}

	// Foreign keys (for future use)
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("statedb: foreign keys: %w", err)
	}

	return &StateDB{db: db, pid: os.Getpid()}, nil
}

// Close checkpoints WAL and closes the database.
func (s *StateDB) Close() error {
	// Checkpoint WAL to merge it back into the main database file
	_, _ = s.db.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
	return s.db.Close()
}

// DB returns the underlying sql.DB for advanced use cases (e.g., testing).
func (s *StateDB) DB() *sql.DB {
	return s.db
}

// Migrate creates tables if they don't exist and runs any pending migrations.
func (s *StateDB) Migrate() error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("statedb: begin migrate: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// metadata table
	if _, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS metadata (
			key   TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)
	`); err != nil {
		return fmt.Errorf("statedb: create metadata: %w", err)
	}

	// instances table
	if _, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS instances (
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
	`); err != nil {
		return fmt.Errorf("statedb: create instances: %w", err)
	}

	// groups table
	if _, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS groups (
			path         TEXT PRIMARY KEY,
			name         TEXT NOT NULL,
			expanded     INTEGER NOT NULL DEFAULT 1,
			sort_order   INTEGER NOT NULL DEFAULT 0,
			default_path TEXT NOT NULL DEFAULT ''
		)
	`); err != nil {
		return fmt.Errorf("statedb: create groups: %w", err)
	}

	// instance heartbeats
	if _, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS instance_heartbeats (
			pid        INTEGER PRIMARY KEY,
			started    INTEGER NOT NULL,
			heartbeat  INTEGER NOT NULL,
			is_primary INTEGER NOT NULL DEFAULT 0
		)
	`); err != nil {
		return fmt.Errorf("statedb: create heartbeats: %w", err)
	}

	// Read existing schema version for conditional migrations
	var existingVersionStr string
	err = tx.QueryRow(`SELECT value FROM metadata WHERE key = 'schema_version'`).Scan(&existingVersionStr)
	existingVersion := 0
	if err == nil {
		fmt.Sscanf(existingVersionStr, "%d", &existingVersion)
	}

	// Schema v2: Kanban board support
	if existingVersion < 2 {
		// Add columns to instances table
		alterStatements := []string{
			`ALTER TABLE instances ADD COLUMN kanban_column TEXT`,
			`ALTER TABLE instances ADD COLUMN kanban_sort_order INTEGER NOT NULL DEFAULT 0`,
			`ALTER TABLE instances ADD COLUMN kanban_last_moved INTEGER`,
			`ALTER TABLE instances ADD COLUMN description TEXT NOT NULL DEFAULT ''`,
			`ALTER TABLE instances ADD COLUMN accept_criteria TEXT NOT NULL DEFAULT ''`,
			`ALTER TABLE instances ADD COLUMN automation_mode TEXT NOT NULL DEFAULT 'interactive'`,
			`ALTER TABLE instances ADD COLUMN yolo_config TEXT`,
		}
		for _, stmt := range alterStatements {
			if _, err := tx.Exec(stmt); err != nil {
				// Column may already exist if partial migration occurred
				if !strings.Contains(err.Error(), "duplicate column") {
					return fmt.Errorf("statedb: alter instances: %w", err)
				}
			}
		}

		// Create group_kanban_configs table
		if _, err := tx.Exec(`
			CREATE TABLE IF NOT EXISTS group_kanban_configs (
				group_path     TEXT PRIMARY KEY,
				kanban_enabled INTEGER NOT NULL DEFAULT 0,
				created_at     INTEGER NOT NULL,
				updated_at     INTEGER NOT NULL
			)
		`); err != nil {
			return fmt.Errorf("statedb: create group_kanban_configs: %w", err)
		}

		// Create column_skill_mappings table
		if _, err := tx.Exec(`
			CREATE TABLE IF NOT EXISTS column_skill_mappings (
				id              INTEGER PRIMARY KEY AUTOINCREMENT,
				group_path      TEXT NOT NULL,
				column_name     TEXT NOT NULL,
				skill_name      TEXT NOT NULL,
				auto_trigger    INTEGER NOT NULL DEFAULT 0,
				trigger_on_enter INTEGER NOT NULL DEFAULT 1,
				UNIQUE(group_path, column_name)
			)
		`); err != nil {
			return fmt.Errorf("statedb: create column_skill_mappings: %w", err)
		}
	}

	// recent_sessions table (schema v2)
	if _, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS recent_sessions (
			id              TEXT PRIMARY KEY,
			title           TEXT NOT NULL,
			project_path    TEXT NOT NULL,
			group_path      TEXT NOT NULL DEFAULT '',
			command         TEXT NOT NULL DEFAULT '',
			wrapper         TEXT NOT NULL DEFAULT '',
			tool            TEXT NOT NULL DEFAULT '',
			tool_options    TEXT NOT NULL DEFAULT '{}',
			sandbox_enabled INTEGER NOT NULL DEFAULT 0,
			gemini_yolo     INTEGER,
			deleted_at      INTEGER NOT NULL
		)
	`); err != nil {
		return fmt.Errorf("statedb: create recent_sessions: %w", err)
	}

	// Set schema version only when missing or changed.
	// Avoiding a write on every open reduces lock contention between CLI processes.
	schemaVersion := fmt.Sprintf("%d", SchemaVersion)
	var currentVersion string
	err = tx.QueryRow(`SELECT value FROM metadata WHERE key = 'schema_version'`).Scan(&currentVersion)
	switch {
	case err == sql.ErrNoRows:
		if _, err := tx.Exec(`
			INSERT INTO metadata (key, value) VALUES ('schema_version', ?)
		`, schemaVersion); err != nil {
			return fmt.Errorf("statedb: insert schema version: %w", err)
		}
	case err != nil:
		return fmt.Errorf("statedb: read schema version: %w", err)
	case currentVersion != schemaVersion:
		if _, err := tx.Exec(`
			UPDATE metadata SET value = ? WHERE key = 'schema_version'
		`, schemaVersion); err != nil {
			return fmt.Errorf("statedb: update schema version: %w", err)
		}
	}

	return tx.Commit()
}

// IsEmpty returns true if the instances table has no rows.
func (s *StateDB) IsEmpty() (bool, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM instances").Scan(&count)
	if err != nil {
		return false, err
	}
	return count == 0, nil
}

// --- Instance CRUD ---

// SaveInstance inserts or replaces a single instance.
func (s *StateDB) SaveInstance(inst *InstanceRow) error {
	toolData := inst.ToolData
	if len(toolData) == 0 {
		toolData = json.RawMessage("{}")
	}

	var kanbanCol any = nil
	if inst.KanbanColumn != nil {
		kanbanCol = *inst.KanbanColumn
	}

	var kanbanMoved any = nil
	if inst.KanbanLastMoved != nil {
		kanbanMoved = *inst.KanbanLastMoved
	}

	var yoloConfig any = nil
	if inst.YOLOConfigJSON != nil {
		yoloConfig = *inst.YOLOConfigJSON
	}

	_, err := s.db.Exec(`
		INSERT OR REPLACE INTO instances (
			id, title, project_path, group_path, sort_order,
			command, wrapper, tool, status, tmux_session,
			created_at, last_accessed,
			parent_session_id, worktree_path, worktree_repo, worktree_branch,
			tool_data,
			kanban_column, kanban_sort_order, kanban_last_moved,
			description, accept_criteria, automation_mode, yolo_config
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		inst.ID, inst.Title, inst.ProjectPath, inst.GroupPath, inst.Order,
		inst.Command, inst.Wrapper, inst.Tool, inst.Status, inst.TmuxSession,
		inst.CreatedAt.Unix(), inst.LastAccessed.Unix(),
		inst.ParentSessionID, inst.WorktreePath, inst.WorktreeRepo, inst.WorktreeBranch,
		string(toolData),
		kanbanCol, inst.KanbanSortOrder, kanbanMoved,
		inst.Description, inst.AcceptCriteria, inst.AutomationMode, yoloConfig,
	)
	return err
}

// SaveInstances inserts or replaces multiple instances in a single transaction.
// It also removes any rows from the database that are not in the provided list,
// ensuring deleted sessions don't reappear on reload.
func (s *StateDB) SaveInstances(insts []*InstanceRow) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// Delete rows not in the new list to prevent deleted sessions from reappearing.
	if len(insts) == 0 {
		if _, err := tx.Exec("DELETE FROM instances"); err != nil {
			return err
		}
	} else {
		placeholders := make([]string, len(insts))
		args := make([]any, len(insts))
		for i, inst := range insts {
			placeholders[i] = "?"
			args[i] = inst.ID
		}
		query := "DELETE FROM instances WHERE id NOT IN (" + strings.Join(placeholders, ",") + ")"
		if _, err := tx.Exec(query, args...); err != nil {
			return err
		}
	}

	stmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO instances (
			id, title, project_path, group_path, sort_order,
			command, wrapper, tool, status, tmux_session,
			created_at, last_accessed,
			parent_session_id, worktree_path, worktree_repo, worktree_branch,
			tool_data,
			kanban_column, kanban_sort_order, kanban_last_moved,
			description, accept_criteria, automation_mode, yolo_config
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, inst := range insts {
		toolData := inst.ToolData
		if len(toolData) == 0 {
			toolData = json.RawMessage("{}")
		}

		var kanbanCol any = nil
		if inst.KanbanColumn != nil {
			kanbanCol = *inst.KanbanColumn
		}

		var kanbanMoved any = nil
		if inst.KanbanLastMoved != nil {
			kanbanMoved = *inst.KanbanLastMoved
		}

		var yoloConfig any = nil
		if inst.YOLOConfigJSON != nil {
			yoloConfig = *inst.YOLOConfigJSON
		}

		if _, err := stmt.Exec(
			inst.ID, inst.Title, inst.ProjectPath, inst.GroupPath, inst.Order,
			inst.Command, inst.Wrapper, inst.Tool, inst.Status, inst.TmuxSession,
			inst.CreatedAt.Unix(), inst.LastAccessed.Unix(),
			inst.ParentSessionID, inst.WorktreePath, inst.WorktreeRepo, inst.WorktreeBranch,
			string(toolData),
			kanbanCol, inst.KanbanSortOrder, kanbanMoved,
			inst.Description, inst.AcceptCriteria, inst.AutomationMode, yoloConfig,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// LoadInstances returns all instances ordered by sort_order.
func (s *StateDB) LoadInstances() ([]*InstanceRow, error) {
	rows, err := s.db.Query(`
		SELECT id, title, project_path, group_path, sort_order,
			command, wrapper, tool, status, tmux_session,
			created_at, last_accessed,
			parent_session_id, worktree_path, worktree_repo, worktree_branch,
			tool_data,
			kanban_column, kanban_sort_order, kanban_last_moved,
			description, accept_criteria, automation_mode, yolo_config
		FROM instances ORDER BY sort_order
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*InstanceRow
	for rows.Next() {
		r := &InstanceRow{}
		var createdUnix, accessedUnix int64
		var toolDataStr string
		var kanbanCol, yoloConfigStr sql.NullString
		var kanbanMoved sql.NullInt64

		if err := rows.Scan(
			&r.ID, &r.Title, &r.ProjectPath, &r.GroupPath, &r.Order,
			&r.Command, &r.Wrapper, &r.Tool, &r.Status, &r.TmuxSession,
			&createdUnix, &accessedUnix,
			&r.ParentSessionID, &r.WorktreePath, &r.WorktreeRepo, &r.WorktreeBranch,
			&toolDataStr,
			&kanbanCol, &r.KanbanSortOrder, &kanbanMoved,
			&r.Description, &r.AcceptCriteria, &r.AutomationMode, &yoloConfigStr,
		); err != nil {
			return nil, err
		}

		r.CreatedAt = time.Unix(createdUnix, 0)
		if accessedUnix > 0 {
			r.LastAccessed = time.Unix(accessedUnix, 0)
		}
		r.ToolData = json.RawMessage(toolDataStr)

		if kanbanCol.Valid {
			r.KanbanColumn = &kanbanCol.String
		}
		if kanbanMoved.Valid {
			r.KanbanLastMoved = &kanbanMoved.Int64
		}
		if yoloConfigStr.Valid {
			r.YOLOConfigJSON = &yoloConfigStr.String
		}

		result = append(result, r)
	}
	return result, rows.Err()
}

// DeleteInstance removes an instance by ID.
func (s *StateDB) DeleteInstance(id string) error {
	_, err := s.db.Exec("DELETE FROM instances WHERE id = ?", id)
	return err
}

// UpdateInstanceField updates a single column for a given instance.
// field must be a valid column name (caller is responsible for safety).
func (s *StateDB) UpdateInstanceField(id, field string, value any) error {
	query := fmt.Sprintf("UPDATE instances SET %s = ? WHERE id = ?", field)
	_, err := s.db.Exec(query, value, id)
	return err
}

// --- Group CRUD ---

// SaveGroups replaces all groups in a single transaction.
func (s *StateDB) SaveGroups(groups []*GroupRow) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// Clear existing groups and re-insert (simpler than diff)
	if _, err := tx.Exec("DELETE FROM groups"); err != nil {
		return err
	}

	stmt, err := tx.Prepare(`
		INSERT INTO groups (path, name, expanded, sort_order, default_path)
		VALUES (?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, g := range groups {
		expanded := 0
		if g.Expanded {
			expanded = 1
		}
		if _, err := stmt.Exec(g.Path, g.Name, expanded, g.Order, g.DefaultPath); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// LoadGroups returns all groups ordered by sort_order.
func (s *StateDB) LoadGroups() ([]*GroupRow, error) {
	rows, err := s.db.Query(`
		SELECT path, name, expanded, sort_order, default_path
		FROM groups ORDER BY sort_order
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*GroupRow
	for rows.Next() {
		g := &GroupRow{}
		var expanded int
		if err := rows.Scan(&g.Path, &g.Name, &expanded, &g.Order, &g.DefaultPath); err != nil {
			return nil, err
		}
		g.Expanded = expanded != 0
		result = append(result, g)
	}
	return result, rows.Err()
}

// DeleteGroup removes a group by path.
func (s *StateDB) DeleteGroup(path string) error {
	_, err := s.db.Exec("DELETE FROM groups WHERE path = ?", path)
	return err
}

// --- Status + Acknowledgment ---

// WriteStatus updates the status and tool for an instance.
func (s *StateDB) WriteStatus(id, status, tool string) error {
	_, err := s.db.Exec(
		`UPDATE instances
		 SET status = ?, tool = ?,
		     acknowledged = CASE WHEN ? = 'running' THEN 0 ELSE acknowledged END
		 WHERE id = ?`,
		status, tool, status, id,
	)
	return err
}

// ReadAllStatuses returns status + acknowledged flag for every instance.
func (s *StateDB) ReadAllStatuses() (map[string]StatusRow, error) {
	rows, err := s.db.Query("SELECT id, status, tool, acknowledged FROM instances")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]StatusRow)
	for rows.Next() {
		var id string
		var sr StatusRow
		var ack int
		if err := rows.Scan(&id, &sr.Status, &sr.Tool, &ack); err != nil {
			return nil, err
		}
		sr.Acknowledged = ack != 0
		result[id] = sr
	}
	return result, rows.Err()
}

// SetAcknowledged sets or clears the acknowledged flag for an instance.
func (s *StateDB) SetAcknowledged(id string, ack bool) error {
	v := 0
	if ack {
		v = 1
	}
	_, err := s.db.Exec("UPDATE instances SET acknowledged = ? WHERE id = ?", v, id)
	return err
}

// --- Heartbeat ---

// RegisterInstance records this process as an active TUI instance.
func (s *StateDB) RegisterInstance(isPrimary bool) error {
	now := time.Now().Unix()
	primary := 0
	if isPrimary {
		primary = 1
	}
	_, err := s.db.Exec(`
		INSERT OR REPLACE INTO instance_heartbeats (pid, started, heartbeat, is_primary)
		VALUES (?, ?, ?, ?)
	`, s.pid, now, now, primary)
	return err
}

// Heartbeat updates the heartbeat timestamp for this process.
func (s *StateDB) Heartbeat() error {
	_, err := s.db.Exec(
		"UPDATE instance_heartbeats SET heartbeat = ? WHERE pid = ?",
		time.Now().Unix(), s.pid,
	)
	return err
}

// UnregisterInstance removes this process from the heartbeat table.
func (s *StateDB) UnregisterInstance() error {
	_, err := s.db.Exec("DELETE FROM instance_heartbeats WHERE pid = ?", s.pid)
	return err
}

// CleanDeadInstances removes heartbeat entries that haven't been updated within timeout.
func (s *StateDB) CleanDeadInstances(timeout time.Duration) error {
	cutoff := time.Now().Add(-timeout).Unix()
	_, err := s.db.Exec("DELETE FROM instance_heartbeats WHERE heartbeat < ?", cutoff)
	return err
}

// AliveInstanceCount returns how many TUI instances have fresh heartbeats.
func (s *StateDB) AliveInstanceCount() (int, error) {
	var count int
	cutoff := time.Now().Add(-30 * time.Second).Unix()
	err := s.db.QueryRow(
		"SELECT COUNT(*) FROM instance_heartbeats WHERE heartbeat >= ?", cutoff,
	).Scan(&count)
	return count, err
}

// --- Primary Election ---

// ElectPrimary attempts to make this instance the primary.
// Returns true if this instance is now (or already was) the primary.
// Uses a transaction to atomically clear stale primaries and claim if available.
func (s *StateDB) ElectPrimary(timeout time.Duration) (bool, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return false, fmt.Errorf("statedb: begin elect: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	cutoff := time.Now().Add(-timeout).Unix()

	// Clear is_primary for any heartbeat older than timeout (stale primary)
	if _, err := tx.Exec(
		"UPDATE instance_heartbeats SET is_primary = 0 WHERE heartbeat < ? AND is_primary = 1",
		cutoff,
	); err != nil {
		return false, fmt.Errorf("statedb: clear stale primary: %w", err)
	}

	// Check if any alive instance already has is_primary=1
	var existingPID int
	err = tx.QueryRow(
		"SELECT pid FROM instance_heartbeats WHERE is_primary = 1 AND heartbeat >= ? LIMIT 1",
		cutoff,
	).Scan(&existingPID)

	if err == nil {
		// An alive primary exists
		if err := tx.Commit(); err != nil {
			return false, fmt.Errorf("statedb: commit elect: %w", err)
		}
		return existingPID == s.pid, nil
	}

	// No alive primary exists: claim it
	if _, err := tx.Exec(
		"UPDATE instance_heartbeats SET is_primary = 1 WHERE pid = ?",
		s.pid,
	); err != nil {
		return false, fmt.Errorf("statedb: claim primary: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("statedb: commit elect: %w", err)
	}
	return true, nil
}

// ResignPrimary clears the is_primary flag for this process.
func (s *StateDB) ResignPrimary() error {
	_, err := s.db.Exec(
		"UPDATE instance_heartbeats SET is_primary = 0 WHERE pid = ?",
		s.pid,
	)
	return err
}

// --- Kanban Config CRUD ---

// SaveGroupKanbanConfig inserts or updates a group's kanban configuration.
func (s *StateDB) SaveGroupKanbanConfig(config *GroupKanbanConfigRow) error {
	enabled := 0
	if config.KanbanEnabled {
		enabled = 1
	}
	_, err := s.db.Exec(`
		INSERT OR REPLACE INTO group_kanban_configs (group_path, kanban_enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?)
	`, config.GroupPath, enabled, config.CreatedAt, config.UpdatedAt)
	return err
}

// LoadGroupKanbanConfig retrieves a group's kanban config. Returns nil if not found.
func (s *StateDB) LoadGroupKanbanConfig(groupPath string) (*GroupKanbanConfigRow, error) {
	var config GroupKanbanConfigRow
	var enabled int
	err := s.db.QueryRow(`
		SELECT group_path, kanban_enabled, created_at, updated_at
		FROM group_kanban_configs WHERE group_path = ?
	`, groupPath).Scan(&config.GroupPath, &enabled, &config.CreatedAt, &config.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	config.KanbanEnabled = enabled != 0
	return &config, nil
}

// LoadAllGroupKanbanConfigs returns all group kanban configurations.
func (s *StateDB) LoadAllGroupKanbanConfigs() ([]*GroupKanbanConfigRow, error) {
	rows, err := s.db.Query(`
		SELECT group_path, kanban_enabled, created_at, updated_at
		FROM group_kanban_configs
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*GroupKanbanConfigRow
	for rows.Next() {
		var config GroupKanbanConfigRow
		var enabled int
		if err := rows.Scan(&config.GroupPath, &enabled, &config.CreatedAt, &config.UpdatedAt); err != nil {
			return nil, err
		}
		config.KanbanEnabled = enabled != 0
		result = append(result, &config)
	}
	return result, rows.Err()
}

// SaveColumnSkillMapping inserts or updates a column skill mapping.
// Uses UNIQUE constraint on (group_path, column_name) for upsert behavior.
func (s *StateDB) SaveColumnSkillMapping(mapping *ColumnSkillMappingRow) error {
	autoTrigger := 0
	if mapping.AutoTrigger {
		autoTrigger = 1
	}
	triggerOnEnter := 0
	if mapping.TriggerOnEnter {
		triggerOnEnter = 1
	}

	_, err := s.db.Exec(`
		INSERT OR REPLACE INTO column_skill_mappings
		(group_path, column_name, skill_name, auto_trigger, trigger_on_enter)
		VALUES (?, ?, ?, ?, ?)
	`, mapping.GroupPath, mapping.ColumnName, mapping.SkillName, autoTrigger, triggerOnEnter)
	return err
}

// LoadColumnSkillMappings returns all column skill mappings for a group.
func (s *StateDB) LoadColumnSkillMappings(groupPath string) ([]*ColumnSkillMappingRow, error) {
	rows, err := s.db.Query(`
		SELECT id, group_path, column_name, skill_name, auto_trigger, trigger_on_enter
		FROM column_skill_mappings WHERE group_path = ?
	`, groupPath)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*ColumnSkillMappingRow
	for rows.Next() {
		var mapping ColumnSkillMappingRow
		var autoTrigger, triggerOnEnter int
		if err := rows.Scan(
			&mapping.ID, &mapping.GroupPath, &mapping.ColumnName,
			&mapping.SkillName, &autoTrigger, &triggerOnEnter,
		); err != nil {
			return nil, err
		}
		mapping.AutoTrigger = autoTrigger != 0
		mapping.TriggerOnEnter = triggerOnEnter != 0
		result = append(result, &mapping)
	}
	return result, rows.Err()
}

// DeleteColumnSkillMapping removes a column skill mapping.
func (s *StateDB) DeleteColumnSkillMapping(groupPath, columnName string) error {
	_, err := s.db.Exec(
		"DELETE FROM column_skill_mappings WHERE group_path = ? AND column_name = ?",
		groupPath, columnName,
	)
	return err
}

// --- Metadata ---

// SetMeta sets a key-value pair in the metadata table.
func (s *StateDB) SetMeta(key, value string) error {
	_, err := s.db.Exec(
		"INSERT OR REPLACE INTO metadata (key, value) VALUES (?, ?)",
		key, value,
	)
	return err
}

// GetMeta gets a value from the metadata table. Returns "" if not found.
func (s *StateDB) GetMeta(key string) (string, error) {
	var value string
	err := s.db.QueryRow("SELECT value FROM metadata WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

// --- Change Detection (replaces fsnotify) ---

// Touch updates a metadata timestamp that other instances can poll to detect changes.
func (s *StateDB) Touch() error {
	return s.SetMeta("last_modified", fmt.Sprintf("%d", time.Now().UnixNano()))
}

// LastModified returns the last_modified timestamp from metadata.
func (s *StateDB) LastModified() (int64, error) {
	val, err := s.GetMeta("last_modified")
	if err != nil || val == "" {
		return 0, err
	}
	var ts int64
	_, err = fmt.Sscanf(val, "%d", &ts)
	return ts, err
}

// --- Recent Sessions ---

// recentSessionDedupID returns a deterministic key for deduplication.
// It includes all persisted recreation fields so different launch configs do
// not overwrite each other.
func recentSessionDedupID(row *RecentSessionRow) string {
	toolOpts := "{}"
	if len(row.ToolOptions) > 0 {
		toolOpts = string(row.ToolOptions)
	}

	geminiYolo := "unset"
	if row.GeminiYoloMode != nil {
		geminiYolo = strconv.FormatBool(*row.GeminiYoloMode)
	}

	payload := strings.Join([]string{
		row.Title,
		row.ProjectPath,
		row.GroupPath,
		row.Command,
		row.Wrapper,
		row.Tool,
		toolOpts,
		strconv.FormatBool(row.SandboxEnabled),
		geminiYolo,
	}, "\x00")

	h := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(h[:16]) // 32-char hex
}

// SaveRecentSession inserts or replaces a recent session entry, then prunes to 20.
func (s *StateDB) SaveRecentSession(row *RecentSessionRow) error {
	id := recentSessionDedupID(row)

	toolOpts := row.ToolOptions
	if len(toolOpts) == 0 {
		toolOpts = json.RawMessage("{}")
	}

	sandbox := 0
	if row.SandboxEnabled {
		sandbox = 1
	}

	var geminiYolo *int
	if row.GeminiYoloMode != nil {
		v := 0
		if *row.GeminiYoloMode {
			v = 1
		}
		geminiYolo = &v
	}

	_, err := s.db.Exec(`
		INSERT OR REPLACE INTO recent_sessions (
			id, title, project_path, group_path,
			command, wrapper, tool, tool_options,
			sandbox_enabled, gemini_yolo, deleted_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		id, row.Title, row.ProjectPath, row.GroupPath,
		row.Command, row.Wrapper, row.Tool, string(toolOpts),
		sandbox, geminiYolo, time.Now().Unix(),
	)
	if err != nil {
		return err
	}

	return s.pruneRecentSessions(20)
}

// LoadRecentSessions returns all recent sessions ordered by most recently deleted.
func (s *StateDB) LoadRecentSessions() ([]*RecentSessionRow, error) {
	rows, err := s.db.Query(`
		SELECT id, title, project_path, group_path,
			command, wrapper, tool, tool_options,
			sandbox_enabled, gemini_yolo, deleted_at
		FROM recent_sessions ORDER BY deleted_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*RecentSessionRow
	for rows.Next() {
		r := &RecentSessionRow{}
		var toolOptsStr string
		var sandbox int
		var geminiYolo *int
		var deletedUnix int64
		if err := rows.Scan(
			&r.ID, &r.Title, &r.ProjectPath, &r.GroupPath,
			&r.Command, &r.Wrapper, &r.Tool, &toolOptsStr,
			&sandbox, &geminiYolo, &deletedUnix,
		); err != nil {
			return nil, err
		}
		r.ToolOptions = json.RawMessage(toolOptsStr)
		r.SandboxEnabled = sandbox != 0
		if geminiYolo != nil {
			v := *geminiYolo != 0
			r.GeminiYoloMode = &v
		}
		r.DeletedAt = time.Unix(deletedUnix, 0)
		result = append(result, r)
	}
	return result, rows.Err()
}

// pruneRecentSessions keeps only the maxCount most recent entries.
func (s *StateDB) pruneRecentSessions(maxCount int) error {
	_, err := s.db.Exec(`
		DELETE FROM recent_sessions WHERE id NOT IN (
			SELECT id FROM recent_sessions ORDER BY deleted_at DESC LIMIT ?
		)
	`, maxCount)
	return err
}
