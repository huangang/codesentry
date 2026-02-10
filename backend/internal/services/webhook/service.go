package webhook

import (
	"context"
	"fmt"
	"github.com/huangang/codesentry/backend/pkg/logger"
	"net/http"
	"strings"
	"time"

	"github.com/huangang/codesentry/backend/internal/config"
	"github.com/huangang/codesentry/backend/internal/models"
	"github.com/huangang/codesentry/backend/internal/services"
	"gorm.io/gorm"
)

// Service handles webhook events from Git platforms
type Service struct {
	db                  *gorm.DB
	projectService      *services.ProjectService
	reviewService       *services.ReviewLogService
	aiService           *services.AIService
	notificationService *services.NotificationService
	configService       *services.SystemConfigService
	fileContextService  *services.FileContextService
	httpClient          *http.Client
}

// NewService creates a new webhook Service instance
func NewService(db *gorm.DB, aiCfg *config.OpenAIConfig) *Service {
	configService := services.NewSystemConfigService(db)
	return &Service{
		db:                  db,
		projectService:      services.NewProjectService(db),
		reviewService:       services.NewReviewLogService(db),
		aiService:           services.NewAIService(db, aiCfg),
		notificationService: services.NewNotificationService(db),
		configService:       configService,
		fileContextService:  services.NewFileContextService(configService),
		httpClient:          &http.Client{Timeout: 30 * time.Second},
	}
}

// GetReviewScore returns the review score for a given commit SHA
func (s *Service) GetReviewScore(commitSHA string) (*ReviewScoreResponse, error) {
	var reviewLog models.ReviewLog
	if err := s.db.Where("commit_hash = ?", commitSHA).Order("created_at DESC").First(&reviewLog).Error; err != nil {
		return nil, fmt.Errorf("review not found for commit: %s", commitSHA)
	}

	resp := &ReviewScoreResponse{
		CommitSHA: commitSHA,
		Status:    reviewLog.ReviewStatus,
		ReviewID:  reviewLog.ID,
	}

	switch reviewLog.ReviewStatus {
	case "pending", "processing", "analyzing":
		resp.Message = "Review in progress"
	case "completed":
		var project models.Project
		s.db.First(&project, reviewLog.ProjectID)
		minScore := s.getEffectiveMinScore(&project)
		passed := reviewLog.Score != nil && *reviewLog.Score >= minScore
		resp.Score = reviewLog.Score
		resp.MinScore = minScore
		resp.Passed = &passed
		resp.Message = "Review completed"
	case "skipped":
		passed := true
		resp.Passed = &passed
		resp.Message = "Skipped: " + reviewLog.ReviewResult
	case "failed":
		resp.Message = "Review failed: " + reviewLog.ErrorMessage
	}

	return resp, nil
}

// SyncReview performs a synchronous review for the given project and request
func (s *Service) SyncReview(ctx context.Context, project *models.Project, req *SyncReviewRequest) (*SyncReviewResponse, error) {
	minScore := s.getEffectiveMinScore(project)

	branch := strings.TrimPrefix(req.Ref, "refs/heads/")
	if s.isBranchIgnored(branch, project.BranchFilter) {
		return &SyncReviewResponse{
			Passed:   true,
			Score:    100,
			MinScore: minScore,
			Message:  "Branch is in ignore list, skipping review",
		}, nil
	}

	if s.isCommitAlreadyReviewed(project.ID, req.CommitSHA) {
		return &SyncReviewResponse{
			Passed:   true,
			Score:    100,
			MinScore: minScore,
			Message:  "Commit already reviewed and passed",
		}, nil
	}

	additions, deletions, filesChanged := ParseDiffStats(req.Diffs)

	reviewLog := &models.ReviewLog{
		ProjectID:     project.ID,
		EventType:     "push",
		Branch:        branch,
		CommitHash:    req.CommitSHA,
		Author:        req.Author,
		CommitMessage: req.Message,
		ReviewStatus:  "pending",
		Additions:     additions,
		Deletions:     deletions,
		FilesChanged:  filesChanged,
	}

	if err := s.reviewService.Create(reviewLog); err != nil {
		return nil, fmt.Errorf("failed to create review log: %w", err)
	}

	reviewLog.ReviewStatus = "processing"
	s.reviewService.Update(reviewLog)

	var fileContext string
	if s.fileContextService.IsEnabled() {
		fileContext, _ = s.fileContextService.BuildFileContext(project, req.Diffs, req.CommitSHA)
		if fileContext != "" {
			logger.Infof("[Webhook] Built file context for sync review: %d chars", len(fileContext))
		}
	}

	result, err := s.aiService.ReviewChunked(ctx, &services.ReviewRequest{
		ProjectID:   project.ID,
		Diffs:       req.Diffs,
		Commits:     req.Message,
		FileContext: fileContext,
	})

	if err != nil {
		reviewLog.ReviewStatus = "failed"
		reviewLog.ErrorMessage = err.Error()
		s.reviewService.Update(reviewLog)
		return nil, fmt.Errorf("AI review failed: %w", err)
	}

	reviewLog.ReviewStatus = "completed"
	reviewLog.ReviewResult = result.Content
	reviewLog.Score = &result.Score
	s.reviewService.Update(reviewLog)

	passed := result.Score >= minScore
	message := fmt.Sprintf("Score: %.0f/100 (min: %.0f)", result.Score, minScore)
	if !passed {
		message = fmt.Sprintf("Review failed: %.0f/100 (min: %.0f required)", result.Score, minScore)
	}

	return &SyncReviewResponse{
		Passed:      passed,
		Score:       result.Score,
		MinScore:    minScore,
		Message:     message,
		ReviewID:    reviewLog.ID,
		FullContent: result.Content,
	}, nil
}

// ProcessReviewTask processes a review task from the async queue
func (s *Service) ProcessReviewTask(ctx context.Context, task *services.ReviewTask) error {
	logger.Infof("[TaskQueue] Processing review task: review_log_id=%d, project=%d, commit=%s",
		task.ReviewLogID, task.ProjectID, task.CommitSHA)

	reviewLog, err := s.reviewService.GetByID(task.ReviewLogID)
	if err != nil {
		return fmt.Errorf("review log not found: %w", err)
	}

	project, err := s.projectService.GetByID(task.ProjectID)
	if err != nil {
		return fmt.Errorf("project not found: %w", err)
	}

	reviewLog.ReviewStatus = "analyzing"
	s.reviewService.Update(reviewLog)
	services.PublishReviewEvent(reviewLog.ID, reviewLog.ProjectID, reviewLog.CommitHash, "analyzing", nil, "")

	filteredDiff := s.filterDiff(task.Diff, project.FileExtensions, project.IgnorePatterns)

	if IsEmptyDiff(filteredDiff) {
		logger.Warnf("[TaskQueue] WARNING: Empty commit detected for review_log_id=%d - skipping AI review", task.ReviewLogID)
		services.LogWarning("TaskQueue", "EmptyCommit", fmt.Sprintf("Empty commit %s detected, skipping AI review", task.CommitSHA[:8]), nil, "", "", map[string]interface{}{
			"project_id":    task.ProjectID,
			"review_log_id": task.ReviewLogID,
			"commit":        task.CommitSHA,
		})
		reviewLog.ReviewStatus = "skipped"
		reviewLog.ReviewResult = "Empty commit - no code changes to review"
		s.reviewService.Update(reviewLog)
		services.PublishReviewEvent(reviewLog.ID, reviewLog.ProjectID, reviewLog.CommitHash, "skipped", nil, "Empty commit - no code changes")
		return nil
	}

	var fileContext string
	if s.fileContextService.IsEnabled() {
		fileContext, _ = s.fileContextService.BuildFileContext(project, filteredDiff, task.CommitSHA)
	}

	result, err := s.aiService.ReviewChunked(ctx, &services.ReviewRequest{
		ProjectID:   project.ID,
		Diffs:       filteredDiff,
		Commits:     task.CommitMessage,
		FileContext: fileContext,
	})

	if err != nil {
		logger.Infof("[TaskQueue] AI review failed: %v", err)
		reviewLog.ReviewStatus = "failed"
		reviewLog.ErrorMessage = err.Error()
		s.reviewService.Update(reviewLog)
		services.PublishReviewEvent(reviewLog.ID, reviewLog.ProjectID, reviewLog.CommitHash, "failed", nil, err.Error())
		return err
	}

	logger.Infof("[TaskQueue] AI review completed, score: %.1f", result.Score)
	reviewLog.ReviewStatus = "completed"
	reviewLog.ReviewResult = result.Content
	reviewLog.Score = &result.Score
	s.reviewService.Update(reviewLog)
	services.PublishReviewEvent(reviewLog.ID, reviewLog.ProjectID, reviewLog.CommitHash, "completed", &result.Score, "")

	s.notificationService.SendReviewNotification(project, &services.ReviewNotification{
		ProjectName:   project.Name,
		Branch:        task.Branch,
		Author:        task.Author,
		CommitMessage: task.CommitMessage,
		Score:         result.Score,
		ReviewResult:  result.Content,
		EventType:     task.EventType,
	})

	if project.CommentEnabled && task.CommitSHA != "" {
		comment := s.formatReviewComment(result.Score, result.Content)
		switch project.Platform {
		case "gitlab":
			if err := s.postGitLabCommitComment(project, task.CommitSHA, comment); err != nil {
				logger.Infof("[TaskQueue] Failed to post GitLab comment: %v", err)
			} else {
				reviewLog.CommentPosted = true
				s.reviewService.Update(reviewLog)
			}
		case "github":
			if err := s.postGitHubCommitComment(project, task.CommitSHA, comment); err != nil {
				logger.Infof("[TaskQueue] Failed to post GitHub comment: %v", err)
			} else {
				reviewLog.CommentPosted = true
				s.reviewService.Update(reviewLog)
			}
		}
	}

	minScore := s.getEffectiveMinScore(project)
	statusState := "success"
	statusDesc := fmt.Sprintf("AI Review Passed: %.0f/%.0f", result.Score, minScore)
	if result.Score < minScore {
		statusState = "failed"
		statusDesc = fmt.Sprintf("AI Review Failed: %.0f (Min: %.0f)", result.Score, minScore)
	}
	s.setCommitStatus(project, task.CommitSHA, statusState, statusDesc, task.GitLabProjectID)

	return nil
}
