package services

import (
	"strings"

	"github.com/huangang/codesentry/backend/internal/models"
	"gorm.io/gorm"
)

type GitCredentialService struct {
	db *gorm.DB
}

func NewGitCredentialService(db *gorm.DB) *GitCredentialService {
	return &GitCredentialService{db: db}
}

type GitCredentialListParams struct {
	Page     int
	PageSize int
	Name     string
	Platform string
	IsActive *bool
}

func (s *GitCredentialService) List(params *GitCredentialListParams) ([]models.GitCredential, int64, error) {
	var credentials []models.GitCredential
	var total int64

	query := s.db.Model(&models.GitCredential{})

	if params.Name != "" {
		query = query.Where("name LIKE ?", "%"+params.Name+"%")
	}
	if params.Platform != "" {
		query = query.Where("platform = ?", params.Platform)
	}
	if params.IsActive != nil {
		query = query.Where("is_active = ?", *params.IsActive)
	}

	query.Count(&total)

	if params.PageSize <= 0 {
		params.PageSize = 20
	}
	if params.Page <= 0 {
		params.Page = 1
	}

	offset := (params.Page - 1) * params.PageSize
	err := query.Order("id DESC").Offset(offset).Limit(params.PageSize).Find(&credentials).Error
	return credentials, total, err
}

func (s *GitCredentialService) GetByID(id uint) (*models.GitCredential, error) {
	var credential models.GitCredential
	err := s.db.First(&credential, id).Error
	return &credential, err
}

func (s *GitCredentialService) Create(credential *models.GitCredential) error {
	return s.db.Create(credential).Error
}

func (s *GitCredentialService) Update(credential *models.GitCredential) error {
	return s.db.Save(credential).Error
}

func (s *GitCredentialService) Delete(id uint) error {
	return s.db.Delete(&models.GitCredential{}, id).Error
}

func (s *GitCredentialService) GetActive() ([]models.GitCredential, error) {
	var credentials []models.GitCredential
	err := s.db.Where("is_active = ?", true).Order("id DESC").Find(&credentials).Error
	return credentials, err
}

func (s *GitCredentialService) FindMatchingCredential(projectURL string, platform string) (*models.GitCredential, error) {
	projectURL = strings.TrimSuffix(projectURL, ".git")
	projectURL = strings.ToLower(projectURL)

	var credentials []models.GitCredential
	err := s.db.Where("platform = ? AND is_active = ? AND auto_create = ?", platform, true, true).
		Order("id DESC").Find(&credentials).Error
	if err != nil {
		return nil, err
	}

	for _, cred := range credentials {
		baseURL := strings.ToLower(strings.TrimSuffix(cred.BaseURL, "/"))
		if baseURL == "" {
			switch platform {
			case "github":
				baseURL = "https://github.com"
			case "gitlab":
				baseURL = "https://gitlab.com"
			}
		}

		if strings.HasPrefix(projectURL, baseURL) {
			return &cred, nil
		}
	}

	return nil, nil
}
