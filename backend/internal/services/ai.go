package services

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/huangang/codesentry/backend/pkg/logger"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/huangang/codesentry/backend/internal/config"
	"github.com/huangang/codesentry/backend/internal/models"
	"github.com/ollama/ollama/api"
	"github.com/sashabaranov/go-openai"
	"google.golang.org/genai"
	"gorm.io/gorm"
)

// Pre-compiled regex patterns for score extraction and file context processing
var (
	scorePatterns = []*regexp.Regexp{
		regexp.MustCompile(`总分[:：]\s*(\d+)分?`),
		regexp.MustCompile(`[Tt]otal\s*[Ss]core[:：]?\s*(\d+)`),
		regexp.MustCompile(`[Ss]core[:：]?\s*(\d+)\s*/\s*100`),
		regexp.MustCompile(`(\d+)\s*/\s*100\s*分?`),
		regexp.MustCompile(`评分[:：]\s*(\d+)`),
	}
	ifBlockRegex = regexp.MustCompile(`(?s)\{\{#if_file_context\}\}(.*?)\{\{/if_file_context\}\}`)
)

type AIService struct {
	db            *gorm.DB
	config        *config.OpenAIConfig
	configService *SystemConfigService
}

func NewAIService(db *gorm.DB, cfg *config.OpenAIConfig) *AIService {
	return &AIService{
		db:            db,
		config:        cfg,
		configService: NewSystemConfigService(db),
	}
}

type ReviewRequest struct {
	ProjectID    uint
	Diffs        string
	Commits      string
	FileContext  string
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

	prompt = s.processFileContextBlock(prompt, req.FileContext)

	logger.Infof("[AI] Prompt length: %d chars, Diffs length: %d chars, Commits length: %d chars, FileContext length: %d chars",
		len(prompt), len(req.Diffs), len(req.Commits), len(req.FileContext))

	if len(prompt) > 500 {
		logger.Infof("[AI] Prompt preview (first 500 chars): %s...", prompt[:500])
	} else {
		logger.Infof("[AI] Prompt: %s", prompt)
	}

	llmConfigs := s.getOrderedLLMConfigs(&project)
	if len(llmConfigs) == 0 {
		return nil, fmt.Errorf("no LLM configuration available")
	}

	var lastErr error
	for i, llmConfig := range llmConfigs {
		logger.Infof("[AI] Attempting LLM %d/%d: %s (model: %s)", i+1, len(llmConfigs), llmConfig.Name, llmConfig.Model)

		result, err := s.callLLM(ctx, &llmConfig, prompt)
		if err == nil {
			logger.Infof("[AI] Success with LLM: %s", llmConfig.Name)
			return result, nil
		}

		lastErr = err
		logger.Infof("[AI] LLM %s failed: %v, trying next...", llmConfig.Name, err)
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

// callLLM dispatches to the appropriate provider-specific function based on Provider field
func (s *AIService) callLLM(ctx context.Context, llmConfig *models.LLMConfig, prompt string) (*ReviewResult, error) {
	logger.Infof("[AI] Using provider: %s, model: %s, baseURL: %s", llmConfig.Provider, llmConfig.Model, llmConfig.BaseURL)

	switch llmConfig.Provider {
	case "anthropic":
		return s.callAnthropic(ctx, llmConfig, prompt)
	case "ollama":
		return s.callOllama(ctx, llmConfig, prompt)
	case "gemini":
		return s.callGemini(ctx, llmConfig, prompt)
	case "azure":
		return s.callAzure(ctx, llmConfig, prompt)
	default:
		// openai and other OpenAI-compatible services
		return s.callOpenAI(ctx, llmConfig, prompt)
	}
}

// callOpenAI handles OpenAI and OpenAI-compatible APIs (including custom endpoints)
func (s *AIService) callOpenAI(ctx context.Context, llmConfig *models.LLMConfig, prompt string) (*ReviewResult, error) {
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
		logger.Infof("[AI] OpenAI API error: %v", err)
		return nil, fmt.Errorf("OpenAI API error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from OpenAI")
	}

	content := resp.Choices[0].Message.Content
	logger.Infof("[AI] OpenAI response length: %d chars", len(content))

	return &ReviewResult{
		Content: content,
		Score:   extractScore(content),
	}, nil
}

// callAnthropic handles Anthropic Claude API using the native SDK
func (s *AIService) callAnthropic(ctx context.Context, llmConfig *models.LLMConfig, prompt string) (*ReviewResult, error) {
	client := anthropic.NewClient(
		option.WithAPIKey(llmConfig.APIKey),
	)

	maxTokens := int64(llmConfig.MaxTokens)
	if maxTokens == 0 {
		maxTokens = 4096
	}

	model := llmConfig.Model
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}

	resp, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(model),
		MaxTokens: maxTokens,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		logger.Infof("[AI] Anthropic API error: %v", err)
		return nil, fmt.Errorf("Anthropic API error: %w", err)
	}

	// Extract text content from response
	var content string
	for _, block := range resp.Content {
		if block.Type == "text" {
			content += block.Text
		}
	}

	logger.Infof("[AI] Anthropic response length: %d chars", len(content))

	return &ReviewResult{
		Content: content,
		Score:   extractScore(content),
	}, nil
}

// callOllama handles Ollama API using the native SDK
func (s *AIService) callOllama(ctx context.Context, llmConfig *models.LLMConfig, prompt string) (*ReviewResult, error) {
	baseURL := llmConfig.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	// Parse URL and create client
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid Ollama base URL: %w", err)
	}
	client := api.NewClient(u, http.DefaultClient)

	model := llmConfig.Model
	if model == "" {
		model = "llama3"
	}

	var content strings.Builder
	err = client.Chat(ctx, &api.ChatRequest{
		Model: model,
		Messages: []api.Message{
			{Role: "user", Content: prompt},
		},
		Options: map[string]interface{}{
			"temperature": llmConfig.Temperature,
		},
	}, func(resp api.ChatResponse) error {
		content.WriteString(resp.Message.Content)
		return nil
	})

	if err != nil {
		logger.Infof("[AI] Ollama API error: %v", err)
		return nil, fmt.Errorf("Ollama API error: %w", err)
	}

	result := content.String()
	logger.Infof("[AI] Ollama response length: %d chars", len(result))

	return &ReviewResult{
		Content: result,
		Score:   extractScore(result),
	}, nil
}

// callGemini handles Google Gemini API using the native SDK
func (s *AIService) callGemini(ctx context.Context, llmConfig *models.LLMConfig, prompt string) (*ReviewResult, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: llmConfig.APIKey,
	})
	if err != nil {
		return nil, fmt.Errorf("Gemini client error: %w", err)
	}

	model := llmConfig.Model
	if model == "" {
		model = "gemini-3.0-flash"
	}

	resp, err := client.Models.GenerateContent(ctx, model, genai.Text(prompt), nil)
	if err != nil {
		logger.Infof("[AI] Gemini API error: %v", err)
		return nil, fmt.Errorf("Gemini API error: %w", err)
	}

	content := resp.Text()
	logger.Infof("[AI] Gemini response length: %d chars", len(content))

	return &ReviewResult{
		Content: content,
		Score:   extractScore(content),
	}, nil
}

// callAzure handles Azure OpenAI API using special configuration
func (s *AIService) callAzure(ctx context.Context, llmConfig *models.LLMConfig, prompt string) (*ReviewResult, error) {
	// Azure requires BaseURL format: https://{resource-name}.openai.azure.com
	// Model field is used as deployment name
	config := openai.DefaultAzureConfig(llmConfig.APIKey, llmConfig.BaseURL)
	client := openai.NewClientWithConfig(config)

	temperature := float32(0.3)
	if llmConfig.Temperature > 0 {
		temperature = float32(llmConfig.Temperature)
	}

	resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: llmConfig.Model, // In Azure, this is the deployment name
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleUser, Content: prompt},
		},
		Temperature: temperature,
	})

	if err != nil {
		logger.Infof("[AI] Azure OpenAI API error: %v", err)
		return nil, fmt.Errorf("Azure OpenAI API error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from Azure OpenAI")
	}

	content := resp.Choices[0].Message.Content
	logger.Infof("[AI] Azure OpenAI response length: %d chars", len(content))

	return &ReviewResult{
		Content: content,
		Score:   extractScore(content),
	}, nil
}

func (s *AIService) getPromptForProject(project *models.Project, customPrompt string) string {
	var prompt string
	var isSystemDefault bool

	if customPrompt != "" {
		logger.Infof("[AI] Using custom prompt from request")
		prompt = customPrompt
	} else if project.AIPrompt != "" {
		logger.Infof("[AI] Using project custom prompt")
		prompt = project.AIPrompt
	} else if project.AIPromptID != nil {
		var promptTemplate models.PromptTemplate
		if err := s.db.First(&promptTemplate, *project.AIPromptID).Error; err == nil {
			logger.Infof("[AI] Using linked prompt template: %s (ID: %d)", promptTemplate.Name, promptTemplate.ID)
			prompt = promptTemplate.Content
		}
	}

	if prompt == "" {
		var defaultPrompt models.PromptTemplate
		if err := s.db.Where("is_default = ?", true).First(&defaultPrompt).Error; err == nil {
			logger.Infof("[AI] Using system default prompt: %s (ID: %d)", defaultPrompt.Name, defaultPrompt.ID)
			prompt = defaultPrompt.Content
		} else {
			logger.Infof("[AI] Using hardcoded default prompt")
			prompt = NewProjectService(s.db).GetDefaultPrompt()
		}
		isSystemDefault = true
	}

	if !isSystemDefault && !containsScoringInstruction(prompt) {
		logger.Infof("[AI] Prompt missing scoring instructions, auto-appending")
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

func (s *AIService) processFileContextBlock(prompt string, fileContext string) string {
	if strings.TrimSpace(fileContext) != "" {
		prompt = ifBlockRegex.ReplaceAllString(prompt, "$1")
		prompt = strings.ReplaceAll(prompt, "{{file_context}}", fileContext)
	} else {
		prompt = ifBlockRegex.ReplaceAllString(prompt, "")
		prompt = strings.ReplaceAll(prompt, "{{file_context}}", "")
	}

	return prompt
}

// extractScore extracts the score from review content
func extractScore(content string) float64 {
	for _, re := range scorePatterns {
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

func (s *AIService) CallWithConfig(ctx context.Context, llmConfigID uint, prompt string) (string, string, error) {
	var llmConfig models.LLMConfig

	if llmConfigID > 0 {
		if err := s.db.Where("id = ? AND is_active = ?", llmConfigID, true).First(&llmConfig).Error; err != nil {
			logger.Infof("[AI] Specified LLM config %d not found or inactive, falling back to default", llmConfigID)
		}
	}

	if llmConfig.ID == 0 {
		if err := s.db.Where("is_default = ? AND is_active = ?", true, true).First(&llmConfig).Error; err != nil {
			var anyConfig models.LLMConfig
			if err := s.db.Where("is_active = ?", true).First(&anyConfig).Error; err != nil {
				return "", "", fmt.Errorf("no active LLM configuration available")
			}
			llmConfig = anyConfig
		}
	}

	logger.Infof("[AI] CallWithConfig using LLM: %s (ID: %d)", llmConfig.Name, llmConfig.ID)

	result, err := s.callLLM(ctx, &llmConfig, prompt)
	if err != nil {
		return "", "", err
	}

	return result.Content, llmConfig.Name, nil
}

func (s *AIService) getChunkedReviewEnabled() bool {
	return s.configService.GetWithDefault("chunked_review_enabled", "true") == "true"
}

func (s *AIService) getChunkThreshold() int {
	val, err := strconv.Atoi(s.configService.GetWithDefault("chunked_review_threshold", "50000"))
	if err != nil || val <= 0 {
		return 50000
	}
	return val
}

func (s *AIService) getMaxTokensPerBatch() int {
	val, err := strconv.Atoi(s.configService.GetWithDefault("chunked_review_max_tokens_per_batch", "30000"))
	if err != nil || val <= 0 {
		return 30000
	}
	return val
}

func (s *AIService) ReviewChunked(ctx context.Context, req *ReviewRequest) (*ReviewResult, error) {
	if !s.getChunkedReviewEnabled() {
		return s.Review(ctx, req)
	}

	diffSize := len(req.Diffs)
	threshold := s.getChunkThreshold()

	if diffSize < threshold {
		return s.Review(ctx, req)
	}

	files := ParseDiffToFiles(req.Diffs)
	if len(files) <= 1 {
		logger.Infof("[AI] Large diff (%d chars) but only %d file(s), using regular review", diffSize, len(files))
		return s.Review(ctx, req)
	}

	maxTokens := s.getMaxTokensPerBatch()
	batches := CreateBatches(files, maxTokens)

	logger.Infof("[AI] Large diff detected (%d chars, %d files), using chunked review with %d batches",
		diffSize, len(files), len(batches))

	var (
		batchResults []BatchResult
		mu           sync.Mutex
		wg           sync.WaitGroup
	)

	for i, batch := range batches {
		wg.Add(1)
		go func(batchIdx int, b ReviewBatch) {
			defer wg.Done()

			batchDiff := ReconstructDiff(b.Files)
			fileNames := GetBatchFileNames(b)
			weight := GetBatchWeight(b)

			logger.Infof("[AI] Reviewing batch %d/%d: %d files, ~%d tokens",
				batchIdx+1, len(batches), len(b.Files), b.TotalTokens)

			result, err := s.Review(ctx, &ReviewRequest{
				ProjectID: req.ProjectID,
				Diffs:     batchDiff,
				Commits:   req.Commits,
			})

			if err != nil {
				logger.Infof("[AI] Batch %d/%d failed: %v", batchIdx+1, len(batches), err)
				return
			}

			mu.Lock()
			batchResults = append(batchResults, BatchResult{
				BatchIndex: batchIdx,
				Files:      fileNames,
				Score:      result.Score,
				Content:    result.Content,
				Weight:     weight,
			})
			mu.Unlock()

			logger.Infof("[AI] Batch %d/%d completed: score=%.0f", batchIdx+1, len(batches), result.Score)
		}(i, batch)
	}

	wg.Wait()

	if len(batchResults) == 0 {
		return nil, fmt.Errorf("all batches failed during chunked review")
	}

	aggregated := AggregateResults(batchResults)

	logger.Infof("[AI] Chunked review completed: %d/%d batches succeeded, aggregated score=%.0f",
		len(batchResults), len(batches), aggregated.Score)

	return &ReviewResult{
		Content: aggregated.Content,
		Score:   aggregated.Score,
	}, nil
}
