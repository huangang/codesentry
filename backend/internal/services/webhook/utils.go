package webhook

import (
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

	"github.com/huangang/codesentry/backend/internal/models"
)

// DefaultIgnorePatterns - files that should be skipped by default (config, lock, generated files)
const DefaultIgnorePatterns = "*.json,*.yaml,*.yml,*.toml,*.xml,*.ini,*.env,*.config," +
	"*.lock,package-lock.json,yarn.lock,pnpm-lock.yaml,go.sum,Cargo.lock,composer.lock,Gemfile.lock,poetry.lock," +
	"*.min.js,*.min.css,*.bundle.js,*.bundle.css," +
	"dist/,build/,out/,target/,.next/," +
	"vendor/,node_modules/,__pycache__/,.venv/,venv/"

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

// IsEmptyDiff checks if the diff content has no actual code changes
func IsEmptyDiff(diff string) bool {
	if diff == "" {
		return true
	}
	lines := strings.Split(diff, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "### Commit:") {
			continue
		}
		return false
	}
	return true
}

func (s *Service) isBranchIgnored(branch string, branchFilter string) bool {
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

func (s *Service) filterDiff(diff string, extensions string, ignorePatterns string) string {
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

func (s *Service) shouldIncludeFile(filePath string, extMap map[string]bool, ignoreList []string) bool {
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

func (s *Service) matchIgnorePattern(filePath, pattern string) bool {
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

func (s *Service) fetchDiff(apiURL, token, tokenHeader string) (string, error) {
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

func (s *Service) formatReviewComment(score float64, reviewResult string) string {
	return fmt.Sprintf("## 🤖 AI Code Review\n\n**Score: %.0f/100**\n\n%s\n\n---\n*Powered by CodeSentry*", score, reviewResult)
}

// ParseDiffStats parses diff content and returns additions, deletions, and files changed
func ParseDiffStats(diff string) (additions, deletions, filesChanged int) {
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

func (s *Service) isCommitAlreadyReviewed(projectID uint, commitSHA string) bool {
	var count int64
	s.db.Model(&models.ReviewLog{}).
		Where("project_id = ? AND commit_hash = ? AND review_status = ?", projectID, commitSHA, "completed").
		Count(&count)
	return count > 0
}

func (s *Service) getEffectiveMinScore(project *models.Project) float64 {
	if project.MinScore > 0 {
		return project.MinScore
	}
	globalMinScore := s.configService.GetWithDefault("system.min_score", "60")
	var minScore float64
	fmt.Sscanf(globalMinScore, "%f", &minScore)
	if minScore > 0 {
		return minScore
	}
	return 60.0
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
	return hmac.Equal([]byte(strings.TrimPrefix(signature, "sha256=")), []byte(expectedMAC))
}

// VerifyBitbucketSignature verifies Bitbucket webhook signature
func VerifyBitbucketSignature(secret string, body []byte, signature string) bool {
	if secret == "" {
		return true
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expectedMAC := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(signature), []byte(expectedMAC))
}

// setCommitStatus dispatches to platform-specific commit status setters
func (s *Service) setCommitStatus(project *models.Project, sha string, state string, description string, gitlabProjectID int) {
	switch project.Platform {
	case "gitlab":
		s.setGitLabCommitStatus(project, sha, state, description, gitlabProjectID)
	case "github":
		s.setGitHubCommitStatus(project, sha, state, description)
	case "bitbucket":
		s.setBitbucketCommitStatus(project, sha, state, description)
	}
}
