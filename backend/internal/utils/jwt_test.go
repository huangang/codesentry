package utils

import (
	"testing"
	"time"
)

func init() {
	SetJWTSecret("test-secret-key-for-testing")
}

func TestGenerateToken(t *testing.T) {
	token, err := GenerateToken(1, "testuser", "admin", 24)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	if token == "" {
		t.Error("GenerateToken() returned empty token")
	}

	if len(token) < 50 {
		t.Errorf("token seems too short: %d chars", len(token))
	}
}

func TestGenerateToken_DifferentTokens(t *testing.T) {
	token1, _ := GenerateToken(1, "user1", "admin", 24)
	token2, _ := GenerateToken(2, "user2", "user", 24)

	if token1 == token2 {
		t.Error("different users should produce different tokens")
	}
}

func TestParseToken(t *testing.T) {
	userID := uint(42)
	username := "testuser"
	role := "admin"

	token, _ := GenerateToken(userID, username, role, 24)

	claims, err := ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken() error = %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("UserID = %d, expected %d", claims.UserID, userID)
	}
	if claims.Username != username {
		t.Errorf("Username = %q, expected %q", claims.Username, username)
	}
	if claims.Role != role {
		t.Errorf("Role = %q, expected %q", claims.Role, role)
	}
}

func TestParseToken_InvalidToken(t *testing.T) {
	invalidTokens := []string{
		"",
		"invalid",
		"not.a.token",
		"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.invalid.signature",
	}

	for _, token := range invalidTokens {
		_, err := ParseToken(token)
		if err == nil {
			t.Errorf("ParseToken(%q) should return error", token)
		}
	}
}

func TestParseToken_WrongSecret(t *testing.T) {
	SetJWTSecret("original-secret")
	token, _ := GenerateToken(1, "user", "admin", 24)

	SetJWTSecret("different-secret")
	_, err := ParseToken(token)

	SetJWTSecret("test-secret-key-for-testing")

	if err == nil {
		t.Error("ParseToken should fail with wrong secret")
	}
}

func TestClaims_Structure(t *testing.T) {
	claims := Claims{
		UserID:   1,
		Username: "test",
		Role:     "admin",
	}

	if claims.UserID != 1 {
		t.Errorf("UserID = %d, expected 1", claims.UserID)
	}
	if claims.Username != "test" {
		t.Errorf("Username = %q, expected %q", claims.Username, "test")
	}
	if claims.Role != "admin" {
		t.Errorf("Role = %q, expected %q", claims.Role, "admin")
	}
}

func TestGenerateToken_Expiration(t *testing.T) {
	token, _ := GenerateToken(1, "user", "admin", 1)
	claims, _ := ParseToken(token)

	expiresAt := claims.ExpiresAt.Time
	now := time.Now()

	if expiresAt.Before(now) {
		t.Error("token should not be expired immediately")
	}

	expectedExpiry := now.Add(1 * time.Hour)
	diff := expiresAt.Sub(expectedExpiry)
	if diff < -time.Minute || diff > time.Minute {
		t.Errorf("expiration time is off by more than 1 minute: %v", diff)
	}
}

func TestSetJWTSecret(t *testing.T) {
	originalSecret := "original"
	newSecret := "new-secret"

	SetJWTSecret(originalSecret)
	token1, _ := GenerateToken(1, "user", "admin", 24)

	SetJWTSecret(newSecret)
	token2, _ := GenerateToken(1, "user", "admin", 24)

	SetJWTSecret("test-secret-key-for-testing")

	if token1 == token2 {
		t.Error("tokens generated with different secrets should be different")
	}
}
