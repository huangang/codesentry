package services

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/huangang/codesentry/backend/internal/config"
	"github.com/huangang/codesentry/backend/internal/models"
	"github.com/sashabaranov/go-openai"
	"gorm.io/gorm"
)

type AIService struct {
	db     *gorm.DB
	config *config.OpenAIConfig
}

func NewAIService(db *gorm.DB, cfg *config.OpenAIConfig) *AIService {
	return &AIService{
		db:     db,
		config: cfg,
	}
}

type ReviewRequest struct {
	ProjectID    uint
	Diffs        string
	Commits      string
	CustomPrompt string
}

type ReviewResult struct {
	Content string
	Score   float64
}

func (s *AIService) Review(ctx context.Context, req *ReviewRequest) (*ReviewResult, error) {
	var project models.Project
	if err := s.db.First(&project, req.ProjectID).Error; err != nil {
		return nil, fmt.Errorf("project not found: %w", err)
	}

	prompt := req.CustomPrompt
	if prompt == "" {
		prompt = project.AIPrompt
	}
	if prompt == "" {
		prompt = NewProjectService(s.db).GetDefaultPrompt()
	}

	prompt = strings.ReplaceAll(prompt, "{{diffs}}", req.Diffs)
	prompt = strings.ReplaceAll(prompt, "{{commits}}", req.Commits)

	log.Printf("[AI] Prompt length: %d chars, Diffs length: %d chars, Commits length: %d chars",
		len(prompt), len(req.Diffs), len(req.Commits))

	if len(prompt) > 500 {
		log.Printf("[AI] Prompt preview (first 500 chars): %s...", prompt[:500])
	} else {
		log.Printf("[AI] Prompt: %s", prompt)
	}

	var llmConfig models.LLMConfig
	if err := s.db.Where("is_default = ? AND is_active = ?", true, true).First(&llmConfig).Error; err != nil {
		llmConfig = models.LLMConfig{
			BaseURL: s.config.BaseURL,
			APIKey:  s.config.APIKey,
			Model:   s.config.Model,
		}
	}

	log.Printf("[AI] Using LLM: %s, model: %s", llmConfig.BaseURL, llmConfig.Model)

	clientConfig := openai.DefaultConfig(llmConfig.APIKey)
	if llmConfig.BaseURL != "" {
		clientConfig.BaseURL = llmConfig.BaseURL
	}
	client := openai.NewClientWithConfig(clientConfig)

	temperature := float32(0.3)
	if llmConfig.Temperature > 0 {
		temperature = float32(llmConfig.Temperature)
	}

	resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: llmConfig.Model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
		Temperature: temperature,
	})

	if err != nil {
		log.Printf("[AI] API error: %v", err)
		return nil, fmt.Errorf("AI review failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from AI")
	}

	content := resp.Choices[0].Message.Content
	log.Printf("[AI] Response length: %d chars", len(content))

	score := extractScore(content)

	return &ReviewResult{
		Content: content,
		Score:   score,
	}, nil
}

// extractScore extracts the score from review content
func extractScore(content string) float64 {
	patterns := []string{
		`总分[:：]\s*(\d+)分?`,
		`[Tt]otal\s*[Ss]core[:：]?\s*(\d+)`,
		`[Ss]core[:：]?\s*(\d+)\s*/\s*100`,
		`(\d+)\s*/\s*100\s*分?`,
		`评分[:：]\s*(\d+)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(content)
		if len(matches) >= 2 {
			if score, err := strconv.ParseFloat(matches[1], 64); err == nil {
				if score >= 0 && score <= 100 {
					return score
				}
			}
		}
	}
	return 0
}
