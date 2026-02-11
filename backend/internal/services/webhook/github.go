package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/huangang/codesentry/backend/pkg/logger"

	"github.com/huangang/codesentry/backend/internal/models"
	"github.com/huangang/codesentry/backend/internal/services"
)

// HandleGitHubWebhook processes GitHub webhook events
func (s *Service) HandleGitHubWebhook(ctx context.Context, projectID uint, eventType string, body []byte) error {
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

func (s *Service) processGitHubPush(ctx context.Context, project *models.Project, event *GitHubPushEvent) error {
	if len(event.Commits) == 0 {
		return nil
	}

	branch := strings.TrimPrefix(event.Ref, "refs/heads/")
	if s.isBranchIgnored(branch, project.BranchFilter) {
		return nil
	}

	if s.isCommitAlreadyReviewed(project.ID, event.After) {
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

	var diff string

	if !isNullSHA(event.Before) && event.Before != "" {
		compareDiff, err := s.getGitHubCompareDiff(project, event.Before, event.After)
		if err != nil {
			logger.Infof("[Webhook] GitHub compare API failed, falling back to single commit diff: %v", err)
		} else if compareDiff != "" {
			diff = compareDiff
			logger.Infof("[Webhook] Got GitHub compare diff (before=%s, after=%s), length: %d bytes",
				event.Before[:8], event.After[:8], len(diff))
		}
	}

	if diff == "" {
		d, err := s.getGitHubDiff(project, event.After)
		if err != nil {
			diff = "Failed to get diff: " + err.Error()
		} else {
			diff = d
		}
	}

	additions, deletions, filesChanged := ParseDiffStats(diff)

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

	// Enqueue review task for async processing
	task := &services.ReviewTask{
		ReviewLogID:   reviewLog.ID,
		ProjectID:     project.ID,
		CommitSHA:     event.After,
		EventType:     "push",
		Branch:        branch,
		Author:        event.Sender.Login,
		AuthorEmail:   event.Pusher.Email,
		AuthorAvatar:  event.Sender.AvatarURL,
		CommitMessage: strings.Join(commits, "\n"),
		Diff:          diff,
		CommitURL:     commitURL,
	}

	if err := services.GetTaskQueue().Enqueue(task); err != nil {
		logger.Infof("[Webhook] Failed to enqueue GitHub push review task: %v", err)
		reviewLog.ReviewStatus = "failed"
		reviewLog.ErrorMessage = "Failed to enqueue: " + err.Error()
		s.reviewService.Update(reviewLog)
		return err
	}

	logger.Infof("[Webhook] GitHub push review task enqueued for project %d, commit %s", project.ID, event.After[:8])
	return nil
}

func (s *Service) processGitHubPR(ctx context.Context, project *models.Project, event *GitHubPREvent) error {
	if event.Action != "opened" && event.Action != "synchronize" {
		return nil
	}

	if s.isBranchIgnored(event.PullRequest.Head.Ref, project.BranchFilter) {
		return nil
	}

	mrNumber := event.Number

	diff, err := s.getGitHubPRDiff(project, mrNumber)
	if err != nil {
		diff = "Failed to get diff: " + err.Error()
	}

	additions, deletions, filesChanged := ParseDiffStats(diff)

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

	// Enqueue review task for async processing
	task := &services.ReviewTask{
		ReviewLogID:   reviewLog.ID,
		ProjectID:     project.ID,
		CommitSHA:     event.PullRequest.Head.SHA,
		EventType:     "merge_request",
		Branch:        event.PullRequest.Head.Ref,
		Author:        event.PullRequest.User.Login,
		AuthorAvatar:  event.PullRequest.User.AvatarURL,
		CommitMessage: event.PullRequest.Title + "\n" + event.PullRequest.Body,
		Diff:          diff,
		MRNumber:      &mrNumber,
		MRURL:         event.PullRequest.HTMLURL,
	}

	if err := services.GetTaskQueue().Enqueue(task); err != nil {
		logger.Infof("[Webhook] Failed to enqueue GitHub PR review task: %v", err)
		reviewLog.ReviewStatus = "failed"
		reviewLog.ErrorMessage = "Failed to enqueue: " + err.Error()
		s.reviewService.Update(reviewLog)
		return err
	}

	logger.Infof("[Webhook] GitHub PR review task enqueued for project %d, PR #%d", project.ID, mrNumber)
	return nil
}

func (s *Service) getGitHubDiff(project *models.Project, commitSHA string) (string, error) {
	info, err := parseRepoInfo(project.URL)
	if err != nil {
		return "", err
	}
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits/%s", info.owner, info.repo, commitSHA)
	return s.fetchGitHubDiff(apiURL, project.AccessToken)
}

func (s *Service) getGitHubCompareDiff(project *models.Project, before, after string) (string, error) {
	info, err := parseRepoInfo(project.URL)
	if err != nil {
		return "", err
	}

	baseURL := "https://api.github.com"
	if info.baseURL != "https://github.com" {
		baseURL = info.baseURL + "/api/v3"
	}

	apiURL := fmt.Sprintf("%s/repos/%s/%s/compare/%s...%s", baseURL, info.owner, info.repo, before, after)
	logger.Infof("[Webhook] Fetching GitHub compare diff: %s...%s", before[:8], after[:8])
	return s.fetchGitHubDiff(apiURL, project.AccessToken)
}

func (s *Service) getGitHubPRDiff(project *models.Project, prNumber int) (string, error) {
	info, err := parseRepoInfo(project.URL)
	if err != nil {
		return "", err
	}
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls/%d", info.owner, info.repo, prNumber)
	return s.fetchGitHubDiff(apiURL, project.AccessToken)
}

func (s *Service) fetchGitHubDiff(apiURL, accessToken string) (string, error) {
	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Set("Accept", "application/vnd.github.v3.diff")
	if accessToken != "" {
		req.Header.Set("Authorization", "token "+accessToken)
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return string(body), nil
}

func (s *Service) setGitHubCommitStatus(project *models.Project, sha, state, description string) {
	info, _ := parseRepoInfo(project.URL)
	githubState := state
	if state == "failed" {
		githubState = "failure"
	}

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/statuses/%s", info.owner, info.repo, sha)
	data := map[string]string{"state": githubState, "context": "codesentry/ai-review", "description": description}
	payload, _ := json.Marshal(data)

	req, _ := http.NewRequest("POST", apiURL, bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	if project.AccessToken != "" {
		req.Header.Set("Authorization", "token "+project.AccessToken)
	}
	s.httpClient.Do(req)
}

func (s *Service) postGitHubPRComment(project *models.Project, prNumber int, comment string) error {
	info, _ := parseRepoInfo(project.URL)
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues/%d/comments", info.owner, info.repo, prNumber)
	body := fmt.Sprintf(`{"body": %q}`, comment)
	req, _ := http.NewRequest("POST", apiURL, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if project.AccessToken != "" {
		req.Header.Set("Authorization", "token "+project.AccessToken)
	}
	s.httpClient.Do(req)
	return nil
}

func (s *Service) postGitHubCommitComment(project *models.Project, commitSHA, comment string) error {
	info, _ := parseRepoInfo(project.URL)
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits/%s/comments", info.owner, info.repo, commitSHA)
	body := fmt.Sprintf(`{"body": %q}`, comment)
	req, _ := http.NewRequest("POST", apiURL, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if project.AccessToken != "" {
		req.Header.Set("Authorization", "token "+project.AccessToken)
	}
	s.httpClient.Do(req)
	return nil
}
