package services

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/huangang/codesentry/backend/internal/models"
)

type FileContextService struct {
	httpClient    *http.Client
	configService *SystemConfigService
}

func NewFileContextService(configService *SystemConfigService) *FileContextService {
	return &FileContextService{
		httpClient:    &http.Client{Timeout: 30 * time.Second},
		configService: configService,
	}
}

type FileContext struct {
	FilePath       string
	Content        string
	ModifiedRanges []LineRange
	Language       string
}

type LineRange struct {
	Start int
	End   int
}

func (s *FileContextService) IsEnabled() bool {
	return s.configService.GetFileContextConfig().Enabled
}

func (s *FileContextService) GetMaxFileSize() int {
	return s.configService.GetFileContextConfig().MaxFileSize
}

func (s *FileContextService) GetMaxFiles() int {
	return s.configService.GetFileContextConfig().MaxFiles
}

func (s *FileContextService) BuildFileContext(project *models.Project, diff string, ref string) (string, error) {
	if !s.IsEnabled() {
		return "", nil
	}

	files := ParseDiffToFiles(diff)
	if len(files) == 0 {
		return "", nil
	}

	maxFiles := s.GetMaxFiles()
	if len(files) > maxFiles {
		sort.Slice(files, func(i, j int) bool {
			return (files[i].Additions + files[i].Deletions) > (files[j].Additions + files[j].Deletions)
		})
		files = files[:maxFiles]
	}

	var contexts []FileContext
	maxFileSize := s.GetMaxFileSize()

	for _, file := range files {
		if file.FilePath == "" || file.FilePath == "unknown" || file.FilePath == "/dev/null" {
			continue
		}

		content, err := s.fetchFileContent(project, file.FilePath, ref)
		if err != nil {
			log.Printf("[FileContext] Failed to fetch %s: %v", file.FilePath, err)
			continue
		}

		if len(content) > maxFileSize {
			log.Printf("[FileContext] File %s exceeds max size (%d > %d), skipping", file.FilePath, len(content), maxFileSize)
			continue
		}

		modifiedRanges := extractModifiedRanges(file.Content)

		contexts = append(contexts, FileContext{
			FilePath:       file.FilePath,
			Content:        content,
			ModifiedRanges: modifiedRanges,
			Language:       detectLanguage(file.FilePath),
		})
	}

	if len(contexts) == 0 {
		return "", nil
	}

	return formatFileContexts(contexts), nil
}

func (s *FileContextService) fetchFileContent(project *models.Project, filePath, ref string) (string, error) {
	switch project.Platform {
	case "gitlab":
		return s.fetchGitLabFile(project, filePath, ref)
	case "github":
		return s.fetchGitHubFile(project, filePath, ref)
	case "bitbucket":
		return s.fetchBitbucketFile(project, filePath, ref)
	default:
		return "", fmt.Errorf("unsupported platform: %s", project.Platform)
	}
}

func (s *FileContextService) fetchGitLabFile(project *models.Project, filePath, ref string) (string, error) {
	info, err := parseRepoInfo(project.URL)
	if err != nil {
		return "", err
	}

	encodedPath := strings.ReplaceAll(filePath, "/", "%2F")
	encodedPath = strings.ReplaceAll(encodedPath, ".", "%2E")

	apiURL := fmt.Sprintf("%s/api/v4/projects/%s/repository/files/%s/raw?ref=%s",
		info.baseURL, strings.ReplaceAll(info.projectPath, "/", "%2F"), encodedPath, ref)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", err
	}
	if project.AccessToken != "" {
		req.Header.Set("PRIVATE-TOKEN", project.AccessToken)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitLab API returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func (s *FileContextService) fetchGitHubFile(project *models.Project, filePath, ref string) (string, error) {
	info, err := parseRepoInfo(project.URL)
	if err != nil {
		return "", err
	}

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s?ref=%s",
		info.owner, info.repo, filePath, ref)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if project.AccessToken != "" {
		req.Header.Set("Authorization", "token "+project.AccessToken)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var result struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if result.Encoding == "base64" {
		decoded, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(result.Content, "\n", ""))
		if err != nil {
			return "", err
		}
		return string(decoded), nil
	}

	return result.Content, nil
}

func (s *FileContextService) fetchBitbucketFile(project *models.Project, filePath, ref string) (string, error) {
	info, err := parseRepoInfo(project.URL)
	if err != nil {
		return "", err
	}

	apiURL := fmt.Sprintf("https://api.bitbucket.org/2.0/repositories/%s/src/%s/%s",
		info.projectPath, ref, filePath)

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

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Bitbucket API returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func extractModifiedRanges(diffContent string) []LineRange {
	var ranges []LineRange

	hunkPattern := regexp.MustCompile(`@@ -\d+(?:,\d+)? \+(\d+)(?:,(\d+))? @@`)
	matches := hunkPattern.FindAllStringSubmatch(diffContent, -1)

	for _, match := range matches {
		start := 1
		count := 1
		if len(match) >= 2 {
			fmt.Sscanf(match[1], "%d", &start)
		}
		if len(match) >= 3 && match[2] != "" {
			fmt.Sscanf(match[2], "%d", &count)
		}
		ranges = append(ranges, LineRange{
			Start: start,
			End:   start + count - 1,
		})
	}

	return ranges
}

func detectLanguage(filePath string) string {
	ext := strings.ToLower(filePath)
	if idx := strings.LastIndex(ext, "."); idx != -1 {
		ext = ext[idx:]
	}

	langMap := map[string]string{
		".go":     "go",
		".js":     "javascript",
		".ts":     "typescript",
		".tsx":    "typescript",
		".jsx":    "javascript",
		".py":     "python",
		".java":   "java",
		".c":      "c",
		".cpp":    "cpp",
		".h":      "c",
		".hpp":    "cpp",
		".cs":     "csharp",
		".rb":     "ruby",
		".php":    "php",
		".swift":  "swift",
		".kt":     "kotlin",
		".rs":     "rust",
		".vue":    "vue",
		".svelte": "svelte",
	}

	if lang, ok := langMap[ext]; ok {
		return lang
	}
	return "text"
}

func formatFileContexts(contexts []FileContext) string {
	var builder strings.Builder

	builder.WriteString("## File Context (Full Source)\n\n")
	builder.WriteString("The following are the complete source files that contain the changes. ")
	builder.WriteString("Modified line ranges are marked with `[MODIFIED]` comments.\n\n")

	for _, ctx := range contexts {
		builder.WriteString(fmt.Sprintf("### File: `%s`\n\n", ctx.FilePath))

		if len(ctx.ModifiedRanges) > 0 {
			builder.WriteString("**Modified ranges:** ")
			for i, r := range ctx.ModifiedRanges {
				if i > 0 {
					builder.WriteString(", ")
				}
				builder.WriteString(fmt.Sprintf("lines %d-%d", r.Start, r.End))
			}
			builder.WriteString("\n\n")
		}

		builder.WriteString(fmt.Sprintf("```%s\n", ctx.Language))

		lines := strings.Split(ctx.Content, "\n")
		for i, line := range lines {
			lineNum := i + 1
			isModified := false
			for _, r := range ctx.ModifiedRanges {
				if lineNum >= r.Start && lineNum <= r.End {
					isModified = true
					break
				}
			}

			if isModified {
				builder.WriteString(fmt.Sprintf("%4d | » %s\n", lineNum, line))
			} else {
				builder.WriteString(fmt.Sprintf("%4d |   %s\n", lineNum, line))
			}
		}

		builder.WriteString("```\n\n")
	}

	return builder.String()
}
