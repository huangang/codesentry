package services

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
			name:        "deeply nested gitlab",
			url:         "https://gitlab.com/a/b/c/d/repo",
			wantOwner:   "d",
			wantRepo:    "repo",
			wantPath:    "a/b/c/d/repo",
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
			name:        "http protocol",
			url:         "http://gitlab.local/team/project",
			wantOwner:   "team",
			wantRepo:    "project",
			wantPath:    "team/project",
			wantBaseURL: "http://gitlab.local",
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
			name:    "only slash",
			url:     "https://github.com/",
			wantErr: true,
		},
		{
			name:    "only owner no repo",
			url:     "https://github.com/owner",
			wantErr: true,
		},
		{
			name:    "empty url",
			url:     "",
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

func TestRepoInfo_Structure(t *testing.T) {
	info := &repoInfo{
		owner:       "myorg",
		repo:        "myrepo",
		projectPath: "myorg/myrepo",
		baseURL:     "https://github.com",
	}

	if info.owner != "myorg" {
		t.Errorf("owner = %q, expected %q", info.owner, "myorg")
	}
	if info.repo != "myrepo" {
		t.Errorf("repo = %q, expected %q", info.repo, "myrepo")
	}
	if info.projectPath != "myorg/myrepo" {
		t.Errorf("projectPath = %q, expected %q", info.projectPath, "myorg/myrepo")
	}
	if info.baseURL != "https://github.com" {
		t.Errorf("baseURL = %q, expected %q", info.baseURL, "https://github.com")
	}
}
