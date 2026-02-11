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

// HandleBitbucketWebhook processes Bitbucket webhook events
func (s *Service) HandleBitbucketWebhook(ctx context.Context, projectID uint, eventType string, body []byte) error {
	project, err := s.projectService.GetByID(projectID)
	if err != nil {
		return fmt.Errorf("project not found: %w", err)
	}

	if !project.AIEnabled {
		return nil
	}

	switch eventType {
	case "repo:push":
		if !strings.Contains(project.ReviewEvents, "push") {
			return nil
		}
		var event BitbucketPushEvent
		if err := json.Unmarshal(body, &event); err != nil {
			return err
		}
		return s.processBitbucketPush(ctx, project, &event)

	case "pullrequest:created", "pullrequest:updated":
		if !strings.Contains(project.ReviewEvents, "merge_request") {
			return nil
		}
		var event BitbucketPREvent
		if err := json.Unmarshal(body, &event); err != nil {
			return err
		}
		return s.processBitbucketPR(ctx, project, &event)
	}

	return nil
}

func (s *Service) processBitbucketPush(ctx context.Context, project *models.Project, event *BitbucketPushEvent) error {
	if len(event.Push.Changes) == 0 {
		return nil
	}

	for _, change := range event.Push.Changes {
		if change.New.Type != "branch" || len(change.Commits) == 0 {
			continue
		}

		branch := change.New.Name
		if s.isBranchIgnored(branch, project.BranchFilter) {
			continue
		}

		commitSHA := change.New.Target.Hash
		if s.isCommitAlreadyReviewed(project.ID, commitSHA) {
			continue
		}

		s.setBitbucketCommitStatus(project, commitSHA, "INPROGRESS", "AI Review in progress...")

		var commits []string
		for _, c := range change.Commits {
			commits = append(commits, fmt.Sprintf("%s: %s", c.Hash[:8], c.Message))
		}

		var diff string

		beforeSHA := change.Old.Target.Hash
		if !isNullSHA(beforeSHA) && beforeSHA != "" {
			compareDiff, err := s.getBitbucketCompareDiff(project, beforeSHA, commitSHA)
			if err != nil {
				logger.Infof("[Webhook] Bitbucket compare API failed, falling back to per-commit diffs: %v", err)
			} else if compareDiff != "" {
				diff = compareDiff
				logger.Infof("[Webhook] Got Bitbucket compare diff (before=%s, after=%s), length: %d bytes",
					beforeSHA[:8], commitSHA[:8], len(diff))
			}
		}

		if diff == "" {
			var allDiffs strings.Builder
			for _, c := range change.Commits {
				d, _ := s.getBitbucketDiff(project, c.Hash)
				allDiffs.WriteString(fmt.Sprintf("\n### Commit: %s\n%s\n", c.Hash[:8], d))
			}
			diff = allDiffs.String()
		}

		additions, deletions, filesChanged := ParseDiffStats(diff)

		reviewLog := &models.ReviewLog{
			ProjectID:     project.ID,
			EventType:     "push",
			CommitHash:    commitSHA,
			CommitURL:     change.New.Target.Links.HTML.Href,
			Branch:        branch,
			Author:        event.Actor.DisplayName,
			AuthorAvatar:  event.Actor.Links.Avatar.Href,
			AuthorURL:     event.Actor.Links.HTML.Href,
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
			CommitSHA:     commitSHA,
			EventType:     "push",
			Branch:        branch,
			Author:        event.Actor.DisplayName,
			AuthorAvatar:  event.Actor.Links.Avatar.Href,
			CommitMessage: strings.Join(commits, "\n"),
			Diff:          diff,
			CommitURL:     change.New.Target.Links.HTML.Href,
		}

		if err := services.GetTaskQueue().Enqueue(task); err != nil {
			logger.Infof("[Webhook] Failed to enqueue Bitbucket push review task: %v", err)
			reviewLog.ReviewStatus = "failed"
			reviewLog.ErrorMessage = "Failed to enqueue: " + err.Error()
			s.reviewService.Update(reviewLog)
			continue
		}

		logger.Infof("[Webhook] Bitbucket push review task enqueued for project %d, commit %s", project.ID, commitSHA[:8])
	}

	return nil
}

func (s *Service) processBitbucketPR(ctx context.Context, project *models.Project, event *BitbucketPREvent) error {
	branch := event.PullRequest.Source.Branch.Name
	if s.isBranchIgnored(branch, project.BranchFilter) {
		return nil
	}

	prNumber := event.PullRequest.ID
	commitSHA := event.PullRequest.Source.Commit.Hash

	s.setBitbucketCommitStatus(project, commitSHA, "INPROGRESS", "AI Review in progress...")

	diff, _ := s.getBitbucketPRDiff(project, prNumber)
	additions, deletions, filesChanged := ParseDiffStats(diff)

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

	// Enqueue review task for async processing
	task := &services.ReviewTask{
		ReviewLogID:   reviewLog.ID,
		ProjectID:     project.ID,
		CommitSHA:     commitSHA,
		EventType:     "merge_request",
		Branch:        branch,
		Author:        event.PullRequest.Author.DisplayName,
		AuthorAvatar:  event.PullRequest.Author.Links.Avatar.Href,
		CommitMessage: event.PullRequest.Title + "\n" + event.PullRequest.Description,
		Diff:          diff,
		MRNumber:      &prNumber,
		MRURL:         event.PullRequest.Links.HTML.Href,
	}

	if err := services.GetTaskQueue().Enqueue(task); err != nil {
		logger.Infof("[Webhook] Failed to enqueue Bitbucket PR review task: %v", err)
		reviewLog.ReviewStatus = "failed"
		reviewLog.ErrorMessage = "Failed to enqueue: " + err.Error()
		s.reviewService.Update(reviewLog)
		return err
	}

	logger.Infof("[Webhook] Bitbucket PR review task enqueued for project %d, PR #%d", project.ID, prNumber)
	return nil
}

func (s *Service) getBitbucketDiff(project *models.Project, commitSHA string) (string, error) {
	info, _ := parseRepoInfo(project.URL)
	apiURL := fmt.Sprintf("https://api.bitbucket.org/2.0/repositories/%s/diff/%s", info.projectPath, commitSHA)
	req, _ := http.NewRequest("GET", apiURL, nil)
	if project.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+project.AccessToken)
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return string(body), nil
}

func (s *Service) getBitbucketCompareDiff(project *models.Project, from, to string) (string, error) {
	info, err := parseRepoInfo(project.URL)
	if err != nil {
		return "", err
	}

	apiURL := fmt.Sprintf("https://api.bitbucket.org/2.0/repositories/%s/diff/%s..%s", info.projectPath, from, to)
	logger.Infof("[Webhook] Fetching Bitbucket compare diff: %s...%s", from[:8], to[:8])

	req, _ := http.NewRequest("GET", apiURL, nil)
	if project.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+project.AccessToken)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Bitbucket compare API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, _ := io.ReadAll(resp.Body)
	return string(body), nil
}

func (s *Service) getBitbucketPRDiff(project *models.Project, prNumber int) (string, error) {
	info, _ := parseRepoInfo(project.URL)
	apiURL := fmt.Sprintf("https://api.bitbucket.org/2.0/repositories/%s/pullrequests/%d/diff", info.projectPath, prNumber)
	req, _ := http.NewRequest("GET", apiURL, nil)
	if project.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+project.AccessToken)
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return string(body), nil
}

func (s *Service) setBitbucketCommitStatus(project *models.Project, sha, state, description string) {
	info, _ := parseRepoInfo(project.URL)
	apiURL := fmt.Sprintf("https://api.bitbucket.org/2.0/repositories/%s/commit/%s/statuses/build", info.projectPath, sha)
	data := map[string]string{"state": state, "key": "codesentry-ai-review", "name": "CodeSentry AI Review", "description": description}
	payload, _ := json.Marshal(data)
	req, _ := http.NewRequest("POST", apiURL, bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	if project.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+project.AccessToken)
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		logger.Infof("[Webhook] Failed to send Bitbucket commit status: %v", err)
		return
	}
	defer resp.Body.Close()
}

func (s *Service) postBitbucketCommitComment(project *models.Project, commitSHA, comment string) error {
	info, _ := parseRepoInfo(project.URL)
	apiURL := fmt.Sprintf("https://api.bitbucket.org/2.0/repositories/%s/commit/%s/comments", info.projectPath, commitSHA)
	data := map[string]interface{}{"content": map[string]string{"raw": comment}}
	payload, _ := json.Marshal(data)
	req, _ := http.NewRequest("POST", apiURL, bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	if project.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+project.AccessToken)
	}
	s.httpClient.Do(req)
	return nil
}

func (s *Service) postBitbucketPRComment(project *models.Project, prNumber int, comment string) error {
	info, _ := parseRepoInfo(project.URL)
	apiURL := fmt.Sprintf("https://api.bitbucket.org/2.0/repositories/%s/pullrequests/%d/comments", info.projectPath, prNumber)
	data := map[string]interface{}{"content": map[string]string{"raw": comment}}
	payload, _ := json.Marshal(data)
	req, _ := http.NewRequest("POST", apiURL, bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	if project.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+project.AccessToken)
	}
	s.httpClient.Do(req)
	return nil
}
