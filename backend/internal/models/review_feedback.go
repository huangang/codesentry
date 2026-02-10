package models

import "time"

// ReviewFeedback represents user feedback on AI review and AI's response
type ReviewFeedback struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	ReviewLogID   uint       `gorm:"index;not null" json:"review_log_id"`
	ReviewLog     *ReviewLog `gorm:"foreignKey:ReviewLogID" json:"review_log,omitempty"`
	UserID        uint       `gorm:"index" json:"user_id"`
	User          *User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	FeedbackType  string     `gorm:"size:50;not null" json:"feedback_type"`         // agree, disagree, question, clarification
	UserMessage   string     `gorm:"type:text;not null" json:"user_message"`        // User's feedback/question
	AIResponse    string     `gorm:"type:text" json:"ai_response"`                  // AI's response to feedback
	PreviousScore *float64   `json:"previous_score"`                                // Score before re-evaluation
	UpdatedScore  *float64   `json:"updated_score"`                                 // Score after re-evaluation (if changed)
	ScoreChanged  bool       `gorm:"default:false" json:"score_changed"`            // Whether score was updated
	ProcessStatus string     `gorm:"size:50;default:pending" json:"process_status"` // pending, processing, completed, failed
	ErrorMessage  string     `gorm:"type:text" json:"error_message"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

func (ReviewFeedback) TableName() string { return "review_feedbacks" }
