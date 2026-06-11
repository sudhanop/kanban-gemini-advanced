package handlers

import (
	"encoding/json"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/kanban-platform/backend/internal/models"
	"github.com/kanban-platform/backend/internal/websocket"
)

type TaskHandler struct {
	db     *gorm.DB
	hub    *websocket.Hub
	logger *zap.Logger
}

func NewTaskHandler(db *gorm.DB, hub *websocket.Hub, logger *zap.Logger) *TaskHandler {
	return &TaskHandler{db: db, hub: hub, logger: logger}
}

// List returns tasks for a board with filtering and pagination
func (h *TaskHandler) List(c *fiber.Ctx) error {
	boardID := c.Params("boardId")

	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 50)
	offset := (page - 1) * limit
	search := c.Query("search")
	priority := c.Query("priority")
	assigneeID := c.Query("assignee_id")
	columnID := c.Query("column_id")

	query := h.db.Where("board_id = ?", boardID)

	if search != "" {
		query = query.Where("title ILIKE ? OR summary ILIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if priority != "" {
		query = query.Where("priority = ?", priority)
	}
	if assigneeID != "" {
		query = query.Joins("JOIN task_assignees ON task_assignees.task_id = tasks.id").
			Where("task_assignees.user_id = ?", assigneeID)
	}
	if columnID != "" {
		query = query.Where("column_id = ?", columnID)
	}

	var total int64
	query.Model(&models.Task{}).Count(&total)

	var tasks []models.Task
	query.Order("order_index ASC").
		Limit(limit).Offset(offset).
		Preload("Assignees.User").
		Preload("Labels").
		Preload("Subtasks").
		Find(&tasks)

	return c.JSON(fiber.Map{
		"tasks": tasks,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// Create creates a new task
func (h *TaskHandler) Create(c *fiber.Ctx) error {
	boardID := c.Params("boardId")
	boardUID, _ := uuid.Parse(boardID)
	userID := c.Locals("user_id").(string)
	userUID, _ := uuid.Parse(userID)

	type CreateRequest struct {
		Title       string           `json:"title"`
		Summary     string           `json:"summary"`
		Description string           `json:"description"`
		ColumnID    string           `json:"column_id"`
		WorkspaceID string           `json:"workspace_id"`
		Priority    models.Priority  `json:"priority"`
		Category    string           `json:"category"`
		Color       string           `json:"color"`
		DueDate     *time.Time       `json:"due_date"`
		StartDate   *time.Time       `json:"start_date"`
		AssigneeIDs []string         `json:"assignee_ids"`
		Labels      []struct {
			Label string `json:"label"`
			Color string `json:"color"`
		} `json:"labels"`
		EstimatedHours float64 `json:"estimated_hours"`
		StoryPoints    int     `json:"story_points"`
		ParentID       *string `json:"parent_id"`
	}

	var req CreateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}
	if req.Title == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Task title required"})
	}

	columnUID, _ := uuid.Parse(req.ColumnID)
	workspaceUID, _ := uuid.Parse(req.WorkspaceID)

	task := models.Task{
		Base:           models.Base{ID: uuid.New()},
		BoardID:        boardUID,
		WorkspaceID:    workspaceUID,
		ColumnID:       columnUID,
		Title:          req.Title,
		Summary:        req.Summary,
		Description:    req.Description,
		Priority:       req.Priority,
		Category:       req.Category,
		Color:          req.Color,
		DueDate:        req.DueDate,
		StartDate:      req.StartDate,
		EstimatedHours: req.EstimatedHours,
		StoryPoints:    req.StoryPoints,
		CreatedBy:      userUID,
	}

	if req.Priority == "" {
		task.Priority = models.PriorityMedium
	}

	if req.ParentID != nil {
		parentUID, _ := uuid.Parse(*req.ParentID)
		task.ParentID = &parentUID
	}

	// Get max order index
	var maxOrder int
	h.db.Model(&models.Task{}).Where("column_id = ?", req.ColumnID).
		Select("COALESCE(MAX(order_index), -1)").Scan(&maxOrder)
	task.OrderIndex = maxOrder + 1

	if err := h.db.Create(&task).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create task"})
	}

	// Add assignees
	for _, aID := range req.AssigneeIDs {
		aUID, err := uuid.Parse(aID)
		if err != nil {
			continue
		}
		h.db.Create(&models.TaskAssignee{
			Base:   models.Base{ID: uuid.New()},
			TaskID: task.ID,
			UserID: aUID,
		})
	}

	// Add labels
	for _, l := range req.Labels {
		h.db.Create(&models.TaskLabel{
			Base:   models.Base{ID: uuid.New()},
			TaskID: task.ID,
			Label:  l.Label,
			Color:  l.Color,
		})
	}

	// Log activity
	h.logActivity(userUID, task.WorkspaceID, &task.BoardID, &task.ID, "task_created", "task", task.ID.String())

	// Broadcast
	h.db.Preload("Assignees.User").Preload("Labels").First(&task, task.ID)
	h.hub.BroadcastToRoom("board:"+boardID, &websocket.Message{
		Type:      websocket.MsgTaskCreated,
		RoomID:    "board:" + boardID,
		UserID:    userID,
		Payload:   task,
		Timestamp: time.Now(),
	}, "")

	return c.Status(fiber.StatusCreated).JSON(task)
}

// Get returns full task detail
func (h *TaskHandler) Get(c *fiber.Ctx) error {
	taskID := c.Params("taskId")

	var task models.Task
	if err := h.db.
		Preload("Assignees.User").
		Preload("Watchers.User").
		Preload("Labels").
		Preload("Subtasks").
		Preload("Comments", func(db *gorm.DB) *gorm.DB {
			return db.Where("parent_id IS NULL").Order("created_at ASC").
				Preload("User").
				Preload("Reactions.User").
				Preload("Replies.User").
				Preload("Replies.Reactions.User")
		}).
		Preload("Attachments.User").
		Preload("Dependencies.DependsOn").
		First(&task, "id = ?", taskID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Task not found"})
	}

	return c.JSON(task)
}

// Update updates task fields
func (h *TaskHandler) Update(c *fiber.Ctx) error {
	taskID := c.Params("taskId")
	userID := c.Locals("user_id").(string)
	userUID, _ := uuid.Parse(userID)

	var updates map[string]interface{}
	if err := c.BodyParser(&updates); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	// Remove protected fields
	delete(updates, "id")
	delete(updates, "created_at")
	delete(updates, "board_id")
	delete(updates, "workspace_id")

	var task models.Task
	if err := h.db.First(&task, "id = ?", taskID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Task not found"})
	}

	h.db.Model(&task).Updates(updates)
	h.logActivity(userUID, task.WorkspaceID, &task.BoardID, &task.ID, "task_updated", "task", taskID)

	h.db.Preload("Assignees.User").Preload("Labels").First(&task, task.ID)

	h.hub.BroadcastToRoom("board:"+task.BoardID.String(), &websocket.Message{
		Type:      websocket.MsgTaskUpdated,
		RoomID:    "board:" + task.BoardID.String(),
		UserID:    userID,
		Payload:   task,
		Timestamp: time.Now(),
	}, "")

	return c.JSON(task)
}

// MoveTask moves a task between columns (drag & drop)
func (h *TaskHandler) MoveTask(c *fiber.Ctx) error {
	taskID := c.Params("taskId")
	boardID := c.Params("boardId")
	userID := c.Locals("user_id").(string)
	userUID, _ := uuid.Parse(userID)

	type MoveRequest struct {
		ColumnID   string `json:"column_id"`
		OrderIndex int    `json:"order_index"`
	}
	var req MoveRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	var task models.Task
	if err := h.db.First(&task, "id = ?", taskID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Task not found"})
	}

	oldColumnID := task.ColumnID
	newColumnUID, _ := uuid.Parse(req.ColumnID)

	// Shift tasks in target column
	h.db.Model(&models.Task{}).
		Where("column_id = ? AND order_index >= ? AND id != ?", req.ColumnID, req.OrderIndex, taskID).
		UpdateColumn("order_index", gorm.Expr("order_index + 1"))

	h.db.Model(&task).Updates(map[string]interface{}{
		"column_id":   newColumnUID,
		"order_index": req.OrderIndex,
	})

	// Log the column change
	h.logActivity(userUID, task.WorkspaceID, &task.BoardID, &task.ID, "task_moved", "task", taskID)

	// Run automation rules for column change
	go h.runAutomations(task.BoardID.String(), "task_moved", map[string]string{
		"task_id":        taskID,
		"from_column_id": oldColumnID.String(),
		"to_column_id":   req.ColumnID,
	})

	h.hub.BroadcastToRoom("board:"+boardID, &websocket.Message{
		Type:   websocket.MsgTaskMoved,
		RoomID: "board:" + boardID,
		UserID: userID,
		Payload: map[string]interface{}{
			"task_id":       taskID,
			"column_id":     req.ColumnID,
			"order_index":   req.OrderIndex,
			"old_column_id": oldColumnID.String(),
		},
		Timestamp: time.Now(),
	}, "")

	return c.JSON(fiber.Map{"message": "Task moved", "task_id": taskID, "column_id": req.ColumnID})
}

// Delete soft-deletes a task
func (h *TaskHandler) Delete(c *fiber.Ctx) error {
	taskID := c.Params("taskId")
	boardID := c.Params("boardId")
	userID := c.Locals("user_id").(string)
	userUID, _ := uuid.Parse(userID)

	var task models.Task
	if err := h.db.First(&task, "id = ?", taskID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Task not found"})
	}

	h.db.Delete(&task)
	h.logActivity(userUID, task.WorkspaceID, &task.BoardID, &task.ID, "task_deleted", "task", taskID)

	h.hub.BroadcastToRoom("board:"+boardID, &websocket.Message{
		Type:      websocket.MsgTaskDeleted,
		RoomID:    "board:" + boardID,
		UserID:    userID,
		Payload:   fiber.Map{"task_id": taskID},
		Timestamp: time.Now(),
	}, "")

	return c.JSON(fiber.Map{"message": "Task deleted"})
}

// AddAssignee adds a user to task assignees
func (h *TaskHandler) AddAssignee(c *fiber.Ctx) error {
	taskID := c.Params("taskId")
	taskUID, _ := uuid.Parse(taskID)

	type Req struct {
		UserID string `json:"user_id"`
	}
	var req Req
	c.BodyParser(&req)

	userUID, _ := uuid.Parse(req.UserID)
	assignee := models.TaskAssignee{
		Base:   models.Base{ID: uuid.New()},
		TaskID: taskUID,
		UserID: userUID,
	}
	h.db.Where(models.TaskAssignee{TaskID: taskUID, UserID: userUID}).FirstOrCreate(&assignee)

	return c.JSON(fiber.Map{"message": "Assignee added"})
}

// RemoveAssignee removes an assignee
func (h *TaskHandler) RemoveAssignee(c *fiber.Ctx) error {
	taskID := c.Params("taskId")
	assigneeUserID := c.Params("userId")

	h.db.Where("task_id = ? AND user_id = ?", taskID, assigneeUserID).
		Delete(&models.TaskAssignee{})

	return c.JSON(fiber.Map{"message": "Assignee removed"})
}

// AddComment adds a comment to a task
func (h *TaskHandler) AddComment(c *fiber.Ctx) error {
	taskID := c.Params("taskId")
	taskUID, _ := uuid.Parse(taskID)
	boardID := c.Params("boardId")
	userID := c.Locals("user_id").(string)
	userUID, _ := uuid.Parse(userID)

	type CommentReq struct {
		Content  string  `json:"content"`
		ParentID *string `json:"parent_id"`
	}
	var req CommentReq
	if err := c.BodyParser(&req); err != nil || req.Content == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Content required"})
	}

	comment := models.Comment{
		Base:    models.Base{ID: uuid.New()},
		TaskID:  taskUID,
		UserID:  userUID,
		Content: req.Content,
	}

	if req.ParentID != nil {
		parentUID, _ := uuid.Parse(*req.ParentID)
		comment.ParentID = &parentUID
	}

	h.db.Create(&comment)
	h.db.Preload("User").Preload("Reactions.User").First(&comment, comment.ID)

	h.hub.BroadcastToRoom("board:"+boardID, &websocket.Message{
		Type:      websocket.MsgCommentAdded,
		RoomID:    "board:" + boardID,
		UserID:    userID,
		Payload:   comment,
		Timestamp: time.Now(),
	}, "")

	return c.Status(fiber.StatusCreated).JSON(comment)
}

// AddReaction adds emoji reaction to comment
func (h *TaskHandler) AddReaction(c *fiber.Ctx) error {
	commentID := c.Params("commentId")
	commentUID, _ := uuid.Parse(commentID)
	userID := c.Locals("user_id").(string)
	userUID, _ := uuid.Parse(userID)

	type ReactionReq struct {
		Emoji string `json:"emoji"`
	}
	var req ReactionReq
	c.BodyParser(&req)

	reaction := models.CommentReaction{
		Base:      models.Base{ID: uuid.New()},
		CommentID: commentUID,
		UserID:    userUID,
		Emoji:     req.Emoji,
	}

	// Toggle: if exists remove, otherwise add
	var existing models.CommentReaction
	result := h.db.Where("comment_id = ? AND user_id = ? AND emoji = ?",
		commentID, userID, req.Emoji).First(&existing)
	if result.Error == nil {
		h.db.Delete(&existing)
		return c.JSON(fiber.Map{"action": "removed"})
	}

	h.db.Create(&reaction)
	return c.JSON(fiber.Map{"action": "added"})
}

// GetSubtasks returns subtasks for a task
func (h *TaskHandler) GetSubtasks(c *fiber.Ctx) error {
	taskID := c.Params("taskId")
	var subtasks []models.Subtask
	h.db.Preload("Assignee").Where("parent_id = ?", taskID).
		Order("order_index ASC").Find(&subtasks)
	return c.JSON(fiber.Map{"subtasks": subtasks})
}

// CreateSubtask creates a subtask
func (h *TaskHandler) CreateSubtask(c *fiber.Ctx) error {
	taskID := c.Params("taskId")
	taskUID, _ := uuid.Parse(taskID)

	type SubReq struct {
		Title      string     `json:"title"`
		AssigneeID *string    `json:"assignee_id"`
		DueDate    *time.Time `json:"due_date"`
	}
	var req SubReq
	c.BodyParser(&req)

	subtask := models.Subtask{
		Base:     models.Base{ID: uuid.New()},
		ParentID: taskUID,
		Title:    req.Title,
		DueDate:  req.DueDate,
	}

	if req.AssigneeID != nil {
		aUID, _ := uuid.Parse(*req.AssigneeID)
		subtask.AssigneeID = &aUID
	}

	h.db.Create(&subtask)
	return c.Status(fiber.StatusCreated).JSON(subtask)
}

// UpdateSubtask updates subtask
func (h *TaskHandler) UpdateSubtask(c *fiber.Ctx) error {
	subtaskID := c.Params("subtaskId")
	var updates map[string]interface{}
	c.BodyParser(&updates)
	h.db.Model(&models.Subtask{}).Where("id = ?", subtaskID).Updates(updates)

	var sub models.Subtask
	h.db.First(&sub, "id = ?", subtaskID)
	return c.JSON(sub)
}

// AddWatcher adds a watcher to a task
func (h *TaskHandler) AddWatcher(c *fiber.Ctx) error {
	taskID := c.Params("taskId")
	taskUID, _ := uuid.Parse(taskID)
	userID := c.Locals("user_id").(string)
	userUID, _ := uuid.Parse(userID)

	watcher := models.TaskWatcher{
		Base:   models.Base{ID: uuid.New()},
		TaskID: taskUID,
		UserID: userUID,
	}
	h.db.Where(models.TaskWatcher{TaskID: taskUID, UserID: userUID}).FirstOrCreate(&watcher)
	return c.JSON(fiber.Map{"message": "Watching task"})
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func (h *TaskHandler) logActivity(userUID, workspaceUID uuid.UUID, boardUID *uuid.UUID, taskUID *uuid.UUID, action, entityType, entityID string) {
	log := models.ActivityLog{
		Base:        models.Base{ID: uuid.New()},
		UserID:      userUID,
		WorkspaceID: workspaceUID,
		BoardID:     boardUID,
		TaskID:      taskUID,
		Action:      action,
		EntityType:  entityType,
	}
	go h.db.Create(&log)
}

func (h *TaskHandler) runAutomations(boardID, trigger string, context map[string]string) {
	var rules []models.AutomationRule
	h.db.Where("board_id = ? AND trigger_type = ? AND is_enabled = true", boardID, trigger).Find(&rules)

	for _, rule := range rules {
		var triggerData map[string]string
		json.Unmarshal([]byte(rule.TriggerData), &triggerData)

		// Simple rule matching
		matched := true
		for k, v := range triggerData {
			if context[k] != v {
				matched = false
				break
			}
		}

		if matched {
			h.db.Model(&rule).UpdateColumn("run_count", gorm.Expr("run_count + 1"))
			// Execute action (simplified)
			h.executeAutomationAction(rule.ActionType, rule.ActionData, context)
		}
	}
}

func (h *TaskHandler) executeAutomationAction(actionType, actionData string, ctx map[string]string) {
	var data map[string]interface{}
	json.Unmarshal([]byte(actionData), &data)

	switch actionType {
	case "notify_reviewer":
		// Send notification to reviewer
	case "update_priority":
		if taskID, ok := ctx["task_id"]; ok {
			if priority, ok := data["priority"].(string); ok {
				h.db.Model(&models.Task{}).Where("id = ?", taskID).Update("priority", priority)
			}
		}
	}
}
