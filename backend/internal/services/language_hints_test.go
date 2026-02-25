package services

import (
	"strings"
	"testing"
)

func TestDetectLanguagesFromDiff(t *testing.T) {
	tests := []struct {
		name     string
		diffs    string
		expected []string
	}{
		{
			name:     "Go files",
			diffs:    "diff --git a/main.go b/main.go\n+++ b/main.go\n@@ -1 +1 @@\n-old\n+new",
			expected: []string{"go"},
		},
		{
			name:     "Python files",
			diffs:    "diff --git a/app.py b/app.py\n+++ b/app.py",
			expected: []string{"python"},
		},
		{
			name:     "TypeScript files",
			diffs:    "diff --git a/src/App.tsx b/src/App.tsx\n+++ b/src/App.tsx",
			expected: []string{"typescript"},
		},
		{
			name:     "Multiple languages",
			diffs:    "diff --git a/main.go b/main.go\n+++ b/main.go\ndiff --git a/test.py b/test.py\n+++ b/test.py",
			expected: []string{"go", "python"},
		},
		{
			name:     "No diff",
			diffs:    "",
			expected: nil,
		},
		{
			name:     "Unknown extension",
			diffs:    "diff --git a/data.csv b/data.csv\n+++ b/data.csv",
			expected: nil,
		},
		{
			name:     "Rust files",
			diffs:    "diff --git a/lib.rs b/lib.rs\n+++ b/lib.rs",
			expected: []string{"rust"},
		},
		{
			name:     "Java files",
			diffs:    "diff --git a/Main.java b/Main.java\n+++ b/Main.java",
			expected: []string{"java"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectLanguagesFromDiff(tt.diffs)
			if len(result) != len(tt.expected) {
				t.Errorf("DetectLanguagesFromDiff() = %v, want %v", result, tt.expected)
				return
			}
			for i, lang := range result {
				if lang != tt.expected[i] {
					t.Errorf("DetectLanguagesFromDiff()[%d] = %q, want %q", i, lang, tt.expected[i])
				}
			}
		})
	}
}

func TestGenerateLanguageHints(t *testing.T) {
	tests := []struct {
		name         string
		diffs        string
		wantEmpty    bool
		wantContains string
	}{
		{
			name:      "Empty diff",
			diffs:     "",
			wantEmpty: true,
		},
		{
			name:         "Go diff has Go hints",
			diffs:        "diff --git a/main.go b/main.go\n+++ b/main.go",
			wantEmpty:    false,
			wantContains: "Go-specific checks",
		},
		{
			name:         "Python diff has Python hints",
			diffs:        "diff --git a/app.py b/app.py\n+++ b/app.py",
			wantEmpty:    false,
			wantContains: "Python-specific checks",
		},
		{
			name:      "Unknown ext produces no hints",
			diffs:     "diff --git a/data.csv b/data.csv\n+++ b/data.csv",
			wantEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateLanguageHints(tt.diffs)
			if tt.wantEmpty && result != "" {
				t.Errorf("GenerateLanguageHints() should be empty, got %q", result)
			}
			if !tt.wantEmpty && result == "" {
				t.Errorf("GenerateLanguageHints() should not be empty")
			}
			if tt.wantContains != "" && !strings.Contains(result, tt.wantContains) {
				t.Errorf("GenerateLanguageHints() should contain %q", tt.wantContains)
			}
		})
	}
}
