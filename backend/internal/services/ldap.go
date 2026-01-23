package services

import (
	"crypto/tls"
	"fmt"
	"strconv"

	"github.com/go-ldap/ldap/v3"
	"gorm.io/gorm"
)

type LDAPService struct {
	db *gorm.DB
}

func NewLDAPService(db *gorm.DB) *LDAPService {
	return &LDAPService{db: db}
}

type ldapConfig struct {
	Enabled      bool
	Host         string
	Port         int
	BaseDN       string
	BindDN       string
	BindPassword string
	UserFilter   string
	UseSSL       bool
}

func (s *LDAPService) getConfig() *ldapConfig {
	configService := NewSystemConfigService(s.db)
	port, _ := strconv.Atoi(configService.GetWithDefault("ldap_port", "389"))
	return &ldapConfig{
		Enabled:      configService.GetWithDefault("ldap_enabled", "false") == "true",
		Host:         configService.GetWithDefault("ldap_host", ""),
		Port:         port,
		BaseDN:       configService.GetWithDefault("ldap_base_dn", ""),
		BindDN:       configService.GetWithDefault("ldap_bind_dn", ""),
		BindPassword: configService.GetWithDefault("ldap_bind_password", ""),
		UserFilter:   configService.GetWithDefault("ldap_user_filter", "(uid=%s)"),
		UseSSL:       configService.GetWithDefault("ldap_use_ssl", "false") == "true",
	}
}

func (s *LDAPService) Authenticate(username, password string) (*LDAPUser, error) {
	cfg := s.getConfig()
	if !cfg.Enabled {
		return nil, fmt.Errorf("LDAP is not enabled")
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	var conn *ldap.Conn
	var err error

	if cfg.UseSSL {
		conn, err = ldap.DialTLS("tcp", addr, &tls.Config{InsecureSkipVerify: true})
	} else {
		conn, err = ldap.Dial("tcp", addr)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to LDAP server: %w", err)
	}
	defer conn.Close()

	if cfg.BindDN != "" {
		err = conn.Bind(cfg.BindDN, cfg.BindPassword)
		if err != nil {
			return nil, fmt.Errorf("failed to bind with service account: %w", err)
		}
	}

	searchFilter := fmt.Sprintf(cfg.UserFilter, ldap.EscapeFilter(username))
	searchRequest := ldap.NewSearchRequest(
		cfg.BaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		searchFilter,
		[]string{"dn", "cn", "mail", "uid", "sAMAccountName"},
		nil,
	)

	result, err := conn.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("LDAP search failed: %w", err)
	}

	if len(result.Entries) == 0 {
		return nil, fmt.Errorf("user not found in LDAP")
	}

	if len(result.Entries) > 1 {
		return nil, fmt.Errorf("multiple users found in LDAP")
	}

	userDN := result.Entries[0].DN

	err = conn.Bind(userDN, password)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	entry := result.Entries[0]
	user := &LDAPUser{
		DN:       userDN,
		Username: entry.GetAttributeValue("uid"),
		Email:    entry.GetAttributeValue("mail"),
		Nickname: entry.GetAttributeValue("cn"),
	}

	if user.Username == "" {
		user.Username = entry.GetAttributeValue("sAMAccountName")
	}

	return user, nil
}

func (s *LDAPService) IsEnabled() bool {
	cfg := s.getConfig()
	return cfg.Enabled
}

type LDAPUser struct {
	DN       string
	Username string
	Email    string
	Nickname string
}
