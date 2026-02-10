package models

import (
	"time"

	"gorm.io/gorm"
)

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

func (ReviewLog) TableName() string { return "review_logs" }
