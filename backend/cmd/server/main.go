package main

import (
	"embed"
	"io/fs"
	"log"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/huangang/codesentry/backend/internal/config"
	"github.com/huangang/codesentry/backend/internal/handlers"
	"github.com/huangang/codesentry/backend/internal/middleware"
	"github.com/huangang/codesentry/backend/internal/models"
	"github.com/huangang/codesentry/backend/internal/services"
	"github.com/huangang/codesentry/backend/internal/utils"
)

//go:embed static/*
var staticFiles embed.FS

func maskDSN(dsn string) string {
	if idx := strings.Index(dsn, "@"); idx > 0 {
		return "***@" + dsn[idx+1:]
	}
	return "***"
}

func main() {
	cfg, err := config.Load(os.Getenv("CONFIG_PATH"))
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Config loaded: driver=%s, dsn=%s", cfg.Database.Driver, maskDSN(cfg.Database.DSN))

	utils.SetJWTSecret(cfg.JWT.Secret)

	// Initialize database
	if err := models.InitDB(&cfg.Database); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto migrate database
	if err := models.AutoMigrate(); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// Seed default data
	if err := models.SeedDefaultData(); err != nil {
		log.Printf("Warning: Failed to seed default data: %v", err)
	}

	// Initialize system logger
	services.InitSystemLogger(models.GetDB())

	// Start system log cleanup scheduler
	services.StartLogCleanupScheduler(models.GetDB())

	// Start retry scheduler for failed reviews
	services.StartRetryScheduler(models.GetDB(), &cfg.OpenAI)

	// Create default admin user
	authHandler := handlers.NewAuthHandler(models.GetDB(), cfg)
	if err := authHandler.CreateAdminIfNotExists(); err != nil {
		log.Printf("Warning: Failed to create admin user: %v", err)
	}

	// Set Gin mode
	gin.SetMode(cfg.Server.Mode)

	// Create router
	r := gin.Default()

	// Disable redirect behaviors that cause issues with webhooks
	r.RedirectTrailingSlash = false
	r.RedirectFixedPath = false

	// Apply CORS middleware
	r.Use(middleware.CORS())

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "codesentry"})
	})

	// Root-level webhook routes (without /api prefix for compatibility)
	webhookHandler := handlers.NewWebhookHandler(models.GetDB(), &cfg.OpenAI)
	r.POST("/webhook", webhookHandler.HandleUnifiedWebhook)
	r.POST("/review/webhook", webhookHandler.HandleUnifiedWebhook)
	r.POST("/review/sync", webhookHandler.HandleSyncReview)
	r.GET("/review/score", webhookHandler.GetReviewScore)

	// API routes
	api := r.Group("/api")
	{
		// Auth routes (public)
		auth := api.Group("/auth")
		{
			auth.POST("/login", authHandler.Login)
			auth.GET("/config", authHandler.GetAuthConfig)
		}

		// Protected routes
		protected := api.Group("")
		protected.Use(middleware.AuthRequired())
		{
			// Auth
			protected.GET("/auth/me", authHandler.GetCurrentUser)
			protected.POST("/auth/logout", authHandler.Logout)
			protected.POST("/auth/change-password", authHandler.ChangePassword)

			// Dashboard (all users)
			dashboardHandler := handlers.NewDashboardHandler(models.GetDB())
			protected.GET("/dashboard/stats", dashboardHandler.GetStats)

			// Projects (read for all users)
			projectHandler := handlers.NewProjectHandler(models.GetDB())
			protected.GET("/projects", projectHandler.List)
			protected.GET("/projects/default-prompt", projectHandler.GetDefaultPrompt)
			protected.GET("/projects/:id", projectHandler.GetByID)

			// Review Logs (read for all users)
			reviewLogHandler := handlers.NewReviewLogHandler(models.GetDB(), &cfg.OpenAI)
			protected.GET("/review-logs", reviewLogHandler.List)
			protected.GET("/review-logs/:id", reviewLogHandler.GetByID)

			// Members (all users)
			memberHandler := handlers.NewMemberHandler(models.GetDB())
			protected.GET("/members", memberHandler.List)
			protected.GET("/members/detail", memberHandler.GetDetail)
		}

		// Admin only routes
		admin := api.Group("")
		admin.Use(middleware.AuthRequired(), middleware.AdminRequired())
		{
			// Projects (write operations)
			projectHandler := handlers.NewProjectHandler(models.GetDB())
			admin.POST("/projects", projectHandler.Create)
			admin.PUT("/projects/:id", projectHandler.Update)
			admin.DELETE("/projects/:id", projectHandler.Delete)

			// Review Logs (write operations)
			reviewLogHandler := handlers.NewReviewLogHandler(models.GetDB(), &cfg.OpenAI)
			admin.POST("/review-logs/:id/retry", reviewLogHandler.Retry)
			admin.DELETE("/review-logs/:id", reviewLogHandler.Delete)

			// Users
			userHandler := handlers.NewUserHandler(models.GetDB())
			admin.GET("/users", userHandler.List)
			admin.PUT("/users/:id", userHandler.Update)
			admin.DELETE("/users/:id", userHandler.Delete)

			// LLM Configs
			llmConfigHandler := handlers.NewLLMConfigHandler(models.GetDB())
			admin.GET("/llm-configs", llmConfigHandler.List)
			admin.GET("/llm-configs/active", llmConfigHandler.GetActive)
			admin.GET("/llm-configs/:id", llmConfigHandler.GetByID)
			admin.POST("/llm-configs", llmConfigHandler.Create)
			admin.PUT("/llm-configs/:id", llmConfigHandler.Update)
			admin.DELETE("/llm-configs/:id", llmConfigHandler.Delete)

			// IM Bots
			imBotHandler := handlers.NewIMBotHandler(models.GetDB())
			admin.GET("/im-bots", imBotHandler.List)
			admin.GET("/im-bots/active", imBotHandler.GetAllActive)
			admin.GET("/im-bots/:id", imBotHandler.GetByID)
			admin.POST("/im-bots", imBotHandler.Create)
			admin.PUT("/im-bots/:id", imBotHandler.Update)
			admin.DELETE("/im-bots/:id", imBotHandler.Delete)

			// Prompts
			promptHandler := handlers.NewPromptHandler(models.GetDB())
			admin.GET("/prompts", promptHandler.List)
			admin.GET("/prompts/default", promptHandler.GetDefault)
			admin.GET("/prompts/active", promptHandler.GetAllActive)
			admin.GET("/prompts/:id", promptHandler.GetByID)
			admin.POST("/prompts", promptHandler.Create)
			admin.PUT("/prompts/:id", promptHandler.Update)
			admin.DELETE("/prompts/:id", promptHandler.Delete)
			admin.POST("/prompts/:id/set-default", promptHandler.SetDefault)

			// System Logs
			systemLogHandler := handlers.NewSystemLogHandler(models.GetDB())
			admin.GET("/system-logs", systemLogHandler.List)
			admin.GET("/system-logs/modules", systemLogHandler.GetModules)
			admin.GET("/system-logs/retention", systemLogHandler.GetRetentionDays)
			admin.PUT("/system-logs/retention", systemLogHandler.SetRetentionDays)
			admin.POST("/system-logs/cleanup", systemLogHandler.Cleanup)

			// Git Credentials
			gitCredentialHandler := handlers.NewGitCredentialHandler(models.GetDB())
			admin.GET("/git-credentials", gitCredentialHandler.List)
			admin.GET("/git-credentials/active", gitCredentialHandler.GetActive)
			admin.GET("/git-credentials/:id", gitCredentialHandler.GetByID)
			admin.POST("/git-credentials", gitCredentialHandler.Create)
			admin.PUT("/git-credentials/:id", gitCredentialHandler.Update)
			admin.DELETE("/git-credentials/:id", gitCredentialHandler.Delete)

			// System Config
			systemConfigHandler := handlers.NewSystemConfigHandler(models.GetDB())
			admin.GET("/system-config/ldap", systemConfigHandler.GetLDAPConfig)
			admin.PUT("/system-config/ldap", systemConfigHandler.UpdateLDAPConfig)
		}

		// Webhook routes (public with signature verification)
		webhookHandler := handlers.NewWebhookHandler(models.GetDB(), &cfg.OpenAI)
		api.POST("/webhook/gitlab/:project_id", webhookHandler.HandleGitLabWebhook)
		api.POST("/webhook/github/:project_id", webhookHandler.HandleGitHubWebhook)
		api.POST("/webhook/gitlab", webhookHandler.HandleGitLabWebhookGeneric)
		api.POST("/webhook/github", webhookHandler.HandleGitHubWebhookGeneric)
		api.POST("/webhook", webhookHandler.HandleUnifiedWebhook)
		api.POST("/review/webhook", webhookHandler.HandleUnifiedWebhook)
		api.POST("/review/sync", webhookHandler.HandleSyncReview)
		api.GET("/review/score", webhookHandler.GetReviewScore)
	}

	// Serve static files (embedded frontend)
	staticFS, staticErr := fs.Sub(staticFiles, "static")
	if staticErr == nil {
		// Helper function to serve index.html
		serveIndex := func(c *gin.Context) {
			data, readErr := fs.ReadFile(staticFS, "index.html")
			if readErr != nil {
				c.String(404, "index.html not found")
				return
			}
			c.Data(200, "text/html; charset=utf-8", data)
		}

		// Serve index.html for root path
		r.GET("/", serveIndex)

		r.NoRoute(func(c *gin.Context) {
			// Try to serve static file
			path := c.Request.URL.Path[1:] // Remove leading /

			data, readErr := fs.ReadFile(staticFS, path)
			if readErr != nil {
				// Fallback to index.html for SPA routing
				serveIndex(c)
				return
			}

			// Determine content type
			contentType := "application/octet-stream"
			if len(path) > 3 {
				switch path[len(path)-3:] {
				case ".js":
					contentType = "application/javascript"
				case "css":
					contentType = "text/css"
				case "tml":
					contentType = "text/html"
				case "son":
					contentType = "application/json"
				case "svg":
					contentType = "image/svg+xml"
				case "png":
					contentType = "image/png"
				case "ico":
					contentType = "image/x-icon"
				}
			}
			c.Data(200, contentType, data)
		})
	}

	// Start server
	addr := cfg.Server.Host + ":" + cfg.Server.Port
	log.Printf("Server starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
