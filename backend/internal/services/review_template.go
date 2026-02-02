package services

import (
	"github.com/huangang/codesentry/backend/internal/models"
	"gorm.io/gorm"
)

type ReviewTemplateService struct {
	db *gorm.DB
}

func NewReviewTemplateService(db *gorm.DB) *ReviewTemplateService {
	return &ReviewTemplateService{db: db}
}

// List returns all active review templates
func (s *ReviewTemplateService) List(templateType string) ([]models.ReviewTemplate, error) {
	var templates []models.ReviewTemplate
	query := s.db.Where("is_active = ?", true)
	if templateType != "" {
		query = query.Where("type = ?", templateType)
	}
	err := query.Order("is_built_in DESC, created_at DESC").Find(&templates).Error
	return templates, err
}

// GetByID returns a template by ID
func (s *ReviewTemplateService) GetByID(id uint) (*models.ReviewTemplate, error) {
	var template models.ReviewTemplate
	err := s.db.First(&template, id).Error
	return &template, err
}

// Create creates a new template
func (s *ReviewTemplateService) Create(template *models.ReviewTemplate) error {
	return s.db.Create(template).Error
}

// Update updates a template (built-in templates cannot be modified except IsActive)
func (s *ReviewTemplateService) Update(template *models.ReviewTemplate) error {
	existing, err := s.GetByID(template.ID)
	if err != nil {
		return err
	}

	// Built-in templates can only toggle IsActive
	if existing.IsBuiltIn {
		return s.db.Model(template).Update("is_active", template.IsActive).Error
	}

	return s.db.Save(template).Error
}

// Delete soft-deletes a template (built-in templates cannot be deleted)
func (s *ReviewTemplateService) Delete(id uint) error {
	template, err := s.GetByID(id)
	if err != nil {
		return err
	}

	if template.IsBuiltIn {
		return ErrBuiltInTemplate
	}

	return s.db.Delete(&models.ReviewTemplate{}, id).Error
}

// GetByType returns templates by type
func (s *ReviewTemplateService) GetByType(templateType string) ([]models.ReviewTemplate, error) {
	var templates []models.ReviewTemplate
	err := s.db.Where("type = ? AND is_active = ?", templateType, true).Find(&templates).Error
	return templates, err
}

// SeedDefaultTemplates creates built-in templates if they don't exist
func (s *ReviewTemplateService) SeedDefaultTemplates() error {
	var count int64
	s.db.Model(&models.ReviewTemplate{}).Where("is_built_in = ?", true).Count(&count)
	if count > 0 {
		return nil // Already seeded
	}

	templates := []models.ReviewTemplate{
		{
			Name:        "通用代码审查",
			Type:        "general",
			Description: "适用于各类项目的通用代码审查模板",
			Content: `你是一位资深的软件开发工程师。请对以下代码变更进行审查，关注：
1. 代码正确性和逻辑问题
2. 安全漏洞
3. 性能问题
4. 代码可读性和最佳实践

请给出0-100分的评分，格式：总分:XX分

**代码变更**：
{{diffs}}

**提交信息**：
{{commits}}`,
			IsBuiltIn: true,
			IsActive:  true,
		},
		{
			Name:        "前端代码审查",
			Type:        "frontend",
			Description: "针对前端项目的专业审查模板（React/Vue/Angular）",
			Content: `你是一位资深的前端开发工程师。请对以下前端代码变更进行审查，重点关注：

## 审查维度
1. **组件设计（25分）**：组件职责单一、可复用性、状态管理合理性
2. **性能优化（25分）**：不必要的重渲染、大组件拆分、懒加载使用
3. **用户体验（20分）**：交互反馈、错误处理、加载状态
4. **代码质量（20分）**：TypeScript类型安全、代码规范、可读性
5. **安全性（10分）**：XSS防护、敏感数据处理

请针对最重要的3个问题给出建议，并给出评分。
格式：总分:XX分

**代码变更**：
{{diffs}}

**提交信息**：
{{commits}}`,
			IsBuiltIn: true,
			IsActive:  true,
		},
		{
			Name:        "后端代码审查",
			Type:        "backend",
			Description: "针对后端项目的专业审查模板（Go/Java/Python）",
			Content: `你是一位资深的后端开发工程师。请对以下后端代码变更进行审查，重点关注：

## 审查维度
1. **业务逻辑（25分）**：逻辑正确性、边界处理、错误处理
2. **安全性（25分）**：SQL注入、权限控制、敏感信息处理、输入验证
3. **性能（20分）**：数据库查询优化、缓存使用、并发处理
4. **可维护性（20分）**：代码结构、命名规范、注释完整性
5. **可测试性（10分）**：单元测试覆盖、依赖注入

请针对最重要的3个问题给出建议，并给出评分。
格式：总分:XX分

**代码变更**：
{{diffs}}

**提交信息**：
{{commits}}`,
			IsBuiltIn: true,
			IsActive:  true,
		},
		{
			Name:        "安全代码审查",
			Type:        "security",
			Description: "专注于安全漏洞检测的审查模板",
			Content: `你是一位资深的安全工程师。请对以下代码变更进行安全审查，重点关注：

## 安全审查维度
1. **注入攻击（30分）**：SQL注入、命令注入、LDAP注入、XPath注入
2. **认证与授权（25分）**：权限验证、会话管理、密码处理
3. **敏感数据（20分）**：密钥硬编码、日志敏感信息、数据加密
4. **输入验证（15分）**：XSS、CSRF、文件上传、参数校验
5. **依赖安全（10分）**：已知漏洞组件、不安全的API调用

请列出发现的所有安全问题，按严重程度排序（高/中/低）。
格式：总分:XX分

**代码变更**：
{{diffs}}

**提交信息**：
{{commits}}`,
			IsBuiltIn: true,
			IsActive:  true,
		},
	}

	for _, t := range templates {
		if err := s.db.Create(&t).Error; err != nil {
			return err
		}
	}

	return nil
}

// Custom error
var ErrBuiltInTemplate = &BuiltInTemplateError{}

type BuiltInTemplateError struct{}

func (e *BuiltInTemplateError) Error() string {
	return "built-in templates cannot be deleted"
}
