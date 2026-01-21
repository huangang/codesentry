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
	Page       int       `form:"page" binding:"min=1"`
	PageSize   int       `form:"page_size" binding:"min=1,max=100"`
	EventType  string    `form:"event_type"`
	ProjectID  uint      `form:"project_id"`
	Author     string    `form:"author"`
	StartDate  time.Time `form:"start_date"`
	EndDate    time.Time `form:"end_date"`
	SearchText string    `form:"search_text"`
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
