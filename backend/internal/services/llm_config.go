package services

import (
	"errors"

	"github.com/huangang/codesentry/backend/internal/models"
	"gorm.io/gorm"
)

type LLMConfigService struct {
	db *gorm.DB
}

func NewLLMConfigService(db *gorm.DB) *LLMConfigService {
	return &LLMConfigService{db: db}
}

type LLMConfigListRequest struct {
	Page     int    `form:"page" binding:"min=1"`
	PageSize int    `form:"page_size" binding:"min=1,max=100"`
	Name     string `form:"name"`
	Provider string `form:"provider"`
	IsActive *bool  `form:"is_active"`
}

type LLMConfigListResponse struct {
	Total    int64              `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"page_size"`
	Items    []models.LLMConfig `json:"items"`
}

type CreateLLMConfigRequest struct {
	Name        string  `json:"name" binding:"required"`
	Provider    string  `json:"provider"`
	BaseURL     string  `json:"base_url" binding:"required"`
	APIKey      string  `json:"api_key" binding:"required"`
	Model       string  `json:"model" binding:"required"`
	MaxTokens   int     `json:"max_tokens"`
	Temperature float64 `json:"temperature"`
	IsDefault   bool    `json:"is_default"`
	IsActive    bool    `json:"is_active"`
}

type UpdateLLMConfigRequest struct {
	Name        string   `json:"name"`
	Provider    string   `json:"provider"`
	BaseURL     string   `json:"base_url"`
	APIKey      string   `json:"api_key"`
	Model       string   `json:"model"`
	MaxTokens   *int     `json:"max_tokens"`
	Temperature *float64 `json:"temperature"`
	IsDefault   *bool    `json:"is_default"`
	IsActive    *bool    `json:"is_active"`
}

// List returns paginated LLM configs
func (s *LLMConfigService) List(req *LLMConfigListRequest) (*LLMConfigListResponse, error) {
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 10
	}

	var configs []models.LLMConfig
	var total int64

	query := s.db.Model(&models.LLMConfig{})

	if req.Name != "" {
		query = query.Where("name LIKE ? OR model LIKE ?", "%"+req.Name+"%", "%"+req.Name+"%")
	}
	if req.Provider != "" {
		query = query.Where("provider = ?", req.Provider)
	}
	if req.IsActive != nil {
		query = query.Where("is_active = ?", *req.IsActive)
	}

	query.Count(&total)

	offset := (req.Page - 1) * req.PageSize
	if err := query.Offset(offset).Limit(req.PageSize).Order("created_at DESC").Find(&configs).Error; err != nil {
		return nil, err
	}

	// Mask API keys for response
	for i := range configs {
		configs[i].APIKeyMask = configs[i].MaskAPIKey()
	}

	return &LLMConfigListResponse{
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
		Items:    configs,
	}, nil
}

// GetByID returns a LLM config by ID
func (s *LLMConfigService) GetByID(id uint) (*models.LLMConfig, error) {
	var config models.LLMConfig
	if err := s.db.First(&config, id).Error; err != nil {
		return nil, err
	}
	config.APIKeyMask = config.MaskAPIKey()
	return &config, nil
}

// GetDefault returns the default LLM config
func (s *LLMConfigService) GetDefault() (*models.LLMConfig, error) {
	var config models.LLMConfig
	if err := s.db.Where("is_default = ? AND is_active = ?", true, true).First(&config).Error; err != nil {
		// If no default, get any active config
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := s.db.Where("is_active = ?", true).First(&config).Error; err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	return &config, nil
}

// Create creates a new LLM config
func (s *LLMConfigService) Create(req *CreateLLMConfigRequest) (*models.LLMConfig, error) {
	if req.Provider == "" {
		req.Provider = "openai"
	}
	if req.MaxTokens == 0 {
		req.MaxTokens = 4096
	}
	if req.Temperature == 0 {
		req.Temperature = 0.3
	}

	config := models.LLMConfig{
		Name:        req.Name,
		Provider:    req.Provider,
		BaseURL:     req.BaseURL,
		APIKey:      req.APIKey,
		Model:       req.Model,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		IsDefault:   req.IsDefault,
		IsActive:    req.IsActive,
	}

	// If this is set as default, unset other defaults
	if req.IsDefault {
		s.db.Model(&models.LLMConfig{}).Where("is_default = ?", true).Update("is_default", false)
	}

	if err := s.db.Create(&config).Error; err != nil {
		return nil, err
	}

	config.APIKeyMask = config.MaskAPIKey()
	return &config, nil
}

// Update updates a LLM config
func (s *LLMConfigService) Update(id uint, req *UpdateLLMConfigRequest) (*models.LLMConfig, error) {
	var config models.LLMConfig
	if err := s.db.First(&config, id).Error; err != nil {
		return nil, err
	}

	updates := make(map[string]interface{})

	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Provider != "" {
		updates["provider"] = req.Provider
	}
	if req.BaseURL != "" {
		updates["base_url"] = req.BaseURL
	}
	if req.APIKey != "" {
		updates["api_key"] = req.APIKey
	}
	if req.Model != "" {
		updates["model"] = req.Model
	}
	if req.MaxTokens != nil {
		updates["max_tokens"] = *req.MaxTokens
	}
	if req.Temperature != nil {
		updates["temperature"] = *req.Temperature
	}
	if req.IsDefault != nil {
		if *req.IsDefault {
			// Unset other defaults
			s.db.Model(&models.LLMConfig{}).Where("is_default = ? AND id != ?", true, id).Update("is_default", false)
		}
		updates["is_default"] = *req.IsDefault
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	if err := s.db.Model(&config).Updates(updates).Error; err != nil {
		return nil, err
	}

	// Reload
	s.db.First(&config, id)
	config.APIKeyMask = config.MaskAPIKey()
	return &config, nil
}

// Delete deletes a LLM config
func (s *LLMConfigService) Delete(id uint) error {
	result := s.db.Delete(&models.LLMConfig{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("llm config not found")
	}
	return nil
}

func (s *LLMConfigService) GetActive() ([]models.LLMConfig, error) {
	var configs []models.LLMConfig
	if err := s.db.Where("is_active = ?", true).Order("is_default DESC, created_at DESC").Find(&configs).Error; err != nil {
		return nil, err
	}
	for i := range configs {
		configs[i].APIKeyMask = configs[i].MaskAPIKey()
	}
	return configs, nil
}
