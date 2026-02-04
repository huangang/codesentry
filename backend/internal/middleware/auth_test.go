package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/huangang/codesentry/backend/internal/utils"
)

func init() {
	gin.SetMode(gin.TestMode)
	utils.SetJWTSecret("test-secret-for-middleware-testing")
}

func TestAuthRequired_NoHeader(t *testing.T) {
	router := gin.New()
	router.Use(AuthRequired())
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuthRequired_InvalidFormat(t *testing.T) {
	router := gin.New()
	router.Use(AuthRequired())
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	testCases := []string{
		"InvalidToken",
		"Basic token123",
		"Bearer",
	}

	for _, authHeader := range testCases {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", authHeader)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("header %q: expected status %d, got %d", authHeader, http.StatusUnauthorized, w.Code)
		}
	}
}

func TestAuthRequired_InvalidToken(t *testing.T) {
	router := gin.New()
	router.Use(AuthRequired())
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid.jwt.token")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuthRequired_ValidToken(t *testing.T) {
	token, _ := utils.GenerateToken(1, "testuser", "admin", 24)

	router := gin.New()
	router.Use(AuthRequired())
	router.GET("/protected", func(c *gin.Context) {
		userID, _ := c.Get(ContextUserID)
		username, _ := c.Get(ContextUsername)
		role, _ := c.Get(ContextRole)
		c.JSON(200, gin.H{
			"user_id":  userID,
			"username": username,
			"role":     role,
		})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestAdminRequired_NoRole(t *testing.T) {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Next()
	})
	router.Use(AdminRequired())
	router.GET("/admin", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/admin", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

func TestAdminRequired_UserRole(t *testing.T) {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(ContextRole, "user")
		c.Next()
	})
	router.Use(AdminRequired())
	router.GET("/admin", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/admin", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

func TestAdminRequired_AdminRole(t *testing.T) {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(ContextRole, "admin")
		c.Next()
	})
	router.Use(AdminRequired())
	router.GET("/admin", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/admin", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestGetUserID(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	if id := GetUserID(c); id != 0 {
		t.Errorf("expected 0 for missing user_id, got %d", id)
	}

	c.Set(ContextUserID, uint(42))
	if id := GetUserID(c); id != 42 {
		t.Errorf("expected 42, got %d", id)
	}
}

func TestGetUsername(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	if name := GetUsername(c); name != "" {
		t.Errorf("expected empty string for missing username, got %q", name)
	}

	c.Set(ContextUsername, "testuser")
	if name := GetUsername(c); name != "testuser" {
		t.Errorf("expected %q, got %q", "testuser", name)
	}
}

func TestGetRole(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	if role := GetRole(c); role != "" {
		t.Errorf("expected empty string for missing role, got %q", role)
	}

	c.Set(ContextRole, "admin")
	if role := GetRole(c); role != "admin" {
		t.Errorf("expected %q, got %q", "admin", role)
	}
}

func TestContextConstants(t *testing.T) {
	if ContextUserID != "user_id" {
		t.Errorf("ContextUserID = %q, expected %q", ContextUserID, "user_id")
	}
	if ContextUsername != "username" {
		t.Errorf("ContextUsername = %q, expected %q", ContextUsername, "username")
	}
	if ContextRole != "role" {
		t.Errorf("ContextRole = %q, expected %q", ContextRole, "role")
	}
}
