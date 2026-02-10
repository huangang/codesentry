package models

import "time"

// SystemLog represents a system operation log
type SystemLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Level     string    `gorm:"size:20;index" json:"level"` // info, warning, error
	Module    string    `gorm:"size:100;index" json:"module"`
	Action    string    `gorm:"size:200;index" json:"action"`
	Message   string    `gorm:"type:text" json:"message"`
	UserID    *uint     `json:"user_id"`
	IP        string    `gorm:"size:50" json:"ip"`
	UserAgent string    `gorm:"size:500" json:"user_agent"`
	Extra     string    `gorm:"type:text" json:"extra"` // JSON extra data
	CreatedAt time.Time `gorm:"index" json:"created_at"`
}

func (SystemLog) TableName() string { return "system_logs" }
