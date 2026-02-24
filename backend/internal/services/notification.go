package services

import (
	"fmt"
	"net/http"
	"time"

	"github.com/huangang/codesentry/backend/internal/models"
	"github.com/huangang/codesentry/backend/pkg/logger"
	"gorm.io/gorm"
)

type NotificationService struct {
	db           *gorm.DB
	emailService *EmailService
	httpClient   *http.Client
}

func NewNotificationService(db *gorm.DB) *NotificationService {
	return &NotificationService{
		db:           db,
		emailService: NewEmailService(db),
		httpClient:   &http.Client{Timeout: 10 * time.Second},
	}
}

type ReviewNotification struct {
	ProjectName   string
	Branch        string
	Author        string
	CommitMessage string
	Score         float64
	ReviewResult  string
	EventType     string
	MRURL         string
}

func (s *NotificationService) SendReviewNotification(project *models.Project, notification *ReviewNotification) error {
	var imErr, emailErr error

	if project.IMEnabled && project.IMBotID != nil {
		var bot models.IMBot
		if err := s.db.First(&bot, *project.IMBotID).Error; err != nil {
			imErr = fmt.Errorf("IM bot not found: %w", err)
		} else if !bot.IsActive {
			logger.Infof("[Notification] IM bot %d is not active", bot.ID)
		} else {
			logger.Infof("[Notification] Sending notification to bot %s (type: %s)", bot.Name, bot.Type)
			adapter := getAdapter(bot.Type)
			imErr = adapter.SendRichMessage(bot.Webhook, &bot, notification)
		}
	}

	if notification.Author != "" {
		var recipients []string
		var reviewLog models.ReviewLog
		s.db.Where("project_id = ? AND author = ?", project.ID, notification.Author).
			Order("created_at DESC").First(&reviewLog)
		if reviewLog.AuthorEmail != "" {
			recipients = append(recipients, reviewLog.AuthorEmail)
		}
		if len(recipients) > 0 {
			emailErr = s.emailService.SendReviewNotification(notification, recipients)
		}
	}

	if imErr != nil {
		logger.Infof("[Notification] IM notification failed: %v", imErr)
	}
	if emailErr != nil {
		logger.Infof("[Notification] Email notification failed: %v", emailErr)
	}

	if imErr != nil {
		return imErr
	}
	return emailErr
}

func (s *NotificationService) SendErrorNotification(bot *models.IMBot, message string) error {
	if !bot.IsActive {
		return nil
	}

	adapter := getAdapter(bot.Type)
	return adapter.SendTextMessage(bot.Webhook, bot, message)
}
