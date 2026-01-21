package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/huangang/codesentry/backend/internal/services"
	"gorm.io/gorm"
)

type IMBotHandler struct {
	imBotService *services.IMBotService
}

func NewIMBotHandler(db *gorm.DB) *IMBotHandler {
	return &IMBotHandler{
		imBotService: services.NewIMBotService(db),
	}
}

// List returns paginated IM bots
// GET /api/im-bots
func (h *IMBotHandler) List(c *gin.Context) {
	var req services.IMBotListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.imBotService.List(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetByID returns an IM bot by ID
// GET /api/im-bots/:id
func (h *IMBotHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid bot id"})
		return
	}

	bot, err := h.imBotService.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "bot not found"})
		return
	}

	c.JSON(http.StatusOK, bot)
}

// Create creates a new IM bot
// POST /api/im-bots
func (h *IMBotHandler) Create(c *gin.Context) {
	var req services.CreateIMBotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	bot, err := h.imBotService.Create(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, bot)
}

// Update updates an IM bot
// PUT /api/im-bots/:id
func (h *IMBotHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid bot id"})
		return
	}

	var req services.UpdateIMBotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	bot, err := h.imBotService.Update(uint(id), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, bot)
}

// Delete deletes an IM bot
// DELETE /api/im-bots/:id
func (h *IMBotHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid bot id"})
		return
	}

	if err := h.imBotService.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "bot deleted successfully"})
}

// GetAllActive returns all active IM bots
// GET /api/im-bots/active
func (h *IMBotHandler) GetAllActive(c *gin.Context) {
	bots, err := h.imBotService.GetAllActive()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, bots)
}
