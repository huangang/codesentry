package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/huangang/codesentry/backend/internal/services"
	"github.com/huangang/codesentry/backend/pkg/response"
	"gorm.io/gorm"
)

type SystemConfigHandler struct {
	configService  *services.SystemConfigService
	holidayService *services.HolidayService
}

func NewSystemConfigHandler(db *gorm.DB) *SystemConfigHandler {
	return &SystemConfigHandler{
		configService:  services.NewSystemConfigService(db),
		holidayService: services.NewHolidayService(),
	}
}

func (h *SystemConfigHandler) GetLDAPConfig(c *gin.Context) {
	config := h.configService.GetLDAPConfig()
	response.Success(c, config)
}

func (h *SystemConfigHandler) UpdateLDAPConfig(c *gin.Context) {
	var req services.UpdateLDAPConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.configService.UpdateLDAPConfig(&req); err != nil {
		response.ServerError(c, err.Error())
		return
	}

	response.Success(c, h.configService.GetLDAPConfig())
}

func (h *SystemConfigHandler) GetDailyReportConfig(c *gin.Context) {
	config := h.configService.GetDailyReportConfig()
	response.Success(c, config)
}

func (h *SystemConfigHandler) UpdateDailyReportConfig(c *gin.Context) {
	var req services.UpdateDailyReportConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.configService.UpdateDailyReportConfig(&req); err != nil {
		response.ServerError(c, err.Error())
		return
	}

	response.Success(c, h.configService.GetDailyReportConfig())
}

func (h *SystemConfigHandler) GetChunkedReviewConfig(c *gin.Context) {
	config := h.configService.GetChunkedReviewConfig()
	response.Success(c, config)
}

func (h *SystemConfigHandler) UpdateChunkedReviewConfig(c *gin.Context) {
	var req services.UpdateChunkedReviewConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.configService.UpdateChunkedReviewConfig(&req); err != nil {
		response.ServerError(c, err.Error())
		return
	}

	response.Success(c, h.configService.GetChunkedReviewConfig())
}

func (h *SystemConfigHandler) GetFileContextConfig(c *gin.Context) {
	config := h.configService.GetFileContextConfig()
	response.Success(c, config)
}

func (h *SystemConfigHandler) UpdateFileContextConfig(c *gin.Context) {
	var req services.UpdateFileContextConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.configService.UpdateFileContextConfig(&req); err != nil {
		response.ServerError(c, err.Error())
		return
	}

	response.Success(c, h.configService.GetFileContextConfig())
}

func (h *SystemConfigHandler) GetHolidayCountries(c *gin.Context) {
	countries := h.holidayService.GetSupportedCountries()
	response.Success(c, countries)
}

func (h *SystemConfigHandler) GetAuthSessionConfig(c *gin.Context) {
	config := h.configService.GetAuthSessionConfig()
	response.Success(c, config)
}

func (h *SystemConfigHandler) UpdateAuthSessionConfig(c *gin.Context) {
	var req services.UpdateAuthSessionConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.configService.UpdateAuthSessionConfig(&req); err != nil {
		response.ServerError(c, err.Error())
		return
	}

	response.Success(c, h.configService.GetAuthSessionConfig())
}
