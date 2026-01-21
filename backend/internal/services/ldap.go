package services

import (
	"crypto/tls"
	"fmt"

	"github.com/huangang/codesentry/backend/internal/config"
	"github.com/go-ldap/ldap/v3"
)

type LDAPService struct {
	config *config.LDAPConfig
}

func NewLDAPService(cfg *config.LDAPConfig) *LDAPService {
	return &LDAPService{config: cfg}
}

// Authenticate authenticates a user against LDAP
func (s *LDAPService) Authenticate(username, password string) (*LDAPUser, error) {
	if !s.config.Enabled {
		return nil, fmt.Errorf("LDAP is not enabled")
	}

	// Connect to LDAP server
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	var conn *ldap.Conn
	var err error

	if s.config.UseSSL {
		conn, err = ldap.DialTLS("tcp", addr, &tls.Config{InsecureSkipVerify: true})
	} else {
		conn, err = ldap.Dial("tcp", addr)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to LDAP server: %w", err)
	}
	defer conn.Close()

	// Bind with service account (if configured)
	if s.config.BindDN != "" {
		err = conn.Bind(s.config.BindDN, s.config.BindPassword)
		if err != nil {
			return nil, fmt.Errorf("failed to bind with service account: %w", err)
		}
	}

	// Search for user
	searchFilter := fmt.Sprintf(s.config.UserFilter, ldap.EscapeFilter(username))
	searchRequest := ldap.NewSearchRequest(
		s.config.BaseDN,
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

	// Bind as user to verify password
	err = conn.Bind(userDN, password)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Extract user info
	entry := result.Entries[0]
	user := &LDAPUser{
		DN:       userDN,
		Username: entry.GetAttributeValue("uid"),
		Email:    entry.GetAttributeValue("mail"),
		Nickname: entry.GetAttributeValue("cn"),
	}

	// Try sAMAccountName if uid is empty (Active Directory)
	if user.Username == "" {
		user.Username = entry.GetAttributeValue("sAMAccountName")
	}

	return user, nil
}

type LDAPUser struct {
	DN       string
	Username string
	Email    string
	Nickname string
}
