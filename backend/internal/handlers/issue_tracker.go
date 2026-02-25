package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/huangang/codesentry/backend/internal/models"
	"github.com/huangang/codesentry/backend/internal/services"
	"github.com/huangang/codesentry/backend/pkg/response"
	"gorm.io/gorm"
)

type IssueTrackerHandler struct {
	service *services.IssueTrackerService
}

func NewIssueTrackerHandler(db *gorm.DB) *IssueTrackerHandler {
	return &IssueTrackerHandler{service: services.NewIssueTrackerService(db)}
}

func (h *IssueTrackerHandler) List(c *gin.Context) {
	trackers, err := h.service.List()
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}
	response.Success(c, trackers)
}

func (h *IssueTrackerHandler) Create(c *gin.Context) {
	var tracker models.IssueTracker
	if err := c.ShouldBindJSON(&tracker); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.service.Create(&tracker); err != nil {
		response.ServerError(c, err.Error())
		return
	}
	response.Success(c, tracker)
}

func (h *IssueTrackerHandler) Update(c *gin.Context) {
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
	tracker, err := h.service.Update(uint(id), updates)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}
	response.Success(c, tracker)
}

func (h *IssueTrackerHandler) Delete(c *gin.Context) {
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
