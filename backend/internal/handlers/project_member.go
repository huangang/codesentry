package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/huangang/codesentry/backend/internal/models"
	"github.com/huangang/codesentry/backend/pkg/response"
	"gorm.io/gorm"
)

// ProjectMemberHandler provides CRUD endpoints for project members.
type ProjectMemberHandler struct {
	db *gorm.DB
}

func NewProjectMemberHandler(db *gorm.DB) *ProjectMemberHandler {
	return &ProjectMemberHandler{db: db}
}

type AddMemberRequest struct {
	UserID uint   `json:"user_id" binding:"required"`
	Role   string `json:"role" binding:"required"` // owner, maintainer, viewer
}

type UpdateMemberRequest struct {
	Role string `json:"role" binding:"required"`
}

// List returns all members of a project.
func (h *ProjectMemberHandler) List(c *gin.Context) {
	projectID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid project id")
		return
	}

	var members []models.ProjectMember
	if err := h.db.Where("project_id = ?", projectID).
		Preload("User").
		Find(&members).Error; err != nil {
		response.ServerError(c, err.Error())
		return
	}

	response.Success(c, members)
}

// Add adds a user to a project with the specified role.
func (h *ProjectMemberHandler) Add(c *gin.Context) {
	projectID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid project id")
		return
	}

	var req AddMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Validate role
	if req.Role != "owner" && req.Role != "maintainer" && req.Role != "viewer" {
		response.BadRequest(c, "invalid role, must be 'owner', 'maintainer', or 'viewer'")
		return
	}

	// Check project exists
	var project models.Project
	if err := h.db.First(&project, projectID).Error; err != nil {
		response.NotFound(c, "project not found")
		return
	}

	// Check user exists
	var user models.User
	if err := h.db.First(&user, req.UserID).Error; err != nil {
		response.NotFound(c, "user not found")
		return
	}

	// Check if member already exists
	var existing models.ProjectMember
	if err := h.db.Where("project_id = ? AND user_id = ?", projectID, req.UserID).First(&existing).Error; err == nil {
		response.BadRequest(c, "user is already a member of this project")
		return
	}

	member := models.ProjectMember{
		ProjectID: uint(projectID),
		UserID:    req.UserID,
		Role:      req.Role,
	}

	if err := h.db.Create(&member).Error; err != nil {
		response.ServerError(c, err.Error())
		return
	}

	// Reload with user info
	h.db.Preload("User").First(&member, member.ID)
	response.Success(c, member)
}

// Update updates a member's role.
func (h *ProjectMemberHandler) Update(c *gin.Context) {
	memberID, err := strconv.ParseUint(c.Param("memberID"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid member id")
		return
	}

	var req UpdateMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if req.Role != "owner" && req.Role != "maintainer" && req.Role != "viewer" {
		response.BadRequest(c, "invalid role, must be 'owner', 'maintainer', or 'viewer'")
		return
	}

	var member models.ProjectMember
	if err := h.db.First(&member, memberID).Error; err != nil {
		response.NotFound(c, "member not found")
		return
	}

	member.Role = req.Role
	if err := h.db.Save(&member).Error; err != nil {
		response.ServerError(c, err.Error())
		return
	}

	h.db.Preload("User").First(&member, member.ID)
	response.Success(c, member)
}

// Remove removes a member from a project.
func (h *ProjectMemberHandler) Remove(c *gin.Context) {
	memberID, err := strconv.ParseUint(c.Param("memberID"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid member id")
		return
	}

	var member models.ProjectMember
	if err := h.db.First(&member, memberID).Error; err != nil {
		response.NotFound(c, "member not found")
		return
	}

	if err := h.db.Delete(&member).Error; err != nil {
		response.ServerError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "member removed"})
}
