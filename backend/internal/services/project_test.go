package services

import (
	"testing"
)

func TestProjectListRequest_Defaults(t *testing.T) {
	req := &ProjectListRequest{}

	if req.Page != 0 {
		t.Errorf("default Page should be 0, got %d", req.Page)
	}
	if req.PageSize != 0 {
		t.Errorf("default PageSize should be 0, got %d", req.PageSize)
	}
}

func TestProjectListRequest_WithFilters(t *testing.T) {
	req := &ProjectListRequest{
		Page:     2,
		PageSize: 25,
		Name:     "myproject",
		Platform: "gitlab",
	}

	if req.Page != 2 {
		t.Errorf("Page = %d, expected 2", req.Page)
	}
	if req.PageSize != 25 {
		t.Errorf("PageSize = %d, expected 25", req.PageSize)
	}
	if req.Name != "myproject" {
		t.Errorf("Name = %q, expected %q", req.Name, "myproject")
	}
	if req.Platform != "gitlab" {
		t.Errorf("Platform = %q, expected %q", req.Platform, "gitlab")
	}
}

func TestProjectListResponse_Structure(t *testing.T) {
	resp := &ProjectListResponse{
		Total:    50,
		Page:     1,
		PageSize: 10,
		Items:    nil,
	}

	if resp.Total != 50 {
		t.Errorf("Total = %d, expected 50", resp.Total)
	}
	if resp.Page != 1 {
		t.Errorf("Page = %d, expected 1", resp.Page)
	}
	if resp.PageSize != 10 {
		t.Errorf("PageSize = %d, expected 10", resp.PageSize)
	}
	if resp.Items != nil {
		t.Error("Items should be nil")
	}
}

func TestCreateProjectRequest_RequiredFields(t *testing.T) {
	req := &CreateProjectRequest{
		Name:     "Test Project",
		URL:      "https://github.com/org/repo",
		Platform: "github",
	}

	if req.Name == "" {
		t.Error("Name is required")
	}
	if req.URL == "" {
		t.Error("URL is required")
	}
	if req.Platform == "" {
		t.Error("Platform is required")
	}
}

func TestCreateProjectRequest_AllFields(t *testing.T) {
	botID := uint(5)
	req := &CreateProjectRequest{
		Name:           "Full Project",
		URL:            "https://gitlab.com/team/app",
		Platform:       "gitlab",
		AccessToken:    "token123",
		WebhookSecret:  "secret456",
		FileExtensions: ".go,.js",
		ReviewEvents:   "push,merge_request",
		AIEnabled:      true,
		AIPrompt:       "Custom prompt",
		IMEnabled:      true,
		IMBotID:        &botID,
		MinScore:       70.0,
	}

	if req.Name != "Full Project" {
		t.Errorf("Name = %q, expected %q", req.Name, "Full Project")
	}
	if req.URL != "https://gitlab.com/team/app" {
		t.Errorf("URL = %q, expected %q", req.URL, "https://gitlab.com/team/app")
	}
	if req.Platform != "gitlab" {
		t.Errorf("Platform = %q, expected %q", req.Platform, "gitlab")
	}
	if req.AccessToken != "token123" {
		t.Errorf("AccessToken = %q, expected %q", req.AccessToken, "token123")
	}
	if req.WebhookSecret != "secret456" {
		t.Errorf("WebhookSecret = %q, expected %q", req.WebhookSecret, "secret456")
	}
	if req.FileExtensions != ".go,.js" {
		t.Errorf("FileExtensions = %q, expected %q", req.FileExtensions, ".go,.js")
	}
	if req.ReviewEvents != "push,merge_request" {
		t.Errorf("ReviewEvents = %q, expected %q", req.ReviewEvents, "push,merge_request")
	}
	if !req.AIEnabled {
		t.Error("AIEnabled should be true")
	}
	if req.AIPrompt != "Custom prompt" {
		t.Errorf("AIPrompt = %q, expected %q", req.AIPrompt, "Custom prompt")
	}
	if !req.IMEnabled {
		t.Error("IMEnabled should be true")
	}
	if req.MinScore != 70.0 {
		t.Errorf("MinScore = %f, expected 70.0", req.MinScore)
	}
	if req.IMBotID == nil || *req.IMBotID != 5 {
		t.Error("IMBotID should be 5")
	}
}

func TestUpdateProjectRequest_PartialUpdate(t *testing.T) {
	enabled := true
	minScore := 80.0

	req := &UpdateProjectRequest{
		Name:      "Updated Name",
		AIEnabled: &enabled,
		MinScore:  &minScore,
	}

	if req.Name != "Updated Name" {
		t.Errorf("Name = %q, expected %q", req.Name, "Updated Name")
	}
	if req.AIEnabled == nil || *req.AIEnabled != true {
		t.Error("AIEnabled should be true")
	}
	if req.MinScore == nil || *req.MinScore != 80.0 {
		t.Error("MinScore should be 80.0")
	}
	if req.URL != "" {
		t.Errorf("URL should be empty, got %q", req.URL)
	}
	if req.Platform != "" {
		t.Errorf("Platform should be empty, got %q", req.Platform)
	}
}

func TestCreateProjectParams_Structure(t *testing.T) {
	params := &CreateProjectParams{
		Name:           "Auto Project",
		URL:            "https://github.com/auto/repo",
		Platform:       "github",
		AccessToken:    "token",
		WebhookSecret:  "secret",
		AIEnabled:      true,
		FileExtensions: ".go,.ts",
		ReviewEvents:   "push",
		IgnorePatterns: "*.lock",
	}

	if params.Name != "Auto Project" {
		t.Errorf("Name = %q, expected %q", params.Name, "Auto Project")
	}
	if params.URL != "https://github.com/auto/repo" {
		t.Errorf("URL = %q, expected %q", params.URL, "https://github.com/auto/repo")
	}
	if params.Platform != "github" {
		t.Errorf("Platform = %q, expected %q", params.Platform, "github")
	}
	if params.AccessToken != "token" {
		t.Errorf("AccessToken = %q, expected %q", params.AccessToken, "token")
	}
	if params.WebhookSecret != "secret" {
		t.Errorf("WebhookSecret = %q, expected %q", params.WebhookSecret, "secret")
	}
	if !params.AIEnabled {
		t.Error("AIEnabled should be true")
	}
	if params.FileExtensions != ".go,.ts" {
		t.Errorf("FileExtensions = %q, expected %q", params.FileExtensions, ".go,.ts")
	}
	if params.ReviewEvents != "push" {
		t.Errorf("ReviewEvents = %q, expected %q", params.ReviewEvents, "push")
	}
	if params.IgnorePatterns != "*.lock" {
		t.Errorf("IgnorePatterns = %q, expected %q", params.IgnorePatterns, "*.lock")
	}
}

func TestProjectService_GetDefaultPrompt(t *testing.T) {
	service := &ProjectService{}

	prompt := service.GetDefaultPrompt()
	if prompt == "" {
		t.Error("GetDefaultPrompt should not return empty string")
	}

	if !containsStr(prompt, "总分") && !containsStr(prompt, "Total Score") {
		t.Error("default prompt should contain scoring instruction")
	}
}

func TestProjectService_GetDefaultPromptByLang(t *testing.T) {
	service := &ProjectService{}

	zhPrompt := service.GetDefaultPromptByLang("zh")
	if !containsStr(zhPrompt, "总分") {
		t.Error("Chinese prompt should contain 总分")
	}
	if !containsStr(zhPrompt, "评分维度") {
		t.Error("Chinese prompt should contain 评分维度")
	}

	enPrompt := service.GetDefaultPromptByLang("en")
	if !containsStr(enPrompt, "Total Score") {
		t.Error("English prompt should contain Total Score")
	}
	if !containsStr(enPrompt, "Scoring Dimensions") {
		t.Error("English prompt should contain Scoring Dimensions")
	}
}

func TestProjectService_GetDefaultPrompt_ContainsPlaceholders(t *testing.T) {
	service := &ProjectService{}
	prompt := service.GetDefaultPrompt()

	placeholders := []string{"{{diffs}}", "{{commits}}", "{{file_context}}"}
	for _, p := range placeholders {
		if !containsStr(prompt, p) {
			t.Errorf("prompt should contain placeholder %q", p)
		}
	}
}

func TestProjectService_GetDefaultPrompt_ContainsConditionalBlock(t *testing.T) {
	service := &ProjectService{}
	prompt := service.GetDefaultPrompt()

	if !containsStr(prompt, "{{#if_file_context}}") {
		t.Error("prompt should contain if_file_context block start")
	}
	if !containsStr(prompt, "{{/if_file_context}}") {
		t.Error("prompt should contain if_file_context block end")
	}
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
