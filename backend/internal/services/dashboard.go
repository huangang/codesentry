package services

import (
	"time"

	"github.com/huangang/codesentry/backend/internal/models"
	"gorm.io/gorm"
)

type DashboardService struct {
	db *gorm.DB
}

func NewDashboardService(db *gorm.DB) *DashboardService {
	return &DashboardService{db: db}
}

type DashboardStatsRequest struct {
	StartDate string `form:"start_date"`
	EndDate   string `form:"end_date"`
}

type DashboardStats struct {
	ActiveProjects int64   `json:"active_projects"`
	Contributors   int64   `json:"contributors"`
	TotalCommits   int64   `json:"total_commits"`
	AverageScore   float64 `json:"average_score"`
}

type ProjectStats struct {
	ProjectID   uint    `json:"project_id"`
	ProjectName string  `json:"project_name"`
	CommitCount int64   `json:"commit_count"`
	AvgScore    float64 `json:"avg_score"`
	Additions   int64   `json:"additions"`
	Deletions   int64   `json:"deletions"`
}

type AuthorStats struct {
	Author      string  `json:"author"`
	CommitCount int64   `json:"commit_count"`
	AvgScore    float64 `json:"avg_score"`
	Additions   int64   `json:"additions"`
	Deletions   int64   `json:"deletions"`
}

type DashboardResponse struct {
	Stats        DashboardStats `json:"stats"`
	ProjectStats []ProjectStats `json:"project_stats"`
	AuthorStats  []AuthorStats  `json:"author_stats"`
}

func (s *DashboardService) GetStats(req *DashboardStatsRequest) (*DashboardResponse, error) {
	var startDate, endDate time.Time
	var err error

	if req.StartDate != "" {
		startDate, err = time.Parse("2006-01-02", req.StartDate)
		if err != nil {
			startDate = time.Now().AddDate(0, 0, -7)
		}
	} else {
		startDate = time.Now().AddDate(0, 0, -7)
	}

	if req.EndDate != "" {
		endDate, err = time.Parse("2006-01-02", req.EndDate)
		if err != nil {
			endDate = time.Now()
		}
		endDate = endDate.Add(24*time.Hour - time.Second)
	} else {
		endDate = time.Now()
	}

	var stats DashboardStats

	s.db.Model(&models.ReviewLog{}).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Distinct("project_id").
		Count(&stats.ActiveProjects)

	s.db.Model(&models.ReviewLog{}).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Distinct("author").
		Count(&stats.Contributors)

	s.db.Model(&models.ReviewLog{}).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Count(&stats.TotalCommits)

	s.db.Model(&models.ReviewLog{}).
		Where("created_at BETWEEN ? AND ? AND score IS NOT NULL", startDate, endDate).
		Select("COALESCE(AVG(score), 0)").
		Scan(&stats.AverageScore)

	var projectStats []ProjectStats
	s.db.Model(&models.ReviewLog{}).
		Select("project_id, COUNT(*) as commit_count, COALESCE(AVG(score), 0) as avg_score, COALESCE(SUM(additions), 0) as additions, COALESCE(SUM(deletions), 0) as deletions").
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Group("project_id").
		Order("commit_count DESC").
		Limit(10).
		Scan(&projectStats)

	for i := range projectStats {
		var project models.Project
		if err := s.db.First(&project, projectStats[i].ProjectID).Error; err == nil {
			projectStats[i].ProjectName = project.Name
		}
	}

	var authorStats []AuthorStats
	s.db.Model(&models.ReviewLog{}).
		Select("author, COUNT(*) as commit_count, COALESCE(AVG(score), 0) as avg_score, COALESCE(SUM(additions), 0) as additions, COALESCE(SUM(deletions), 0) as deletions").
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Group("author").
		Order("commit_count DESC").
		Limit(10).
		Scan(&authorStats)

	return &DashboardResponse{
		Stats:        stats,
		ProjectStats: projectStats,
		AuthorStats:  authorStats,
	}, nil
}
