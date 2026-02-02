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

func (s *FileContextService) GetExtractFunctions() bool {
	return s.configService.GetFileContextConfig().ExtractFunctions
}

func (s *FileContextService) BuildFileContext(project *models.Project, diff string, ref string) (string, error) {
	if !s.IsEnabled() {
		return "", nil
	}

	// Use function extraction mode if enabled
	if s.GetExtractFunctions() {
		return s.BuildFunctionContext(project, diff, ref)
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

// BuildFunctionContext extracts function/method definitions that contain modified lines
// This provides more focused context to AI by only including relevant code blocks
func (s *FileContextService) BuildFunctionContext(project *models.Project, diff string, ref string) (string, error) {
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

	var builder strings.Builder
	builder.WriteString("## Function Context (Modified Functions Only)\n\n")
	builder.WriteString("The following are the complete function/method definitions that contain the modified code:\n\n")

	maxFileSize := s.GetMaxFileSize()
	totalFunctions := 0

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
			log.Printf("[FileContext] File %s exceeds max size, skipping", file.FilePath)
			continue
		}

		modifiedRanges := extractModifiedRanges(file.Content)
		language := detectLanguage(file.FilePath)

		// Extract function definitions
		functions := ExtractFunctionsFromContext(content, modifiedRanges, language)

		if len(functions) > 0 {
			// Set file path for each function
			for i := range functions {
				functions[i].FilePath = file.FilePath
			}

			builder.WriteString(FormatFunctionDefinitions(functions, file.FilePath))
			totalFunctions += len(functions)
		}
	}

	if totalFunctions == 0 {
		return "", nil
	}

	log.Printf("[FileContext] Extracted %d function(s) from modified files", totalFunctions)
	return builder.String(), nil
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

// FunctionDefinition represents an extracted function/method definition
type FunctionDefinition struct {
	Name      string
	StartLine int
	EndLine   int
	Content   string
	Language  string
	FilePath  string
}

// ExtractFunctionsFromContext extracts function definitions that contain modified lines
func ExtractFunctionsFromContext(content string, modifiedRanges []LineRange, language string) []FunctionDefinition {
	lines := strings.Split(content, "\n")
	var functions []FunctionDefinition

	switch language {
	case "go":
		functions = extractGoFunctions(lines, modifiedRanges)
	case "javascript", "typescript":
		functions = extractJSFunctions(lines, modifiedRanges)
	case "python":
		functions = extractPythonFunctions(lines, modifiedRanges)
	case "java", "kotlin", "csharp":
		functions = extractJavaStyleFunctions(lines, modifiedRanges)
	default:
		// For unknown languages, extract surrounding context (20 lines before/after)
		functions = extractGenericContext(lines, modifiedRanges)
	}

	return functions
}

// extractGoFunctions extracts Go function/method definitions
func extractGoFunctions(lines []string, modifiedRanges []LineRange) []FunctionDefinition {
	var functions []FunctionDefinition
	funcPattern := regexp.MustCompile(`^func\s+(\([^)]+\)\s+)?(\w+)\s*\(`)

	type funcBoundary struct {
		name      string
		startLine int
		endLine   int
	}
	var boundaries []funcBoundary

	// Find all function boundaries
	braceCount := 0
	var currentFunc *funcBoundary
	for i, line := range lines {
		lineNum := i + 1
		trimmed := strings.TrimSpace(line)

		// Check for function start
		if matches := funcPattern.FindStringSubmatch(trimmed); len(matches) > 0 {
			if currentFunc != nil && braceCount == 0 {
				currentFunc.endLine = lineNum - 1
				boundaries = append(boundaries, *currentFunc)
			}
			funcName := matches[2]
			currentFunc = &funcBoundary{name: funcName, startLine: lineNum}
			braceCount = 0
		}

		// Track braces
		braceCount += strings.Count(line, "{") - strings.Count(line, "}")

		// Check for function end
		if currentFunc != nil && braceCount == 0 && strings.Contains(line, "}") {
			currentFunc.endLine = lineNum
			boundaries = append(boundaries, *currentFunc)
			currentFunc = nil
		}
	}

	// Close last function if still open
	if currentFunc != nil {
		currentFunc.endLine = len(lines)
		boundaries = append(boundaries, *currentFunc)
	}

	// Find functions that contain modified lines
	seen := make(map[string]bool)
	for _, fb := range boundaries {
		for _, r := range modifiedRanges {
			// Check if modified range overlaps with function
			if r.Start <= fb.endLine && r.End >= fb.startLine {
				if !seen[fb.name] {
					seen[fb.name] = true
					content := strings.Join(lines[fb.startLine-1:fb.endLine], "\n")
					functions = append(functions, FunctionDefinition{
						Name:      fb.name,
						StartLine: fb.startLine,
						EndLine:   fb.endLine,
						Content:   content,
						Language:  "go",
					})
				}
				break
			}
		}
	}

	return functions
}

// extractJSFunctions extracts JavaScript/TypeScript function definitions
func extractJSFunctions(lines []string, modifiedRanges []LineRange) []FunctionDefinition {
	var functions []FunctionDefinition
	// Match: function name, const name = function, const name = () =>, export function, etc.
	funcPatterns := []*regexp.Regexp{
		regexp.MustCompile(`^\s*(?:export\s+)?(?:async\s+)?function\s+(\w+)`),
		regexp.MustCompile(`^\s*(?:export\s+)?(?:const|let|var)\s+(\w+)\s*=\s*(?:async\s+)?(?:function|\([^)]*\)\s*=>|\w+\s*=>)`),
		regexp.MustCompile(`^\s*(?:async\s+)?(\w+)\s*\([^)]*\)\s*{`), // method shorthand
	}

	type funcBoundary struct {
		name      string
		startLine int
		endLine   int
	}
	var boundaries []funcBoundary

	braceCount := 0
	var currentFunc *funcBoundary
	for i, line := range lines {
		lineNum := i + 1

		// Check for function start
		for _, pattern := range funcPatterns {
			if matches := pattern.FindStringSubmatch(line); len(matches) > 0 {
				if currentFunc != nil && braceCount == 0 {
					currentFunc.endLine = lineNum - 1
					boundaries = append(boundaries, *currentFunc)
				}
				currentFunc = &funcBoundary{name: matches[1], startLine: lineNum}
				braceCount = 0
				break
			}
		}

		braceCount += strings.Count(line, "{") - strings.Count(line, "}")

		if currentFunc != nil && braceCount == 0 && strings.Contains(line, "}") {
			currentFunc.endLine = lineNum
			boundaries = append(boundaries, *currentFunc)
			currentFunc = nil
		}
	}

	if currentFunc != nil {
		currentFunc.endLine = len(lines)
		boundaries = append(boundaries, *currentFunc)
	}

	// Find functions that contain modified lines
	seen := make(map[string]bool)
	for _, fb := range boundaries {
		for _, r := range modifiedRanges {
			if r.Start <= fb.endLine && r.End >= fb.startLine {
				if !seen[fb.name] {
					seen[fb.name] = true
					content := strings.Join(lines[fb.startLine-1:fb.endLine], "\n")
					functions = append(functions, FunctionDefinition{
						Name:      fb.name,
						StartLine: fb.startLine,
						EndLine:   fb.endLine,
						Content:   content,
						Language:  "javascript",
					})
				}
				break
			}
		}
	}

	return functions
}

// extractPythonFunctions extracts Python function/method definitions
func extractPythonFunctions(lines []string, modifiedRanges []LineRange) []FunctionDefinition {
	var functions []FunctionDefinition
	funcPattern := regexp.MustCompile(`^(\s*)(?:async\s+)?def\s+(\w+)\s*\(`)

	type funcBoundary struct {
		name      string
		startLine int
		endLine   int
		indent    int
	}
	var boundaries []funcBoundary
	var currentFunc *funcBoundary

	for i, line := range lines {
		lineNum := i + 1

		if matches := funcPattern.FindStringSubmatch(line); len(matches) > 0 {
			indent := len(matches[1])
			if currentFunc != nil {
				currentFunc.endLine = lineNum - 1
				boundaries = append(boundaries, *currentFunc)
			}
			currentFunc = &funcBoundary{
				name:      matches[2],
				startLine: lineNum,
				indent:    indent,
			}
		} else if currentFunc != nil && len(strings.TrimSpace(line)) > 0 {
			// Check if we've dedented past the function
			currentIndent := len(line) - len(strings.TrimLeft(line, " \t"))
			if currentIndent <= currentFunc.indent && !strings.HasPrefix(strings.TrimSpace(line), "#") {
				currentFunc.endLine = lineNum - 1
				boundaries = append(boundaries, *currentFunc)
				currentFunc = nil
			}
		}
	}

	if currentFunc != nil {
		currentFunc.endLine = len(lines)
		boundaries = append(boundaries, *currentFunc)
	}

	// Find functions that contain modified lines
	seen := make(map[string]bool)
	for _, fb := range boundaries {
		for _, r := range modifiedRanges {
			if r.Start <= fb.endLine && r.End >= fb.startLine {
				if !seen[fb.name] {
					seen[fb.name] = true
					content := strings.Join(lines[fb.startLine-1:fb.endLine], "\n")
					functions = append(functions, FunctionDefinition{
						Name:      fb.name,
						StartLine: fb.startLine,
						EndLine:   fb.endLine,
						Content:   content,
						Language:  "python",
					})
				}
				break
			}
		}
	}

	return functions
}

// extractJavaStyleFunctions extracts Java/Kotlin/C# method definitions
func extractJavaStyleFunctions(lines []string, modifiedRanges []LineRange) []FunctionDefinition {
	var functions []FunctionDefinition
	// Match method definitions
	methodPattern := regexp.MustCompile(`^\s*(?:public|private|protected|internal|static|final|override|suspend|async)?\s*(?:public|private|protected|internal|static|final|override|suspend|async)?\s*(?:\w+(?:<[^>]+>)?)\s+(\w+)\s*\(`)

	type funcBoundary struct {
		name      string
		startLine int
		endLine   int
	}
	var boundaries []funcBoundary

	braceCount := 0
	var currentFunc *funcBoundary
	for i, line := range lines {
		lineNum := i + 1

		if matches := methodPattern.FindStringSubmatch(line); len(matches) > 0 {
			if currentFunc != nil && braceCount == 0 {
				currentFunc.endLine = lineNum - 1
				boundaries = append(boundaries, *currentFunc)
			}
			currentFunc = &funcBoundary{name: matches[1], startLine: lineNum}
			braceCount = 0
		}

		braceCount += strings.Count(line, "{") - strings.Count(line, "}")

		if currentFunc != nil && braceCount == 0 && strings.Contains(line, "}") {
			currentFunc.endLine = lineNum
			boundaries = append(boundaries, *currentFunc)
			currentFunc = nil
		}
	}

	if currentFunc != nil {
		currentFunc.endLine = len(lines)
		boundaries = append(boundaries, *currentFunc)
	}

	// Find functions that contain modified lines
	seen := make(map[string]bool)
	for _, fb := range boundaries {
		for _, r := range modifiedRanges {
			if r.Start <= fb.endLine && r.End >= fb.startLine {
				if !seen[fb.name] {
					seen[fb.name] = true
					content := strings.Join(lines[fb.startLine-1:fb.endLine], "\n")
					functions = append(functions, FunctionDefinition{
						Name:      fb.name,
						StartLine: fb.startLine,
						EndLine:   fb.endLine,
						Content:   content,
						Language:  "java",
					})
				}
				break
			}
		}
	}

	return functions
}

// extractGenericContext extracts surrounding context for unknown languages
func extractGenericContext(lines []string, modifiedRanges []LineRange) []FunctionDefinition {
	var functions []FunctionDefinition
	contextLines := 15 // Lines of context before/after

	for _, r := range modifiedRanges {
		start := r.Start - contextLines
		if start < 1 {
			start = 1
		}
		end := r.End + contextLines
		if end > len(lines) {
			end = len(lines)
		}

		content := strings.Join(lines[start-1:end], "\n")
		functions = append(functions, FunctionDefinition{
			Name:      fmt.Sprintf("context_%d_%d", r.Start, r.End),
			StartLine: start,
			EndLine:   end,
			Content:   content,
			Language:  "text",
		})
	}

	return functions
}

// FormatFunctionDefinitions formats extracted functions for AI context
func FormatFunctionDefinitions(functions []FunctionDefinition, filePath string) string {
	if len(functions) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("### Modified Functions in `%s`\n\n", filePath))

	for _, fn := range functions {
		builder.WriteString(fmt.Sprintf("#### Function: `%s` (lines %d-%d)\n\n", fn.Name, fn.StartLine, fn.EndLine))
		builder.WriteString(fmt.Sprintf("```%s\n%s\n```\n\n", fn.Language, fn.Content))
	}

	return builder.String()
}
