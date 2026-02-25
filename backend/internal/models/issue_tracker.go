package models

import (
	"time"

	"gorm.io/gorm"
)

// IssueTracker represents a Jira/Linear/GitHub Issues integration.
type IssueTracker struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	Name           string         `gorm:"size:100;not null" json:"name"`
	Type           string         `gorm:"size:50;not null" json:"type"` // jira, linear, github_issues
	BaseURL        string         `gorm:"size:500" json:"base_url"`     // e.g., https://yourcompany.atlassian.net
	APIToken       string         `gorm:"size:500" json:"-"`
	APITokenMask   string         `gorm:"-" json:"api_token_mask"`
	ProjectKey     string         `gorm:"size:100" json:"project_key"`            // Jira project key or Linear team key
	IssueType      string         `gorm:"size:100;default:Bug" json:"issue_type"` // Bug, Task, etc.
	ScoreThreshold float64        `gorm:"default:60" json:"score_threshold"`      // Auto-create issue if score < threshold
	IsActive       bool           `gorm:"default:true" json:"is_active"`
	AssigneeField  string         `gorm:"size:100" json:"assignee_field"` // Optional: auto-assign
	Labels         string         `gorm:"size:500" json:"labels"`         // Comma-separated labels
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

func (IssueTracker) TableName() string { return "issue_trackers" }
