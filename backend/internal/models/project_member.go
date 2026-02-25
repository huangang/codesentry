package models

import (
	"time"

	"gorm.io/gorm"
)

// ProjectMember represents a user's membership and role within a project.
type ProjectMember struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	ProjectID uint           `gorm:"uniqueIndex:idx_project_user;not null" json:"project_id"`
	Project   *Project       `gorm:"foreignKey:ProjectID" json:"project,omitempty"`
	UserID    uint           `gorm:"uniqueIndex:idx_project_user;not null" json:"user_id"`
	User      *User          `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Role      string         `gorm:"size:50;default:viewer" json:"role"` // owner, maintainer, viewer
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (ProjectMember) TableName() string { return "project_members" }
