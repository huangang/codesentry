package services

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
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

// loadAliasMap returns a map of alias -> canonical_name for all defined aliases.
func (s *MemberService) loadAliasMap() map[string]string {
	var aliases []models.MemberAlias
	s.db.Find(&aliases)
	m := make(map[string]string, len(aliases))
	for _, a := range aliases {
		m[a.Alias] = a.CanonicalName
	}
	return m
}

// expandAuthors returns all author names (including aliases) that map to the given name.
// If name is itself an alias it is resolved to its canonical name first.
func (s *MemberService) expandAuthors(name string) (canonical string, authors []string) {
	// Check if name is itself an alias
	var selfAlias models.MemberAlias
	if err := s.db.Where("alias = ?", name).First(&selfAlias).Error; err == nil {
		canonical = selfAlias.CanonicalName
	} else {
		canonical = name
	}
	authors = []string{canonical}
	// Add all aliases that map to canonical
	var aliases []models.MemberAlias
	s.db.Where("canonical_name = ?", canonical).Find(&aliases)
	for _, a := range aliases {
		if a.Alias != canonical {
			authors = append(authors, a.Alias)
		}
	}
	return canonical, authors
}

// applyAliasesToMembers merges members with the same canonical name.
func applyAliasesToMembers(members []MemberStats, aliasMap map[string]string) []MemberStats {
	if len(aliasMap) == 0 {
		return members
	}
	type mergedEntry struct {
		stats      MemberStats
		scoreSum   float64
		scoreCount int64
	}
	order := make([]string, 0, len(members))
	merged := make(map[string]*mergedEntry)

	for _, m := range members {
		canonical := m.Author
		if c, ok := aliasMap[m.Author]; ok {
			canonical = c
		}
		if e, exists := merged[canonical]; exists {
			e.stats.CommitCount += m.CommitCount
			e.stats.Additions += m.Additions
			e.stats.Deletions += m.Deletions
			e.stats.FilesChanged += m.FilesChanged
			if m.ProjectCount > e.stats.ProjectCount {
				e.stats.ProjectCount = m.ProjectCount
			}
			if m.MaxScore > e.stats.MaxScore {
				e.stats.MaxScore = m.MaxScore
			}
			if m.MinScore > 0 && (e.stats.MinScore == 0 || m.MinScore < e.stats.MinScore) {
				e.stats.MinScore = m.MinScore
			}
			e.scoreSum += m.AvgScore * float64(m.CommitCount)
			e.scoreCount += m.CommitCount
		} else {
			order = append(order, canonical)
			merged[canonical] = &mergedEntry{
				stats: MemberStats{
					Author:       canonical,
					AuthorEmail:  m.AuthorEmail,
					CommitCount:  m.CommitCount,
					AvgScore:     m.AvgScore,
					MaxScore:     m.MaxScore,
					MinScore:     m.MinScore,
					Additions:    m.Additions,
					Deletions:    m.Deletions,
					FilesChanged: m.FilesChanged,
					ProjectCount: m.ProjectCount,
				},
				scoreSum:   m.AvgScore * float64(m.CommitCount),
				scoreCount: m.CommitCount,
			}
		}
	}

	result := make([]MemberStats, 0, len(order))
	for _, canonical := range order {
		e := merged[canonical]
		if e.scoreCount > 0 {
			e.stats.AvgScore = e.scoreSum / float64(e.scoreCount)
		}
		result = append(result, e.stats)
	}
	return result
}

// Alias CRUD

type MemberAliasItem struct {
	ID            uint   `json:"id"`
	Alias         string `json:"alias"`
	CanonicalName string `json:"canonical_name"`
}

func (s *MemberService) ListAliases() ([]MemberAliasItem, error) {
	var aliases []models.MemberAlias
	if err := s.db.Order("canonical_name, alias").Find(&aliases).Error; err != nil {
		return nil, err
	}
	items := make([]MemberAliasItem, len(aliases))
	for i, a := range aliases {
		items[i] = MemberAliasItem{ID: a.ID, Alias: a.Alias, CanonicalName: a.CanonicalName}
	}
	return items, nil
}

type CreateAliasRequest struct {
	Alias         string `json:"alias" binding:"required"`
	CanonicalName string `json:"canonical_name" binding:"required"`
}

func (s *MemberService) CreateAlias(req *CreateAliasRequest) (*MemberAliasItem, error) {
	alias := models.MemberAlias{
		Alias:         req.Alias,
		CanonicalName: req.CanonicalName,
	}
	if err := s.db.Create(&alias).Error; err != nil {
		return nil, err
	}
	return &MemberAliasItem{ID: alias.ID, Alias: alias.Alias, CanonicalName: alias.CanonicalName}, nil
}

func (s *MemberService) DeleteAlias(id uint) error {
	return s.db.Delete(&models.MemberAlias{}, id).Error
}

// sortMembers sorts a slice of MemberStats in place.
func sortMembers(members []MemberStats, sortBy, sortOrder string) {
	asc := strings.EqualFold(sortOrder, "ASC")
	sort.Slice(members, func(i, j int) bool {
		var leftLessRight bool
		var leftGreaterRight bool
		switch sortBy {
		case "avg_score":
			leftLessRight = members[i].AvgScore < members[j].AvgScore
			leftGreaterRight = members[i].AvgScore > members[j].AvgScore
		case "additions":
			leftLessRight = members[i].Additions < members[j].Additions
			leftGreaterRight = members[i].Additions > members[j].Additions
		case "deletions":
			leftLessRight = members[i].Deletions < members[j].Deletions
			leftGreaterRight = members[i].Deletions > members[j].Deletions
		case "files_changed":
			leftLessRight = members[i].FilesChanged < members[j].FilesChanged
			leftGreaterRight = members[i].FilesChanged > members[j].FilesChanged
		default: // commit_count
			leftLessRight = members[i].CommitCount < members[j].CommitCount
			leftGreaterRight = members[i].CommitCount > members[j].CommitCount
		}
		if asc {
			return leftLessRight
		}
		return leftGreaterRight
	})
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

	// Fetch all (no pagination yet) so aliases can be merged across pages
	var allMembers []MemberStats
	query.Scan(&allMembers)

	// Apply alias merging
	aliasMap := s.loadAliasMap()
	allMembers = applyAliasesToMembers(allMembers, aliasMap)

	// Re-sort after merging
	sortMembers(allMembers, sortBy, sortOrder)

	// Filter by name after alias merge
	if req.Name != "" {
		filtered := allMembers[:0]
		for _, m := range allMembers {
			if strings.Contains(strings.ToLower(m.Author), strings.ToLower(req.Name)) {
				filtered = append(filtered, m)
			}
		}
		allMembers = filtered
	}

	total := int64(len(allMembers))
	offset := (req.Page - 1) * req.PageSize
	end := offset + req.PageSize
	if offset > len(allMembers) {
		offset = len(allMembers)
	}
	if end > len(allMembers) {
		end = len(allMembers)
	}
	members := allMembers[offset:end]

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

	// Resolve aliases: query for all author names that belong to this person
	canonical, authorNames := s.expandAuthors(req.Author)

	var totalStats MemberStats
	s.db.Model(&models.ReviewLog{}).
		Select(`
			MAX(author) as author,
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
		Where("author IN ?", authorNames).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Scan(&totalStats)
	totalStats.Author = canonical

	var projectStats []MemberProjectStats
	s.db.Model(&models.ReviewLog{}).
		Select(`
			project_id,
			COUNT(*) as commit_count,
			COALESCE(AVG(CASE WHEN is_manual = false THEN score END), 0) as avg_score,
			COALESCE(SUM(additions), 0) as additions,
			COALESCE(SUM(deletions), 0) as deletions
		`).
		Where("author IN ?", authorNames).
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
		Where("author IN ?", authorNames).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Group("DATE(created_at)").
		Order("date ASC").
		Scan(&trend)

	return &MemberDetailResponse{
		Author:       canonical,
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
		_, authorNames := s.expandAuthors(req.Author)
		query = query.Where("author IN ?", authorNames)
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
	Author string `form:"author" json:"author" binding:"required"`
	Date   string `form:"date" json:"date"` // YYYY-MM-DD, defaults to today
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

type WorkPeriodSummaryRequest struct {
	Author    string `form:"author" json:"author" binding:"required"`
	StartDate string `form:"start_date" json:"start_date"` // YYYY-MM-DD, defaults to last 7 days
	EndDate   string `form:"end_date" json:"end_date"`     // YYYY-MM-DD, defaults to today
}

type WorkPeriodDailySummary struct {
	Date         string   `json:"date"`
	CommitCount  int      `json:"commit_count"`
	Additions    int      `json:"additions"`
	Deletions    int      `json:"deletions"`
	FilesChanged int      `json:"files_changed"`
	Projects     []string `json:"projects"`
	Summary      string   `json:"summary"`
	Cached       bool     `json:"cached"`
}

type WorkPeriodSummaryData struct {
	CommitCount  int      `json:"commit_count"`
	Additions    int      `json:"additions"`
	Deletions    int      `json:"deletions"`
	FilesChanged int      `json:"files_changed"`
	Projects     []string `json:"projects"`
}

type WorkPeriodSummaryResponse struct {
	ID          uint                  `json:"id,omitempty"`
	Author      string                `json:"author"`
	StartDate   string                `json:"start_date"`
	EndDate     string                `json:"end_date"`
	Summary     string                `json:"summary"`
	RawData     WorkPeriodSummaryData `json:"raw_data"`
	AIModelUsed string                `json:"ai_model_used"`
	Cached      bool                  `json:"cached"`
	UpdatedAt   string                `json:"updated_at,omitempty"`
}

type WorkPeriodSummaryListRequest struct {
	Author   string `form:"author" binding:"required"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

type WorkPeriodSummaryListItem struct {
	ID          uint   `json:"id"`
	Author      string `json:"author"`
	StartDate   string `json:"start_date"`
	EndDate     string `json:"end_date"`
	AIModelUsed string `json:"ai_model_used"`
	HasSummary  bool   `json:"has_summary"`
	UpdatedAt   string `json:"updated_at"`
}

type WorkPeriodSummaryListResponse struct {
	Total    int64                       `json:"total"`
	Page     int                         `json:"page"`
	PageSize int                         `json:"page_size"`
	Items    []WorkPeriodSummaryListItem `json:"items"`
}

const maxWorkPeriodSummaryDays = 366

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
	sort.Strings(projects)

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

func parseWorkPeriodSummaryRange(startDate, endDate string) (time.Time, time.Time, error) {
	now := time.Now()
	loc := now.Location()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	start := today.AddDate(0, 0, -6)
	end := today

	if startDate != "" {
		parsed, err := time.Parse("2006-01-02", startDate)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid start_date, expected YYYY-MM-DD")
		}
		start = time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, loc)
	}

	if endDate != "" {
		parsed, err := time.Parse("2006-01-02", endDate)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid end_date, expected YYYY-MM-DD")
		}
		end = time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, loc)
	}

	if startDate == "" && endDate != "" {
		start = end
	}
	if endDate == "" && startDate != "" {
		end = start
	}
	if start.After(end) {
		return time.Time{}, time.Time{}, fmt.Errorf("start_date cannot be after end_date")
	}

	dayCount := int(end.Sub(start).Hours()/24) + 1
	if dayCount > maxWorkPeriodSummaryDays {
		return time.Time{}, time.Time{}, fmt.Errorf("date range cannot exceed %d days", maxWorkPeriodSummaryDays)
	}

	return start, end, nil
}

func (s *MemberService) GenerateWorkPeriodSummary(req *WorkPeriodSummaryRequest) (*WorkPeriodSummaryResponse, error) {
	start, end, err := parseWorkPeriodSummaryRange(req.StartDate, req.EndDate)
	if err != nil {
		return nil, err
	}

	startStr := start.Format("2006-01-02")
	endStr := end.Format("2006-01-02")
	projectSet := make(map[string]bool)
	rawData := WorkPeriodSummaryData{}
	var dailySummaries []WorkPeriodDailySummary

	for day := start; !day.After(end); day = day.AddDate(0, 0, 1) {
		dateStr := day.Format("2006-01-02")
		dayReq := &WorkSummaryRequest{Author: req.Author, Date: dateStr}
		daySummary, err := s.GetWorkSummary(dayReq)
		if err != nil {
			return nil, err
		}

		if strings.TrimSpace(daySummary.Summary) == "" {
			daySummary, err = s.GenerateWorkSummary(dayReq)
			if err != nil {
				return nil, err
			}
		}

		if daySummary.RawData.CommitCount == 0 && strings.TrimSpace(daySummary.Summary) == "" {
			continue
		}

		daily := WorkPeriodDailySummary{
			Date:         dateStr,
			CommitCount:  daySummary.RawData.CommitCount,
			Additions:    daySummary.RawData.Additions,
			Deletions:    daySummary.RawData.Deletions,
			FilesChanged: daySummary.RawData.FilesChanged,
			Projects:     daySummary.RawData.Projects,
			Summary:      daySummary.Summary,
			Cached:       daySummary.Cached,
		}
		dailySummaries = append(dailySummaries, daily)
		rawData.CommitCount += daily.CommitCount
		rawData.Additions += daily.Additions
		rawData.Deletions += daily.Deletions
		rawData.FilesChanged += daily.FilesChanged
		for _, project := range daily.Projects {
			if project != "" {
				projectSet[project] = true
			}
		}
	}

	for project := range projectSet {
		rawData.Projects = append(rawData.Projects, project)
	}
	sort.Strings(rawData.Projects)

	if len(dailySummaries) == 0 {
		return &WorkPeriodSummaryResponse{
			Author:    req.Author,
			StartDate: startStr,
			EndDate:   endStr,
			RawData:   rawData,
		}, nil
	}

	summary, modelUsed := s.generateAIPeriodSummary(req.Author, startStr, endStr, rawData, dailySummaries)
	resp := &WorkPeriodSummaryResponse{
		Author:      req.Author,
		StartDate:   startStr,
		EndDate:     endStr,
		Summary:     summary,
		RawData:     rawData,
		AIModelUsed: modelUsed,
		Cached:      false,
	}

	if summary != "" {
		rawDataJSON, _ := json.Marshal(rawData)
		var existing models.MemberWorkPeriodSummary
		err := s.db.Where("author = ? AND start_date = ? AND end_date = ?", req.Author, startStr, endStr).First(&existing).Error
		if err != nil {
			record := models.MemberWorkPeriodSummary{
				Author:      req.Author,
				StartDate:   startStr,
				EndDate:     endStr,
				Summary:     summary,
				RawData:     string(rawDataJSON),
				AIModelUsed: modelUsed,
			}
			if createErr := s.db.Create(&record).Error; createErr != nil {
				logger.Infof("[Member] Failed to save work period summary: %v", createErr)
			} else {
				resp.ID = record.ID
				resp.UpdatedAt = record.UpdatedAt.Format(time.RFC3339)
			}
		} else {
			existing.Summary = summary
			existing.RawData = string(rawDataJSON)
			existing.AIModelUsed = modelUsed
			if updateErr := s.db.Save(&existing).Error; updateErr != nil {
				logger.Infof("[Member] Failed to update work period summary: %v", updateErr)
			} else {
				resp.ID = existing.ID
				resp.UpdatedAt = existing.UpdatedAt.Format(time.RFC3339)
			}
		}
	}

	return resp, nil
}

func (s *MemberService) GetWorkPeriodSummary(req *WorkPeriodSummaryRequest) (*WorkPeriodSummaryResponse, error) {
	start, end, err := parseWorkPeriodSummaryRange(req.StartDate, req.EndDate)
	if err != nil {
		return nil, err
	}

	startStr := start.Format("2006-01-02")
	endStr := end.Format("2006-01-02")
	var record models.MemberWorkPeriodSummary
	if err := s.db.Where("author = ? AND start_date = ? AND end_date = ?", req.Author, startStr, endStr).First(&record).Error; err != nil {
		return &WorkPeriodSummaryResponse{
			Author:    req.Author,
			StartDate: startStr,
			EndDate:   endStr,
			Cached:    false,
		}, nil
	}

	var rawData WorkPeriodSummaryData
	if record.RawData != "" {
		json.Unmarshal([]byte(record.RawData), &rawData)
	}

	return &WorkPeriodSummaryResponse{
		ID:          record.ID,
		Author:      record.Author,
		StartDate:   record.StartDate,
		EndDate:     record.EndDate,
		Summary:     record.Summary,
		RawData:     rawData,
		AIModelUsed: record.AIModelUsed,
		Cached:      true,
		UpdatedAt:   record.UpdatedAt.Format(time.RFC3339),
	}, nil
}

func (s *MemberService) ListWorkPeriodSummaries(req *WorkPeriodSummaryListRequest) (*WorkPeriodSummaryListResponse, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}

	var total int64
	s.db.Model(&models.MemberWorkPeriodSummary{}).Where("author = ?", req.Author).Count(&total)

	var records []models.MemberWorkPeriodSummary
	offset := (req.Page - 1) * req.PageSize
	s.db.Where("author = ?", req.Author).
		Order("start_date DESC, end_date DESC").
		Offset(offset).
		Limit(req.PageSize).
		Find(&records)

	items := make([]WorkPeriodSummaryListItem, len(records))
	for i, r := range records {
		items[i] = WorkPeriodSummaryListItem{
			ID:          r.ID,
			Author:      r.Author,
			StartDate:   r.StartDate,
			EndDate:     r.EndDate,
			AIModelUsed: r.AIModelUsed,
			HasSummary:  r.Summary != "",
			UpdatedAt:   r.UpdatedAt.Format(time.RFC3339),
		}
	}

	return &WorkPeriodSummaryListResponse{
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

func (s *MemberService) generateAIPeriodSummary(author, startDate, endDate string, data WorkPeriodSummaryData, dailySummaries []WorkPeriodDailySummary) (string, string) {
	var dailySections []string
	for _, daily := range dailySummaries {
		summary := truncateRunes(strings.TrimSpace(daily.Summary), 2000)
		section := fmt.Sprintf(`### %s
- 提交数: %d
- 涉及项目: %s
- 代码变更: +%d/-%d, %d 个文件

%s`,
			daily.Date,
			daily.CommitCount,
			strings.Join(daily.Projects, ", "),
			daily.Additions,
			daily.Deletions,
			daily.FilesChanged,
			summary,
		)
		dailySections = append(dailySections, section)
	}

	prompt := fmt.Sprintf(`你是一位技术周报/阶段总结助手。请根据下面的「每日工作总结」为 %s 生成一份 %s 至 %s 的跨周期工作总结。

## 每日工作总结
%s

## 周期统计
- 总提交数: %d
- 涉及项目: %s
- 新增行数: %d
- 删除行数: %d
- 变更文件: %d

## 要求
1. 用中文输出
2. 先归纳周期内最重要的成果，不要逐天流水账
3. 按项目或主题合并相关工作，体现连续推进关系
4. 保留必要的关键数字，但不要堆砌所有统计
5. 如果发现重复修复、风险收敛、跨项目协作等模式，要单独指出
6. 输出 Markdown 格式，适合直接粘贴到周报、月报或阶段总结中
7. 最后给出一行简短的「周期小结」

格式示例：
## 工作总结 - YYYY-MM-DD 至 YYYY-MM-DD

### 总体概览
- 本周期主要完成了 xxx

### 重点工作
- 围绕 xxx，完成了 xxx

### 风险与跟进
- xxx 仍需继续观察

**周期小结**: 本周期围绕 xxx 持续推进，主要产出是 xxx。`,
		author,
		startDate,
		endDate,
		strings.Join(dailySections, "\n\n"),
		data.CommitCount,
		strings.Join(data.Projects, ", "),
		data.Additions,
		data.Deletions,
		data.FilesChanged,
	)

	if s.aiService == nil {
		logger.Infof("[Member] AI service not available, using default period summary")
		return s.buildDefaultWorkPeriodSummary(author, startDate, endDate, data, dailySummaries), ""
	}

	content, modelName, err := s.aiService.CallWithConfig(context.Background(), 0, prompt)
	if err != nil {
		logger.Infof("[Member] AI work period summary failed: %v, using default", err)
		return s.buildDefaultWorkPeriodSummary(author, startDate, endDate, data, dailySummaries), ""
	}

	return content, modelName
}

func truncateRunes(text string, maxRunes int) string {
	runes := []rune(text)
	if len(runes) <= maxRunes {
		return text
	}
	return string(runes[:maxRunes]) + "\n...(已截断)"
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

func (s *MemberService) buildDefaultWorkPeriodSummary(author, startDate, endDate string, data WorkPeriodSummaryData, dailySummaries []WorkPeriodDailySummary) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("## 工作总结 - %s 至 %s\n\n", startDate, endDate))
	sb.WriteString(fmt.Sprintf("**%s** 的周期工作总结\n\n", author))
	sb.WriteString("### 总体概览\n")
	sb.WriteString(fmt.Sprintf("- 本周期共 %d 次提交，涉及 %d 个项目，代码变更 +%d/-%d，影响 %d 个文件。\n\n",
		data.CommitCount, len(data.Projects), data.Additions, data.Deletions, data.FilesChanged))

	sb.WriteString("### 每日摘要\n")
	for _, daily := range dailySummaries {
		if strings.TrimSpace(daily.Summary) == "" {
			continue
		}
		sb.WriteString(fmt.Sprintf("- **%s**: %d 次提交，涉及 %s。\n",
			daily.Date, daily.CommitCount, strings.Join(daily.Projects, ", ")))
	}

	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("**周期小结**: %s 至 %s 期间主要围绕 %s 推进工作。\n",
		startDate, endDate, strings.Join(data.Projects, ", ")))

	return sb.String()
}
