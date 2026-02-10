package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/huangang/codesentry/backend/internal/config"
	"github.com/huangang/codesentry/backend/internal/services"
	"github.com/huangang/codesentry/backend/pkg/response"
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

func (h *ReviewLogHandler) List(c *gin.Context) {
	var req services.ReviewLogListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	resp, err := h.reviewLogService.List(&req)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}

	response.Success(c, resp)
}

func (h *ReviewLogHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid review log id")
		return
	}

	log, err := h.reviewLogService.GetByID(uint(id))
	if err != nil {
		response.NotFound(c, "review log not found")
		return
	}

	response.Success(c, log)
}

func (h *ReviewLogHandler) Retry(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid review log id")
		return
	}

	if err := h.retryService.ManualRetry(uint(id)); err != nil {
		response.ServerError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "retry initiated"})
}

func (h *ReviewLogHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid review log id")
		return
	}

	if err := h.reviewLogService.Delete(uint(id)); err != nil {
		response.ServerError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "review log deleted"})
}

func (h *ReviewLogHandler) CreateManualCommit(c *gin.Context) {
	var req services.ManualCommitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	log, err := h.reviewLogService.CreateManualCommit(&req)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}

	response.Success(c, log)
}

func (h *ReviewLogHandler) ImportCommits(c *gin.Context) {
	var req services.ImportCommitsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	resp, err := h.importCommitsService.ImportCommits(&req)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}

	response.Success(c, resp)
}
