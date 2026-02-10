package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/huangang/codesentry/backend/internal/services"
	"github.com/huangang/codesentry/backend/pkg/response"
	"gorm.io/gorm"
)

type MemberHandler struct {
	memberService *services.MemberService
}

func NewMemberHandler(db *gorm.DB) *MemberHandler {
	return &MemberHandler{
		memberService: services.NewMemberService(db),
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
