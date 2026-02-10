package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/huangang/codesentry/backend/internal/services"
	"github.com/huangang/codesentry/backend/pkg/response"
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

func (h *DashboardHandler) GetStats(c *gin.Context) {
	var req services.DashboardStatsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	resp, err := h.dashboardService.GetStats(&req)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}

	response.Success(c, resp)
}
