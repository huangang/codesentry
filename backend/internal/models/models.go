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

// ReviewLog represents a code review record
type ReviewLog struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	ProjectID     uint           `gorm:"index;not null" json:"project_id"`
	Project       *Project       `gorm:"foreignKey:ProjectID" json:"project,omitempty"`
	EventType     string         `gorm:"size:50;not null" json:"event_type"` // push, merge_request
	CommitHash    string         `gorm:"size:100;index" json:"commit_hash"`
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
	CommentPosted bool           `gorm:"default:false" json:"comment_posted"`
	ErrorMessage  string         `gorm:"type:text" json:"error_message"`
	RetryCount    int            `gorm:"default:0" json:"retry_count"`
	IsManual      bool           `gorm:"default:false" json:"is_manual"`
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
	Key       string    `gorm:"column:key;uniqueIndex;size:100;not null" json:"key"`
	Value     string    `gorm:"type:text" json:"value"`
	Type      string    `gorm:"size:20;default:string" json:"type"`      // string, int, bool, json
	Group     string    `gorm:"column:group;size:50;index" json:"group"` // general, ldap, notification, etc.
	Label     string    `gorm:"size:200" json:"label"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// IMBot represents an IM notification bot
type IMBot struct {
	ID                 uint           `gorm:"primaryKey" json:"id"`
	Name               string         `gorm:"size:100;not null" json:"name"`
	Type               string         `gorm:"size:50;not null" json:"type"` // wechat_work, dingtalk, feishu, slack, discord, teams, telegram
	Webhook            string         `gorm:"size:500;not null" json:"webhook"`
	Secret             string         `gorm:"size:255" json:"-"`
	Extra              string         `gorm:"size:500" json:"extra"` // Extra config (e.g., Telegram chat_id)
	IsActive           bool           `gorm:"default:true" json:"is_active"`
	ErrorNotify        bool           `gorm:"default:false" json:"error_notify"`         // Whether to receive error notifications
	DailyReportEnabled bool           `gorm:"default:false" json:"daily_report_enabled"` // Whether to receive daily reports
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"-"`
}

// DailyReport represents a daily code review report
type DailyReport struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	ReportDate time.Time `gorm:"uniqueIndex;not null" json:"report_date"`
	ReportType string    `gorm:"size:20;default:daily" json:"report_type"` // daily, weekly

	TotalProjects  int     `json:"total_projects"`
	TotalCommits   int     `json:"total_commits"`
	TotalAuthors   int     `json:"total_authors"`
	TotalAdditions int     `json:"total_additions"`
	TotalDeletions int     `json:"total_deletions"`
	AverageScore   float64 `json:"average_score"`
	PassedCount    int     `json:"passed_count"`
	FailedCount    int     `json:"failed_count"`
	PendingCount   int     `json:"pending_count"`

	TopProjects     string `gorm:"type:text" json:"top_projects"`
	TopAuthors      string `gorm:"type:text" json:"top_authors"`
	LowScoreReviews string `gorm:"type:text" json:"low_score_reviews"`

	AIAnalysis  string `gorm:"type:text" json:"ai_analysis"`
	AIModelUsed string `gorm:"size:100" json:"ai_model_used"`

	NotifiedAt  *time.Time `json:"notified_at"`
	NotifyError string     `gorm:"type:text" json:"notify_error"`

	CreatedAt time.Time `json:"created_at"`
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

// GitCredential represents a Git platform credential for auto-creating projects
type GitCredential struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	Name           string         `gorm:"size:200;not null" json:"name"`
	Platform       string         `gorm:"size:50;not null" json:"platform"`    // github, gitlab
	BaseURL        string         `gorm:"size:500" json:"base_url"`            // For self-hosted GitLab, e.g., https://gitlab.example.com
	AccessToken    string         `gorm:"size:500" json:"-"`                   // Token for API access
	WebhookSecret  string         `gorm:"size:255" json:"-"`                   // Secret for webhook verification
	AutoCreate     bool           `gorm:"default:true" json:"auto_create"`     // Auto-create projects on webhook
	DefaultEnabled bool           `gorm:"default:true" json:"default_enabled"` // Default AI enabled for new projects
	FileExtensions string         `gorm:"size:1000" json:"file_extensions"`    // Default file extensions for new projects
	ReviewEvents   string         `gorm:"size:200" json:"review_events"`       // Default review events: push,merge_request
	IgnorePatterns string         `gorm:"size:2000" json:"ignore_patterns"`    // Default ignore patterns
	IsActive       bool           `gorm:"default:true" json:"is_active"`       // Whether this credential is active
	CreatedBy      uint           `json:"created_by"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

// SchedulerLock represents a distributed lock for scheduled tasks
type SchedulerLock struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	LockName  string    `gorm:"uniqueIndex:idx_lock_name_key;size:100;not null" json:"lock_name"`
	LockKey   string    `gorm:"uniqueIndex:idx_lock_name_key;size:100;not null" json:"lock_key"`
	LockedBy  string    `gorm:"size:100" json:"locked_by"`
	LockedAt  time.Time `json:"locked_at"`
	ExpiresAt time.Time `gorm:"index" json:"expires_at"`
}

// ReviewTemplate represents a predefined review template
type ReviewTemplate struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"size:100;not null" json:"name"`
	Type        string         `gorm:"size:50;not null;index" json:"type"` // frontend, backend, security, general, custom
	Description string         `gorm:"size:500" json:"description"`
	Content     string         `gorm:"type:text;not null" json:"content"` // The actual prompt content
	IsBuiltIn   bool           `gorm:"default:false" json:"is_built_in"`  // System built-in templates cannot be deleted
	IsActive    bool           `gorm:"default:true" json:"is_active"`
	CreatedBy   uint           `json:"created_by"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// ReviewFeedback represents user feedback on AI review and AI's response
type ReviewFeedback struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	ReviewLogID   uint       `gorm:"index;not null" json:"review_log_id"`
	ReviewLog     *ReviewLog `gorm:"foreignKey:ReviewLogID" json:"review_log,omitempty"`
	UserID        uint       `gorm:"index" json:"user_id"`
	User          *User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	FeedbackType  string     `gorm:"size:50;not null" json:"feedback_type"`         // agree, disagree, question, clarification
	UserMessage   string     `gorm:"type:text;not null" json:"user_message"`        // User's feedback/question
	AIResponse    string     `gorm:"type:text" json:"ai_response"`                  // AI's response to feedback
	PreviousScore *float64   `json:"previous_score"`                                // Score before re-evaluation
	UpdatedScore  *float64   `json:"updated_score"`                                 // Score after re-evaluation (if changed)
	ScoreChanged  bool       `gorm:"default:false" json:"score_changed"`            // Whether score was updated
	ProcessStatus string     `gorm:"size:50;default:pending" json:"process_status"` // pending, processing, completed, failed
	ErrorMessage  string     `gorm:"type:text" json:"error_message"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
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
func (GitCredential) TableName() string  { return "git_credentials" }
func (DailyReport) TableName() string    { return "daily_reports" }
func (SchedulerLock) TableName() string  { return "scheduler_locks" }
func (ReviewTemplate) TableName() string { return "review_templates" }
func (ReviewFeedback) TableName() string { return "review_feedbacks" }

// MaskAPIKey returns masked API key for display
func (l *LLMConfig) MaskAPIKey() string {
	if len(l.APIKey) <= 8 {
		return "****"
	}
	return l.APIKey[:4] + "****" + l.APIKey[len(l.APIKey)-4:]
}

// MaskAccessToken returns masked access token for display
func (g *GitCredential) MaskAccessToken() string {
	if len(g.AccessToken) <= 8 {
		return "****"
	}
	return g.AccessToken[:4] + "****" + g.AccessToken[len(g.AccessToken)-4:]
}

// MaskWebhookSecret returns masked webhook secret for display
func (g *GitCredential) MaskWebhookSecret() string {
	if len(g.WebhookSecret) <= 8 {
		return "****"
	}
	return g.WebhookSecret[:4] + "****" + g.WebhookSecret[len(g.WebhookSecret)-4:]
}
