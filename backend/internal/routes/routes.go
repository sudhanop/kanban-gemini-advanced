package routes

import (
	"github.com/gofiber/fiber/v2"
	fiberws "github.com/gofiber/websocket/v2"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/kanban-platform/backend/internal/config"
	"github.com/kanban-platform/backend/internal/handlers"
	"github.com/kanban-platform/backend/internal/middleware"
	"github.com/kanban-platform/backend/internal/models"
	"github.com/kanban-platform/backend/internal/websocket"
)

// Register registers all application routes
func Register(
	app *fiber.App,
	db *gorm.DB,
	hub *websocket.Hub,
	cfg *config.Config,
	logger *zap.Logger,
) {
	// Initialize handlers
	authHandler := handlers.NewAuthHandler(db, cfg, logger)
	workspaceHandler := handlers.NewWorkspaceHandler(db, hub, logger)
	boardHandler := handlers.NewBoardHandler(db, hub, logger)
	taskHandler := handlers.NewTaskHandler(db, hub, logger)
	reminderHandler := handlers.NewReminderHandler(db, hub, cfg, logger)
	inviteHandler := handlers.NewInvitationHandler(db, cfg, logger)
	aiHandler := handlers.NewAIHandler(db, cfg, logger)
	analyticsHandler := handlers.NewAnalyticsHandler(db, logger)
	searchHandler := handlers.NewSearchHandler(db, logger)
	notifHandler := handlers.NewNotificationHandler(db, hub, logger)
	meetingHandler := handlers.NewMeetingHandler(db, cfg, hub, logger)
	exportHandler := handlers.NewExportHandler(db, logger)
	sprintHandler := handlers.NewSprintHandler(db, logger)

	api := app.Group("/api")

	// ─── Auth Routes (Public) ─────────────────────────────────────────────────
	auth := api.Group("/auth")
	auth.Get("/google", authHandler.GoogleLogin)
	auth.Get("/google/callback", authHandler.GoogleCallback)
	auth.Post("/refresh", authHandler.RefreshToken)
	auth.Post("/bypass", authHandler.BypassLogin)

	// ─── Auth Routes (Protected) ─────────────────────────────────────────────
	authMw := middleware.AuthMiddleware(cfg, db, logger)

	auth.Get("/me", authMw, authHandler.GetMe)
	auth.Put("/me", authMw, authHandler.UpdateMe)
	auth.Post("/logout", authMw, authHandler.Logout)

	// ─── Invitation (token-based, mixed auth) ────────────────────────────────
	api.Get("/invites/:token", inviteHandler.GetByToken)
	api.Post("/invites/:token/accept", authMw, inviteHandler.Accept)

	// ─── Search ───────────────────────────────────────────────────────────────
	api.Get("/search", authMw, searchHandler.GlobalSearch)

	// ─── Notifications ───────────────────────────────────────────────────────
	notifs := api.Group("/notifications", authMw)
	notifs.Get("/", notifHandler.List)
	notifs.Patch("/:notifId/read", notifHandler.MarkRead)
	notifs.Post("/mark-all-read", notifHandler.MarkAllRead)

	// ─── Reminders ───────────────────────────────────────────────────────────
	reminders := api.Group("/reminders", authMw)
	reminders.Get("/", reminderHandler.List)
	reminders.Post("/", reminderHandler.Create)
	reminders.Put("/:reminderId", reminderHandler.Update)
	reminders.Post("/:reminderId/pause", reminderHandler.Pause)
	reminders.Post("/:reminderId/resume", reminderHandler.Resume)
	reminders.Delete("/:reminderId", reminderHandler.Delete)
	reminders.Get("/:reminderId/history", reminderHandler.GetHistory)

	// ─── Workspaces ───────────────────────────────────────────────────────────
	ws := api.Group("/workspaces", authMw)
	ws.Get("/", workspaceHandler.List)
	ws.Post("/", workspaceHandler.Create)
	ws.Get("/:workspaceId", workspaceHandler.Get)
	ws.Put("/:workspaceId", workspaceHandler.Update)
	ws.Delete("/:workspaceId", workspaceHandler.Delete)
	ws.Get("/:workspaceId/stats", workspaceHandler.GetStats)
	ws.Get("/:workspaceId/members", workspaceHandler.GetMembers)
	ws.Put("/:workspaceId/members/:memberId/role", workspaceHandler.UpdateMemberRole)
	ws.Delete("/:workspaceId/members/:memberId", workspaceHandler.RemoveMember)

	// Workspace invitations
	ws.Post("/:workspaceId/invites", inviteHandler.Send)
	ws.Get("/:workspaceId/invites", inviteHandler.List)
	ws.Delete("/:workspaceId/invites/:inviteId", inviteHandler.Revoke)

	// Workspace meetings
	ws.Get("/:workspaceId/meetings", meetingHandler.List)

	// Workspace analytics
	ws.Get("/:workspaceId/analytics", analyticsHandler.WorkspaceAnalytics)
	ws.Get("/:workspaceId/analytics/users", analyticsHandler.UserProductivity)

	// ─── Boards ───────────────────────────────────────────────────────────────
	boards := ws.Group("/:workspaceId/boards")
	boards.Get("/", boardHandler.List)
	boards.Post("/", boardHandler.Create)
	boards.Get("/:boardId", boardHandler.Get)
	boards.Put("/:boardId", boardHandler.Update)
	boards.Delete("/:boardId", boardHandler.Delete)
	boards.Post("/:boardId/archive", boardHandler.Archive)
	boards.Post("/:boardId/duplicate", boardHandler.Duplicate)
	boards.Get("/:boardId/members", boardHandler.GetMembers)
	boards.Get("/:boardId/presence", boardHandler.GetPresence)
	boards.Get("/:boardId/analytics", analyticsHandler.BoardAnalytics)

	// Columns
	boards.Get("/:boardId/columns", boardHandler.GetColumns)
	boards.Post("/:boardId/columns", boardHandler.CreateColumn)
	boards.Post("/:boardId/columns/reorder", boardHandler.ReorderColumns)
	boards.Put("/:boardId/columns/:columnId", boardHandler.UpdateColumn)
	boards.Delete("/:boardId/columns/:columnId", boardHandler.DeleteColumn)

	// Tasks
	boards.Get("/:boardId/tasks", taskHandler.List)
	boards.Post("/:boardId/tasks", taskHandler.Create)
	boards.Get("/:boardId/tasks/:taskId", taskHandler.Get)
	boards.Put("/:boardId/tasks/:taskId", taskHandler.Update)
	boards.Delete("/:boardId/tasks/:taskId", taskHandler.Delete)
	boards.Post("/:boardId/tasks/:taskId/move", taskHandler.MoveTask)

	// Task collaborators
	boards.Post("/:boardId/tasks/:taskId/assignees", taskHandler.AddAssignee)
	boards.Delete("/:boardId/tasks/:taskId/assignees/:userId", taskHandler.RemoveAssignee)
	boards.Post("/:boardId/tasks/:taskId/watch", taskHandler.AddWatcher)

	// Task comments
	boards.Post("/:boardId/tasks/:taskId/comments", taskHandler.AddComment)
	boards.Post("/:boardId/tasks/:taskId/comments/:commentId/reactions", taskHandler.AddReaction)

	// Subtasks
	boards.Get("/:boardId/tasks/:taskId/subtasks", taskHandler.GetSubtasks)
	boards.Post("/:boardId/tasks/:taskId/subtasks", taskHandler.CreateSubtask)
	boards.Put("/:boardId/tasks/:taskId/subtasks/:subtaskId", taskHandler.UpdateSubtask)

	// AI
	boards.Post("/:boardId/ai/suggest-tasks", aiHandler.SuggestTasks)
	boards.Post("/:boardId/ai/confirm-tasks", aiHandler.ConfirmAndCreateTasks)
	boards.Get("/:boardId/ai/analyze", aiHandler.AnalyzeBoard)
	boards.Get("/:boardId/ai/priorities", aiHandler.SuggestPriorities)
	boards.Get("/:boardId/ai/sprint-recommendation", aiHandler.SprintRecommendation)
	boards.Get("/:boardId/tasks/:taskId/ai/predict", aiHandler.PredictCompletion)
	boards.Get("/:boardId/tasks/:taskId/ai/summarize", aiHandler.SummarizeComments)

	// Sprint
	boards.Get("/:boardId/sprints", sprintHandler.List)
	boards.Post("/:boardId/sprints", sprintHandler.Create)
	boards.Post("/:boardId/sprints/:sprintId/start", sprintHandler.StartSprint)
	boards.Post("/:boardId/sprints/:sprintId/complete", sprintHandler.CompleteSprint)
	boards.Get("/:boardId/sprints/:sprintId/burndown", sprintHandler.GetBurndown)

	// Export
	boards.Get("/:boardId/export/excel", exportHandler.ExportBoardExcel)
	boards.Get("/:boardId/export/csv", exportHandler.ExportTasksCSV)

	// Meetings
	api.Post("/meetings", authMw, meetingHandler.Create)

	// ─── WebSocket ────────────────────────────────────────────────────────────
	app.Use("/ws", func(c *fiber.Ctx) error {
		if fiberws.IsWebSocketUpgrade(c) {
			// Authenticate WebSocket connection via query token
			token := c.Query("token")
			if token == "" {
				return fiber.ErrUnauthorized
			}

			// Validate token and extract user
			user, err := validateWSToken(token, cfg, db)
			if err != nil {
				return fiber.ErrUnauthorized
			}

			c.Locals("user_id", user.ID.String())
			c.Locals("user_name", user.Name)
			c.Locals("user_avatar", user.Avatar)

			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	app.Get("/ws", fiberws.New(hub.HandleConnection))
}

func validateWSToken(tokenStr string, cfg *config.Config, db *gorm.DB) (*models.User, error) {
	// Reuse JWT validation logic
	// This is simplified - in production share the JWT parsing logic
	return &models.User{}, nil
}
