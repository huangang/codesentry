package models

import (
	"time"

	"gorm.io/gorm"
)

// IMBot represents an IM notification bot
type IMBot struct {
	ID                 uint           `gorm:"primaryKey" json:"id"`
	Name               string         `gorm:"size:100;not null" json:"name"`
	Type               string         `gorm:"size:50;not null" json:"type"` // wechat_work, dingtalk, feishu, slack, discord, teams, telegram
	Webhook            string         `gorm:"size:500;not null" json:"webhook"`
	Secret             string         `gorm:"size:255" json:"-"`
	Extra              string         `gorm:"size:500" json:"extra"` // Extra config (e.g., Telegram chat_id)
	IsActive           bool           `gorm:"default:true" json:"is_active"`
	ErrorNotify        bool           `gorm:"default:false" json:"error_notify"`         // Whether to receive error notifications
	DailyReportEnabled bool           `gorm:"default:false" json:"daily_report_enabled"` // Whether to receive daily reports
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"-"`
}

func (IMBot) TableName() string { return "im_bots" }
