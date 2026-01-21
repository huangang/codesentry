package services

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/huangang/codesentry/backend/internal/config"
	"github.com/huangang/codesentry/backend/internal/models"
	"gorm.io/gorm"
)

type WebhookService struct {
	db                  *gorm.DB
	projectService      *ProjectService
	reviewService       *ReviewLogService
	aiService           *AIService
	notificationService *NotificationService
}

func NewWebhookService(db *gorm.DB, aiCfg *config.OpenAIConfig) *WebhookService {
	return &WebhookService{
		db:                  db,
		projectService:      NewProjectService(db),
		reviewService:       NewReviewLogService(db),
		aiService:           NewAIService(db, aiCfg),
		notificationService: NewNotificationService(db),
	}
}

// GitLabPushEvent represents a GitLab push webhook event
type GitLabPushEvent struct {
	ObjectKind  string `json:"object_kind"`
	EventName   string `json:"event_name"`
	Ref         string `json:"ref"`
	CheckoutSHA string `json:"checkout_sha"`
	UserName    string `json:"user_name"`
	UserEmail   string `json:"user_email"`
	ProjectID   int    `json:"project_id"`
	Project     struct {
		Name      string `json:"name"`
		URL       string `json:"url"`
		Namespace string `json:"namespace"`
	} `json:"project"`
	Commits []struct {
		ID        string `json:"id"`
		Message   string `json:"message"`
		Timestamp string `json:"timestamp"`
		Author    struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		} `json:"author"`
		Added    []string `json:"added"`
		Modified []string `json:"modified"`
		Removed  []string `json:"removed"`
	} `json:"commits"`
	TotalCommitsCount int `json:"total_commits_count"`
}

// GitLabMREvent represents a GitLab merge request webhook event
type GitLabMREvent struct {
	ObjectKind string `json:"object_kind"`
	User       struct {
		Name     string `json:"name"`
		Username string `json:"username"`
		Email    string `json:"email"`
	} `json:"user"`
	Project struct {
		ID        int    `json:"id"`
		Name      string `json:"name"`
		URL       string `json:"url"`
		Namespace string `json:"namespace"`
	} `json:"project"`
	ObjectAttributes struct {
		IID          int    `json:"iid"`
		Title        string `json:"title"`
		Description  string `json:"description"`
		SourceBranch string `json:"source_branch"`
		TargetBranch string `json:"target_branch"`
		State        string `json:"state"`
		Action       string `json:"action"`
		URL          string `json:"url"`
	} `json:"object_attributes"`
}

// GitHubPushEvent represents a GitHub push webhook event
type GitHubPushEvent struct {
	Ref    string `json:"ref"`
	After  string `json:"after"`
	Pusher struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"pusher"`
	Repository struct {
		ID       int    `json:"id"`
		Name     string `json:"name"`
		FullName string `json:"full_name"`
		URL      string `json:"url"`
	} `json:"repository"`
	Commits []struct {
		ID        string `json:"id"`
		Message   string `json:"message"`
		Timestamp string `json:"timestamp"`
		Author    struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		} `json:"author"`
		Added    []string `json:"added"`
		Modified []string `json:"modified"`
		Removed  []string `json:"removed"`
	} `json:"commits"`
}

// GitHubPREvent represents a GitHub pull request webhook event
type GitHubPREvent struct {
	Action      string `json:"action"`
	Number      int    `json:"number"`
	PullRequest struct {
		ID    int    `json:"id"`
		Title string `json:"title"`
		Body  string `json:"body"`
		State string `json:"state"`
		Head  struct {
			Ref string `json:"ref"`
			SHA string `json:"sha"`
		} `json:"head"`
		Base struct {
			Ref string `json:"ref"`
		} `json:"base"`
		User struct {
			Login string `json:"login"`
		} `json:"user"`
		HTMLURL string `json:"html_url"`
	} `json:"pull_request"`
	Repository struct {
		ID       int    `json:"id"`
		Name     string `json:"name"`
		FullName string `json:"full_name"`
	} `json:"repository"`
}

// HandleGitLabWebhook processes GitLab webhook events
func (s *WebhookService) HandleGitLabWebhook(ctx context.Context, projectID uint, eventType string, body []byte) error {
	project, err := s.projectService.GetByID(projectID)
	if err != nil {
		return fmt.Errorf("project not found: %w", err)
	}

	if !project.AIEnabled {
		return nil // AI review is disabled
	}

	switch eventType {
	case "Push Hook":
		if !strings.Contains(project.ReviewEvents, "push") {
			return nil
		}
		var event GitLabPushEvent
		if err := json.Unmarshal(body, &event); err != nil {
			return err
		}
		return s.processGitLabPush(ctx, project, &event)

	case "Merge Request Hook":
		if !strings.Contains(project.ReviewEvents, "merge_request") {
			return nil
		}
		var event GitLabMREvent
		if err := json.Unmarshal(body, &event); err != nil {
			return err
		}
		return s.processGitLabMR(ctx, project, &event)
	}

	return nil
}

// HandleGitHubWebhook processes GitHub webhook events
func (s *WebhookService) HandleGitHubWebhook(ctx context.Context, projectID uint, eventType string, body []byte) error {
	project, err := s.projectService.GetByID(projectID)
	if err != nil {
		return fmt.Errorf("project not found: %w", err)
	}

	if !project.AIEnabled {
		return nil
	}

	switch eventType {
	case "push":
		if !strings.Contains(project.ReviewEvents, "push") {
			return nil
		}
		var event GitHubPushEvent
		if err := json.Unmarshal(body, &event); err != nil {
			return err
		}
		return s.processGitHubPush(ctx, project, &event)

	case "pull_request":
		if !strings.Contains(project.ReviewEvents, "merge_request") {
			return nil
		}
		var event GitHubPREvent
		if err := json.Unmarshal(body, &event); err != nil {
			return err
		}
		return s.processGitHubPR(ctx, project, &event)
	}

	return nil
}

func (s *WebhookService) processGitLabPush(ctx context.Context, project *models.Project, event *GitLabPushEvent) error {
	if len(event.Commits) == 0 {
		return nil
	}

	var commits []string
	var additions, deletions int
	for _, c := range event.Commits {
		commits = append(commits, fmt.Sprintf("%s: %s", c.ID[:8], c.Message))
		additions += len(c.Added) + len(c.Modified)
		deletions += len(c.Removed)
	}

	commitSHA := event.CheckoutSHA
	if commitSHA == "" {
		commitSHA = event.Commits[len(event.Commits)-1].ID
	}

	log.Printf("[Webhook] Processing GitLab push: %d commits, checkout_sha=%s, using commit=%s",
		len(event.Commits), event.CheckoutSHA, commitSHA)

	var allDiffs strings.Builder
	for _, c := range event.Commits {
		diff, err := s.getGitLabDiff(project, c.ID)
		if err != nil {
			log.Printf("[Webhook] Failed to get diff for commit %s: %v", c.ID[:8], err)
			continue
		}
		allDiffs.WriteString(fmt.Sprintf("\n### Commit: %s\n%s\n", c.ID[:8], diff))
	}

	diff := allDiffs.String()
	if diff == "" {
		diff = "Failed to get diff for all commits"
		log.Printf("[Webhook] No diffs retrieved for any commits")
	} else {
		log.Printf("[Webhook] Got combined diffs, total length: %d bytes", len(diff))
	}

	branch := strings.TrimPrefix(event.Ref, "refs/heads/")
	reviewLog := &models.ReviewLog{
		ProjectID:     project.ID,
		EventType:     "push",
		CommitHash:    commitSHA,
		Branch:        branch,
		Author:        event.UserName,
		AuthorEmail:   event.UserEmail,
		CommitMessage: strings.Join(commits, "\n"),
		FilesChanged:  additions + deletions,
		Additions:     additions,
		Deletions:     deletions,
		ReviewStatus:  "pending",
	}
	s.reviewService.Create(reviewLog)

	log.Printf("[Webhook] Starting AI review for project %d, commit %s", project.ID, commitSHA)
	result, err := s.aiService.Review(ctx, &ReviewRequest{
		ProjectID: project.ID,
		Diffs:     diff,
		Commits:   strings.Join(commits, "\n"),
	})

	if err != nil {
		log.Printf("[Webhook] AI review failed: %v", err)
		reviewLog.ReviewStatus = "failed"
		reviewLog.ErrorMessage = err.Error()
	} else {
		log.Printf("[Webhook] AI review completed, score: %.1f, result length: %d", result.Score, len(result.Content))
		reviewLog.ReviewStatus = "completed"
		reviewLog.ReviewResult = result.Content
		reviewLog.Score = &result.Score

		s.notificationService.SendReviewNotification(project, &ReviewNotification{
			ProjectName:   project.Name,
			Branch:        branch,
			Author:        event.UserName,
			CommitMessage: strings.Join(commits, "\n"),
			Score:         result.Score,
			ReviewResult:  result.Content,
			EventType:     "push",
		})
	}

	return s.reviewService.Update(reviewLog)
}

func (s *WebhookService) processGitLabMR(ctx context.Context, project *models.Project, event *GitLabMREvent) error {
	if event.ObjectAttributes.Action != "open" && event.ObjectAttributes.Action != "update" {
		return nil
	}

	mrNumber := event.ObjectAttributes.IID

	// Create review log
	reviewLog := &models.ReviewLog{
		ProjectID:     project.ID,
		EventType:     "merge_request",
		Branch:        event.ObjectAttributes.SourceBranch,
		Author:        event.User.Name,
		AuthorEmail:   event.User.Email,
		CommitMessage: event.ObjectAttributes.Title,
		MRNumber:      &mrNumber,
		MRURL:         event.ObjectAttributes.URL,
		ReviewStatus:  "pending",
	}
	s.reviewService.Create(reviewLog)

	// Get MR diff
	diff, err := s.getGitLabMRDiff(project, mrNumber)
	if err != nil {
		diff = "Failed to get diff: " + err.Error()
	}

	// Perform AI review
	result, err := s.aiService.Review(ctx, &ReviewRequest{
		ProjectID: project.ID,
		Diffs:     diff,
		Commits:   event.ObjectAttributes.Title + "\n" + event.ObjectAttributes.Description,
	})

	if err != nil {
		reviewLog.ReviewStatus = "failed"
		reviewLog.ErrorMessage = err.Error()
	} else {
		reviewLog.ReviewStatus = "completed"
		reviewLog.ReviewResult = result.Content
		reviewLog.Score = &result.Score

		s.notificationService.SendReviewNotification(project, &ReviewNotification{
			ProjectName:   project.Name,
			Branch:        event.ObjectAttributes.SourceBranch,
			Author:        event.User.Name,
			CommitMessage: event.ObjectAttributes.Title,
			Score:         result.Score,
			ReviewResult:  result.Content,
			EventType:     "merge_request",
			MRURL:         event.ObjectAttributes.URL,
		})
	}

	return s.reviewService.Update(reviewLog)
}

func (s *WebhookService) processGitHubPush(ctx context.Context, project *models.Project, event *GitHubPushEvent) error {
	if len(event.Commits) == 0 {
		return nil
	}

	var commits []string
	var additions, deletions int
	for _, c := range event.Commits {
		commits = append(commits, fmt.Sprintf("%s: %s", c.ID[:8], c.Message))
		additions += len(c.Added) + len(c.Modified)
		deletions += len(c.Removed)
	}

	branch := strings.TrimPrefix(event.Ref, "refs/heads/")
	reviewLog := &models.ReviewLog{
		ProjectID:     project.ID,
		EventType:     "push",
		CommitHash:    event.After,
		Branch:        branch,
		Author:        event.Pusher.Name,
		AuthorEmail:   event.Pusher.Email,
		CommitMessage: strings.Join(commits, "\n"),
		FilesChanged:  additions + deletions,
		Additions:     additions,
		Deletions:     deletions,
		ReviewStatus:  "pending",
	}
	s.reviewService.Create(reviewLog)

	// Get diff from GitHub API
	diff, err := s.getGitHubDiff(project, event.After)
	if err != nil {
		diff = "Failed to get diff: " + err.Error()
	}

	result, err := s.aiService.Review(ctx, &ReviewRequest{
		ProjectID: project.ID,
		Diffs:     diff,
		Commits:   strings.Join(commits, "\n"),
	})

	if err != nil {
		reviewLog.ReviewStatus = "failed"
		reviewLog.ErrorMessage = err.Error()
	} else {
		reviewLog.ReviewStatus = "completed"
		reviewLog.ReviewResult = result.Content
		reviewLog.Score = &result.Score

		s.notificationService.SendReviewNotification(project, &ReviewNotification{
			ProjectName:   project.Name,
			Branch:        branch,
			Author:        event.Pusher.Name,
			CommitMessage: strings.Join(commits, "\n"),
			Score:         result.Score,
			ReviewResult:  result.Content,
			EventType:     "push",
		})
	}

	return s.reviewService.Update(reviewLog)
}

func (s *WebhookService) processGitHubPR(ctx context.Context, project *models.Project, event *GitHubPREvent) error {
	if event.Action != "opened" && event.Action != "synchronize" {
		return nil
	}

	mrNumber := event.Number

	reviewLog := &models.ReviewLog{
		ProjectID:     project.ID,
		EventType:     "merge_request",
		CommitHash:    event.PullRequest.Head.SHA,
		Branch:        event.PullRequest.Head.Ref,
		Author:        event.PullRequest.User.Login,
		CommitMessage: event.PullRequest.Title,
		MRNumber:      &mrNumber,
		MRURL:         event.PullRequest.HTMLURL,
		ReviewStatus:  "pending",
	}
	s.reviewService.Create(reviewLog)

	diff, err := s.getGitHubPRDiff(project, mrNumber)
	if err != nil {
		diff = "Failed to get diff: " + err.Error()
	}

	result, err := s.aiService.Review(ctx, &ReviewRequest{
		ProjectID: project.ID,
		Diffs:     diff,
		Commits:   event.PullRequest.Title + "\n" + event.PullRequest.Body,
	})

	if err != nil {
		reviewLog.ReviewStatus = "failed"
		reviewLog.ErrorMessage = err.Error()
	} else {
		reviewLog.ReviewStatus = "completed"
		reviewLog.ReviewResult = result.Content
		reviewLog.Score = &result.Score

		s.notificationService.SendReviewNotification(project, &ReviewNotification{
			ProjectName:   project.Name,
			Branch:        event.PullRequest.Head.Ref,
			Author:        event.PullRequest.User.Login,
			CommitMessage: event.PullRequest.Title,
			Score:         result.Score,
			ReviewResult:  result.Content,
			EventType:     "merge_request",
			MRURL:         event.PullRequest.HTMLURL,
		})
	}

	return s.reviewService.Update(reviewLog)
}

// Helper functions for getting diffs from Git platforms

func (s *WebhookService) getGitLabDiff(project *models.Project, commitSHA string) (string, error) {
	url := strings.TrimSuffix(project.URL, ".git")
	parts := strings.Split(url, "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid project URL")
	}
	projectPath := strings.Join(parts[len(parts)-2:], "/")

	baseURL := strings.TrimSuffix(url, "/"+projectPath)
	apiURL := fmt.Sprintf("%s/api/v4/projects/%s/repository/commits/%s/diff",
		baseURL, strings.ReplaceAll(projectPath, "/", "%2F"), commitSHA)

	log.Printf("[Webhook] GitLab project URL: %s, projectPath: %s, baseURL: %s", project.URL, projectPath, baseURL)
	log.Printf("[Webhook] GitLab Access Token configured: %v", project.AccessToken != "")

	return s.fetchDiff(apiURL, project.AccessToken, "PRIVATE-TOKEN")
}

func (s *WebhookService) getGitLabMRDiff(project *models.Project, mrIID int) (string, error) {
	url := strings.TrimSuffix(project.URL, ".git")
	parts := strings.Split(url, "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid project URL")
	}
	projectPath := strings.Join(parts[len(parts)-2:], "/")
	baseURL := strings.TrimSuffix(url, "/"+projectPath)

	apiURL := fmt.Sprintf("%s/api/v4/projects/%s/merge_requests/%d/changes",
		baseURL, strings.ReplaceAll(projectPath, "/", "%2F"), mrIID)

	return s.fetchDiff(apiURL, project.AccessToken, "PRIVATE-TOKEN")
}

func (s *WebhookService) getGitHubDiff(project *models.Project, commitSHA string) (string, error) {
	url := strings.TrimSuffix(project.URL, ".git")
	parts := strings.Split(url, "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid project URL")
	}
	owner := parts[len(parts)-2]
	repo := parts[len(parts)-1]

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits/%s", owner, repo, commitSHA)

	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Set("Accept", "application/vnd.github.v3.diff")
	if project.AccessToken != "" {
		req.Header.Set("Authorization", "token "+project.AccessToken)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func (s *WebhookService) getGitHubPRDiff(project *models.Project, prNumber int) (string, error) {
	url := strings.TrimSuffix(project.URL, ".git")
	parts := strings.Split(url, "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid project URL")
	}
	owner := parts[len(parts)-2]
	repo := parts[len(parts)-1]

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls/%d", owner, repo, prNumber)

	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Set("Accept", "application/vnd.github.v3.diff")
	if project.AccessToken != "" {
		req.Header.Set("Authorization", "token "+project.AccessToken)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func (s *WebhookService) fetchDiff(apiURL, token, tokenHeader string) (string, error) {
	log.Printf("[Webhook] Fetching diff from: %s", apiURL)

	req, _ := http.NewRequest("GET", apiURL, nil)
	if token != "" {
		req.Header.Set(tokenHeader, token)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	log.Printf("[Webhook] Diff API response status: %d, body length: %d", resp.StatusCode, len(body))

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var diffs []struct {
		Diff    string `json:"diff"`
		OldPath string `json:"old_path"`
		NewPath string `json:"new_path"`
	}
	if err := json.Unmarshal(body, &diffs); err != nil {
		return string(body), nil
	}

	var result strings.Builder
	for _, d := range diffs {
		result.WriteString(fmt.Sprintf("--- %s\n+++ %s\n%s\n", d.OldPath, d.NewPath, d.Diff))
	}

	return result.String(), nil
}

// VerifyGitLabSignature verifies GitLab webhook signature
func VerifyGitLabSignature(secret, token string) bool {
	return secret == token
}

// VerifyGitHubSignature verifies GitHub webhook signature
func VerifyGitHubSignature(secret string, body []byte, signature string) bool {
	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expectedMAC := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature[7:]), []byte(expectedMAC))
}
