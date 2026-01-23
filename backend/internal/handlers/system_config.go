package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/huangang/codesentry/backend/internal/services"
	"gorm.io/gorm"
)

type SystemConfigHandler struct {
	configService *services.SystemConfigService
}

func NewSystemConfigHandler(db *gorm.DB) *SystemConfigHandler {
	return &SystemConfigHandler{
		configService: services.NewSystemConfigService(db),
	}
}

func (h *SystemConfigHandler) GetLDAPConfig(c *gin.Context) {
	config := h.configService.GetLDAPConfig()
	c.JSON(http.StatusOK, config)
}

func (h *SystemConfigHandler) UpdateLDAPConfig(c *gin.Context) {
	var req services.UpdateLDAPConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.configService.UpdateLDAPConfig(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, h.configService.GetLDAPConfig())
}

func (h *SystemConfigHandler) GetDailyReportConfig(c *gin.Context) {
	config := h.configService.GetDailyReportConfig()
	c.JSON(http.StatusOK, config)
}

func (h *SystemConfigHandler) UpdateDailyReportConfig(c *gin.Context) {
	var req services.UpdateDailyReportConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.configService.UpdateDailyReportConfig(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, h.configService.GetDailyReportConfig())
}
