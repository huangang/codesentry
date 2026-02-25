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
	Role      string         `gorm:"size:50;default:user" json:"role"`       // admin, developer, user
	AuthType  string         `gorm:"size:20;default:local" json:"auth_type"` // local, ldap
	IsActive  bool           `gorm:"default:true" json:"is_active"`
	LastLogin *time.Time     `json:"last_login"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (User) TableName() string { return "users" }
