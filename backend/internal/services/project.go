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
	Page     int    `form:"page" binding:"omitempty,min=1"`
	PageSize int    `form:"page_size" binding:"omitempty,min=1,max=100"`
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
	Name           string  `json:"name" binding:"required"`
	URL            string  `json:"url" binding:"required"`
	Platform       string  `json:"platform" binding:"required,oneof=github gitlab"`
	AccessToken    string  `json:"access_token"`
	WebhookSecret  string  `json:"webhook_secret"`
	FileExtensions string  `json:"file_extensions"`
	ReviewEvents   string  `json:"review_events"`
	AIEnabled      bool    `json:"ai_enabled"`
	AIPrompt       string  `json:"ai_prompt"`
	IMEnabled      bool    `json:"im_enabled"`
	IMBotID        *uint   `json:"im_bot_id"`
	MinScore       float64 `json:"min_score"`
}

type UpdateProjectRequest struct {
	Name           string   `json:"name"`
	URL            string   `json:"url"`
	Platform       string   `json:"platform" binding:"omitempty,oneof=github gitlab"`
	AccessToken    string   `json:"access_token"`
	WebhookSecret  string   `json:"webhook_secret"`
	FileExtensions string   `json:"file_extensions"`
	ReviewEvents   string   `json:"review_events"`
	AIEnabled      *bool    `json:"ai_enabled"`
	AIPromptID     *uint    `json:"ai_prompt_id"`
	AIPrompt       *string  `json:"ai_prompt"`
	LLMConfigID    *uint    `json:"llm_config_id"`
	IgnorePatterns *string  `json:"ignore_patterns"`
	CommentEnabled *bool    `json:"comment_enabled"`
	IMEnabled      *bool    `json:"im_enabled"`
	IMBotID        *uint    `json:"im_bot_id"`
	MinScore       *float64 `json:"min_score"`
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
		MinScore:       req.MinScore,
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
	if req.AIPromptID != nil {
		updates["a_iprompt_id"] = req.AIPromptID
	}
	if req.AIPrompt != nil {
		updates["a_iprompt"] = *req.AIPrompt
	}
	if req.LLMConfigID != nil {
		updates["llm_config_id"] = req.LLMConfigID
	}
	if req.IgnorePatterns != nil {
		updates["ignore_patterns"] = *req.IgnorePatterns
	}
	if req.CommentEnabled != nil {
		updates["comment_enabled"] = *req.CommentEnabled
	}
	if req.IMEnabled != nil {
		updates["im_enabled"] = *req.IMEnabled
	}
	if req.IMBotID != nil {
		updates["im_bot_id"] = req.IMBotID
	}
	if req.MinScore != nil {
		updates["min_score"] = *req.MinScore
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

func (s *ProjectService) GetByURL(url string) (*models.Project, error) {
	url = strings.TrimSuffix(url, ".git")
	url = strings.TrimSuffix(url, "/")

	var project models.Project
	if err := s.db.Where("url = ? OR url = ?", url, url+".git").First(&project).Error; err != nil {
		return nil, err
	}
	return &project, nil
}

type CreateProjectParams struct {
	Name           string
	URL            string
	Platform       string
	AccessToken    string
	WebhookSecret  string
	AIEnabled      bool
	FileExtensions string
	ReviewEvents   string
	IgnorePatterns string
}

func (s *ProjectService) CreateFromCredential(params *CreateProjectParams) (*models.Project, error) {
	if params.FileExtensions == "" {
		params.FileExtensions = ".go,.js,.ts,.jsx,.tsx,.py,.java,.c,.cpp,.h,.hpp,.cs,.rb,.php,.swift,.kt,.rs,.vue,.svelte"
	}
	if params.ReviewEvents == "" {
		params.ReviewEvents = "push,merge_request"
	}

	project := models.Project{
		Name:           params.Name,
		URL:            strings.TrimSuffix(params.URL, ".git"),
		Platform:       params.Platform,
		AccessToken:    params.AccessToken,
		WebhookSecret:  params.WebhookSecret,
		FileExtensions: params.FileExtensions,
		ReviewEvents:   params.ReviewEvents,
		IgnorePatterns: params.IgnorePatterns,
		AIEnabled:      params.AIEnabled,
		CreatedBy:      0,
	}

	if err := s.db.Create(&project).Error; err != nil {
		return nil, err
	}

	return &project, nil
}

func (s *ProjectService) FillFromCredential(project *models.Project, credential *models.GitCredential) error {
	updates := make(map[string]interface{})

	if project.AccessToken == "" && credential.AccessToken != "" {
		updates["access_token"] = credential.AccessToken
		project.AccessToken = credential.AccessToken
	}
	if project.WebhookSecret == "" && credential.WebhookSecret != "" {
		updates["webhook_secret"] = credential.WebhookSecret
		project.WebhookSecret = credential.WebhookSecret
	}
	if project.FileExtensions == "" && credential.FileExtensions != "" {
		updates["file_extensions"] = credential.FileExtensions
		project.FileExtensions = credential.FileExtensions
	}
	if project.ReviewEvents == "" && credential.ReviewEvents != "" {
		updates["review_events"] = credential.ReviewEvents
		project.ReviewEvents = credential.ReviewEvents
	}
	if project.IgnorePatterns == "" && credential.IgnorePatterns != "" {
		updates["ignore_patterns"] = credential.IgnorePatterns
		project.IgnorePatterns = credential.IgnorePatterns
	}

	if len(updates) > 0 {
		return s.db.Model(project).Updates(updates).Error
	}
	return nil
}

// GetDefaultPrompt returns the default AI review prompt
func (s *ProjectService) GetDefaultPrompt() string {
	return s.GetDefaultPromptByLang("zh")
}

func (s *ProjectService) GetDefaultPromptByLang(lang string) string {
	if lang == "en" {
		return `You are a senior software engineer focused on code correctness, security, stability, and engineering best practices. Your task is to provide professional, restrained, and high-value code reviews.

## Scoring Dimensions (Total: 100 points)
1. **Functional Correctness & Robustness (40 points)**: Is the logic correct? Are edge cases and exceptions handled properly?
2. **Security & Potential Risks (30 points)**: Are there security vulnerabilities (SQL injection, XSS, privilege escalation, sensitive data exposure, etc.)?
3. **Best Practices & Maintainability (20 points)**: Does it follow mainstream engineering best practices (structure, naming, readability, comments)?
4. **Performance & Resource Utilization (5 points)**: Are there obvious performance bottlenecks or unnecessary resource waste?
5. **Commit Message Quality (5 points)**: Are commit messages clear, accurate, and traceable?

## Important Rules (Must Follow Strictly)
- **Only focus on and output the top 3 most important issues**. No more than 3.
- If fewer than 3 issues exist, output only the actual number found.
- **When file context is provided, use it to understand the full picture before judging the code changes.**

## Output Format (Markdown)
Please strictly follow this structure:

### 1. Key Issues & Suggestions (Top 3 Only)
- Rank by importance (Issue 1 is most critical).
- Each issue must include: Problem description, Impact analysis, Optimization suggestion, Code example if necessary.

### 2. Score Breakdown
- Provide specific scores for each of the 5 dimensions with brief reasoning.

### 3. Total Score (Critical)
- Format must be: "Total Score: XX/100" (e.g., Total Score: 80/100).
- Must be parseable by regex pattern: r"[Tt]otal\s*[Ss]core[:：]?\s*(\d+)"

---
{{file_context}}
**Code Changes**:
{{diffs}}

**Commit History**:
{{commits}}`
	}

	return `你是一位资深的软件开发工程师，专注于代码的功能正确性、安全性、稳定性以及工程最佳实践。你的任务是对提交的代码进行专业、克制且高价值的代码审查。

## 评分维度（总分100分）
1. **功能实现的正确性与健壮性（40分）**：逻辑是否正确，是否能正确处理边界情况与异常场景。
2. **安全性与潜在风险（30分）**：是否存在安全隐患（如 SQL 注入、XSS、越权、敏感信息泄露等）。
3. **最佳实践与可维护性（20分）**：是否符合主流工程最佳实践（结构、命名、可读性、注释）。
4. **性能与资源利用（5分）**：是否存在明显性能瓶颈或不必要的资源浪费。
5. **提交信息质量（5分）**：commit 信息是否清晰、准确、可追溯。

## 重要规则（必须严格遵守）
- 请**仅关注并输出最重要的前三个问题（Top 3）**，不得多于 3 个。
- 若问题不足 3 个，则按实际数量输出。
- **当提供了完整文件上下文时，请结合上下文理解代码变更的完整背景，避免仅根据 diff 片段做出片面判断。**

## 输出格式（Markdown）
请严格按照以下结构输出：

### 一、关键问题与优化建议（仅限 Top 3）
- 按重要性从高到低排序（问题 1 最重要）。
- 每个问题需包含：问题描述、影响分析、优化建议，如果必要，给出代码示例。

### 二、评分明细
- 按五个评分维度分别给出具体分数，并简要说明理由。

### 三、总分（特别重要）
- 格式必须为："总分:XX分"（例如：总分:80分）。
- 必须确保可通过正则表达式 r"总分[:：]\s*(\d+)分?" 正确解析出总分值。

---
{{file_context}}
**代码变更内容**：
{{diffs}}

**提交历史（commits）**：
{{commits}}`
}
