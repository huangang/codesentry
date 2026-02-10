package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/huangang/codesentry/backend/internal/services"
	"github.com/huangang/codesentry/backend/pkg/response"
	"gorm.io/gorm"
)

type SystemLogHandler struct {
	systemLogService *services.SystemLogService
}

func NewSystemLogHandler(db *gorm.DB) *SystemLogHandler {
	return &SystemLogHandler{
		systemLogService: services.NewSystemLogService(db),
	}
}

func (h *SystemLogHandler) List(c *gin.Context) {
	var req services.SystemLogListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	resp, err := h.systemLogService.List(&req)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}

	response.Success(c, resp)
}

func (h *SystemLogHandler) GetModules(c *gin.Context) {
	modules, err := h.systemLogService.GetModules()
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}
	response.Success(c, gin.H{"modules": modules})
}

func (h *SystemLogHandler) Cleanup(c *gin.Context) {
	var req struct {
		Days int `json:"days"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	days := req.Days
	if days <= 0 {
		days = h.systemLogService.GetRetentionDays()
	}

	deleted, err := h.systemLogService.CleanupOldLogs(days)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"deleted":        deleted,
		"retention_days": days,
	})
}

func (h *SystemLogHandler) GetRetentionDays(c *gin.Context) {
	days := h.systemLogService.GetRetentionDays()
	response.Success(c, gin.H{"retention_days": days})
}

func (h *SystemLogHandler) SetRetentionDays(c *gin.Context) {
	var req struct {
		Days int `json:"days" binding:"required,min=1,max=365"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.systemLogService.SetRetentionDays(req.Days); err != nil {
		response.ServerError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"retention_days": req.Days})
}
