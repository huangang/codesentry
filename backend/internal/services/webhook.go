package services

import (
	"bytes"
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

// DefaultIgnorePatterns - files that should be skipped by default (config, lock, generated files)
const DefaultIgnorePatterns = "*.json,*.yaml,*.yml,*.toml,*.xml,*.ini,*.env,*.config," +
	"*.lock,package-lock.json,yarn.lock,pnpm-lock.yaml,go.sum,Cargo.lock,composer.lock,Gemfile.lock,poetry.lock," +
	"*.min.js,*.min.css,*.bundle.js,*.bundle.css," +
	"dist/,build/,out/,target/,.next/," +
	"vendor/,node_modules/,__pycache__/,.venv/,venv/"

type WebhookService struct {
	db                  *gorm.DB
	projectService      *ProjectService
	reviewService       *ReviewLogService
	aiService           *AIService
	notificationService *NotificationService
	configService       *SystemConfigService
	fileContextService  *FileContextService
	httpClient          *http.Client
}

func NewWebhookService(db *gorm.DB, aiCfg *config.OpenAIConfig) *WebhookService {
	configService := NewSystemConfigService(db)
	return &WebhookService{
		db:                  db,
		projectService:      NewProjectService(db),
		reviewService:       NewReviewLogService(db),
		aiService:           NewAIService(db, aiCfg),
		notificationService: NewNotificationService(db),
		configService:       configService,
		fileContextService:  NewFileContextService(configService),
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
	urlStr := strings.TrimSuffix(projectURL, ".git")

	protocolIdx := strings.Index(urlStr, "://")
	if protocolIdx == -1 {
		return nil, fmt.Errorf("invalid project URL (no protocol): %s", projectURL)
	}

	protocol := urlStr[:protocolIdx+3]
	rest := urlStr[protocolIdx+3:]

	slashIdx := strings.Index(rest, "/")
	if slashIdx == -1 {
		return nil, fmt.Errorf("invalid project URL (no path): %s", projectURL)
	}

	host := rest[:slashIdx]
	projectPath := rest[slashIdx+1:]

	if projectPath == "" {
		return nil, fmt.Errorf("invalid project URL (empty project path): %s", projectURL)
	}

	pathParts := strings.Split(projectPath, "/")
	if len(pathParts) < 2 {
		return nil, fmt.Errorf("invalid project URL (need at least owner/repo): %s", projectURL)
	}

	return &repoInfo{
		owner:       pathParts[len(pathParts)-2],
		repo:        pathParts[len(pathParts)-1],
		projectPath: projectPath,
		baseURL:     protocol + host,
	}, nil
}

// isEmptyDiff checks if the diff content has no actual code changes
// A diff is considered empty if it only contains commit headers or whitespace
func isEmptyDiff(diff string) bool {
	if diff == "" {
		return true
	}
	// Remove commit headers like "### Commit: abc12345"
	lines := strings.Split(diff, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Skip commit header lines
		if strings.HasPrefix(line, "### Commit:") {
			continue
		}
		// If we find any other non-empty line, it's not empty
		return false
	}
	return true
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

// BitbucketPushEvent represents a Bitbucket push webhook event
type BitbucketPushEvent struct {
	Push struct {
		Changes []struct {
			New struct {
				Name   string `json:"name"`
				Type   string `json:"type"`
				Target struct {
					Hash    string `json:"hash"`
					Message string `json:"message"`
					Date    string `json:"date"`
					Author  struct {
						Raw  string `json:"raw"`
						User struct {
							DisplayName string `json:"display_name"`
							UUID        string `json:"uuid"`
							AccountID   string `json:"account_id"`
							Nickname    string `json:"nickname"`
							Links       struct {
								Avatar struct {
									Href string `json:"href"`
								} `json:"avatar"`
								HTML struct {
									Href string `json:"href"`
								} `json:"html"`
							} `json:"links"`
						} `json:"user"`
					} `json:"author"`
					Links struct {
						HTML struct {
							Href string `json:"href"`
						} `json:"html"`
					} `json:"links"`
				} `json:"target"`
			} `json:"new"`
			Old struct {
				Name   string `json:"name"`
				Target struct {
					Hash string `json:"hash"`
				} `json:"target"`
			} `json:"old"`
			Commits []struct {
				Hash    string `json:"hash"`
				Message string `json:"message"`
				Author  struct {
					Raw  string `json:"raw"`
					User struct {
						DisplayName string `json:"display_name"`
					} `json:"user"`
				} `json:"author"`
				Links struct {
					HTML struct {
						Href string `json:"href"`
					} `json:"html"`
				} `json:"links"`
			} `json:"commits"`
		} `json:"changes"`
	} `json:"push"`
	Repository struct {
		UUID     string `json:"uuid"`
		Name     string `json:"name"`
		FullName string `json:"full_name"`
		Links    struct {
			HTML struct {
				Href string `json:"href"`
			} `json:"html"`
		} `json:"links"`
	} `json:"repository"`
	Actor struct {
		DisplayName string `json:"display_name"`
		UUID        string `json:"uuid"`
		AccountID   string `json:"account_id"`
		Nickname    string `json:"nickname"`
		Links       struct {
			Avatar struct {
				Href string `json:"href"`
			} `json:"avatar"`
			HTML struct {
				Href string `json:"href"`
			} `json:"html"`
		} `json:"links"`
	} `json:"actor"`
}

// BitbucketPREvent represents a Bitbucket pull request webhook event
type BitbucketPREvent struct {
	PullRequest struct {
		ID          int    `json:"id"`
		Title       string `json:"title"`
		Description string `json:"description"`
		State       string `json:"state"`
		Source      struct {
			Branch struct {
				Name string `json:"name"`
			} `json:"branch"`
			Commit struct {
				Hash string `json:"hash"`
			} `json:"commit"`
		} `json:"source"`
		Destination struct {
			Branch struct {
				Name string `json:"name"`
			} `json:"branch"`
		} `json:"destination"`
		Author struct {
			DisplayName string `json:"display_name"`
			UUID        string `json:"uuid"`
			AccountID   string `json:"account_id"`
			Nickname    string `json:"nickname"`
			Links       struct {
				Avatar struct {
					Href string `json:"href"`
				} `json:"avatar"`
				HTML struct {
					Href string `json:"href"`
				} `json:"html"`
			} `json:"links"`
		} `json:"author"`
		Links struct {
			HTML struct {
				Href string `json:"href"`
			} `json:"html"`
		} `json:"links"`
	} `json:"pullrequest"`
	Repository struct {
		UUID     string `json:"uuid"`
		Name     string `json:"name"`
		FullName string `json:"full_name"`
		Links    struct {
			HTML struct {
				Href string `json:"href"`
			} `json:"html"`
		} `json:"links"`
	} `json:"repository"`
	Actor struct {
		DisplayName string `json:"display_name"`
		UUID        string `json:"uuid"`
		Links       struct {
			Avatar struct {
				Href string `json:"href"`
			} `json:"avatar"`
		} `json:"links"`
	} `json:"actor"`
}

// HandleGitLabWebhook processes GitLab webhook events
func (s *WebhookService) HandleGitLabWebhook(ctx context.Context, projectID uint, eventType string, body []byte) error {
	log.Printf("[Webhook] Received GitLab webhook: projectID=%d, eventType=%s", projectID, eventType)

	project, err := s.projectService.GetByID(projectID)
	if err != nil {
		log.Printf("[Webhook] Project not found: %d, error: %v", projectID, err)
		return fmt.Errorf("project not found: %w", err)
	}

	if !project.AIEnabled {
		log.Printf("[Webhook] AI disabled for project %d, skipping", projectID)
		return nil
	}

	switch eventType {
	case "Push Hook":
		if !strings.Contains(project.ReviewEvents, "push") {
			log.Printf("[Webhook] Push events not enabled for project %d, skipping", projectID)
			return nil
		}
		var event GitLabPushEvent
		if err := json.Unmarshal(body, &event); err != nil {
			log.Printf("[Webhook] Failed to parse GitLab push event: %v", err)
			return err
		}
		return s.processGitLabPush(ctx, project, &event)

	case "Merge Request Hook":
		if !strings.Contains(project.ReviewEvents, "merge_request") {
			log.Printf("[Webhook] MR events not enabled for project %d, skipping", projectID)
			return nil
		}
		var event GitLabMREvent
		if err := json.Unmarshal(body, &event); err != nil {
			log.Printf("[Webhook] Failed to parse GitLab MR event: %v", err)
			return err
		}
		return s.processGitLabMR(ctx, project, &event)
	default:
		log.Printf("[Webhook] Unknown event type: %s, skipping", eventType)
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

	branch := strings.TrimPrefix(event.Ref, "refs/heads/")
	if s.isBranchIgnored(branch, project.BranchFilter) {
		log.Printf("[Webhook] Branch %s is in ignore list, skipping review", branch)
		return nil
	}

	var commits []string
	for _, c := range event.Commits {
		commits = append(commits, fmt.Sprintf("%s: %s", c.ID[:8], c.Message))
	}

	commitSHA := event.CheckoutSHA
	if commitSHA == "" {
		commitSHA = event.Commits[len(event.Commits)-1].ID
	}

	if s.isCommitAlreadyReviewed(project.ID, commitSHA) {
		log.Printf("[Webhook] Commit %s already reviewed, skipping", commitSHA[:8])
		return nil
	}

	log.Printf("[Webhook] Processing GitLab push: %d commits, checkout_sha=%s, using commit=%s",
		len(event.Commits), event.CheckoutSHA, commitSHA)

	LogInfo("Webhook", "GitLabPush", fmt.Sprintf("Processing push from %s: %d commits", event.UserName, len(event.Commits)), nil, "", "", map[string]interface{}{
		"project_id": project.ID,
		"branch":     strings.TrimPrefix(event.Ref, "refs/heads/"),
		"commit":     commitSHA,
	})

	// Set pending status
	s.setCommitStatus(project, commitSHA, "pending", "AI Review in progress...", event.ProjectID)

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

	additions, deletions, filesChanged := parseDiffStats(diff)

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
		FilesChanged:  filesChanged,
		Additions:     additions,
		Deletions:     deletions,
		ReviewStatus:  "pending",
	}
	s.reviewService.Create(reviewLog)

	// Publish SSE event: pending
	PublishReviewEvent(reviewLog.ID, reviewLog.ProjectID, reviewLog.CommitHash, "pending", nil, "")

	log.Printf("[Webhook] Starting AI review for project %d, commit %s", project.ID, commitSHA)

	filteredDiff := s.filterDiff(diff, project.FileExtensions, project.IgnorePatterns)
	if filteredDiff != diff {
		log.Printf("[Webhook] Filtered diff by extensions (%s) and ignore patterns (%s): %d -> %d bytes",
			project.FileExtensions, project.IgnorePatterns, len(diff), len(filteredDiff))
	}

	// Skip AI review for empty commits (no actual code changes after filtering)
	if isEmptyDiff(filteredDiff) {
		log.Printf("[Webhook] WARNING: Empty commit detected for project %d, commit %s - skipping AI review", project.ID, commitSHA)
		LogWarning("Webhook", "EmptyCommit", fmt.Sprintf("Empty commit %s detected, skipping AI review", commitSHA[:8]), nil, "", "", map[string]interface{}{
			"project_id": project.ID,
			"commit":     commitSHA,
			"branch":     branch,
		})
		reviewLog.ReviewStatus = "skipped"
		reviewLog.ReviewResult = "Empty commit - no code changes to review"
		s.reviewService.Update(reviewLog)
		PublishReviewEvent(reviewLog.ID, reviewLog.ProjectID, reviewLog.CommitHash, "skipped", nil, "Empty commit - no code changes")
		s.setCommitStatus(project, commitSHA, "success", "AI Review Skipped: Empty commit", event.ProjectID)
		return nil
	}

	var fileContext string
	if s.fileContextService.IsEnabled() {
		fileContext, _ = s.fileContextService.BuildFileContext(project, filteredDiff, commitSHA)
		if fileContext != "" {
			log.Printf("[Webhook] Built file context: %d chars", len(fileContext))
		}
	}

	// Publish SSE event: analyzing
	PublishReviewEvent(reviewLog.ID, reviewLog.ProjectID, reviewLog.CommitHash, "analyzing", nil, "")

	result, err := s.aiService.ReviewChunked(ctx, &ReviewRequest{
		ProjectID:   project.ID,
		Diffs:       filteredDiff,
		Commits:     strings.Join(commits, "\n"),
		FileContext: fileContext,
	})

	if err != nil {
		log.Printf("[Webhook] AI review failed: %v", err)
		LogError("AIReview", "ReviewFailed", err.Error(), nil, "", "", map[string]interface{}{
			"project_id": project.ID,
			"commit":     commitSHA,
		})
		reviewLog.ReviewStatus = "failed"
		reviewLog.ErrorMessage = err.Error()
		// Publish SSE event: failed
		PublishReviewEvent(reviewLog.ID, reviewLog.ProjectID, reviewLog.CommitHash, "failed", nil, err.Error())
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
		// Publish SSE event: completed
		PublishReviewEvent(reviewLog.ID, reviewLog.ProjectID, reviewLog.CommitHash, "completed", &result.Score, "")

		s.notificationService.SendReviewNotification(project, &ReviewNotification{
			ProjectName:   project.Name,
			Branch:        branch,
			Author:        event.UserName,
			CommitMessage: strings.Join(commits, "\n"),
			Score:         result.Score,
			ReviewResult:  result.Content,
			EventType:     "push",
		})

		if project.CommentEnabled {
			comment := s.formatReviewComment(result.Score, result.Content)
			if err := s.postGitLabCommitComment(project, commitSHA, comment); err != nil {
				log.Printf("[Webhook] Failed to post GitLab commit comment: %v", err)
			} else {
				reviewLog.CommentPosted = true
			}
		}

		// Set final status
		minScore := s.getEffectiveMinScore(project)
		if result.Score >= minScore {
			s.setCommitStatus(project, commitSHA, "success", fmt.Sprintf("AI Review Passed: %.0f/%.0f", result.Score, minScore), event.ProjectID)
		} else {
			s.setCommitStatus(project, commitSHA, "failed", fmt.Sprintf("AI Review Failed: %.0f (Min: %.0f)", result.Score, minScore), event.ProjectID)
		}
	}

	return s.reviewService.Update(reviewLog)
}

func (s *WebhookService) processGitLabMR(ctx context.Context, project *models.Project, event *GitLabMREvent) error {
	if event.ObjectAttributes.Action != "open" && event.ObjectAttributes.Action != "update" {
		return nil
	}

	if s.isBranchIgnored(event.ObjectAttributes.SourceBranch, project.BranchFilter) {
		log.Printf("[Webhook] Branch %s is in ignore list, skipping review", event.ObjectAttributes.SourceBranch)
		return nil
	}

	mrNumber := event.ObjectAttributes.IID

	mrSHA, _ := s.getGitLabRequestSHA(project, mrNumber)
	if mrSHA != "" {
		s.setCommitStatus(project, mrSHA, "pending", "AI Review in progress...", event.Project.ID)
	}

	diff, err := s.getGitLabMRDiff(project, mrNumber)
	if err != nil {
		diff = "Failed to get diff: " + err.Error()
	}

	additions, deletions, filesChanged := parseDiffStats(diff)

	reviewLog := &models.ReviewLog{
		ProjectID:     project.ID,
		EventType:     "merge_request",
		Branch:        event.ObjectAttributes.SourceBranch,
		Author:        event.User.Name,
		AuthorEmail:   event.User.Email,
		AuthorAvatar:  event.User.AvatarURL,
		CommitMessage: event.ObjectAttributes.Title,
		FilesChanged:  filesChanged,
		Additions:     additions,
		Deletions:     deletions,
		MRNumber:      &mrNumber,
		MRURL:         event.ObjectAttributes.URL,
		ReviewStatus:  "pending",
	}
	s.reviewService.Create(reviewLog)

	filteredDiff := s.filterDiff(diff, project.FileExtensions, project.IgnorePatterns)

	// Skip AI review for empty MR (no actual code changes after filtering)
	if isEmptyDiff(filteredDiff) {
		log.Printf("[Webhook] WARNING: Empty MR detected for project %d, MR !%d - skipping AI review", project.ID, mrNumber)
		LogWarning("Webhook", "EmptyMR", fmt.Sprintf("Empty MR !%d detected, skipping AI review", mrNumber), nil, "", "", map[string]interface{}{
			"project_id": project.ID,
			"mr_number":  mrNumber,
			"branch":     event.ObjectAttributes.SourceBranch,
		})
		reviewLog.ReviewStatus = "skipped"
		reviewLog.ReviewResult = "Empty merge request - no code changes to review"
		s.reviewService.Update(reviewLog)
		if mrSHA != "" {
			s.setCommitStatus(project, mrSHA, "success", "AI Review Skipped: Empty MR", event.Project.ID)
		}
		return nil
	}

	var fileContext string
	if s.fileContextService.IsEnabled() {
		if mrSHA != "" {
			fileContext, _ = s.fileContextService.BuildFileContext(project, filteredDiff, mrSHA)
		}
		if fileContext != "" {
			log.Printf("[Webhook] Built file context for MR: %d chars", len(fileContext))
		}
	}

	result, err := s.aiService.ReviewChunked(ctx, &ReviewRequest{
		ProjectID:   project.ID,
		Diffs:       filteredDiff,
		Commits:     event.ObjectAttributes.Title + "\n" + event.ObjectAttributes.Description,
		FileContext: fileContext,
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
			} else {
				reviewLog.CommentPosted = true
			}
		}
	}

	s.reviewService.Update(reviewLog)

	if mrSHA != "" {
		minScore := s.getEffectiveMinScore(project)
		if reviewLog.ReviewStatus == "completed" && reviewLog.Score != nil {
			if *reviewLog.Score >= minScore {
				s.setCommitStatus(project, mrSHA, "success", fmt.Sprintf("AI Review Passed: %.0f/%.0f", *reviewLog.Score, minScore), event.Project.ID)
			} else {
				s.setCommitStatus(project, mrSHA, "failed", fmt.Sprintf("AI Review Failed: %.0f (Min: %.0f)", *reviewLog.Score, minScore), event.Project.ID)
			}
		} else {
			s.setCommitStatus(project, mrSHA, "failed", "AI Review Failed/Error", event.Project.ID)
		}
	}

	return nil
}

// HandleBitbucketWebhook processes Bitbucket webhook events
func (s *WebhookService) HandleBitbucketWebhook(ctx context.Context, projectID uint, eventType string, body []byte) error {
	log.Printf("[Webhook] Received Bitbucket webhook: projectID=%d, eventType=%s", projectID, eventType)

	project, err := s.projectService.GetByID(projectID)
	if err != nil {
		log.Printf("[Webhook] Project not found: %d, error: %v", projectID, err)
		return fmt.Errorf("project not found: %w", err)
	}

	if !project.AIEnabled {
		log.Printf("[Webhook] AI disabled for project %d, skipping", projectID)
		return nil
	}

	switch eventType {
	case "repo:push":
		if !strings.Contains(project.ReviewEvents, "push") {
			log.Printf("[Webhook] Push events not enabled for project %d, skipping", projectID)
			return nil
		}
		var event BitbucketPushEvent
		if err := json.Unmarshal(body, &event); err != nil {
			log.Printf("[Webhook] Failed to parse Bitbucket push event: %v", err)
			return err
		}
		return s.processBitbucketPush(ctx, project, &event)

	case "pullrequest:created", "pullrequest:updated":
		if !strings.Contains(project.ReviewEvents, "merge_request") {
			log.Printf("[Webhook] PR events not enabled for project %d, skipping", projectID)
			return nil
		}
		var event BitbucketPREvent
		if err := json.Unmarshal(body, &event); err != nil {
			log.Printf("[Webhook] Failed to parse Bitbucket PR event: %v", err)
			return err
		}
		return s.processBitbucketPR(ctx, project, &event)
	default:
		log.Printf("[Webhook] Unknown Bitbucket event type: %s, skipping", eventType)
	}

	return nil
}

func (s *WebhookService) processBitbucketPush(ctx context.Context, project *models.Project, event *BitbucketPushEvent) error {
	if len(event.Push.Changes) == 0 {
		return nil
	}

	for _, change := range event.Push.Changes {
		if change.New.Type != "branch" || len(change.Commits) == 0 {
			continue
		}

		branch := change.New.Name
		if s.isBranchIgnored(branch, project.BranchFilter) {
			log.Printf("[Webhook] Branch %s is in ignore list, skipping review", branch)
			continue
		}

		commitSHA := change.New.Target.Hash
		if s.isCommitAlreadyReviewed(project.ID, commitSHA) {
			log.Printf("[Webhook] Commit %s already reviewed, skipping", commitSHA[:8])
			continue
		}

		var commits []string
		for _, c := range change.Commits {
			commits = append(commits, fmt.Sprintf("%s: %s", c.Hash[:8], c.Message))
		}

		log.Printf("[Webhook] Processing Bitbucket push: %d commits, branch=%s, commit=%s",
			len(change.Commits), branch, commitSHA[:8])

		LogInfo("Webhook", "BitbucketPush", fmt.Sprintf("Processing push from %s: %d commits", event.Actor.DisplayName, len(change.Commits)), nil, "", "", map[string]interface{}{
			"project_id": project.ID,
			"branch":     branch,
			"commit":     commitSHA,
		})

		s.setBitbucketCommitStatus(project, commitSHA, "INPROGRESS", "AI Review in progress...")

		var allDiffs strings.Builder
		for _, c := range change.Commits {
			diff, err := s.getBitbucketDiff(project, c.Hash)
			if err != nil {
				log.Printf("[Webhook] Failed to get diff for commit %s: %v", c.Hash[:8], err)
				continue
			}
			allDiffs.WriteString(fmt.Sprintf("\n### Commit: %s\n%s\n", c.Hash[:8], diff))
		}

		diff := allDiffs.String()
		if diff == "" {
			diff = "Failed to get diff for all commits"
			log.Printf("[Webhook] No diffs retrieved for any commits")
		} else {
			log.Printf("[Webhook] Got combined diffs, total length: %d bytes", len(diff))
		}

		additions, deletions, filesChanged := parseDiffStats(diff)

		var commitURL string
		if change.New.Target.Links.HTML.Href != "" {
			commitURL = change.New.Target.Links.HTML.Href
		}

		authorName := event.Actor.DisplayName
		authorAvatar := event.Actor.Links.Avatar.Href
		authorURL := event.Actor.Links.HTML.Href
		if change.New.Target.Author.User.DisplayName != "" {
			authorName = change.New.Target.Author.User.DisplayName
			authorAvatar = change.New.Target.Author.User.Links.Avatar.Href
			authorURL = change.New.Target.Author.User.Links.HTML.Href
		}

		reviewLog := &models.ReviewLog{
			ProjectID:     project.ID,
			EventType:     "push",
			CommitHash:    commitSHA,
			CommitURL:     commitURL,
			Branch:        branch,
			Author:        authorName,
			AuthorAvatar:  authorAvatar,
			AuthorURL:     authorURL,
			CommitMessage: strings.Join(commits, "\n"),
			FilesChanged:  filesChanged,
			Additions:     additions,
			Deletions:     deletions,
			ReviewStatus:  "pending",
		}
		s.reviewService.Create(reviewLog)

		log.Printf("[Webhook] Starting AI review for project %d, commit %s", project.ID, commitSHA[:8])

		filteredDiff := s.filterDiff(diff, project.FileExtensions, project.IgnorePatterns)
		if filteredDiff != diff {
			log.Printf("[Webhook] Filtered diff by extensions (%s) and ignore patterns (%s): %d -> %d bytes",
				project.FileExtensions, project.IgnorePatterns, len(diff), len(filteredDiff))
		}

		// Skip AI review for empty commits (no actual code changes after filtering)
		if isEmptyDiff(filteredDiff) {
			log.Printf("[Webhook] WARNING: Empty commit detected for project %d, commit %s - skipping AI review", project.ID, commitSHA[:8])
			LogWarning("Webhook", "EmptyCommit", fmt.Sprintf("Empty commit %s detected, skipping AI review", commitSHA[:8]), nil, "", "", map[string]interface{}{
				"project_id": project.ID,
				"commit":     commitSHA,
				"branch":     branch,
			})
			reviewLog.ReviewStatus = "skipped"
			reviewLog.ReviewResult = "Empty commit - no code changes to review"
			s.reviewService.Update(reviewLog)
			s.setBitbucketCommitStatus(project, commitSHA, "SUCCESSFUL", "AI Review Skipped: Empty commit")
			continue
		}

		var fileContext string
		if s.fileContextService.IsEnabled() {
			fileContext, _ = s.fileContextService.BuildFileContext(project, filteredDiff, commitSHA)
			if fileContext != "" {
				log.Printf("[Webhook] Built file context: %d chars", len(fileContext))
			}
		}

		result, err := s.aiService.ReviewChunked(ctx, &ReviewRequest{
			ProjectID:   project.ID,
			Diffs:       filteredDiff,
			Commits:     strings.Join(commits, "\n"),
			FileContext: fileContext,
		})

		if err != nil {
			log.Printf("[Webhook] AI review failed: %v", err)
			LogError("AIReview", "ReviewFailed", err.Error(), nil, "", "", map[string]interface{}{
				"project_id": project.ID,
				"commit":     commitSHA,
			})
			reviewLog.ReviewStatus = "failed"
			reviewLog.ErrorMessage = err.Error()
			s.setBitbucketCommitStatus(project, commitSHA, "FAILED", "AI Review Failed")
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
				Author:        authorName,
				CommitMessage: strings.Join(commits, "\n"),
				Score:         result.Score,
				ReviewResult:  result.Content,
				EventType:     "push",
			})

			if project.CommentEnabled {
				comment := s.formatReviewComment(result.Score, result.Content)
				if err := s.postBitbucketCommitComment(project, commitSHA, comment); err != nil {
					log.Printf("[Webhook] Failed to post Bitbucket commit comment: %v", err)
				} else {
					reviewLog.CommentPosted = true
				}
			}

			minScore := s.getEffectiveMinScore(project)
			if result.Score >= minScore {
				s.setBitbucketCommitStatus(project, commitSHA, "SUCCESSFUL", fmt.Sprintf("AI Review Passed: %.0f/%.0f", result.Score, minScore))
			} else {
				s.setBitbucketCommitStatus(project, commitSHA, "FAILED", fmt.Sprintf("AI Review Failed: %.0f (Min: %.0f)", result.Score, minScore))
			}
		}

		if err := s.reviewService.Update(reviewLog); err != nil {
			log.Printf("[Webhook] Failed to update review log: %v", err)
		}
	}

	return nil
}

func (s *WebhookService) processBitbucketPR(ctx context.Context, project *models.Project, event *BitbucketPREvent) error {
	branch := event.PullRequest.Source.Branch.Name
	if s.isBranchIgnored(branch, project.BranchFilter) {
		log.Printf("[Webhook] Branch %s is in ignore list, skipping review", branch)
		return nil
	}

	prNumber := event.PullRequest.ID
	commitSHA := event.PullRequest.Source.Commit.Hash

	s.setBitbucketCommitStatus(project, commitSHA, "INPROGRESS", "AI Review in progress...")

	diff, err := s.getBitbucketPRDiff(project, prNumber)
	if err != nil {
		diff = "Failed to get diff: " + err.Error()
	}

	additions, deletions, filesChanged := parseDiffStats(diff)

	reviewLog := &models.ReviewLog{
		ProjectID:     project.ID,
		EventType:     "merge_request",
		CommitHash:    commitSHA,
		Branch:        branch,
		Author:        event.PullRequest.Author.DisplayName,
		AuthorAvatar:  event.PullRequest.Author.Links.Avatar.Href,
		AuthorURL:     event.PullRequest.Author.Links.HTML.Href,
		CommitMessage: event.PullRequest.Title,
		FilesChanged:  filesChanged,
		Additions:     additions,
		Deletions:     deletions,
		MRNumber:      &prNumber,
		MRURL:         event.PullRequest.Links.HTML.Href,
		ReviewStatus:  "pending",
	}
	s.reviewService.Create(reviewLog)

	filteredDiff := s.filterDiff(diff, project.FileExtensions, project.IgnorePatterns)

	// Skip AI review for empty PR (no actual code changes after filtering)
	if isEmptyDiff(filteredDiff) {
		log.Printf("[Webhook] WARNING: Empty PR detected for project %d, PR #%d - skipping AI review", project.ID, prNumber)
		LogWarning("Webhook", "EmptyPR", fmt.Sprintf("Empty PR #%d detected, skipping AI review", prNumber), nil, "", "", map[string]interface{}{
			"project_id": project.ID,
			"pr_number":  prNumber,
			"branch":     branch,
		})
		reviewLog.ReviewStatus = "skipped"
		reviewLog.ReviewResult = "Empty pull request - no code changes to review"
		s.reviewService.Update(reviewLog)
		s.setBitbucketCommitStatus(project, commitSHA, "SUCCESSFUL", "AI Review Skipped: Empty PR")
		return nil
	}

	var fileContext string
	if s.fileContextService.IsEnabled() {
		fileContext, _ = s.fileContextService.BuildFileContext(project, filteredDiff, commitSHA)
		if fileContext != "" {
			log.Printf("[Webhook] Built file context for Bitbucket PR: %d chars", len(fileContext))
		}
	}

	result, err := s.aiService.ReviewChunked(ctx, &ReviewRequest{
		ProjectID:   project.ID,
		Diffs:       filteredDiff,
		Commits:     event.PullRequest.Title + "\n" + event.PullRequest.Description,
		FileContext: fileContext,
	})

	if err != nil {
		reviewLog.ReviewStatus = "failed"
		reviewLog.ErrorMessage = err.Error()
		s.setBitbucketCommitStatus(project, commitSHA, "FAILED", "AI Review Failed")
	} else {
		reviewLog.ReviewStatus = "completed"
		reviewLog.ReviewResult = result.Content
		reviewLog.Score = &result.Score

		s.notificationService.SendReviewNotification(project, &ReviewNotification{
			ProjectName:   project.Name,
			Branch:        branch,
			Author:        event.PullRequest.Author.DisplayName,
			CommitMessage: event.PullRequest.Title,
			Score:         result.Score,
			ReviewResult:  result.Content,
			EventType:     "merge_request",
			MRURL:         event.PullRequest.Links.HTML.Href,
		})

		if project.CommentEnabled {
			comment := s.formatReviewComment(result.Score, result.Content)
			if err := s.postBitbucketPRComment(project, prNumber, comment); err != nil {
				log.Printf("[Webhook] Failed to post Bitbucket PR comment: %v", err)
			} else {
				reviewLog.CommentPosted = true
			}
		}

		minScore := s.getEffectiveMinScore(project)
		if result.Score >= minScore {
			s.setBitbucketCommitStatus(project, commitSHA, "SUCCESSFUL", fmt.Sprintf("AI Review Passed: %.0f/%.0f", result.Score, minScore))
		} else {
			s.setBitbucketCommitStatus(project, commitSHA, "FAILED", fmt.Sprintf("AI Review Failed: %.0f (Min: %.0f)", result.Score, minScore))
		}
	}

	return s.reviewService.Update(reviewLog)
}

func (s *WebhookService) processGitHubPush(ctx context.Context, project *models.Project, event *GitHubPushEvent) error {
	if len(event.Commits) == 0 {
		return nil
	}

	branch := strings.TrimPrefix(event.Ref, "refs/heads/")
	if s.isBranchIgnored(branch, project.BranchFilter) {
		log.Printf("[Webhook] Branch %s is in ignore list, skipping review", branch)
		return nil
	}

	if s.isCommitAlreadyReviewed(project.ID, event.After) {
		log.Printf("[Webhook] Commit %s already reviewed, skipping", event.After[:8])
		return nil
	}

	var commits []string
	var commitURL string
	for _, c := range event.Commits {
		commits = append(commits, fmt.Sprintf("%s: %s", c.ID[:8], c.Message))
		if commitURL == "" && c.URL != "" {
			commitURL = c.URL
		}
	}

	diff, err := s.getGitHubDiff(project, event.After)
	if err != nil {
		diff = "Failed to get diff: " + err.Error()
	}

	additions, deletions, filesChanged := parseDiffStats(diff)

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
		FilesChanged:  filesChanged,
		Additions:     additions,
		Deletions:     deletions,
		ReviewStatus:  "pending",
	}
	s.reviewService.Create(reviewLog)

	filteredDiff := s.filterDiff(diff, project.FileExtensions, project.IgnorePatterns)

	// Skip AI review for empty commits (no actual code changes after filtering)
	if isEmptyDiff(filteredDiff) {
		log.Printf("[Webhook] WARNING: Empty commit detected for project %d, commit %s - skipping AI review", project.ID, event.After[:8])
		LogWarning("Webhook", "EmptyCommit", fmt.Sprintf("Empty commit %s detected, skipping AI review", event.After[:8]), nil, "", "", map[string]interface{}{
			"project_id": project.ID,
			"commit":     event.After,
			"branch":     branch,
		})
		reviewLog.ReviewStatus = "skipped"
		reviewLog.ReviewResult = "Empty commit - no code changes to review"
		s.reviewService.Update(reviewLog)
		return nil
	}

	var fileContext string
	if s.fileContextService.IsEnabled() {
		fileContext, _ = s.fileContextService.BuildFileContext(project, filteredDiff, event.After)
		if fileContext != "" {
			log.Printf("[Webhook] Built file context for GitHub push: %d chars", len(fileContext))
		}
	}

	result, err := s.aiService.ReviewChunked(ctx, &ReviewRequest{
		ProjectID:   project.ID,
		Diffs:       filteredDiff,
		Commits:     strings.Join(commits, "\n"),
		FileContext: fileContext,
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

		if project.CommentEnabled {
			comment := s.formatReviewComment(result.Score, result.Content)
			if err := s.postGitHubCommitComment(project, event.After, comment); err != nil {
				log.Printf("[Webhook] Failed to post GitHub commit comment: %v", err)
			} else {
				reviewLog.CommentPosted = true
			}
		}

		minScore := s.getEffectiveMinScore(project)
		if result.Score >= minScore {
			s.setCommitStatus(project, event.After, "success", fmt.Sprintf("AI Review Passed: %.0f/%.0f", result.Score, minScore), 0)
		} else {
			s.setCommitStatus(project, event.After, "failed", fmt.Sprintf("AI Review Failed: %.0f (Min: %.0f)", result.Score, minScore), 0)
		}
	}

	return s.reviewService.Update(reviewLog)
}

func (s *WebhookService) processGitHubPR(ctx context.Context, project *models.Project, event *GitHubPREvent) error {
	if event.Action != "opened" && event.Action != "synchronize" {
		return nil
	}

	if s.isBranchIgnored(event.PullRequest.Head.Ref, project.BranchFilter) {
		log.Printf("[Webhook] Branch %s is in ignore list, skipping review", event.PullRequest.Head.Ref)
		return nil
	}

	mrNumber := event.Number

	diff, err := s.getGitHubPRDiff(project, mrNumber)
	if err != nil {
		diff = "Failed to get diff: " + err.Error()
	}

	additions, deletions, filesChanged := parseDiffStats(diff)

	reviewLog := &models.ReviewLog{
		ProjectID:     project.ID,
		EventType:     "merge_request",
		CommitHash:    event.PullRequest.Head.SHA,
		Branch:        event.PullRequest.Head.Ref,
		Author:        event.PullRequest.User.Login,
		AuthorAvatar:  event.PullRequest.User.AvatarURL,
		AuthorURL:     event.PullRequest.User.HTMLURL,
		CommitMessage: event.PullRequest.Title,
		FilesChanged:  filesChanged,
		Additions:     additions,
		Deletions:     deletions,
		MRNumber:      &mrNumber,
		MRURL:         event.PullRequest.HTMLURL,
		ReviewStatus:  "pending",
	}
	s.reviewService.Create(reviewLog)

	filteredDiff := s.filterDiff(diff, project.FileExtensions, project.IgnorePatterns)

	// Skip AI review for empty PR (no actual code changes after filtering)
	if isEmptyDiff(filteredDiff) {
		log.Printf("[Webhook] WARNING: Empty PR detected for project %d, PR #%d - skipping AI review", project.ID, mrNumber)
		LogWarning("Webhook", "EmptyPR", fmt.Sprintf("Empty PR #%d detected, skipping AI review", mrNumber), nil, "", "", map[string]interface{}{
			"project_id": project.ID,
			"pr_number":  mrNumber,
			"branch":     event.PullRequest.Head.Ref,
		})
		reviewLog.ReviewStatus = "skipped"
		reviewLog.ReviewResult = "Empty pull request - no code changes to review"
		s.reviewService.Update(reviewLog)
		return nil
	}

	var fileContext string
	if s.fileContextService.IsEnabled() {
		fileContext, _ = s.fileContextService.BuildFileContext(project, filteredDiff, event.PullRequest.Head.SHA)
		if fileContext != "" {
			log.Printf("[Webhook] Built file context for GitHub PR: %d chars", len(fileContext))
		}
	}

	result, err := s.aiService.ReviewChunked(ctx, &ReviewRequest{
		ProjectID:   project.ID,
		Diffs:       filteredDiff,
		Commits:     event.PullRequest.Title + "\n" + event.PullRequest.Body,
		FileContext: fileContext,
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
			} else {
				reviewLog.CommentPosted = true
			}
		}

		minScore := s.getEffectiveMinScore(project)
		if result.Score >= minScore {
			s.setCommitStatus(project, event.PullRequest.Head.SHA, "success", fmt.Sprintf("AI Review Passed: %.0f/%.0f", result.Score, minScore), 0)
		} else {
			s.setCommitStatus(project, event.PullRequest.Head.SHA, "failed", fmt.Sprintf("AI Review Failed: %.0f (Min: %.0f)", result.Score, minScore), 0)
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
		Diff        string `json:"diff"`
		OldPath     string `json:"old_path"`
		NewPath     string `json:"new_path"`
		NewFile     bool   `json:"new_file"`
		RenamedFile bool   `json:"renamed_file"`
		DeletedFile bool   `json:"deleted_file"`
	}
	if err := json.Unmarshal(body, &diffs); err != nil {
		return string(body), nil
	}

	var result strings.Builder
	for _, d := range diffs {
		result.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", d.OldPath, d.NewPath))
		result.WriteString(fmt.Sprintf("--- a/%s\n+++ b/%s\n", d.OldPath, d.NewPath))
		result.WriteString(d.Diff)
		if !strings.HasSuffix(d.Diff, "\n") {
			result.WriteString("\n")
		}
	}

	return result.String(), nil
}

func (s *WebhookService) isBranchIgnored(branch string, branchFilter string) bool {
	if branchFilter == "" {
		return false
	}

	for _, pattern := range strings.Split(branchFilter, ",") {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}

		if strings.HasSuffix(pattern, "*") {
			prefix := strings.TrimSuffix(pattern, "*")
			if strings.HasPrefix(branch, prefix) {
				return true
			}
		} else if pattern == branch {
			return true
		}
	}
	return false
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

	ignoreSet := make(map[string]bool)
	for _, pattern := range strings.Split(DefaultIgnorePatterns, ",") {
		pattern = strings.TrimSpace(pattern)
		if pattern != "" {
			ignoreSet[pattern] = true
		}
	}
	if ignorePatterns != "" {
		for _, pattern := range strings.Split(ignorePatterns, ",") {
			pattern = strings.TrimSpace(pattern)
			if pattern != "" {
				ignoreSet[pattern] = true
			}
		}
	}

	var ignoreList []string
	for pattern := range ignoreSet {
		ignoreList = append(ignoreList, pattern)
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
	return hmac.Equal([]byte(signature), []byte("sha256="+expectedMAC))
}

func (s *WebhookService) setCommitStatus(project *models.Project, sha string, state string, description string, gitlabProjectID int) {
	switch project.Platform {
	case "gitlab":
		s.setGitLabCommitStatus(project, sha, state, description, gitlabProjectID)
	case "github":
		s.setGitHubCommitStatus(project, sha, state, description)
	}
}

func (s *WebhookService) setGitLabCommitStatus(project *models.Project, sha string, state string, description string, gitlabProjectID int) {
	info, err := parseRepoInfo(project.URL)
	if err != nil {
		log.Printf("[Webhook] Failed to parse repo info for status update: %v", err)
		return
	}

	projectIdentifier := strings.ReplaceAll(info.projectPath, "/", "%2F")
	if gitlabProjectID != 0 {
		projectIdentifier = fmt.Sprintf("%d", gitlabProjectID)
	}

	apiURL := fmt.Sprintf("%s/api/v4/projects/%s/statuses/%s",
		info.baseURL, projectIdentifier, sha)

	data := map[string]string{
		"state":       state,
		"context":     "codesentry/ai-review",
		"description": description,
		// "target_url": ... logic to link to review detail ...
	}

	payload, _ := json.Marshal(data)
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(payload))
	if err != nil {
		log.Printf("[Webhook] Failed to create request: %v", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	if project.AccessToken != "" {
		req.Header.Set("PRIVATE-TOKEN", project.AccessToken)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		log.Printf("[Webhook] Failed to send commit status: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("[Webhook] Failed to set commit status (code %d): %s", resp.StatusCode, string(body))
	} else {
		log.Printf("[Webhook] Set commit status for %s to %s", sha[:8], state)
	}
}

func (s *WebhookService) setGitHubCommitStatus(project *models.Project, sha string, state string, description string) {
	info, err := parseRepoInfo(project.URL)
	if err != nil {
		log.Printf("[Webhook] Failed to parse repo info for GitHub status update: %v", err)
		return
	}

	githubState := state
	switch state {
	case "pending":
		githubState = "pending"
	case "success":
		githubState = "success"
	case "failed":
		githubState = "failure"
	default:
		githubState = "error"
	}

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/statuses/%s", info.owner, info.repo, sha)

	data := map[string]string{
		"state":       githubState,
		"context":     "codesentry/ai-review",
		"description": description,
	}

	payload, _ := json.Marshal(data)
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(payload))
	if err != nil {
		log.Printf("[Webhook] Failed to create GitHub status request: %v", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if project.AccessToken != "" {
		req.Header.Set("Authorization", "token "+project.AccessToken)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		log.Printf("[Webhook] Failed to send GitHub commit status: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("[Webhook] Failed to set GitHub commit status (code %d): %s", resp.StatusCode, string(body))
	} else {
		log.Printf("[Webhook] Set GitHub commit status for %s to %s", sha[:8], githubState)
	}
}

func (s *WebhookService) getEffectiveMinScore(project *models.Project) float64 {
	// 1. Check Project level
	if project.MinScore > 0 {
		return project.MinScore
	}

	// 2. Check System level
	var score float64
	if _, err := fmt.Sscanf(s.configService.GetWithDefault("system.min_score", "60"), "%f", &score); err == nil {
		return score
	}

	// 3. Default fallback
	return 60.0
}

func (s *WebhookService) getGitLabRequestSHA(project *models.Project, mrIID int) (string, error) {
	info, err := parseRepoInfo(project.URL)
	if err != nil {
		return "", err
	}

	apiURL := fmt.Sprintf("%s/api/v4/projects/%s/merge_requests/%d",
		info.baseURL, strings.ReplaceAll(info.projectPath, "/", "%2F"), mrIID)

	req, _ := http.NewRequest("GET", apiURL, nil)
	if project.AccessToken != "" {
		req.Header.Set("PRIVATE-TOKEN", project.AccessToken)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}

	// Note: GitLab MR API returns "sha" field in the root as the HEAD sha of the MR
	// Or sometimes "diff_refs.head_sha" is safer.
	// Let's decode to a generic map to be safe
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if sha, ok := result["sha"].(string); ok {
		return sha, nil
	}

	return "", fmt.Errorf("sha not found")
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

func (s *WebhookService) postGitLabCommitComment(project *models.Project, commitSHA string, comment string) error {
	info, err := parseRepoInfo(project.URL)
	if err != nil {
		return err
	}

	apiURL := fmt.Sprintf("%s/api/v4/projects/%s/repository/commits/%s/comments",
		info.baseURL, strings.ReplaceAll(info.projectPath, "/", "%2F"), commitSHA)

	body := fmt.Sprintf(`{"note": %q}`, comment)
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

	log.Printf("[Webhook] Posted comment to GitLab commit %s", commitSHA[:8])
	return nil
}

func (s *WebhookService) postGitHubCommitComment(project *models.Project, commitSHA string, comment string) error {
	info, err := parseRepoInfo(project.URL)
	if err != nil {
		return err
	}

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits/%s/comments", info.owner, info.repo, commitSHA)

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

	log.Printf("[Webhook] Posted comment to GitHub commit %s", commitSHA[:8])
	return nil
}

func (s *WebhookService) getBitbucketDiff(project *models.Project, commitSHA string) (string, error) {
	info, err := parseRepoInfo(project.URL)
	if err != nil {
		return "", err
	}

	apiURL := fmt.Sprintf("https://api.bitbucket.org/2.0/repositories/%s/diff/%s",
		info.projectPath, commitSHA)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", err
	}
	if project.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+project.AccessToken)
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

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Bitbucket API returned status %d: %s", resp.StatusCode, string(body))
	}

	return string(body), nil
}

func (s *WebhookService) getBitbucketPRDiff(project *models.Project, prNumber int) (string, error) {
	info, err := parseRepoInfo(project.URL)
	if err != nil {
		return "", err
	}

	apiURL := fmt.Sprintf("https://api.bitbucket.org/2.0/repositories/%s/pullrequests/%d/diff",
		info.projectPath, prNumber)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", err
	}
	if project.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+project.AccessToken)
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

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Bitbucket API returned status %d: %s", resp.StatusCode, string(body))
	}

	return string(body), nil
}

func (s *WebhookService) setBitbucketCommitStatus(project *models.Project, sha string, state string, description string) {
	info, err := parseRepoInfo(project.URL)
	if err != nil {
		log.Printf("[Webhook] Failed to parse repo info for Bitbucket status update: %v", err)
		return
	}

	apiURL := fmt.Sprintf("https://api.bitbucket.org/2.0/repositories/%s/commit/%s/statuses/build",
		info.projectPath, sha)

	data := map[string]string{
		"state":       state,
		"key":         "codesentry-ai-review",
		"name":        "CodeSentry AI Review",
		"description": description,
	}

	payload, _ := json.Marshal(data)
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(payload))
	if err != nil {
		log.Printf("[Webhook] Failed to create Bitbucket status request: %v", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	if project.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+project.AccessToken)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		log.Printf("[Webhook] Failed to send Bitbucket commit status: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("[Webhook] Failed to set Bitbucket commit status (code %d): %s", resp.StatusCode, string(body))
	} else {
		log.Printf("[Webhook] Set Bitbucket commit status for %s to %s", sha[:8], state)
	}
}

func (s *WebhookService) postBitbucketCommitComment(project *models.Project, commitSHA string, comment string) error {
	info, err := parseRepoInfo(project.URL)
	if err != nil {
		return err
	}

	apiURL := fmt.Sprintf("https://api.bitbucket.org/2.0/repositories/%s/commit/%s/comments",
		info.projectPath, commitSHA)

	data := map[string]interface{}{
		"content": map[string]string{
			"raw": comment,
		},
	}

	payload, _ := json.Marshal(data)
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if project.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+project.AccessToken)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Bitbucket API returned %d: %s", resp.StatusCode, string(respBody))
	}

	log.Printf("[Webhook] Posted comment to Bitbucket commit %s", commitSHA[:8])
	return nil
}

func (s *WebhookService) postBitbucketPRComment(project *models.Project, prNumber int, comment string) error {
	info, err := parseRepoInfo(project.URL)
	if err != nil {
		return err
	}

	apiURL := fmt.Sprintf("https://api.bitbucket.org/2.0/repositories/%s/pullrequests/%d/comments",
		info.projectPath, prNumber)

	data := map[string]interface{}{
		"content": map[string]string{
			"raw": comment,
		},
	}

	payload, _ := json.Marshal(data)
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if project.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+project.AccessToken)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Bitbucket API returned %d: %s", resp.StatusCode, string(respBody))
	}

	log.Printf("[Webhook] Posted comment to Bitbucket PR %d", prNumber)
	return nil
}

func VerifyBitbucketSignature(secret string, body []byte, signature string) bool {
	if secret == "" {
		return true
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expectedMAC := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(signature), []byte(expectedMAC))
}

func (s *WebhookService) formatReviewComment(score float64, reviewResult string) string {
	return fmt.Sprintf("## 🤖 AI Code Review\n\n**Score: %.0f/100**\n\n%s\n\n---\n*Powered by CodeSentry*", score, reviewResult)
}

func parseDiffStats(diff string) (additions, deletions, filesChanged int) {
	lines := strings.Split(diff, "\n")
	fileSet := make(map[string]bool)

	for _, line := range lines {
		if strings.HasPrefix(line, "+++ ") || strings.HasPrefix(line, "--- ") {
			continue
		}
		if strings.HasPrefix(line, "diff --git") {
			parts := strings.Split(line, " ")
			if len(parts) >= 4 {
				fileSet[parts[len(parts)-1]] = true
			}
			continue
		}
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			additions++
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			deletions++
		}
	}

	filesChanged = len(fileSet)
	return
}

func (s *WebhookService) isCommitAlreadyReviewed(projectID uint, commitSHA string) bool {
	var count int64
	s.db.Model(&models.ReviewLog{}).
		Where("project_id = ? AND commit_hash = ? AND review_status = ?", projectID, commitSHA, "completed").
		Count(&count)
	return count > 0
}

type ReviewScoreResponse struct {
	CommitSHA string   `json:"commit_sha"`
	Status    string   `json:"status"`
	Score     *float64 `json:"score,omitempty"`
	MinScore  float64  `json:"min_score,omitempty"`
	Passed    *bool    `json:"passed,omitempty"`
	ReviewID  uint     `json:"review_id"`
	Message   string   `json:"message"`
}

func (s *WebhookService) GetReviewScore(commitSHA string) (*ReviewScoreResponse, error) {
	var reviewLog models.ReviewLog
	if err := s.db.Where("commit_hash = ?", commitSHA).Order("created_at DESC").First(&reviewLog).Error; err != nil {
		return nil, fmt.Errorf("review not found for commit: %s", commitSHA)
	}

	resp := &ReviewScoreResponse{
		CommitSHA: commitSHA,
		Status:    reviewLog.ReviewStatus,
		ReviewID:  reviewLog.ID,
	}

	switch reviewLog.ReviewStatus {
	case "pending", "processing", "analyzing":
		resp.Message = "Review in progress"
	case "completed":
		var project models.Project
		s.db.First(&project, reviewLog.ProjectID)
		minScore := s.getEffectiveMinScore(&project)
		passed := reviewLog.Score != nil && *reviewLog.Score >= minScore
		resp.Score = reviewLog.Score
		resp.MinScore = minScore
		resp.Passed = &passed
		resp.Message = "Review completed"
	case "skipped":
		// Empty commits are skipped but should pass CI
		passed := true
		resp.Passed = &passed
		resp.Message = "Skipped: " + reviewLog.ReviewResult
	case "failed":
		resp.Message = "Review failed: " + reviewLog.ErrorMessage
	}

	return resp, nil
}

type SyncReviewRequest struct {
	ProjectURL string
	CommitSHA  string
	Ref        string
	Author     string
	Message    string
	Diffs      string
}

type SyncReviewResponse struct {
	Passed      bool    `json:"passed"`
	Score       float64 `json:"score"`
	MinScore    float64 `json:"min_score"`
	Message     string  `json:"message"`
	ReviewID    uint    `json:"review_id,omitempty"`
	FullContent string  `json:"full_content,omitempty"`
}

func (s *WebhookService) SyncReview(ctx context.Context, project *models.Project, req *SyncReviewRequest) (*SyncReviewResponse, error) {
	minScore := s.getEffectiveMinScore(project)

	branch := strings.TrimPrefix(req.Ref, "refs/heads/")
	if s.isBranchIgnored(branch, project.BranchFilter) {
		return &SyncReviewResponse{
			Passed:   true,
			Score:    100,
			MinScore: minScore,
			Message:  "Branch is in ignore list, skipping review",
		}, nil
	}

	if s.isCommitAlreadyReviewed(project.ID, req.CommitSHA) {
		return &SyncReviewResponse{
			Passed:   true,
			Score:    100,
			MinScore: minScore,
			Message:  "Commit already reviewed and passed",
		}, nil
	}

	additions, deletions, filesChanged := parseDiffStats(req.Diffs)

	reviewLog := &models.ReviewLog{
		ProjectID:     project.ID,
		EventType:     "push",
		Branch:        branch,
		CommitHash:    req.CommitSHA,
		Author:        req.Author,
		CommitMessage: req.Message,
		ReviewStatus:  "pending",
		Additions:     additions,
		Deletions:     deletions,
		FilesChanged:  filesChanged,
	}

	if err := s.reviewService.Create(reviewLog); err != nil {
		return nil, fmt.Errorf("failed to create review log: %w", err)
	}

	reviewLog.ReviewStatus = "processing"
	s.reviewService.Update(reviewLog)

	var fileContext string
	if s.fileContextService.IsEnabled() {
		fileContext, _ = s.fileContextService.BuildFileContext(project, req.Diffs, req.CommitSHA)
		if fileContext != "" {
			log.Printf("[Webhook] Built file context for sync review: %d chars", len(fileContext))
		}
	}

	result, err := s.aiService.ReviewChunked(ctx, &ReviewRequest{
		ProjectID:   project.ID,
		Diffs:       req.Diffs,
		Commits:     req.Message,
		FileContext: fileContext,
	})

	if err != nil {
		reviewLog.ReviewStatus = "failed"
		reviewLog.ErrorMessage = err.Error()
		s.reviewService.Update(reviewLog)
		return nil, fmt.Errorf("AI review failed: %w", err)
	}

	reviewLog.ReviewStatus = "completed"
	reviewLog.ReviewResult = result.Content
	reviewLog.Score = &result.Score
	s.reviewService.Update(reviewLog)

	passed := result.Score >= minScore
	message := fmt.Sprintf("Score: %.0f/100 (min: %.0f)", result.Score, minScore)
	if !passed {
		message = fmt.Sprintf("Review failed: %.0f/100 (min: %.0f required)", result.Score, minScore)
	}

	return &SyncReviewResponse{
		Passed:      passed,
		Score:       result.Score,
		MinScore:    minScore,
		Message:     message,
		ReviewID:    reviewLog.ID,
		FullContent: result.Content,
	}, nil
}

// ProcessReviewTask processes a review task from the async queue
// This method re-runs the AI review for an existing review log
func (s *WebhookService) ProcessReviewTask(ctx context.Context, task *ReviewTask) error {
	log.Printf("[TaskQueue] Processing review task: review_log_id=%d, project=%d, commit=%s",
		task.ReviewLogID, task.ProjectID, task.CommitSHA)

	// Get the review log
	reviewLog, err := s.reviewService.GetByID(task.ReviewLogID)
	if err != nil {
		return fmt.Errorf("review log not found: %w", err)
	}

	// Get the project
	project, err := s.projectService.GetByID(task.ProjectID)
	if err != nil {
		return fmt.Errorf("project not found: %w", err)
	}

	// Update status to analyzing
	reviewLog.ReviewStatus = "analyzing"
	s.reviewService.Update(reviewLog)
	PublishReviewEvent(reviewLog.ID, reviewLog.ProjectID, reviewLog.CommitHash, "analyzing", nil, "")

	// Filter diff
	filteredDiff := s.filterDiff(task.Diff, project.FileExtensions, project.IgnorePatterns)

	// Skip AI review for empty commits (no actual code changes after filtering)
	if isEmptyDiff(filteredDiff) {
		log.Printf("[TaskQueue] WARNING: Empty commit detected for review_log_id=%d - skipping AI review", task.ReviewLogID)
		LogWarning("TaskQueue", "EmptyCommit", fmt.Sprintf("Empty commit %s detected, skipping AI review", task.CommitSHA[:8]), nil, "", "", map[string]interface{}{
			"project_id":    task.ProjectID,
			"review_log_id": task.ReviewLogID,
			"commit":        task.CommitSHA,
		})
		reviewLog.ReviewStatus = "skipped"
		reviewLog.ReviewResult = "Empty commit - no code changes to review"
		s.reviewService.Update(reviewLog)
		PublishReviewEvent(reviewLog.ID, reviewLog.ProjectID, reviewLog.CommitHash, "skipped", nil, "Empty commit - no code changes")
		return nil
	}

	// Build file context if enabled
	var fileContext string
	if s.fileContextService.IsEnabled() {
		fileContext, _ = s.fileContextService.BuildFileContext(project, filteredDiff, task.CommitSHA)
	}

	// Run AI review
	result, err := s.aiService.ReviewChunked(ctx, &ReviewRequest{
		ProjectID:   project.ID,
		Diffs:       filteredDiff,
		Commits:     task.CommitMessage,
		FileContext: fileContext,
	})

	if err != nil {
		log.Printf("[TaskQueue] AI review failed: %v", err)
		reviewLog.ReviewStatus = "failed"
		reviewLog.ErrorMessage = err.Error()
		s.reviewService.Update(reviewLog)
		PublishReviewEvent(reviewLog.ID, reviewLog.ProjectID, reviewLog.CommitHash, "failed", nil, err.Error())
		return err
	}

	log.Printf("[TaskQueue] AI review completed, score: %.1f", result.Score)
	reviewLog.ReviewStatus = "completed"
	reviewLog.ReviewResult = result.Content
	reviewLog.Score = &result.Score
	s.reviewService.Update(reviewLog)
	PublishReviewEvent(reviewLog.ID, reviewLog.ProjectID, reviewLog.CommitHash, "completed", &result.Score, "")

	// Send notification
	s.notificationService.SendReviewNotification(project, &ReviewNotification{
		ProjectName:   project.Name,
		Branch:        task.Branch,
		Author:        task.Author,
		CommitMessage: task.CommitMessage,
		Score:         result.Score,
		ReviewResult:  result.Content,
		EventType:     task.EventType,
	})

	// Post comment if enabled
	if project.CommentEnabled && task.CommitSHA != "" {
		comment := s.formatReviewComment(result.Score, result.Content)
		switch project.Platform {
		case "gitlab":
			if err := s.postGitLabCommitComment(project, task.CommitSHA, comment); err != nil {
				log.Printf("[TaskQueue] Failed to post GitLab comment: %v", err)
			} else {
				reviewLog.CommentPosted = true
				s.reviewService.Update(reviewLog)
			}
		case "github":
			if err := s.postGitHubCommitComment(project, task.CommitSHA, comment); err != nil {
				log.Printf("[TaskQueue] Failed to post GitHub comment: %v", err)
			} else {
				reviewLog.CommentPosted = true
				s.reviewService.Update(reviewLog)
			}
		}
	}

	// Set commit status
	minScore := s.getEffectiveMinScore(project)
	statusState := "success"
	statusDesc := fmt.Sprintf("AI Review Passed: %.0f/%.0f", result.Score, minScore)
	if result.Score < minScore {
		statusState = "failed"
		statusDesc = fmt.Sprintf("AI Review Failed: %.0f (Min: %.0f)", result.Score, minScore)
	}
	s.setCommitStatus(project, task.CommitSHA, statusState, statusDesc, task.GitLabProjectID)

	return nil
}
