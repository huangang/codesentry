package handlers

import (
	"context"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/huangang/codesentry/backend/internal/config"
	"github.com/huangang/codesentry/backend/internal/models"
	"github.com/huangang/codesentry/backend/internal/services"
	"github.com/huangang/codesentry/backend/pkg/response"
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

type CreateFeedbackRequest struct {
	ReviewLogID  uint   `json:"review_log_id" binding:"required"`
	FeedbackType string `json:"feedback_type" binding:"required,oneof=agree disagree question clarification"`
	UserMessage  string `json:"user_message" binding:"required"`
}

func (h *ReviewFeedbackHandler) Create(c *gin.Context) {
	var req CreateFeedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "unauthorized")
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
		response.ServerError(c, err.Error())
		return
	}

	response.Created(c, feedback)
}

func (h *ReviewFeedbackHandler) ListByReview(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid review ID")
		return
	}

	feedbacks, err := h.service.ListByReviewLog(uint(id))
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}

	response.Success(c, feedbacks)
}

func (h *ReviewFeedbackHandler) Get(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid feedback ID")
		return
	}

	feedback, err := h.service.GetByID(uint(id))
	if err != nil {
		response.NotFound(c, "feedback not found")
		return
	}

	response.Success(c, feedback)
}
