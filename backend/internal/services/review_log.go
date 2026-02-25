package services

import (
	"time"

	"github.com/huangang/codesentry/backend/internal/models"
	"gorm.io/gorm"
)

type ReviewLogService struct {
	db *gorm.DB
}

func NewReviewLogService(db *gorm.DB) *ReviewLogService {
	return &ReviewLogService{db: db}
}

type ReviewLogListRequest struct {
	Page         int       `form:"page" binding:"omitempty,min=1"`
	PageSize     int       `form:"page_size" binding:"omitempty,min=1,max=100"`
	EventType    string    `form:"event_type"`
	ProjectID    uint      `form:"project_id"`
	Author       string    `form:"author"`
	StartDate    time.Time `form:"start_date"`
	EndDate      time.Time `form:"end_date"`
	SearchText   string    `form:"search_text"`
	ReviewStatus string    `form:"review_status"`
	MinScore     *float64  `form:"min_score"`
	MaxScore     *float64  `form:"max_score"`
}

type ReviewLogListResponse struct {
	Total    int64              `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"page_size"`
	Items    []models.ReviewLog `json:"items"`
}

// List returns paginated review logs
func (s *ReviewLogService) List(req *ReviewLogListRequest) (*ReviewLogListResponse, error) {
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 10
	}

	var logs []models.ReviewLog
	var total int64

	query := s.db.Model(&models.ReviewLog{}).Preload("Project")

	if req.EventType != "" {
		query = query.Where("event_type = ?", req.EventType)
	}
	if req.ProjectID > 0 {
		query = query.Where("project_id = ?", req.ProjectID)
	}
	if req.Author != "" {
		query = query.Where("author LIKE ?", "%"+req.Author+"%")
	}
	if !req.StartDate.IsZero() {
		query = query.Where("created_at >= ?", req.StartDate)
	}
	if !req.EndDate.IsZero() {
		query = query.Where("created_at <= ?", req.EndDate)
	}
	if req.SearchText != "" {
		query = query.Where("commit_message LIKE ?", "%"+req.SearchText+"%")
	}
	if req.ReviewStatus != "" {
		query = query.Where("review_status = ?", req.ReviewStatus)
	}
	if req.MinScore != nil {
		query = query.Where("score >= ?", *req.MinScore)
	}
	if req.MaxScore != nil {
		query = query.Where("score <= ?", *req.MaxScore)
	}

	query.Count(&total)

	offset := (req.Page - 1) * req.PageSize
	if err := query.Offset(offset).Limit(req.PageSize).Order("created_at DESC").Find(&logs).Error; err != nil {
		return nil, err
	}

	return &ReviewLogListResponse{
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
		Items:    logs,
	}, nil
}

// GetByID returns a review log by ID
func (s *ReviewLogService) GetByID(id uint) (*models.ReviewLog, error) {
	var log models.ReviewLog
	if err := s.db.Preload("Project").First(&log, id).Error; err != nil {
		return nil, err
	}
	return &log, nil
}

// Create creates a new review log
func (s *ReviewLogService) Create(log *models.ReviewLog) error {
	return s.db.Create(log).Error
}

// Update updates a review log
func (s *ReviewLogService) Update(log *models.ReviewLog) error {
	return s.db.Save(log).Error
}

// Delete deletes a review log by ID
func (s *ReviewLogService) Delete(id uint) error {
	return s.db.Delete(&models.ReviewLog{}, id).Error
}

type ManualCommitRequest struct {
	ProjectID     uint   `json:"project_id" binding:"required"`
	CommitHash    string `json:"commit_hash" binding:"required"`
	CommitURL     string `json:"commit_url"`
	Branch        string `json:"branch"`
	Author        string `json:"author" binding:"required"`
	AuthorEmail   string `json:"author_email"`
	CommitMessage string `json:"commit_message"`
	FilesChanged  int    `json:"files_changed"`
	Additions     int    `json:"additions"`
	Deletions     int    `json:"deletions"`
	CommitDate    string `json:"commit_date"`
}

func (s *ReviewLogService) CreateManualCommit(req *ManualCommitRequest) (*models.ReviewLog, error) {
	var project models.Project
	if err := s.db.First(&project, req.ProjectID).Error; err != nil {
		return nil, err
	}

	commitDate := time.Now()
	if req.CommitDate != "" {
		if parsed, err := time.Parse("2006-01-02", req.CommitDate); err == nil {
			commitDate = parsed
		}
	}

	log := &models.ReviewLog{
		ProjectID:     req.ProjectID,
		EventType:     "push",
		CommitHash:    req.CommitHash,
		CommitURL:     req.CommitURL,
		Branch:        req.Branch,
		Author:        req.Author,
		AuthorEmail:   req.AuthorEmail,
		CommitMessage: req.CommitMessage,
		FilesChanged:  req.FilesChanged,
		Additions:     req.Additions,
		Deletions:     req.Deletions,
		ReviewStatus:  "manual",
		IsManual:      true,
		CreatedAt:     commitDate,
	}

	if err := s.db.Create(log).Error; err != nil {
		return nil, err
	}

	return log, nil
}
