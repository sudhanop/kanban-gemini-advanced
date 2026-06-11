package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gopkg.in/gomail.v2"
	"gorm.io/gorm"

	"github.com/kanban-platform/backend/internal/config"
	"github.com/kanban-platform/backend/internal/models"
)

type InvitationHandler struct {
	db     *gorm.DB
	cfg    *config.Config
	logger *zap.Logger
}

func NewInvitationHandler(db *gorm.DB, cfg *config.Config, logger *zap.Logger) *InvitationHandler {
	return &InvitationHandler{db: db, cfg: cfg, logger: logger}
}

// Send sends an invitation email
func (h *InvitationHandler) Send(c *fiber.Ctx) error {
	workspaceID := c.Params("workspaceId")
	workspaceUID, _ := uuid.Parse(workspaceID)
	userID := c.Locals("user_id").(string)
	userUID, _ := uuid.Parse(userID)

	type InviteReq struct {
		Email   string          `json:"email"`
		Role    models.UserRole `json:"role"`
		BoardID *string         `json:"board_id"`
	}

	var req InviteReq
	if err := c.BodyParser(&req); err != nil || req.Email == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Email is required"})
	}

	if req.Role == "" {
		req.Role = models.RoleDeveloper
	}

	// Check not already a member
	var existing models.WorkspaceMember
	h.db.Joins("JOIN users ON users.id = workspace_members.user_id").
		Where("workspace_members.workspace_id = ? AND users.email = ?", workspaceID, req.Email).
		First(&existing)
	if existing.ID != uuid.Nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "User is already a member"})
	}

	// Check for pending invite
	var pendingInvite models.Invitation
	h.db.Where("workspace_id = ? AND email = ? AND status = ?",
		workspaceID, req.Email, models.InvitationPending).First(&pendingInvite)
	if pendingInvite.ID != uuid.Nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "Invitation already sent"})
	}

	// Generate secure token
	tokenBytes := make([]byte, 32)
	rand.Read(tokenBytes)
	token := hex.EncodeToString(tokenBytes)

	invitation := models.Invitation{
		Base:        models.Base{ID: uuid.New()},
		WorkspaceID: workspaceUID,
		Email:       req.Email,
		Role:        req.Role,
		Token:       token,
		InvitedBy:   userUID,
		Status:      models.InvitationPending,
		ExpiresAt:   time.Now().Add(time.Duration(h.cfg.InvitationExpiryHours) * time.Hour),
	}

	if req.BoardID != nil {
		boardUID, _ := uuid.Parse(*req.BoardID)
		invitation.BoardID = &boardUID
	}

	if err := h.db.Create(&invitation).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create invitation"})
	}

	// Get workspace info for email
	var workspace models.Workspace
	h.db.Preload("Owner").First(&workspace, "id = ?", workspaceID)

	// Get inviter name
	var inviter models.User
	h.db.First(&inviter, "id = ?", userID)

	// Send invitation email
	go h.sendInviteEmail(req.Email, token, inviter.Name, workspace.Name)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message":    "Invitation sent",
		"invitation": invitation,
	})
}

// List returns all pending invitations for a workspace
func (h *InvitationHandler) List(c *fiber.Ctx) error {
	workspaceID := c.Params("workspaceId")

	var invitations []models.Invitation
	h.db.Preload("InvitedByUser").
		Where("workspace_id = ?", workspaceID).
		Order("created_at DESC").
		Find(&invitations)

	return c.JSON(fiber.Map{"invitations": invitations})
}

// Accept accepts an invitation using the token
func (h *InvitationHandler) Accept(c *fiber.Ctx) error {
	token := c.Params("token")
	userID := c.Locals("user_id").(string)
	userUID, _ := uuid.Parse(userID)

	var invite models.Invitation
	if err := h.db.Preload("InvitedByUser").
		Where("token = ? AND status = ?", token, models.InvitationPending).
		First(&invite).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Invitation not found or already used"})
	}

	// Check expiry
	if time.Now().After(invite.ExpiresAt) {
		h.db.Model(&invite).Update("status", models.InvitationExpired)
		return c.Status(fiber.StatusGone).JSON(fiber.Map{"error": "Invitation has expired"})
	}

	// Verify email matches logged-in user
	var user models.User
	h.db.First(&user, "id = ?", userID)
	if user.Email != invite.Email {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": fmt.Sprintf("This invitation was sent to %s. Please log in with that account.", invite.Email),
		})
	}

	// Add user to workspace
	member := models.WorkspaceMember{
		Base:        models.Base{ID: uuid.New()},
		WorkspaceID: invite.WorkspaceID,
		UserID:      userUID,
		Role:        invite.Role,
		JoinedAt:    time.Now(),
	}
	h.db.Where(models.WorkspaceMember{WorkspaceID: invite.WorkspaceID, UserID: userUID}).
		FirstOrCreate(&member)

	// Add to board if specified
	if invite.BoardID != nil {
		h.db.Where(models.BoardMember{BoardID: *invite.BoardID, UserID: userUID}).
			FirstOrCreate(&models.BoardMember{
				Base:    models.Base{ID: uuid.New()},
				BoardID: *invite.BoardID,
				UserID:  userUID,
				Role:    invite.Role,
			})
	}

	// Mark invitation as accepted
	now := time.Now()
	h.db.Model(&invite).Updates(map[string]interface{}{
		"status":      models.InvitationAccepted,
		"accepted_at": now,
	})

	return c.JSON(fiber.Map{
		"message":      "Invitation accepted",
		"workspace_id": invite.WorkspaceID,
	})
}

// Revoke revokes a pending invitation
func (h *InvitationHandler) Revoke(c *fiber.Ctx) error {
	inviteID := c.Params("inviteId")
	workspaceID := c.Params("workspaceId")

	h.db.Model(&models.Invitation{}).
		Where("id = ? AND workspace_id = ? AND status = ?", inviteID, workspaceID, models.InvitationPending).
		Update("status", models.InvitationRevoked)

	return c.JSON(fiber.Map{"message": "Invitation revoked"})
}

// GetByToken returns invitation details (for accept page preview)
func (h *InvitationHandler) GetByToken(c *fiber.Ctx) error {
	token := c.Params("token")

	var invite models.Invitation
	if err := h.db.Preload("InvitedByUser").
		Where("token = ?", token).First(&invite).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Invitation not found"})
	}

	var workspace models.Workspace
	h.db.Select("id, name, color, type").First(&workspace, "id = ?", invite.WorkspaceID)

	return c.JSON(fiber.Map{
		"invitation": invite,
		"workspace":  workspace,
	})
}

func (h *InvitationHandler) sendInviteEmail(toEmail, token, inviterName, workspaceName string) {
	if h.cfg.SMTPUser == "" {
		h.logger.Warn("SMTP not configured, skipping email")
		return
	}

	inviteURL := fmt.Sprintf("%s/invite/%s", h.cfg.FrontendURL, token)

	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head><style>
  body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; background: #0f0f1a; color: #e2e8f0; margin: 0; padding: 20px; }
  .container { max-width: 600px; margin: 0 auto; background: #1a1a2e; border-radius: 12px; padding: 40px; }
  .logo { font-size: 24px; font-weight: 700; color: #818cf8; margin-bottom: 24px; }
  h1 { font-size: 28px; font-weight: 700; margin-bottom: 16px; }
  p { color: #94a3b8; line-height: 1.6; margin-bottom: 16px; }
  .btn { display: inline-block; background: linear-gradient(135deg, #6366f1, #8b5cf6); color: #fff; 
         text-decoration: none; padding: 14px 32px; border-radius: 8px; font-weight: 600; font-size: 16px; }
  .footer { margin-top: 32px; padding-top: 24px; border-top: 1px solid #2d2d44; color: #475569; font-size: 14px; }
</style></head>
<body>
  <div class="container">
    <div class="logo">⚡ FlowBoard</div>
    <h1>You're invited!</h1>
    <p><strong>%s</strong> has invited you to join the <strong>%s</strong> workspace on FlowBoard.</p>
    <p>FlowBoard is an AI-powered workflow platform for teams. Click below to accept your invitation and get started.</p>
    <a href="%s" class="btn">Accept Invitation →</a>
    <div class="footer">
      <p>This invitation expires in %d hours. If you didn't expect this, you can ignore this email.</p>
    </div>
  </div>
</body>
</html>`, inviterName, workspaceName, inviteURL, h.cfg.InvitationExpiryHours)

	m := gomail.NewMessage()
	m.SetHeader("From", fmt.Sprintf("%s <%s>", h.cfg.SMTPFromName, h.cfg.SMTPFromEmail))
	m.SetHeader("To", toEmail)
	m.SetHeader("Subject", fmt.Sprintf("%s invited you to %s on FlowBoard", inviterName, workspaceName))
	m.SetBody("text/html", htmlBody)

	d := gomail.NewDialer(h.cfg.SMTPHost, h.cfg.SMTPPort, h.cfg.SMTPUser, h.cfg.SMTPPassword)
	if err := d.DialAndSend(m); err != nil {
		h.logger.Error("Failed to send invitation email", zap.Error(err), zap.String("to", toEmail))
	}
}
