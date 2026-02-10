package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/huangang/codesentry/backend/internal/middleware"
	"github.com/huangang/codesentry/backend/internal/services"
	"github.com/huangang/codesentry/backend/pkg/response"
	"gorm.io/gorm"
)

type ProjectHandler struct {
	projectService *services.ProjectService
}

func NewProjectHandler(db *gorm.DB) *ProjectHandler {
	return &ProjectHandler{
		projectService: services.NewProjectService(db),
	}
}

// List returns paginated projects
// GET /api/projects
func (h *ProjectHandler) List(c *gin.Context) {
	var req services.ProjectListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	resp, err := h.projectService.List(&req)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}

	response.Success(c, resp)
}

// GetByID returns a project by ID
// GET /api/projects/:id
func (h *ProjectHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid project id")
		return
	}

	project, err := h.projectService.GetByID(uint(id))
	if err != nil {
		response.NotFound(c, "project not found")
		return
	}

	response.Success(c, project)
}

// Create creates a new project
// POST /api/projects
func (h *ProjectHandler) Create(c *gin.Context) {
	var req services.CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	userID := middleware.GetUserID(c)
	project, err := h.projectService.Create(&req, userID)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}

	response.Created(c, project)
}

// Update updates a project
// PUT /api/projects/:id
func (h *ProjectHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid project id")
		return
	}

	var req services.UpdateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	project, err := h.projectService.Update(uint(id), &req)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}

	response.Success(c, project)
}

// Delete deletes a project
// DELETE /api/projects/:id
func (h *ProjectHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid project id")
		return
	}

	if err := h.projectService.Delete(uint(id)); err != nil {
		response.ServerError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "project deleted successfully"})
}

// GetDefaultPrompt returns the default AI review prompt
// GET /api/projects/default-prompt
func (h *ProjectHandler) GetDefaultPrompt(c *gin.Context) {
	prompt := h.projectService.GetDefaultPrompt()
	response.Success(c, gin.H{"prompt": prompt})
}
