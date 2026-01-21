package handlers

import (
	"net/http"

	"github.com/huangang/codesentry/backend/internal/config"
	"github.com/huangang/codesentry/backend/internal/services"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AuthHandler struct {
	authService *services.AuthService
	ldapEnabled bool
}

func NewAuthHandler(db *gorm.DB, cfg *config.Config) *AuthHandler {
	authService := services.NewAuthService(db, &cfg.LDAP, &cfg.JWT)
	return &AuthHandler{
		authService: authService,
		ldapEnabled: cfg.LDAP.Enabled,
	}
}

// Login handles user login
// POST /api/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req services.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.authService.Login(&req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetCurrentUser returns the current logged-in user
// GET /api/auth/me
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	userID, _ := c.Get("user_id")
	user, err := h.authService.GetUserByID(userID.(uint))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// GetAuthConfig returns authentication configuration
// GET /api/auth/config
func (h *AuthHandler) GetAuthConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"ldap_enabled": h.ldapEnabled,
	})
}

// Logout handles user logout (client-side token removal)
// POST /api/auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "logged out successfully"})
}

// CreateAdminIfNotExists creates default admin user
func (h *AuthHandler) CreateAdminIfNotExists() error {
	return h.authService.CreateAdminIfNotExists()
}
