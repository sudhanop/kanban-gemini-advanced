package handlers

import (
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/kanban-platform/backend/internal/config"
	"github.com/kanban-platform/backend/internal/models"
	"github.com/kanban-platform/backend/internal/websocket"
	"github.com/xuri/excelize/v2"
)

// ─── Search Handler ───────────────────────────────────────────────────────────

type SearchHandler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewSearchHandler(db *gorm.DB, logger *zap.Logger) *SearchHandler {
	return &SearchHandler{db: db, logger: logger}
}

// GlobalSearch searches across all entities
func (h *SearchHandler) GlobalSearch(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	query := c.Query("q")
	entityType := c.Query("type") // "task", "board", "workspace", "comment"
	limit := c.QueryInt("limit", 20)

	if query == "" || len(query) < 2 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Query must be at least 2 characters"})
	}

	results := fiber.Map{}

	// Get user's accessible workspaces
	var workspaceIDs []string
	h.db.Model(&models.WorkspaceMember{}).
		Select("workspace_id").
		Where("user_id = ?", userID).
		Pluck("workspace_id", &workspaceIDs)

	searchTerm := "%" + strings.ToLower(query) + "%"

	if entityType == "" || entityType == "task" {
		var tasks []struct {
			ID          uuid.UUID `json:"id"`
			Title       string    `json:"title"`
			Summary     string    `json:"summary"`
			Priority    string    `json:"priority"`
			BoardID     uuid.UUID `json:"board_id"`
			WorkspaceID uuid.UUID `json:"workspace_id"`
		}
		h.db.Model(&models.Task{}).
			Select("id, title, summary, priority, board_id, workspace_id").
			Where("workspace_id IN ? AND (LOWER(title) LIKE ? OR LOWER(summary) LIKE ? OR LOWER(description) LIKE ?)",
				workspaceIDs, searchTerm, searchTerm, searchTerm).
			Limit(limit).Find(&tasks)
		results["tasks"] = tasks
	}

	if entityType == "" || entityType == "board" {
		var boards []struct {
			ID          uuid.UUID `json:"id"`
			Name        string    `json:"name"`
			Description string    `json:"description"`
			WorkspaceID uuid.UUID `json:"workspace_id"`
		}
		h.db.Model(&models.Board{}).
			Select("id, name, description, workspace_id").
			Where("workspace_id IN ? AND (LOWER(name) LIKE ? OR LOWER(description) LIKE ?)",
				workspaceIDs, searchTerm, searchTerm).
			Limit(limit).Find(&boards)
		results["boards"] = boards
	}

	if entityType == "" || entityType == "workspace" {
		var workspaces []struct {
			ID          uuid.UUID `json:"id"`
			Name        string    `json:"name"`
			Description string    `json:"description"`
		}
		h.db.Model(&models.Workspace{}).
			Select("id, name, description").
			Where("id IN ? AND (LOWER(name) LIKE ? OR LOWER(description) LIKE ?)",
				workspaceIDs, searchTerm, searchTerm).
			Limit(limit).Find(&workspaces)
		results["workspaces"] = workspaces
	}

	if entityType == "" || entityType == "comment" {
		var comments []struct {
			ID      uuid.UUID `json:"id"`
			Content string    `json:"content"`
			TaskID  uuid.UUID `json:"task_id"`
		}
		h.db.Model(&models.Comment{}).
			Select("comments.id, comments.content, comments.task_id").
			Joins("JOIN tasks ON tasks.id = comments.task_id").
			Where("tasks.workspace_id IN ? AND LOWER(comments.content) LIKE ?",
				workspaceIDs, searchTerm).
			Limit(limit).Find(&comments)
		results["comments"] = comments
	}

	return c.JSON(fiber.Map{
		"query":   query,
		"results": results,
	})
}

// ─── Notification Handler ─────────────────────────────────────────────────────

type NotificationHandler struct {
	db     *gorm.DB
	hub    *websocket.Hub
	logger *zap.Logger
}

func NewNotificationHandler(db *gorm.DB, hub *websocket.Hub, logger *zap.Logger) *NotificationHandler {
	return &NotificationHandler{db: db, hub: hub, logger: logger}
}

// List returns notifications for current user
func (h *NotificationHandler) List(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	unreadOnly := c.QueryBool("unread_only", false)
	limit := c.QueryInt("limit", 20)

	query := h.db.Where("user_id = ?", userID)
	if unreadOnly {
		query = query.Where("is_read = false")
	}

	var notifications []models.Notification
	query.Order("created_at DESC").Limit(limit).Find(&notifications)

	var unreadCount int64
	h.db.Model(&models.Notification{}).
		Where("user_id = ? AND is_read = false", userID).Count(&unreadCount)

	return c.JSON(fiber.Map{
		"notifications": notifications,
		"unread_count":  unreadCount,
	})
}

// MarkRead marks a notification as read
func (h *NotificationHandler) MarkRead(c *fiber.Ctx) error {
	notifID := c.Params("notifId")
	userID := c.Locals("user_id").(string)

	now := time.Now()
	h.db.Model(&models.Notification{}).
		Where("id = ? AND user_id = ?", notifID, userID).
		Updates(map[string]interface{}{"is_read": true, "read_at": now})

	return c.JSON(fiber.Map{"message": "Marked as read"})
}

// MarkAllRead marks all notifications as read
func (h *NotificationHandler) MarkAllRead(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	now := time.Now()

	h.db.Model(&models.Notification{}).
		Where("user_id = ? AND is_read = false", userID).
		Updates(map[string]interface{}{"is_read": true, "read_at": now})

	return c.JSON(fiber.Map{"message": "All notifications marked as read"})
}

// ─── Meeting Handler ──────────────────────────────────────────────────────────

type MeetingHandler struct {
	db     *gorm.DB
	cfg    *config.Config
	hub    *websocket.Hub
	logger *zap.Logger
}

func NewMeetingHandler(db *gorm.DB, cfg *config.Config, hub *websocket.Hub, logger *zap.Logger) *MeetingHandler {
	return &MeetingHandler{db: db, cfg: cfg, hub: hub, logger: logger}
}

// Create creates a meeting (Jitsi or Google Meet)
func (h *MeetingHandler) Create(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	userUID, _ := uuid.Parse(userID)

	type MeetingReq struct {
		WorkspaceID string             `json:"workspace_id"`
		BoardID     *string            `json:"board_id"`
		TaskID      *string            `json:"task_id"`
		Title       string             `json:"title"`
		Description string             `json:"description"`
		MeetingType models.MeetingType `json:"meeting_type"`
		MeetingLink string             `json:"meeting_link"` // for Google Meet
		ScheduledAt *time.Time         `json:"scheduled_at"`
		Duration    int                `json:"duration"`
		ParticipantIDs []string        `json:"participant_ids"`
	}

	var req MeetingReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	workspaceUID, _ := uuid.Parse(req.WorkspaceID)

	meeting := models.Meeting{
		Base:        models.Base{ID: uuid.New()},
		WorkspaceID: workspaceUID,
		Title:       req.Title,
		Description: req.Description,
		MeetingType: req.MeetingType,
		ScheduledAt: req.ScheduledAt,
		Duration:    req.Duration,
		CreatedBy:   userUID,
	}

	if req.BoardID != nil {
		boardUID, _ := uuid.Parse(*req.BoardID)
		meeting.BoardID = &boardUID
	}
	if req.TaskID != nil {
		taskUID, _ := uuid.Parse(*req.TaskID)
		meeting.TaskID = &taskUID
	}

	// Generate Jitsi room or store Google Meet link
	switch req.MeetingType {
	case models.MeetingJitsi:
		roomName := fmt.Sprintf("flowboard-%s", uuid.New().String()[:12])
		meeting.RoomName = roomName
		meeting.MeetingLink = fmt.Sprintf("https://%s/%s", h.cfg.JitsiDomain, roomName)
	case models.MeetingGMeet:
		meeting.MeetingLink = req.MeetingLink
	default:
		meeting.MeetingLink = req.MeetingLink
	}

	if err := h.db.Create(&meeting).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create meeting"})
	}

	// Add participants
	for _, pID := range req.ParticipantIDs {
		pUID, err := uuid.Parse(pID)
		if err != nil {
			continue
		}
		h.db.Create(&models.MeetingParticipant{
			Base:      models.Base{ID: uuid.New()},
			MeetingID: meeting.ID,
			UserID:    pUID,
			Status:    "invited",
		})
	}

	h.db.Preload("Participants.User").First(&meeting, meeting.ID)
	return c.Status(fiber.StatusCreated).JSON(meeting)
}

// List returns meetings for a workspace/board/task
func (h *MeetingHandler) List(c *fiber.Ctx) error {
	workspaceID := c.Params("workspaceId")
	boardID := c.Query("board_id")
	taskID := c.Query("task_id")

	query := h.db.Preload("Participants.User").Where("workspace_id = ?", workspaceID)
	if boardID != "" {
		query = query.Where("board_id = ?", boardID)
	}
	if taskID != "" {
		query = query.Where("task_id = ?", taskID)
	}

	var meetings []models.Meeting
	query.Order("scheduled_at DESC").Find(&meetings)

	return c.JSON(fiber.Map{"meetings": meetings})
}

// ─── Export Handler ───────────────────────────────────────────────────────────

type ExportHandler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewExportHandler(db *gorm.DB, logger *zap.Logger) *ExportHandler {
	return &ExportHandler{db: db, logger: logger}
}

// ExportBoardExcel exports a board to Excel
func (h *ExportHandler) ExportBoardExcel(c *fiber.Ctx) error {
	boardID := c.Params("boardId")

	var board models.Board
	if err := h.db.First(&board, "id = ?", boardID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Board not found"})
	}

	var tasks []models.Task
	h.db.Preload("Assignees.User").Preload("Labels").
		Where("board_id = ?", boardID).
		Order("column_id, order_index").
		Find(&tasks)

	var columns []models.Column
	h.db.Where("board_id = ?", boardID).Order("order_index").Find(&columns)

	columnMap := map[uuid.UUID]string{}
	for _, col := range columns {
		columnMap[col.ID] = col.Name
	}

	f := excelize.NewFile()
	defer f.Close()

	// Create Tasks sheet
	sheetName := "Tasks"
	f.SetSheetName("Sheet1", sheetName)

	// Header style
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "#FFFFFF", Size: 11},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#6366F1"}, Pattern: 1},
		Border: []excelize.Border{
			{Type: "bottom", Color: "#4F46E5", Style: 2},
		},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})

	// Headers
	headers := []string{
		"ID", "Title", "Summary", "Column/Stage", "Priority", "Progress %",
		"Assignees", "Labels", "Start Date", "Due Date", "Completed At",
		"Estimated Hours", "Actual Hours", "Story Points", "Category",
		"GitHub Repo", "Branch", "PR Link", "Client Name", "Risk Level",
	}

	for i, h := range headers {
		col, _ := excelize.ColumnNumberToName(i + 1)
		f.SetCellValue(sheetName, col+"1", h)
		f.SetCellStyle(sheetName, col+"1", col+"1", headerStyle)
		f.SetColWidth(sheetName, col, col, 18)
	}

	// Data rows
	for rowIdx, task := range tasks {
		row := rowIdx + 2

		assigneeNames := []string{}
		for _, a := range task.Assignees {
			assigneeNames = append(assigneeNames, a.User.Name)
		}
		labelNames := []string{}
		for _, l := range task.Labels {
			labelNames = append(labelNames, l.Label)
		}

		values := []interface{}{
			task.ID.String()[:8],
			task.Title,
			task.Summary,
			columnMap[task.ColumnID],
			string(task.Priority),
			task.Progress,
			strings.Join(assigneeNames, ", "),
			strings.Join(labelNames, ", "),
			formatDate(task.StartDate),
			formatDate(task.DueDate),
			formatDate(task.CompletedAt),
			task.EstimatedHours,
			task.ActualHours,
			task.StoryPoints,
			task.Category,
			task.GitHubRepo,
			task.BranchName,
			task.PullRequestURL,
			task.ClientName,
			task.RiskLevel,
		}

		for colIdx, val := range values {
			colName, _ := excelize.ColumnNumberToName(colIdx + 1)
			f.SetCellValue(sheetName, fmt.Sprintf("%s%d", colName, row), val)
		}
	}

	// Create Analytics sheet
	analyticsSheet := "Analytics"
	f.NewSheet(analyticsSheet)

	var totalTasks, completedTasks, overdueTasks int64
	h.db.Model(&models.Task{}).Where("board_id = ?", boardID).Count(&totalTasks)
	h.db.Model(&models.Task{}).Where("board_id = ? AND completed_at IS NOT NULL", boardID).Count(&completedTasks)
	h.db.Model(&models.Task{}).Where("board_id = ? AND due_date < ? AND completed_at IS NULL", boardID, time.Now()).Count(&overdueTasks)

	analyticsData := [][]interface{}{
		{"Board Name", board.Name},
		{"Export Date", time.Now().Format("2006-01-02 15:04")},
		{""},
		{"Metric", "Value"},
		{"Total Tasks", totalTasks},
		{"Completed Tasks", completedTasks},
		{"Overdue Tasks", overdueTasks},
		{"Completion Rate", fmt.Sprintf("%.1f%%", safeDiv(float64(completedTasks), float64(totalTasks))*100)},
	}

	for rowIdx, row := range analyticsData {
		for colIdx, val := range row {
			colName, _ := excelize.ColumnNumberToName(colIdx + 1)
			f.SetCellValue(analyticsSheet, fmt.Sprintf("%s%d", colName, rowIdx+1), val)
		}
	}

	// Set response headers
	filename := fmt.Sprintf("board-%s-%s.xlsx", board.Name, time.Now().Format("20060102"))
	c.Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))

	buf, err := f.WriteToBuffer()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to generate Excel"})
	}

	return c.Send(buf.Bytes())
}

// ExportTasksCSV exports tasks as CSV
func (h *ExportHandler) ExportTasksCSV(c *fiber.Ctx) error {
	boardID := c.Params("boardId")

	var tasks []models.Task
	h.db.Where("board_id = ?", boardID).Find(&tasks)

	var sb strings.Builder
	sb.WriteString("ID,Title,Priority,Status,Estimated Hours,Actual Hours,Due Date,Completed At\n")

	for _, t := range tasks {
		sb.WriteString(fmt.Sprintf("%s,%s,%s,%s,%.1f,%.1f,%s,%s\n",
			t.ID.String()[:8],
			t.Title,
			t.Priority,
			t.Status,
			t.EstimatedHours,
			t.ActualHours,
			formatDate(t.DueDate),
			formatDate(t.CompletedAt),
		))
	}

	c.Set("Content-Type", "text/csv")
	c.Set("Content-Disposition", "attachment; filename=tasks.csv")
	return c.SendString(sb.String())
}

func formatDate(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02")
}

// ─── Sprint Handler ───────────────────────────────────────────────────────────

type SprintHandler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewSprintHandler(db *gorm.DB, logger *zap.Logger) *SprintHandler {
	return &SprintHandler{db: db, logger: logger}
}

// List returns sprints for a board
func (h *SprintHandler) List(c *fiber.Ctx) error {
	boardID := c.Params("boardId")
	var sprints []models.SprintPlan
	h.db.Preload("SprintTasks.Task").
		Where("board_id = ?", boardID).
		Order("created_at DESC").
		Find(&sprints)
	return c.JSON(fiber.Map{"sprints": sprints})
}

// Create creates a sprint
func (h *SprintHandler) Create(c *fiber.Ctx) error {
	boardID := c.Params("boardId")
	boardUID, _ := uuid.Parse(boardID)

	type SprintReq struct {
		Name      string     `json:"name"`
		Goal      string     `json:"goal"`
		StartDate *time.Time `json:"start_date"`
		EndDate   *time.Time `json:"end_date"`
		TaskIDs   []string   `json:"task_ids"`
	}
	var req SprintReq
	c.BodyParser(&req)

	sprint := models.SprintPlan{
		Base:      models.Base{ID: uuid.New()},
		BoardID:   boardUID,
		Name:      req.Name,
		Goal:      req.Goal,
		Status:    models.SprintPlanning,
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
	}
	h.db.Create(&sprint)

	for _, tID := range req.TaskIDs {
		tUID, _ := uuid.Parse(tID)
		h.db.Create(&models.SprintTask{
			Base:     models.Base{ID: uuid.New()},
			SprintID: sprint.ID,
			TaskID:   tUID,
			AddedAt:  time.Now(),
		})
		h.db.Model(&models.Task{}).Where("id = ?", tID).Update("sprint_id", sprint.ID)
	}

	return c.Status(fiber.StatusCreated).JSON(sprint)
}

// StartSprint activates a sprint
func (h *SprintHandler) StartSprint(c *fiber.Ctx) error {
	sprintID := c.Params("sprintId")
	h.db.Model(&models.SprintPlan{}).Where("id = ?", sprintID).
		Update("status", models.SprintActive)
	return c.JSON(fiber.Map{"message": "Sprint started"})
}

// CompleteSprint marks sprint as complete
func (h *SprintHandler) CompleteSprint(c *fiber.Ctx) error {
	sprintID := c.Params("sprintId")
	now := time.Now()
	h.db.Model(&models.SprintPlan{}).Where("id = ?", sprintID).
		Updates(map[string]interface{}{
			"status":       models.SprintCompleted,
			"completed_at": now,
		})
	return c.JSON(fiber.Map{"message": "Sprint completed"})
}

// GetBurndown returns burndown chart data
func (h *SprintHandler) GetBurndown(c *fiber.Ctx) error {
	sprintID := c.Params("sprintId")

	var sprint models.SprintPlan
	if err := h.db.Preload("SprintTasks.Task").First(&sprint, "id = ?", sprintID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Sprint not found"})
	}

	if sprint.StartDate == nil || sprint.EndDate == nil {
		return c.JSON(fiber.Map{"burndown": []interface{}{}})
	}

	// Calculate total story points
	totalPoints := 0
	for _, st := range sprint.SprintTasks {
		totalPoints += st.Task.StoryPoints
	}

	// Generate ideal burndown line
	days := int(sprint.EndDate.Sub(*sprint.StartDate).Hours() / 24)
	if days <= 0 {
		days = 1
	}
	pointsPerDay := float64(totalPoints) / float64(days)

	type BurndownPoint struct {
		Date          string  `json:"date"`
		Ideal         float64 `json:"ideal"`
		Actual        float64 `json:"actual"`
		CompletedPts  int     `json:"completed_pts"`
	}

	var burndown []BurndownPoint
	remaining := float64(totalPoints)

	for i := 0; i <= days; i++ {
		date := sprint.StartDate.AddDate(0, 0, i)
		ideal := float64(totalPoints) - float64(i)*pointsPerDay

		// Count completed tasks up to this date
		completedPts := 0
		for _, st := range sprint.SprintTasks {
			if st.Task.CompletedAt != nil && !st.Task.CompletedAt.After(date) {
				completedPts += st.Task.StoryPoints
			}
		}
		remaining = float64(totalPoints - completedPts)

		burndown = append(burndown, BurndownPoint{
			Date:         date.Format("2006-01-02"),
			Ideal:        ideal,
			Actual:       remaining,
			CompletedPts: completedPts,
		})
	}

	return c.JSON(fiber.Map{
		"sprint":       sprint,
		"burndown":     burndown,
		"total_points": totalPoints,
	})
}
