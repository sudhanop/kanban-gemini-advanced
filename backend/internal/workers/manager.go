package workers

import (
	"context"
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"gopkg.in/gomail.v2"
	"gorm.io/gorm"

	"github.com/kanban-platform/backend/internal/config"
	"github.com/kanban-platform/backend/internal/models"
	"github.com/kanban-platform/backend/internal/websocket"
)

// Manager manages all background workers
type Manager struct {
	db     *gorm.DB
	hub    *websocket.Hub
	cfg    *config.Config
	logger *zap.Logger
	cron   *cron.Cron
	ctx    context.Context
	cancel context.CancelFunc
}

// NewManager creates a new worker manager
func NewManager(db *gorm.DB, hub *websocket.Hub, cfg *config.Config, logger *zap.Logger) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		db:     db,
		hub:    hub,
		cfg:    cfg,
		logger: logger,
		cron:   cron.New(cron.WithSeconds()),
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start starts all background workers
func (m *Manager) Start() {
	// Reminder escalation engine - runs every minute
	m.cron.AddFunc("0 * * * * *", m.processReminders)

	// Overdue task detection - runs every hour
	m.cron.AddFunc("0 0 * * * *", m.detectOverdueTasks)

	// Suspicious activity cleanup - runs every day
	m.cron.AddFunc("0 0 2 * * *", m.cleanupOldData)

	// Sprint velocity calculation - runs every night
	m.cron.AddFunc("0 0 1 * * *", m.calculateSprintVelocity)

	m.cron.Start()
	m.logger.Info("Background workers started",
		zap.Strings("jobs", []string{
			"reminder_engine (every minute)",
			"overdue_detector (every hour)",
			"cleanup (daily at 2am)",
			"sprint_velocity (daily at 1am)",
		}),
	)
}

// Stop gracefully stops all workers
func (m *Manager) Stop() {
	m.cron.Stop()
	m.cancel()
	m.logger.Info("Background workers stopped")
}

// ─── Reminder Engine (3-Stage Escalation) ────────────────────────────────────

func (m *Manager) processReminders() {
	now := time.Now()

	var reminders []models.Reminder
	m.db.Preload("Task").Preload("User").
		Where("status = ? AND next_run_at <= ?", models.ReminderActive, now).
		Find(&reminders)

	for _, reminder := range reminders {
		m.fireReminder(&reminder, now)
	}
}

func (m *Manager) fireReminder(r *models.Reminder, now time.Time) {
	defer func() {
		if rec := recover(); rec != nil {
			m.logger.Error("Panic in fireReminder", zap.Any("recover", rec))
		}
	}()

	// Determine current escalation level
	level := r.CurrentLevel

	m.logger.Info("Firing reminder",
		zap.String("reminder_id", r.ID.String()),
		zap.String("task", r.Task.Title),
		zap.Int("level", int(level)),
	)

	// Send notification
	title := r.Title
	if title == "" {
		title = r.Task.Title
	}

	var levelLabel string
	switch level {
	case models.Level1:
		levelLabel = "⏰ Reminder"
	case models.Level2:
		levelLabel = "⚠️ Follow-up Reminder"
	case models.Level3:
		levelLabel = "🚨 CRITICAL Reminder"
	}

	message := fmt.Sprintf("%s: %s", levelLabel, title)

	// In-app notification
	if r.InAppEnabled {
		notif := models.Notification{
			Base:       models.Base{},
			UserID:     r.UserID,
			Type:       models.NotifReminderFired,
			Title:      levelLabel,
			Message:    message,
			EntityID:   &r.TaskID,
			EntityType: "task",
			ActionURL:  fmt.Sprintf("/tasks/%s", r.TaskID.String()),
		}
		notif.ID = notif.ID // uuid is auto-set
		m.db.Create(&notif)

		// Push via WebSocket if user is online
		m.publishWSNotification(r.UserID.String(), notif)
	}

	// Email notification
	if r.EmailEnabled && m.cfg.SMTPUser != "" {
		go m.sendReminderEmail(r.User.Email, r.User.Name, message, r.Task.Title, level)
	}

	// Log this fire
	log := models.ReminderLog{
		ReminderID: r.ID,
		Level:      level,
		Channel:    "both",
		SentAt:     now,
		Status:     "sent",
	}
	m.db.Create(&log)

	// Increment ignore count and escalate if needed
	newIgnoreCount := r.IgnoreCount + 1
	nextLevel := level
	var nextRunAt *time.Time

	if newIgnoreCount >= r.EscalateAfter {
		// Escalate to next level
		switch level {
		case models.Level1:
			if r.Level2At != nil {
				nextLevel = models.Level2
				nextRunAt = r.Level2At
			}
		case models.Level2:
			if r.Level3At != nil {
				nextLevel = models.Level3
				nextRunAt = r.Level3At
			}
		case models.Level3:
			// Max level - keep firing at Level3 interval or mark done
			if r.Frequency == models.FrequencyOnce {
				m.db.Model(r).Update("status", models.ReminderExpired)
				return
			}
		}
		newIgnoreCount = 0
	} else {
		// Stay at current level, next run based on current level schedule
		switch level {
		case models.Level1:
			nextRunAt = r.Level2At
		case models.Level2:
			nextRunAt = r.Level3At
		case models.Level3:
			// Recurring: re-schedule based on frequency
			if r.Frequency != models.FrequencyOnce {
				next := m.calculateNextRun(r.Frequency, now)
				nextRunAt = &next
			}
		}
	}

	updates := map[string]interface{}{
		"ignore_count":  newIgnoreCount,
		"current_level": nextLevel,
		"last_run_at":   now,
	}

	if nextRunAt != nil {
		updates["next_run_at"] = nextRunAt
	} else if r.Frequency == models.FrequencyOnce {
		updates["status"] = models.ReminderDone
	} else {
		next := m.calculateNextRun(r.Frequency, now)
		updates["next_run_at"] = next
		updates["current_level"] = models.Level1
	}

	m.db.Model(r).Updates(updates)
}

func (m *Manager) calculateNextRun(freq models.ReminderFrequency, from time.Time) time.Time {
	switch freq {
	case models.FrequencyDaily:
		return from.AddDate(0, 0, 1)
	case models.FrequencyWeekly:
		return from.AddDate(0, 0, 7)
	case models.FrequencyMonthly:
		return from.AddDate(0, 1, 0)
	default:
		return from.Add(24 * time.Hour)
	}
}

func (m *Manager) sendReminderEmail(toEmail, userName, message, taskTitle string, level models.ReminderLevel) {
	var urgencyColor string
	switch level {
	case models.Level1:
		urgencyColor = "#6366f1"
	case models.Level2:
		urgencyColor = "#f59e0b"
	case models.Level3:
		urgencyColor = "#ef4444"
	}

	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head><style>
  body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; background: #0f0f1a; color: #e2e8f0; margin: 0; padding: 20px; }
  .container { max-width: 600px; margin: 0 auto; background: #1a1a2e; border-radius: 12px; padding: 40px; border-top: 4px solid %s; }
  .logo { font-size: 24px; font-weight: 700; color: #818cf8; margin-bottom: 24px; }
  h1 { font-size: 24px; font-weight: 700; margin-bottom: 16px; color: %s; }
  p { color: #94a3b8; line-height: 1.6; margin-bottom: 16px; }
  .task { background: #252542; border-radius: 8px; padding: 16px; margin: 16px 0; border-left: 4px solid %s; }
</style></head>
<body>
  <div class="container">
    <div class="logo">⚡ FlowBoard</div>
    <h1>%s</h1>
    <p>Hi %s,</p>
    <p>%s</p>
    <div class="task"><strong>Task:</strong> %s</div>
    <p>Please review this task on FlowBoard.</p>
  </div>
</body>
</html>`, urgencyColor, urgencyColor, urgencyColor, message, userName, message, taskTitle)

	mail := gomail.NewMessage()
	mail.SetHeader("From", fmt.Sprintf("%s <%s>", m.cfg.SMTPFromName, m.cfg.SMTPFromEmail))
	mail.SetHeader("To", toEmail)
	mail.SetHeader("Subject", message)
	mail.SetBody("text/html", htmlBody)

	d := gomail.NewDialer(m.cfg.SMTPHost, m.cfg.SMTPPort, m.cfg.SMTPUser, m.cfg.SMTPPassword)
	if err := d.DialAndSend(mail); err != nil {
		m.logger.Error("Failed to send reminder email", zap.Error(err))
	}
}

func (m *Manager) publishWSNotification(userID string, notif models.Notification) {
	if m.hub != nil {
		m.hub.SendToUser(userID, &websocket.Message{
			Type:      websocket.MsgNotification,
			RoomID:    "",
			UserID:    userID,
			UserName:  "",
			Payload:   notif,
			Timestamp: time.Now(),
		})
	}
}

// ─── Overdue Task Detection ───────────────────────────────────────────────────

func (m *Manager) detectOverdueTasks() {
	now := time.Now()

	var tasks []models.Task
	m.db.Preload("Assignees.User").
		Where("due_date < ? AND completed_at IS NULL AND deleted_at IS NULL", now).
		Find(&tasks)

	for _, task := range tasks {
		// Update AI risk score
		daysPastDue := now.Sub(*task.DueDate).Hours() / 24
		riskScore := 0.5 + (daysPastDue * 0.1)
		if riskScore > 1.0 {
			riskScore = 1.0
		}
		m.db.Model(&task).Update("ai_risk_score", riskScore)

		// Notify assignees
		for _, assignee := range task.Assignees {
			var existingNotif int64
			m.db.Model(&models.Notification{}).
				Where("user_id = ? AND entity_id = ? AND type = ? AND created_at > ?",
					assignee.UserID, task.ID, models.NotifTaskDue, now.Add(-24*time.Hour)).
				Count(&existingNotif)

			if existingNotif == 0 {
				m.db.Create(&models.Notification{
					UserID:     assignee.UserID,
					Type:       models.NotifTaskDue,
					Title:      "Task Overdue",
					Message:    fmt.Sprintf("Task '%s' is overdue by %.0f days", task.Title, daysPastDue),
					EntityID:   &task.ID,
					EntityType: "task",
					ActionURL:  fmt.Sprintf("/tasks/%s", task.ID.String()),
				})
			}
		}
	}

	m.logger.Info("Overdue task check complete", zap.Int("overdue", len(tasks)))
}

// ─── Cleanup Worker ───────────────────────────────────────────────────────────

func (m *Manager) cleanupOldData() {
	cutoff := time.Now().AddDate(0, -3, 0) // 3 months ago

	// Clean expired invitations
	m.db.Where("expires_at < ? AND status = ?", time.Now(), models.InvitationPending).
		Updates(map[string]interface{}{"status": models.InvitationExpired})

	// Clean old notifications (keep 90 days)
	m.db.Where("created_at < ? AND is_read = true", cutoff).
		Delete(&models.Notification{})

	// Clean old activity logs (keep 90 days)
	m.db.Where("created_at < ?", cutoff).Delete(&models.ActivityLog{})

	m.logger.Info("Cleanup job complete")
}

// ─── Sprint Velocity Calculation ─────────────────────────────────────────────

func (m *Manager) calculateSprintVelocity() {
	var completedSprints []models.SprintPlan
	m.db.Preload("SprintTasks.Task").
		Where("status = ?", models.SprintCompleted).
		Find(&completedSprints)

	for _, sprint := range completedSprints {
		if sprint.Velocity > 0 {
			continue // Already calculated
		}

		totalPoints := 0
		completedPoints := 0
		for _, st := range sprint.SprintTasks {
			totalPoints += st.Task.StoryPoints
			if st.Task.CompletedAt != nil {
				completedPoints += st.Task.StoryPoints
			}
		}

		m.db.Model(&sprint).Update("velocity", completedPoints)
	}

	m.logger.Info("Sprint velocity calculation complete")
}
