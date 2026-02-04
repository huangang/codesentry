package utils

import (
	"testing"
)

func TestHashPassword(t *testing.T) {
	password := "testpassword123"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	if hash == "" {
		t.Error("HashPassword() returned empty string")
	}

	if hash == password {
		t.Error("HashPassword() should not return plaintext password")
	}

	if len(hash) < 50 {
		t.Errorf("hash seems too short: %d chars", len(hash))
	}
}

func TestHashPassword_DifferentHashes(t *testing.T) {
	password := "testpassword"

	hash1, _ := HashPassword(password)
	hash2, _ := HashPassword(password)

	if hash1 == hash2 {
		t.Error("same password should produce different hashes (due to salt)")
	}
}

func TestCheckPassword(t *testing.T) {
	password := "correctpassword"
	hash, _ := HashPassword(password)

	tests := []struct {
		name     string
		password string
		expected bool
	}{
		{"correct password", "correctpassword", true},
		{"wrong password", "wrongpassword", false},
		{"empty password", "", false},
		{"similar password", "correctpassword1", false},
		{"case sensitive", "CorrectPassword", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CheckPassword(tt.password, hash)
			if result != tt.expected {
				t.Errorf("CheckPassword(%q) = %v, expected %v", tt.password, result, tt.expected)
			}
		})
	}
}

func TestCheckPassword_InvalidHash(t *testing.T) {
	result := CheckPassword("password", "invalid_hash")
	if result {
		t.Error("CheckPassword should return false for invalid hash")
	}
}

func TestCheckPassword_EmptyHash(t *testing.T) {
	result := CheckPassword("password", "")
	if result {
		t.Error("CheckPassword should return false for empty hash")
	}
}
