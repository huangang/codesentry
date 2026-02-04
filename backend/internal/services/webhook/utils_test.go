package webhook

import (
	"testing"
)

func TestParseRepoInfo(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		wantOwner   string
		wantRepo    string
		wantPath    string
		wantBaseURL string
		wantErr     bool
	}{
		{
			name:        "github https",
			url:         "https://github.com/owner/repo",
			wantOwner:   "owner",
			wantRepo:    "repo",
			wantPath:    "owner/repo",
			wantBaseURL: "https://github.com",
		},
		{
			name:        "github with .git suffix",
			url:         "https://github.com/owner/repo.git",
			wantOwner:   "owner",
			wantRepo:    "repo",
			wantPath:    "owner/repo",
			wantBaseURL: "https://github.com",
		},
		{
			name:        "gitlab nested groups",
			url:         "https://gitlab.com/group/subgroup/project",
			wantOwner:   "subgroup",
			wantRepo:    "project",
			wantPath:    "group/subgroup/project",
			wantBaseURL: "https://gitlab.com",
		},
		{
			name:        "self-hosted gitlab",
			url:         "https://git.company.com/team/app",
			wantOwner:   "team",
			wantRepo:    "app",
			wantPath:    "team/app",
			wantBaseURL: "https://git.company.com",
		},
		{
			name:        "bitbucket",
			url:         "https://bitbucket.org/workspace/repo",
			wantOwner:   "workspace",
			wantRepo:    "repo",
			wantPath:    "workspace/repo",
			wantBaseURL: "https://bitbucket.org",
		},
		{
			name:    "no protocol",
			url:     "github.com/owner/repo",
			wantErr: true,
		},
		{
			name:    "no path",
			url:     "https://github.com",
			wantErr: true,
		},
		{
			name:    "only owner no repo",
			url:     "https://github.com/owner",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := parseRepoInfo(tt.url)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if info.owner != tt.wantOwner {
				t.Errorf("owner = %q, want %q", info.owner, tt.wantOwner)
			}
			if info.repo != tt.wantRepo {
				t.Errorf("repo = %q, want %q", info.repo, tt.wantRepo)
			}
			if info.projectPath != tt.wantPath {
				t.Errorf("projectPath = %q, want %q", info.projectPath, tt.wantPath)
			}
			if info.baseURL != tt.wantBaseURL {
				t.Errorf("baseURL = %q, want %q", info.baseURL, tt.wantBaseURL)
			}
		})
	}
}

func TestIsEmptyDiff(t *testing.T) {
	tests := []struct {
		name     string
		diff     string
		expected bool
	}{
		{
			name:     "empty string",
			diff:     "",
			expected: true,
		},
		{
			name:     "whitespace only",
			diff:     "   \n\t\n  ",
			expected: true,
		},
		{
			name:     "only commit headers",
			diff:     "### Commit: abc123\n### Commit: def456\n",
			expected: true,
		},
		{
			name:     "has actual diff",
			diff:     "### Commit: abc123\n+added line\n-removed line",
			expected: false,
		},
		{
			name:     "normal diff",
			diff:     "diff --git a/file.go b/file.go\n+new line",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsEmptyDiff(tt.diff)
			if result != tt.expected {
				t.Errorf("IsEmptyDiff() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestVerifyGitLabSignature(t *testing.T) {
	tests := []struct {
		name     string
		secret   string
		token    string
		expected bool
	}{
		{"matching", "secret123", "secret123", true},
		{"not matching", "secret123", "wrong", false},
		{"empty both", "", "", true},
		{"empty secret", "", "token", false},
		{"empty token", "secret", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := VerifyGitLabSignature(tt.secret, tt.token)
			if result != tt.expected {
				t.Errorf("VerifyGitLabSignature() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestVerifyGitHubSignature(t *testing.T) {
	tests := []struct {
		name      string
		secret    string
		body      string
		signature string
		expected  bool
	}{
		{
			name:      "valid signature",
			secret:    "mysecret",
			body:      `{"test":"data"}`,
			signature: "sha256=5d5f632dc1a57a7dbf8e8e7c71e1c6e3a4b9f8c2e1d0a9b8c7d6e5f4a3b2c1d0",
			expected:  false,
		},
		{
			name:      "wrong prefix",
			secret:    "mysecret",
			body:      `{"test":"data"}`,
			signature: "sha1=abc123",
			expected:  false,
		},
		{
			name:      "no prefix",
			secret:    "mysecret",
			body:      `{"test":"data"}`,
			signature: "abc123",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := VerifyGitHubSignature(tt.secret, []byte(tt.body), tt.signature)
			if result != tt.expected {
				t.Errorf("VerifyGitHubSignature() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestVerifyBitbucketSignature(t *testing.T) {
	tests := []struct {
		name      string
		secret    string
		body      string
		signature string
		expected  bool
	}{
		{
			name:      "empty secret allows all",
			secret:    "",
			body:      "any body",
			signature: "any",
			expected:  true,
		},
		{
			name:      "wrong signature",
			secret:    "mysecret",
			body:      "test body",
			signature: "wronghash",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := VerifyBitbucketSignature(tt.secret, []byte(tt.body), tt.signature)
			if result != tt.expected {
				t.Errorf("VerifyBitbucketSignature() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestParseDiffStats(t *testing.T) {
	tests := []struct {
		name             string
		diff             string
		wantAdditions    int
		wantDeletions    int
		wantFilesChanged int
	}{
		{
			name:             "empty diff",
			diff:             "",
			wantAdditions:    0,
			wantDeletions:    0,
			wantFilesChanged: 0,
		},
		{
			name: "single file",
			diff: `diff --git a/file.go b/file.go
--- a/file.go
+++ b/file.go
+added line 1
+added line 2
-removed line
`,
			wantAdditions:    2,
			wantDeletions:    1,
			wantFilesChanged: 1,
		},
		{
			name: "multiple files",
			diff: `diff --git a/a.go b/a.go
+line
diff --git a/b.go b/b.go
+line
-line
`,
			wantAdditions:    2,
			wantDeletions:    1,
			wantFilesChanged: 2,
		},
		{
			name: "ignore header lines",
			diff: `diff --git a/file.go b/file.go
--- a/file.go
+++ b/file.go
+real addition
`,
			wantAdditions:    1,
			wantDeletions:    0,
			wantFilesChanged: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			additions, deletions, filesChanged := ParseDiffStats(tt.diff)
			if additions != tt.wantAdditions {
				t.Errorf("additions = %d, want %d", additions, tt.wantAdditions)
			}
			if deletions != tt.wantDeletions {
				t.Errorf("deletions = %d, want %d", deletions, tt.wantDeletions)
			}
			if filesChanged != tt.wantFilesChanged {
				t.Errorf("filesChanged = %d, want %d", filesChanged, tt.wantFilesChanged)
			}
		})
	}
}

func TestDefaultIgnorePatterns(t *testing.T) {
	patterns := DefaultIgnorePatterns

	expectedPatterns := []string{
		"*.json",
		"*.yaml",
		"*.lock",
		"package-lock.json",
		"go.sum",
		"node_modules/",
		"vendor/",
		"dist/",
	}

	for _, expected := range expectedPatterns {
		if !containsPattern(patterns, expected) {
			t.Errorf("DefaultIgnorePatterns should contain %q", expected)
		}
	}
}

func containsPattern(patterns, pattern string) bool {
	for i := 0; i <= len(patterns)-len(pattern); i++ {
		if patterns[i:i+len(pattern)] == pattern {
			return true
		}
	}
	return false
}
