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
		&Project{},
		&ReviewLog{},
		&LLMConfig{},
		&PromptTemplate{},
		&SystemConfig{},
		&IMBot{},
		&SystemLog{},
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
			Content: `你是一位资深软件开发工程师，专注于代码审查。请根据以下维度对代码变更进行评审：

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
			Content: `You are a senior software engineer specializing in code review. Please review the code changes based on the following dimensions:

## Scoring Dimensions (Total: 100 points)
1. Functional Correctness & Robustness (40 points)
2. Security & Potential Risks (30 points)
3. Best Practices & Maintainability (20 points)
4. Performance & Resource Utilization (5 points)
5. Commit Message Quality (5 points)

## Review Rules
- Only output the top 3 most important issues
- Use Markdown format for output

## Output Format
### Key Issues & Suggestions
1. [Issue description and suggestion]
2. [Issue description and suggestion]
3. [Issue description and suggestion]

### Score Breakdown
- Functional Correctness & Robustness: X/40
- Security & Potential Risks: X/30
- Best Practices & Maintainability: X/20
- Performance & Resource Utilization: X/5
- Commit Message Quality: X/5

### Total Score: X/100

---
Code Changes:
{{diffs}}

Commit Messages:
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
	}

	for _, cfg := range defaultConfigs {
		var count int64
		DB.Model(&SystemConfig{}).Where("key = ?", cfg.Key).Count(&count)
		if count == 0 {
			if err := DB.Create(&cfg).Error; err != nil {
				return err
			}
		}
	}

	return nil
}
