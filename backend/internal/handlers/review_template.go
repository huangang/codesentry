package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/huangang/codesentry/backend/internal/models"
	"github.com/huangang/codesentry/backend/internal/services"
	"github.com/huangang/codesentry/backend/pkg/response"
	"gorm.io/gorm"
)

type ReviewTemplateHandler struct {
	service *services.ReviewTemplateService
}

func NewReviewTemplateHandler(db *gorm.DB) *ReviewTemplateHandler {
	return &ReviewTemplateHandler{
		service: services.NewReviewTemplateService(db),
	}
}

func (h *ReviewTemplateHandler) List(c *gin.Context) {
	templateType := c.Query("type")
	templates, err := h.service.List(templateType)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}
	response.Success(c, templates)
}

func (h *ReviewTemplateHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}

	template, err := h.service.GetByID(uint(id))
	if err != nil {
		response.NotFound(c, "template not found")
		return
	}

	response.Success(c, template)
}

func (h *ReviewTemplateHandler) Create(c *gin.Context) {
	var template models.ReviewTemplate
	if err := c.ShouldBindJSON(&template); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if userID, exists := c.Get("user_id"); exists {
		template.CreatedBy = userID.(uint)
	}

	template.IsBuiltIn = false

	if err := h.service.Create(&template); err != nil {
		response.ServerError(c, err.Error())
		return
	}

	response.Created(c, template)
}

func (h *ReviewTemplateHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}

	var template models.ReviewTemplate
	if err := c.ShouldBindJSON(&template); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	template.ID = uint(id)
	if err := h.service.Update(&template); err != nil {
		response.ServerError(c, err.Error())
		return
	}

	response.Success(c, template)
}

func (h *ReviewTemplateHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}

	if err := h.service.Delete(uint(id)); err != nil {
		if err == services.ErrBuiltInTemplate {
			response.Forbidden(c, err.Error())
			return
		}
		response.ServerError(c, err.Error())
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *ReviewTemplateHandler) SeedTemplates() error {
	return h.service.SeedDefaultTemplates()
}
