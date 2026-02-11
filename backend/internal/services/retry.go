package services

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/huangang/codesentry/backend/pkg/logger"
	"io"
	"net/http"
	"strings"
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
	httpClient          *http.Client
}

func NewRetryService(db *gorm.DB, aiCfg *config.OpenAIConfig) *RetryService {
	return &RetryService{
		db:                  db,
		aiService:           NewAIService(db, aiCfg),
		notificationService: NewNotificationService(db),
		httpClient:          &http.Client{Timeout: 30 * time.Second},
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
				logger.Infof("[Retry] Scheduler stopped")
				return
			}
		}
	}()

	logger.Infof("[Retry] Scheduler started, interval: %v, max retries: %d, stuck timeout: %v", RetryInterval, MaxRetryCount, StuckTimeout)
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
		logger.Infof("[Retry] Failed to fetch stuck reviews: %v", err)
		return
	}

	if len(stuckReviews) == 0 {
		return
	}

	logger.Warnf("[Retry] WARNING: Found %d stuck reviews (pending/analyzing > %v), marking as failed", len(stuckReviews), StuckTimeout)
	LogWarning("Retry", "StuckReviews", "Found stuck reviews to be marked as failed", nil, "", "", map[string]interface{}{
		"count":   len(stuckReviews),
		"timeout": StuckTimeout.String(),
	})

	for _, review := range stuckReviews {
		oldStatus := review.ReviewStatus
		review.ReviewStatus = "failed"
		review.ErrorMessage = "Review timeout: stuck in " + oldStatus + " status for more than " + StuckTimeout.String()

		if err := s.db.Save(&review).Error; err != nil {
			logger.Infof("[Retry] Failed to update stuck review %d: %v", review.ID, err)
			continue
		}

		logger.Infof("[Retry] Marked review %d as failed (was %s since %v)", review.ID, oldStatus, review.UpdatedAt)

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
		logger.Infof("[Retry] Failed to fetch failed reviews: %v", err)
		return
	}

	if len(failedReviews) == 0 {
		return
	}

	logger.Infof("[Retry] Processing %d failed reviews", len(failedReviews))

	for _, review := range failedReviews {
		s.retryReview(&review)
	}
}

func (s *RetryService) retryReview(review *models.ReviewLog) {
	logger.Infof("[Retry] Retrying review ID %d (attempt %d/%d)", review.ID, review.RetryCount+1, MaxRetryCount)

	var project models.Project
	if err := s.db.First(&project, review.ProjectID).Error; err != nil {
		logger.Infof("[Retry] Project not found for review %d: %v", review.ID, err)
		return
	}

	review.RetryCount++

	diff, err := s.fetchCommitDiff(&project, review.CommitHash)
	if err != nil {
		logger.Infof("[Retry] Failed to re-fetch diff for review %d: %v", review.ID, err)
		review.ErrorMessage = fmt.Sprintf("Failed to re-fetch diff: %v", err)
		s.db.Save(review)
		return
	}

	if diff == "" {
		logger.Infof("[Retry] Empty diff for review %d (likely a merge commit), marking as skipped", review.ID)
		review.ReviewStatus = "skipped"
		review.ReviewResult = "Empty commit - no code changes to review (merge commit)"
		review.ErrorMessage = ""
		s.db.Save(review)
		PublishReviewEvent(review.ID, review.ProjectID, review.CommitHash, "skipped", nil, "Empty commit - merge commit with no direct changes")
		return
	}

	result, err := s.aiService.Review(context.Background(), &ReviewRequest{
		ProjectID: project.ID,
		Diffs:     diff,
		Commits:   review.CommitMessage,
	})

	if err != nil {
		logger.Infof("[Retry] Review %d failed again: %v", review.ID, err)
		review.ErrorMessage = err.Error()
		if review.RetryCount >= MaxRetryCount {
			logger.Infof("[Retry] Review %d exceeded max retries, marking as permanently failed", review.ID)
		}
	} else {
		logger.Infof("[Retry] Review %d succeeded on retry", review.ID)
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

func (s *RetryService) fetchCommitDiff(project *models.Project, commitSHA string) (string, error) {
	if project.URL == "" || project.AccessToken == "" {
		return "", fmt.Errorf("project URL or access token not configured")
	}

	switch project.Platform {
	case "gitlab":
		return s.fetchGitLabCommitDiff(project, commitSHA)
	case "github":
		return s.fetchGitHubCommitDiff(project, commitSHA)
	case "bitbucket":
		return s.fetchBitbucketCommitDiff(project, commitSHA)
	default:
		return "", fmt.Errorf("unsupported platform: %s", project.Platform)
	}
}

func (s *RetryService) fetchGitLabCommitDiff(project *models.Project, commitSHA string) (string, error) {
	urlStr := strings.TrimSuffix(project.URL, ".git")
	protocolIdx := strings.Index(urlStr, "://")
	if protocolIdx == -1 {
		return "", fmt.Errorf("invalid project URL: %s", project.URL)
	}
	rest := urlStr[protocolIdx+3:]
	slashIdx := strings.Index(rest, "/")
	if slashIdx == -1 {
		return "", fmt.Errorf("invalid project URL: %s", project.URL)
	}
	baseURL := urlStr[:protocolIdx+3+slashIdx]
	projectPath := rest[slashIdx+1:]

	apiURL := fmt.Sprintf("%s/api/v4/projects/%s/repository/commits/%s/diff",
		baseURL, strings.ReplaceAll(projectPath, "/", "%2F"), commitSHA)

	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Set("PRIVATE-TOKEN", project.AccessToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitLab API returned status %d", resp.StatusCode)
	}

	var diffs []struct {
		Diff    string `json:"diff"`
		OldPath string `json:"old_path"`
		NewPath string `json:"new_path"`
	}
	if err := json.Unmarshal(body, &diffs); err != nil {
		return string(body), nil
	}

	var result strings.Builder
	for _, d := range diffs {
		result.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", d.OldPath, d.NewPath))
		result.WriteString(fmt.Sprintf("--- a/%s\n+++ b/%s\n", d.OldPath, d.NewPath))
		result.WriteString(d.Diff)
		if !strings.HasSuffix(d.Diff, "\n") {
			result.WriteString("\n")
		}
	}

	return result.String(), nil
}

func (s *RetryService) fetchGitHubCommitDiff(project *models.Project, commitSHA string) (string, error) {
	urlStr := strings.TrimSuffix(project.URL, ".git")
	parts := strings.Split(urlStr, "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid project URL: %s", project.URL)
	}
	owner := parts[len(parts)-2]
	repo := parts[len(parts)-1]

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits/%s", owner, repo, commitSHA)

	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Set("Accept", "application/vnd.github.v3.diff")
	req.Header.Set("Authorization", "token "+project.AccessToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	return string(body), nil
}

func (s *RetryService) fetchBitbucketCommitDiff(project *models.Project, commitSHA string) (string, error) {
	urlStr := strings.TrimSuffix(project.URL, ".git")
	parts := strings.Split(urlStr, "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid project URL: %s", project.URL)
	}
	projectPath := parts[len(parts)-2] + "/" + parts[len(parts)-1]

	apiURL := fmt.Sprintf("https://api.bitbucket.org/2.0/repositories/%s/diff/%s", projectPath, commitSHA)

	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Set("Authorization", "Bearer "+project.AccessToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	return string(body), nil
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
