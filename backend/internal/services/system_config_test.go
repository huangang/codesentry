package services

import (
	"testing"
)

func TestSplitAndTrim(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		sep      string
		expected []string
	}{
		{
			name:     "empty string",
			input:    "",
			sep:      ",",
			expected: nil,
		},
		{
			name:     "single value",
			input:    "value",
			sep:      ",",
			expected: []string{"value"},
		},
		{
			name:     "multiple values",
			input:    "a,b,c",
			sep:      ",",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "with spaces",
			input:    " a , b , c ",
			sep:      ",",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "empty parts filtered",
			input:    "a,,b,  ,c",
			sep:      ",",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "different separator",
			input:    "a;b;c",
			sep:      ";",
			expected: []string{"a", "b", "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitAndTrim(tt.input, tt.sep)
			if len(result) != len(tt.expected) {
				t.Errorf("splitAndTrim() returned %d items, expected %d", len(result), len(tt.expected))
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("splitAndTrim()[%d] = %q, expected %q", i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestLDAPConfigResponse_Defaults(t *testing.T) {
	cfg := &LDAPConfigResponse{
		Enabled:     false,
		Host:        "",
		Port:        389,
		BaseDN:      "",
		BindDN:      "",
		UserFilter:  "(uid=%s)",
		UseSSL:      false,
		PasswordSet: false,
	}

	if cfg.Enabled {
		t.Error("Enabled should be false by default")
	}
	if cfg.Host != "" {
		t.Errorf("Host should be empty, got %s", cfg.Host)
	}
	if cfg.Port != 389 {
		t.Errorf("default port should be 389, got %d", cfg.Port)
	}
	if cfg.BaseDN != "" {
		t.Errorf("BaseDN should be empty, got %s", cfg.BaseDN)
	}
	if cfg.BindDN != "" {
		t.Errorf("BindDN should be empty, got %s", cfg.BindDN)
	}
	if cfg.UserFilter != "(uid=%s)" {
		t.Errorf("default UserFilter should be (uid=%%s), got %s", cfg.UserFilter)
	}
	if cfg.UseSSL {
		t.Error("UseSSL should be false by default")
	}
	if cfg.PasswordSet {
		t.Error("PasswordSet should be false by default")
	}
}

func TestDailyReportConfigResponse_Structure(t *testing.T) {
	cfg := &DailyReportConfigResponse{
		Enabled:     true,
		Time:        "18:00",
		Timezone:    "Asia/Shanghai",
		LowScore:    60,
		LLMConfigID: 1,
		IMBotIDs:    []int{1, 2, 3},
	}

	if !cfg.Enabled {
		t.Error("Enabled should be true")
	}
	if cfg.Time != "18:00" {
		t.Errorf("Time = %q, expected %q", cfg.Time, "18:00")
	}
	if cfg.Timezone != "Asia/Shanghai" {
		t.Errorf("Timezone = %q, expected %q", cfg.Timezone, "Asia/Shanghai")
	}
	if cfg.LowScore != 60 {
		t.Errorf("LowScore = %d, expected 60", cfg.LowScore)
	}
	if cfg.LLMConfigID != 1 {
		t.Errorf("LLMConfigID = %d, expected 1", cfg.LLMConfigID)
	}
	if len(cfg.IMBotIDs) != 3 {
		t.Errorf("IMBotIDs should have 3 items, got %d", len(cfg.IMBotIDs))
	}
}

func TestChunkedReviewConfigResponse_Structure(t *testing.T) {
	cfg := &ChunkedReviewConfigResponse{
		Enabled:           true,
		Threshold:         50000,
		MaxTokensPerBatch: 30000,
	}

	if !cfg.Enabled {
		t.Error("Enabled should be true")
	}
	if cfg.Threshold != 50000 {
		t.Errorf("Threshold = %d, expected 50000", cfg.Threshold)
	}
	if cfg.MaxTokensPerBatch != 30000 {
		t.Errorf("MaxTokensPerBatch = %d, expected 30000", cfg.MaxTokensPerBatch)
	}
}

func TestFileContextConfigResponse_Structure(t *testing.T) {
	cfg := &FileContextConfigResponse{
		Enabled:          true,
		MaxFileSize:      102400,
		MaxFiles:         10,
		ExtractFunctions: true,
	}

	if !cfg.Enabled {
		t.Error("Enabled should be true")
	}
	if cfg.MaxFileSize != 102400 {
		t.Errorf("MaxFileSize = %d, expected 102400", cfg.MaxFileSize)
	}
	if cfg.MaxFiles != 10 {
		t.Errorf("MaxFiles = %d, expected 10", cfg.MaxFiles)
	}
	if !cfg.ExtractFunctions {
		t.Error("ExtractFunctions should be true")
	}
}

func TestUpdateLDAPConfigRequest_PartialUpdate(t *testing.T) {
	enabled := true
	host := "ldap.example.com"
	port := 636

	req := &UpdateLDAPConfigRequest{
		Enabled: &enabled,
		Host:    &host,
		Port:    &port,
	}

	if req.Enabled == nil || *req.Enabled != true {
		t.Error("Enabled should be set to true")
	}
	if req.Host == nil || *req.Host != "ldap.example.com" {
		t.Error("Host should be set")
	}
	if req.Port == nil || *req.Port != 636 {
		t.Error("Port should be set to 636")
	}
	if req.BaseDN != nil {
		t.Error("BaseDN should be nil (not set)")
	}
	if req.BindPassword != nil {
		t.Error("BindPassword should be nil (not set)")
	}
}

func TestUpdateDailyReportConfigRequest_PartialUpdate(t *testing.T) {
	enabled := false
	lowScore := 70
	ids := []int{1, 2}

	req := &UpdateDailyReportConfigRequest{
		Enabled:  &enabled,
		LowScore: &lowScore,
		IMBotIDs: ids,
	}

	if req.Enabled == nil || *req.Enabled != false {
		t.Error("Enabled should be set to false")
	}
	if req.LowScore == nil || *req.LowScore != 70 {
		t.Error("LowScore should be set to 70")
	}
	if len(req.IMBotIDs) != 2 {
		t.Errorf("IMBotIDs should have 2 items, got %d", len(req.IMBotIDs))
	}
	if req.Time != nil {
		t.Error("Time should be nil (not set)")
	}
}

func TestUpdateChunkedReviewConfigRequest_PartialUpdate(t *testing.T) {
	threshold := 100000
	maxTokens := 50000

	req := &UpdateChunkedReviewConfigRequest{
		Threshold:         &threshold,
		MaxTokensPerBatch: &maxTokens,
	}

	if req.Enabled != nil {
		t.Error("Enabled should be nil (not set)")
	}
	if req.Threshold == nil || *req.Threshold != 100000 {
		t.Error("Threshold should be set to 100000")
	}
	if req.MaxTokensPerBatch == nil || *req.MaxTokensPerBatch != 50000 {
		t.Error("MaxTokensPerBatch should be set to 50000")
	}
}

func TestUpdateFileContextConfigRequest_PartialUpdate(t *testing.T) {
	enabled := true
	maxSize := 204800

	req := &UpdateFileContextConfigRequest{
		Enabled:     &enabled,
		MaxFileSize: &maxSize,
	}

	if req.Enabled == nil || *req.Enabled != true {
		t.Error("Enabled should be set to true")
	}
	if req.MaxFileSize == nil || *req.MaxFileSize != 204800 {
		t.Error("MaxFileSize should be set to 204800")
	}
	if req.MaxFiles != nil {
		t.Error("MaxFiles should be nil (not set)")
	}
	if req.ExtractFunctions != nil {
		t.Error("ExtractFunctions should be nil (not set)")
	}
}
