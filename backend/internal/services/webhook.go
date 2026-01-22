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
	"path/filepath"
	"strings"
	"time"

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
	httpClient          *http.Client
}

func NewWebhookService(db *gorm.DB, aiCfg *config.OpenAIConfig) *WebhookService {
	return &WebhookService{
		db:                  db,
		projectService:      NewProjectService(db),
		reviewService:       NewReviewLogService(db),
		aiService:           NewAIService(db, aiCfg),
		notificationService: NewNotificationService(db),
		httpClient:          &http.Client{Timeout: 30 * time.Second},
	}
}

type repoInfo struct {
	owner       string
	repo        string
	projectPath string
	baseURL     string
}

func parseRepoInfo(projectURL string) (*repoInfo, error) {
	url := strings.TrimSuffix(projectURL, ".git")
	parts := strings.Split(url, "/")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid project URL: %s", projectURL)
	}
	return &repoInfo{
		owner:       parts[len(parts)-2],
		repo:        parts[len(parts)-1],
		projectPath: parts[len(parts)-2] + "/" + parts[len(parts)-1],
		baseURL:     strings.TrimSuffix(url, "/"+parts[len(parts)-2]+"/"+parts[len(parts)-1]),
	}, nil
}

// GitLabPushEvent represents a GitLab push webhook event
type GitLabPushEvent struct {
	ObjectKind  string `json:"object_kind"`
	EventName   string `json:"event_name"`
	Ref         string `json:"ref"`
	CheckoutSHA string `json:"checkout_sha"`
	UserName    string `json:"user_name"`
	UserEmail   string `json:"user_email"`
	UserAvatar  string `json:"user_avatar"`
	ProjectID   int    `json:"project_id"`
	Project     struct {
		Name      string `json:"name"`
		URL       string `json:"url"`
		WebURL    string `json:"web_url"`
		Namespace string `json:"namespace"`
	} `json:"project"`
	Commits []struct {
		ID        string `json:"id"`
		Message   string `json:"message"`
		Timestamp string `json:"timestamp"`
		URL       string `json:"url"`
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
		Name      string `json:"name"`
		Username  string `json:"username"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	} `json:"user"`
	Project struct {
		ID        int    `json:"id"`
		Name      string `json:"name"`
		URL       string `json:"url"`
		WebURL    string `json:"web_url"`
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
	Sender struct {
		Login     string `json:"login"`
		AvatarURL string `json:"avatar_url"`
		HTMLURL   string `json:"html_url"`
	} `json:"sender"`
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
		URL       string `json:"url"`
		Author    struct {
			Name     string `json:"name"`
			Email    string `json:"email"`
			Username string `json:"username"`
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
			Login     string `json:"login"`
			AvatarURL string `json:"avatar_url"`
			HTMLURL   string `json:"html_url"`
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

	LogInfo("Webhook", "GitLabPush", fmt.Sprintf("Processing push from %s: %d commits", event.UserName, len(event.Commits)), nil, "", "", map[string]interface{}{
		"project_id": project.ID,
		"branch":     strings.TrimPrefix(event.Ref, "refs/heads/"),
		"commit":     commitSHA,
	})

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

	var commitURL string
	if len(event.Commits) > 0 && event.Commits[len(event.Commits)-1].URL != "" {
		commitURL = event.Commits[len(event.Commits)-1].URL
	}

	reviewLog := &models.ReviewLog{
		ProjectID:     project.ID,
		EventType:     "push",
		CommitHash:    commitSHA,
		CommitURL:     commitURL,
		Branch:        branch,
		Author:        event.UserName,
		AuthorEmail:   event.UserEmail,
		AuthorAvatar:  event.UserAvatar,
		CommitMessage: strings.Join(commits, "\n"),
		FilesChanged:  additions + deletions,
		Additions:     additions,
		Deletions:     deletions,
		ReviewStatus:  "pending",
	}
	s.reviewService.Create(reviewLog)

	log.Printf("[Webhook] Starting AI review for project %d, commit %s", project.ID, commitSHA)

	filteredDiff := s.filterDiff(diff, project.FileExtensions, project.IgnorePatterns)
	if filteredDiff != diff {
		log.Printf("[Webhook] Filtered diff by extensions (%s) and ignore patterns (%s): %d -> %d bytes",
			project.FileExtensions, project.IgnorePatterns, len(diff), len(filteredDiff))
	}

	result, err := s.aiService.Review(ctx, &ReviewRequest{
		ProjectID: project.ID,
		Diffs:     filteredDiff,
		Commits:   strings.Join(commits, "\n"),
	})

	if err != nil {
		log.Printf("[Webhook] AI review failed: %v", err)
		LogError("AIReview", "ReviewFailed", err.Error(), nil, "", "", map[string]interface{}{
			"project_id": project.ID,
			"commit":     commitSHA,
		})
		reviewLog.ReviewStatus = "failed"
		reviewLog.ErrorMessage = err.Error()
	} else {
		log.Printf("[Webhook] AI review completed, score: %.1f, result length: %d", result.Score, len(result.Content))
		LogInfo("AIReview", "ReviewCompleted", fmt.Sprintf("Review completed with score %.0f", result.Score), nil, "", "", map[string]interface{}{
			"project_id": project.ID,
			"commit":     commitSHA,
			"score":      result.Score,
		})
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

	reviewLog := &models.ReviewLog{
		ProjectID:     project.ID,
		EventType:     "merge_request",
		Branch:        event.ObjectAttributes.SourceBranch,
		Author:        event.User.Name,
		AuthorEmail:   event.User.Email,
		AuthorAvatar:  event.User.AvatarURL,
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

	filteredDiff := s.filterDiff(diff, project.FileExtensions, project.IgnorePatterns)
	result, err := s.aiService.Review(ctx, &ReviewRequest{
		ProjectID: project.ID,
		Diffs:     filteredDiff,
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

		if project.CommentEnabled {
			comment := s.formatReviewComment(result.Score, result.Content)
			if err := s.postGitLabMRComment(project, mrNumber, comment); err != nil {
				log.Printf("[Webhook] Failed to post GitLab MR comment: %v", err)
			}
		}
	}

	return s.reviewService.Update(reviewLog)
}

func (s *WebhookService) processGitHubPush(ctx context.Context, project *models.Project, event *GitHubPushEvent) error {
	if len(event.Commits) == 0 {
		return nil
	}

	var commits []string
	var additions, deletions int
	var commitURL string
	for _, c := range event.Commits {
		commits = append(commits, fmt.Sprintf("%s: %s", c.ID[:8], c.Message))
		additions += len(c.Added) + len(c.Modified)
		deletions += len(c.Removed)
		if commitURL == "" && c.URL != "" {
			commitURL = c.URL
		}
	}

	branch := strings.TrimPrefix(event.Ref, "refs/heads/")
	reviewLog := &models.ReviewLog{
		ProjectID:     project.ID,
		EventType:     "push",
		CommitHash:    event.After,
		CommitURL:     commitURL,
		Branch:        branch,
		Author:        event.Sender.Login,
		AuthorEmail:   event.Pusher.Email,
		AuthorAvatar:  event.Sender.AvatarURL,
		AuthorURL:     event.Sender.HTMLURL,
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

	filteredDiff := s.filterDiff(diff, project.FileExtensions, project.IgnorePatterns)
	result, err := s.aiService.Review(ctx, &ReviewRequest{
		ProjectID: project.ID,
		Diffs:     filteredDiff,
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
		AuthorAvatar:  event.PullRequest.User.AvatarURL,
		AuthorURL:     event.PullRequest.User.HTMLURL,
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

	filteredDiff := s.filterDiff(diff, project.FileExtensions, project.IgnorePatterns)
	result, err := s.aiService.Review(ctx, &ReviewRequest{
		ProjectID: project.ID,
		Diffs:     filteredDiff,
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

		if project.CommentEnabled {
			comment := s.formatReviewComment(result.Score, result.Content)
			if err := s.postGitHubPRComment(project, mrNumber, comment); err != nil {
				log.Printf("[Webhook] Failed to post GitHub PR comment: %v", err)
			}
		}
	}

	return s.reviewService.Update(reviewLog)
}

// Helper functions for getting diffs from Git platforms

func (s *WebhookService) getGitLabDiff(project *models.Project, commitSHA string) (string, error) {
	info, err := parseRepoInfo(project.URL)
	if err != nil {
		return "", err
	}

	apiURL := fmt.Sprintf("%s/api/v4/projects/%s/repository/commits/%s/diff",
		info.baseURL, strings.ReplaceAll(info.projectPath, "/", "%2F"), commitSHA)

	log.Printf("[Webhook] GitLab project URL: %s, projectPath: %s, baseURL: %s", project.URL, info.projectPath, info.baseURL)
	log.Printf("[Webhook] GitLab Access Token configured: %v", project.AccessToken != "")

	return s.fetchDiff(apiURL, project.AccessToken, "PRIVATE-TOKEN")
}

func (s *WebhookService) getGitLabMRDiff(project *models.Project, mrIID int) (string, error) {
	info, err := parseRepoInfo(project.URL)
	if err != nil {
		return "", err
	}

	apiURL := fmt.Sprintf("%s/api/v4/projects/%s/merge_requests/%d/changes",
		info.baseURL, strings.ReplaceAll(info.projectPath, "/", "%2F"), mrIID)

	return s.fetchDiff(apiURL, project.AccessToken, "PRIVATE-TOKEN")
}

func (s *WebhookService) getGitHubDiff(project *models.Project, commitSHA string) (string, error) {
	info, err := parseRepoInfo(project.URL)
	if err != nil {
		return "", err
	}

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits/%s", info.owner, info.repo, commitSHA)
	return s.fetchGitHubDiff(apiURL, project.AccessToken)
}

func (s *WebhookService) getGitHubPRDiff(project *models.Project, prNumber int) (string, error) {
	info, err := parseRepoInfo(project.URL)
	if err != nil {
		return "", err
	}

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls/%d", info.owner, info.repo, prNumber)
	return s.fetchGitHubDiff(apiURL, project.AccessToken)
}

func (s *WebhookService) fetchGitHubDiff(apiURL, accessToken string) (string, error) {
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github.v3.diff")
	if accessToken != "" {
		req.Header.Set("Authorization", "token "+accessToken)
	}

	resp, err := s.httpClient.Do(req)
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

	resp, err := s.httpClient.Do(req)
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

func (s *WebhookService) filterDiff(diff string, extensions string, ignorePatterns string) string {
	extMap := make(map[string]bool)
	if extensions != "" {
		for _, ext := range strings.Split(extensions, ",") {
			ext = strings.TrimSpace(ext)
			if ext != "" {
				if !strings.HasPrefix(ext, ".") {
					ext = "." + ext
				}
				extMap[strings.ToLower(ext)] = true
			}
		}
	}

	var ignoreList []string
	if ignorePatterns != "" {
		for _, pattern := range strings.Split(ignorePatterns, ",") {
			pattern = strings.TrimSpace(pattern)
			if pattern != "" {
				ignoreList = append(ignoreList, pattern)
			}
		}
	}

	if len(extMap) == 0 && len(ignoreList) == 0 {
		return diff
	}

	lines := strings.Split(diff, "\n")
	var result strings.Builder
	var include bool

	for _, line := range lines {
		if strings.HasPrefix(line, "--- ") || strings.HasPrefix(line, "+++ ") {
			filePath := strings.TrimPrefix(strings.TrimPrefix(line, "--- "), "+++ ")
			filePath = strings.TrimPrefix(filePath, "a/")
			filePath = strings.TrimPrefix(filePath, "b/")

			if strings.HasPrefix(line, "--- ") {
				include = s.shouldIncludeFile(filePath, extMap, ignoreList)
			}
			if include {
				result.WriteString(line + "\n")
			}
		} else if include {
			result.WriteString(line + "\n")
		}
	}

	filtered := result.String()
	if filtered == "" {
		return diff
	}
	return filtered
}

func (s *WebhookService) shouldIncludeFile(filePath string, extMap map[string]bool, ignoreList []string) bool {
	for _, pattern := range ignoreList {
		if s.matchIgnorePattern(filePath, pattern) {
			return false
		}
	}

	if len(extMap) == 0 {
		return true
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	return extMap[ext]
}

func (s *WebhookService) matchIgnorePattern(filePath, pattern string) bool {
	pattern = strings.ToLower(pattern)
	filePath = strings.ToLower(filePath)

	if strings.HasSuffix(pattern, "/") {
		dir := strings.TrimSuffix(pattern, "/")
		if strings.HasPrefix(filePath, dir+"/") || strings.Contains(filePath, "/"+dir+"/") {
			return true
		}
	}

	if strings.HasPrefix(pattern, "*") {
		suffix := strings.TrimPrefix(pattern, "*")
		if strings.HasSuffix(filePath, suffix) {
			return true
		}
	}

	if matched, _ := filepath.Match(pattern, filepath.Base(filePath)); matched {
		return true
	}

	return strings.Contains(filePath, pattern)
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

func (s *WebhookService) postGitLabMRComment(project *models.Project, mrIID int, comment string) error {
	info, err := parseRepoInfo(project.URL)
	if err != nil {
		return err
	}

	apiURL := fmt.Sprintf("%s/api/v4/projects/%s/merge_requests/%d/notes",
		info.baseURL, strings.ReplaceAll(info.projectPath, "/", "%2F"), mrIID)

	body := fmt.Sprintf(`{"body": %q}`, comment)
	req, err := http.NewRequest("POST", apiURL, strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if project.AccessToken != "" {
		req.Header.Set("PRIVATE-TOKEN", project.AccessToken)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GitLab API returned %d: %s", resp.StatusCode, string(respBody))
	}

	log.Printf("[Webhook] Posted comment to GitLab MR %d", mrIID)
	return nil
}

func (s *WebhookService) postGitHubPRComment(project *models.Project, prNumber int, comment string) error {
	info, err := parseRepoInfo(project.URL)
	if err != nil {
		return err
	}

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues/%d/comments", info.owner, info.repo, prNumber)

	body := fmt.Sprintf(`{"body": %q}`, comment)
	req, err := http.NewRequest("POST", apiURL, strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if project.AccessToken != "" {
		req.Header.Set("Authorization", "token "+project.AccessToken)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GitHub API returned %d: %s", resp.StatusCode, string(respBody))
	}

	log.Printf("[Webhook] Posted comment to GitHub PR %d", prNumber)
	return nil
}

func (s *WebhookService) formatReviewComment(score float64, reviewResult string) string {
	return fmt.Sprintf("## 🤖 AI Code Review\n\n**Score: %.0f/100**\n\n%s\n\n---\n*Powered by CodeSentry*", score, reviewResult)
}
