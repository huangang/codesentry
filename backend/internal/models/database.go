package models

import (
	"fmt"

	"github.com/huangang/codesentry/backend/internal/config"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func InitDB(cfg *config.DatabaseConfig) error {
	var dialector gorm.Dialector

	switch cfg.Driver {
	case "sqlite":
		dialector = sqlite.Open(cfg.DSN)
	case "mysql":
		dialector = mysql.Open(cfg.DSN)
	case "postgres":
		dialector = postgres.Open(cfg.DSN)
	default:
		return fmt.Errorf("unsupported database driver: %s", cfg.Driver)
	}

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}

	db, err := gorm.Open(dialector, gormConfig)
	if err != nil {
		return fmt.Errorf("failed to connect database: %w", err)
	}

	DB = db
	return nil
}

func AutoMigrate() error {
	return DB.AutoMigrate(
		&User{},
		&RefreshToken{},
		&Project{},
		&ReviewLog{},
		&LLMConfig{},
		&PromptTemplate{},
		&SystemConfig{},
		&IMBot{},
		&SystemLog{},
		&GitCredential{},
		&DailyReport{},
		&SchedulerLock{},
		&ReviewTemplate{},
		&ReviewFeedback{},
		&AIUsageLog{},
		&ProjectMember{},
		&IssueTracker{},
		&ReviewRule{},
	)
}

func GetDB() *gorm.DB {
	return DB
}

// SeedDefaultData creates default data if not exists
func SeedDefaultData() error {
	// Create default prompt templates (Chinese and English)
	var promptCount int64
	DB.Model(&PromptTemplate{}).Where("is_system = ?", true).Count(&promptCount)
	if promptCount == 0 {
		defaultPromptZh := PromptTemplate{
			Name:        "Default Code Review (Chinese)",
			Description: "Default AI code review prompt with scoring (Chinese)",
			Content: `你是一位资深的软件开发工程师，专注于代码的功能正确性、安全性、稳定性以及工程最佳实践。你的任务是对提交的代码进行专业、克制且高价值的代码审查。

## 评分维度（总分100分）
1. **功能实现的正确性与健壮性（40分）**：逻辑是否正确，是否能正确处理边界情况与异常场景。
2. **安全性与潜在风险（30分）**：是否存在安全隐患（如 SQL 注入、XSS、越权、敏感信息泄露等）。
3. **最佳实践与可维护性（20分）**：是否符合主流工程最佳实践（结构、命名、可读性、注释）。
4. **性能与资源利用（5分）**：是否存在明显性能瓶颈或不必要的资源浪费。
5. **提交信息质量（5分）**：commit 信息是否清晰、准确、可追溯。

## 重要规则（必须严格遵守）
- 请**仅关注并输出最重要的前三个问题（Top 3）**，不得多于 3 个。
- 若问题不足 3 个，则按实际数量输出。

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
**代码变更内容**：
{{diffs}}

**提交历史（commits）**：
{{commits}}`,
			Variables: `["diffs", "commits"]`,
			IsDefault: true,
			IsSystem:  true,
		}
		if err := DB.Create(&defaultPromptZh).Error; err != nil {
			return err
		}

		defaultPromptEn := PromptTemplate{
			Name:        "Default Code Review (English)",
			Description: "Default AI code review prompt with scoring (English)",
			Content: `You are a senior software engineer focused on code correctness, security, stability, and engineering best practices. Your task is to provide professional, restrained, and high-value code reviews.

## Scoring Dimensions (Total: 100 points)
1. **Functional Correctness & Robustness (40 points)**: Is the logic correct? Are edge cases and exceptions handled properly?
2. **Security & Potential Risks (30 points)**: Are there security vulnerabilities (SQL injection, XSS, privilege escalation, sensitive data exposure, etc.)?
3. **Best Practices & Maintainability (20 points)**: Does it follow mainstream engineering best practices (structure, naming, readability, comments)?
4. **Performance & Resource Utilization (5 points)**: Are there obvious performance bottlenecks or unnecessary resource waste?
5. **Commit Message Quality (5 points)**: Are commit messages clear, accurate, and traceable?

## Important Rules (Must Follow Strictly)
- **Only focus on and output the top 3 most important issues**. No more than 3.
- If fewer than 3 issues exist, output only the actual number found.

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
**Code Changes**:
{{diffs}}

**Commit History**:
{{commits}}`,
			Variables: `["diffs", "commits"]`,
			IsDefault: false,
			IsSystem:  true,
		}
		if err := DB.Create(&defaultPromptEn).Error; err != nil {
			return err
		}
	}

	// Create default system configs
	defaultConfigs := []SystemConfig{
		{Key: "ldap_enabled", Value: "false", Type: "bool", Group: "ldap", Label: "Enable LDAP Authentication"},
		{Key: "ldap_host", Value: "", Type: "string", Group: "ldap", Label: "LDAP Server Host"},
		{Key: "ldap_port", Value: "389", Type: "int", Group: "ldap", Label: "LDAP Server Port"},
		{Key: "ldap_base_dn", Value: "", Type: "string", Group: "ldap", Label: "LDAP Base DN"},
		{Key: "ldap_bind_dn", Value: "", Type: "string", Group: "ldap", Label: "LDAP Bind DN"},
		{Key: "ldap_bind_password", Value: "", Type: "string", Group: "ldap", Label: "LDAP Bind Password"},
		{Key: "ldap_user_filter", Value: "(uid=%s)", Type: "string", Group: "ldap", Label: "LDAP User Filter"},
		{Key: "ldap_use_ssl", Value: "false", Type: "bool", Group: "ldap", Label: "Use SSL/TLS"},
		{Key: "log_retention_days", Value: "30", Type: "int", Group: "system", Label: "System Log Retention Days"},
		{Key: "daily_report_enabled", Value: "false", Type: "bool", Group: "daily_report", Label: "Enable Daily Report"},
		{Key: "daily_report_time", Value: "18:00", Type: "string", Group: "daily_report", Label: "Daily Report Time"},
		{Key: "daily_report_low_score", Value: "60", Type: "int", Group: "daily_report", Label: "Low Score Threshold"},
		{Key: "auth_access_token_expire_hours", Value: "2", Type: "int", Group: "auth", Label: "Access Token Expire Hours"},
		{Key: "auth_refresh_token_expire_hours", Value: "720", Type: "int", Group: "auth", Label: "Refresh Token Expire Hours"},
	}

	for _, cfg := range defaultConfigs {
		var count int64
		DB.Model(&SystemConfig{}).Where("`key` = ?", cfg.Key).Count(&count)
		if count == 0 {
			if err := DB.Create(&cfg).Error; err != nil {
				return err
			}
		}
	}

	return nil
}
