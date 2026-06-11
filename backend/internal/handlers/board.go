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

type BoardHandler struct {
	db     *gorm.DB
	hub    *websocket.Hub
	logger *zap.Logger
}

func NewBoardHandler(db *gorm.DB, hub *websocket.Hub, logger *zap.Logger) *BoardHandler {
	return &BoardHandler{db: db, hub: hub, logger: logger}
}

var defaultColumns = []struct {
	Name       string
	Color      string
	OrderIndex int
}{
	{"Inbox", "#6b7280", 0},
	{"Idea", "#8b5cf6", 1},
	{"Research", "#3b82f6", 2},
	{"Planning", "#f59e0b", 3},
	{"Design", "#ec4899", 4},
	{"Development", "#10b981", 5},
	{"Testing", "#f97316", 6},
	{"Review", "#6366f1", 7},
	{"Deployment", "#14b8a6", 8},
	{"Completed", "#22c55e", 9},
	{"Blocked", "#ef4444", 10},
}

// List returns all boards for a workspace
func (h *BoardHandler) List(c *fiber.Ctx) error {
	workspaceID := c.Params("workspaceId")
	userID := c.Locals("user_id").(string)

	var boards []models.Board
	h.db.Where("workspace_id = ? AND is_archived = false", workspaceID).
		Preload("Columns").
		Order("created_at ASC").
		Find(&boards)

	// Filter to boards user has access to
	accessible := boards[:0]
	for _, b := range boards {
		var count int64
		h.db.Model(&models.BoardMember{}).
			Where("board_id = ? AND user_id = ?", b.ID, userID).
			Count(&count)
		if count > 0 {
			accessible = append(accessible, b)
		} else {
			// Also allow workspace members
			var wcount int64
			h.db.Model(&models.WorkspaceMember{}).
				Where("workspace_id = ? AND user_id = ?", workspaceID, userID).
				Count(&wcount)
			if wcount > 0 {
				accessible = append(accessible, b)
			}
		}
	}

	return c.JSON(fiber.Map{"boards": accessible})
}

// Create creates a new board with default columns
func (h *BoardHandler) Create(c *fiber.Ctx) error {
	workspaceID := c.Params("workspaceId")
	workspaceUID, _ := uuid.Parse(workspaceID)
	userID := c.Locals("user_id").(string)
	userUID, _ := uuid.Parse(userID)

	type CreateRequest struct {
		Name         string                `json:"name"`
		Description  string                `json:"description"`
		DefaultView  models.BoardViewType  `json:"default_view"`
		Color        string                `json:"color"`
		Icon         string                `json:"icon"`
		TemplateType string                `json:"template_type"`
	}

	var req CreateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}
	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Board name required"})
	}
	if req.DefaultView == "" {
		req.DefaultView = models.ViewKanban
	}
	if req.Color == "" {
		req.Color = "#6366f1"
	}

	board := models.Board{
		Base:         models.Base{ID: uuid.New()},
		WorkspaceID:  workspaceUID,
		Name:         req.Name,
		Description:  req.Description,
		DefaultView:  req.DefaultView,
		Color:        req.Color,
		Icon:         req.Icon,
		TemplateType: req.TemplateType,
		CreatedBy:    userUID,
	}

	if err := h.db.Create(&board).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create board"})
	}

	// Create default columns
	cols := defaultColumns
	if req.TemplateType != "" {
		cols = getTemplateColumns(req.TemplateType)
	}

	for _, col := range cols {
		h.db.Create(&models.Column{
			Base:       models.Base{ID: uuid.New()},
			BoardID:    board.ID,
			Name:       col.Name,
			Color:      col.Color,
			OrderIndex: col.OrderIndex,
		})
	}

	// Add creator as owner board member
	h.db.Create(&models.BoardMember{
		Base:    models.Base{ID: uuid.New()},
		BoardID: board.ID,
		UserID:  userUID,
		Role:    models.RoleOwner,
	})

	h.db.Preload("Columns").First(&board, board.ID)
	return c.Status(fiber.StatusCreated).JSON(board)
}

// Get returns a full board with columns and tasks
func (h *BoardHandler) Get(c *fiber.Ctx) error {
	boardID := c.Params("boardId")

	var board models.Board
	if err := h.db.
		Preload("Columns", func(db *gorm.DB) *gorm.DB {
			return db.Order("order_index ASC")
		}).
		Preload("Columns.Tasks", func(db *gorm.DB) *gorm.DB {
			return db.Order("order_index ASC").
				Preload("Assignees.User").
				Preload("Labels")
		}).
		Preload("Members.User").
		First(&board, "id = ?", boardID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Board not found"})
	}

	return c.JSON(board)
}

// Update updates board metadata
func (h *BoardHandler) Update(c *fiber.Ctx) error {
	boardID := c.Params("boardId")

	type UpdateRequest struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Color       string `json:"color"`
		Icon        string `json:"icon"`
		DefaultView string `json:"default_view"`
	}

	var req UpdateRequest
	c.BodyParser(&req)

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
	if req.Icon != "" {
		updates["icon"] = req.Icon
	}
	if req.DefaultView != "" {
		updates["default_view"] = req.DefaultView
	}

	h.db.Model(&models.Board{}).Where("id = ?", boardID).Updates(updates)

	var board models.Board
	h.db.First(&board, "id = ?", boardID)
	return c.JSON(board)
}

// Delete soft-deletes a board
func (h *BoardHandler) Delete(c *fiber.Ctx) error {
	boardID := c.Params("boardId")
	h.db.Delete(&models.Board{}, "id = ?", boardID)
	return c.JSON(fiber.Map{"message": "Board deleted"})
}

// Archive toggles board archive status
func (h *BoardHandler) Archive(c *fiber.Ctx) error {
	boardID := c.Params("boardId")

	var board models.Board
	h.db.First(&board, "id = ?", boardID)
	h.db.Model(&board).Update("is_archived", !board.IsArchived)

	return c.JSON(fiber.Map{"is_archived": board.IsArchived})
}

// Duplicate creates a copy of a board
func (h *BoardHandler) Duplicate(c *fiber.Ctx) error {
	boardID := c.Params("boardId")
	userID := c.Locals("user_id").(string)
	userUID, _ := uuid.Parse(userID)

	var original models.Board
	if err := h.db.Preload("Columns").First(&original, "id = ?", boardID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Board not found"})
	}

	newBoard := models.Board{
		Base:        models.Base{ID: uuid.New()},
		WorkspaceID: original.WorkspaceID,
		Name:        original.Name + " (Copy)",
		Description: original.Description,
		DefaultView: original.DefaultView,
		Color:       original.Color,
		Icon:        original.Icon,
		CreatedBy:   userUID,
	}
	h.db.Create(&newBoard)

	for _, col := range original.Columns {
		h.db.Create(&models.Column{
			Base:       models.Base{ID: uuid.New()},
			BoardID:    newBoard.ID,
			Name:       col.Name,
			Color:      col.Color,
			OrderIndex: col.OrderIndex,
		})
	}

	h.db.Create(&models.BoardMember{
		Base:    models.Base{ID: uuid.New()},
		BoardID: newBoard.ID,
		UserID:  userUID,
		Role:    models.RoleOwner,
	})

	return c.Status(fiber.StatusCreated).JSON(newBoard)
}

// GetColumns returns board columns
func (h *BoardHandler) GetColumns(c *fiber.Ctx) error {
	boardID := c.Params("boardId")
	var columns []models.Column
	h.db.Where("board_id = ?", boardID).Order("order_index ASC").Find(&columns)
	return c.JSON(fiber.Map{"columns": columns})
}

// CreateColumn adds a new column
func (h *BoardHandler) CreateColumn(c *fiber.Ctx) error {
	boardID := c.Params("boardId")
	boardUID, _ := uuid.Parse(boardID)

	type ColRequest struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	}
	var req ColRequest
	c.BodyParser(&req)

	var maxOrder int
	h.db.Model(&models.Column{}).Where("board_id = ?", boardID).
		Select("COALESCE(MAX(order_index), -1)").Scan(&maxOrder)

	col := models.Column{
		Base:       models.Base{ID: uuid.New()},
		BoardID:    boardUID,
		Name:       req.Name,
		Color:      req.Color,
		OrderIndex: maxOrder + 1,
	}
	h.db.Create(&col)

	// Broadcast to board room
	h.hub.BroadcastToRoom("board:"+boardID, &websocket.Message{
		Type:      websocket.MsgColumnReordered,
		RoomID:    "board:" + boardID,
		Payload:   col,
		Timestamp: time.Now(),
	}, "")

	return c.Status(fiber.StatusCreated).JSON(col)
}

// ReorderColumns reorders columns
func (h *BoardHandler) ReorderColumns(c *fiber.Ctx) error {
	boardID := c.Params("boardId")

	type ReorderRequest struct {
		ColumnOrders []struct {
			ID         string `json:"id"`
			OrderIndex int    `json:"order_index"`
		} `json:"column_orders"`
	}

	var req ReorderRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	for _, col := range req.ColumnOrders {
		h.db.Model(&models.Column{}).Where("id = ? AND board_id = ?", col.ID, boardID).
			Update("order_index", col.OrderIndex)
	}

	h.hub.BroadcastToRoom("board:"+boardID, &websocket.Message{
		Type:      websocket.MsgColumnReordered,
		RoomID:    "board:" + boardID,
		Payload:   req.ColumnOrders,
		Timestamp: time.Now(),
	}, "")

	return c.JSON(fiber.Map{"message": "Columns reordered"})
}

// UpdateColumn updates a single column
func (h *BoardHandler) UpdateColumn(c *fiber.Ctx) error {
	boardID := c.Params("boardId")
	columnID := c.Params("columnId")

	type UpdateCol struct {
		Name     string `json:"name"`
		Color    string `json:"color"`
		WipLimit int    `json:"wip_limit"`
	}
	var req UpdateCol
	c.BodyParser(&req)

	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Color != "" {
		updates["color"] = req.Color
	}
	if req.WipLimit >= 0 {
		updates["wip_limit"] = req.WipLimit
	}

	h.db.Model(&models.Column{}).Where("id = ? AND board_id = ?", columnID, boardID).Updates(updates)

	var col models.Column
	h.db.First(&col, "id = ?", columnID)
	return c.JSON(col)
}

// DeleteColumn removes a column (moves tasks to Inbox or first column)
func (h *BoardHandler) DeleteColumn(c *fiber.Ctx) error {
	boardID := c.Params("boardId")
	columnID := c.Params("columnId")

	// Move tasks to first column
	var firstCol models.Column
	if err := h.db.Where("board_id = ? AND id != ?", boardID, columnID).
		Order("order_index ASC").First(&firstCol).Error; err == nil {
		h.db.Model(&models.Task{}).Where("column_id = ?", columnID).
			Update("column_id", firstCol.ID)
	}

	h.db.Delete(&models.Column{}, "id = ? AND board_id = ?", columnID, boardID)
	return c.JSON(fiber.Map{"message": "Column deleted"})
}

// GetMembers returns board members
func (h *BoardHandler) GetMembers(c *fiber.Ctx) error {
	boardID := c.Params("boardId")
	var members []models.BoardMember
	h.db.Preload("User").Where("board_id = ?", boardID).Find(&members)
	return c.JSON(fiber.Map{"members": members})
}

// GetPresence returns who is currently viewing the board
func (h *BoardHandler) GetPresence(c *fiber.Ctx) error {
	boardID := c.Params("boardId")
	presence := h.hub.GetRoomPresence("board:" + boardID)
	return c.JSON(fiber.Map{"presence": presence})
}

func getTemplateColumns(templateType string) []struct {
	Name       string
	Color      string
	OrderIndex int
} {
	templates := map[string][]struct {
		Name       string
		Color      string
		OrderIndex int
	}{
		"software": {
			{"Backlog", "#6b7280", 0},
			{"Ready", "#8b5cf6", 1},
			{"In Progress", "#3b82f6", 2},
			{"Code Review", "#f59e0b", 3},
			{"Testing", "#f97316", 4},
			{"Deployment", "#14b8a6", 5},
			{"Done", "#22c55e", 6},
		},
		"freelance": {
			{"Lead", "#6b7280", 0},
			{"Proposal", "#8b5cf6", 1},
			{"In Progress", "#3b82f6", 2},
			{"Review", "#f59e0b", 3},
			{"Revisions", "#f97316", 4},
			{"Invoiced", "#14b8a6", 5},
			{"Paid", "#22c55e", 6},
		},
		"sprint": {
			{"Backlog", "#6b7280", 0},
			{"Sprint Backlog", "#8b5cf6", 1},
			{"In Progress", "#3b82f6", 2},
			{"Review", "#f59e0b", 3},
			{"Testing", "#f97316", 4},
			{"Done", "#22c55e", 5},
		},
	}

	if cols, ok := templates[templateType]; ok {
		return cols
	}
	return defaultColumns
}
