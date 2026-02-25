package services

import (
	"crypto/sha256"
	"fmt"

	"github.com/huangang/codesentry/backend/internal/models"
	"github.com/huangang/codesentry/backend/pkg/logger"
	"gorm.io/gorm"
)

// ReviewCacheService provides diff-hash-based review result dedup.
type ReviewCacheService struct {
	db *gorm.DB
}

func NewReviewCacheService(db *gorm.DB) *ReviewCacheService {
	return &ReviewCacheService{db: db}
}

// ComputeDiffHash returns the SHA-256 hex digest of the given diff string.
func ComputeDiffHash(diff string) string {
	h := sha256.Sum256([]byte(diff))
	return fmt.Sprintf("%x", h)
}

// CachedResult holds a cached review result.
type CachedResult struct {
	ReviewResult string
	Score        float64
	SourceID     uint // ID of the original review log
}

// FindCachedReview looks for a completed review in the same project with the same diff hash.
func (s *ReviewCacheService) FindCachedReview(projectID uint, diffHash string) *CachedResult {
	if diffHash == "" {
		return nil
	}

	var existing models.ReviewLog
	err := s.db.Where(
		"project_id = ? AND diff_hash = ? AND review_status = ? AND deleted_at IS NULL",
		projectID, diffHash, "completed",
	).Order("created_at DESC").First(&existing).Error

	if err != nil {
		return nil
	}

	score := 0.0
	if existing.Score != nil {
		score = *existing.Score
	}

	logger.Infof("[ReviewCache] Cache HIT: project=%d, hash=%s..., source_review=%d, score=%.0f",
		projectID, diffHash[:8], existing.ID, score)

	return &CachedResult{
		ReviewResult: existing.ReviewResult,
		Score:        score,
		SourceID:     existing.ID,
	}
}
