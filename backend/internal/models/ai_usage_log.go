package models

import "time"

// AIUsageLog records each LLM API call for cost and usage tracking.
type AIUsageLog struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	ProjectID        *uint     `gorm:"index" json:"project_id"`
	ReviewLogID      *uint     `gorm:"index" json:"review_log_id"`
	LLMConfigID      uint      `gorm:"index" json:"llm_config_id"`
	Provider         string    `gorm:"size:50" json:"provider"`
	Model            string    `gorm:"size:100" json:"model"`
	PromptTokens     int       `json:"prompt_tokens"`
	CompletionTokens int       `json:"completion_tokens"`
	TotalTokens      int       `json:"total_tokens"`
	LatencyMs        int64     `json:"latency_ms"`
	Success          bool      `json:"success"`
	ErrorMessage     string    `gorm:"size:500" json:"error_message,omitempty"`
	CreatedAt        time.Time `gorm:"index" json:"created_at"`
}

func (AIUsageLog) TableName() string { return "ai_usage_logs" }
