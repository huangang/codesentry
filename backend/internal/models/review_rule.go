package models

import (
	"time"

	"gorm.io/gorm"
)

// ReviewRule defines an automated policy rule for CI/CD gating.
type ReviewRule struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	Name        string `gorm:"size:200;not null" json:"name"`
	Description string `gorm:"size:1000" json:"description"`
	ProjectID   *uint  `gorm:"index" json:"project_id"` // nil = global rule
	IsActive    bool   `gorm:"default:true" json:"is_active"`
	Priority    int    `gorm:"default:0" json:"priority"` // Higher = evaluated first

	// Conditions
	Condition string  `gorm:"size:50;not null" json:"condition"` // score_below, files_changed_above, has_keyword
	Threshold float64 `json:"threshold"`                         // e.g., 60 for score_below
	Keyword   string  `gorm:"size:500" json:"keyword"`           // For has_keyword condition

	// Actions
	Action      string `gorm:"size:50;not null" json:"action"` // block, warn, notify, label
	ActionValue string `gorm:"size:500" json:"action_value"`   // e.g., label name or notification message

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (ReviewRule) TableName() string { return "review_rules" }
