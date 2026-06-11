package handlers

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/kanban-platform/backend/internal/config"
	"github.com/kanban-platform/backend/internal/models"
	"github.com/kanban-platform/backend/internal/websocket"
)

type ReminderHandler struct {
	db     *gorm.DB
	hub    *websocket.Hub
	cfg    *config.Config
	logger *zap.Logger
}

func NewReminderHandler(db *gorm.DB, hub *websocket.Hub, cfg *config.Config, logger *zap.Logger) *ReminderHandler {
	return &ReminderHandler{db: db, hub: hub, cfg: cfg, logger: logger}
}

// List returns all reminders for the current user
func (h *ReminderHandler) List(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)

	var reminders []models.Reminder
	h.db.Preload("Task").
		Where("user_id = ?", userID).
		Order("next_run_at ASC").
		Find(&reminders)

	return c.JSON(fiber.Map{"reminders": reminders})
}

// Create creates a new 3-stage reminder
func (h *ReminderHandler) Create(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	userUID, _ := uuid.Parse(userID)

	type CreateReminderReq struct {
		TaskID         string                  `json:"task_id"`
		Title          string                  `json:"title"`
		Frequency      models.ReminderFrequency `json:"frequency"`
		CronExpression string                  `json:"cron_expression"`
		Level1At       *time.Time              `json:"level1_at"`
		Level2At       *time.Time              `json:"level2_at"`
		Level3At       *time.Time              `json:"level3_at"`
		EmailEnabled   bool                    `json:"email_enabled"`
		InAppEnabled   bool                    `json:"in_app_enabled"`
		EscalateAfter  int                     `json:"escalate_after"`
	}

	var req CreateReminderReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	taskUID, _ := uuid.Parse(req.TaskID)

	if req.EscalateAfter == 0 {
		req.EscalateAfter = 3
	}

	// Default: level1 = now, level2 = +1h, level3 = +3h if not provided
	now := time.Now()
	if req.Level1At == nil {
		t := now.Add(15 * time.Minute)
		req.Level1At = &t
	}
	if req.Level2At == nil {
		t := req.Level1At.Add(1 * time.Hour)
		req.Level2At = &t
	}
	if req.Level3At == nil {
		t := req.Level2At.Add(2 * time.Hour)
		req.Level3At = &t
	}

	reminder := models.Reminder{
		Base:          models.Base{ID: uuid.New()},
		TaskID:        taskUID,
		UserID:        userUID,
		Title:         req.Title,
		Frequency:     req.Frequency,
		CronExpression: req.CronExpression,
		Status:        models.ReminderActive,
		CurrentLevel:  models.Level1,
		Level1At:      req.Level1At,
		Level2At:      req.Level2At,
		Level3At:      req.Level3At,
		NextRunAt:     req.Level1At,
		EmailEnabled:  req.EmailEnabled,
		InAppEnabled:  req.InAppEnabled,
		EscalateAfter: req.EscalateAfter,
	}

	if req.Frequency == "" {
		reminder.Frequency = models.FrequencyOnce
	}

	if err := h.db.Create(&reminder).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create reminder"})
	}

	return c.Status(fiber.StatusCreated).JSON(reminder)
}

// Update updates a reminder (reschedule, pause, etc.)
func (h *ReminderHandler) Update(c *fiber.Ctx) error {
	reminderID := c.Params("reminderId")
	userID := c.Locals("user_id").(string)

	type UpdateReq struct {
		Status       *models.ReminderStatus `json:"status"`
		Level1At     *time.Time             `json:"level1_at"`
		Level2At     *time.Time             `json:"level2_at"`
		Level3At     *time.Time             `json:"level3_at"`
		EmailEnabled *bool                  `json:"email_enabled"`
		InAppEnabled *bool                  `json:"in_app_enabled"`
	}

	var req UpdateReq
	c.BodyParser(&req)

	updates := map[string]interface{}{}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.Level1At != nil {
		updates["level1_at"] = req.Level1At
		updates["next_run_at"] = req.Level1At
		updates["current_level"] = models.Level1
	}
	if req.Level2At != nil {
		updates["level2_at"] = req.Level2At
	}
	if req.Level3At != nil {
		updates["level3_at"] = req.Level3At
	}
	if req.EmailEnabled != nil {
		updates["email_enabled"] = *req.EmailEnabled
	}
	if req.InAppEnabled != nil {
		updates["in_app_enabled"] = *req.InAppEnabled
	}

	h.db.Model(&models.Reminder{}).
		Where("id = ? AND user_id = ?", reminderID, userID).
		Updates(updates)

	var reminder models.Reminder
	h.db.First(&reminder, "id = ?", reminderID)
	return c.JSON(reminder)
}

// Pause pauses an active reminder
func (h *ReminderHandler) Pause(c *fiber.Ctx) error {
	reminderID := c.Params("reminderId")
	userID := c.Locals("user_id").(string)

	h.db.Model(&models.Reminder{}).
		Where("id = ? AND user_id = ?", reminderID, userID).
		Update("status", models.ReminderPaused)

	return c.JSON(fiber.Map{"message": "Reminder paused"})
}

// Resume resumes a paused reminder
func (h *ReminderHandler) Resume(c *fiber.Ctx) error {
	reminderID := c.Params("reminderId")
	userID := c.Locals("user_id").(string)

	h.db.Model(&models.Reminder{}).
		Where("id = ? AND user_id = ?", reminderID, userID).
		Update("status", models.ReminderActive)

	return c.JSON(fiber.Map{"message": "Reminder resumed"})
}

// Delete deletes a reminder
func (h *ReminderHandler) Delete(c *fiber.Ctx) error {
	reminderID := c.Params("reminderId")
	userID := c.Locals("user_id").(string)

	h.db.Where("id = ? AND user_id = ?", reminderID, userID).
		Delete(&models.Reminder{})

	return c.JSON(fiber.Map{"message": "Reminder deleted"})
}

// GetHistory returns reminder fire history
func (h *ReminderHandler) GetHistory(c *fiber.Ctx) error {
	reminderID := c.Params("reminderId")

	var logs []models.ReminderLog
	h.db.Where("reminder_id = ?", reminderID).
		Order("sent_at DESC").
		Limit(50).
		Find(&logs)

	return c.JSON(fiber.Map{"logs": logs})
}

// ─── Analytics Handler ────────────────────────────────────────────────────────

type AnalyticsHandler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewAnalyticsHandler(db *gorm.DB, logger *zap.Logger) *AnalyticsHandler {
	return &AnalyticsHandler{db: db, logger: logger}
}

// BoardAnalytics returns analytics for a specific board
func (h *AnalyticsHandler) BoardAnalytics(c *fiber.Ctx) error {
	boardID := c.Params("boardId")
	since := c.Query("since", "30d")

	var sinceTime time.Time
	switch since {
	case "7d":
		sinceTime = time.Now().AddDate(0, 0, -7)
	case "90d":
		sinceTime = time.Now().AddDate(0, 0, -90)
	default:
		sinceTime = time.Now().AddDate(0, 0, -30)
	}

	// Task completion rate
	var totalTasks, completedTasks, overdueTasks int64
	h.db.Model(&models.Task{}).Where("board_id = ? AND created_at >= ?", boardID, sinceTime).Count(&totalTasks)
	h.db.Model(&models.Task{}).Where("board_id = ? AND completed_at >= ?", boardID, sinceTime).Count(&completedTasks)
	h.db.Model(&models.Task{}).Where("board_id = ? AND due_date < ? AND completed_at IS NULL", boardID, time.Now()).Count(&overdueTasks)

	// Time tracking
	type TimeResult struct {
		TotalEstimated float64
		TotalActual    float64
	}
	var timeResult TimeResult
	h.db.Model(&models.Task{}).
		Select("SUM(estimated_hours) as total_estimated, SUM(actual_hours) as total_actual").
		Where("board_id = ?", boardID).
		Scan(&timeResult)

	// Column distribution
	type ColDist struct {
		ColumnName string
		Count      int64
	}
	var colDist []ColDist
	h.db.Model(&models.Task{}).
		Select("columns.name as column_name, COUNT(tasks.id) as count").
		Joins("JOIN columns ON columns.id = tasks.column_id").
		Where("tasks.board_id = ? AND tasks.deleted_at IS NULL", boardID).
		Group("columns.name").
		Scan(&colDist)

	// Priority distribution
	type PriorityDist struct {
		Priority string
		Count    int64
	}
	var priorityDist []PriorityDist
	h.db.Model(&models.Task{}).
		Select("priority, COUNT(*) as count").
		Where("board_id = ? AND deleted_at IS NULL", boardID).
		Group("priority").
		Scan(&priorityDist)

	// Velocity: tasks completed per day
	type DailyCompletion struct {
		Date  time.Time
		Count int64
	}
	var dailyCompletion []DailyCompletion
	h.db.Model(&models.Task{}).
		Select("DATE_TRUNC('day', completed_at) as date, COUNT(*) as count").
		Where("board_id = ? AND completed_at >= ?", boardID, sinceTime).
		Group("date").
		Order("date ASC").
		Scan(&dailyCompletion)

	// Assignee workload
	type AssigneeWorkload struct {
		UserName   string
		TaskCount  int64
		TotalHours float64
	}
	var workload []AssigneeWorkload
	h.db.Raw(`
		SELECT u.name as user_name, COUNT(t.id) as task_count, SUM(t.estimated_hours) as total_hours
		FROM tasks t
		JOIN task_assignees ta ON ta.task_id = t.id
		JOIN users u ON u.id = ta.user_id
		WHERE t.board_id = ? AND t.deleted_at IS NULL
		GROUP BY u.name
		ORDER BY task_count DESC
	`, boardID).Scan(&workload)

	completionRate := 0.0
	if totalTasks > 0 {
		completionRate = float64(completedTasks) / float64(totalTasks) * 100
	}

	timeVariance := 0.0
	if timeResult.TotalEstimated > 0 {
		timeVariance = (timeResult.TotalActual - timeResult.TotalEstimated) / timeResult.TotalEstimated * 100
	}

	return c.JSON(fiber.Map{
		"summary": fiber.Map{
			"total_tasks":      totalTasks,
			"completed_tasks":  completedTasks,
			"overdue_tasks":    overdueTasks,
			"completion_rate":  fmt.Sprintf("%.1f%%", completionRate),
			"total_estimated":  timeResult.TotalEstimated,
			"total_actual":     timeResult.TotalActual,
			"time_variance":    fmt.Sprintf("%.1f%%", timeVariance),
		},
		"column_distribution":  colDist,
		"priority_distribution": priorityDist,
		"daily_completion":     dailyCompletion,
		"assignee_workload":    workload,
		"since":                sinceTime,
	})
}

// WorkspaceAnalytics returns workspace-wide analytics
func (h *AnalyticsHandler) WorkspaceAnalytics(c *fiber.Ctx) error {
	workspaceID := c.Params("workspaceId")

	var totalTasks, completedTasks, overdueTasks int64
	h.db.Model(&models.Task{}).Where("workspace_id = ?", workspaceID).Count(&totalTasks)
	h.db.Model(&models.Task{}).Where("workspace_id = ? AND completed_at IS NOT NULL", workspaceID).Count(&completedTasks)
	h.db.Model(&models.Task{}).Where("workspace_id = ? AND due_date < ? AND completed_at IS NULL", workspaceID, time.Now()).Count(&overdueTasks)

	var boardCount, memberCount int64
	h.db.Model(&models.Board{}).Where("workspace_id = ?", workspaceID).Count(&boardCount)
	h.db.Model(&models.WorkspaceMember{}).Where("workspace_id = ?", workspaceID).Count(&memberCount)

	// Recent activity
	var recentActivity []models.ActivityLog
	h.db.Preload("User").
		Where("workspace_id = ?", workspaceID).
		Order("created_at DESC").
		Limit(20).
		Find(&recentActivity)

	return c.JSON(fiber.Map{
		"total_tasks":      totalTasks,
		"completed_tasks":  completedTasks,
		"overdue_tasks":    overdueTasks,
		"board_count":      boardCount,
		"member_count":     memberCount,
		"completion_rate":  safeDiv(float64(completedTasks), float64(totalTasks)) * 100,
		"recent_activity":  recentActivity,
	})
}

// UserProductivity returns productivity stats per user
func (h *AnalyticsHandler) UserProductivity(c *fiber.Ctx) error {
	workspaceID := c.Params("workspaceId")

	type UserStats struct {
		UserID         string
		UserName       string
		UserAvatar     string
		TasksCompleted int64
		TasksInProgress int64
		TotalHours     float64
		OverdueTasks   int64
	}

	var stats []UserStats
	h.db.Raw(`
		SELECT 
			u.id as user_id,
			u.name as user_name,
			u.avatar as user_avatar,
			COUNT(CASE WHEN t.completed_at IS NOT NULL THEN 1 END) as tasks_completed,
			COUNT(CASE WHEN t.completed_at IS NULL THEN 1 END) as tasks_in_progress,
			SUM(t.actual_hours) as total_hours,
			COUNT(CASE WHEN t.due_date < NOW() AND t.completed_at IS NULL THEN 1 END) as overdue_tasks
		FROM users u
		JOIN task_assignees ta ON ta.user_id = u.id
		JOIN tasks t ON t.id = ta.task_id
		WHERE t.workspace_id = ? AND t.deleted_at IS NULL
		GROUP BY u.id, u.name, u.avatar
		ORDER BY tasks_completed DESC
	`, workspaceID).Scan(&stats)

	return c.JSON(fiber.Map{"users": stats})
}
