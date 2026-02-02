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
	"github.com/huangang/codesentry/backend/internal/models"
	"github.com/huangang/codesentry/backend/internal/services"
	"gorm.io/gorm"
)

type WebhookHandler struct {
	webhookService       *services.WebhookService
	projectService       *services.ProjectService
	gitCredentialService *services.GitCredentialService
}

func NewWebhookHandler(db *gorm.DB, aiCfg *config.OpenAIConfig) *WebhookHandler {
	return &WebhookHandler{
		webhookService:       services.NewWebhookService(db, aiCfg),
		projectService:       services.NewProjectService(db),
		gitCredentialService: services.NewGitCredentialService(db),
	}
}

type webhookContext struct {
	platform    string
	projectURL  string
	projectName string
	eventType   string
	body        []byte
	clientIP    string
	userAgent   string
}

type signatureVerifier func(secret string, body []byte, signature string) bool

func (h *WebhookHandler) resolveProject(ctx *webhookContext, signature string, verifyFn signatureVerifier) (*models.Project, error, int) {
	project, err := h.projectService.GetByURL(ctx.projectURL)
	if err != nil {
		log.Printf("[Webhook] Project not found for URL: %s, checking for matching credential", ctx.projectURL)

		credential, credErr := h.gitCredentialService.FindMatchingCredential(ctx.projectURL, ctx.platform)
		if credErr != nil || credential == nil {
			services.LogError("Webhook", "ProjectNotFound", "Project not registered and no matching credential: "+ctx.projectURL, nil, ctx.clientIP, ctx.userAgent, map[string]interface{}{
				"project_url": ctx.projectURL,
				"event_type":  ctx.eventType,
			})
			return nil, err, http.StatusNotFound
		}

		if credential.WebhookSecret != "" && !verifyFn(credential.WebhookSecret, ctx.body, signature) {
			services.LogWarning("Webhook", "InvalidSignature", "Invalid webhook signature for credential", nil, ctx.clientIP, ctx.userAgent, map[string]interface{}{
				"credential_id": credential.ID,
				"project_url":   ctx.projectURL,
			})
			return nil, err, http.StatusUnauthorized
		}

		newProject := &services.CreateProjectParams{
			Name:           ctx.projectName,
			URL:            ctx.projectURL,
			Platform:       ctx.platform,
			AccessToken:    credential.AccessToken,
			WebhookSecret:  credential.WebhookSecret,
			AIEnabled:      credential.DefaultEnabled,
			FileExtensions: credential.FileExtensions,
			ReviewEvents:   credential.ReviewEvents,
			IgnorePatterns: credential.IgnorePatterns,
		}

		project, err = h.projectService.CreateFromCredential(newProject)
		if err != nil {
			services.LogError("Webhook", "AutoCreateFailed", "Failed to auto-create project: "+err.Error(), nil, ctx.clientIP, ctx.userAgent, map[string]interface{}{
				"project_url":   ctx.projectURL,
				"credential_id": credential.ID,
			})
			return nil, err, http.StatusInternalServerError
		}

		services.LogInfo("Webhook", "AutoCreated", "Project auto-created from credential", nil, ctx.clientIP, ctx.userAgent, map[string]interface{}{
			"project_id":    project.ID,
			"project_name":  project.Name,
			"credential_id": credential.ID,
		})
		return project, nil, http.StatusOK
	}

	if project.WebhookSecret != "" && !verifyFn(project.WebhookSecret, ctx.body, signature) {
		services.LogWarning("Webhook", "InvalidSignature", "Invalid webhook signature", nil, ctx.clientIP, ctx.userAgent, map[string]interface{}{
			"project_id":  project.ID,
			"project_url": ctx.projectURL,
		})
		return nil, err, http.StatusUnauthorized
	}

	h.tryFillFromCredential(project, ctx)
	return project, nil, http.StatusOK
}

func (h *WebhookHandler) tryFillFromCredential(project *models.Project, ctx *webhookContext) {
	if project.AccessToken != "" {
		return
	}
	credential, err := h.gitCredentialService.FindMatchingCredential(ctx.projectURL, ctx.platform)
	if err != nil || credential == nil || credential.AccessToken == "" {
		return
	}
	h.projectService.FillFromCredential(project, credential)
	services.LogInfo("Webhook", "CredentialFilled", "Project credentials filled from git credential", nil, ctx.clientIP, ctx.userAgent, map[string]interface{}{
		"project_id":    project.ID,
		"credential_id": credential.ID,
	})
}

func gitlabVerifier(secret string, _ []byte, token string) bool {
	return services.VerifyGitLabSignature(secret, token)
}

func githubVerifier(secret string, body []byte, signature string) bool {
	return services.VerifyGitHubSignature(secret, body, signature)
}

func bitbucketVerifier(secret string, body []byte, signature string) bool {
	return services.VerifyBitbucketSignature(secret, body, signature)
}

func (h *WebhookHandler) HandleGitLabWebhook(c *gin.Context) {
	projectID, err := strconv.ParseUint(c.Param("project_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id"})
		return
	}

	project, err := h.projectService.GetByID(uint(projectID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		return
	}

	token := c.GetHeader("X-Gitlab-Token")
	if project.WebhookSecret != "" && !services.VerifyGitLabSignature(project.WebhookSecret, token) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid webhook token"})
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	eventType := c.GetHeader("X-Gitlab-Event")

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		_ = h.webhookService.HandleGitLabWebhook(ctx, uint(projectID), eventType, body)
	}()

	c.JSON(http.StatusOK, gin.H{"message": "webhook received"})
}

func (h *WebhookHandler) HandleGitHubWebhook(c *gin.Context) {
	projectID, err := strconv.ParseUint(c.Param("project_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id"})
		return
	}

	project, err := h.projectService.GetByID(uint(projectID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	signature := c.GetHeader("X-Hub-Signature-256")
	if project.WebhookSecret != "" && !services.VerifyGitHubSignature(project.WebhookSecret, body, signature) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid webhook signature"})
		return
	}

	eventType := c.GetHeader("X-GitHub-Event")

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
			Name       string `json:"name"`
			WebURL     string `json:"web_url"`
			GitHTTPURL string `json:"git_http_url"`
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

	projectName := payload.Project.Name
	if projectName == "" {
		parts := strings.Split(projectURL, "/")
		projectName = parts[len(parts)-1]
	}

	ctx := &webhookContext{
		platform:    "gitlab",
		projectURL:  projectURL,
		projectName: projectName,
		eventType:   c.GetHeader("X-Gitlab-Event"),
		body:        body,
		clientIP:    c.ClientIP(),
		userAgent:   c.GetHeader("User-Agent"),
	}

	token := c.GetHeader("X-Gitlab-Token")
	project, resolveErr, statusCode := h.resolveProject(ctx, token, gitlabVerifier)
	if resolveErr != nil {
		switch statusCode {
		case http.StatusUnauthorized:
			c.JSON(statusCode, gin.H{"error": "invalid webhook token"})
		case http.StatusNotFound:
			c.JSON(statusCode, gin.H{"error": "project not found for URL: " + projectURL})
		default:
			c.JSON(statusCode, gin.H{"error": "failed to auto-create project"})
		}
		return
	}

	services.LogInfo("Webhook", "Received", "Webhook received from GitLab", nil, ctx.clientIP, ctx.userAgent, map[string]interface{}{
		"project_id":   project.ID,
		"project_name": project.Name,
		"event_type":   ctx.eventType,
	})

	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		_ = h.webhookService.HandleGitLabWebhook(bgCtx, project.ID, ctx.eventType, body)
	}()

	c.JSON(http.StatusOK, gin.H{"message": "webhook received", "project_id": project.ID})
}

func (h *WebhookHandler) HandleGitHubWebhookGeneric(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	var payload struct {
		Repository struct {
			Name    string `json:"name"`
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

	projectName := payload.Repository.Name
	if projectName == "" {
		parts := strings.Split(projectURL, "/")
		projectName = parts[len(parts)-1]
	}

	ctx := &webhookContext{
		platform:    "github",
		projectURL:  projectURL,
		projectName: projectName,
		eventType:   c.GetHeader("X-GitHub-Event"),
		body:        body,
		clientIP:    c.ClientIP(),
		userAgent:   c.GetHeader("User-Agent"),
	}

	signature := c.GetHeader("X-Hub-Signature-256")
	project, resolveErr, statusCode := h.resolveProject(ctx, signature, githubVerifier)
	if resolveErr != nil {
		switch statusCode {
		case http.StatusUnauthorized:
			c.JSON(statusCode, gin.H{"error": "invalid webhook signature"})
		case http.StatusNotFound:
			c.JSON(statusCode, gin.H{"error": "project not found for URL: " + projectURL})
		default:
			c.JSON(statusCode, gin.H{"error": "failed to auto-create project"})
		}
		return
	}

	services.LogInfo("Webhook", "Received", "Webhook received from GitHub", nil, ctx.clientIP, ctx.userAgent, map[string]interface{}{
		"project_id":   project.ID,
		"project_name": project.Name,
		"event_type":   ctx.eventType,
	})

	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		_ = h.webhookService.HandleGitHubWebhook(bgCtx, project.ID, ctx.eventType, body)
	}()

	c.JSON(http.StatusOK, gin.H{"message": "webhook received", "project_id": project.ID})
}

func (h *WebhookHandler) HandleBitbucketWebhook(c *gin.Context) {
	projectID, err := strconv.ParseUint(c.Param("project_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id"})
		return
	}

	project, err := h.projectService.GetByID(uint(projectID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	signature := c.GetHeader("X-Hub-Signature")
	if project.WebhookSecret != "" && !services.VerifyBitbucketSignature(project.WebhookSecret, body, signature) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid webhook signature"})
		return
	}

	eventType := c.GetHeader("X-Event-Key")

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		_ = h.webhookService.HandleBitbucketWebhook(ctx, uint(projectID), eventType, body)
	}()

	c.JSON(http.StatusOK, gin.H{"message": "webhook received"})
}

func (h *WebhookHandler) HandleBitbucketWebhookGeneric(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	var payload struct {
		Repository struct {
			Name     string `json:"name"`
			FullName string `json:"full_name"`
			Links    struct {
				HTML struct {
					Href string `json:"href"`
				} `json:"html"`
			} `json:"links"`
		} `json:"repository"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse body"})
		return
	}

	projectURL := payload.Repository.Links.HTML.Href
	if projectURL == "" && payload.Repository.FullName != "" {
		projectURL = "https://bitbucket.org/" + payload.Repository.FullName
	}
	projectURL = strings.TrimSuffix(projectURL, ".git")

	if projectURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "repository URL not found in webhook payload"})
		return
	}

	projectName := payload.Repository.Name
	if projectName == "" {
		parts := strings.Split(projectURL, "/")
		projectName = parts[len(parts)-1]
	}

	ctx := &webhookContext{
		platform:    "bitbucket",
		projectURL:  projectURL,
		projectName: projectName,
		eventType:   c.GetHeader("X-Event-Key"),
		body:        body,
		clientIP:    c.ClientIP(),
		userAgent:   c.GetHeader("User-Agent"),
	}

	signature := c.GetHeader("X-Hub-Signature")
	project, resolveErr, statusCode := h.resolveProject(ctx, signature, bitbucketVerifier)
	if resolveErr != nil {
		switch statusCode {
		case http.StatusUnauthorized:
			c.JSON(statusCode, gin.H{"error": "invalid webhook signature"})
		case http.StatusNotFound:
			c.JSON(statusCode, gin.H{"error": "project not found for URL: " + projectURL})
		default:
			c.JSON(statusCode, gin.H{"error": "failed to auto-create project"})
		}
		return
	}

	services.LogInfo("Webhook", "Received", "Webhook received from Bitbucket", nil, ctx.clientIP, ctx.userAgent, map[string]interface{}{
		"project_id":   project.ID,
		"project_name": project.Name,
		"event_type":   ctx.eventType,
	})

	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		_ = h.webhookService.HandleBitbucketWebhook(bgCtx, project.ID, ctx.eventType, body)
	}()

	c.JSON(http.StatusOK, gin.H{"message": "webhook received", "project_id": project.ID})
}

func (h *WebhookHandler) HandleUnifiedWebhook(c *gin.Context) {
	gitlabEvent := c.GetHeader("X-Gitlab-Event")
	githubEvent := c.GetHeader("X-GitHub-Event")
	bitbucketEvent := c.GetHeader("X-Event-Key")

	if gitlabEvent != "" {
		h.HandleGitLabWebhookGeneric(c)
	} else if githubEvent != "" {
		h.HandleGitHubWebhookGeneric(c)
	} else if bitbucketEvent != "" {
		h.HandleBitbucketWebhookGeneric(c)
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown webhook source, missing X-Gitlab-Event, X-GitHub-Event, or X-Event-Key header"})
	}
}

func (h *WebhookHandler) HandleSyncReview(c *gin.Context) {
	var req struct {
		ProjectURL string `json:"project_url" binding:"required"`
		CommitSHA  string `json:"commit_sha" binding:"required"`
		Ref        string `json:"ref"`
		Author     string `json:"author"`
		Message    string `json:"message"`
		Diffs      string `json:"diffs" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	projectURL := strings.TrimSuffix(req.ProjectURL, ".git")
	project, err := h.projectService.GetByURL(projectURL)
	if err != nil {
		services.LogError("SyncReview", "ProjectNotFound", "Project not registered: "+projectURL, nil, c.ClientIP(), c.GetHeader("User-Agent"), map[string]interface{}{
			"project_url": projectURL,
			"commit_sha":  req.CommitSHA,
		})
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found for URL: " + projectURL})
		return
	}

	apiKey := c.GetHeader("X-API-Key")
	if project.WebhookSecret != "" && apiKey != project.WebhookSecret {
		services.LogWarning("SyncReview", "InvalidAPIKey", "Invalid API key", nil, c.ClientIP(), c.GetHeader("User-Agent"), map[string]interface{}{
			"project_id":  project.ID,
			"project_url": projectURL,
		})
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid API key"})
		return
	}

	services.LogInfo("SyncReview", "Received", "Sync review request received", nil, c.ClientIP(), c.GetHeader("User-Agent"), map[string]interface{}{
		"project_id":   project.ID,
		"project_name": project.Name,
		"commit_sha":   req.CommitSHA,
		"ref":          req.Ref,
	})

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Minute)
	defer cancel()

	result, err := h.webhookService.SyncReview(ctx, project, &services.SyncReviewRequest{
		ProjectURL: req.ProjectURL,
		CommitSHA:  req.CommitSHA,
		Ref:        req.Ref,
		Author:     req.Author,
		Message:    req.Message,
		Diffs:      req.Diffs,
	})
	if err != nil {
		services.LogError("SyncReview", "ReviewFailed", err.Error(), nil, c.ClientIP(), c.GetHeader("User-Agent"), map[string]interface{}{
			"project_id": project.ID,
			"commit_sha": req.CommitSHA,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "review failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *WebhookHandler) GetReviewScore(c *gin.Context) {
	commitSHA := c.Query("commit_sha")
	if commitSHA == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "commit_sha is required"})
		return
	}

	result, err := h.webhookService.GetReviewScore(commitSHA)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}
