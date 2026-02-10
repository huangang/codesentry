package models

import (
	"time"

	"gorm.io/gorm"
)

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

func (ReviewTemplate) TableName() string { return "review_templates" }
