package models

import (
	"time"

	"gorm.io/gorm"
)

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

func (PromptTemplate) TableName() string { return "prompt_templates" }
