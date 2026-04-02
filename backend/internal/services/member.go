package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/huangang/codesentry/backend/internal/models"
	"github.com/huangang/codesentry/backend/pkg/logger"
	"gorm.io/gorm"
)

type MemberService struct {
	db        *gorm.DB
	aiService *AIService
}

func NewMemberService(db *gorm.DB, aiService *AIService) *MemberService {
	return &MemberService{db: db, aiService: aiService}
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

// Work Summary types and methods

type WorkSummaryRequest struct {
	Author string `form:"author" binding:"required"`
	Date   string `form:"date"` // YYYY-MM-DD, defaults to today
}

type WorkSummaryCommit struct {
	ProjectName   string   `json:"project_name"`
	CommitMessage string   `json:"commit_message"`
	Branch        string   `json:"branch"`
	Additions     int      `json:"additions"`
	Deletions     int      `json:"deletions"`
	FilesChanged  int      `json:"files_changed"`
	Score         *float64 `json:"score"`
	CreatedAt     string   `json:"created_at"`
}

type WorkSummaryData struct {
	CommitCount   int                 `json:"commit_count"`
	Additions     int                 `json:"additions"`
	Deletions     int                 `json:"deletions"`
	FilesChanged  int                 `json:"files_changed"`
	Projects      []string            `json:"projects"`
	CommitDetails []WorkSummaryCommit `json:"commit_details"`
}

type WorkSummaryResponse struct {
	ID          uint            `json:"id,omitempty"`
	Author      string          `json:"author"`
	Date        string          `json:"date"`
	Summary     string          `json:"summary"`
	RawData     WorkSummaryData `json:"raw_data"`
	AIModelUsed string          `json:"ai_model_used"`
	Cached      bool            `json:"cached"`
	UpdatedAt   string          `json:"updated_at,omitempty"`
}

type WorkSummaryListRequest struct {
	Author   string `form:"author" binding:"required"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

type WorkSummaryListItem struct {
	ID          uint   `json:"id"`
	Author      string `json:"author"`
	Date        string `json:"date"`
	AIModelUsed string `json:"ai_model_used"`
	HasSummary  bool   `json:"has_summary"`
	UpdatedAt   string `json:"updated_at"`
}

type WorkSummaryListResponse struct {
	Total    int64                 `json:"total"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"page_size"`
	Items    []WorkSummaryListItem `json:"items"`
}

func (s *MemberService) GenerateWorkSummary(req *WorkSummaryRequest) (*WorkSummaryResponse, error) {
	// Parse date, default to today. Always use local timezone for accurate day boundaries.
	now := time.Now()
	year, month, day := now.Year(), now.Month(), now.Day()
	if req.Date != "" {
		parsed, err := time.Parse("2006-01-02", req.Date)
		if err == nil {
			year, month, day = parsed.Year(), parsed.Month(), parsed.Day()
		}
	}

	startOfDay := time.Date(year, month, day, 0, 0, 0, 0, now.Location())
	endOfDay := time.Date(year, month, day, 23, 59, 59, 999999999, now.Location())
	dateStr := startOfDay.Format("2006-01-02")

	// Query review logs for this author on this date
	var logs []models.ReviewLog
	if err := s.db.Preload("Project").
		Where("author = ? AND created_at BETWEEN ? AND ?", req.Author, startOfDay, endOfDay).
		Order("created_at ASC").
		Find(&logs).Error; err != nil {
		return nil, fmt.Errorf("failed to query review logs: %w", err)
	}

	if len(logs) == 0 {
		return &WorkSummaryResponse{
			Author:  req.Author,
			Date:    dateStr,
			Summary: "",
			RawData: WorkSummaryData{},
		}, nil
	}

	// Build raw data
	projectSet := make(map[string]bool)
	var totalAdditions, totalDeletions, totalFiles int
	var commits []WorkSummaryCommit

	for _, log := range logs {
		projectName := ""
		if log.Project != nil {
			projectName = log.Project.Name
			projectSet[projectName] = true
		}

		totalAdditions += log.Additions
		totalDeletions += log.Deletions
		totalFiles += log.FilesChanged

		commits = append(commits, WorkSummaryCommit{
			ProjectName:   projectName,
			CommitMessage: log.CommitMessage,
			Branch:        log.Branch,
			Additions:     log.Additions,
			Deletions:     log.Deletions,
			FilesChanged:  log.FilesChanged,
			Score:         log.Score,
			CreatedAt:     log.CreatedAt.Format("15:04:05"),
		})
	}

	var projects []string
	for p := range projectSet {
		projects = append(projects, p)
	}

	rawData := WorkSummaryData{
		CommitCount:   len(logs),
		Additions:     totalAdditions,
		Deletions:     totalDeletions,
		FilesChanged:  totalFiles,
		Projects:      projects,
		CommitDetails: commits,
	}

	// Generate AI summary
	summary, modelUsed := s.generateAISummary(req.Author, dateStr, rawData)

	resp := &WorkSummaryResponse{
		Author:      req.Author,
		Date:        dateStr,
		Summary:     summary,
		RawData:     rawData,
		AIModelUsed: modelUsed,
		Cached:      false,
	}

	// Persist: upsert by author+date
	if summary != "" {
		rawDataJSON, _ := json.Marshal(rawData)
		var existing models.MemberWorkSummary
		err := s.db.Where("author = ? AND date = ?", req.Author, dateStr).First(&existing).Error
		if err != nil {
			// Create new
			record := models.MemberWorkSummary{
				Author:      req.Author,
				Date:        dateStr,
				Summary:     summary,
				RawData:     string(rawDataJSON),
				AIModelUsed: modelUsed,
			}
			if createErr := s.db.Create(&record).Error; createErr != nil {
				logger.Infof("[Member] Failed to save work summary: %v", createErr)
			} else {
				resp.ID = record.ID
				resp.UpdatedAt = record.UpdatedAt.Format(time.RFC3339)
			}
		} else {
			// Update existing
			existing.Summary = summary
			existing.RawData = string(rawDataJSON)
			existing.AIModelUsed = modelUsed
			if updateErr := s.db.Save(&existing).Error; updateErr != nil {
				logger.Infof("[Member] Failed to update work summary: %v", updateErr)
			} else {
				resp.ID = existing.ID
				resp.UpdatedAt = existing.UpdatedAt.Format(time.RFC3339)
			}
		}
	}

	return resp, nil
}

// GetWorkSummary returns a cached work summary for the given author and date.
func (s *MemberService) GetWorkSummary(req *WorkSummaryRequest) (*WorkSummaryResponse, error) {
	now := time.Now()
	year, month, day := now.Year(), now.Month(), now.Day()
	if req.Date != "" {
		parsed, err := time.Parse("2006-01-02", req.Date)
		if err == nil {
			year, month, day = parsed.Year(), parsed.Month(), parsed.Day()
		}
	}
	dateStr := time.Date(year, month, day, 0, 0, 0, 0, now.Location()).Format("2006-01-02")

	var record models.MemberWorkSummary
	if err := s.db.Where("author = ? AND date = ?", req.Author, dateStr).First(&record).Error; err != nil {
		// No cached summary
		return &WorkSummaryResponse{
			Author: req.Author,
			Date:   dateStr,
			Cached: false,
		}, nil
	}

	// Decode raw data
	var rawData WorkSummaryData
	if record.RawData != "" {
		json.Unmarshal([]byte(record.RawData), &rawData)
	}

	return &WorkSummaryResponse{
		ID:          record.ID,
		Author:      record.Author,
		Date:        record.Date,
		Summary:     record.Summary,
		RawData:     rawData,
		AIModelUsed: record.AIModelUsed,
		Cached:      true,
		UpdatedAt:   record.UpdatedAt.Format(time.RFC3339),
	}, nil
}

// ListWorkSummaries returns paginated work summary history for an author.
func (s *MemberService) ListWorkSummaries(req *WorkSummaryListRequest) (*WorkSummaryListResponse, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}

	var total int64
	s.db.Model(&models.MemberWorkSummary{}).Where("author = ?", req.Author).Count(&total)

	var records []models.MemberWorkSummary
	offset := (req.Page - 1) * req.PageSize
	s.db.Where("author = ?", req.Author).
		Order("date DESC").
		Offset(offset).
		Limit(req.PageSize).
		Find(&records)

	items := make([]WorkSummaryListItem, len(records))
	for i, r := range records {
		items[i] = WorkSummaryListItem{
			ID:          r.ID,
			Author:      r.Author,
			Date:        r.Date,
			AIModelUsed: r.AIModelUsed,
			HasSummary:  r.Summary != "",
			UpdatedAt:   r.UpdatedAt.Format(time.RFC3339),
		}
	}

	return &WorkSummaryListResponse{
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
		Items:    items,
	}, nil
}

func (s *MemberService) generateAISummary(author, date string, data WorkSummaryData) (string, string) {
	// Build commit details for prompt
	var commitLines []string
	for _, c := range data.CommitDetails {
		line := fmt.Sprintf("- [%s] %s | %s (分支: %s, +%d/-%d, %d 文件)",
			c.CreatedAt, c.ProjectName, c.CommitMessage, c.Branch, c.Additions, c.Deletions, c.FilesChanged)
		commitLines = append(commitLines, line)
	}

	commitList := strings.Join(commitLines, "\n")

	prompt := fmt.Sprintf(`你是一位技术日报助手。请根据以下代码提交记录，为 %s 生成一份 %s 的工作总结（日报格式）。

## 提交记录
%s

## 统计
- 总提交数: %d
- 涉及项目: %s
- 新增行数: %d
- 删除行数: %d
- 变更文件: %d

## 要求
1. 用中文输出
2. 按项目分组总结今日工作内容
3. 从 commit message 中提炼出具体做了什么（功能开发、Bug 修复、重构等）
4. 输出 Markdown 格式，适合直接粘贴到日报中
5. 简洁明了，不要重复 commit message 原文，而是归纳总结
6. 如果有多个相关的小提交，合并为一条工作项
7. 最后给出一行简短的「今日小结」

格式示例：
## 今日工作 - YYYY-MM-DD

### 项目名称
- 完成了 xxx 功能的开发
- 修复了 xxx 的 Bug

### 另一个项目
- 重构了 xxx 模块

**今日小结**: 主要完成了 xxx，推进了 xxx 的进度。`,
		author, date, commitList, data.CommitCount,
		strings.Join(data.Projects, ", "), data.Additions, data.Deletions, data.FilesChanged)

	if s.aiService == nil {
		logger.Infof("[Member] AI service not available, using default summary")
		return s.buildDefaultWorkSummary(author, date, data), ""
	}

	content, modelName, err := s.aiService.CallWithConfig(context.Background(), 0, prompt)
	if err != nil {
		logger.Infof("[Member] AI work summary failed: %v, using default", err)
		return s.buildDefaultWorkSummary(author, date, data), ""
	}

	return content, modelName
}

func (s *MemberService) buildDefaultWorkSummary(author, date string, data WorkSummaryData) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("## 今日工作 - %s\n\n", date))
	sb.WriteString(fmt.Sprintf("**%s** 的工作总结\n\n", author))

	// Group commits by project
	projectCommits := make(map[string][]WorkSummaryCommit)
	for _, c := range data.CommitDetails {
		name := c.ProjectName
		if name == "" {
			name = "其他"
		}
		projectCommits[name] = append(projectCommits[name], c)
	}

	for project, commits := range projectCommits {
		sb.WriteString(fmt.Sprintf("### %s\n", project))
		for _, c := range commits {
			sb.WriteString(fmt.Sprintf("- %s (+%d/-%d)\n", c.CommitMessage, c.Additions, c.Deletions))
		}
		sb.WriteString("\n")
	}

	sb.WriteString(fmt.Sprintf("**统计**: %d 次提交, +%d/-%d 行, %d 个文件\n",
		data.CommitCount, data.Additions, data.Deletions, data.FilesChanged))

	return sb.String()
}
