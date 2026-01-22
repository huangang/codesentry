package services

import (
	"context"
	"log"
	"time"

	"github.com/huangang/codesentry/backend/internal/config"
	"github.com/huangang/codesentry/backend/internal/models"
	"gorm.io/gorm"
)

const (
	MaxRetryCount  = 3
	RetryInterval  = 5 * time.Minute
	RetryBatchSize = 10
)

type RetryService struct {
	db                  *gorm.DB
	aiService           *AIService
	notificationService *NotificationService
}

func NewRetryService(db *gorm.DB, aiCfg *config.OpenAIConfig) *RetryService {
	return &RetryService{
		db:                  db,
		aiService:           NewAIService(db, aiCfg),
		notificationService: NewNotificationService(db),
	}
}

func StartRetryScheduler(db *gorm.DB, aiCfg *config.OpenAIConfig) {
	service := NewRetryService(db, aiCfg)
	ticker := time.NewTicker(RetryInterval)

	go func() {
		for range ticker.C {
			service.ProcessFailedReviews()
		}
	}()

	log.Printf("[Retry] Scheduler started, interval: %v, max retries: %d", RetryInterval, MaxRetryCount)
}

func (s *RetryService) ProcessFailedReviews() {
	var failedReviews []models.ReviewLog

	err := s.db.Where("review_status = ? AND retry_count < ?", "failed", MaxRetryCount).
		Order("created_at DESC").
		Limit(RetryBatchSize).
		Find(&failedReviews).Error

	if err != nil {
		log.Printf("[Retry] Failed to fetch failed reviews: %v", err)
		return
	}

	if len(failedReviews) == 0 {
		return
	}

	log.Printf("[Retry] Processing %d failed reviews", len(failedReviews))

	for _, review := range failedReviews {
		s.retryReview(&review)
	}
}

func (s *RetryService) retryReview(review *models.ReviewLog) {
	log.Printf("[Retry] Retrying review ID %d (attempt %d/%d)", review.ID, review.RetryCount+1, MaxRetryCount)

	var project models.Project
	if err := s.db.First(&project, review.ProjectID).Error; err != nil {
		log.Printf("[Retry] Project not found for review %d: %v", review.ID, err)
		return
	}

	review.RetryCount++

	result, err := s.aiService.Review(context.Background(), &ReviewRequest{
		ProjectID: project.ID,
		Diffs:     "Retry - diff not available",
		Commits:   review.CommitMessage,
	})

	if err != nil {
		log.Printf("[Retry] Review %d failed again: %v", review.ID, err)
		review.ErrorMessage = err.Error()
		if review.RetryCount >= MaxRetryCount {
			log.Printf("[Retry] Review %d exceeded max retries, marking as permanently failed", review.ID)
		}
	} else {
		log.Printf("[Retry] Review %d succeeded on retry", review.ID)
		review.ReviewStatus = "completed"
		review.ReviewResult = result.Content
		review.Score = &result.Score
		review.ErrorMessage = ""

		s.notificationService.SendReviewNotification(&project, &ReviewNotification{
			ProjectName:   project.Name,
			Branch:        review.Branch,
			Author:        review.Author,
			CommitMessage: review.CommitMessage,
			Score:         result.Score,
			ReviewResult:  result.Content,
			EventType:     review.EventType,
			MRURL:         review.MRURL,
		})
	}

	s.db.Save(review)
}

func (s *RetryService) ManualRetry(reviewID uint) error {
	var review models.ReviewLog
	if err := s.db.First(&review, reviewID).Error; err != nil {
		return err
	}

	if review.ReviewStatus != "failed" {
		return nil
	}

	review.RetryCount = 0
	s.retryReview(&review)
	return nil
}
