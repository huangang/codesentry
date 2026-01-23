package services

import (
	"errors"

	"github.com/huangang/codesentry/backend/internal/models"
	"gorm.io/gorm"
)

type IMBotService struct {
	db *gorm.DB
}

func NewIMBotService(db *gorm.DB) *IMBotService {
	return &IMBotService{db: db}
}

type IMBotListRequest struct {
	Page     int    `form:"page" binding:"omitempty,min=1"`
	PageSize int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	Name     string `form:"name"`
	Type     string `form:"type"`
	IsActive *bool  `form:"is_active"`
}

type IMBotListResponse struct {
	Total    int64          `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
	Items    []models.IMBot `json:"items"`
}

type CreateIMBotRequest struct {
	Name        string `json:"name" binding:"required"`
	Type        string `json:"type" binding:"required,oneof=wechat_work dingtalk feishu slack discord teams telegram"`
	Webhook     string `json:"webhook" binding:"required"`
	Secret      string `json:"secret"`
	Extra       string `json:"extra"`
	IsActive    bool   `json:"is_active"`
	ErrorNotify bool   `json:"error_notify"`
}

type UpdateIMBotRequest struct {
	Name        string `json:"name"`
	Type        string `json:"type" binding:"omitempty,oneof=wechat_work dingtalk feishu slack discord teams telegram"`
	Webhook     string `json:"webhook"`
	Secret      string `json:"secret"`
	Extra       string `json:"extra"`
	IsActive    *bool  `json:"is_active"`
	ErrorNotify *bool  `json:"error_notify"`
}

// List returns paginated IM bots
func (s *IMBotService) List(req *IMBotListRequest) (*IMBotListResponse, error) {
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 10
	}

	var bots []models.IMBot
	var total int64

	query := s.db.Model(&models.IMBot{})

	if req.Name != "" {
		query = query.Where("name LIKE ?", "%"+req.Name+"%")
	}
	if req.Type != "" {
		query = query.Where("type = ?", req.Type)
	}
	if req.IsActive != nil {
		query = query.Where("is_active = ?", *req.IsActive)
	}

	query.Count(&total)

	offset := (req.Page - 1) * req.PageSize
	if err := query.Offset(offset).Limit(req.PageSize).Order("created_at DESC").Find(&bots).Error; err != nil {
		return nil, err
	}

	return &IMBotListResponse{
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
		Items:    bots,
	}, nil
}

// GetByID returns an IM bot by ID
func (s *IMBotService) GetByID(id uint) (*models.IMBot, error) {
	var bot models.IMBot
	if err := s.db.First(&bot, id).Error; err != nil {
		return nil, err
	}
	return &bot, nil
}

// Create creates a new IM bot
func (s *IMBotService) Create(req *CreateIMBotRequest) (*models.IMBot, error) {
	bot := models.IMBot{
		Name:        req.Name,
		Type:        req.Type,
		Webhook:     req.Webhook,
		Secret:      req.Secret,
		Extra:       req.Extra,
		IsActive:    req.IsActive,
		ErrorNotify: req.ErrorNotify,
	}

	if err := s.db.Create(&bot).Error; err != nil {
		return nil, err
	}

	return &bot, nil
}

// Update updates an IM bot
func (s *IMBotService) Update(id uint, req *UpdateIMBotRequest) (*models.IMBot, error) {
	var bot models.IMBot
	if err := s.db.First(&bot, id).Error; err != nil {
		return nil, err
	}

	updates := make(map[string]interface{})

	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Type != "" {
		updates["type"] = req.Type
	}
	if req.Webhook != "" {
		updates["webhook"] = req.Webhook
	}
	if req.Secret != "" {
		updates["secret"] = req.Secret
	}
	if req.Extra != "" {
		updates["extra"] = req.Extra
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	if req.ErrorNotify != nil {
		updates["error_notify"] = *req.ErrorNotify
	}

	if err := s.db.Model(&bot).Updates(updates).Error; err != nil {
		return nil, err
	}

	// Reload
	s.db.First(&bot, id)
	return &bot, nil
}

// Delete deletes an IM bot
func (s *IMBotService) Delete(id uint) error {
	result := s.db.Delete(&models.IMBot{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("im bot not found")
	}
	return nil
}

// GetAllActive returns all active IM bots
func (s *IMBotService) GetAllActive() ([]models.IMBot, error) {
	var bots []models.IMBot
	if err := s.db.Where("is_active = ?", true).Find(&bots).Error; err != nil {
		return nil, err
	}
	return bots, nil
}

// GetErrorNotifyBots returns all active bots with error notification enabled
func (s *IMBotService) GetErrorNotifyBots() ([]models.IMBot, error) {
	var bots []models.IMBot
	if err := s.db.Where("is_active = ? AND error_notify = ?", true, true).Find(&bots).Error; err != nil {
		return nil, err
	}
	return bots, nil
}
