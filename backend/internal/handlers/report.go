package handlers

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/huangang/codesentry/backend/internal/models"
	"github.com/huangang/codesentry/backend/pkg/response"
	"gorm.io/gorm"
)

// ReportHandler provides report generation endpoints.
type ReportHandler struct {
	db *gorm.DB
}

func NewReportHandler(db *gorm.DB) *ReportHandler {
	return &ReportHandler{db: db}
}

type PeriodStats struct {
	Period        string  `json:"period"` // "weekly" or "monthly"
	StartDate     string  `json:"start_date"`
	EndDate       string  `json:"end_date"`
	TotalReviews  int64   `json:"total_reviews"`
	Completed     int64   `json:"completed"`
	Failed        int64   `json:"failed"`
	AvgScore      float64 `json:"avg_score"`
	TotalFiles    int64   `json:"total_files"`
	TotalAdds     int64   `json:"total_additions"`
	TotalDels     int64   `json:"total_deletions"`
	ActiveAuthors int64   `json:"active_authors"`
}

type TrendItem struct {
	Date      string  `json:"date"`
	Reviews   int64   `json:"reviews"`
	AvgScore  float64 `json:"avg_score"`
	Additions int64   `json:"additions"`
	Deletions int64   `json:"deletions"`
}

type AuthorRanking struct {
	Author      string  `json:"author"`
	ReviewCount int64   `json:"review_count"`
	AvgScore    float64 `json:"avg_score"`
	TotalAdds   int64   `json:"total_additions"`
	TotalDels   int64   `json:"total_deletions"`
}

type ReportResponse struct {
	Current  PeriodStats     `json:"current"`
	Previous PeriodStats     `json:"previous"`
	Trend    []TrendItem     `json:"trend"`
	Rankings []AuthorRanking `json:"rankings"`
}

// GetReport generates a report with current/previous period comparison and trends.
func (h *ReportHandler) GetReport(c *gin.Context) {
	period := c.DefaultQuery("period", "weekly") // weekly or monthly
	projectID := c.Query("project_id")

	now := time.Now()
	var currentStart, previousStart, previousEnd time.Time

	if period == "monthly" {
		currentStart = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		previousEnd = currentStart.Add(-1)
		previousStart = time.Date(previousEnd.Year(), previousEnd.Month(), 1, 0, 0, 0, 0, now.Location())
	} else {
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		currentStart = time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, now.Location())
		previousEnd = currentStart.Add(-1)
		previousStart = currentStart.AddDate(0, 0, -7)
	}

	current := h.getPeriodStats(period, currentStart, now, projectID)
	previous := h.getPeriodStats(period, previousStart, previousEnd, projectID)

	// Daily trend for last 14 days
	trend := h.getDailyTrend(now.AddDate(0, 0, -13), now, projectID)

	// Author rankings for current period
	rankings := h.getAuthorRankings(currentStart, now, projectID)

	response.Success(c, ReportResponse{
		Current:  current,
		Previous: previous,
		Trend:    trend,
		Rankings: rankings,
	})
}

func (h *ReportHandler) getPeriodStats(period string, start, end time.Time, projectID string) PeriodStats {
	stats := PeriodStats{
		Period:    period,
		StartDate: start.Format("2006-01-02"),
		EndDate:   end.Format("2006-01-02"),
	}

	query := h.db.Model(&models.ReviewLog{}).Where("created_at BETWEEN ? AND ?", start, end)
	if projectID != "" {
		query = query.Where("project_id = ?", projectID)
	}

	query.Count(&stats.TotalReviews)

	h.db.Model(&models.ReviewLog{}).Where("created_at BETWEEN ? AND ? AND review_status = 'completed'", start, end).
		Where(h.projectFilter(projectID)).Count(&stats.Completed)
	h.db.Model(&models.ReviewLog{}).Where("created_at BETWEEN ? AND ? AND review_status = 'failed'", start, end).
		Where(h.projectFilter(projectID)).Count(&stats.Failed)

	var avgScore *float64
	h.db.Model(&models.ReviewLog{}).Where("created_at BETWEEN ? AND ? AND score IS NOT NULL", start, end).
		Where(h.projectFilter(projectID)).Select("AVG(score)").Scan(&avgScore)
	if avgScore != nil {
		stats.AvgScore = *avgScore
	}

	var totalFiles, totalAdds, totalDels *int64
	baseQ := h.db.Model(&models.ReviewLog{}).Where("created_at BETWEEN ? AND ?", start, end)
	if projectID != "" {
		baseQ = baseQ.Where("project_id = ?", projectID)
	}
	baseQ.Select("SUM(files_changed)").Scan(&totalFiles)
	baseQ2 := h.db.Model(&models.ReviewLog{}).Where("created_at BETWEEN ? AND ?", start, end)
	if projectID != "" {
		baseQ2 = baseQ2.Where("project_id = ?", projectID)
	}
	baseQ2.Select("SUM(additions)").Scan(&totalAdds)
	baseQ3 := h.db.Model(&models.ReviewLog{}).Where("created_at BETWEEN ? AND ?", start, end)
	if projectID != "" {
		baseQ3 = baseQ3.Where("project_id = ?", projectID)
	}
	baseQ3.Select("SUM(deletions)").Scan(&totalDels)
	if totalFiles != nil {
		stats.TotalFiles = *totalFiles
	}
	if totalAdds != nil {
		stats.TotalAdds = *totalAdds
	}
	if totalDels != nil {
		stats.TotalDels = *totalDels
	}

	h.db.Model(&models.ReviewLog{}).Where("created_at BETWEEN ? AND ?", start, end).
		Where(h.projectFilter(projectID)).Distinct("author").Count(&stats.ActiveAuthors)

	return stats
}

func (h *ReportHandler) projectFilter(projectID string) string {
	if projectID != "" {
		return "project_id = " + projectID
	}
	return "1=1"
}

func (h *ReportHandler) getDailyTrend(start, end time.Time, projectID string) []TrendItem {
	var results []TrendItem
	query := h.db.Model(&models.ReviewLog{}).
		Select("DATE(created_at) as date, COUNT(*) as reviews, COALESCE(AVG(score),0) as avg_score, COALESCE(SUM(additions),0) as additions, COALESCE(SUM(deletions),0) as deletions").
		Where("created_at BETWEEN ? AND ?", start, end).
		Group("DATE(created_at)").Order("date ASC")

	if projectID != "" {
		query = query.Where("project_id = ?", projectID)
	}

	query.Scan(&results)
	return results
}

func (h *ReportHandler) getAuthorRankings(start, end time.Time, projectID string) []AuthorRanking {
	var results []AuthorRanking
	query := h.db.Model(&models.ReviewLog{}).
		Select("author, COUNT(*) as review_count, COALESCE(AVG(score),0) as avg_score, COALESCE(SUM(additions),0) as total_additions, COALESCE(SUM(deletions),0) as total_deletions").
		Where("created_at BETWEEN ? AND ?", start, end).
		Group("author").Order("review_count DESC").Limit(20)

	if projectID != "" {
		query = query.Where("project_id = ?", projectID)
	}

	query.Scan(&results)
	return results
}
