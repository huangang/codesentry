package models

import "time"

// DailyReport represents a daily code review report
type DailyReport struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	ReportDate time.Time `gorm:"uniqueIndex;not null" json:"report_date"`
	ReportType string    `gorm:"size:20;default:daily" json:"report_type"` // daily, weekly

	TotalProjects  int     `json:"total_projects"`
	TotalCommits   int     `json:"total_commits"`
	TotalAuthors   int     `json:"total_authors"`
	TotalAdditions int     `json:"total_additions"`
	TotalDeletions int     `json:"total_deletions"`
	AverageScore   float64 `json:"average_score"`
	PassedCount    int     `json:"passed_count"`
	FailedCount    int     `json:"failed_count"`
	PendingCount   int     `json:"pending_count"`

	TopProjects     string `gorm:"type:text" json:"top_projects"`
	TopAuthors      string `gorm:"type:text" json:"top_authors"`
	LowScoreReviews string `gorm:"type:text" json:"low_score_reviews"`

	AIAnalysis  string `gorm:"type:text" json:"ai_analysis"`
	AIModelUsed string `gorm:"size:100" json:"ai_model_used"`

	NotifiedAt  *time.Time `json:"notified_at"`
	NotifyError string     `gorm:"type:text" json:"notify_error"`

	CreatedAt time.Time `json:"created_at"`
}

func (DailyReport) TableName() string { return "daily_reports" }
