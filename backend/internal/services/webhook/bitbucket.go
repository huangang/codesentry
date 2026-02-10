package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"github.com/huangang/codesentry/backend/pkg/logger"
	"net/http"
	"strings"

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
		var allDiffs strings.Builder
		for _, c := range change.Commits {
			commits = append(commits, fmt.Sprintf("%s: %s", c.Hash[:8], c.Message))
			diff, _ := s.getBitbucketDiff(project, c.Hash)
			allDiffs.WriteString(fmt.Sprintf("\n### Commit: %s\n%s\n", c.Hash[:8], diff))
		}

		diff := allDiffs.String()
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

		filteredDiff := s.filterDiff(diff, project.FileExtensions, project.IgnorePatterns)

		if IsEmptyDiff(filteredDiff) {
			reviewLog.ReviewStatus = "skipped"
			reviewLog.ReviewResult = "Empty commit - no code changes to review"
			s.reviewService.Update(reviewLog)
			s.setBitbucketCommitStatus(project, commitSHA, "SUCCESSFUL", "AI Review Skipped")
			continue
		}

		var fileContext string
		if s.fileContextService.IsEnabled() {
			fileContext, _ = s.fileContextService.BuildFileContext(project, filteredDiff, commitSHA)
		}

		result, err := s.aiService.ReviewChunked(ctx, &services.ReviewRequest{
			ProjectID:   project.ID,
			Diffs:       filteredDiff,
			Commits:     strings.Join(commits, "\n"),
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

			s.notificationService.SendReviewNotification(project, &services.ReviewNotification{
				ProjectName:   project.Name,
				Branch:        branch,
				Author:        event.Actor.DisplayName,
				CommitMessage: strings.Join(commits, "\n"),
				Score:         result.Score,
				ReviewResult:  result.Content,
				EventType:     "push",
			})

			if project.CommentEnabled {
				s.postBitbucketCommitComment(project, commitSHA, s.formatReviewComment(result.Score, result.Content))
			}

			minScore := s.getEffectiveMinScore(project)
			if result.Score >= minScore {
				s.setBitbucketCommitStatus(project, commitSHA, "SUCCESSFUL", fmt.Sprintf("AI Review Passed: %.0f/%.0f", result.Score, minScore))
			} else {
				s.setBitbucketCommitStatus(project, commitSHA, "FAILED", fmt.Sprintf("AI Review Failed: %.0f (Min: %.0f)", result.Score, minScore))
			}
		}

		s.reviewService.Update(reviewLog)
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

	filteredDiff := s.filterDiff(diff, project.FileExtensions, project.IgnorePatterns)

	if IsEmptyDiff(filteredDiff) {
		reviewLog.ReviewStatus = "skipped"
		reviewLog.ReviewResult = "Empty pull request - no code changes to review"
		s.reviewService.Update(reviewLog)
		s.setBitbucketCommitStatus(project, commitSHA, "SUCCESSFUL", "AI Review Skipped")
		return nil
	}

	var fileContext string
	if s.fileContextService.IsEnabled() {
		fileContext, _ = s.fileContextService.BuildFileContext(project, filteredDiff, commitSHA)
	}

	result, err := s.aiService.ReviewChunked(ctx, &services.ReviewRequest{
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

		s.notificationService.SendReviewNotification(project, &services.ReviewNotification{
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
			s.postBitbucketPRComment(project, prNumber, s.formatReviewComment(result.Score, result.Content))
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
