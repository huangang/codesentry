package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/huangang/codesentry/backend/internal/services"
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.systemLogService.List(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *SystemLogHandler) GetModules(c *gin.Context) {
	modules, err := h.systemLogService.GetModules()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"modules": modules})
}
