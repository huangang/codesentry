package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/huangang/codesentry/backend/internal/config"
	"github.com/huangang/codesentry/backend/internal/services"
	"github.com/huangang/codesentry/backend/pkg/response"
	"gorm.io/gorm"
)

const refreshTokenCookieName = "refresh_token"

type AuthHandler struct {
	authService *services.AuthService
}

func NewAuthHandler(db *gorm.DB, cfg *config.Config) *AuthHandler {
	authService := services.NewAuthService(db, &cfg.JWT)
	return &AuthHandler{
		authService: authService,
	}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req services.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	result, err := h.authService.Login(&req, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		services.LogWarning("Auth", "LoginFailed", "Login failed: "+err.Error(), nil, c.ClientIP(), c.Request.UserAgent(), map[string]string{"username": req.Username})
		response.Unauthorized(c, err.Error())
		return
	}

	h.setRefreshCookie(c, result.RefreshToken, result.RefreshExpireAt)

	services.LogInfo("Auth", "LoginSuccess", "User logged in: "+req.Username, &result.User.ID, c.ClientIP(), c.Request.UserAgent(), nil)
	response.Success(c, gin.H{
		"token":     result.AccessToken,
		"user":      result.User,
		"expire_at": result.AccessExpireAt,
	})
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	refreshToken, err := c.Cookie(refreshTokenCookieName)
	if err != nil || refreshToken == "" {
		response.Unauthorized(c, "refresh token required")
		return
	}

	result, err := h.authService.Refresh(refreshToken, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	h.setRefreshCookie(c, result.RefreshToken, result.RefreshExpireAt)
	response.Success(c, gin.H{
		"token":     result.AccessToken,
		"expire_at": result.AccessExpireAt,
	})
}

func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	userID, _ := c.Get("user_id")
	user, err := h.authService.GetUserByID(userID.(uint))
	if err != nil {
		response.NotFound(c, "user not found")
		return
	}

	response.Success(c, user)
}

func (h *AuthHandler) GetAuthConfig(c *gin.Context) {
	response.Success(c, gin.H{
		"ldap_enabled": h.authService.IsLDAPEnabled(),
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	refreshToken, _ := c.Cookie(refreshTokenCookieName)
	_ = h.authService.RevokeRefreshToken(refreshToken)
	h.clearRefreshCookie(c)

	userID, exists := c.Get("user_id")
	if exists {
		uid := userID.(uint)
		services.LogInfo("Auth", "Logout", "User logged out", &uid, c.ClientIP(), c.Request.UserAgent(), nil)
	}
	response.Success(c, gin.H{"message": "logged out successfully"})
}

func (h *AuthHandler) setRefreshCookie(c *gin.Context, token string, expiresAt time.Time) {
	maxAge := int(time.Until(expiresAt).Seconds())
	if maxAge < 0 {
		maxAge = 0
	}

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(
		refreshTokenCookieName,
		token,
		maxAge,
		"/api/auth",
		"",
		c.Request.TLS != nil,
		true,
	)
}

func (h *AuthHandler) clearRefreshCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(refreshTokenCookieName, "", -1, "/api/auth", "", c.Request.TLS != nil, true)
}

func (h *AuthHandler) CreateAdminIfNotExists() error {
	return h.authService.CreateAdminIfNotExists()
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req services.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.authService.ChangePassword(userID.(uint), &req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	uid := userID.(uint)
	services.LogInfo("Auth", "ChangePassword", "User changed password", &uid, c.ClientIP(), c.Request.UserAgent(), nil)
	response.Success(c, gin.H{"message": "password changed successfully"})
}
