package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/huangang/codesentry/backend/internal/services"
	"github.com/huangang/codesentry/backend/pkg/response"
	"gorm.io/gorm"
)

type LLMConfigHandler struct {
	llmConfigService *services.LLMConfigService
}

func NewLLMConfigHandler(db *gorm.DB) *LLMConfigHandler {
	return &LLMConfigHandler{
		llmConfigService: services.NewLLMConfigService(db),
	}
}

func (h *LLMConfigHandler) List(c *gin.Context) {
	var req services.LLMConfigListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	resp, err := h.llmConfigService.List(&req)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}

	response.Success(c, resp)
}

func (h *LLMConfigHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid config id")
		return
	}

	config, err := h.llmConfigService.GetByID(uint(id))
	if err != nil {
		response.NotFound(c, "config not found")
		return
	}

	response.Success(c, config)
}

func (h *LLMConfigHandler) Create(c *gin.Context) {
	var req services.CreateLLMConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	config, err := h.llmConfigService.Create(&req)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}

	response.Created(c, config)
}

func (h *LLMConfigHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid config id")
		return
	}

	var req services.UpdateLLMConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	config, err := h.llmConfigService.Update(uint(id), &req)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}

	response.Success(c, config)
}

func (h *LLMConfigHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid config id")
		return
	}

	if err := h.llmConfigService.Delete(uint(id)); err != nil {
		response.ServerError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "config deleted successfully"})
}

func (h *LLMConfigHandler) GetActive(c *gin.Context) {
	configs, err := h.llmConfigService.GetActive()
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}
	response.Success(c, configs)
}
