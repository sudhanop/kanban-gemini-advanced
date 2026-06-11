package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/kanban-platform/backend/internal/models"
	"github.com/kanban-platform/backend/internal/websocket"
)

type WorkspaceHandler struct {
	db     *gorm.DB
	hub    *websocket.Hub
	logger *zap.Logger
}

func NewWorkspaceHandler(db *gorm.DB, hub *websocket.Hub, logger *zap.Logger) *WorkspaceHandler {
	return &WorkspaceHandler{db: db, hub: hub, logger: logger}
}

// List returns all workspaces for the current user
func (h *WorkspaceHandler) List(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)

	var workspaces []models.Workspace
	h.db.Joins("JOIN workspace_members ON workspace_members.workspace_id = workspaces.id").
		Where("workspace_members.user_id = ? AND workspaces.deleted_at IS NULL", userID).
		Preload("Owner").
		Order("workspaces.created_at DESC").
		Find(&workspaces)

	return c.JSON(fiber.Map{"workspaces": workspaces, "total": len(workspaces)})
}

// Create creates a new workspace
func (h *WorkspaceHandler) Create(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	userUID, _ := uuid.Parse(userID)

	type CreateRequest struct {
		Name        string                `json:"name" validate:"required,min=1,max=100"`
		Description string                `json:"description"`
		Type        models.WorkspaceType  `json:"type"`
		Color       string                `json:"color"`
	}

	var req CreateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Workspace name is required"})
	}

	if req.Type == "" {
		req.Type = models.WorkspacePersonal
	}
	if req.Color == "" {
		req.Color = "#6366f1"
	}

	workspace := models.Workspace{
		Base:        models.Base{ID: uuid.New()},
		Name:        req.Name,
		Slug:        generateSlug(req.Name),
		Description: req.Description,
		Type:        req.Type,
		OwnerID:     userUID,
		Color:       req.Color,
	}

	if err := h.db.Create(&workspace).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create workspace"})
	}

	// Add creator as owner
	h.db.Create(&models.WorkspaceMember{
		Base:        models.Base{ID: uuid.New()},
		WorkspaceID: workspace.ID,
		UserID:      userUID,
		Role:        models.RoleOwner,
		JoinedAt:    time.Now(),
	})

	h.db.Preload("Owner").First(&workspace, workspace.ID)

	return c.Status(fiber.StatusCreated).JSON(workspace)
}

// Get returns a single workspace
func (h *WorkspaceHandler) Get(c *fiber.Ctx) error {
	workspaceID := c.Params("workspaceId")
	userID := c.Locals("user_id").(string)

	var workspace models.Workspace
	if err := h.db.Preload("Owner").Preload("Members.User").
		First(&workspace, "id = ?", workspaceID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Workspace not found"})
	}

	// Verify access
	var member models.WorkspaceMember
	if err := h.db.Where("workspace_id = ? AND user_id = ?", workspaceID, userID).First(&member).Error; err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Access denied"})
	}

	return c.JSON(workspace)
}

// Update updates workspace details
func (h *WorkspaceHandler) Update(c *fiber.Ctx) error {
	workspaceID := c.Params("workspaceId")

	type UpdateRequest struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Color       string `json:"color"`
	}

	var req UpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	var workspace models.Workspace
	if err := h.db.First(&workspace, "id = ?", workspaceID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Workspace not found"})
	}

	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.Color != "" {
		updates["color"] = req.Color
	}

	h.db.Model(&workspace).Updates(updates)
	return c.JSON(workspace)
}

// Delete soft-deletes a workspace
func (h *WorkspaceHandler) Delete(c *fiber.Ctx) error {
	workspaceID := c.Params("workspaceId")
	userID := c.Locals("user_id").(string)

	var workspace models.Workspace
	if err := h.db.First(&workspace, "id = ? AND owner_id = ?", workspaceID, userID).Error; err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Only the owner can delete a workspace"})
	}

	h.db.Delete(&workspace)
	return c.JSON(fiber.Map{"message": "Workspace deleted successfully"})
}

// GetMembers returns workspace members
func (h *WorkspaceHandler) GetMembers(c *fiber.Ctx) error {
	workspaceID := c.Params("workspaceId")

	var members []models.WorkspaceMember
	h.db.Preload("User").Where("workspace_id = ?", workspaceID).Find(&members)

	return c.JSON(fiber.Map{"members": members})
}

// UpdateMemberRole updates a member's role
func (h *WorkspaceHandler) UpdateMemberRole(c *fiber.Ctx) error {
	workspaceID := c.Params("workspaceId")
	memberID := c.Params("memberId")

	type RoleRequest struct {
		Role models.UserRole `json:"role"`
	}

	var req RoleRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	h.db.Model(&models.WorkspaceMember{}).
		Where("id = ? AND workspace_id = ?", memberID, workspaceID).
		Update("role", req.Role)

	return c.JSON(fiber.Map{"message": "Role updated"})
}

// RemoveMember removes a member from workspace
func (h *WorkspaceHandler) RemoveMember(c *fiber.Ctx) error {
	workspaceID := c.Params("workspaceId")
	memberID := c.Params("memberId")

	h.db.Where("id = ? AND workspace_id = ?", memberID, workspaceID).
		Delete(&models.WorkspaceMember{})

	return c.JSON(fiber.Map{"message": "Member removed"})
}

// GetStats returns workspace-level analytics
func (h *WorkspaceHandler) GetStats(c *fiber.Ctx) error {
	workspaceID := c.Params("workspaceId")

	var totalTasks, completedTasks, overdueTasks int64
	h.db.Model(&models.Task{}).Where("workspace_id = ?", workspaceID).Count(&totalTasks)
	h.db.Model(&models.Task{}).Where("workspace_id = ? AND completed_at IS NOT NULL", workspaceID).Count(&completedTasks)
	h.db.Model(&models.Task{}).Where("workspace_id = ? AND due_date < ? AND completed_at IS NULL", workspaceID, time.Now()).Count(&overdueTasks)

	var memberCount int64
	h.db.Model(&models.WorkspaceMember{}).Where("workspace_id = ?", workspaceID).Count(&memberCount)

	var boardCount int64
	h.db.Model(&models.Board{}).Where("workspace_id = ?", workspaceID).Count(&boardCount)

	return c.JSON(fiber.Map{
		"total_tasks":     totalTasks,
		"completed_tasks": completedTasks,
		"overdue_tasks":   overdueTasks,
		"member_count":    memberCount,
		"board_count":     boardCount,
		"completion_rate": safeDiv(float64(completedTasks), float64(totalTasks)) * 100,
	})
}

func safeDiv(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return a / b
}

func generateSlug(name string) string {
	slug := ""
	for _, c := range name {
		if c == ' ' {
			slug += "-"
		} else if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			slug += string(c)
		} else if c >= 'A' && c <= 'Z' {
			slug += string(c + 32)
		}
	}
	return slug + "-" + uuid.New().String()[:8]
}
