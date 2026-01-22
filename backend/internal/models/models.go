package models

import (
	"time"

	"gorm.io/gorm"
)

// User represents a system user
type User struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Username  string         `gorm:"uniqueIndex;size:100;not null" json:"username"`
	Password  string         `gorm:"size:255" json:"-"` // Hashed password, empty for LDAP users
	Email     string         `gorm:"size:255" json:"email"`
	Nickname  string         `gorm:"size:100" json:"nickname"`
	Avatar    string         `gorm:"size:500" json:"avatar"`
	Role      string         `gorm:"size:50;default:user" json:"role"`       // admin, user
	AuthType  string         `gorm:"size:20;default:local" json:"auth_type"` // local, ldap
	IsActive  bool           `gorm:"default:true" json:"is_active"`
	LastLogin *time.Time     `json:"last_login"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

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
	AIEnabled      bool           `gorm:"default:true" json:"ai_enabled"`
	AIPromptID     *uint          `json:"ai_prompt_id"`                     // Reference to PromptTemplate
	AIPrompt       string         `gorm:"type:text" json:"ai_prompt"`       // Custom prompt override
	LLMConfigID    *uint          `json:"llm_config_id"`                    // Reference to LLMConfig
	IgnorePatterns string         `gorm:"size:2000" json:"ignore_patterns"` // Patterns to ignore: vendor/,node_modules/,*.min.js
	CommentEnabled bool           `gorm:"default:false" json:"comment_enabled"`
	IMEnabled      bool           `gorm:"default:false" json:"im_enabled"`
	IMBotID        *uint          `json:"im_bot_id"`
	CreatedBy      uint           `json:"created_by"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

// ReviewLog represents a code review record
type ReviewLog struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	ProjectID     uint           `gorm:"index;not null" json:"project_id"`
	Project       *Project       `gorm:"foreignKey:ProjectID" json:"project,omitempty"`
	EventType     string         `gorm:"size:50;not null" json:"event_type"` // push, merge_request
	CommitHash    string         `gorm:"size:100" json:"commit_hash"`
	CommitURL     string         `gorm:"size:500" json:"commit_url"`
	Branch        string         `gorm:"size:200" json:"branch"`
	Author        string         `gorm:"size:200" json:"author"`
	AuthorEmail   string         `gorm:"size:255" json:"author_email"`
	AuthorAvatar  string         `gorm:"size:500" json:"author_avatar"`
	AuthorURL     string         `gorm:"size:500" json:"author_url"`
	CommitMessage string         `gorm:"type:text" json:"commit_message"`
	FilesChanged  int            `json:"files_changed"`
	Additions     int            `json:"additions"`
	Deletions     int            `json:"deletions"`
	Score         *float64       `json:"score"`
	ReviewResult  string         `gorm:"type:text" json:"review_result"`
	ReviewStatus  string         `gorm:"size:50;default:pending" json:"review_status"` // pending, completed, failed
	ErrorMessage  string         `gorm:"type:text" json:"error_message"`
	RetryCount    int            `gorm:"default:0" json:"retry_count"`
	LLMConfigID   *uint          `json:"llm_config_id"` // Which LLM was used
	MRNumber      *int           `json:"mr_number"`     // Merge Request number
	MRURL         string         `gorm:"size:500" json:"mr_url"`
	CreatedAt     time.Time      `gorm:"index" json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

// LLMConfig represents a large language model configuration (stored in database)
type LLMConfig struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"size:100;not null" json:"name"`
	Provider    string         `gorm:"size:50;default:openai" json:"provider"` // openai, azure, anthropic, etc.
	BaseURL     string         `gorm:"size:500;not null" json:"base_url"`
	APIKey      string         `gorm:"size:500" json:"-"`
	APIKeyMask  string         `gorm:"-" json:"api_key_mask"` // For display only
	Model       string         `gorm:"size:100" json:"model"`
	MaxTokens   int            `gorm:"default:4096" json:"max_tokens"`
	Temperature float64        `gorm:"default:0.3" json:"temperature"`
	IsDefault   bool           `gorm:"default:false" json:"is_default"`
	IsActive    bool           `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// PromptTemplate represents a reusable AI prompt template (stored in database)
type PromptTemplate struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"size:100;not null" json:"name"`
	Description string         `gorm:"size:500" json:"description"`
	Content     string         `gorm:"type:text;not null" json:"content"`
	Variables   string         `gorm:"size:500" json:"variables"` // JSON array: ["diffs", "commits"]
	IsDefault   bool           `gorm:"default:false" json:"is_default"`
	IsSystem    bool           `gorm:"default:false" json:"is_system"` // System prompts cannot be deleted
	CreatedBy   uint           `json:"created_by"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// SystemConfig represents system-wide configuration (stored in database)
type SystemConfig struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Key       string    `gorm:"uniqueIndex;size:100;not null" json:"key"`
	Value     string    `gorm:"type:text" json:"value"`
	Type      string    `gorm:"size:20;default:string" json:"type"` // string, int, bool, json
	Group     string    `gorm:"size:50;index" json:"group"`         // general, ldap, notification, etc.
	Label     string    `gorm:"size:200" json:"label"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// IMBot represents an IM notification bot
type IMBot struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"size:100;not null" json:"name"`
	Type      string         `gorm:"size:50;not null" json:"type"` // wechat_work, dingtalk, feishu, slack
	Webhook   string         `gorm:"size:500;not null" json:"webhook"`
	Secret    string         `gorm:"size:255" json:"-"`
	IsActive  bool           `gorm:"default:true" json:"is_active"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// SystemLog represents a system operation log
type SystemLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Level     string    `gorm:"size:20;index" json:"level"` // info, warning, error
	Module    string    `gorm:"size:100;index" json:"module"`
	Action    string    `gorm:"size:200;index" json:"action"`
	Message   string    `gorm:"type:text" json:"message"`
	UserID    *uint     `json:"user_id"`
	IP        string    `gorm:"size:50" json:"ip"`
	UserAgent string    `gorm:"size:500" json:"user_agent"`
	Extra     string    `gorm:"type:text" json:"extra"` // JSON extra data
	CreatedAt time.Time `gorm:"index" json:"created_at"`
}

// TableName overrides
func (User) TableName() string           { return "users" }
func (Project) TableName() string        { return "projects" }
func (ReviewLog) TableName() string      { return "review_logs" }
func (LLMConfig) TableName() string      { return "llm_configs" }
func (PromptTemplate) TableName() string { return "prompt_templates" }
func (SystemConfig) TableName() string   { return "system_configs" }
func (IMBot) TableName() string          { return "im_bots" }
func (SystemLog) TableName() string      { return "system_logs" }

// MaskAPIKey returns masked API key for display
func (l *LLMConfig) MaskAPIKey() string {
	if len(l.APIKey) <= 8 {
		return "****"
	}
	return l.APIKey[:4] + "****" + l.APIKey[len(l.APIKey)-4:]
}
