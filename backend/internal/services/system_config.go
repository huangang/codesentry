package services

import (
	"strconv"

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
