package services

import (
	"testing"
	"time"
)

func TestReviewLogListRequest_Defaults(t *testing.T) {
	req := &ReviewLogListRequest{}

	if req.Page != 0 {
		t.Errorf("default Page should be 0, got %d", req.Page)
	}
	if req.PageSize != 0 {
		t.Errorf("default PageSize should be 0, got %d", req.PageSize)
	}
}

func TestReviewLogListRequest_WithFilters(t *testing.T) {
	now := time.Now()
	req := &ReviewLogListRequest{
		Page:       3,
		PageSize:   50,
		EventType:  "push",
		ProjectID:  42,
		Author:     "john",
		StartDate:  now.Add(-24 * time.Hour),
		EndDate:    now,
		SearchText: "feature",
	}

	if req.Page != 3 {
		t.Errorf("Page = %d, expected 3", req.Page)
	}
	if req.PageSize != 50 {
		t.Errorf("PageSize = %d, expected 50", req.PageSize)
	}
	if req.EventType != "push" {
		t.Errorf("EventType = %q, expected %q", req.EventType, "push")
	}
	if req.ProjectID != 42 {
		t.Errorf("ProjectID = %d, expected 42", req.ProjectID)
	}
	if req.Author != "john" {
		t.Errorf("Author = %q, expected %q", req.Author, "john")
	}
	if req.SearchText != "feature" {
		t.Errorf("SearchText = %q, expected %q", req.SearchText, "feature")
	}
	if req.StartDate.IsZero() {
		t.Error("StartDate should not be zero")
	}
	if req.EndDate.IsZero() {
		t.Error("EndDate should not be zero")
	}
}

func TestReviewLogListResponse_Structure(t *testing.T) {
	resp := &ReviewLogListResponse{
		Total:    100,
		Page:     1,
		PageSize: 10,
		Items:    nil,
	}

	if resp.Total != 100 {
		t.Errorf("Total = %d, expected 100", resp.Total)
	}
	if resp.Page != 1 {
		t.Errorf("Page = %d, expected 1", resp.Page)
	}
	if resp.PageSize != 10 {
		t.Errorf("PageSize = %d, expected 10", resp.PageSize)
	}
	if resp.Items != nil {
		t.Error("Items should be nil")
	}
}

func TestReviewLogListRequest_DateRange(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

	req := &ReviewLogListRequest{
		StartDate: start,
		EndDate:   end,
	}

	if req.StartDate.IsZero() {
		t.Error("StartDate should not be zero")
	}
	if req.EndDate.IsZero() {
		t.Error("EndDate should not be zero")
	}
	if req.EndDate.Before(req.StartDate) {
		t.Error("EndDate should be after StartDate")
	}
}

func TestReviewLogListRequest_EmptyDates(t *testing.T) {
	req := &ReviewLogListRequest{}

	if !req.StartDate.IsZero() {
		t.Error("StartDate should be zero by default")
	}
	if !req.EndDate.IsZero() {
		t.Error("EndDate should be zero by default")
	}
}
