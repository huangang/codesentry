package handlers

import (
	"net/http"

	"github.com/huangang/codesentry/backend/internal/services"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type DashboardHandler struct {
	dashboardService *services.DashboardService
}

func NewDashboardHandler(db *gorm.DB) *DashboardHandler {
	return &DashboardHandler{
		dashboardService: services.NewDashboardService(db),
	}
}

// GetStats returns dashboard statistics
// GET /api/dashboard/stats
func (h *DashboardHandler) GetStats(c *gin.Context) {
	var req services.DashboardStatsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.dashboardService.GetStats(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}
