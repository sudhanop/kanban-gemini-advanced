package database

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/kanban-platform/backend/internal/config"
	"github.com/kanban-platform/backend/internal/models"
)

var (
	saveMutex sync.Mutex
)

// dbFilePath returns the path to db.json, respecting DATA_DIR env var for Docker support.
func dbFilePath() string {
	if dir := os.Getenv("DATA_DIR"); dir != "" {
		_ = os.MkdirAll(dir, 0755)
		return filepath.Join(dir, "db.json")
	}
	return "db.json"
}

// JSONDatabaseState represents the full DB schema in JSON format
type JSONDatabaseState struct {
	Users                  []models.User                  `json:"users"`
	UserSessions           []models.UserSession           `json:"user_sessions"`
	Workspaces             []models.Workspace             `json:"workspaces"`
	WorkspaceMembers       []models.WorkspaceMember       `json:"workspace_members"`
	Boards                 []models.Board                 `json:"boards"`
	BoardMembers           []models.BoardMember           `json:"board_members"`
	Columns                []models.Column                `json:"columns"`
	Tasks                  []models.Task                  `json:"tasks"`
	TaskAssignees          []models.TaskAssignee          `json:"task_assignees"`
	TaskWatchers           []models.TaskWatcher           `json:"task_watchers"`
	TaskLabels             []models.TaskLabel             `json:"task_labels"`
	TaskDependencies       []models.TaskDependency        `json:"task_dependencies"`
	Subtasks               []models.Subtask               `json:"subtasks"`
	Comments               []models.Comment               `json:"comments"`
	CommentReactions       []models.CommentReaction       `json:"comment_reactions"`
	Reminders              []models.Reminder              `json:"reminders"`
	ReminderLogs           []models.ReminderLog           `json:"reminder_logs"`
	Invitations            []models.Invitation            `json:"invitations"`
	Notifications          []models.Notification          `json:"notifications"`
	Meetings               []models.Meeting               `json:"meetings"`
	MeetingParticipants    []models.MeetingParticipant    `json:"meeting_participants"`
	ActivityLogs           []models.ActivityLog           `json:"activity_logs"`
	SprintPlans            []models.SprintPlan            `json:"sprint_plans"`
	SprintTasks            []models.SprintTask            `json:"sprint_tasks"`
	Attachments            []models.Attachment            `json:"attachments"`
	AutomationRules        []models.AutomationRule        `json:"automation_rules"`
	BoardTemplates         []models.BoardTemplate         `json:"board_templates"`
	CustomFields           []models.CustomField           `json:"custom_fields"`
	TaskCustomFieldValues  []models.TaskCustomFieldValue  `json:"task_custom_field_values"`
	AuditLogs              []models.AuditLog              `json:"audit_logs"`
	SuspiciousActivities   []models.SuspiciousActivity    `json:"suspicious_activities"`
}

// InitSQLite initializes an in-memory SQLite database
func InitSQLite(cfg *config.Config) (*gorm.DB, error) {
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	}
	if cfg.AppEnv == "development" {
		gormConfig.Logger = logger.Default.LogMode(logger.Info)
	}

	// Connect to in-memory SQLite database with shared cache so it persists for the process lifetime
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	return db, nil
}

// AutoMigrateSQLite runs SQLite migrations
func AutoMigrateSQLite(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.User{},
		&models.UserSession{},
		&models.Workspace{},
		&models.WorkspaceMember{},
		&models.Board{},
		&models.BoardMember{},
		&models.Column{},
		&models.Task{},
		&models.TaskAssignee{},
		&models.TaskWatcher{},
		&models.TaskLabel{},
		&models.TaskDependency{},
		&models.Subtask{},
		&models.Comment{},
		&models.CommentReaction{},
		&models.Reminder{},
		&models.ReminderLog{},
		&models.Invitation{},
		&models.Notification{},
		&models.Meeting{},
		&models.MeetingParticipant{},
		&models.ActivityLog{},
		&models.SprintPlan{},
		&models.SprintTask{},
		&models.Attachment{},
		&models.AutomationRule{},
		&models.BoardTemplate{},
		&models.CustomField{},
		&models.TaskCustomFieldValue{},
		&models.AuditLog{},
		&models.SuspiciousActivity{},
	)
}

// LoadFromJSON loads data from db.json into the SQLite database
func LoadFromJSON(db *gorm.DB) error {
	path := dbFilePath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil // No file, which is normal for first start
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", path, err)
	}

	var state JSONDatabaseState
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("failed to unmarshal db.json: %w", err)
	}

	db.Exec("PRAGMA foreign_keys = OFF")
	defer db.Exec("PRAGMA foreign_keys = ON")

	// Helper to load collections into memory SQLite DB
	loadTable := func(slice interface{}) {
		// Use a transaction or simply Create to insert
		db.Session(&gorm.Session{SkipHooks: true}).Create(slice)
	}

	if len(state.Users) > 0 { loadTable(&state.Users) }
	if len(state.UserSessions) > 0 { loadTable(&state.UserSessions) }
	if len(state.Workspaces) > 0 { loadTable(&state.Workspaces) }
	if len(state.WorkspaceMembers) > 0 { loadTable(&state.WorkspaceMembers) }
	if len(state.Boards) > 0 { loadTable(&state.Boards) }
	if len(state.BoardMembers) > 0 { loadTable(&state.BoardMembers) }
	if len(state.Columns) > 0 { loadTable(&state.Columns) }
	if len(state.Tasks) > 0 { loadTable(&state.Tasks) }
	if len(state.TaskAssignees) > 0 { loadTable(&state.TaskAssignees) }
	if len(state.TaskWatchers) > 0 { loadTable(&state.TaskWatchers) }
	if len(state.TaskLabels) > 0 { loadTable(&state.TaskLabels) }
	if len(state.TaskDependencies) > 0 { loadTable(&state.TaskDependencies) }
	if len(state.Subtasks) > 0 { loadTable(&state.Subtasks) }
	if len(state.Comments) > 0 { loadTable(&state.Comments) }
	if len(state.CommentReactions) > 0 { loadTable(&state.CommentReactions) }
	if len(state.Reminders) > 0 { loadTable(&state.Reminders) }
	if len(state.ReminderLogs) > 0 { loadTable(&state.ReminderLogs) }
	if len(state.Invitations) > 0 { loadTable(&state.Invitations) }
	if len(state.Notifications) > 0 { loadTable(&state.Notifications) }
	if len(state.Meetings) > 0 { loadTable(&state.Meetings) }
	if len(state.MeetingParticipants) > 0 { loadTable(&state.MeetingParticipants) }
	if len(state.ActivityLogs) > 0 { loadTable(&state.ActivityLogs) }
	if len(state.SprintPlans) > 0 { loadTable(&state.SprintPlans) }
	if len(state.SprintTasks) > 0 { loadTable(&state.SprintTasks) }
	if len(state.Attachments) > 0 { loadTable(&state.Attachments) }
	if len(state.AutomationRules) > 0 { loadTable(&state.AutomationRules) }
	if len(state.BoardTemplates) > 0 { loadTable(&state.BoardTemplates) }
	if len(state.CustomFields) > 0 { loadTable(&state.CustomFields) }
	if len(state.TaskCustomFieldValues) > 0 { loadTable(&state.TaskCustomFieldValues) }
	if len(state.AuditLogs) > 0 { loadTable(&state.AuditLogs) }
	if len(state.SuspiciousActivities) > 0 { loadTable(&state.SuspiciousActivities) }

	return nil
}

// SaveToJSON saves all database tables into db.json
func SaveToJSON(db *gorm.DB) error {
	saveMutex.Lock()
	defer saveMutex.Unlock()

	path := dbFilePath()
	var state JSONDatabaseState
	
	// Query all data, skipping deleted records (standard soft-delete behavior)
	db.Session(&gorm.Session{SkipHooks: true}).Find(&state.Users)
	db.Session(&gorm.Session{SkipHooks: true}).Find(&state.UserSessions)
	db.Session(&gorm.Session{SkipHooks: true}).Find(&state.Workspaces)
	db.Session(&gorm.Session{SkipHooks: true}).Find(&state.WorkspaceMembers)
	db.Session(&gorm.Session{SkipHooks: true}).Find(&state.Boards)
	db.Session(&gorm.Session{SkipHooks: true}).Find(&state.BoardMembers)
	db.Session(&gorm.Session{SkipHooks: true}).Find(&state.Columns)
	db.Session(&gorm.Session{SkipHooks: true}).Find(&state.Tasks)
	db.Session(&gorm.Session{SkipHooks: true}).Find(&state.TaskAssignees)
	db.Session(&gorm.Session{SkipHooks: true}).Find(&state.TaskWatchers)
	db.Session(&gorm.Session{SkipHooks: true}).Find(&state.TaskLabels)
	db.Session(&gorm.Session{SkipHooks: true}).Find(&state.TaskDependencies)
	db.Session(&gorm.Session{SkipHooks: true}).Find(&state.Subtasks)
	db.Session(&gorm.Session{SkipHooks: true}).Find(&state.Comments)
	db.Session(&gorm.Session{SkipHooks: true}).Find(&state.CommentReactions)
	db.Session(&gorm.Session{SkipHooks: true}).Find(&state.Reminders)
	db.Session(&gorm.Session{SkipHooks: true}).Find(&state.ReminderLogs)
	db.Session(&gorm.Session{SkipHooks: true}).Find(&state.Invitations)
	db.Session(&gorm.Session{SkipHooks: true}).Find(&state.Notifications)
	db.Session(&gorm.Session{SkipHooks: true}).Find(&state.Meetings)
	db.Session(&gorm.Session{SkipHooks: true}).Find(&state.MeetingParticipants)
	db.Session(&gorm.Session{SkipHooks: true}).Find(&state.ActivityLogs)
	db.Session(&gorm.Session{SkipHooks: true}).Find(&state.SprintPlans)
	db.Session(&gorm.Session{SkipHooks: true}).Find(&state.SprintTasks)
	db.Session(&gorm.Session{SkipHooks: true}).Find(&state.Attachments)
	db.Session(&gorm.Session{SkipHooks: true}).Find(&state.AutomationRules)
	db.Session(&gorm.Session{SkipHooks: true}).Find(&state.BoardTemplates)
	db.Session(&gorm.Session{SkipHooks: true}).Find(&state.CustomFields)
	db.Session(&gorm.Session{SkipHooks: true}).Find(&state.TaskCustomFieldValues)
	db.Session(&gorm.Session{SkipHooks: true}).Find(&state.AuditLogs)
	db.Session(&gorm.Session{SkipHooks: true}).Find(&state.SuspiciousActivities)

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	tempFile := path + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := os.Rename(tempFile, path); err != nil {
		_ = os.Remove(tempFile)
		return fmt.Errorf("failed to replace db.json: %w", err)
	}

	return nil
}

// RegisterSaveCallbacks registers GORM hooks to trigger SaveToJSON on modifications
func RegisterSaveCallbacks(db *gorm.DB) {
	saveHook := func(tx *gorm.DB) {
		if tx.Error == nil {
			_ = SaveToJSON(db)
		}
	}

	// Register after hooks for create, update, delete
	_ = db.Callback().Create().After("gorm:create").Register("save_json_after_create", saveHook)
	_ = db.Callback().Update().After("gorm:update").Register("save_json_after_update", saveHook)
	_ = db.Callback().Delete().After("gorm:delete").Register("save_json_after_delete", saveHook)
}
