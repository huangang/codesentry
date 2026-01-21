package services

import (
	"time"

	"github.com/huangang/codesentry/backend/internal/models"
	"gorm.io/gorm"
)

type MemberService struct {
	db *gorm.DB
}

func NewMemberService(db *gorm.DB) *MemberService {
	return &MemberService{db: db}
}

type MemberListRequest struct {
	Page      int    `form:"page"`
	PageSize  int    `form:"page_size"`
	Name      string `form:"name"`
	ProjectID *uint  `form:"project_id"`
	StartDate string `form:"start_date"`
	EndDate   string `form:"end_date"`
	SortBy    string `form:"sort_by"`
	SortOrder string `form:"sort_order"`
}

type MemberStats struct {
	Author       string  `json:"author"`
	AuthorEmail  string  `json:"author_email"`
	CommitCount  int64   `json:"commit_count"`
	AvgScore     float64 `json:"avg_score"`
	MaxScore     float64 `json:"max_score"`
	MinScore     float64 `json:"min_score"`
	Additions    int64   `json:"additions"`
	Deletions    int64   `json:"deletions"`
	FilesChanged int64   `json:"files_changed"`
	ProjectCount int64   `json:"project_count"`
}

type MemberListResponse struct {
	Total    int64         `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"page_size"`
	Items    []MemberStats `json:"items"`
}

type MemberDetailRequest struct {
	Author    string `form:"author"`
	StartDate string `form:"start_date"`
	EndDate   string `form:"end_date"`
}

type MemberProjectStats struct {
	ProjectID   uint    `json:"project_id"`
	ProjectName string  `json:"project_name"`
	CommitCount int64   `json:"commit_count"`
	AvgScore    float64 `json:"avg_score"`
	Additions   int64   `json:"additions"`
	Deletions   int64   `json:"deletions"`
}

type MemberTrendItem struct {
	Date        string  `json:"date"`
	CommitCount int64   `json:"commit_count"`
	AvgScore    float64 `json:"avg_score"`
}

type MemberDetailResponse struct {
	Author       string               `json:"author"`
	AuthorEmail  string               `json:"author_email"`
	TotalStats   MemberStats          `json:"total_stats"`
	ProjectStats []MemberProjectStats `json:"project_stats"`
	Trend        []MemberTrendItem    `json:"trend"`
}

func (s *MemberService) List(req *MemberListRequest) (*MemberListResponse, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}

	var startDate, endDate time.Time
	var err error

	if req.StartDate != "" {
		startDate, err = time.Parse("2006-01-02", req.StartDate)
		if err != nil {
			startDate = time.Now().AddDate(0, 0, -30)
		}
	} else {
		startDate = time.Now().AddDate(0, 0, -30)
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

	var total int64
	countQuery := s.db.Model(&models.ReviewLog{}).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Where("author != ''")

	if req.Name != "" {
		countQuery = countQuery.Where("author LIKE ?", "%"+req.Name+"%")
	}
	if req.ProjectID != nil {
		countQuery = countQuery.Where("project_id = ?", *req.ProjectID)
	}

	countQuery.Distinct("author").Count(&total)

	var members []MemberStats

	query := s.db.Model(&models.ReviewLog{}).
		Select(`
			author,
			MAX(author_email) as author_email,
			COUNT(*) as commit_count,
			COALESCE(AVG(score), 0) as avg_score,
			COALESCE(MAX(score), 0) as max_score,
			COALESCE(MIN(NULLIF(score, 0)), 0) as min_score,
			COALESCE(SUM(additions), 0) as additions,
			COALESCE(SUM(deletions), 0) as deletions,
			COALESCE(SUM(files_changed), 0) as files_changed,
			COUNT(DISTINCT project_id) as project_count
		`).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Where("author != ''").
		Group("author")

	if req.Name != "" {
		query = query.Where("author LIKE ?", "%"+req.Name+"%")
	}
	if req.ProjectID != nil {
		query = query.Where("project_id = ?", *req.ProjectID)
	}

	sortBy := "commit_count"
	if req.SortBy != "" {
		sortBy = req.SortBy
	}
	sortOrder := "DESC"
	if req.SortOrder == "asc" {
		sortOrder = "ASC"
	}
	query = query.Order(sortBy + " " + sortOrder)

	offset := (req.Page - 1) * req.PageSize
	query.Offset(offset).Limit(req.PageSize).Scan(&members)

	return &MemberListResponse{
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
		Items:    members,
	}, nil
}

func (s *MemberService) GetDetail(req *MemberDetailRequest) (*MemberDetailResponse, error) {
	var startDate, endDate time.Time
	var err error

	if req.StartDate != "" {
		startDate, err = time.Parse("2006-01-02", req.StartDate)
		if err != nil {
			startDate = time.Now().AddDate(0, 0, -30)
		}
	} else {
		startDate = time.Now().AddDate(0, 0, -30)
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

	var totalStats MemberStats
	s.db.Model(&models.ReviewLog{}).
		Select(`
			author,
			MAX(author_email) as author_email,
			COUNT(*) as commit_count,
			COALESCE(AVG(score), 0) as avg_score,
			COALESCE(MAX(score), 0) as max_score,
			COALESCE(MIN(NULLIF(score, 0)), 0) as min_score,
			COALESCE(SUM(additions), 0) as additions,
			COALESCE(SUM(deletions), 0) as deletions,
			COALESCE(SUM(files_changed), 0) as files_changed,
			COUNT(DISTINCT project_id) as project_count
		`).
		Where("author = ?", req.Author).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Group("author").
		Scan(&totalStats)

	var projectStats []MemberProjectStats
	s.db.Model(&models.ReviewLog{}).
		Select(`
			project_id,
			COUNT(*) as commit_count,
			COALESCE(AVG(score), 0) as avg_score,
			COALESCE(SUM(additions), 0) as additions,
			COALESCE(SUM(deletions), 0) as deletions
		`).
		Where("author = ?", req.Author).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Group("project_id").
		Order("commit_count DESC").
		Scan(&projectStats)

	for i := range projectStats {
		var project models.Project
		if err := s.db.First(&project, projectStats[i].ProjectID).Error; err == nil {
			projectStats[i].ProjectName = project.Name
		}
	}

	var trend []MemberTrendItem
	s.db.Model(&models.ReviewLog{}).
		Select(`
			DATE(created_at) as date,
			COUNT(*) as commit_count,
			COALESCE(AVG(score), 0) as avg_score
		`).
		Where("author = ?", req.Author).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Group("DATE(created_at)").
		Order("date ASC").
		Scan(&trend)

	return &MemberDetailResponse{
		Author:       req.Author,
		AuthorEmail:  totalStats.AuthorEmail,
		TotalStats:   totalStats,
		ProjectStats: projectStats,
		Trend:        trend,
	}, nil
}
