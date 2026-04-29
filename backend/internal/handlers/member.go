package handlers

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/huangang/codesentry/backend/internal/config"
	"github.com/huangang/codesentry/backend/internal/services"
	"github.com/huangang/codesentry/backend/pkg/response"
	"gorm.io/gorm"
)

type MemberHandler struct {
	memberService *services.MemberService
}

func NewMemberHandler(db *gorm.DB, openAICfg *config.OpenAIConfig) *MemberHandler {
	aiService := services.NewAIService(db, openAICfg)
	return &MemberHandler{
		memberService: services.NewMemberService(db, aiService),
	}
}

func (h *MemberHandler) List(c *gin.Context) {
	var req services.MemberListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	result, err := h.memberService.List(&req)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}

	response.Success(c, result)
}

func (h *MemberHandler) GetDetail(c *gin.Context) {
	var req services.MemberDetailRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	author := c.Query("author")
	if author == "" {
		response.BadRequest(c, "author is required")
		return
	}
	req.Author = author

	result, err := h.memberService.GetDetail(&req)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}

	response.Success(c, result)
}

func (h *MemberHandler) GetTeamOverview(c *gin.Context) {
	var req services.TeamOverviewRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	result, err := h.memberService.GetTeamOverview(&req)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}

	response.Success(c, result)
}

func (h *MemberHandler) GetHeatmap(c *gin.Context) {
	var req services.HeatmapRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	result, err := h.memberService.GetHeatmap(&req)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}

	response.Success(c, result)
}

func (h *MemberHandler) GetWorkSummary(c *gin.Context) {
	author := c.Query("author")
	if author == "" {
		response.BadRequest(c, "author is required")
		return
	}

	req := &services.WorkSummaryRequest{
		Author: author,
		Date:   c.Query("date"),
	}

	result, err := h.memberService.GetWorkSummary(req)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}

	response.Success(c, result)
}

func (h *MemberHandler) GenerateWorkSummary(c *gin.Context) {
	var req services.WorkSummaryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Fallback to query params for backward compatibility
		req.Author = c.Query("author")
		req.Date = c.Query("date")
	}

	if req.Author == "" {
		response.BadRequest(c, "author is required")
		return
	}

	result, err := h.memberService.GenerateWorkSummary(&req)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}

	response.Success(c, result)
}

func (h *MemberHandler) ListWorkSummaries(c *gin.Context) {
	var req services.WorkSummaryListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	author := c.Query("author")
	if author == "" {
		response.BadRequest(c, "author is required")
		return
	}
	req.Author = author

	result, err := h.memberService.ListWorkSummaries(&req)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}

	response.Success(c, result)
}

// Alias management

func (h *MemberHandler) ListAliases(c *gin.Context) {
	items, err := h.memberService.ListAliases()
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}
	response.Success(c, items)
}

func (h *MemberHandler) CreateAlias(c *gin.Context) {
	var req services.CreateAliasRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	item, err := h.memberService.CreateAlias(&req)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}
	response.Success(c, item)
}

func (h *MemberHandler) DeleteAlias(c *gin.Context) {
	var id uint
	if _, err := fmt.Sscanf(c.Param("id"), "%d", &id); err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	if err := h.memberService.DeleteAlias(id); err != nil {
		response.ServerError(c, err.Error())
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}
