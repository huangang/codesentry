package models

import "time"

type RefreshToken struct {
	ID                uint       `gorm:"primaryKey" json:"id"`
	UserID            uint       `gorm:"index;not null" json:"user_id"`
	TokenHash         string     `gorm:"uniqueIndex;size:64;not null" json:"-"`
	ExpiresAt         time.Time  `gorm:"index;not null" json:"expires_at"`
	RevokedAt         *time.Time `gorm:"index" json:"revoked_at,omitempty"`
	ReplacedByTokenID *uint      `gorm:"index" json:"replaced_by_token_id,omitempty"`
	CreatedByIP       string     `gorm:"size:64" json:"created_by_ip,omitempty"`
	UserAgent         string     `gorm:"size:255" json:"user_agent,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

func (RefreshToken) TableName() string { return "refresh_tokens" }
