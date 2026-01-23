package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/huangang/codesentry/backend/internal/services"
)

type DailyReportHandler struct {
	service *services.DailyReportService
}

func NewDailyReportHandler(service *services.DailyReportService) *DailyReportHandler {
	return &DailyReportHandler{service: service}
}

func (h *DailyReportHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	reports, total, err := h.service.List(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"items":     reports,
	})
}

func (h *DailyReportHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	report, err := h.service.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "report not found"})
		return
	}

	c.JSON(http.StatusOK, report)
}

func (h *DailyReportHandler) Generate(c *gin.Context) {
	report, err := h.service.GenerateReport()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, report)
}

func (h *DailyReportHandler) Resend(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.service.ResendNotification(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "notification resent"})
}
