package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ─── Base Model ───────────────────────────────────────────────────────────────

type Base struct {
	ID        uuid.UUID      `gorm:"type:uuid;primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (b *Base) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}

// ─── User ─────────────────────────────────────────────────────────────────────

type UserRole string
type UserStatus string

const (
	RoleOwner     UserRole = "owner"
	RoleAdmin     UserRole = "admin"
	RoleDeveloper UserRole = "developer"
	RoleViewer    UserRole = "viewer"
	RoleGuest     UserRole = "guest"
)

const (
	StatusActive    UserStatus = "active"
	StatusInactive  UserStatus = "inactive"
	StatusSuspended UserStatus = "suspended"
)

type User struct {
	Base
	Name            string     `gorm:"not null" json:"name"`
	Email           string     `gorm:"uniqueIndex;not null" json:"email"`
	GoogleID        string     `gorm:"uniqueIndex" json:"google_id"`
	Avatar          string     `json:"avatar"`
	Role            UserRole   `gorm:"default:'developer'" json:"role"`
	Status          UserStatus `gorm:"default:'active'" json:"status"`
	Timezone        string     `gorm:"default:'UTC'" json:"timezone"`
	Theme           string     `gorm:"default:'dark'" json:"theme"`
	NotifyEmail     bool       `gorm:"default:true" json:"notify_email"`
	NotifyInApp     bool       `gorm:"default:true" json:"notify_in_app"`
	LastLoginAt     *time.Time `json:"last_login_at"`
	LoginCount      int        `gorm:"default:0" json:"login_count"`
	LoginIP         string     `json:"login_ip,omitempty"`
}

type UserSession struct {
	Base
	UserID       uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	User         User      `gorm:"foreignKey:UserID" json:"-"`
	AccessToken  string    `gorm:"not null" json:"-"`
	RefreshToken string    `gorm:"uniqueIndex;not null" json:"-"`
	IPAddress    string    `json:"ip_address"`
	UserAgent    string    `json:"user_agent"`
	ExpiresAt    time.Time `json:"expires_at"`
	IsActive     bool      `gorm:"default:true" json:"is_active"`
}

// ─── Workspace ────────────────────────────────────────────────────────────────

type WorkspaceType string

const (
	WorkspacePersonal  WorkspaceType = "personal"
	WorkspaceFreelance WorkspaceType = "freelance"
	WorkspaceStartup   WorkspaceType = "startup"
	WorkspaceCompany   WorkspaceType = "company"
	WorkspaceResearch  WorkspaceType = "research"
)

type Workspace struct {
	Base
	Name        string        `gorm:"not null" json:"name"`
	Slug        string        `gorm:"uniqueIndex;not null" json:"slug"`
	Description string        `json:"description"`
	Type        WorkspaceType `gorm:"default:'personal'" json:"type"`
	OwnerID     uuid.UUID     `gorm:"type:uuid;not null" json:"owner_id"`
	Owner       User          `gorm:"foreignKey:OwnerID" json:"owner,omitempty"`
	Avatar      string        `json:"avatar"`
	Color       string        `gorm:"default:'#6366f1'" json:"color"`
	IsArchived  bool          `gorm:"default:false" json:"is_archived"`
	Members     []WorkspaceMember `gorm:"foreignKey:WorkspaceID" json:"members,omitempty"`
	Boards      []Board           `gorm:"foreignKey:WorkspaceID" json:"boards,omitempty"`
}

type WorkspaceMember struct {
	Base
	WorkspaceID uuid.UUID `gorm:"type:uuid;not null;index" json:"workspace_id"`
	UserID      uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	User        User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Role        UserRole  `gorm:"default:'developer'" json:"role"`
	JoinedAt    time.Time `json:"joined_at"`
}

// ─── Board ────────────────────────────────────────────────────────────────────

type BoardViewType string

const (
	ViewKanban   BoardViewType = "kanban"
	ViewTimeline BoardViewType = "timeline"
	ViewSprint   BoardViewType = "sprint"
	ViewCalendar BoardViewType = "calendar"
	ViewAnalytics BoardViewType = "analytics"
)

type Board struct {
	Base
	WorkspaceID  uuid.UUID     `gorm:"type:uuid;not null;index" json:"workspace_id"`
	Name         string        `gorm:"not null" json:"name"`
	Description  string        `json:"description"`
	DefaultView  BoardViewType `gorm:"default:'kanban'" json:"default_view"`
	Color        string        `gorm:"default:'#6366f1'" json:"color"`
	Icon         string        `json:"icon"`
	IsArchived   bool          `gorm:"default:false" json:"is_archived"`
	IsTemplate   bool          `gorm:"default:false" json:"is_template"`
	TemplateType string        `json:"template_type"`
	CreatedBy    uuid.UUID     `gorm:"type:uuid" json:"created_by"`
	Columns      []Column      `gorm:"foreignKey:BoardID;orderBy:order_index" json:"columns,omitempty"`
	Members      []BoardMember `gorm:"foreignKey:BoardID" json:"members,omitempty"`
}

type BoardMember struct {
	Base
	BoardID uuid.UUID `gorm:"type:uuid;not null;index" json:"board_id"`
	UserID  uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	User    User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Role    UserRole  `gorm:"default:'developer'" json:"role"`
}

// ─── Column (Stage) ───────────────────────────────────────────────────────────

type Column struct {
	Base
	BoardID    uuid.UUID `gorm:"type:uuid;not null;index" json:"board_id"`
	Name       string    `gorm:"not null" json:"name"`
	OrderIndex int       `gorm:"not null;default:0" json:"order_index"`
	Color      string    `gorm:"default:'#6366f1'" json:"color"`
	WipLimit   int       `gorm:"default:0" json:"wip_limit"`
	IsDefault  bool      `gorm:"default:false" json:"is_default"`
	StatusType string    `gorm:"default:'active'" json:"status_type"`
	Tasks      []Task    `gorm:"foreignKey:ColumnID;orderBy:order_index" json:"tasks,omitempty"`
}

// ─── Task ─────────────────────────────────────────────────────────────────────

type Priority string
type TaskStatus string

const (
	PriorityCritical Priority = "critical"
	PriorityHigh     Priority = "high"
	PriorityMedium   Priority = "medium"
	PriorityLow      Priority = "low"
	PriorityNone     Priority = "none"
)

type Task struct {
	Base
	// Identity
	BoardID     uuid.UUID  `gorm:"type:uuid;not null;index" json:"board_id"`
	WorkspaceID uuid.UUID  `gorm:"type:uuid;not null;index" json:"workspace_id"`
	ColumnID    uuid.UUID  `gorm:"type:uuid;not null;index" json:"column_id"`
	ParentID    *uuid.UUID `gorm:"type:uuid;index" json:"parent_id"`
	Title       string     `gorm:"not null" json:"title"`
	Summary     string     `json:"summary"`
	Description string     `gorm:"type:text" json:"description"`
	OrderIndex  int        `gorm:"default:0" json:"order_index"`

	// Status & Classification
	Status   string   `gorm:"default:'open'" json:"status"`
	Priority Priority `gorm:"default:'medium'" json:"priority"`
	Category string   `json:"category"`
	Color    string   `json:"color"`
	Icon     string   `json:"icon"`
	Progress int      `gorm:"default:0;check:progress >= 0 AND progress <= 100" json:"progress"`

	// Time Intelligence
	EstimatedHours   float64    `json:"estimated_hours"`
	ActualHours      float64    `json:"actual_hours"`
	TimeVariance     float64    `json:"time_variance"`
	DelayPercentage  float64    `json:"delay_percentage"`
	StartDate        *time.Time `json:"start_date"`
	DueDate          *time.Time `json:"due_date"`
	CompletedAt      *time.Time `json:"completed_at"`
	PomodoroCount    int        `gorm:"default:0" json:"pomodoro_count"`
	WorkSessionMins  int        `gorm:"default:0" json:"work_session_mins"`
	IdleTimeMins     int        `gorm:"default:0" json:"idle_time_mins"`

	// Assignment
	CreatedBy  uuid.UUID  `gorm:"type:uuid" json:"created_by"`
	ReviewerID *uuid.UUID `gorm:"type:uuid" json:"reviewer_id"`

	// Technical Requirements
	RequiredSoftware  string `json:"required_software"`
	RequiredHardware  string `json:"required_hardware"`
	RequiredAPIs      string `json:"required_apis"`
	GitHubRepo        string `json:"github_repo"`
	BranchName        string `json:"branch_name"`
	PullRequestURL    string `json:"pull_request_url"`
	DatabaseDeps      string `json:"database_deps"`
	EnvVariables      string `json:"env_variables"`
	DeploymentURL     string `json:"deployment_url"`
	DockerSupport     bool   `json:"docker_support"`
	SetupInstructions string `gorm:"type:text" json:"setup_instructions"`

	// Workflow Intelligence
	BlockedReason     string     `json:"blocked_reason"`
	WaitingFor        string     `json:"waiting_for"`
	ApprovalRequired  bool       `gorm:"default:false" json:"approval_required"`
	ApprovedBy        *uuid.UUID `gorm:"type:uuid" json:"approved_by"`
	ApprovedAt        *time.Time `json:"approved_at"`
	QAStatus          string     `json:"qa_status"`
	DeploymentStatus  string     `json:"deployment_status"`
	RollbackTracking  string     `json:"rollback_tracking"`
	IncidentNotes     string     `gorm:"type:text" json:"incident_notes"`

	// AI Intelligence
	AIRiskScore          float64 `json:"ai_risk_score"`
	AIProductivityScore  float64 `json:"ai_productivity_score"`
	AICompletionPredict  float64 `json:"ai_completion_predict"`
	AITimePrediction     float64 `json:"ai_time_prediction"`

	// Business Data
	StoryPoints     int     `json:"story_points"`
	Budget          float64 `json:"budget"`
	CostEstimation  float64 `json:"cost_estimation"`
	RevenueImpact   float64 `json:"revenue_impact"`
	ClientName      string  `json:"client_name"`
	RiskLevel       string  `json:"risk_level"`
	SecurityClass   string  `json:"security_class"`

	// Sprint
	SprintID *uuid.UUID `gorm:"type:uuid;index" json:"sprint_id"`

	// Relations
	Subtasks     []Subtask     `gorm:"foreignKey:ParentID" json:"subtasks,omitempty"`
	Assignees    []TaskAssignee `gorm:"foreignKey:TaskID" json:"assignees,omitempty"`
	Watchers     []TaskWatcher  `gorm:"foreignKey:TaskID" json:"watchers,omitempty"`
	Labels       []TaskLabel    `gorm:"foreignKey:TaskID" json:"labels,omitempty"`
	Comments     []Comment      `gorm:"foreignKey:TaskID" json:"comments,omitempty"`
	Attachments  []Attachment   `gorm:"foreignKey:TaskID" json:"attachments,omitempty"`
	Dependencies []TaskDependency `gorm:"foreignKey:TaskID" json:"dependencies,omitempty"`
}

type TaskAssignee struct {
	Base
	TaskID uuid.UUID `gorm:"type:uuid;not null;index" json:"task_id"`
	UserID uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	User   User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

type TaskWatcher struct {
	Base
	TaskID uuid.UUID `gorm:"type:uuid;not null;index" json:"task_id"`
	UserID uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	User   User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

type TaskLabel struct {
	Base
	TaskID uuid.UUID `gorm:"type:uuid;not null;index" json:"task_id"`
	Label  string    `gorm:"not null" json:"label"`
	Color  string    `gorm:"default:'#6366f1'" json:"color"`
}

type TaskDependency struct {
	Base
	TaskID       uuid.UUID `gorm:"type:uuid;not null;index" json:"task_id"`
	DependsOnID  uuid.UUID `gorm:"type:uuid;not null;index" json:"depends_on_id"`
	DependsOn    Task      `gorm:"foreignKey:DependsOnID" json:"depends_on,omitempty"`
	DependencyType string  `gorm:"default:'blocks'" json:"dependency_type"`
}

// ─── Subtask ──────────────────────────────────────────────────────────────────

type Subtask struct {
	Base
	ParentID    uuid.UUID  `gorm:"type:uuid;not null;index" json:"parent_id"`
	Title       string     `gorm:"not null" json:"title"`
	Description string     `json:"description"`
	IsCompleted bool       `gorm:"default:false" json:"is_completed"`
	AssigneeID  *uuid.UUID `gorm:"type:uuid" json:"assignee_id"`
	Assignee    *User      `gorm:"foreignKey:AssigneeID" json:"assignee,omitempty"`
	DueDate     *time.Time `json:"due_date"`
	OrderIndex  int        `gorm:"default:0" json:"order_index"`
}

// ─── Comment ──────────────────────────────────────────────────────────────────

type Comment struct {
	Base
	TaskID   uuid.UUID  `gorm:"type:uuid;not null;index" json:"task_id"`
	UserID   uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	User     User       `gorm:"foreignKey:UserID" json:"user,omitempty"`
	ParentID *uuid.UUID `gorm:"type:uuid;index" json:"parent_id"`
	Content  string     `gorm:"type:text;not null" json:"content"`
	IsEdited bool       `gorm:"default:false" json:"is_edited"`
	Replies  []Comment  `gorm:"foreignKey:ParentID" json:"replies,omitempty"`
	Reactions []CommentReaction `gorm:"foreignKey:CommentID" json:"reactions,omitempty"`
}

type CommentReaction struct {
	Base
	CommentID uuid.UUID `gorm:"type:uuid;not null;index" json:"comment_id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	User      User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Emoji     string    `gorm:"not null" json:"emoji"`
}

// ─── Reminder ─────────────────────────────────────────────────────────────────

type ReminderFrequency string
type ReminderStatus string
type ReminderLevel int

const (
	FrequencyOnce    ReminderFrequency = "once"
	FrequencyDaily   ReminderFrequency = "daily"
	FrequencyWeekly  ReminderFrequency = "weekly"
	FrequencyMonthly ReminderFrequency = "monthly"
	FrequencyCustom  ReminderFrequency = "custom"

	ReminderActive  ReminderStatus = "active"
	ReminderPaused  ReminderStatus = "paused"
	ReminderDone    ReminderStatus = "done"
	ReminderExpired ReminderStatus = "expired"

	Level1 ReminderLevel = 1
	Level2 ReminderLevel = 2
	Level3 ReminderLevel = 3
)

type Reminder struct {
	Base
	TaskID          uuid.UUID         `gorm:"type:uuid;not null;index" json:"task_id"`
	Task            Task              `gorm:"foreignKey:TaskID" json:"task,omitempty"`
	UserID          uuid.UUID         `gorm:"type:uuid;not null;index" json:"user_id"`
	User            User              `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Title           string            `json:"title"`
	Frequency       ReminderFrequency `gorm:"default:'once'" json:"frequency"`
	CronExpression  string            `json:"cron_expression"`
	Status          ReminderStatus    `gorm:"default:'active'" json:"status"`
	CurrentLevel    ReminderLevel     `gorm:"default:1" json:"current_level"`
	Level1At        *time.Time        `json:"level1_at"`
	Level2At        *time.Time        `json:"level2_at"`
	Level3At        *time.Time        `json:"level3_at"`
	NextRunAt       *time.Time        `gorm:"index" json:"next_run_at"`
	LastRunAt       *time.Time        `json:"last_run_at"`
	IgnoreCount     int               `gorm:"default:0" json:"ignore_count"`
	EmailEnabled    bool              `gorm:"default:true" json:"email_enabled"`
	InAppEnabled    bool              `gorm:"default:true" json:"in_app_enabled"`
	EscalateAfter   int               `gorm:"default:3" json:"escalate_after"`
}

type ReminderLog struct {
	Base
	ReminderID uuid.UUID     `gorm:"type:uuid;not null;index" json:"reminder_id"`
	Level      ReminderLevel `json:"level"`
	Channel    string        `json:"channel"`
	SentAt     time.Time     `json:"sent_at"`
	Status     string        `json:"status"`
	Error      string        `json:"error"`
}

// ─── Invitation ───────────────────────────────────────────────────────────────

type InvitationStatus string

const (
	InvitationPending  InvitationStatus = "pending"
	InvitationAccepted InvitationStatus = "accepted"
	InvitationExpired  InvitationStatus = "expired"
	InvitationRevoked  InvitationStatus = "revoked"
)

type Invitation struct {
	Base
	WorkspaceID uuid.UUID        `gorm:"type:uuid;not null;index" json:"workspace_id"`
	BoardID     *uuid.UUID       `gorm:"type:uuid;index" json:"board_id"`
	Email       string           `gorm:"not null;index" json:"email"`
	Role        UserRole         `gorm:"default:'developer'" json:"role"`
	Token       string           `gorm:"uniqueIndex;not null" json:"token"`
	InvitedBy   uuid.UUID        `gorm:"type:uuid;not null" json:"invited_by"`
	InvitedByUser User           `gorm:"foreignKey:InvitedBy" json:"invited_by_user,omitempty"`
	Status      InvitationStatus `gorm:"default:'pending'" json:"status"`
	ExpiresAt   time.Time        `json:"expires_at"`
	AcceptedAt  *time.Time       `json:"accepted_at"`
}

// ─── Notification ─────────────────────────────────────────────────────────────

type NotificationType string

const (
	NotifTaskAssigned   NotificationType = "task_assigned"
	NotifTaskCommented  NotificationType = "task_commented"
	NotifTaskDue        NotificationType = "task_due"
	NotifReminderFired  NotificationType = "reminder_fired"
	NotifInvitation     NotificationType = "invitation"
	NotifMentioned      NotificationType = "mentioned"
	NotifMeetingCreated NotificationType = "meeting_created"
	NotifTaskMoved      NotificationType = "task_moved"
	NotifTaskCompleted  NotificationType = "task_completed"
)

type Notification struct {
	Base
	UserID      uuid.UUID        `gorm:"type:uuid;not null;index" json:"user_id"`
	Type        NotificationType `gorm:"not null" json:"type"`
	Title       string           `gorm:"not null" json:"title"`
	Message     string           `gorm:"type:text" json:"message"`
	EntityID    *uuid.UUID       `gorm:"type:uuid" json:"entity_id"`
	EntityType  string           `json:"entity_type"`
	IsRead      bool             `gorm:"default:false;index" json:"is_read"`
	ReadAt      *time.Time       `json:"read_at"`
	ActionURL   string           `json:"action_url"`
}

// ─── Meeting ──────────────────────────────────────────────────────────────────

type MeetingType string

const (
	MeetingJitsi  MeetingType = "jitsi"
	MeetingGMeet  MeetingType = "google_meet"
	MeetingZoom   MeetingType = "zoom"
	MeetingCustom MeetingType = "custom"
)

type Meeting struct {
	Base
	WorkspaceID  uuid.UUID   `gorm:"type:uuid;not null;index" json:"workspace_id"`
	BoardID      *uuid.UUID  `gorm:"type:uuid;index" json:"board_id"`
	TaskID       *uuid.UUID  `gorm:"type:uuid;index" json:"task_id"`
	Title        string      `gorm:"not null" json:"title"`
	Description  string      `json:"description"`
	MeetingType  MeetingType `gorm:"default:'jitsi'" json:"meeting_type"`
	MeetingLink  string      `json:"meeting_link"`
	RoomName     string      `json:"room_name"`
	ScheduledAt  *time.Time  `json:"scheduled_at"`
	Duration     int         `json:"duration"`
	CreatedBy    uuid.UUID   `gorm:"type:uuid;not null" json:"created_by"`
	Participants []MeetingParticipant `gorm:"foreignKey:MeetingID" json:"participants,omitempty"`
}

type MeetingParticipant struct {
	Base
	MeetingID uuid.UUID `gorm:"type:uuid;not null;index" json:"meeting_id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	User      User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Status    string    `gorm:"default:'invited'" json:"status"`
}

// ─── Activity Log ─────────────────────────────────────────────────────────────

type ActivityLog struct {
	Base
	WorkspaceID uuid.UUID  `gorm:"type:uuid;index" json:"workspace_id"`
	BoardID     *uuid.UUID `gorm:"type:uuid;index" json:"board_id"`
	TaskID      *uuid.UUID `gorm:"type:uuid;index" json:"task_id"`
	UserID      uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	User        User       `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Action      string     `gorm:"not null" json:"action"`
	EntityType  string     `json:"entity_type"`
	EntityID    *uuid.UUID `gorm:"type:uuid" json:"entity_id"`
	OldValue    string     `gorm:"type:text" json:"old_value"`
	NewValue    string     `gorm:"type:text" json:"new_value"`
	IPAddress   string     `json:"ip_address"`
	UserAgent   string     `json:"user_agent"`
}

// ─── Sprint ───────────────────────────────────────────────────────────────────

type SprintStatus string

const (
	SprintPlanning   SprintStatus = "planning"
	SprintActive     SprintStatus = "active"
	SprintCompleted  SprintStatus = "completed"
	SprintCancelled  SprintStatus = "cancelled"
)

type SprintPlan struct {
	Base
	BoardID     uuid.UUID    `gorm:"type:uuid;not null;index" json:"board_id"`
	Name        string       `gorm:"not null" json:"name"`
	Goal        string       `gorm:"type:text" json:"goal"`
	Status      SprintStatus `gorm:"default:'planning'" json:"status"`
	StartDate   *time.Time   `json:"start_date"`
	EndDate     *time.Time   `json:"end_date"`
	Velocity    float64      `json:"velocity"`
	CompletedAt *time.Time   `json:"completed_at"`
	SprintTasks []SprintTask `gorm:"foreignKey:SprintID" json:"sprint_tasks,omitempty"`
}

type SprintTask struct {
	Base
	SprintID uuid.UUID `gorm:"type:uuid;not null;index" json:"sprint_id"`
	TaskID   uuid.UUID `gorm:"type:uuid;not null;index" json:"task_id"`
	Task     Task      `gorm:"foreignKey:TaskID" json:"task,omitempty"`
	AddedAt  time.Time `json:"added_at"`
}

// ─── Attachment ───────────────────────────────────────────────────────────────

type Attachment struct {
	Base
	TaskID      uuid.UUID `gorm:"type:uuid;not null;index" json:"task_id"`
	UploadedBy  uuid.UUID `gorm:"type:uuid;not null" json:"uploaded_by"`
	User        User      `gorm:"foreignKey:UploadedBy" json:"user,omitempty"`
	FileName    string    `gorm:"not null" json:"file_name"`
	FileSize    int64     `json:"file_size"`
	FileType    string    `json:"file_type"`
	StoragePath string    `gorm:"not null" json:"storage_path"`
	PublicURL   string    `json:"public_url"`
}

// ─── Automation Rule ──────────────────────────────────────────────────────────

type AutomationRule struct {
	Base
	BoardID     uuid.UUID `gorm:"type:uuid;not null;index" json:"board_id"`
	WorkspaceID uuid.UUID `gorm:"type:uuid;not null;index" json:"workspace_id"`
	Name        string    `gorm:"not null" json:"name"`
	TriggerType string    `gorm:"not null" json:"trigger_type"`
	TriggerData string    `gorm:"type:jsonb" json:"trigger_data"`
	ActionType  string    `gorm:"not null" json:"action_type"`
	ActionData  string    `gorm:"type:jsonb" json:"action_data"`
	IsEnabled   bool      `gorm:"default:true" json:"is_enabled"`
	RunCount    int       `gorm:"default:0" json:"run_count"`
	CreatedBy   uuid.UUID `gorm:"type:uuid;not null" json:"created_by"`
}

// ─── Board Template ───────────────────────────────────────────────────────────

type BoardTemplate struct {
	Base
	Name         string `gorm:"not null;uniqueIndex" json:"name"`
	Description  string `json:"description"`
	TemplateType string `json:"template_type"`
	Color        string `json:"color"`
	Icon         string `json:"icon"`
	ColumnConfig string `gorm:"type:jsonb" json:"column_config"`
	IsBuiltIn    bool   `gorm:"default:true" json:"is_built_in"`
}

// ─── Custom Field ─────────────────────────────────────────────────────────────

type CustomField struct {
	Base
	BoardID     uuid.UUID `gorm:"type:uuid;not null;index" json:"board_id"`
	Name        string    `gorm:"not null" json:"name"`
	FieldType   string    `gorm:"not null" json:"field_type"`
	Options     string    `gorm:"type:jsonb" json:"options"`
	IsRequired  bool      `gorm:"default:false" json:"is_required"`
	OrderIndex  int       `gorm:"default:0" json:"order_index"`
}

type TaskCustomFieldValue struct {
	Base
	TaskID        uuid.UUID `gorm:"type:uuid;not null;index" json:"task_id"`
	CustomFieldID uuid.UUID `gorm:"type:uuid;not null;index" json:"custom_field_id"`
	Value         string    `gorm:"type:text" json:"value"`
}

// ─── Audit Log ────────────────────────────────────────────────────────────────

type AuditLog struct {
	Base
	UserID      uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	WorkspaceID *uuid.UUID `gorm:"type:uuid;index" json:"workspace_id"`
	Action      string    `gorm:"not null" json:"action"`
	Resource    string    `gorm:"not null" json:"resource"`
	ResourceID  *uuid.UUID `gorm:"type:uuid" json:"resource_id"`
	Details     string    `gorm:"type:jsonb" json:"details"`
	IPAddress   string    `json:"ip_address"`
	UserAgent   string    `json:"user_agent"`
	Severity    string    `gorm:"default:'info'" json:"severity"`
}

// ─── Suspicious Activity ──────────────────────────────────────────────────────

type SuspiciousActivity struct {
	Base
	UserID      *uuid.UUID `gorm:"type:uuid;index" json:"user_id"`
	IPAddress   string     `gorm:"not null;index" json:"ip_address"`
	Action      string     `gorm:"not null" json:"action"`
	Count       int        `gorm:"default:1" json:"count"`
	WindowStart time.Time  `json:"window_start"`
	IsFlagged   bool       `gorm:"default:false;index" json:"is_flagged"`
	ResolvedAt  *time.Time `json:"resolved_at"`
	Notes       string     `json:"notes"`
}
