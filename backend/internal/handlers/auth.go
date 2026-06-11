package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"gorm.io/gorm"

	"github.com/kanban-platform/backend/internal/config"
	"github.com/kanban-platform/backend/internal/models"
)

type AuthHandler struct {
	db          *gorm.DB
	cfg         *config.Config
	logger      *zap.Logger
	oauthConfig *oauth2.Config
}

type GoogleUser struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	VerifiedEmail bool   `json:"verified_email"`
}

type JWTClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func NewAuthHandler(db *gorm.DB, cfg *config.Config, logger *zap.Logger) *AuthHandler {
	oauthConfig := &oauth2.Config{
		ClientID:     cfg.GoogleClientID,
		ClientSecret: cfg.GoogleClientSecret,
		RedirectURL:  cfg.GoogleRedirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}

	return &AuthHandler{
		db:          db,
		cfg:         cfg,
		logger:      logger,
		oauthConfig: oauthConfig,
	}
}

// GoogleLogin redirects to Google OAuth login
func (h *AuthHandler) GoogleLogin(c *fiber.Ctx) error {
	state := generateRandomState()
	// Store state in cookie for CSRF protection
	c.Cookie(&fiber.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Expires:  time.Now().Add(10 * time.Minute),
		HTTPOnly: true,
		Secure:   h.cfg.AppEnv == "production",
		SameSite: "Lax",
	})

	url := h.oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
	return c.Redirect(url)
}

// GoogleCallback handles the OAuth2 callback
func (h *AuthHandler) GoogleCallback(c *fiber.Ctx) error {
	// Validate state for CSRF protection
	state := c.Query("state")
	cookieState := c.Cookies("oauth_state")
	if state == "" || state != cookieState {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid OAuth state",
		})
	}

	code := c.Query("code")
	if code == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Authorization code not provided",
		})
	}

	// Exchange code for token
	token, err := h.oauthConfig.Exchange(context.Background(), code)
	if err != nil {
		h.logger.Error("Failed to exchange OAuth code", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to exchange OAuth token",
		})
	}

	// Get Google user info
	googleUser, err := h.getGoogleUser(token)
	if err != nil {
		h.logger.Error("Failed to get Google user", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get user information",
		})
	}

	// Find or create user
	user, err := h.findOrCreateUser(googleUser, c.IP())
	if err != nil {
		h.logger.Error("Failed to find/create user", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to authenticate user",
		})
	}

	// Check if user is suspended
	if user.Status == models.StatusSuspended {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Your account has been suspended",
		})
	}

	// Generate JWT tokens
	accessToken, refreshToken, err := h.generateTokens(user)
	if err != nil {
		h.logger.Error("Failed to generate tokens", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate authentication tokens",
		})
	}

	// Store session in DB
	session := &models.UserSession{
		UserID:       user.ID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		IPAddress:    c.IP(),
		UserAgent:    string(c.Request().Header.UserAgent()),
		ExpiresAt:    time.Now().Add(time.Duration(h.cfg.RefreshTokenExpiryDays) * 24 * time.Hour),
	}
	h.db.Create(session)

	// Redirect to frontend with token
	frontendURL := fmt.Sprintf("%s/auth/callback?token=%s&refresh=%s",
		h.cfg.FrontendURL, accessToken, refreshToken)
	return c.Redirect(frontendURL)
}

// BypassLogin handles demo/direct authentication for local use
func (h *AuthHandler) BypassLogin(c *fiber.Ctx) error {
	type BypassRequest struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}

	var req BypassRequest
	if err := c.BodyParser(&req); err != nil {
		// Use query params as fallback
		req.Email = c.Query("email")
		req.Name = c.Query("name")
	}

	if req.Email == "" {
		req.Email = "demo@example.com"
	}
	if req.Name == "" {
		req.Name = "Demo User"
	}

	googleUser := &GoogleUser{
		ID:            "bypass-" + req.Email,
		Email:         req.Email,
		Name:          req.Name,
		Picture:       "https://api.dicebear.com/7.x/bottts/svg?seed=" + req.Name,
		VerifiedEmail: true,
	}

	user, err := h.findOrCreateUser(googleUser, c.IP())
	if err != nil {
		h.logger.Error("Bypass login failed to find/create user", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to authenticate demo user",
		})
	}

	accessToken, refreshToken, err := h.generateTokens(user)
	if err != nil {
		h.logger.Error("Failed to generate tokens", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate authentication tokens",
		})
	}

	session := &models.UserSession{
		UserID:       user.ID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		IPAddress:    c.IP(),
		UserAgent:    string(c.Request().Header.UserAgent()),
		ExpiresAt:    time.Now().Add(time.Duration(h.cfg.RefreshTokenExpiryDays) * 24 * time.Hour),
	}
	h.db.Create(session)

	return c.JSON(fiber.Map{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"user":          user,
	})
}

// RefreshToken issues new access token using refresh token
func (h *AuthHandler) RefreshToken(c *fiber.Ctx) error {
	type RefreshRequest struct {
		RefreshToken string `json:"refresh_token"`
	}

	var req RefreshRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	// Find session
	var session models.UserSession
	if err := h.db.Preload("User").Where("refresh_token = ? AND is_active = true", req.RefreshToken).First(&session).Error; err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid refresh token"})
	}

	if time.Now().After(session.ExpiresAt) {
		h.db.Model(&session).Update("is_active", false)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Refresh token expired"})
	}

	// Generate new access token
	accessToken, _, err := h.generateTokens(&session.User)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to refresh token"})
	}

	h.db.Model(&session).Update("access_token", accessToken)

	return c.JSON(fiber.Map{
		"access_token": accessToken,
		"expires_in":   h.cfg.JWTExpiryHours * 3600,
	})
}

// Logout invalidates the session
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	h.db.Where("user_id = ? AND is_active = true", userID).Updates(map[string]interface{}{
		"is_active": false,
	})

	return c.JSON(fiber.Map{"message": "Logged out successfully"})
}

// GetMe returns the current user
func (h *AuthHandler) GetMe(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	var user models.User
	if err := h.db.First(&user, "id = ?", userID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}

	return c.JSON(user)
}

// UpdateMe updates current user profile
func (h *AuthHandler) UpdateMe(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	type UpdateRequest struct {
		Name        string `json:"name"`
		Theme       string `json:"theme"`
		Timezone    string `json:"timezone"`
		NotifyEmail bool   `json:"notify_email"`
		NotifyInApp bool   `json:"notify_in_app"`
	}

	var req UpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	var user models.User
	if err := h.db.First(&user, "id = ?", userID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}

	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Theme != "" {
		updates["theme"] = req.Theme
	}
	if req.Timezone != "" {
		updates["timezone"] = req.Timezone
	}
	updates["notify_email"] = req.NotifyEmail
	updates["notify_in_app"] = req.NotifyInApp

	h.db.Model(&user).Updates(updates)

	return c.JSON(user)
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func (h *AuthHandler) getGoogleUser(token *oauth2.Token) (*GoogleUser, error) {
	client := h.oauthConfig.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get user info: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var googleUser GoogleUser
	if err := json.Unmarshal(body, &googleUser); err != nil {
		return nil, err
	}

	return &googleUser, nil
}

func (h *AuthHandler) findOrCreateUser(googleUser *GoogleUser, ip string) (*models.User, error) {
	var user models.User

	// Try to find by Google ID first
	result := h.db.Where("google_id = ?", googleUser.ID).First(&user)
	if result.Error == nil {
		// Update last login
		now := time.Now()
		h.db.Model(&user).Updates(map[string]interface{}{
			"last_login_at": now,
			"login_ip":      ip,
			"login_count":   user.LoginCount + 1,
			"avatar":        googleUser.Picture,
		})
		return &user, nil
	}

	// Try by email
	result = h.db.Where("email = ?", googleUser.Email).First(&user)
	if result.Error == nil {
		// Link Google ID
		now := time.Now()
		h.db.Model(&user).Updates(map[string]interface{}{
			"google_id":     googleUser.ID,
			"last_login_at": now,
			"login_ip":      ip,
			"login_count":   user.LoginCount + 1,
			"avatar":        googleUser.Picture,
		})
		return &user, nil
	}

	// Create new user
	now := time.Now()
	newUser := models.User{
		Base:        models.Base{ID: uuid.New()},
		Name:        googleUser.Name,
		Email:       googleUser.Email,
		GoogleID:    googleUser.ID,
		Avatar:      googleUser.Picture,
		Status:      models.StatusActive,
		LastLoginAt: &now,
		LoginIP:     ip,
		LoginCount:  1,
	}

	if err := h.db.Create(&newUser).Error; err != nil {
		return nil, err
	}

	// Create default personal workspace for new user
	go h.createDefaultWorkspace(&newUser)

	return &newUser, nil
}

func (h *AuthHandler) createDefaultWorkspace(user *models.User) {
	workspace := models.Workspace{
		Base:    models.Base{ID: uuid.New()},
		Name:    fmt.Sprintf("%s's Workspace", strings.Split(user.Name, " ")[0]),
		Slug:    fmt.Sprintf("%s-%s", strings.ToLower(strings.ReplaceAll(user.Name, " ", "-")), uuid.New().String()[:8]),
		Type:    models.WorkspacePersonal,
		OwnerID: user.ID,
		Color:   "#6366f1",
	}
	h.db.Create(&workspace)

	// Add user as owner member
	h.db.Create(&models.WorkspaceMember{
		Base:        models.Base{ID: uuid.New()},
		WorkspaceID: workspace.ID,
		UserID:      user.ID,
		Role:        models.RoleOwner,
		JoinedAt:    time.Now(),
	})
}

func (h *AuthHandler) generateTokens(user *models.User) (string, string, error) {
	// Access token
	accessClaims := JWTClaims{
		UserID: user.ID.String(),
		Email:  user.Email,
		Role:   string(user.Role),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(h.cfg.JWTExpiryHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   user.ID.String(),
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	signedAccess, err := accessToken.SignedString([]byte(h.cfg.JWTSecret))
	if err != nil {
		return "", "", err
	}

	// Refresh token (longer-lived, random)
	refreshBytes := make([]byte, 32)
	rand.Read(refreshBytes)
	refreshToken := base64.URLEncoding.EncodeToString(refreshBytes)

	return signedAccess, refreshToken, nil
}

func generateRandomState() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}
