package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
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
			reviewLogHandler := handlers.NewReviewLogHandler(models.GetDB())
			protected.GET("/review-logs", reviewLogHandler.List)
			protected.GET("/review-logs/:id", reviewLogHandler.GetByID)

			// LLM Configs
			llmConfigHandler := handlers.NewLLMConfigHandler(models.GetDB())
			protected.GET("/llm-configs", llmConfigHandler.List)
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
	}

	// Serve static files (embedded frontend)
	staticFS, err := fs.Sub(staticFiles, "static")
	if err == nil {
		r.NoRoute(func(c *gin.Context) {
			// Try to serve static file
			path := c.Request.URL.Path
			if path == "/" {
				path = "/index.html"
			}

			file, err := staticFS.Open(path[1:]) // Remove leading /
			if err != nil {
				// Fallback to index.html for SPA routing
				c.FileFromFS("index.html", http.FS(staticFS))
				return
			}
			file.Close()

			c.FileFromFS(path[1:], http.FS(staticFS))
		})
	}

	// Start server
	addr := cfg.Server.Host + ":" + cfg.Server.Port
	log.Printf("Server starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
