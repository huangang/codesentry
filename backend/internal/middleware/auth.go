package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/huangang/codesentry/backend/internal/utils"
	"github.com/huangang/codesentry/backend/pkg/response"
)

const (
	ContextUserID   = "user_id"
	ContextUsername = "username"
	ContextRole     = "role"
)

// AuthRequired is a middleware that checks for a valid JWT token
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(c, "authorization header required")
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.Unauthorized(c, "invalid authorization header format")
			c.Abort()
			return
		}

		tokenString := parts[1]
		claims, err := utils.ParseToken(tokenString)
		if err != nil {
			response.Unauthorized(c, "invalid or expired token")
			c.Abort()
			return
		}

		// Set user info in context
		c.Set(ContextUserID, claims.UserID)
		c.Set(ContextUsername, claims.Username)
		c.Set(ContextRole, claims.Role)

		c.Next()
	}
}

// AdminRequired is a middleware that checks for admin role
func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get(ContextRole)
		if !exists || role != "admin" {
			response.Forbidden(c, "admin access required")
			c.Abort()
			return
		}
		c.Next()
	}
}

// RoleRequired is a middleware that checks if the user has one of the allowed roles.
func RoleRequired(allowedRoles ...string) gin.HandlerFunc {
	roleSet := make(map[string]bool, len(allowedRoles))
	for _, r := range allowedRoles {
		roleSet[r] = true
	}
	return func(c *gin.Context) {
		role, exists := c.Get(ContextRole)
		if !exists {
			response.Forbidden(c, "access denied")
			c.Abort()
			return
		}
		roleStr, ok := role.(string)
		if !ok || !roleSet[roleStr] {
			response.Forbidden(c, "insufficient permissions")
			c.Abort()
			return
		}
		c.Next()
	}
}

// GetUserID gets the current user ID from context
func GetUserID(c *gin.Context) uint {
	if id, exists := c.Get(ContextUserID); exists {
		return id.(uint)
	}
	return 0
}

// GetUsername gets the current username from context
func GetUsername(c *gin.Context) string {
	if username, exists := c.Get(ContextUsername); exists {
		return username.(string)
	}
	return ""
}

// GetRole gets the current user role from context
func GetRole(c *gin.Context) string {
	if role, exists := c.Get(ContextRole); exists {
		return role.(string)
	}
	return ""
}
