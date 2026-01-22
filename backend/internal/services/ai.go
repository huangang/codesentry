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

	prompt := s.getPromptForProject(&project, req.CustomPrompt)

	prompt = strings.ReplaceAll(prompt, "{{diffs}}", req.Diffs)
	prompt = strings.ReplaceAll(prompt, "{{commits}}", req.Commits)

	log.Printf("[AI] Prompt length: %d chars, Diffs length: %d chars, Commits length: %d chars",
		len(prompt), len(req.Diffs), len(req.Commits))

	if len(prompt) > 500 {
		log.Printf("[AI] Prompt preview (first 500 chars): %s...", prompt[:500])
	} else {
		log.Printf("[AI] Prompt: %s", prompt)
	}

	llmConfigs := s.getOrderedLLMConfigs(&project)
	if len(llmConfigs) == 0 {
		return nil, fmt.Errorf("no LLM configuration available")
	}

	var lastErr error
	for i, llmConfig := range llmConfigs {
		log.Printf("[AI] Attempting LLM %d/%d: %s (model: %s)", i+1, len(llmConfigs), llmConfig.Name, llmConfig.Model)

		result, err := s.callLLM(ctx, &llmConfig, prompt)
		if err == nil {
			log.Printf("[AI] Success with LLM: %s", llmConfig.Name)
			return result, nil
		}

		lastErr = err
		log.Printf("[AI] LLM %s failed: %v, trying next...", llmConfig.Name, err)
	}

	return nil, fmt.Errorf("all LLMs failed, last error: %w", lastErr)
}

func (s *AIService) getOrderedLLMConfigs(project *models.Project) []models.LLMConfig {
	var configs []models.LLMConfig

	if project.LLMConfigID != nil {
		var projectConfig models.LLMConfig
		if err := s.db.Where("id = ? AND is_active = ?", *project.LLMConfigID, true).First(&projectConfig).Error; err == nil {
			configs = append(configs, projectConfig)
		}
	}

	var defaultConfig models.LLMConfig
	if err := s.db.Where("is_default = ? AND is_active = ?", true, true).First(&defaultConfig).Error; err == nil {
		if len(configs) == 0 || configs[0].ID != defaultConfig.ID {
			configs = append(configs, defaultConfig)
		}
	}

	var backupConfigs []models.LLMConfig
	existingIDs := make(map[uint]bool)
	for _, c := range configs {
		existingIDs[c.ID] = true
	}
	s.db.Where("is_active = ?", true).Order("id ASC").Find(&backupConfigs)
	for _, c := range backupConfigs {
		if !existingIDs[c.ID] {
			configs = append(configs, c)
		}
	}

	if len(configs) == 0 {
		configs = append(configs, models.LLMConfig{
			Name:    "fallback",
			BaseURL: s.config.BaseURL,
			APIKey:  s.config.APIKey,
			Model:   s.config.Model,
		})
	}

	return configs
}

func (s *AIService) callLLM(ctx context.Context, llmConfig *models.LLMConfig, prompt string) (*ReviewResult, error) {
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

func (s *AIService) getPromptForProject(project *models.Project, customPrompt string) string {
	var prompt string
	var isSystemDefault bool

	if customPrompt != "" {
		log.Printf("[AI] Using custom prompt from request")
		prompt = customPrompt
	} else if project.AIPrompt != "" {
		log.Printf("[AI] Using project custom prompt")
		prompt = project.AIPrompt
	} else if project.AIPromptID != nil {
		var promptTemplate models.PromptTemplate
		if err := s.db.First(&promptTemplate, *project.AIPromptID).Error; err == nil {
			log.Printf("[AI] Using linked prompt template: %s (ID: %d)", promptTemplate.Name, promptTemplate.ID)
			prompt = promptTemplate.Content
		}
	}

	if prompt == "" {
		var defaultPrompt models.PromptTemplate
		if err := s.db.Where("is_default = ?", true).First(&defaultPrompt).Error; err == nil {
			log.Printf("[AI] Using system default prompt: %s (ID: %d)", defaultPrompt.Name, defaultPrompt.ID)
			prompt = defaultPrompt.Content
		} else {
			log.Printf("[AI] Using hardcoded default prompt")
			prompt = NewProjectService(s.db).GetDefaultPrompt()
		}
		isSystemDefault = true
	}

	if !isSystemDefault && !containsScoringInstruction(prompt) {
		log.Printf("[AI] Prompt missing scoring instructions, auto-appending")
		prompt = appendScoringInstruction(prompt)
	}

	return prompt
}

func containsScoringInstruction(prompt string) bool {
	lowerPrompt := strings.ToLower(prompt)
	chineseKeywords := []string{"总分", "评分", "分数", "打分", "得分", "x/100", "/100分"}
	englishKeywords := []string{"total score", "score:", "scoring", "points", "x/100", "/100 points", "rate the", "rating"}
	scoringKeywords := append(chineseKeywords, englishKeywords...)

	for _, keyword := range scoringKeywords {
		if strings.Contains(lowerPrompt, keyword) {
			return true
		}
	}
	return false
}

func appendScoringInstruction(prompt string) string {
	scoringInstruction := `

---
## Scoring Requirement (Auto-appended)
Please provide a score for the code review. Use the following format at the end of your review:

### Total Score: X/100

Score breakdown (adjust based on your review focus):
- Code Quality: X/40
- Security: X/30  
- Best Practices: X/20
- Other: X/10
`
	return prompt + scoringInstruction
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
