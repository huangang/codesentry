package main

import (
	"github.com/gin-gonic/gin"
	"github.com/huangang/codesentry/backend/internal/handlers"
	"github.com/huangang/codesentry/backend/internal/middleware"
	"github.com/huangang/codesentry/backend/internal/models"
	"github.com/huangang/codesentry/backend/internal/services"
	"github.com/huangang/codesentry/backend/pkg/logger"
)

// registerRoutes sets up all HTTP routes on the given Gin engine.
func registerRoutes(r *gin.Engine, svc *appServices) {
	// Middleware
	r.Use(logger.GinLogger(), logger.GinRecovery())
	r.RedirectTrailingSlash = false
	r.RedirectFixedPath = false
	r.Use(middleware.CORS())

	// Rate limiter for webhook routes
	webhookLimiter := middleware.NewRateLimiter(10, 20)

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "codesentry"})
	})

	// Root-level webhook routes (without /api prefix for compatibility)
	rootWebhook := r.Group("", webhookLimiter.Middleware())
	{
		rootWebhook.POST("/webhook", svc.webhookHandler.HandleUnifiedWebhook)
		rootWebhook.POST("/review/webhook", svc.webhookHandler.HandleUnifiedWebhook)
		rootWebhook.POST("/review/sync", svc.webhookHandler.HandleSyncReview)
		rootWebhook.GET("/review/score", svc.webhookHandler.GetReviewScore)
	}

	// API routes
	api := r.Group("/api")
	{
		// Auth routes (public)
		auth := api.Group("/auth")
		{
			auth.POST("/login", svc.authHandler.Login)
			auth.POST("/refresh", svc.authHandler.Refresh)
			auth.GET("/config", svc.authHandler.GetAuthConfig)
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
			protected.GET("/auth/me", svc.authHandler.GetCurrentUser)
			protected.POST("/auth/logout", svc.authHandler.Logout)
			protected.POST("/auth/change-password", svc.authHandler.ChangePassword)

			// Dashboard (all users)
			dashboardHandler := handlers.NewDashboardHandler(models.GetDB())
			protected.GET("/dashboard/stats", dashboardHandler.GetStats)

			// Projects (read for all users)
			projectHandler := handlers.NewProjectHandler(models.GetDB())
			protected.GET("/projects", projectHandler.List)
			protected.GET("/projects/default-prompt", projectHandler.GetDefaultPrompt)
			protected.GET("/projects/:id", projectHandler.GetByID)

			// Review Logs (read for all users)
			reviewLogHandler := handlers.NewReviewLogHandler(models.GetDB(), svc.openAICfg)
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
			reviewFeedbackHandler := handlers.NewReviewFeedbackHandler(models.GetDB(), svc.openAICfg)
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
			reviewLogHandler := handlers.NewReviewLogHandler(models.GetDB(), svc.openAICfg)
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
			admin.GET("/system-config/auth-session", systemConfigHandler.GetAuthSessionConfig)
			admin.PUT("/system-config/auth-session", systemConfigHandler.UpdateAuthSessionConfig)
			admin.GET("/system-config/daily-report", systemConfigHandler.GetDailyReportConfig)
			admin.PUT("/system-config/daily-report", systemConfigHandler.UpdateDailyReportConfig)
			admin.GET("/system-config/chunked-review", systemConfigHandler.GetChunkedReviewConfig)
			admin.PUT("/system-config/chunked-review", systemConfigHandler.UpdateChunkedReviewConfig)
			admin.GET("/system-config/file-context", systemConfigHandler.GetFileContextConfig)
			admin.PUT("/system-config/file-context", systemConfigHandler.UpdateFileContextConfig)
			admin.GET("/system-config/holiday-countries", systemConfigHandler.GetHolidayCountries)

			// Daily Reports
			dailyReportHandler := handlers.NewDailyReportHandler(svc.dailyReportService)
			admin.GET("/daily-reports", dailyReportHandler.List)
			admin.GET("/daily-reports/:id", dailyReportHandler.Get)
			admin.POST("/daily-reports/generate", dailyReportHandler.Generate)
			admin.POST("/daily-reports/:id/resend", dailyReportHandler.Resend)
		}

		// Webhook routes (public with signature verification, rate limited)
		apiWebhook := api.Group("", webhookLimiter.Middleware())
		{
			apiWebhook.POST("/webhook/gitlab/:project_id", svc.webhookHandler.HandleGitLabWebhook)
			apiWebhook.POST("/webhook/github/:project_id", svc.webhookHandler.HandleGitHubWebhook)
			apiWebhook.POST("/webhook/bitbucket/:project_id", svc.webhookHandler.HandleBitbucketWebhook)
			apiWebhook.POST("/webhook/gitlab", svc.webhookHandler.HandleGitLabWebhookGeneric)
			apiWebhook.POST("/webhook/github", svc.webhookHandler.HandleGitHubWebhookGeneric)
			apiWebhook.POST("/webhook/bitbucket", svc.webhookHandler.HandleBitbucketWebhookGeneric)
			apiWebhook.POST("/webhook", svc.webhookHandler.HandleUnifiedWebhook)
			apiWebhook.POST("/review/webhook", svc.webhookHandler.HandleUnifiedWebhook)
			apiWebhook.POST("/review/sync", svc.webhookHandler.HandleSyncReview)
			apiWebhook.GET("/review/score", svc.webhookHandler.GetReviewScore)
		}
	}
}
