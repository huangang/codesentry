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
	StuckTimeout   = 10 * time.Minute // Reviews stuck in pending/analyzing for more than this will be marked as failed
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

var retryStopChan chan struct{}

func StartRetryScheduler(db *gorm.DB, aiCfg *config.OpenAIConfig) {
	service := NewRetryService(db, aiCfg)
	ticker := time.NewTicker(RetryInterval)
	retryStopChan = make(chan struct{})

	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				service.ProcessStuckReviews() // Clean up stuck reviews first
				service.ProcessFailedReviews()
			case <-retryStopChan:
				log.Println("[Retry] Scheduler stopped")
				return
			}
		}
	}()

	log.Printf("[Retry] Scheduler started, interval: %v, max retries: %d, stuck timeout: %v", RetryInterval, MaxRetryCount, StuckTimeout)
}

func StopRetryScheduler() {
	if retryStopChan != nil {
		close(retryStopChan)
	}
}

// ProcessStuckReviews finds reviews stuck in pending/analyzing status for too long
// and marks them as failed so they can be retried
func (s *RetryService) ProcessStuckReviews() {
	cutoffTime := time.Now().Add(-StuckTimeout)

	var stuckReviews []models.ReviewLog
	err := s.db.Where("review_status IN (?, ?) AND updated_at < ?", "pending", "analyzing", cutoffTime).
		Order("created_at DESC").
		Limit(RetryBatchSize).
		Find(&stuckReviews).Error

	if err != nil {
		log.Printf("[Retry] Failed to fetch stuck reviews: %v", err)
		return
	}

	if len(stuckReviews) == 0 {
		return
	}

	log.Printf("[Retry] WARNING: Found %d stuck reviews (pending/analyzing > %v), marking as failed", len(stuckReviews), StuckTimeout)
	LogWarning("Retry", "StuckReviews", "Found stuck reviews to be marked as failed", nil, "", "", map[string]interface{}{
		"count":   len(stuckReviews),
		"timeout": StuckTimeout.String(),
	})

	for _, review := range stuckReviews {
		oldStatus := review.ReviewStatus
		review.ReviewStatus = "failed"
		review.ErrorMessage = "Review timeout: stuck in " + oldStatus + " status for more than " + StuckTimeout.String()

		if err := s.db.Save(&review).Error; err != nil {
			log.Printf("[Retry] Failed to update stuck review %d: %v", review.ID, err)
			continue
		}

		log.Printf("[Retry] Marked review %d as failed (was %s since %v)", review.ID, oldStatus, review.UpdatedAt)

		// Publish SSE event to notify frontend
		PublishReviewEvent(review.ID, review.ProjectID, review.CommitHash, "failed", nil, review.ErrorMessage)
	}
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
