package services

import (
	"github.com/huangang/codesentry/backend/internal/models"
	"gorm.io/gorm"
)

type PromptService struct {
	db *gorm.DB
}

func NewPromptService(db *gorm.DB) *PromptService {
	return &PromptService{db: db}
}

type PromptListParams struct {
	Page     int
	PageSize int
	Name     string
	IsSystem *bool
}

type PromptListResult struct {
	Items []models.PromptTemplate `json:"items"`
	Total int64                   `json:"total"`
}

func (s *PromptService) List(params PromptListParams) (*PromptListResult, error) {
	var prompts []models.PromptTemplate
	var total int64

	query := s.db.Model(&models.PromptTemplate{})

	if params.Name != "" {
		query = query.Where("name LIKE ?", "%"+params.Name+"%")
	}
	if params.IsSystem != nil {
		query = query.Where("is_system = ?", *params.IsSystem)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	offset := (params.Page - 1) * params.PageSize
	if err := query.Offset(offset).Limit(params.PageSize).Order("is_system DESC, is_default DESC, id DESC").Find(&prompts).Error; err != nil {
		return nil, err
	}

	return &PromptListResult{
		Items: prompts,
		Total: total,
	}, nil
}

func (s *PromptService) GetByID(id uint) (*models.PromptTemplate, error) {
	var prompt models.PromptTemplate
	if err := s.db.First(&prompt, id).Error; err != nil {
		return nil, err
	}
	return &prompt, nil
}

func (s *PromptService) GetDefault() (*models.PromptTemplate, error) {
	var prompt models.PromptTemplate
	if err := s.db.Where("is_default = ?", true).First(&prompt).Error; err != nil {
		return nil, err
	}
	return &prompt, nil
}

func (s *PromptService) Create(prompt *models.PromptTemplate) error {
	// User-created prompts are not system prompts
	prompt.IsSystem = false
	return s.db.Create(prompt).Error
}

func (s *PromptService) Update(id uint, updates map[string]interface{}) error {
	// Check if it's a system prompt
	var prompt models.PromptTemplate
	if err := s.db.First(&prompt, id).Error; err != nil {
		return err
	}

	// System prompts cannot have their is_system flag changed
	delete(updates, "is_system")

	return s.db.Model(&models.PromptTemplate{}).Where("id = ?", id).Updates(updates).Error
}

func (s *PromptService) Delete(id uint) error {
	// Check if it's a system prompt
	var prompt models.PromptTemplate
	if err := s.db.First(&prompt, id).Error; err != nil {
		return err
	}

	if prompt.IsSystem {
		return gorm.ErrRecordNotFound // Cannot delete system prompts
	}

	return s.db.Delete(&models.PromptTemplate{}, id).Error
}

func (s *PromptService) SetDefault(id uint) error {
	// Unset current default
	if err := s.db.Model(&models.PromptTemplate{}).Where("is_default = ?", true).Update("is_default", false).Error; err != nil {
		return err
	}

	// Set new default
	return s.db.Model(&models.PromptTemplate{}).Where("id = ?", id).Update("is_default", true).Error
}

// GetAllActive returns all active prompts for selection
func (s *PromptService) GetAllActive() ([]models.PromptTemplate, error) {
	var prompts []models.PromptTemplate
	if err := s.db.Order("is_system DESC, is_default DESC, name ASC").Find(&prompts).Error; err != nil {
		return nil, err
	}
	return prompts, nil
}
