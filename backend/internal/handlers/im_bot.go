package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/huangang/codesentry/backend/internal/services"
	"github.com/huangang/codesentry/backend/pkg/response"
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

func (h *IMBotHandler) List(c *gin.Context) {
	var req services.IMBotListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	resp, err := h.imBotService.List(&req)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}

	response.Success(c, resp)
}

func (h *IMBotHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid bot id")
		return
	}

	bot, err := h.imBotService.GetByID(uint(id))
	if err != nil {
		response.NotFound(c, "bot not found")
		return
	}

	response.Success(c, bot)
}

func (h *IMBotHandler) Create(c *gin.Context) {
	var req services.CreateIMBotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	bot, err := h.imBotService.Create(&req)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}

	response.Created(c, bot)
}

func (h *IMBotHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid bot id")
		return
	}

	var req services.UpdateIMBotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	bot, err := h.imBotService.Update(uint(id), &req)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}

	response.Success(c, bot)
}

func (h *IMBotHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid bot id")
		return
	}

	if err := h.imBotService.Delete(uint(id)); err != nil {
		response.ServerError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "bot deleted successfully"})
}

func (h *IMBotHandler) GetAllActive(c *gin.Context) {
	bots, err := h.imBotService.GetAllActive()
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}

	response.Success(c, bots)
}
