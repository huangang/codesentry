package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/huangang/codesentry/backend/internal/models"
	"github.com/huangang/codesentry/backend/internal/services"
)

// HandleGitLabWebhook processes GitLab webhook events
func (s *Service) HandleGitLabWebhook(ctx context.Context, projectID uint, eventType string, body []byte) error {
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
		log.Printf("[Webhook] Unknown GitLab event type: %s, skipping", eventType)
	}

	return nil
}

func (s *Service) processGitLabPush(ctx context.Context, project *models.Project, event *GitLabPushEvent) error {
	if len(event.Commits) == 0 {
		return nil
	}

	branch := strings.TrimPrefix(event.Ref, "refs/heads/")
	if s.isBranchIgnored(branch, project.BranchFilter) {
		log.Printf("[Webhook] Branch %s is in ignore list, skipping review", branch)
		return nil
	}

	commitSHA := event.CheckoutSHA
	if commitSHA == "" && len(event.Commits) > 0 {
		commitSHA = event.Commits[len(event.Commits)-1].ID
	}

	if s.isCommitAlreadyReviewed(project.ID, commitSHA) {
		log.Printf("[Webhook] Commit %s already reviewed, skipping", commitSHA[:8])
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

	log.Printf("[Webhook] Processing GitLab push: %d commits, branch=%s, commit=%s",
		len(event.Commits), branch, commitSHA[:8])

	services.LogInfo("Webhook", "GitLabPush", fmt.Sprintf("Processing push from %s: %d commits", event.UserName, len(event.Commits)), nil, "", "", map[string]interface{}{
		"project_id": project.ID,
		"branch":     branch,
		"commit":     commitSHA,
	})

	s.setGitLabCommitStatus(project, commitSHA, "pending", "AI Review in progress...", event.ProjectID)

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

	additions, deletions, filesChanged := ParseDiffStats(diff)

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

	log.Printf("[Webhook] Starting AI review for project %d, commit %s", project.ID, commitSHA[:8])

	filteredDiff := s.filterDiff(diff, project.FileExtensions, project.IgnorePatterns)
	if filteredDiff != diff {
		log.Printf("[Webhook] Filtered diff by extensions (%s) and ignore patterns (%s): %d -> %d bytes",
			project.FileExtensions, project.IgnorePatterns, len(diff), len(filteredDiff))
	}

	if IsEmptyDiff(filteredDiff) {
		log.Printf("[Webhook] WARNING: Empty commit detected for project %d, commit %s - skipping AI review", project.ID, commitSHA[:8])
		services.LogWarning("Webhook", "EmptyCommit", fmt.Sprintf("Empty commit %s detected, skipping AI review", commitSHA[:8]), nil, "", "", map[string]interface{}{
			"project_id": project.ID,
			"commit":     commitSHA,
			"branch":     branch,
		})
		reviewLog.ReviewStatus = "skipped"
		reviewLog.ReviewResult = "Empty commit - no code changes to review"
		s.reviewService.Update(reviewLog)
		s.setGitLabCommitStatus(project, commitSHA, "success", "AI Review Skipped: Empty commit", event.ProjectID)
		return nil
	}

	var fileContext string
	if s.fileContextService.IsEnabled() {
		fileContext, _ = s.fileContextService.BuildFileContext(project, filteredDiff, commitSHA)
		if fileContext != "" {
			log.Printf("[Webhook] Built file context: %d chars", len(fileContext))
		}
	}

	result, err := s.aiService.ReviewChunked(ctx, &services.ReviewRequest{
		ProjectID:   project.ID,
		Diffs:       filteredDiff,
		Commits:     strings.Join(commits, "\n"),
		FileContext: fileContext,
	})

	if err != nil {
		log.Printf("[Webhook] AI review failed: %v", err)
		services.LogError("AIReview", "ReviewFailed", err.Error(), nil, "", "", map[string]interface{}{
			"project_id": project.ID,
			"commit":     commitSHA,
		})
		reviewLog.ReviewStatus = "failed"
		reviewLog.ErrorMessage = err.Error()
		s.setGitLabCommitStatus(project, commitSHA, "failed", "AI Review Failed", event.ProjectID)
	} else {
		log.Printf("[Webhook] AI review completed, score: %.1f, result length: %d", result.Score, len(result.Content))
		services.LogInfo("AIReview", "ReviewCompleted", fmt.Sprintf("Review completed with score %.0f", result.Score), nil, "", "", map[string]interface{}{
			"project_id": project.ID,
			"commit":     commitSHA,
			"score":      result.Score,
		})
		reviewLog.ReviewStatus = "completed"
		reviewLog.ReviewResult = result.Content
		reviewLog.Score = &result.Score

		s.notificationService.SendReviewNotification(project, &services.ReviewNotification{
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

		minScore := s.getEffectiveMinScore(project)
		if result.Score >= minScore {
			s.setGitLabCommitStatus(project, commitSHA, "success", fmt.Sprintf("AI Review Passed: %.0f/%.0f", result.Score, minScore), event.ProjectID)
		} else {
			s.setGitLabCommitStatus(project, commitSHA, "failed", fmt.Sprintf("AI Review Failed: %.0f (Min: %.0f)", result.Score, minScore), event.ProjectID)
		}
	}

	return s.reviewService.Update(reviewLog)
}

func (s *Service) processGitLabMR(ctx context.Context, project *models.Project, event *GitLabMREvent) error {
	if event.ObjectAttributes.Action != "open" && event.ObjectAttributes.Action != "update" {
		return nil
	}

	if s.isBranchIgnored(event.ObjectAttributes.SourceBranch, project.BranchFilter) {
		log.Printf("[Webhook] Branch %s is in ignore list, skipping review", event.ObjectAttributes.SourceBranch)
		return nil
	}

	mrIID := event.ObjectAttributes.IID
	commitSHA, err := s.getGitLabRequestSHA(project, mrIID)
	if err != nil {
		log.Printf("[Webhook] Failed to get MR commit SHA: %v", err)
		return err
	}

	s.setGitLabCommitStatus(project, commitSHA, "pending", "AI Review in progress...", event.Project.ID)

	diff, err := s.getGitLabMRDiff(project, mrIID)
	if err != nil {
		diff = "Failed to get diff: " + err.Error()
	}

	additions, deletions, filesChanged := ParseDiffStats(diff)

	reviewLog := &models.ReviewLog{
		ProjectID:     project.ID,
		EventType:     "merge_request",
		CommitHash:    commitSHA,
		Branch:        event.ObjectAttributes.SourceBranch,
		Author:        event.User.Username,
		AuthorEmail:   event.User.Email,
		AuthorAvatar:  event.User.AvatarURL,
		CommitMessage: event.ObjectAttributes.Title,
		FilesChanged:  filesChanged,
		Additions:     additions,
		Deletions:     deletions,
		MRNumber:      &mrIID,
		MRURL:         event.ObjectAttributes.URL,
		ReviewStatus:  "pending",
	}
	s.reviewService.Create(reviewLog)

	filteredDiff := s.filterDiff(diff, project.FileExtensions, project.IgnorePatterns)

	if IsEmptyDiff(filteredDiff) {
		log.Printf("[Webhook] WARNING: Empty MR detected for project %d, MR #%d - skipping AI review", project.ID, mrIID)
		services.LogWarning("Webhook", "EmptyMR", fmt.Sprintf("Empty MR #%d detected, skipping AI review", mrIID), nil, "", "", map[string]interface{}{
			"project_id": project.ID,
			"mr_iid":     mrIID,
			"branch":     event.ObjectAttributes.SourceBranch,
		})
		reviewLog.ReviewStatus = "skipped"
		reviewLog.ReviewResult = "Empty merge request - no code changes to review"
		s.reviewService.Update(reviewLog)
		s.setGitLabCommitStatus(project, commitSHA, "success", "AI Review Skipped: Empty MR", event.Project.ID)
		return nil
	}

	var fileContext string
	if s.fileContextService.IsEnabled() {
		fileContext, _ = s.fileContextService.BuildFileContext(project, filteredDiff, commitSHA)
		if fileContext != "" {
			log.Printf("[Webhook] Built file context for MR: %d chars", len(fileContext))
		}
	}

	result, err := s.aiService.ReviewChunked(ctx, &services.ReviewRequest{
		ProjectID:   project.ID,
		Diffs:       filteredDiff,
		Commits:     event.ObjectAttributes.Title + "\n" + event.ObjectAttributes.Description,
		FileContext: fileContext,
	})

	if err != nil {
		reviewLog.ReviewStatus = "failed"
		reviewLog.ErrorMessage = err.Error()
		s.setGitLabCommitStatus(project, commitSHA, "failed", "AI Review Failed", event.Project.ID)
	} else {
		reviewLog.ReviewStatus = "completed"
		reviewLog.ReviewResult = result.Content
		reviewLog.Score = &result.Score

		s.notificationService.SendReviewNotification(project, &services.ReviewNotification{
			ProjectName:   project.Name,
			Branch:        event.ObjectAttributes.SourceBranch,
			Author:        event.User.Username,
			CommitMessage: event.ObjectAttributes.Title,
			Score:         result.Score,
			ReviewResult:  result.Content,
			EventType:     "merge_request",
			MRURL:         event.ObjectAttributes.URL,
		})

		if project.CommentEnabled {
			comment := s.formatReviewComment(result.Score, result.Content)
			if err := s.postGitLabMRComment(project, mrIID, comment); err != nil {
				log.Printf("[Webhook] Failed to post GitLab MR comment: %v", err)
			} else {
				reviewLog.CommentPosted = true
			}
		}

		minScore := s.getEffectiveMinScore(project)
		if result.Score >= minScore {
			s.setGitLabCommitStatus(project, commitSHA, "success", fmt.Sprintf("AI Review Passed: %.0f/%.0f", result.Score, minScore), event.Project.ID)
		} else {
			s.setGitLabCommitStatus(project, commitSHA, "failed", fmt.Sprintf("AI Review Failed: %.0f (Min: %.0f)", result.Score, minScore), event.Project.ID)
		}
	}

	return s.reviewService.Update(reviewLog)
}

func (s *Service) getGitLabDiff(project *models.Project, commitSHA string) (string, error) {
	info, err := parseRepoInfo(project.URL)
	if err != nil {
		return "", err
	}

	apiURL := fmt.Sprintf("%s/api/v4/projects/%s/repository/commits/%s/diff",
		info.baseURL, strings.ReplaceAll(info.projectPath, "/", "%2F"), commitSHA)

	return s.fetchDiff(apiURL, project.AccessToken, "PRIVATE-TOKEN")
}

func (s *Service) getGitLabMRDiff(project *models.Project, mrIID int) (string, error) {
	info, err := parseRepoInfo(project.URL)
	if err != nil {
		return "", err
	}

	apiURL := fmt.Sprintf("%s/api/v4/projects/%s/merge_requests/%d/diffs",
		info.baseURL, strings.ReplaceAll(info.projectPath, "/", "%2F"), mrIID)

	return s.fetchDiff(apiURL, project.AccessToken, "PRIVATE-TOKEN")
}

func (s *Service) getGitLabRequestSHA(project *models.Project, mrIID int) (string, error) {
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

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if sha, ok := result["sha"].(string); ok {
		return sha, nil
	}
	return "", fmt.Errorf("sha not found")
}

func (s *Service) setGitLabCommitStatus(project *models.Project, sha string, state string, description string, gitlabProjectID int) {
	info, err := parseRepoInfo(project.URL)
	if err != nil {
		log.Printf("[Webhook] Failed to parse repo info for GitLab status update: %v", err)
		return
	}

	projectIdentifier := strings.ReplaceAll(info.projectPath, "/", "%2F")
	if gitlabProjectID > 0 {
		projectIdentifier = fmt.Sprintf("%d", gitlabProjectID)
	}

	apiURL := fmt.Sprintf("%s/api/v4/projects/%s/statuses/%s",
		info.baseURL, projectIdentifier, sha)

	data := map[string]string{
		"state":       state,
		"context":     "codesentry/ai-review",
		"description": description,
	}

	payload, _ := json.Marshal(data)
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(payload))
	if err != nil {
		log.Printf("[Webhook] Failed to create GitLab status request: %v", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	if project.AccessToken != "" {
		req.Header.Set("PRIVATE-TOKEN", project.AccessToken)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		log.Printf("[Webhook] Failed to send GitLab commit status: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("[Webhook] Failed to set GitLab commit status (code %d): %s", resp.StatusCode, string(body))
	} else {
		log.Printf("[Webhook] Set GitLab commit status for %s to %s", sha[:8], state)
	}
}

func (s *Service) postGitLabMRComment(project *models.Project, mrIID int, comment string) error {
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

func (s *Service) postGitLabCommitComment(project *models.Project, commitSHA string, comment string) error {
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
