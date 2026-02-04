package services

import (
	"testing"
)

func TestLoginRequest_Structure(t *testing.T) {
	req := LoginRequest{
		Username: "testuser",
		Password: "password123",
		AuthType: "local",
	}

	if req.Username != "testuser" {
		t.Errorf("Username = %q, expected %q", req.Username, "testuser")
	}
	if req.Password != "password123" {
		t.Errorf("Password = %q, expected %q", req.Password, "password123")
	}
	if req.AuthType != "local" {
		t.Errorf("AuthType = %q, expected %q", req.AuthType, "local")
	}
}

func TestLoginRequest_DefaultAuthType(t *testing.T) {
	req := LoginRequest{
		Username: "user",
		Password: "pass",
	}

	if req.AuthType != "" {
		t.Errorf("AuthType should be empty by default, got %q", req.AuthType)
	}
	if req.Username != "user" {
		t.Errorf("Username = %q, expected %q", req.Username, "user")
	}
	if req.Password != "pass" {
		t.Errorf("Password = %q, expected %q", req.Password, "pass")
	}
}

func TestLoginRequest_LDAPAuthType(t *testing.T) {
	req := LoginRequest{
		Username: "ldapuser",
		Password: "ldappass",
		AuthType: "ldap",
	}

	if req.AuthType != "ldap" {
		t.Errorf("AuthType = %q, expected %q", req.AuthType, "ldap")
	}
	if req.Username != "ldapuser" {
		t.Errorf("Username = %q, expected %q", req.Username, "ldapuser")
	}
	if req.Password != "ldappass" {
		t.Errorf("Password = %q, expected %q", req.Password, "ldappass")
	}
}

func TestLoginResponse_Structure(t *testing.T) {
	resp := LoginResponse{
		Token: "jwt.token.here",
		User:  nil,
	}

	if resp.Token != "jwt.token.here" {
		t.Errorf("Token = %q, expected %q", resp.Token, "jwt.token.here")
	}
	if resp.User != nil {
		t.Error("User should be nil")
	}
}

func TestChangePasswordRequest_Structure(t *testing.T) {
	req := ChangePasswordRequest{
		OldPassword: "oldpass",
		NewPassword: "newpass123",
	}

	if req.OldPassword != "oldpass" {
		t.Errorf("OldPassword = %q, expected %q", req.OldPassword, "oldpass")
	}
	if req.NewPassword != "newpass123" {
		t.Errorf("NewPassword = %q, expected %q", req.NewPassword, "newpass123")
	}
}

func TestChangePasswordRequest_MinLength(t *testing.T) {
	req := ChangePasswordRequest{
		OldPassword: "old",
		NewPassword: "123456",
	}

	if len(req.NewPassword) < 6 {
		t.Errorf("NewPassword length should be at least 6, got %d", len(req.NewPassword))
	}
	if req.OldPassword != "old" {
		t.Errorf("OldPassword = %q, expected %q", req.OldPassword, "old")
	}
}
