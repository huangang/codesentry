package services

import (
	"fmt"
	"strings"
)

// repoInfo contains parsed repository information
type repoInfo struct {
	owner       string
	repo        string
	projectPath string
	baseURL     string
}

// parseRepoInfo extracts repository information from a project URL
func parseRepoInfo(projectURL string) (*repoInfo, error) {
	urlStr := strings.TrimSuffix(projectURL, ".git")

	protocolIdx := strings.Index(urlStr, "://")
	if protocolIdx == -1 {
		return nil, fmt.Errorf("invalid project URL (no protocol): %s", projectURL)
	}

	protocol := urlStr[:protocolIdx+3]
	rest := urlStr[protocolIdx+3:]

	slashIdx := strings.Index(rest, "/")
	if slashIdx == -1 {
		return nil, fmt.Errorf("invalid project URL (no path): %s", projectURL)
	}

	host := rest[:slashIdx]
	projectPath := rest[slashIdx+1:]

	if projectPath == "" {
		return nil, fmt.Errorf("invalid project URL (empty project path): %s", projectURL)
	}

	pathParts := strings.Split(projectPath, "/")
	if len(pathParts) < 2 {
		return nil, fmt.Errorf("invalid project URL (need at least owner/repo): %s", projectURL)
	}

	return &repoInfo{
		owner:       pathParts[len(pathParts)-2],
		repo:        pathParts[len(pathParts)-1],
		projectPath: projectPath,
		baseURL:     protocol + host,
	}, nil
}
