package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/huangang/codesentry/backend/pkg/logger"

	"github.com/huangang/codesentry/backend/internal/config"
	"github.com/huangang/codesentry/backend/internal/models"
	"gorm.io/gorm"
)

type ReviewFeedbackService struct {
	db        *gorm.DB
	aiService *AIService
}

func NewReviewFeedbackService(db *gorm.DB, openAICfg *config.OpenAIConfig) *ReviewFeedbackService {
	return &ReviewFeedbackService{
		db:        db,
		aiService: NewAIService(db, openAICfg),
	}
}

// Create creates a new feedback and triggers AI re-evaluation
func (s *ReviewFeedbackService) Create(ctx context.Context, feedback *models.ReviewFeedback) error {
	// Get original review
	var reviewLog models.ReviewLog
	if err := s.db.First(&reviewLog, feedback.ReviewLogID).Error; err != nil {
		return fmt.Errorf("review not found: %w", err)
	}

	// Store previous score
	feedback.PreviousScore = reviewLog.Score
	feedback.ProcessStatus = "processing"

	if err := s.db.Create(feedback).Error; err != nil {
		return err
	}

	// Process feedback asynchronously
	go s.processFeedback(feedback.ID, reviewLog.ID, reviewLog.LLMConfigID)

	return nil
}

// processFeedback handles AI re-evaluation based on user feedback
func (s *ReviewFeedbackService) processFeedback(feedbackID, reviewLogID uint, llmConfigID *uint) {
	logger.Infof("[Feedback] Processing feedback ID=%d for review ID=%d", feedbackID, reviewLogID)

	// Reload feedback and review
	var feedback models.ReviewFeedback
	if err := s.db.First(&feedback, feedbackID).Error; err != nil {
		logger.Infof("[Feedback] Failed to load feedback: %v", err)
		return
	}

	var reviewLog models.ReviewLog
	if err := s.db.First(&reviewLog, reviewLogID).Error; err != nil {
		logger.Infof("[Feedback] Failed to load review: %v", err)
		s.db.Model(&feedback).Updates(map[string]interface{}{
			"process_status": "failed",
			"error_message":  "Failed to load review log",
		})
		return
	}

	// Build prompt for AI response
	prompt := s.buildFeedbackPrompt(&feedback, &reviewLog)

	// Get LLM config ID to use
	var configID uint = 0
	if llmConfigID != nil {
		configID = *llmConfigID
	}

	// Get AI response using CallWithConfig
	ctx := context.Background()
	content, _, err := s.aiService.CallWithConfig(ctx, configID, prompt)
	if err != nil {
		logger.Infof("[Feedback] AI call failed: %v", err)
		s.db.Model(&feedback).Updates(map[string]interface{}{
			"process_status": "failed",
			"error_message":  err.Error(),
		})
		return
	}

	// Parse AI response for potential score update
	newScore := s.parseScoreFromResponse(content)

	updates := map[string]interface{}{
		"ai_response":    content,
		"process_status": "completed",
	}

	// Check if score should be updated
	if newScore != nil && reviewLog.Score != nil && *newScore != *reviewLog.Score {
		updates["updated_score"] = *newScore
		updates["score_changed"] = true

		// Update the original review's score and status
		reviewUpdates := map[string]interface{}{"score": *newScore}
		if *newScore == 0 {
			reviewUpdates["review_status"] = "unreviewable"
		}
		s.db.Model(&reviewLog).Updates(reviewUpdates)
		logger.Infof("[Feedback] Score updated: %.1f -> %.1f", *reviewLog.Score, *newScore)
	}

	s.db.Model(&feedback).Updates(updates)
	logger.Infof("[Feedback] Completed processing feedback ID=%d", feedbackID)
}

// buildFeedbackPrompt creates the prompt for AI feedback response
func (s *ReviewFeedbackService) buildFeedbackPrompt(feedback *models.ReviewFeedback, reviewLog *models.ReviewLog) string {
	var builder strings.Builder

	builder.WriteString("You are a code review assistant. A user has provided feedback on your previous review.\n\n")

	builder.WriteString("## Original Review Result\n")
	if reviewLog.Score != nil {
		builder.WriteString(fmt.Sprintf("**Score:** %.1f/100\n\n", *reviewLog.Score))
	}
	builder.WriteString(fmt.Sprintf("```\n%s\n```\n\n", reviewLog.ReviewResult))

	builder.WriteString("## User Feedback\n")
	builder.WriteString(fmt.Sprintf("**Type:** %s\n", feedback.FeedbackType))
	builder.WriteString(fmt.Sprintf("**Message:** %s\n\n", feedback.UserMessage))

	builder.WriteString("## Instructions\n")
	builder.WriteString("1. Carefully consider the user's feedback\n")
	builder.WriteString("2. If the user raises valid points that should change the score, provide a new score\n")
	builder.WriteString("3. Explain your reasoning clearly\n")
	builder.WriteString("4. If adjusting the score, clearly state: \"Updated Score: XX/100\"\n")
	builder.WriteString("5. Be respectful and constructive in your response\n")

	return builder.String()
}

// parseScoreFromResponse extracts updated score from AI response if present
func (s *ReviewFeedbackService) parseScoreFromResponse(response string) *float64 {
	lowerResponse := strings.ToLower(response)

	// Look for patterns like "Updated Score: 85/100" or "新评分: 85分"
	keywords := []string{"updated score", "新评分", "调整后评分", "revised score", "new score"}

	for _, keyword := range keywords {
		if strings.Contains(lowerResponse, keyword) {
			// Find the line containing the keyword and extract number
			for _, line := range strings.Split(response, "\n") {
				lineLower := strings.ToLower(line)
				if strings.Contains(lineLower, keyword) {
					// Extract numbers from the line
					var score float64
					for _, word := range strings.Fields(line) {
						word = strings.Trim(word, ",:：分*/")
						if _, err := fmt.Sscanf(word, "%f", &score); err == nil && score >= 0 && score <= 100 {
							return &score
						}
					}
				}
			}
		}
	}
	return nil
}

// ListByReviewLog returns all feedback for a review
func (s *ReviewFeedbackService) ListByReviewLog(reviewLogID uint) ([]models.ReviewFeedback, error) {
	var feedbacks []models.ReviewFeedback
	err := s.db.Where("review_log_id = ?", reviewLogID).
		Preload("User").
		Order("created_at DESC").
		Find(&feedbacks).Error
	return feedbacks, err
}

// GetByID returns a feedback by ID
func (s *ReviewFeedbackService) GetByID(id uint) (*models.ReviewFeedback, error) {
	var feedback models.ReviewFeedback
	err := s.db.Preload("User").Preload("ReviewLog").First(&feedback, id).Error
	return &feedback, err
}
