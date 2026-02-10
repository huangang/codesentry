package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/huangang/codesentry/backend/internal/models"
	"github.com/huangang/codesentry/backend/internal/services"
	"github.com/huangang/codesentry/backend/pkg/response"
	"gorm.io/gorm"
)

type PromptHandler struct {
	service *services.PromptService
}

func NewPromptHandler(db *gorm.DB) *PromptHandler {
	return &PromptHandler{
		service: services.NewPromptService(db),
	}
}

func (h *PromptHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	name := c.Query("name")

	var isSystem *bool
	if isSystemStr := c.Query("is_system"); isSystemStr != "" {
		val := isSystemStr == "true"
		isSystem = &val
	}

	result, err := h.service.List(services.PromptListParams{
		Page:     page,
		PageSize: pageSize,
		Name:     name,
		IsSystem: isSystem,
	})
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}

	response.Success(c, result)
}

func (h *PromptHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "Invalid ID")
		return
	}

	prompt, err := h.service.GetByID(uint(id))
	if err != nil {
		response.NotFound(c, "Prompt not found")
		return
	}

	response.Success(c, prompt)
}

func (h *PromptHandler) GetDefault(c *gin.Context) {
	prompt, err := h.service.GetDefault()
	if err != nil {
		response.NotFound(c, "Default prompt not found")
		return
	}

	response.Success(c, prompt)
}

func (h *PromptHandler) GetAllActive(c *gin.Context) {
	prompts, err := h.service.GetAllActive()
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}

	response.Success(c, prompts)
}

func (h *PromptHandler) Create(c *gin.Context) {
	var prompt models.PromptTemplate
	if err := c.ShouldBindJSON(&prompt); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.service.Create(&prompt); err != nil {
		response.ServerError(c, err.Error())
		return
	}

	response.Created(c, prompt)
}

func (h *PromptHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "Invalid ID")
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.service.Update(uint(id), updates); err != nil {
		response.ServerError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "Updated successfully"})
}

func (h *PromptHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "Invalid ID")
		return
	}

	if err := h.service.Delete(uint(id)); err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Forbidden(c, "Cannot delete system prompt")
			return
		}
		response.ServerError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "Deleted successfully"})
}

func (h *PromptHandler) SetDefault(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "Invalid ID")
		return
	}

	if err := h.service.SetDefault(uint(id)); err != nil {
		response.ServerError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "Set as default successfully"})
}
