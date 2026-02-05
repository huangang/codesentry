package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/huangang/codesentry/backend/internal/config"
	"github.com/huangang/codesentry/backend/internal/services"
	"gorm.io/gorm"
)

type ReviewLogHandler struct {
	db                   *gorm.DB
	reviewLogService     *services.ReviewLogService
	retryService         *services.RetryService
	importCommitsService *services.ImportCommitsService
}

func NewReviewLogHandler(db *gorm.DB, aiCfg *config.OpenAIConfig) *ReviewLogHandler {
	return &ReviewLogHandler{
		db:                   db,
		reviewLogService:     services.NewReviewLogService(db),
		retryService:         services.NewRetryService(db, aiCfg),
		importCommitsService: services.NewImportCommitsService(db),
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

func (h *ReviewLogHandler) Retry(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid review log id"})
		return
	}

	if err := h.retryService.ManualRetry(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "retry initiated"})
}

func (h *ReviewLogHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid review log id"})
		return
	}

	if err := h.reviewLogService.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "review log deleted"})
}

func (h *ReviewLogHandler) CreateManualCommit(c *gin.Context) {
	var req services.ManualCommitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log, err := h.reviewLogService.CreateManualCommit(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, log)
}

func (h *ReviewLogHandler) ImportCommits(c *gin.Context) {
	var req services.ImportCommitsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.importCommitsService.ImportCommits(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}
