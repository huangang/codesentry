package models

import (
	"time"

	"gorm.io/gorm"
)

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

func (GitCredential) TableName() string { return "git_credentials" }

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
