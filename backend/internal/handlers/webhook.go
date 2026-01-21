package handlers

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/huangang/codesentry/backend/internal/config"
	"github.com/huangang/codesentry/backend/internal/services"
	"gorm.io/gorm"
)

type WebhookHandler struct {
	webhookService *services.WebhookService
	projectService *services.ProjectService
}

func NewWebhookHandler(db *gorm.DB, aiCfg *config.OpenAIConfig) *WebhookHandler {
	return &WebhookHandler{
		webhookService: services.NewWebhookService(db, aiCfg),
		projectService: services.NewProjectService(db),
	}
}

// HandleGitLabWebhook handles GitLab webhook requests
// POST /api/webhook/gitlab/:project_id
func (h *WebhookHandler) HandleGitLabWebhook(c *gin.Context) {
	projectID, err := strconv.ParseUint(c.Param("project_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id"})
		return
	}

	// Get project for webhook secret verification
	project, err := h.projectService.GetByID(uint(projectID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		return
	}

	// Verify webhook token
	token := c.GetHeader("X-Gitlab-Token")
	if project.WebhookSecret != "" && !services.VerifyGitLabSignature(project.WebhookSecret, token) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid webhook token"})
		return
	}

	// Read body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	// Get event type
	eventType := c.GetHeader("X-Gitlab-Event")

	// Process webhook async
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		_ = h.webhookService.HandleGitLabWebhook(ctx, uint(projectID), eventType, body)
	}()

	c.JSON(http.StatusOK, gin.H{"message": "webhook received"})
}

// HandleGitHubWebhook handles GitHub webhook requests
// POST /api/webhook/github/:project_id
func (h *WebhookHandler) HandleGitHubWebhook(c *gin.Context) {
	projectID, err := strconv.ParseUint(c.Param("project_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id"})
		return
	}

	// Get project for webhook secret verification
	project, err := h.projectService.GetByID(uint(projectID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		return
	}

	// Read body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	// Verify webhook signature
	signature := c.GetHeader("X-Hub-Signature-256")
	if project.WebhookSecret != "" && !services.VerifyGitHubSignature(project.WebhookSecret, body, signature) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid webhook signature"})
		return
	}

	// Get event type
	eventType := c.GetHeader("X-GitHub-Event")

	// Process webhook async
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		_ = h.webhookService.HandleGitHubWebhook(ctx, uint(projectID), eventType, body)
	}()

	c.JSON(http.StatusOK, gin.H{"message": "webhook received"})
}
