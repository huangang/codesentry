package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/huangang/codesentry/backend/internal/config"
	"github.com/huangang/codesentry/backend/internal/services"
	"github.com/huangang/codesentry/backend/pkg/response"
	"gorm.io/gorm"
)

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

	resp, err := h.authService.Login(&req)
	if err != nil {
		services.LogWarning("Auth", "LoginFailed", "Login failed: "+err.Error(), nil, c.ClientIP(), c.Request.UserAgent(), map[string]string{"username": req.Username})
		response.Unauthorized(c, err.Error())
		return
	}

	services.LogInfo("Auth", "LoginSuccess", "User logged in: "+req.Username, &resp.User.ID, c.ClientIP(), c.Request.UserAgent(), nil)
	response.Success(c, resp)
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
	userID, exists := c.Get("user_id")
	if exists {
		uid := userID.(uint)
		services.LogInfo("Auth", "Logout", "User logged out", &uid, c.ClientIP(), c.Request.UserAgent(), nil)
	}
	response.Success(c, gin.H{"message": "logged out successfully"})
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
