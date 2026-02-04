package services

import (
	"testing"
)

func TestPromptListParams_Structure(t *testing.T) {
	isSystem := true
	params := PromptListParams{
		Page:     1,
		PageSize: 20,
		Name:     "default",
		IsSystem: &isSystem,
	}

	if params.Page != 1 {
		t.Errorf("Page = %d, expected 1", params.Page)
	}
	if params.PageSize != 20 {
		t.Errorf("PageSize = %d, expected 20", params.PageSize)
	}
	if params.Name != "default" {
		t.Errorf("Name = %q, expected %q", params.Name, "default")
	}
	if params.IsSystem == nil || *params.IsSystem != true {
		t.Error("IsSystem should be true")
	}
}

func TestPromptListParams_NilIsSystem(t *testing.T) {
	params := PromptListParams{
		Page:     1,
		PageSize: 10,
	}

	if params.IsSystem != nil {
		t.Error("IsSystem should be nil by default")
	}
	if params.Page != 1 {
		t.Errorf("Page = %d, expected 1", params.Page)
	}
	if params.PageSize != 10 {
		t.Errorf("PageSize = %d, expected 10", params.PageSize)
	}
}

func TestPromptListResult_Structure(t *testing.T) {
	result := &PromptListResult{
		Items: nil,
		Total: 5,
	}

	if result.Total != 5 {
		t.Errorf("Total = %d, expected 5", result.Total)
	}
	if result.Items != nil {
		t.Error("Items should be nil")
	}
}
