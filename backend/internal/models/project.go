package models

import (
	"time"

	"gorm.io/gorm"
)

// Project represents a code repository project
type Project struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	Name           string         `gorm:"size:200;not null" json:"name"`
	URL            string         `gorm:"size:500;not null" json:"url"`
	Platform       string         `gorm:"size:50;not null" json:"platform"` // github, gitlab
	AccessToken    string         `gorm:"size:500" json:"-"`
	WebhookSecret  string         `gorm:"size:255" json:"-"`
	FileExtensions string         `gorm:"size:1000" json:"file_extensions"` // .js,.ts,.go,...
	ReviewEvents   string         `gorm:"size:200" json:"review_events"`    // push,merge_request
	BranchFilter   string         `gorm:"size:1000" json:"branch_filter"`   // Branches to ignore: main,master,release/*
	AIEnabled      bool           `gorm:"column:ai_enabled;default:true" json:"ai_enabled"`
	AIPromptID     *uint          `gorm:"column:a_iprompt_id" json:"ai_prompt_id"`     // Reference to PromptTemplate
	AIPrompt       string         `gorm:"column:a_iprompt;type:text" json:"ai_prompt"` // Custom prompt override
	LLMConfigID    *uint          `gorm:"column:llm_config_id" json:"llm_config_id"`   // Reference to LLMConfig
	IgnorePatterns string         `gorm:"size:2000" json:"ignore_patterns"`            // Patterns to ignore: vendor/,node_modules/,*.min.js
	CommentEnabled bool           `gorm:"default:false" json:"comment_enabled"`
	IMEnabled      bool           `gorm:"default:false" json:"im_enabled"`
	IMBotID        *uint          `json:"im_bot_id"`
	MinScore       float64        `gorm:"default:0" json:"min_score"` // Minimum score to pass (0 = use system default)
	CreatedBy      uint           `json:"created_by"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Project) TableName() string { return "projects" }
