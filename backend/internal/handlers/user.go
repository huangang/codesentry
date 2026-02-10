package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/huangang/codesentry/backend/internal/middleware"
	"github.com/huangang/codesentry/backend/internal/models"
	"github.com/huangang/codesentry/backend/pkg/response"
	"gorm.io/gorm"
)

type UserHandler struct {
	db *gorm.DB
}

func NewUserHandler(db *gorm.DB) *UserHandler {
	return &UserHandler{db: db}
}

func (h *UserHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	username := c.Query("username")
	role := c.Query("role")
	authType := c.Query("auth_type")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var users []models.User
	var total int64

	query := h.db.Model(&models.User{})

	if username != "" {
		query = query.Where("username LIKE ?", "%"+username+"%")
	}
	if role != "" {
		query = query.Where("role = ?", role)
	}
	if authType != "" {
		query = query.Where("auth_type = ?", authType)
	}

	query.Count(&total)
	query.Order("id ASC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&users)

	response.Success(c, gin.H{
		"items":     users,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

type UpdateUserRequest struct {
	Role     *string `json:"role"`
	IsActive *bool   `json:"is_active"`
	Nickname *string `json:"nickname"`
}

func (h *UserHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}

	currentUserID := middleware.GetUserID(c)
	if uint(id) == currentUserID {
		response.BadRequest(c, "cannot modify your own account")
		return
	}

	var user models.User
	if err := h.db.First(&user, id).Error; err != nil {
		response.NotFound(c, "user not found")
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	updates := make(map[string]interface{})
	if req.Role != nil {
		if *req.Role != "admin" && *req.Role != "user" {
			response.BadRequest(c, "invalid role, must be 'admin' or 'user'")
			return
		}
		updates["role"] = *req.Role
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	if req.Nickname != nil {
		updates["nickname"] = *req.Nickname
	}

	if len(updates) == 0 {
		response.BadRequest(c, "no fields to update")
		return
	}

	if err := h.db.Model(&user).Updates(updates).Error; err != nil {
		response.ServerError(c, err.Error())
		return
	}

	h.db.First(&user, id)
	response.Success(c, user)
}

func (h *UserHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}

	currentUserID := middleware.GetUserID(c)
	if uint(id) == currentUserID {
		response.BadRequest(c, "cannot delete your own account")
		return
	}

	var user models.User
	if err := h.db.First(&user, id).Error; err != nil {
		response.NotFound(c, "user not found")
		return
	}

	if err := h.db.Delete(&user).Error; err != nil {
		response.ServerError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "user deleted"})
}
