package handlers

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
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

func (h *WebhookHandler) HandleGitLabWebhookGeneric(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	var payload struct {
		Project struct {
			WebURL     string `json:"web_url"`
			GitHTTPURL string `json:"git_http_url"`
			GitSSHURL  string `json:"git_ssh_url"`
		} `json:"project"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse body"})
		return
	}

	projectURL := payload.Project.WebURL
	if projectURL == "" {
		projectURL = payload.Project.GitHTTPURL
	}
	projectURL = strings.TrimSuffix(projectURL, ".git")

	if projectURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project URL not found in webhook payload"})
		return
	}

	eventType := c.GetHeader("X-Gitlab-Event")

	project, err := h.projectService.GetByURL(projectURL)
	if err != nil {
		log.Printf("[Webhook] Project not found for URL: %s", projectURL)
		services.LogError("Webhook", "ProjectNotFound", "Project not registered: "+projectURL, nil, c.ClientIP(), c.GetHeader("User-Agent"), map[string]interface{}{
			"project_url": projectURL,
			"event_type":  eventType,
		})
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found for URL: " + projectURL})
		return
	}

	token := c.GetHeader("X-Gitlab-Token")
	if project.WebhookSecret != "" && !services.VerifyGitLabSignature(project.WebhookSecret, token) {
		services.LogWarning("Webhook", "InvalidToken", "Invalid webhook token", nil, c.ClientIP(), c.GetHeader("User-Agent"), map[string]interface{}{
			"project_id":  project.ID,
			"project_url": projectURL,
		})
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid webhook token"})
		return
	}

	services.LogInfo("Webhook", "Received", "Webhook received from GitLab", nil, c.ClientIP(), c.GetHeader("User-Agent"), map[string]interface{}{
		"project_id":   project.ID,
		"project_name": project.Name,
		"event_type":   eventType,
	})

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		_ = h.webhookService.HandleGitLabWebhook(ctx, project.ID, eventType, body)
	}()

	c.JSON(http.StatusOK, gin.H{"message": "webhook received", "project_id": project.ID})
}

func (h *WebhookHandler) HandleUnifiedWebhook(c *gin.Context) {
	gitlabEvent := c.GetHeader("X-Gitlab-Event")
	githubEvent := c.GetHeader("X-GitHub-Event")

	if gitlabEvent != "" {
		h.HandleGitLabWebhookGeneric(c)
	} else if githubEvent != "" {
		h.HandleGitHubWebhookGeneric(c)
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown webhook source, missing X-Gitlab-Event or X-GitHub-Event header"})
	}
}

func (h *WebhookHandler) HandleGitHubWebhookGeneric(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	var payload struct {
		Repository struct {
			HTMLURL string `json:"html_url"`
			URL     string `json:"url"`
		} `json:"repository"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse body"})
		return
	}

	projectURL := payload.Repository.HTMLURL
	if projectURL == "" {
		projectURL = payload.Repository.URL
	}
	projectURL = strings.TrimSuffix(projectURL, ".git")

	if projectURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "repository URL not found in webhook payload"})
		return
	}

	eventType := c.GetHeader("X-GitHub-Event")

	project, err := h.projectService.GetByURL(projectURL)
	if err != nil {
		log.Printf("[Webhook] Project not found for URL: %s", projectURL)
		services.LogError("Webhook", "ProjectNotFound", "Project not registered: "+projectURL, nil, c.ClientIP(), c.GetHeader("User-Agent"), map[string]interface{}{
			"project_url": projectURL,
			"event_type":  eventType,
		})
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found for URL: " + projectURL})
		return
	}

	signature := c.GetHeader("X-Hub-Signature-256")
	if project.WebhookSecret != "" && !services.VerifyGitHubSignature(project.WebhookSecret, body, signature) {
		services.LogWarning("Webhook", "InvalidSignature", "Invalid webhook signature", nil, c.ClientIP(), c.GetHeader("User-Agent"), map[string]interface{}{
			"project_id":  project.ID,
			"project_url": projectURL,
		})
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid webhook signature"})
		return
	}

	services.LogInfo("Webhook", "Received", "Webhook received from GitHub", nil, c.ClientIP(), c.GetHeader("User-Agent"), map[string]interface{}{
		"project_id":   project.ID,
		"project_name": project.Name,
		"event_type":   eventType,
	})

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		_ = h.webhookService.HandleGitHubWebhook(ctx, project.ID, eventType, body)
	}()

	c.JSON(http.StatusOK, gin.H{"message": "webhook received", "project_id": project.ID})
}
