package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/kanban-platform/backend/internal/config"
	"github.com/kanban-platform/backend/internal/handlers"
	"github.com/kanban-platform/backend/internal/models"
)

// AuthMiddleware validates JWT and injects user context
func AuthMiddleware(cfg *config.Config, db *gorm.DB, logger *zap.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			// Try cookie-based token
			authHeader = "Bearer " + c.Cookies("access_token")
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Authentication required",
			})
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenStr == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Token not provided",
			})
		}

		// Parse and validate JWT
		claims := &handlers.JWTClaims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fiber.ErrUnauthorized
			}
			return []byte(cfg.JWTSecret), nil
		})

		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid or expired token",
			})
		}

		// Check user still active in DB
		var user models.User
		if err := db.Select("id, email, role, status").First(&user, "id = ?", claims.UserID).Error; err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "User not found",
			})
		}

		if user.Status == models.StatusSuspended {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Account suspended",
			})
		}

		// Inject into context
		c.Locals("user_id", claims.UserID)
		c.Locals("user_email", claims.Email)
		c.Locals("user_role", claims.Role)
		c.Locals("user", &user)

		return c.Next()
	}
}

// RequireRole enforces minimum role level
func RequireRole(minRole models.UserRole) fiber.Handler {
	roleLevel := map[models.UserRole]int{
		models.RoleOwner:     5,
		models.RoleAdmin:     4,
		models.RoleDeveloper: 3,
		models.RoleViewer:    2,
		models.RoleGuest:     1,
	}

	return func(c *fiber.Ctx) error {
		userRole := models.UserRole(c.Locals("user_role").(string))
		if roleLevel[userRole] < roleLevel[minRole] {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Insufficient permissions",
				"required_role": string(minRole),
			})
		}
		return c.Next()
	}
}

// WorkspacePermission checks user has access to workspace
func WorkspacePermission(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := c.Locals("user_id").(string)
		workspaceID := c.Params("workspaceId")
		if workspaceID == "" {
			return c.Next()
		}

		var member models.WorkspaceMember
		if err := db.Where("workspace_id = ? AND user_id = ?", workspaceID, userID).First(&member).Error; err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Access denied to this workspace",
			})
		}

		c.Locals("workspace_role", string(member.Role))
		return c.Next()
	}
}

// BoardPermission checks user has access to a board
func BoardPermission(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := c.Locals("user_id").(string)
		boardID := c.Params("boardId")
		if boardID == "" {
			return c.Next()
		}

		// Check if user is a board member or workspace member
		var count int64
		db.Raw(`
			SELECT COUNT(*) FROM (
				SELECT id FROM board_members WHERE board_id = ? AND user_id = ? AND deleted_at IS NULL
				UNION
				SELECT bm.id FROM boards b
				JOIN workspace_members wm ON wm.workspace_id = b.workspace_id
				JOIN board_members bm ON bm.board_id = b.id
				WHERE b.id = ? AND wm.user_id = ? AND b.deleted_at IS NULL AND wm.deleted_at IS NULL
			) t
		`, boardID, userID, boardID, userID).Scan(&count)

		if count == 0 {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Access denied to this board",
			})
		}

		return c.Next()
	}
}
