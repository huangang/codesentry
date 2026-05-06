package models

import "time"

// MemberWorkSummary stores a personal daily work summary for a member.
// Each member can have at most one summary per day (unique index on author+date).
type MemberWorkSummary struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Author      string    `gorm:"size:200;not null;uniqueIndex:idx_member_summary_author_date" json:"author"`
	Date        string    `gorm:"size:10;not null;uniqueIndex:idx_member_summary_author_date" json:"date"` // YYYY-MM-DD
	Summary     string    `gorm:"type:MEDIUMTEXT" json:"summary"`
	RawData     string    `gorm:"type:MEDIUMTEXT" json:"raw_data"` // JSON-encoded WorkSummaryData
	AIModelUsed string    `gorm:"size:100" json:"ai_model_used"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (MemberWorkSummary) TableName() string { return "member_work_summaries" }

// MemberWorkPeriodSummary stores a cross-period summary built from daily summaries.
// Each member can have at most one period summary for the same date range.
type MemberWorkPeriodSummary struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Author      string    `gorm:"size:200;not null;uniqueIndex:idx_member_period_summary_author_range" json:"author"`
	StartDate   string    `gorm:"size:10;not null;uniqueIndex:idx_member_period_summary_author_range" json:"start_date"` // YYYY-MM-DD
	EndDate     string    `gorm:"size:10;not null;uniqueIndex:idx_member_period_summary_author_range" json:"end_date"`   // YYYY-MM-DD
	Summary     string    `gorm:"type:MEDIUMTEXT" json:"summary"`
	RawData     string    `gorm:"type:MEDIUMTEXT" json:"raw_data"` // JSON-encoded WorkPeriodSummaryData
	AIModelUsed string    `gorm:"size:100" json:"ai_model_used"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (MemberWorkPeriodSummary) TableName() string { return "member_work_period_summaries" }
