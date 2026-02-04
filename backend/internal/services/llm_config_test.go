package services

import (
	"testing"
)

func TestLLMConfigListRequest_Defaults(t *testing.T) {
	req := &LLMConfigListRequest{}

	if req.Page != 0 {
		t.Errorf("default Page should be 0, got %d", req.Page)
	}
	if req.PageSize != 0 {
		t.Errorf("default PageSize should be 0, got %d", req.PageSize)
	}
}

func TestLLMConfigListRequest_WithFilters(t *testing.T) {
	active := true
	req := &LLMConfigListRequest{
		Page:     1,
		PageSize: 20,
		Name:     "gpt-4",
		Provider: "openai",
		IsActive: &active,
	}

	if req.Page != 1 {
		t.Errorf("Page = %d, expected 1", req.Page)
	}
	if req.PageSize != 20 {
		t.Errorf("PageSize = %d, expected 20", req.PageSize)
	}
	if req.Name != "gpt-4" {
		t.Errorf("Name = %q, expected %q", req.Name, "gpt-4")
	}
	if req.Provider != "openai" {
		t.Errorf("Provider = %q, expected %q", req.Provider, "openai")
	}
	if req.IsActive == nil || *req.IsActive != true {
		t.Error("IsActive should be true")
	}
}

func TestLLMConfigListResponse_Structure(t *testing.T) {
	resp := &LLMConfigListResponse{
		Total:    3,
		Page:     1,
		PageSize: 10,
		Items:    nil,
	}

	if resp.Total != 3 {
		t.Errorf("Total = %d, expected 3", resp.Total)
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

func TestCreateLLMConfigRequest_RequiredFields(t *testing.T) {
	req := &CreateLLMConfigRequest{
		Name:    "My LLM",
		BaseURL: "https://api.openai.com/v1",
		APIKey:  "sk-xxx",
		Model:   "gpt-4",
	}

	if req.Name == "" {
		t.Error("Name is required")
	}
	if req.BaseURL == "" {
		t.Error("BaseURL is required")
	}
	if req.APIKey == "" {
		t.Error("APIKey is required")
	}
	if req.Model == "" {
		t.Error("Model is required")
	}
}

func TestCreateLLMConfigRequest_AllFields(t *testing.T) {
	req := &CreateLLMConfigRequest{
		Name:        "Claude",
		Provider:    "anthropic",
		BaseURL:     "https://api.anthropic.com",
		APIKey:      "sk-ant-xxx",
		Model:       "claude-3-opus",
		MaxTokens:   8192,
		Temperature: 0.7,
		IsDefault:   true,
		IsActive:    true,
	}

	if req.Name != "Claude" {
		t.Errorf("Name = %q, expected %q", req.Name, "Claude")
	}
	if req.Provider != "anthropic" {
		t.Errorf("Provider = %q, expected %q", req.Provider, "anthropic")
	}
	if req.BaseURL != "https://api.anthropic.com" {
		t.Errorf("BaseURL = %q, expected %q", req.BaseURL, "https://api.anthropic.com")
	}
	if req.APIKey != "sk-ant-xxx" {
		t.Errorf("APIKey = %q, expected %q", req.APIKey, "sk-ant-xxx")
	}
	if req.Model != "claude-3-opus" {
		t.Errorf("Model = %q, expected %q", req.Model, "claude-3-opus")
	}
	if req.MaxTokens != 8192 {
		t.Errorf("MaxTokens = %d, expected 8192", req.MaxTokens)
	}
	if req.Temperature != 0.7 {
		t.Errorf("Temperature = %f, expected 0.7", req.Temperature)
	}
	if !req.IsDefault {
		t.Error("IsDefault should be true")
	}
	if !req.IsActive {
		t.Error("IsActive should be true")
	}
}

func TestUpdateLLMConfigRequest_PartialUpdate(t *testing.T) {
	maxTokens := 4096
	temp := 0.5
	isDefault := false

	req := &UpdateLLMConfigRequest{
		Name:        "Updated Name",
		MaxTokens:   &maxTokens,
		Temperature: &temp,
		IsDefault:   &isDefault,
	}

	if req.Name != "Updated Name" {
		t.Errorf("Name = %q, expected %q", req.Name, "Updated Name")
	}
	if req.MaxTokens == nil || *req.MaxTokens != 4096 {
		t.Error("MaxTokens should be 4096")
	}
	if req.Temperature == nil || *req.Temperature != 0.5 {
		t.Error("Temperature should be 0.5")
	}
	if req.IsDefault == nil || *req.IsDefault != false {
		t.Error("IsDefault should be false")
	}
	if req.Provider != "" {
		t.Errorf("Provider should be empty, got %q", req.Provider)
	}
	if req.APIKey != "" {
		t.Errorf("APIKey should be empty, got %q", req.APIKey)
	}
}

func TestLLMProviders(t *testing.T) {
	providers := []string{
		"openai",
		"anthropic",
		"gemini",
		"ollama",
		"azure",
	}

	for _, provider := range providers {
		req := &CreateLLMConfigRequest{
			Name:     "Test",
			Provider: provider,
			BaseURL:  "https://api.example.com",
			APIKey:   "key",
			Model:    "model",
		}

		if req.Provider != provider {
			t.Errorf("Provider = %q, expected %q", req.Provider, provider)
		}
		if req.Name != "Test" {
			t.Errorf("Name = %q, expected %q", req.Name, "Test")
		}
		if req.BaseURL != "https://api.example.com" {
			t.Errorf("BaseURL = %q, expected %q", req.BaseURL, "https://api.example.com")
		}
		if req.APIKey != "key" {
			t.Errorf("APIKey = %q, expected %q", req.APIKey, "key")
		}
		if req.Model != "model" {
			t.Errorf("Model = %q, expected %q", req.Model, "model")
		}
	}
}
