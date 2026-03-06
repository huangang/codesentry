package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/huangang/codesentry/backend/internal/config"
	"github.com/huangang/codesentry/backend/internal/services"
	"github.com/huangang/codesentry/backend/pkg/response"
	"gorm.io/gorm"
)

type AutoFixHandler struct {
	service *services.AutoFixService
}

func NewAutoFixHandler(db *gorm.DB, aiCfg *config.OpenAIConfig) *AutoFixHandler {
	return &AutoFixHandler{service: services.NewAutoFixService(db, aiCfg)}
}

func (h *AutoFixHandler) RequestFix(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	result, err := h.service.RequestFix(c.Request.Context(), uint(id))
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}
	response.Success(c, gin.H{
		"message": "Fix PR created successfully",
		"pr_url":  result.PRURL,
	})
}

func (h *AutoFixHandler) GetFixStatus(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	status, prURL, err := h.service.GetFixStatus(uint(id))
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}
	response.Success(c, gin.H{
		"fix_status": status,
		"fix_pr_url": prURL,
	})
}
