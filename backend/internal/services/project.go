package services

import (
	"errors"
	"strings"

	"github.com/huangang/codesentry/backend/internal/models"
	"gorm.io/gorm"
)

type ProjectService struct {
	db *gorm.DB
}

func NewProjectService(db *gorm.DB) *ProjectService {
	return &ProjectService{db: db}
}

type ProjectListRequest struct {
	Page     int    `form:"page" binding:"min=1"`
	PageSize int    `form:"page_size" binding:"min=1,max=100"`
	Name     string `form:"name"`
	Platform string `form:"platform"`
}

type ProjectListResponse struct {
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"page_size"`
	Items    []models.Project `json:"items"`
}

type CreateProjectRequest struct {
	Name           string `json:"name" binding:"required"`
	URL            string `json:"url" binding:"required"`
	Platform       string `json:"platform" binding:"required,oneof=github gitlab"`
	AccessToken    string `json:"access_token"`
	WebhookSecret  string `json:"webhook_secret"`
	FileExtensions string `json:"file_extensions"`
	ReviewEvents   string `json:"review_events"`
	AIEnabled      bool   `json:"ai_enabled"`
	AIPrompt       string `json:"ai_prompt"`
	IMEnabled      bool   `json:"im_enabled"`
	IMBotID        *uint  `json:"im_bot_id"`
}

type UpdateProjectRequest struct {
	Name           string `json:"name"`
	URL            string `json:"url"`
	Platform       string `json:"platform" binding:"omitempty,oneof=github gitlab"`
	AccessToken    string `json:"access_token"`
	WebhookSecret  string `json:"webhook_secret"`
	FileExtensions string `json:"file_extensions"`
	ReviewEvents   string `json:"review_events"`
	AIEnabled      *bool  `json:"ai_enabled"`
	AIPrompt       string `json:"ai_prompt"`
	IMEnabled      *bool  `json:"im_enabled"`
	IMBotID        *uint  `json:"im_bot_id"`
}

// List returns paginated projects
func (s *ProjectService) List(req *ProjectListRequest) (*ProjectListResponse, error) {
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 10
	}

	var projects []models.Project
	var total int64

	query := s.db.Model(&models.Project{})

	if req.Name != "" {
		query = query.Where("name LIKE ?", "%"+req.Name+"%")
	}
	if req.Platform != "" {
		query = query.Where("platform = ?", req.Platform)
	}

	query.Count(&total)

	offset := (req.Page - 1) * req.PageSize
	if err := query.Offset(offset).Limit(req.PageSize).Order("created_at DESC").Find(&projects).Error; err != nil {
		return nil, err
	}

	return &ProjectListResponse{
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
		Items:    projects,
	}, nil
}

// GetByID returns a project by ID
func (s *ProjectService) GetByID(id uint) (*models.Project, error) {
	var project models.Project
	if err := s.db.First(&project, id).Error; err != nil {
		return nil, err
	}
	return &project, nil
}

// Create creates a new project
func (s *ProjectService) Create(req *CreateProjectRequest, userID uint) (*models.Project, error) {
	// Set default file extensions if not provided
	if req.FileExtensions == "" {
		req.FileExtensions = ".go,.js,.ts,.jsx,.tsx,.py,.java,.c,.cpp,.h,.hpp,.cs,.rb,.php,.swift,.kt,.rs,.vue,.svelte"
	}
	if req.ReviewEvents == "" {
		req.ReviewEvents = "push,merge_request"
	}

	project := models.Project{
		Name:           req.Name,
		URL:            strings.TrimSuffix(req.URL, ".git"),
		Platform:       req.Platform,
		AccessToken:    req.AccessToken,
		WebhookSecret:  req.WebhookSecret,
		FileExtensions: req.FileExtensions,
		ReviewEvents:   req.ReviewEvents,
		AIEnabled:      req.AIEnabled,
		AIPrompt:       req.AIPrompt,
		IMEnabled:      req.IMEnabled,
		IMBotID:        req.IMBotID,
		CreatedBy:      userID,
	}

	if err := s.db.Create(&project).Error; err != nil {
		return nil, err
	}

	return &project, nil
}

// Update updates a project
func (s *ProjectService) Update(id uint, req *UpdateProjectRequest) (*models.Project, error) {
	var project models.Project
	if err := s.db.First(&project, id).Error; err != nil {
		return nil, err
	}

	updates := make(map[string]interface{})

	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.URL != "" {
		updates["url"] = strings.TrimSuffix(req.URL, ".git")
	}
	if req.Platform != "" {
		updates["platform"] = req.Platform
	}
	if req.AccessToken != "" {
		updates["access_token"] = req.AccessToken
	}
	if req.WebhookSecret != "" {
		updates["webhook_secret"] = req.WebhookSecret
	}
	if req.FileExtensions != "" {
		updates["file_extensions"] = req.FileExtensions
	}
	if req.ReviewEvents != "" {
		updates["review_events"] = req.ReviewEvents
	}
	if req.AIEnabled != nil {
		updates["ai_enabled"] = *req.AIEnabled
	}
	if req.AIPrompt != "" {
		updates["ai_prompt"] = req.AIPrompt
	}
	if req.IMEnabled != nil {
		updates["im_enabled"] = *req.IMEnabled
	}
	if req.IMBotID != nil {
		updates["im_bot_id"] = req.IMBotID
	}

	if err := s.db.Model(&project).Updates(updates).Error; err != nil {
		return nil, err
	}

	return &project, nil
}

// Delete deletes a project
func (s *ProjectService) Delete(id uint) error {
	result := s.db.Delete(&models.Project{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("project not found")
	}
	return nil
}

// GetByWebhookSecret finds a project by webhook secret
func (s *ProjectService) GetByWebhookSecret(secret string) (*models.Project, error) {
	var project models.Project
	if err := s.db.Where("webhook_secret = ?", secret).First(&project).Error; err != nil {
		return nil, err
	}
	return &project, nil
}

// GetDefaultPrompt returns the default AI review prompt
func (s *ProjectService) GetDefaultPrompt() string {
	return `你是一位资深软件开发工程师，专注于代码审查。请根据以下维度对代码变更进行评审：

## 评分维度（总分100分）
1. 功能正确性与健壮性 (40分)
2. 安全性与潜在风险 (30分)
3. 最佳实践与可维护性 (20分)
4. 性能与资源利用 (5分)
5. 提交信息质量 (5分)

## 审查规则
- 仅输出最重要的前3个问题
- 使用Markdown格式输出

## 输出格式
### 关键问题与优化建议
1. [问题描述及建议]
2. [问题描述及建议]
3. [问题描述及建议]

### 评分明细
- 功能正确性与健壮性: X/40
- 安全性与潜在风险: X/30
- 最佳实践与可维护性: X/20
- 性能与资源利用: X/5
- 提交信息质量: X/5

### 总分: X分

---
代码变更内容:
{{diffs}}

提交信息:
{{commits}}`
}
