package models

import "time"

// MemberAlias maps a git author name (alias) to a canonical display name.
// This allows merging multiple identities (e.g. "bo.xiong" and "熊波") into one person.
type MemberAlias struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	Alias         string    `gorm:"size:200;not null;uniqueIndex" json:"alias"`    // original git author name
	CanonicalName string    `gorm:"size:200;not null;index" json:"canonical_name"` // merged display name
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (MemberAlias) TableName() string { return "member_aliases" }
