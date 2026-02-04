package services

import (
	"testing"
)

func TestDashboardStatsRequest_Defaults(t *testing.T) {
	req := &DashboardStatsRequest{}

	if req.StartDate != "" {
		t.Errorf("StartDate should be empty by default, got %q", req.StartDate)
	}
	if req.EndDate != "" {
		t.Errorf("EndDate should be empty by default, got %q", req.EndDate)
	}
	if req.ProjectLimit != 0 {
		t.Errorf("ProjectLimit should be 0 by default, got %d", req.ProjectLimit)
	}
	if req.AuthorLimit != 0 {
		t.Errorf("AuthorLimit should be 0 by default, got %d", req.AuthorLimit)
	}
}

func TestDashboardStatsRequest_WithValues(t *testing.T) {
	req := &DashboardStatsRequest{
		StartDate:    "2024-01-01",
		EndDate:      "2024-01-31",
		ProjectLimit: 5,
		AuthorLimit:  10,
	}

	if req.StartDate != "2024-01-01" {
		t.Errorf("StartDate = %q, expected %q", req.StartDate, "2024-01-01")
	}
	if req.EndDate != "2024-01-31" {
		t.Errorf("EndDate = %q, expected %q", req.EndDate, "2024-01-31")
	}
	if req.ProjectLimit != 5 {
		t.Errorf("ProjectLimit = %d, expected 5", req.ProjectLimit)
	}
	if req.AuthorLimit != 10 {
		t.Errorf("AuthorLimit = %d, expected 10", req.AuthorLimit)
	}
}

func TestDashboardStats_Structure(t *testing.T) {
	stats := DashboardStats{
		ActiveProjects: 10,
		Contributors:   25,
		TotalCommits:   150,
		AverageScore:   78.5,
	}

	if stats.ActiveProjects != 10 {
		t.Errorf("ActiveProjects = %d, expected 10", stats.ActiveProjects)
	}
	if stats.Contributors != 25 {
		t.Errorf("Contributors = %d, expected 25", stats.Contributors)
	}
	if stats.TotalCommits != 150 {
		t.Errorf("TotalCommits = %d, expected 150", stats.TotalCommits)
	}
	if stats.AverageScore != 78.5 {
		t.Errorf("AverageScore = %f, expected 78.5", stats.AverageScore)
	}
}

func TestProjectStats_Structure(t *testing.T) {
	stats := ProjectStats{
		ProjectID:   1,
		ProjectName: "codesentry",
		CommitCount: 50,
		AvgScore:    85.0,
		Additions:   1000,
		Deletions:   200,
	}

	if stats.ProjectID != 1 {
		t.Errorf("ProjectID = %d, expected 1", stats.ProjectID)
	}
	if stats.ProjectName != "codesentry" {
		t.Errorf("ProjectName = %q, expected %q", stats.ProjectName, "codesentry")
	}
	if stats.CommitCount != 50 {
		t.Errorf("CommitCount = %d, expected 50", stats.CommitCount)
	}
	if stats.AvgScore != 85.0 {
		t.Errorf("AvgScore = %f, expected 85.0", stats.AvgScore)
	}
	if stats.Additions != 1000 {
		t.Errorf("Additions = %d, expected 1000", stats.Additions)
	}
	if stats.Deletions != 200 {
		t.Errorf("Deletions = %d, expected 200", stats.Deletions)
	}
}

func TestAuthorStats_Structure(t *testing.T) {
	stats := AuthorStats{
		Author:      "developer",
		CommitCount: 30,
		AvgScore:    90.5,
		Additions:   500,
		Deletions:   100,
	}

	if stats.Author != "developer" {
		t.Errorf("Author = %q, expected %q", stats.Author, "developer")
	}
	if stats.CommitCount != 30 {
		t.Errorf("CommitCount = %d, expected 30", stats.CommitCount)
	}
	if stats.AvgScore != 90.5 {
		t.Errorf("AvgScore = %f, expected 90.5", stats.AvgScore)
	}
	if stats.Additions != 500 {
		t.Errorf("Additions = %d, expected 500", stats.Additions)
	}
	if stats.Deletions != 100 {
		t.Errorf("Deletions = %d, expected 100", stats.Deletions)
	}
}

func TestDashboardResponse_Structure(t *testing.T) {
	resp := DashboardResponse{
		Stats: DashboardStats{
			ActiveProjects: 5,
			TotalCommits:   100,
		},
		ProjectStats: []ProjectStats{
			{ProjectID: 1, ProjectName: "proj1"},
		},
		AuthorStats: []AuthorStats{
			{Author: "dev1"},
		},
	}

	if resp.Stats.ActiveProjects != 5 {
		t.Errorf("Stats.ActiveProjects = %d, expected 5", resp.Stats.ActiveProjects)
	}
	if len(resp.ProjectStats) != 1 {
		t.Errorf("ProjectStats length = %d, expected 1", len(resp.ProjectStats))
	}
	if len(resp.AuthorStats) != 1 {
		t.Errorf("AuthorStats length = %d, expected 1", len(resp.AuthorStats))
	}
}

func TestDashboardStatsRequest_DateFormat(t *testing.T) {
	validDates := []string{
		"2024-01-01",
		"2024-12-31",
		"2023-06-15",
	}

	for _, date := range validDates {
		req := &DashboardStatsRequest{
			StartDate: date,
		}

		if len(req.StartDate) != 10 {
			t.Errorf("date %q should be 10 characters", date)
		}
	}
}
