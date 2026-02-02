package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/huangang/codesentry/backend/internal/config"
	"github.com/huangang/codesentry/backend/internal/models"
	"github.com/huangang/codesentry/backend/internal/services"
	"gorm.io/gorm"
)

type ReviewFeedbackHandler struct {
	service *services.ReviewFeedbackService
}

func NewReviewFeedbackHandler(db *gorm.DB, openAICfg *config.OpenAIConfig) *ReviewFeedbackHandler {
	return &ReviewFeedbackHandler{
		service: services.NewReviewFeedbackService(db, openAICfg),
	}
}

// CreateFeedbackRequest represents the request body for creating feedback
type CreateFeedbackRequest struct {
	ReviewLogID  uint   `json:"review_log_id" binding:"required"`
	FeedbackType string `json:"feedback_type" binding:"required,oneof=agree disagree question clarification"`
	UserMessage  string `json:"user_message" binding:"required"`
}

// Create handles POST /review-feedbacks
func (h *ReviewFeedbackHandler) Create(c *gin.Context) {
	var req CreateFeedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	feedback := &models.ReviewFeedback{
		ReviewLogID:   req.ReviewLogID,
		UserID:        userID.(uint),
		FeedbackType:  req.FeedbackType,
		UserMessage:   req.UserMessage,
		ProcessStatus: "pending",
	}

	ctx := context.Background()
	if err := h.service.Create(ctx, feedback); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, feedback)
}

// ListByReview handles GET /review-logs/:id/feedbacks
func (h *ReviewFeedbackHandler) ListByReview(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid review ID"})
		return
	}

	feedbacks, err := h.service.ListByReviewLog(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, feedbacks)
}

// Get handles GET /review-feedbacks/:id
func (h *ReviewFeedbackHandler) Get(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid feedback ID"})
		return
	}

	feedback, err := h.service.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "feedback not found"})
		return
	}

	c.JSON(http.StatusOK, feedback)
}
