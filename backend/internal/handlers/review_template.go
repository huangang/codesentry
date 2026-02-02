package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/huangang/codesentry/backend/internal/models"
	"github.com/huangang/codesentry/backend/internal/services"
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

// List godoc
// @Summary List review templates
// @Tags ReviewTemplates
// @Param type query string false "Filter by type (general, frontend, backend, security)"
// @Success 200 {array} models.ReviewTemplate
// @Router /api/review-templates [get]
func (h *ReviewTemplateHandler) List(c *gin.Context) {
	templateType := c.Query("type")
	templates, err := h.service.List(templateType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, templates)
}

// Get godoc
// @Summary Get review template by ID
// @Tags ReviewTemplates
// @Param id path int true "Template ID"
// @Success 200 {object} models.ReviewTemplate
// @Router /api/review-templates/{id} [get]
func (h *ReviewTemplateHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	template, err := h.service.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "template not found"})
		return
	}

	c.JSON(http.StatusOK, template)
}

// Create godoc
// @Summary Create review template
// @Tags ReviewTemplates
// @Accept json
// @Produce json
// @Param template body models.ReviewTemplate true "Template"
// @Success 201 {object} models.ReviewTemplate
// @Router /api/review-templates [post]
func (h *ReviewTemplateHandler) Create(c *gin.Context) {
	var template models.ReviewTemplate
	if err := c.ShouldBindJSON(&template); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user ID from context
	if userID, exists := c.Get("user_id"); exists {
		template.CreatedBy = userID.(uint)
	}

	template.IsBuiltIn = false // User-created templates are never built-in

	if err := h.service.Create(&template); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, template)
}

// Update godoc
// @Summary Update review template
// @Tags ReviewTemplates
// @Accept json
// @Produce json
// @Param id path int true "Template ID"
// @Param template body models.ReviewTemplate true "Template"
// @Success 200 {object} models.ReviewTemplate
// @Router /api/review-templates/{id} [put]
func (h *ReviewTemplateHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var template models.ReviewTemplate
	if err := c.ShouldBindJSON(&template); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	template.ID = uint(id)
	if err := h.service.Update(&template); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, template)
}

// Delete godoc
// @Summary Delete review template
// @Tags ReviewTemplates
// @Param id path int true "Template ID"
// @Success 204
// @Router /api/review-templates/{id} [delete]
func (h *ReviewTemplateHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.service.Delete(uint(id)); err != nil {
		if err == services.ErrBuiltInTemplate {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// SeedTemplates seeds default templates (called on startup)
func (h *ReviewTemplateHandler) SeedTemplates() error {
	return h.service.SeedDefaultTemplates()
}
