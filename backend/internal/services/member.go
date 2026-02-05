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
			COALESCE(AVG(CASE WHEN is_manual = false THEN score END), 0) as avg_score,
			COALESCE(MAX(CASE WHEN is_manual = false THEN score END), 0) as max_score,
			COALESCE(MIN(NULLIF(CASE WHEN is_manual = false THEN score END, 0)), 0) as min_score,
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
			COALESCE(AVG(CASE WHEN is_manual = false THEN score END), 0) as avg_score,
			COALESCE(MAX(CASE WHEN is_manual = false THEN score END), 0) as max_score,
			COALESCE(MIN(NULLIF(CASE WHEN is_manual = false THEN score END, 0)), 0) as min_score,
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
			COALESCE(AVG(CASE WHEN is_manual = false THEN score END), 0) as avg_score,
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
			COALESCE(AVG(CASE WHEN is_manual = false THEN score END), 0) as avg_score
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

// Team Overview types
type TeamOverviewRequest struct {
	StartDate string `form:"start_date"`
	EndDate   string `form:"end_date"`
	ProjectID *uint  `form:"project_id"`
}

type TeamOverviewResponse struct {
	TotalMembers   int64             `json:"total_members"`
	TotalCommits   int64             `json:"total_commits"`
	AvgScore       float64           `json:"avg_score"`
	TotalAdditions int64             `json:"total_additions"`
	TotalDeletions int64             `json:"total_deletions"`
	Trend          []MemberTrendItem `json:"trend"`
	TopMembers     []MemberStats     `json:"top_members"`
	ScoreDistrib   ScoreDistribution `json:"score_distribution"`
}

type ScoreDistribution struct {
	Excellent int64 `json:"excellent"` // >= 80
	Good      int64 `json:"good"`      // 60-79
	NeedWork  int64 `json:"need_work"` // < 60
}

type HeatmapRequest struct {
	StartDate string `form:"start_date"`
	EndDate   string `form:"end_date"`
	ProjectID *uint  `form:"project_id"`
	Author    string `form:"author"`
}

type HeatmapDataPoint struct {
	Date       string `json:"date"`
	Count      int64  `json:"count"`
	Additions  int64  `json:"additions"`
	Deletions  int64  `json:"deletions"`
	WeekDay    int    `json:"week_day"`
	WeekOfYear int    `json:"week_of_year"`
}

type HeatmapResponse struct {
	Data       []HeatmapDataPoint `json:"data"`
	TotalCount int64              `json:"total_count"`
	MaxCount   int64              `json:"max_count"`
	StartDate  string             `json:"start_date"`
	EndDate    string             `json:"end_date"`
}

func (s *MemberService) GetHeatmap(req *HeatmapRequest) (*HeatmapResponse, error) {
	var startDate, endDate time.Time
	var err error

	if req.StartDate != "" {
		startDate, err = time.Parse("2006-01-02", req.StartDate)
		if err != nil {
			startDate = time.Now().AddDate(-1, 0, 0)
		}
	} else {
		startDate = time.Now().AddDate(-1, 0, 0)
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

	query := s.db.Model(&models.ReviewLog{}).
		Select(`
			DATE(created_at) as date,
			COUNT(*) as count,
			COALESCE(SUM(additions), 0) as additions,
			COALESCE(SUM(deletions), 0) as deletions
		`).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Where("author != ''").
		Group("DATE(created_at)").
		Order("date ASC")

	if req.ProjectID != nil {
		query = query.Where("project_id = ?", *req.ProjectID)
	}
	if req.Author != "" {
		query = query.Where("author = ?", req.Author)
	}

	var rawData []struct {
		Date      time.Time
		Count     int64
		Additions int64
		Deletions int64
	}
	query.Scan(&rawData)

	dataMap := make(map[string]HeatmapDataPoint)
	var totalCount, maxCount int64

	for _, d := range rawData {
		dateStr := d.Date.Format("2006-01-02")
		_, week := d.Date.ISOWeek()
		point := HeatmapDataPoint{
			Date:       dateStr,
			Count:      d.Count,
			Additions:  d.Additions,
			Deletions:  d.Deletions,
			WeekDay:    int(d.Date.Weekday()),
			WeekOfYear: week,
		}
		dataMap[dateStr] = point
		totalCount += d.Count
		if d.Count > maxCount {
			maxCount = d.Count
		}
	}

	var result []HeatmapDataPoint
	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("2006-01-02")
		if point, exists := dataMap[dateStr]; exists {
			result = append(result, point)
		} else {
			_, week := d.ISOWeek()
			result = append(result, HeatmapDataPoint{
				Date:       dateStr,
				Count:      0,
				Additions:  0,
				Deletions:  0,
				WeekDay:    int(d.Weekday()),
				WeekOfYear: week,
			})
		}
	}

	return &HeatmapResponse{
		Data:       result,
		TotalCount: totalCount,
		MaxCount:   maxCount,
		StartDate:  startDate.Format("2006-01-02"),
		EndDate:    endDate.Format("2006-01-02"),
	}, nil
}

func (s *MemberService) GetTeamOverview(req *TeamOverviewRequest) (*TeamOverviewResponse, error) {
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

	baseQuery := s.db.Model(&models.ReviewLog{}).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Where("author != ''")

	if req.ProjectID != nil {
		baseQuery = baseQuery.Where("project_id = ?", *req.ProjectID)
	}

	// Total stats
	var totalMembers, totalCommits int64
	var avgScore float64
	var totalAdditions, totalDeletions int64

	baseQuery.Distinct("author").Count(&totalMembers)

	s.db.Model(&models.ReviewLog{}).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Where("author != ''").
		Select("COUNT(*) as total_commits, COALESCE(AVG(CASE WHEN is_manual = false THEN score END), 0) as avg_score, COALESCE(SUM(additions), 0) as total_additions, COALESCE(SUM(deletions), 0) as total_deletions").
		Row().Scan(&totalCommits, &avgScore, &totalAdditions, &totalDeletions)

	// Team trend
	var trend []MemberTrendItem
	trendQuery := s.db.Model(&models.ReviewLog{}).
		Select(`DATE(created_at) as date, COUNT(*) as commit_count, COALESCE(AVG(CASE WHEN is_manual = false THEN score END), 0) as avg_score`).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Where("author != ''").
		Group("DATE(created_at)").
		Order("date ASC")

	if req.ProjectID != nil {
		trendQuery = trendQuery.Where("project_id = ?", *req.ProjectID)
	}
	trendQuery.Scan(&trend)

	// Top 10 members
	var topMembers []MemberStats
	topQuery := s.db.Model(&models.ReviewLog{}).
		Select(`
			author,
			MAX(author_email) as author_email,
			COUNT(*) as commit_count,
			COALESCE(AVG(CASE WHEN is_manual = false THEN score END), 0) as avg_score,
			COALESCE(MAX(CASE WHEN is_manual = false THEN score END), 0) as max_score,
			COALESCE(MIN(NULLIF(CASE WHEN is_manual = false THEN score END, 0)), 0) as min_score,
			COALESCE(SUM(additions), 0) as additions,
			COALESCE(SUM(deletions), 0) as deletions,
			COALESCE(SUM(files_changed), 0) as files_changed,
			COUNT(DISTINCT project_id) as project_count
		`).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Where("author != ''").
		Group("author").
		Order("commit_count DESC").
		Limit(10)

	if req.ProjectID != nil {
		topQuery = topQuery.Where("project_id = ?", *req.ProjectID)
	}
	topQuery.Scan(&topMembers)

	// Score distribution (exclude manual records)
	var excellent, good, needWork int64

	s.db.Model(&models.ReviewLog{}).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Where("author != ''").
		Where("is_manual = false").
		Where("score >= 80").
		Distinct("author").Count(&excellent)

	s.db.Model(&models.ReviewLog{}).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Where("author != ''").
		Where("is_manual = false").
		Where("score >= 60 AND score < 80").
		Distinct("author").Count(&good)

	s.db.Model(&models.ReviewLog{}).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Where("author != ''").
		Where("is_manual = false").
		Where("score < 60 AND score > 0").
		Distinct("author").Count(&needWork)

	return &TeamOverviewResponse{
		TotalMembers:   totalMembers,
		TotalCommits:   totalCommits,
		AvgScore:       avgScore,
		TotalAdditions: totalAdditions,
		TotalDeletions: totalDeletions,
		Trend:          trend,
		TopMembers:     topMembers,
		ScoreDistrib: ScoreDistribution{
			Excellent: excellent,
			Good:      good,
			NeedWork:  needWork,
		},
	}, nil
}
