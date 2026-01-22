package main

import (
	"embed"
	"io/fs"
	"log"
	"os"

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

func main() {
	// Load configuration
	cfg, err := config.Load(os.Getenv("CONFIG_PATH"))
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize JWT secret
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

			// Dashboard
			dashboardHandler := handlers.NewDashboardHandler(models.GetDB())
			protected.GET("/dashboard/stats", dashboardHandler.GetStats)

			// Projects
			projectHandler := handlers.NewProjectHandler(models.GetDB())
			protected.GET("/projects", projectHandler.List)
			protected.GET("/projects/default-prompt", projectHandler.GetDefaultPrompt)
			protected.GET("/projects/:id", projectHandler.GetByID)
			protected.POST("/projects", projectHandler.Create)
			protected.PUT("/projects/:id", projectHandler.Update)
			protected.DELETE("/projects/:id", projectHandler.Delete)

			// Review Logs
			reviewLogHandler := handlers.NewReviewLogHandler(models.GetDB(), &cfg.OpenAI)
			protected.GET("/review-logs", reviewLogHandler.List)
			protected.GET("/review-logs/:id", reviewLogHandler.GetByID)
			protected.POST("/review-logs/:id/retry", reviewLogHandler.Retry)

			// LLM Configs
			llmConfigHandler := handlers.NewLLMConfigHandler(models.GetDB())
			protected.GET("/llm-configs", llmConfigHandler.List)
			protected.GET("/llm-configs/active", llmConfigHandler.GetActive)
			protected.GET("/llm-configs/:id", llmConfigHandler.GetByID)
			protected.POST("/llm-configs", llmConfigHandler.Create)
			protected.PUT("/llm-configs/:id", llmConfigHandler.Update)
			protected.DELETE("/llm-configs/:id", llmConfigHandler.Delete)

			// IM Bots
			imBotHandler := handlers.NewIMBotHandler(models.GetDB())
			protected.GET("/im-bots", imBotHandler.List)
			protected.GET("/im-bots/active", imBotHandler.GetAllActive)
			protected.GET("/im-bots/:id", imBotHandler.GetByID)
			protected.POST("/im-bots", imBotHandler.Create)
			protected.PUT("/im-bots/:id", imBotHandler.Update)
			protected.DELETE("/im-bots/:id", imBotHandler.Delete)

			// Prompts
			promptHandler := handlers.NewPromptHandler(models.GetDB())
			protected.GET("/prompts", promptHandler.List)
			protected.GET("/prompts/default", promptHandler.GetDefault)
			protected.GET("/prompts/active", promptHandler.GetAllActive)
			protected.GET("/prompts/:id", promptHandler.GetByID)
			protected.POST("/prompts", promptHandler.Create)
			protected.PUT("/prompts/:id", promptHandler.Update)
			protected.DELETE("/prompts/:id", promptHandler.Delete)
			protected.POST("/prompts/:id/set-default", promptHandler.SetDefault)

			// Members
			memberHandler := handlers.NewMemberHandler(models.GetDB())
			protected.GET("/members", memberHandler.List)
			protected.GET("/members/detail", memberHandler.GetDetail)

			// System Logs
			systemLogHandler := handlers.NewSystemLogHandler(models.GetDB())
			protected.GET("/system-logs", systemLogHandler.List)
			protected.GET("/system-logs/modules", systemLogHandler.GetModules)
			protected.GET("/system-logs/retention", systemLogHandler.GetRetentionDays)
			protected.PUT("/system-logs/retention", systemLogHandler.SetRetentionDays)
			protected.POST("/system-logs/cleanup", systemLogHandler.Cleanup)
		}

		// Webhook routes (public with signature verification)
		webhookHandler := handlers.NewWebhookHandler(models.GetDB(), &cfg.OpenAI)
		api.POST("/webhook/gitlab/:project_id", webhookHandler.HandleGitLabWebhook)
		api.POST("/webhook/github/:project_id", webhookHandler.HandleGitHubWebhook)
		api.POST("/webhook/gitlab", webhookHandler.HandleGitLabWebhookGeneric)
		api.POST("/webhook/github", webhookHandler.HandleGitHubWebhookGeneric)
		api.POST("/webhook", webhookHandler.HandleUnifiedWebhook)
		api.POST("/review/webhook", webhookHandler.HandleUnifiedWebhook)
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
