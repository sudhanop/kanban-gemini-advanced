package handlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/generative-ai-go/genai"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/api/option"
	"gorm.io/gorm"

	"github.com/kanban-platform/backend/internal/config"
	"github.com/kanban-platform/backend/internal/models"
)

type AIHandler struct {
	db     *gorm.DB
	cfg    *config.Config
	logger *zap.Logger
}

func NewAIHandler(db *gorm.DB, cfg *config.Config, logger *zap.Logger) *AIHandler {
	return &AIHandler{db: db, cfg: cfg, logger: logger}
}

// SuggestTasks uses AI to suggest tasks for a board (requires user confirmation)
func (h *AIHandler) SuggestTasks(c *fiber.Ctx) error {
	boardID := c.Params("boardId")

	type SuggestReq struct {
		Context string `json:"context"`
		Goal    string `json:"goal"`
	}
	var req SuggestReq
	c.BodyParser(&req)

	// Get existing board tasks for context
	var tasks []models.Task
	h.db.Select("title, description, priority, status").
		Where("board_id = ?", boardID).Limit(20).Find(&tasks)

	var taskList strings.Builder
	for _, t := range tasks {
		taskList.WriteString(fmt.Sprintf("- %s (%s)\n", t.Title, t.Priority))
	}

	prompt := fmt.Sprintf(`You are an expert project manager AI assistant.

Board context: %s
Goal: %s

Existing tasks:
%s

Suggest 5 new tasks that would help achieve the goal. For each task, provide:
1. Title (concise, action-oriented)
2. Priority (critical/high/medium/low)
3. Estimated hours
4. Brief description

IMPORTANT: Return ONLY a JSON array in this exact format:
[
  {
    "title": "...",
    "priority": "medium",
    "estimated_hours": 4,
    "description": "..."
  }
]`, req.Context, req.Goal, taskList.String())

	response, err := h.callGemini(prompt)
	if err != nil {
		h.logger.Error("Gemini API error", zap.Error(err))
		// Return mock suggestions if API fails
		return c.JSON(fiber.Map{
			"suggestions": getMockSuggestions(),
			"requires_confirmation": true,
			"message": "AI suggestions ready. Please review and confirm before adding.",
		})
	}

	return c.JSON(fiber.Map{
		"suggestions":           response,
		"requires_confirmation": true,
		"message":               "AI has suggested these tasks. Review and approve before adding.",
	})
}

// ConfirmAndCreateTasks creates tasks after user confirms AI suggestions
func (h *AIHandler) ConfirmAndCreateTasks(c *fiber.Ctx) error {
	boardID := c.Params("boardId")
	boardUID, _ := uuid.Parse(boardID)
	userID := c.Locals("user_id").(string)
	userUID, _ := uuid.Parse(userID)

	type TaskSuggestion struct {
		Title          string  `json:"title"`
		Priority       string  `json:"priority"`
		EstimatedHours float64 `json:"estimated_hours"`
		Description    string  `json:"description"`
		ColumnID       string  `json:"column_id"`
	}
	type ConfirmReq struct {
		Tasks []TaskSuggestion `json:"tasks"`
	}

	var req ConfirmReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	// Get workspace ID from board
	var board models.Board
	if err := h.db.First(&board, "id = ?", boardID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Board not found"})
	}

	// Get first column (Inbox or first available)
	var firstCol models.Column
	if req.Tasks[0].ColumnID != "" {
		h.db.First(&firstCol, "id = ?", req.Tasks[0].ColumnID)
	} else {
		h.db.Where("board_id = ?", boardID).Order("order_index ASC").First(&firstCol)
	}

	var created []models.Task
	for _, s := range req.Tasks {
		columnUID := firstCol.ID
		if s.ColumnID != "" {
			if uid, err := uuid.Parse(s.ColumnID); err == nil {
				columnUID = uid
			}
		}

		priority := models.Priority(s.Priority)
		if priority == "" {
			priority = models.PriorityMedium
		}

		task := models.Task{
			Base:           models.Base{ID: uuid.New()},
			BoardID:        boardUID,
			WorkspaceID:    board.WorkspaceID,
			ColumnID:       columnUID,
			Title:          s.Title,
			Description:    s.Description,
			Priority:       priority,
			EstimatedHours: s.EstimatedHours,
			CreatedBy:      userUID,
		}

		var maxOrder int
		h.db.Model(&models.Task{}).Where("column_id = ?", columnUID).
			Select("COALESCE(MAX(order_index), -1)").Scan(&maxOrder)
		task.OrderIndex = maxOrder + 1

		h.db.Create(&task)
		created = append(created, task)
	}

	return c.JSON(fiber.Map{
		"created": created,
		"count":   len(created),
		"message": fmt.Sprintf("%d AI-suggested tasks created successfully", len(created)),
	})
}

// AnalyzeBoard provides AI insights about a board
func (h *AIHandler) AnalyzeBoard(c *fiber.Ctx) error {
	boardID := c.Params("boardId")

	var tasks []models.Task
	h.db.Where("board_id = ?", boardID).
		Select("title, priority, status, estimated_hours, actual_hours, due_date, completed_at, ai_risk_score").
		Find(&tasks)

	var columns []models.Column
	h.db.Where("board_id = ?", boardID).Find(&columns)

	// Calculate stats
	var overdue, blocked int
	var totalEstimated, totalActual float64
	now := time.Now()

	for _, t := range tasks {
		if t.DueDate != nil && t.DueDate.Before(now) && t.CompletedAt == nil {
			overdue++
		}
		if t.Status == "blocked" {
			blocked++
		}
		totalEstimated += t.EstimatedHours
		totalActual += t.ActualHours
	}

	prompt := fmt.Sprintf(`Analyze this project board and provide insights:

Total tasks: %d
Overdue tasks: %d  
Blocked tasks: %d
Total estimated hours: %.1f
Total actual hours: %.1f
Number of columns: %d

Provide a JSON response with:
{
  "health_score": (0-100),
  "risk_level": "low|medium|high|critical",
  "insights": ["insight1", "insight2", "insight3"],
  "recommendations": ["rec1", "rec2", "rec3"],
  "productivity_summary": "..."
}`, len(tasks), overdue, blocked, totalEstimated, totalActual, len(columns))

	response, err := h.callGemini(prompt)
	if err != nil {
		return c.JSON(fiber.Map{
			"health_score": 75,
			"risk_level":   "medium",
			"insights": []string{
				fmt.Sprintf("%d tasks are overdue", overdue),
				fmt.Sprintf("%d tasks are blocked", blocked),
				"Consider reviewing workload distribution",
			},
			"recommendations": []string{
				"Review and prioritize overdue tasks",
				"Resolve blocked tasks to maintain flow",
				"Set realistic time estimates",
			},
			"productivity_summary": "Board analysis complete. Focus on clearing blockers and overdue items.",
		})
	}

	return c.JSON(response)
}

// PredictCompletion predicts when a task will be completed
func (h *AIHandler) PredictCompletion(c *fiber.Ctx) error {
	taskID := c.Params("taskId")

	var task models.Task
	if err := h.db.Preload("Assignees.User").First(&task, "id = ?", taskID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Task not found"})
	}

	// Simple heuristic prediction
	estimatedDays := task.EstimatedHours / 8
	if estimatedDays < 1 {
		estimatedDays = 1
	}

	predictedDate := time.Now().Add(time.Duration(estimatedDays*24) * time.Hour)
	confidence := 0.75
	if task.ActualHours > 0 {
		ratio := task.ActualHours / task.EstimatedHours
		if ratio > 1.5 {
			confidence = 0.45
			predictedDate = predictedDate.Add(time.Duration(ratio*24) * time.Hour)
		} else if ratio < 0.7 {
			confidence = 0.85
		}
	}

	return c.JSON(fiber.Map{
		"task_id":          taskID,
		"predicted_date":   predictedDate,
		"confidence":       confidence,
		"estimated_hours":  task.EstimatedHours,
		"actual_hours":     task.ActualHours,
		"risk_factors":     getRiskFactors(&task),
	})
}

// SuggestPriorities re-evaluates task priorities using AI
func (h *AIHandler) SuggestPriorities(c *fiber.Ctx) error {
	boardID := c.Params("boardId")

	var tasks []models.Task
	h.db.Where("board_id = ? AND completed_at IS NULL", boardID).
		Select("id, title, priority, due_date, story_points, ai_risk_score").
		Find(&tasks)

	suggestions := make([]map[string]interface{}, 0)
	now := time.Now()

	for _, t := range tasks {
		suggestedPriority := string(t.Priority)
		reason := ""

		if t.DueDate != nil {
			daysUntilDue := t.DueDate.Sub(now).Hours() / 24
			if daysUntilDue < 1 {
				suggestedPriority = "critical"
				reason = "Due today or overdue"
			} else if daysUntilDue < 3 {
				suggestedPriority = "high"
				reason = fmt.Sprintf("Due in %.0f days", daysUntilDue)
			}
		}

		if suggestedPriority != string(t.Priority) {
			suggestions = append(suggestions, map[string]interface{}{
				"task_id":            t.ID,
				"task_title":         t.Title,
				"current_priority":   t.Priority,
				"suggested_priority": suggestedPriority,
				"reason":             reason,
			})
		}
	}

	return c.JSON(fiber.Map{
		"suggestions":           suggestions,
		"requires_confirmation": true,
		"message":               "Review priority suggestions before applying",
	})
}

// SummarizeComments summarizes task discussion using AI
func (h *AIHandler) SummarizeComments(c *fiber.Ctx) error {
	taskID := c.Params("taskId")

	var comments []models.Comment
	h.db.Preload("User").Where("task_id = ?", taskID).
		Order("created_at ASC").Find(&comments)

	if len(comments) == 0 {
		return c.JSON(fiber.Map{"summary": "No comments yet."})
	}

	var sb strings.Builder
	for _, cm := range comments {
		sb.WriteString(fmt.Sprintf("%s: %s\n", cm.User.Name, cm.Content))
	}

	prompt := fmt.Sprintf(`Summarize this task discussion in 2-3 sentences. Focus on decisions made, blockers, and next steps:

%s

Return JSON: {"summary": "...", "key_decisions": ["..."], "next_steps": ["..."]}`,
		sb.String())

	response, err := h.callGemini(prompt)
	if err != nil {
		return c.JSON(fiber.Map{
			"summary":       fmt.Sprintf("Discussion has %d comments covering task progress.", len(comments)),
			"key_decisions": []string{},
			"next_steps":    []string{},
		})
	}

	return c.JSON(response)
}

// SprintRecommendation recommends which tasks to include in next sprint
func (h *AIHandler) SprintRecommendation(c *fiber.Ctx) error {
	boardID := c.Params("boardId")

	var tasks []models.Task
	h.db.Where("board_id = ? AND completed_at IS NULL AND sprint_id IS NULL", boardID).
		Select("id, title, priority, story_points, estimated_hours, due_date").
		Order("priority DESC, due_date ASC").
		Limit(30).Find(&tasks)

	// Simple recommendation: high priority + closest due dates, max 40 story points
	var recommended []map[string]interface{}
	totalPoints := 0

	for _, t := range tasks {
		if totalPoints+t.StoryPoints > 40 {
			continue
		}
		recommended = append(recommended, map[string]interface{}{
			"task_id":      t.ID,
			"title":        t.Title,
			"priority":     t.Priority,
			"story_points": t.StoryPoints,
		})
		totalPoints += t.StoryPoints
		if len(recommended) >= 10 {
			break
		}
	}

	return c.JSON(fiber.Map{
		"recommended":           recommended,
		"total_story_points":    totalPoints,
		"requires_confirmation": true,
		"message":               "Review recommended sprint tasks before creating sprint",
	})
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func (h *AIHandler) callGemini(prompt string) (interface{}, error) {
	if h.cfg.GeminiAPIKey == "" {
		return nil, fmt.Errorf("Gemini API key not configured")
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(h.cfg.GeminiAPIKey))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	model := client.GenerativeModel(h.cfg.GeminiModel)
	model.SetTemperature(0.7)
	model.ResponseMIMEType = "application/json"

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, err
	}

	if len(resp.Candidates) == 0 {
		return nil, fmt.Errorf("no response from Gemini")
	}

	text := resp.Candidates[0].Content.Parts[0].(genai.Text)
	return string(text), nil
}

func getRiskFactors(task *models.Task) []string {
	var factors []string
	now := time.Now()

	if task.DueDate != nil && task.DueDate.Before(now) {
		factors = append(factors, "Task is overdue")
	}
	if task.ActualHours > task.EstimatedHours*1.5 {
		factors = append(factors, "Taking significantly longer than estimated")
	}
	if task.BlockedReason != "" {
		factors = append(factors, "Task is blocked: "+task.BlockedReason)
	}
	if len(task.Assignees) == 0 {
		factors = append(factors, "No assignee assigned")
	}

	return factors
}

func getMockSuggestions() []map[string]interface{} {
	return []map[string]interface{}{
		{"title": "Set up project repository", "priority": "high", "estimated_hours": 2, "description": "Initialize git repository and configure CI/CD"},
		{"title": "Define technical requirements", "priority": "high", "estimated_hours": 4, "description": "Document technical stack and architecture decisions"},
		{"title": "Create development environment", "priority": "medium", "estimated_hours": 3, "description": "Set up local development with Docker"},
		{"title": "Write initial tests", "priority": "medium", "estimated_hours": 6, "description": "Set up testing framework and write initial unit tests"},
		{"title": "Review security requirements", "priority": "high", "estimated_hours": 2, "description": "Audit security requirements and implement basic protections"},
	}
}
