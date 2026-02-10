package main

import (
	"context"
	"embed"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/huangang/codesentry/backend/internal/config"
	"github.com/huangang/codesentry/backend/internal/handlers"
	"github.com/huangang/codesentry/backend/internal/middleware"
	"github.com/huangang/codesentry/backend/internal/models"
	"github.com/huangang/codesentry/backend/internal/services"
	"github.com/huangang/codesentry/backend/internal/services/webhook"
	"github.com/huangang/codesentry/backend/internal/utils"
	"github.com/huangang/codesentry/backend/pkg/logger"
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
		logger.Fatalf("Failed to load config: %v", err)
	}

	// Initialize structured logger
	logLevel := "info"
	if cfg.Server.Mode == "debug" {
		logLevel = "debug"
	}
	logger.Init(logLevel)

	logger.Info().Str("driver", cfg.Database.Driver).Str("dsn", maskDSN(cfg.Database.DSN)).Msg("Config loaded")

	utils.SetJWTSecret(cfg.JWT.Secret)

	// Initialize database
	if err := models.InitDB(&cfg.Database); err != nil {
		logger.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto migrate database
	if err := models.AutoMigrate(); err != nil {
		logger.Fatalf("Failed to migrate database: %v", err)
	}

	// Seed default data
	if err := models.SeedDefaultData(); err != nil {
		logger.Warn().Err(err).Msg("Failed to seed default data")
	}

	// Seed default review templates
	reviewTemplateHandler := handlers.NewReviewTemplateHandler(models.GetDB())
	if err := reviewTemplateHandler.SeedTemplates(); err != nil {
		logger.Warn().Err(err).Msg("Failed to seed review templates")
	}

	// Initialize system logger
	services.InitSystemLogger(models.GetDB())

	// Start system log cleanup scheduler
	services.StartLogCleanupScheduler(models.GetDB())

	// Start retry scheduler for failed reviews
	services.StartRetryScheduler(models.GetDB(), &cfg.OpenAI)

	// Initialize and start daily report scheduler
	aiService := services.NewAIService(models.GetDB(), &cfg.OpenAI)
	notificationService := services.NewNotificationService(models.GetDB())
	dailyReportService := services.NewDailyReportService(models.GetDB(), aiService, notificationService)
	dailyReportService.StartScheduler()

	// Initialize task queue (uses Redis if enabled, otherwise sync mode)
	webhookService := webhook.NewService(models.GetDB(), &cfg.OpenAI)
	taskQueue := services.InitTaskQueue(cfg)
	if syncQueue, ok := taskQueue.(*services.SyncQueue); ok {
		syncQueue.SetProcessor(webhookService.ProcessReviewTask)
	}

	// Start async worker if Redis is enabled
	var worker *services.Worker
	if cfg.Redis.Enabled {
		worker = services.InitWorker(&cfg.Redis)
		if worker != nil {
			worker.SetProcessor(webhookService.ProcessReviewTask)
			worker.Start()
		}
	}

	// Create default admin user
	authHandler := handlers.NewAuthHandler(models.GetDB(), cfg)
	if err := authHandler.CreateAdminIfNotExists(); err != nil {
		logger.Warn().Err(err).Msg("Failed to create admin user")
	}

	// Set Gin mode
	gin.SetMode(cfg.Server.Mode)

	// Create router with structured logging middleware
	r := gin.New()
	r.Use(logger.GinLogger(), logger.GinRecovery())

	// Disable redirect behaviors that cause issues with webhooks
	r.RedirectTrailingSlash = false
	r.RedirectFixedPath = false

	// Apply CORS middleware
	r.Use(middleware.CORS())

	// Rate limiter for webhook routes
	webhookLimiter := middleware.NewRateLimiter(10, 20)

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "codesentry"})
	})

	// Root-level webhook routes (without /api prefix for compatibility)
	webhookHandler := handlers.NewWebhookHandler(models.GetDB(), &cfg.OpenAI)
	rootWebhook := r.Group("", webhookLimiter.Middleware())
	{
		rootWebhook.POST("/webhook", webhookHandler.HandleUnifiedWebhook)
		rootWebhook.POST("/review/webhook", webhookHandler.HandleUnifiedWebhook)
		rootWebhook.POST("/review/sync", webhookHandler.HandleSyncReview)
		rootWebhook.GET("/review/score", webhookHandler.GetReviewScore)
	}

	// API routes
	api := r.Group("/api")
	{
		// Auth routes (public)
		auth := api.Group("/auth")
		{
			auth.POST("/login", authHandler.Login)
			auth.GET("/config", authHandler.GetAuthConfig)
		}

		// SSE Events (public route with internal token validation)
		sseHandler := handlers.NewSSEHandler(services.GetSSEHub())
		api.GET("/events/reviews", sseHandler.StreamReviewEvents)
		api.GET("/events/imports", sseHandler.StreamImportEvents)

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
			protected.GET("/members/overview", memberHandler.GetTeamOverview)
			protected.GET("/members/heatmap", memberHandler.GetHeatmap)

			// Prompts (read for all users)
			promptHandler := handlers.NewPromptHandler(models.GetDB())
			protected.GET("/prompts", promptHandler.List)
			protected.GET("/prompts/default", promptHandler.GetDefault)
			protected.GET("/prompts/active", promptHandler.GetAllActive)
			protected.GET("/prompts/:id", promptHandler.GetByID)

			// Review Templates (read for all users)
			reviewTemplateHandler := handlers.NewReviewTemplateHandler(models.GetDB())
			protected.GET("/review-templates", reviewTemplateHandler.List)
			protected.GET("/review-templates/:id", reviewTemplateHandler.Get)

			// Review Feedbacks (interactive AI feedback)
			reviewFeedbackHandler := handlers.NewReviewFeedbackHandler(models.GetDB(), &cfg.OpenAI)
			protected.GET("/review-logs/:id/feedbacks", reviewFeedbackHandler.ListByReview)
			protected.GET("/review-feedbacks/:id", reviewFeedbackHandler.Get)
			protected.POST("/review-feedbacks", reviewFeedbackHandler.Create)
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
			admin.POST("/review-logs/manual", reviewLogHandler.CreateManualCommit)
			admin.POST("/review-logs/import", reviewLogHandler.ImportCommits)
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
			admin.POST("/prompts", promptHandler.Create)
			admin.PUT("/prompts/:id", promptHandler.Update)
			admin.DELETE("/prompts/:id", promptHandler.Delete)
			admin.POST("/prompts/:id/set-default", promptHandler.SetDefault)

			// Review Templates (admin only for write operations)
			reviewTemplateHandler := handlers.NewReviewTemplateHandler(models.GetDB())
			admin.POST("/review-templates", reviewTemplateHandler.Create)
			admin.PUT("/review-templates/:id", reviewTemplateHandler.Update)
			admin.DELETE("/review-templates/:id", reviewTemplateHandler.Delete)

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
			admin.GET("/system-config/daily-report", systemConfigHandler.GetDailyReportConfig)
			admin.PUT("/system-config/daily-report", systemConfigHandler.UpdateDailyReportConfig)
			admin.GET("/system-config/chunked-review", systemConfigHandler.GetChunkedReviewConfig)
			admin.PUT("/system-config/chunked-review", systemConfigHandler.UpdateChunkedReviewConfig)
			admin.GET("/system-config/file-context", systemConfigHandler.GetFileContextConfig)
			admin.PUT("/system-config/file-context", systemConfigHandler.UpdateFileContextConfig)
			admin.GET("/system-config/holiday-countries", systemConfigHandler.GetHolidayCountries)

			// Daily Reports
			dailyReportHandler := handlers.NewDailyReportHandler(dailyReportService)
			admin.GET("/daily-reports", dailyReportHandler.List)
			admin.GET("/daily-reports/:id", dailyReportHandler.Get)
			admin.POST("/daily-reports/generate", dailyReportHandler.Generate)
			admin.POST("/daily-reports/:id/resend", dailyReportHandler.Resend)
		}

		// Webhook routes (public with signature verification, rate limited)
		apiWebhook := api.Group("", webhookLimiter.Middleware())
		{
			apiWebhook.POST("/webhook/gitlab/:project_id", webhookHandler.HandleGitLabWebhook)
			apiWebhook.POST("/webhook/github/:project_id", webhookHandler.HandleGitHubWebhook)
			apiWebhook.POST("/webhook/bitbucket/:project_id", webhookHandler.HandleBitbucketWebhook)
			apiWebhook.POST("/webhook/gitlab", webhookHandler.HandleGitLabWebhookGeneric)
			apiWebhook.POST("/webhook/github", webhookHandler.HandleGitHubWebhookGeneric)
			apiWebhook.POST("/webhook/bitbucket", webhookHandler.HandleBitbucketWebhookGeneric)
			apiWebhook.POST("/webhook", webhookHandler.HandleUnifiedWebhook)
			apiWebhook.POST("/review/webhook", webhookHandler.HandleUnifiedWebhook)
			apiWebhook.POST("/review/sync", webhookHandler.HandleSyncReview)
			apiWebhook.GET("/review/score", webhookHandler.GetReviewScore)
		}
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

			// Determine content type using standard library
			ext := filepath.Ext(path)
			contentType := mime.TypeByExtension(ext)
			if contentType == "" {
				contentType = "application/octet-stream"
			}
			c.Data(200, contentType, data)
		})
	}

	// Start server with graceful shutdown
	addr := cfg.Server.Host + ":" + cfg.Server.Port
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	go func() {
		logger.Info().Str("addr", addr).Msg("Server starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info().Msg("Shutting down server...")

	// Stop all schedulers
	dailyReportService.StopScheduler()
	services.StopLogCleanupScheduler()
	services.StopRetryScheduler()
	logger.Info().Msg("All schedulers stopped")

	// Stop async worker if running
	if worker != nil {
		worker.Stop()
	}
	// Close task queue
	if taskQueue != nil {
		taskQueue.Close()
	}

	// Graceful shutdown with 30 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatalf("Server forced to shutdown: %v", err)
	}

	// Close database connection
	if sqlDB, err := models.GetDB().DB(); err == nil {
		sqlDB.Close()
		logger.Info().Msg("Database connection closed")
	}

	logger.Info().Msg("Server exited gracefully")
}
