package services

import (
	"time"

	"github.com/huangang/codesentry/backend/internal/models"
	"github.com/huangang/codesentry/backend/pkg/logger"
	"gorm.io/gorm"
)

// AIUsageService manages AI usage tracking and statistics.
type AIUsageService struct {
	db *gorm.DB
}

func NewAIUsageService(db *gorm.DB) *AIUsageService {
	return &AIUsageService{db: db}
}

// Record saves a usage log entry asynchronously.
func (s *AIUsageService) Record(log *models.AIUsageLog) {
	go func() {
		if err := s.db.Create(log).Error; err != nil {
			logger.Infof("[AIUsage] Failed to record usage: %v", err)
		}
	}()
}

// UsageStats holds aggregated AI usage statistics.
type UsageStats struct {
	TotalCalls       int64   `json:"total_calls"`
	TotalTokens      int64   `json:"total_tokens"`
	PromptTokens     int64   `json:"prompt_tokens"`
	CompletionTokens int64   `json:"completion_tokens"`
	AvgLatencyMs     float64 `json:"avg_latency_ms"`
	SuccessRate      float64 `json:"success_rate"`
	SuccessCount     int64   `json:"success_count"`
	FailureCount     int64   `json:"failure_count"`
}

// GetStats returns aggregated usage statistics for the given time range.
func (s *AIUsageService) GetStats(startDate, endDate string, projectID *uint) (*UsageStats, error) {
	query := s.db.Model(&models.AIUsageLog{})
	if startDate != "" {
		query = query.Where("created_at >= ?", startDate)
	}
	if endDate != "" {
		query = query.Where("created_at <= ?", endDate+" 23:59:59")
	}
	if projectID != nil && *projectID > 0 {
		query = query.Where("project_id = ?", *projectID)
	}

	var stats UsageStats
	err := query.Select(
		"COUNT(*) as total_calls, " +
			"COALESCE(SUM(total_tokens), 0) as total_tokens, " +
			"COALESCE(SUM(prompt_tokens), 0) as prompt_tokens, " +
			"COALESCE(SUM(completion_tokens), 0) as completion_tokens, " +
			"COALESCE(AVG(latency_ms), 0) as avg_latency_ms, " +
			"COALESCE(SUM(CASE WHEN success = 1 THEN 1 ELSE 0 END), 0) as success_count, " +
			"COALESCE(SUM(CASE WHEN success = 0 THEN 1 ELSE 0 END), 0) as failure_count",
	).Scan(&stats).Error
	if err != nil {
		return nil, err
	}

	if stats.TotalCalls > 0 {
		stats.SuccessRate = float64(stats.SuccessCount) / float64(stats.TotalCalls) * 100
	}
	return &stats, nil
}

// DailyUsage holds usage data for a single day.
type DailyUsage struct {
	Date         string `json:"date"`
	Calls        int    `json:"calls"`
	TotalTokens  int    `json:"total_tokens"`
	AvgLatencyMs int    `json:"avg_latency_ms"`
}

// GetDailyTrend returns daily aggregated usage for charting.
func (s *AIUsageService) GetDailyTrend(startDate, endDate string, projectID *uint) ([]DailyUsage, error) {
	query := s.db.Model(&models.AIUsageLog{})
	if startDate != "" {
		query = query.Where("created_at >= ?", startDate)
	}
	if endDate != "" {
		query = query.Where("created_at <= ?", endDate+" 23:59:59")
	}
	if projectID != nil && *projectID > 0 {
		query = query.Where("project_id = ?", *projectID)
	}

	var results []DailyUsage
	err := query.Select(
		"DATE(created_at) as date, " +
			"COUNT(*) as calls, " +
			"COALESCE(SUM(total_tokens), 0) as total_tokens, " +
			"COALESCE(AVG(latency_ms), 0) as avg_latency_ms",
	).Group("DATE(created_at)").Order("date ASC").Scan(&results).Error
	if err != nil {
		return nil, err
	}

	if results == nil {
		results = []DailyUsage{}
	}
	return results, nil
}

// ProviderUsage holds usage data grouped by provider.
type ProviderUsage struct {
	Provider     string  `json:"provider"`
	Model        string  `json:"model"`
	Calls        int     `json:"calls"`
	TotalTokens  int     `json:"total_tokens"`
	AvgLatencyMs float64 `json:"avg_latency_ms"`
	SuccessRate  float64 `json:"success_rate"`
}

// GetProviderBreakdown returns usage grouped by provider and model.
func (s *AIUsageService) GetProviderBreakdown(startDate, endDate string) ([]ProviderUsage, error) {
	query := s.db.Model(&models.AIUsageLog{})
	if startDate != "" {
		query = query.Where("created_at >= ?", startDate)
	}
	if endDate != "" {
		query = query.Where("created_at <= ?", endDate+" 23:59:59")
	}

	var results []ProviderUsage
	err := query.Select(
		"provider, model, " +
			"COUNT(*) as calls, " +
			"COALESCE(SUM(total_tokens), 0) as total_tokens, " +
			"COALESCE(AVG(latency_ms), 0) as avg_latency_ms, " +
			"COALESCE(AVG(CASE WHEN success = 1 THEN 100.0 ELSE 0.0 END), 0) as success_rate",
	).Group("provider, model").Order("calls DESC").Scan(&results).Error
	if err != nil {
		return nil, err
	}

	if results == nil {
		results = []ProviderUsage{}
	}
	return results, nil
}

// CleanupBefore deletes usage logs older than the given time.
func (s *AIUsageService) CleanupBefore(before time.Time) (int64, error) {
	result := s.db.Where("created_at < ?", before).Delete(&models.AIUsageLog{})
	return result.RowsAffected, result.Error
}
