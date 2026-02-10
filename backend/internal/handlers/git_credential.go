package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/huangang/codesentry/backend/internal/models"
	"github.com/huangang/codesentry/backend/internal/services"
	"github.com/huangang/codesentry/backend/pkg/response"
	"gorm.io/gorm"
)

type GitCredentialHandler struct {
	service *services.GitCredentialService
}

func NewGitCredentialHandler(db *gorm.DB) *GitCredentialHandler {
	return &GitCredentialHandler{
		service: services.NewGitCredentialService(db),
	}
}

type GitCredentialResponse struct {
	ID               uint   `json:"id"`
	Name             string `json:"name"`
	Platform         string `json:"platform"`
	BaseURL          string `json:"base_url"`
	AccessTokenMask  string `json:"access_token_mask"`
	WebhookSecretSet bool   `json:"webhook_secret_set"`
	AutoCreate       bool   `json:"auto_create"`
	DefaultEnabled   bool   `json:"default_enabled"`
	FileExtensions   string `json:"file_extensions"`
	ReviewEvents     string `json:"review_events"`
	IgnorePatterns   string `json:"ignore_patterns"`
	IsActive         bool   `json:"is_active"`
	CreatedBy        uint   `json:"created_by"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

func toGitCredentialResponse(cred *models.GitCredential) GitCredentialResponse {
	return GitCredentialResponse{
		ID:               cred.ID,
		Name:             cred.Name,
		Platform:         cred.Platform,
		BaseURL:          cred.BaseURL,
		AccessTokenMask:  cred.MaskAccessToken(),
		WebhookSecretSet: cred.WebhookSecret != "",
		AutoCreate:       cred.AutoCreate,
		DefaultEnabled:   cred.DefaultEnabled,
		FileExtensions:   cred.FileExtensions,
		ReviewEvents:     cred.ReviewEvents,
		IgnorePatterns:   cred.IgnorePatterns,
		IsActive:         cred.IsActive,
		CreatedBy:        cred.CreatedBy,
		CreatedAt:        cred.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:        cred.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}

func (h *GitCredentialHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	params := &services.GitCredentialListParams{
		Page:     page,
		PageSize: pageSize,
		Name:     c.Query("name"),
		Platform: c.Query("platform"),
	}

	if isActiveStr := c.Query("is_active"); isActiveStr != "" {
		isActive := isActiveStr == "true"
		params.IsActive = &isActive
	}

	credentials, total, err := h.service.List(params)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}

	items := make([]GitCredentialResponse, len(credentials))
	for i, cred := range credentials {
		items[i] = toGitCredentialResponse(&cred)
	}

	response.Success(c, gin.H{
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"items":     items,
	})
}

func (h *GitCredentialHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}

	credential, err := h.service.GetByID(uint(id))
	if err != nil {
		response.NotFound(c, "credential not found")
		return
	}

	response.Success(c, toGitCredentialResponse(credential))
}

func (h *GitCredentialHandler) GetActive(c *gin.Context) {
	credentials, err := h.service.GetActive()
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}

	items := make([]GitCredentialResponse, len(credentials))
	for i, cred := range credentials {
		items[i] = toGitCredentialResponse(&cred)
	}

	response.Success(c, items)
}

type CreateGitCredentialRequest struct {
	Name           string `json:"name" binding:"required"`
	Platform       string `json:"platform" binding:"required"`
	BaseURL        string `json:"base_url"`
	AccessToken    string `json:"access_token"`
	WebhookSecret  string `json:"webhook_secret"`
	AutoCreate     bool   `json:"auto_create"`
	DefaultEnabled bool   `json:"default_enabled"`
	FileExtensions string `json:"file_extensions"`
	ReviewEvents   string `json:"review_events"`
	IgnorePatterns string `json:"ignore_patterns"`
	IsActive       bool   `json:"is_active"`
}

func (h *GitCredentialHandler) Create(c *gin.Context) {
	var req CreateGitCredentialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	userID, _ := c.Get("user_id")

	credential := &models.GitCredential{
		Name:           req.Name,
		Platform:       req.Platform,
		BaseURL:        req.BaseURL,
		AccessToken:    req.AccessToken,
		WebhookSecret:  req.WebhookSecret,
		AutoCreate:     req.AutoCreate,
		DefaultEnabled: req.DefaultEnabled,
		FileExtensions: req.FileExtensions,
		ReviewEvents:   req.ReviewEvents,
		IgnorePatterns: req.IgnorePatterns,
		IsActive:       req.IsActive,
		CreatedBy:      userID.(uint),
	}

	if credential.FileExtensions == "" {
		credential.FileExtensions = ".go,.js,.ts,.jsx,.tsx,.py,.java,.c,.cpp,.h,.hpp,.cs,.rb,.php,.swift,.kt,.rs,.vue,.svelte"
	}
	if credential.ReviewEvents == "" {
		credential.ReviewEvents = "push,merge_request"
	}

	if err := h.service.Create(credential); err != nil {
		response.ServerError(c, err.Error())
		return
	}

	services.LogInfo("GitCredential", "Create", "Git credential created: "+credential.Name, &credential.CreatedBy, c.ClientIP(), c.GetHeader("User-Agent"), nil)

	response.Success(c, toGitCredentialResponse(credential))
}

type UpdateGitCredentialRequest struct {
	Name           string `json:"name"`
	Platform       string `json:"platform"`
	BaseURL        string `json:"base_url"`
	AccessToken    string `json:"access_token"`
	WebhookSecret  string `json:"webhook_secret"`
	AutoCreate     *bool  `json:"auto_create"`
	DefaultEnabled *bool  `json:"default_enabled"`
	FileExtensions string `json:"file_extensions"`
	ReviewEvents   string `json:"review_events"`
	IgnorePatterns string `json:"ignore_patterns"`
	IsActive       *bool  `json:"is_active"`
}

func (h *GitCredentialHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}

	credential, err := h.service.GetByID(uint(id))
	if err != nil {
		response.NotFound(c, "credential not found")
		return
	}

	var req UpdateGitCredentialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if req.Name != "" {
		credential.Name = req.Name
	}
	if req.Platform != "" {
		credential.Platform = req.Platform
	}
	if req.BaseURL != "" {
		credential.BaseURL = req.BaseURL
	}
	if req.AccessToken != "" {
		credential.AccessToken = req.AccessToken
	}
	if req.WebhookSecret != "" {
		credential.WebhookSecret = req.WebhookSecret
	}
	if req.AutoCreate != nil {
		credential.AutoCreate = *req.AutoCreate
	}
	if req.DefaultEnabled != nil {
		credential.DefaultEnabled = *req.DefaultEnabled
	}
	if req.FileExtensions != "" {
		credential.FileExtensions = req.FileExtensions
	}
	if req.ReviewEvents != "" {
		credential.ReviewEvents = req.ReviewEvents
	}
	if req.IgnorePatterns != "" {
		credential.IgnorePatterns = req.IgnorePatterns
	}
	if req.IsActive != nil {
		credential.IsActive = *req.IsActive
	}

	if err := h.service.Update(credential); err != nil {
		response.ServerError(c, err.Error())
		return
	}

	userID, _ := c.Get("user_id")
	uid := userID.(uint)
	services.LogInfo("GitCredential", "Update", "Git credential updated: "+credential.Name, &uid, c.ClientIP(), c.GetHeader("User-Agent"), nil)

	response.Success(c, toGitCredentialResponse(credential))
}

func (h *GitCredentialHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}

	credential, err := h.service.GetByID(uint(id))
	if err != nil {
		response.NotFound(c, "credential not found")
		return
	}

	if err := h.service.Delete(uint(id)); err != nil {
		response.ServerError(c, err.Error())
		return
	}

	userID, _ := c.Get("user_id")
	uid := userID.(uint)
	services.LogInfo("GitCredential", "Delete", "Git credential deleted: "+credential.Name, &uid, c.ClientIP(), c.GetHeader("User-Agent"), nil)

	response.Success(c, gin.H{"message": "credential deleted"})
}
