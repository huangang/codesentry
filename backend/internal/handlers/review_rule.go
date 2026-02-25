package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/huangang/codesentry/backend/internal/models"
	"github.com/huangang/codesentry/backend/internal/services"
	"github.com/huangang/codesentry/backend/pkg/response"
	"gorm.io/gorm"
)

type ReviewRuleHandler struct {
	service *services.RuleEngineService
}

func NewReviewRuleHandler(db *gorm.DB) *ReviewRuleHandler {
	return &ReviewRuleHandler{service: services.NewRuleEngineService(db)}
}

func (h *ReviewRuleHandler) List(c *gin.Context) {
	var projectID *uint
	if pid := c.Query("project_id"); pid != "" {
		id, _ := strconv.ParseUint(pid, 10, 32)
		uid := uint(id)
		projectID = &uid
	}

	rules, err := h.service.List(projectID)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}
	response.Success(c, rules)
}

func (h *ReviewRuleHandler) Create(c *gin.Context) {
	var rule models.ReviewRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.service.Create(&rule); err != nil {
		response.ServerError(c, err.Error())
		return
	}
	response.Success(c, rule)
}

func (h *ReviewRuleHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	rule, rErr := h.service.Update(uint(id), updates)
	if rErr != nil {
		response.ServerError(c, rErr.Error())
		return
	}
	response.Success(c, rule)
}

func (h *ReviewRuleHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	if err := h.service.Delete(uint(id)); err != nil {
		response.ServerError(c, err.Error())
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}

// Evaluate runs rules against a specific review log (for testing).
func (h *ReviewRuleHandler) Evaluate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid review log id")
		return
	}

	db := models.GetDB()
	var reviewLog models.ReviewLog
	if err := db.First(&reviewLog, uint(id)).Error; err != nil {
		response.NotFound(c, "review log not found")
		return
	}

	result := h.service.Evaluate(&reviewLog)
	response.Success(c, result)
}
