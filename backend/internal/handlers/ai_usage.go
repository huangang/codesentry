package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/huangang/codesentry/backend/internal/services"
	"github.com/huangang/codesentry/backend/pkg/response"
	"gorm.io/gorm"
)

// AIUsageHandler provides endpoints for AI usage statistics.
type AIUsageHandler struct {
	usageService *services.AIUsageService
}

func NewAIUsageHandler(db *gorm.DB) *AIUsageHandler {
	return &AIUsageHandler{
		usageService: services.NewAIUsageService(db),
	}
}

// GetStats returns aggregated AI usage statistics.
func (h *AIUsageHandler) GetStats(c *gin.Context) {
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	var projectID *uint
	if pidStr := c.Query("project_id"); pidStr != "" {
		if pid, err := strconv.ParseUint(pidStr, 10, 32); err == nil {
			p := uint(pid)
			projectID = &p
		}
	}

	stats, err := h.usageService.GetStats(startDate, endDate, projectID)
	if err != nil {
		response.ServerError(c, "failed to get AI usage stats: "+err.Error())
		return
	}

	response.Success(c, stats)
}

// GetDailyTrend returns daily AI usage data for charting.
func (h *AIUsageHandler) GetDailyTrend(c *gin.Context) {
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	var projectID *uint
	if pidStr := c.Query("project_id"); pidStr != "" {
		if pid, err := strconv.ParseUint(pidStr, 10, 32); err == nil {
			p := uint(pid)
			projectID = &p
		}
	}

	trend, err := h.usageService.GetDailyTrend(startDate, endDate, projectID)
	if err != nil {
		response.ServerError(c, "failed to get AI usage trend: "+err.Error())
		return
	}

	response.Success(c, trend)
}

// GetProviderBreakdown returns AI usage grouped by provider/model.
func (h *AIUsageHandler) GetProviderBreakdown(c *gin.Context) {
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	providers, err := h.usageService.GetProviderBreakdown(startDate, endDate)
	if err != nil {
		response.ServerError(c, "failed to get provider breakdown: "+err.Error())
		return
	}

	response.Success(c, providers)
}
