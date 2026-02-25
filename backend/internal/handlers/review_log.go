package handlers

import (
	"encoding/csv"
	"fmt"
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

type BatchIDsRequest struct {
	IDs []uint `json:"ids" binding:"required,min=1"`
}

// BatchRetry retries multiple failed reviews.
func (h *ReviewLogHandler) BatchRetry(c *gin.Context) {
	var req BatchIDsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	success, failed := 0, 0
	for _, id := range req.IDs {
		if err := h.retryService.ManualRetry(id); err != nil {
			failed++
		} else {
			success++
		}
	}

	response.Success(c, gin.H{"success": success, "failed": failed})
}

// BatchDelete deletes multiple review logs.
func (h *ReviewLogHandler) BatchDelete(c *gin.Context) {
	var req BatchIDsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	success, failed := 0, 0
	for _, id := range req.IDs {
		if err := h.reviewLogService.Delete(id); err != nil {
			failed++
		} else {
			success++
		}
	}

	response.Success(c, gin.H{"success": success, "failed": failed})
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

// Export exports review logs as CSV with the same filters as List.
func (h *ReviewLogHandler) Export(c *gin.Context) {
	var req services.ReviewLogListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Override pagination to fetch all matching records
	req.Page = 1
	req.PageSize = 10000

	resp, err := h.reviewLogService.List(&req)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=review_logs.csv")

	w := csv.NewWriter(c.Writer)
	// Header
	w.Write([]string{"ID", "Project", "Author", "Branch", "Event Type", "Commit Hash", "Commit Message", "Score", "Status", "Files Changed", "Additions", "Deletions", "Created At"})

	for _, log := range resp.Items {
		projectName := ""
		if log.Project != nil {
			projectName = log.Project.Name
		}
		score := ""
		if log.Score != nil {
			score = fmt.Sprintf("%.0f", *log.Score)
		}
		w.Write([]string{
			strconv.FormatUint(uint64(log.ID), 10),
			projectName,
			log.Author,
			log.Branch,
			log.EventType,
			log.CommitHash,
			log.CommitMessage,
			score,
			log.ReviewStatus,
			strconv.Itoa(log.FilesChanged),
			strconv.Itoa(log.Additions),
			strconv.Itoa(log.Deletions),
			log.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	w.Flush()
}
