package main

import (
	"github.com/huangang/codesentry/backend/internal/config"
	"github.com/huangang/codesentry/backend/internal/handlers"
	"github.com/huangang/codesentry/backend/internal/models"
	"github.com/huangang/codesentry/backend/internal/services"
	"github.com/huangang/codesentry/backend/internal/services/webhook"
	"github.com/huangang/codesentry/backend/internal/utils"
	"github.com/huangang/codesentry/backend/pkg/logger"
)

// appServices holds all initialized services and handlers needed by the application.
type appServices struct {
	openAICfg          *config.OpenAIConfig
	webhookService     *webhook.Service
	dailyReportService *services.DailyReportService
	taskQueue          services.TaskQueue
	worker             *services.Worker
	authHandler        *handlers.AuthHandler
	webhookHandler     *handlers.WebhookHandler
}

// bootstrap initializes all application dependencies: database, services, schedulers.
func bootstrap(cfg *config.Config) *appServices {
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

	return &appServices{
		openAICfg:          &cfg.OpenAI,
		webhookService:     webhookService,
		dailyReportService: dailyReportService,
		taskQueue:          taskQueue,
		worker:             worker,
		authHandler:        authHandler,
		webhookHandler:     handlers.NewWebhookHandler(models.GetDB(), &cfg.OpenAI),
	}
}

// shutdown gracefully stops all services.
func (s *appServices) shutdown() {
	s.dailyReportService.StopScheduler()
	services.StopLogCleanupScheduler()
	services.StopRetryScheduler()
	logger.Info().Msg("All schedulers stopped")

	if s.worker != nil {
		s.worker.Stop()
	}
	if s.taskQueue != nil {
		s.taskQueue.Close()
	}
}
