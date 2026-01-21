package handlers

import (
	"net/http"
	"strconv"

	"github.com/huangang/codesentry/backend/internal/services"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ReviewLogHandler struct {
	reviewLogService *services.ReviewLogService
}

func NewReviewLogHandler(db *gorm.DB) *ReviewLogHandler {
	return &ReviewLogHandler{
		reviewLogService: services.NewReviewLogService(db),
	}
}

// List returns paginated review logs
// GET /api/review-logs
func (h *ReviewLogHandler) List(c *gin.Context) {
	var req services.ReviewLogListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.reviewLogService.List(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetByID returns a review log by ID
// GET /api/review-logs/:id
func (h *ReviewLogHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid review log id"})
		return
	}

	log, err := h.reviewLogService.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "review log not found"})
		return
	}

	c.JSON(http.StatusOK, log)
}
