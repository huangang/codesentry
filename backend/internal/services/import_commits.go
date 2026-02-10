package services

import (
	"encoding/json"
	"fmt"
	"io"
	"github.com/huangang/codesentry/backend/pkg/logger"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/huangang/codesentry/backend/internal/models"
	"gorm.io/gorm"
)

type ImportCommitsService struct {
	db         *gorm.DB
	httpClient *http.Client
}

func NewImportCommitsService(db *gorm.DB) *ImportCommitsService {
	return &ImportCommitsService{
		db:         db,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

type ImportCommitsRequest struct {
	ProjectID uint   `json:"project_id" binding:"required"`
	StartDate string `json:"start_date" binding:"required"` // Format: 2006-01-02
	EndDate   string `json:"end_date" binding:"required"`   // Format: 2006-01-02
}

type ImportCommitsResponse struct {
	Imported int      `json:"imported"`
	Skipped  int      `json:"skipped"`
	Errors   []string `json:"errors,omitempty"`
	Async    bool     `json:"async,omitempty"`
	Message  string   `json:"message,omitempty"`
}

// GitLab commit structure
type gitLabCommit struct {
	ID            string    `json:"id"`
	ShortID       string    `json:"short_id"`
	Title         string    `json:"title"`
	Message       string    `json:"message"`
	AuthorName    string    `json:"author_name"`
	AuthorEmail   string    `json:"author_email"`
	CommittedDate time.Time `json:"committed_date"`
	WebURL        string    `json:"web_url"`
	Stats         *struct {
		Additions int `json:"additions"`
		Deletions int `json:"deletions"`
		Total     int `json:"total"`
	} `json:"stats"`
}

// GitHub commit structure
type gitHubCommit struct {
	SHA    string `json:"sha"`
	Commit struct {
		Message string `json:"message"`
		Author  struct {
			Name  string    `json:"name"`
			Email string    `json:"email"`
			Date  time.Time `json:"date"`
		} `json:"author"`
	} `json:"commit"`
	HTMLURL string `json:"html_url"`
	Stats   *struct {
		Additions int `json:"additions"`
		Deletions int `json:"deletions"`
		Total     int `json:"total"`
	} `json:"stats"`
	Files []struct {
		Filename string `json:"filename"`
	} `json:"files"`
}

// Bitbucket commit structure
type bitbucketCommitResponse struct {
	Values []struct {
		Hash    string `json:"hash"`
		Message string `json:"message"`
		Author  struct {
			Raw  string `json:"raw"`
			User struct {
				DisplayName string `json:"display_name"`
			} `json:"user"`
		} `json:"author"`
		Date  time.Time `json:"date"`
		Links struct {
			HTML struct {
				Href string `json:"href"`
			} `json:"html"`
		} `json:"links"`
	} `json:"values"`
	Next string `json:"next"`
}

func (s *ImportCommitsService) ImportCommits(req *ImportCommitsRequest) (*ImportCommitsResponse, error) {
	var project models.Project
	if err := s.db.First(&project, req.ProjectID).Error; err != nil {
		return nil, fmt.Errorf("project not found: %w", err)
	}

	if project.AccessToken == "" {
		return nil, fmt.Errorf("project does not have an access token configured")
	}

	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		return nil, fmt.Errorf("invalid start_date format: %w", err)
	}

	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		return nil, fmt.Errorf("invalid end_date format: %w", err)
	}

	// Add one day to end date to include the entire day
	endDate = endDate.Add(24*time.Hour - time.Second)

	logger.Infof("[ImportCommits] Starting async import for project %d (%s) from %s to %s",
		project.ID, project.Name, req.StartDate, req.EndDate)

	go s.importCommitsAsync(&project, startDate, endDate)

	return &ImportCommitsResponse{
		Async:   true,
		Message: "Import started, you will be notified when complete",
	}, nil
}

func (s *ImportCommitsService) importCommitsAsync(project *models.Project, startDate, endDate time.Time) {
	var response *ImportCommitsResponse
	var err error

	switch project.Platform {
	case "gitlab":
		response, err = s.importGitLabCommits(project, startDate, endDate)
	case "github":
		response, err = s.importGitHubCommits(project, startDate, endDate)
	case "bitbucket":
		response, err = s.importBitbucketCommits(project, startDate, endDate)
	default:
		err = fmt.Errorf("unsupported platform: %s", project.Platform)
	}

	if err != nil {
		logger.Infof("[ImportCommits] Async import failed for project %d: %v", project.ID, err)
		PublishImportEvent(project.ID, project.Name, 0, 0, err.Error())
	} else {
		logger.Infof("[ImportCommits] Async import complete for project %d: imported=%d, skipped=%d",
			project.ID, response.Imported, response.Skipped)
		PublishImportEvent(project.ID, project.Name, response.Imported, response.Skipped, "")
	}
}

func (s *ImportCommitsService) importGitLabCommits(project *models.Project, startDate, endDate time.Time) (*ImportCommitsResponse, error) {
	info, err := parseRepoInfo(project.URL)
	if err != nil {
		return nil, err
	}

	encodedPath := url.PathEscape(info.projectPath)
	apiURL := fmt.Sprintf("%s/api/v4/projects/%s/repository/commits?since=%s&until=%s&with_stats=true&per_page=100",
		info.baseURL, encodedPath, startDate.Format(time.RFC3339), endDate.Format(time.RFC3339))

	response := &ImportCommitsResponse{}
	page := 1

	for {
		pageURL := fmt.Sprintf("%s&page=%d", apiURL, page)
		req, err := http.NewRequest("GET", pageURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("PRIVATE-TOKEN", project.AccessToken)

		resp, err := s.httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("GitLab API returned %d: %s", resp.StatusCode, string(body))
		}

		var commits []gitLabCommit
		if err := json.NewDecoder(resp.Body).Decode(&commits); err != nil {
			return nil, err
		}

		if len(commits) == 0 {
			break
		}

		for _, commit := range commits {
			if s.isCommitExists(project.ID, commit.ID) {
				response.Skipped++
				continue
			}

			additions := 0
			deletions := 0
			if commit.Stats != nil {
				additions = commit.Stats.Additions
				deletions = commit.Stats.Deletions
			}

			reviewLog := &models.ReviewLog{
				ProjectID:     project.ID,
				EventType:     "push",
				CommitHash:    commit.ID,
				CommitURL:     commit.WebURL,
				Author:        commit.AuthorName,
				AuthorEmail:   commit.AuthorEmail,
				CommitMessage: commit.Message,
				Additions:     additions,
				Deletions:     deletions,
				ReviewStatus:  "manual",
				IsManual:      true,
				CreatedAt:     commit.CommittedDate,
			}

			if err := s.db.Create(reviewLog).Error; err != nil {
				response.Errors = append(response.Errors, fmt.Sprintf("Failed to create record for %s: %v", commit.ShortID, err))
				continue
			}
			response.Imported++
		}

		page++
	}

	logger.Infof("[ImportCommits] GitLab import complete: imported=%d, skipped=%d", response.Imported, response.Skipped)
	return response, nil
}

func (s *ImportCommitsService) importGitHubCommits(project *models.Project, startDate, endDate time.Time) (*ImportCommitsResponse, error) {
	info, err := parseRepoInfo(project.URL)
	if err != nil {
		return nil, err
	}

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits?since=%s&until=%s&per_page=100",
		info.owner, info.repo, startDate.Format(time.RFC3339), endDate.Format(time.RFC3339))

	response := &ImportCommitsResponse{}
	page := 1

	for {
		pageURL := fmt.Sprintf("%s&page=%d", apiURL, page)
		req, err := http.NewRequest("GET", pageURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/vnd.github.v3+json")
		req.Header.Set("Authorization", "token "+project.AccessToken)

		resp, err := s.httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("GitHub API returned %d: %s", resp.StatusCode, string(body))
		}

		var commits []gitHubCommit
		if err := json.NewDecoder(resp.Body).Decode(&commits); err != nil {
			return nil, err
		}

		if len(commits) == 0 {
			break
		}

		for _, commit := range commits {
			if s.isCommitExists(project.ID, commit.SHA) {
				response.Skipped++
				continue
			}

			// Fetch commit details to get stats
			additions, deletions, filesChanged := s.fetchGitHubCommitStats(project, info, commit.SHA)

			reviewLog := &models.ReviewLog{
				ProjectID:     project.ID,
				EventType:     "push",
				CommitHash:    commit.SHA,
				CommitURL:     commit.HTMLURL,
				Author:        commit.Commit.Author.Name,
				AuthorEmail:   commit.Commit.Author.Email,
				CommitMessage: commit.Commit.Message,
				Additions:     additions,
				Deletions:     deletions,
				FilesChanged:  filesChanged,
				ReviewStatus:  "manual",
				IsManual:      true,
				CreatedAt:     commit.Commit.Author.Date,
			}

			if err := s.db.Create(reviewLog).Error; err != nil {
				response.Errors = append(response.Errors, fmt.Sprintf("Failed to create record for %s: %v", commit.SHA[:8], err))
				continue
			}
			response.Imported++
		}

		page++
	}

	logger.Infof("[ImportCommits] GitHub import complete: imported=%d, skipped=%d", response.Imported, response.Skipped)
	return response, nil
}

func (s *ImportCommitsService) fetchGitHubCommitStats(project *models.Project, info *repoInfo, sha string) (additions, deletions, filesChanged int) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits/%s", info.owner, info.repo, sha)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Authorization", "token "+project.AccessToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return
	}

	var commit gitHubCommit
	if err := json.NewDecoder(resp.Body).Decode(&commit); err != nil {
		return
	}

	if commit.Stats != nil {
		additions = commit.Stats.Additions
		deletions = commit.Stats.Deletions
	}
	filesChanged = len(commit.Files)
	return
}

func (s *ImportCommitsService) importBitbucketCommits(project *models.Project, startDate, endDate time.Time) (*ImportCommitsResponse, error) {
	info, err := parseRepoInfo(project.URL)
	if err != nil {
		return nil, err
	}

	// Bitbucket uses workspace/repo format
	apiURL := fmt.Sprintf("https://api.bitbucket.org/2.0/repositories/%s/commits?pagelen=50", info.projectPath)

	response := &ImportCommitsResponse{}
	nextURL := apiURL

	for nextURL != "" {
		req, err := http.NewRequest("GET", nextURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+project.AccessToken)

		resp, err := s.httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("Bitbucket API returned %d: %s", resp.StatusCode, string(body))
		}

		var commitsResp bitbucketCommitResponse
		if err := json.NewDecoder(resp.Body).Decode(&commitsResp); err != nil {
			return nil, err
		}

		for _, commit := range commitsResp.Values {
			// Check date range
			if commit.Date.Before(startDate) {
				// Bitbucket returns commits in reverse chronological order
				// If we hit a commit before start date, we can stop
				nextURL = ""
				break
			}
			if commit.Date.After(endDate) {
				continue
			}

			if s.isCommitExists(project.ID, commit.Hash) {
				response.Skipped++
				continue
			}

			// Parse author from raw format "Name <email>"
			authorName := commit.Author.Raw
			authorEmail := ""
			if idx := strings.Index(commit.Author.Raw, " <"); idx != -1 {
				authorName = commit.Author.Raw[:idx]
				if endIdx := strings.Index(commit.Author.Raw, ">"); endIdx != -1 {
					authorEmail = commit.Author.Raw[idx+2 : endIdx]
				}
			}
			if commit.Author.User.DisplayName != "" {
				authorName = commit.Author.User.DisplayName
			}

			reviewLog := &models.ReviewLog{
				ProjectID:     project.ID,
				EventType:     "push",
				CommitHash:    commit.Hash,
				CommitURL:     commit.Links.HTML.Href,
				Author:        authorName,
				AuthorEmail:   authorEmail,
				CommitMessage: commit.Message,
				ReviewStatus:  "manual",
				IsManual:      true,
				CreatedAt:     commit.Date,
			}

			if err := s.db.Create(reviewLog).Error; err != nil {
				response.Errors = append(response.Errors, fmt.Sprintf("Failed to create record for %s: %v", commit.Hash[:8], err))
				continue
			}
			response.Imported++
		}

		nextURL = commitsResp.Next
	}

	logger.Infof("[ImportCommits] Bitbucket import complete: imported=%d, skipped=%d", response.Imported, response.Skipped)
	return response, nil
}

func (s *ImportCommitsService) isCommitExists(projectID uint, commitHash string) bool {
	var count int64
	s.db.Model(&models.ReviewLog{}).Where("project_id = ? AND commit_hash = ?", projectID, commitHash).Count(&count)
	return count > 0
}
