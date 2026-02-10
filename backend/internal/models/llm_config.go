package models

import (
	"time"

	"gorm.io/gorm"
)

// LLMConfig represents a large language model configuration (stored in database)
type LLMConfig struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"size:100;not null" json:"name"`
	Provider    string         `gorm:"size:50;default:openai" json:"provider"` // openai, azure, anthropic, etc.
	BaseURL     string         `gorm:"size:500;not null" json:"base_url"`
	APIKey      string         `gorm:"size:500" json:"-"`
	APIKeyMask  string         `gorm:"-" json:"api_key_mask"` // For display only
	Model       string         `gorm:"size:100" json:"model"`
	MaxTokens   int            `gorm:"default:4096" json:"max_tokens"`
	Temperature float64        `gorm:"default:0.3" json:"temperature"`
	IsDefault   bool           `gorm:"default:false" json:"is_default"`
	IsActive    bool           `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (LLMConfig) TableName() string { return "llm_configs" }

// MaskAPIKey returns masked API key for display
func (l *LLMConfig) MaskAPIKey() string {
	if len(l.APIKey) <= 8 {
		return "****"
	}
	return l.APIKey[:4] + "****" + l.APIKey[len(l.APIKey)-4:]
}
