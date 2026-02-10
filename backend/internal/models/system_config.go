package models

import "time"

// SystemConfig represents system-wide configuration (stored in database)
type SystemConfig struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Key       string    `gorm:"column:key;uniqueIndex;size:100;not null" json:"key"`
	Value     string    `gorm:"type:text" json:"value"`
	Type      string    `gorm:"size:20;default:string" json:"type"`      // string, int, bool, json
	Group     string    `gorm:"column:group;size:50;index" json:"group"` // general, ldap, notification, etc.
	Label     string    `gorm:"size:200" json:"label"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (SystemConfig) TableName() string { return "system_configs" }
