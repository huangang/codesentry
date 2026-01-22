package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/huangang/codesentry/backend/internal/services"
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

// List returns paginated LLM configs
// GET /api/llm-configs
func (h *LLMConfigHandler) List(c *gin.Context) {
	var req services.LLMConfigListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.llmConfigService.List(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetByID returns a LLM config by ID
// GET /api/llm-configs/:id
func (h *LLMConfigHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid config id"})
		return
	}

	config, err := h.llmConfigService.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "config not found"})
		return
	}

	c.JSON(http.StatusOK, config)
}

// Create creates a new LLM config
// POST /api/llm-configs
func (h *LLMConfigHandler) Create(c *gin.Context) {
	var req services.CreateLLMConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config, err := h.llmConfigService.Create(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, config)
}

// Update updates a LLM config
// PUT /api/llm-configs/:id
func (h *LLMConfigHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid config id"})
		return
	}

	var req services.UpdateLLMConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config, err := h.llmConfigService.Update(uint(id), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, config)
}

// Delete deletes a LLM config
// DELETE /api/llm-configs/:id
func (h *LLMConfigHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid config id"})
		return
	}

	if err := h.llmConfigService.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "config deleted successfully"})
}

// GetActive returns all active LLM configs
// GET /api/llm-configs/active
func (h *LLMConfigHandler) GetActive(c *gin.Context) {
	configs, err := h.llmConfigService.GetActive()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, configs)
}
