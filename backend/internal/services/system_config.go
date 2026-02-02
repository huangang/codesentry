package services

import (
	"strconv"
	"strings"

	"github.com/huangang/codesentry/backend/internal/models"
	"gorm.io/gorm"
)

type SystemConfigService struct {
	db *gorm.DB
}

func NewSystemConfigService(db *gorm.DB) *SystemConfigService {
	return &SystemConfigService{db: db}
}

func (s *SystemConfigService) Get(key string) (string, error) {
	var cfg models.SystemConfig
	if err := s.db.Where("`key` = ?", key).First(&cfg).Error; err != nil {
		return "", err
	}
	return cfg.Value, nil
}

func (s *SystemConfigService) GetWithDefault(key, defaultValue string) string {
	value, err := s.Get(key)
	if err != nil {
		return defaultValue
	}
	return value
}

func (s *SystemConfigService) Set(key, value string) error {
	var cfg models.SystemConfig
	err := s.db.Where("`key` = ?", key).First(&cfg).Error
	if err == gorm.ErrRecordNotFound {
		cfg = models.SystemConfig{
			Key:   key,
			Value: value,
		}
		return s.db.Create(&cfg).Error
	}
	if err != nil {
		return err
	}
	return s.db.Model(&cfg).Update("value", value).Error
}

func (s *SystemConfigService) GetByGroup(group string) ([]models.SystemConfig, error) {
	var configs []models.SystemConfig
	if err := s.db.Where("`group` = ?", group).Find(&configs).Error; err != nil {
		return nil, err
	}
	return configs, nil
}

type LDAPConfigResponse struct {
	Enabled     bool   `json:"enabled"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	BaseDN      string `json:"base_dn"`
	BindDN      string `json:"bind_dn"`
	UserFilter  string `json:"user_filter"`
	UseSSL      bool   `json:"use_ssl"`
	PasswordSet bool   `json:"password_set"`
}

func (s *SystemConfigService) GetLDAPConfig() *LDAPConfigResponse {
	port, _ := strconv.Atoi(s.GetWithDefault("ldap_port", "389"))
	return &LDAPConfigResponse{
		Enabled:     s.GetWithDefault("ldap_enabled", "false") == "true",
		Host:        s.GetWithDefault("ldap_host", ""),
		Port:        port,
		BaseDN:      s.GetWithDefault("ldap_base_dn", ""),
		BindDN:      s.GetWithDefault("ldap_bind_dn", ""),
		UserFilter:  s.GetWithDefault("ldap_user_filter", "(uid=%s)"),
		UseSSL:      s.GetWithDefault("ldap_use_ssl", "false") == "true",
		PasswordSet: s.GetWithDefault("ldap_bind_password", "") != "",
	}
}

type UpdateLDAPConfigRequest struct {
	Enabled      *bool   `json:"enabled"`
	Host         *string `json:"host"`
	Port         *int    `json:"port"`
	BaseDN       *string `json:"base_dn"`
	BindDN       *string `json:"bind_dn"`
	BindPassword *string `json:"bind_password"`
	UserFilter   *string `json:"user_filter"`
	UseSSL       *bool   `json:"use_ssl"`
}

func (s *SystemConfigService) UpdateLDAPConfig(req *UpdateLDAPConfigRequest) error {
	if req.Enabled != nil {
		if err := s.Set("ldap_enabled", strconv.FormatBool(*req.Enabled)); err != nil {
			return err
		}
	}
	if req.Host != nil {
		if err := s.Set("ldap_host", *req.Host); err != nil {
			return err
		}
	}
	if req.Port != nil {
		if err := s.Set("ldap_port", strconv.Itoa(*req.Port)); err != nil {
			return err
		}
	}
	if req.BaseDN != nil {
		if err := s.Set("ldap_base_dn", *req.BaseDN); err != nil {
			return err
		}
	}
	if req.BindDN != nil {
		if err := s.Set("ldap_bind_dn", *req.BindDN); err != nil {
			return err
		}
	}
	if req.BindPassword != nil && *req.BindPassword != "" {
		if err := s.Set("ldap_bind_password", *req.BindPassword); err != nil {
			return err
		}
	}
	if req.UserFilter != nil {
		if err := s.Set("ldap_user_filter", *req.UserFilter); err != nil {
			return err
		}
	}
	if req.UseSSL != nil {
		if err := s.Set("ldap_use_ssl", strconv.FormatBool(*req.UseSSL)); err != nil {
			return err
		}
	}
	return nil
}

// Daily Report Config
type DailyReportConfigResponse struct {
	Enabled     bool   `json:"enabled"`
	Time        string `json:"time"`
	Timezone    string `json:"timezone"`
	LowScore    int    `json:"low_score"`
	LLMConfigID int    `json:"llm_config_id"`
	IMBotIDs    []int  `json:"im_bot_ids"`
}

func (s *SystemConfigService) GetDailyReportConfig() *DailyReportConfigResponse {
	lowScore, _ := strconv.Atoi(s.GetWithDefault("daily_report_low_score", "60"))
	llmConfigID, _ := strconv.Atoi(s.GetWithDefault("daily_report_llm_config_id", "0"))
	imBotIDsStr := s.GetWithDefault("daily_report_im_bot_ids", "")
	var imBotIDs []int
	if imBotIDsStr != "" {
		for _, idStr := range splitAndTrim(imBotIDsStr, ",") {
			if id, err := strconv.Atoi(idStr); err == nil {
				imBotIDs = append(imBotIDs, id)
			}
		}
	}
	return &DailyReportConfigResponse{
		Enabled:     s.GetWithDefault("daily_report_enabled", "false") == "true",
		Time:        s.GetWithDefault("daily_report_time", "18:00"),
		Timezone:    s.GetWithDefault("daily_report_timezone", "Asia/Shanghai"),
		LowScore:    lowScore,
		LLMConfigID: llmConfigID,
		IMBotIDs:    imBotIDs,
	}
}

func splitAndTrim(s, sep string) []string {
	if s == "" {
		return nil
	}
	parts := make([]string, 0)
	for _, p := range strings.Split(s, sep) {
		p = strings.TrimSpace(p)
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

type UpdateDailyReportConfigRequest struct {
	Enabled     *bool   `json:"enabled"`
	Time        *string `json:"time"`
	Timezone    *string `json:"timezone"`
	LowScore    *int    `json:"low_score"`
	LLMConfigID *int    `json:"llm_config_id"`
	IMBotIDs    []int   `json:"im_bot_ids"`
}

func (s *SystemConfigService) UpdateDailyReportConfig(req *UpdateDailyReportConfigRequest) error {
	if req.Enabled != nil {
		if err := s.Set("daily_report_enabled", strconv.FormatBool(*req.Enabled)); err != nil {
			return err
		}
	}
	if req.Time != nil {
		if err := s.Set("daily_report_time", *req.Time); err != nil {
			return err
		}
	}
	if req.Timezone != nil {
		if err := s.Set("daily_report_timezone", *req.Timezone); err != nil {
			return err
		}
	}
	if req.LowScore != nil {
		if err := s.Set("daily_report_low_score", strconv.Itoa(*req.LowScore)); err != nil {
			return err
		}
	}
	if req.LLMConfigID != nil {
		if err := s.Set("daily_report_llm_config_id", strconv.Itoa(*req.LLMConfigID)); err != nil {
			return err
		}
	}
	if req.IMBotIDs != nil {
		var ids []string
		for _, id := range req.IMBotIDs {
			ids = append(ids, strconv.Itoa(id))
		}
		if err := s.Set("daily_report_im_bot_ids", strings.Join(ids, ",")); err != nil {
			return err
		}
	}
	return nil
}

// Chunked Review Config
type ChunkedReviewConfigResponse struct {
	Enabled           bool `json:"enabled"`
	Threshold         int  `json:"threshold"`
	MaxTokensPerBatch int  `json:"max_tokens_per_batch"`
}

func (s *SystemConfigService) GetChunkedReviewConfig() *ChunkedReviewConfigResponse {
	threshold, _ := strconv.Atoi(s.GetWithDefault("chunked_review_threshold", "50000"))
	maxTokens, _ := strconv.Atoi(s.GetWithDefault("chunked_review_max_tokens_per_batch", "30000"))
	return &ChunkedReviewConfigResponse{
		Enabled:           s.GetWithDefault("chunked_review_enabled", "true") == "true",
		Threshold:         threshold,
		MaxTokensPerBatch: maxTokens,
	}
}

type UpdateChunkedReviewConfigRequest struct {
	Enabled           *bool `json:"enabled"`
	Threshold         *int  `json:"threshold"`
	MaxTokensPerBatch *int  `json:"max_tokens_per_batch"`
}

func (s *SystemConfigService) UpdateChunkedReviewConfig(req *UpdateChunkedReviewConfigRequest) error {
	if req.Enabled != nil {
		if err := s.Set("chunked_review_enabled", strconv.FormatBool(*req.Enabled)); err != nil {
			return err
		}
	}
	if req.Threshold != nil {
		if err := s.Set("chunked_review_threshold", strconv.Itoa(*req.Threshold)); err != nil {
			return err
		}
	}
	if req.MaxTokensPerBatch != nil {
		if err := s.Set("chunked_review_max_tokens_per_batch", strconv.Itoa(*req.MaxTokensPerBatch)); err != nil {
			return err
		}
	}
	return nil
}

// File Context Config - for enhanced code review with full file context
type FileContextConfigResponse struct {
	Enabled          bool `json:"enabled"`
	MaxFileSize      int  `json:"max_file_size"`     // Max file size in bytes to fetch (default 100KB)
	MaxFiles         int  `json:"max_files"`         // Max number of files to fetch context for (default 10)
	ExtractFunctions bool `json:"extract_functions"` // Extract only modified function definitions instead of full files
}

func (s *SystemConfigService) GetFileContextConfig() *FileContextConfigResponse {
	maxFileSize, _ := strconv.Atoi(s.GetWithDefault("file_context_max_file_size", "102400"))
	maxFiles, _ := strconv.Atoi(s.GetWithDefault("file_context_max_files", "10"))
	return &FileContextConfigResponse{
		Enabled:          s.GetWithDefault("file_context_enabled", "false") == "true",
		MaxFileSize:      maxFileSize,
		MaxFiles:         maxFiles,
		ExtractFunctions: s.GetWithDefault("file_context_extract_functions", "true") == "true",
	}
}

type UpdateFileContextConfigRequest struct {
	Enabled          *bool `json:"enabled"`
	MaxFileSize      *int  `json:"max_file_size"`
	MaxFiles         *int  `json:"max_files"`
	ExtractFunctions *bool `json:"extract_functions"`
}

func (s *SystemConfigService) UpdateFileContextConfig(req *UpdateFileContextConfigRequest) error {
	if req.Enabled != nil {
		if err := s.Set("file_context_enabled", strconv.FormatBool(*req.Enabled)); err != nil {
			return err
		}
	}
	if req.MaxFileSize != nil {
		if err := s.Set("file_context_max_file_size", strconv.Itoa(*req.MaxFileSize)); err != nil {
			return err
		}
	}
	if req.MaxFiles != nil {
		if err := s.Set("file_context_max_files", strconv.Itoa(*req.MaxFiles)); err != nil {
			return err
		}
	}
	if req.ExtractFunctions != nil {
		if err := s.Set("file_context_extract_functions", strconv.FormatBool(*req.ExtractFunctions)); err != nil {
			return err
		}
	}
	return nil
}
