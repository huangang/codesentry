package services

import (
	"testing"
)

func TestIMBotListRequest_Defaults(t *testing.T) {
	req := &IMBotListRequest{}

	if req.Page != 0 {
		t.Errorf("default Page should be 0, got %d", req.Page)
	}
	if req.PageSize != 0 {
		t.Errorf("default PageSize should be 0, got %d", req.PageSize)
	}
}

func TestIMBotListRequest_WithValues(t *testing.T) {
	active := true
	req := &IMBotListRequest{
		Page:     2,
		PageSize: 20,
		Name:     "test",
		Type:     "slack",
		IsActive: &active,
	}

	if req.Page != 2 {
		t.Errorf("Page = %d, expected 2", req.Page)
	}
	if req.PageSize != 20 {
		t.Errorf("PageSize = %d, expected 20", req.PageSize)
	}
	if req.Name != "test" {
		t.Errorf("Name = %q, expected %q", req.Name, "test")
	}
	if req.Type != "slack" {
		t.Errorf("Type = %q, expected %q", req.Type, "slack")
	}
	if req.IsActive == nil || *req.IsActive != true {
		t.Error("IsActive should be true")
	}
}

func TestIMBotListResponse_Structure(t *testing.T) {
	resp := &IMBotListResponse{
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

func TestCreateIMBotRequest_AllFields(t *testing.T) {
	req := &CreateIMBotRequest{
		Name:               "My Bot",
		Type:               "slack",
		Webhook:            "https://hooks.slack.com/xxx",
		Secret:             "secret123",
		Extra:              "extra data",
		IsActive:           true,
		ErrorNotify:        true,
		DailyReportEnabled: false,
	}

	if req.Name != "My Bot" {
		t.Errorf("Name = %q, expected %q", req.Name, "My Bot")
	}
	if req.Type != "slack" {
		t.Errorf("Type = %q, expected %q", req.Type, "slack")
	}
	if req.Webhook != "https://hooks.slack.com/xxx" {
		t.Errorf("Webhook = %q, expected %q", req.Webhook, "https://hooks.slack.com/xxx")
	}
	if req.Secret != "secret123" {
		t.Errorf("Secret = %q, expected %q", req.Secret, "secret123")
	}
	if req.Extra != "extra data" {
		t.Errorf("Extra = %q, expected %q", req.Extra, "extra data")
	}
	if !req.IsActive {
		t.Error("IsActive should be true")
	}
	if !req.ErrorNotify {
		t.Error("ErrorNotify should be true")
	}
	if req.DailyReportEnabled {
		t.Error("DailyReportEnabled should be false")
	}
}

func TestUpdateIMBotRequest_PartialUpdate(t *testing.T) {
	active := false
	errorNotify := true

	req := &UpdateIMBotRequest{
		Name:        "Updated Name",
		IsActive:    &active,
		ErrorNotify: &errorNotify,
	}

	if req.Name != "Updated Name" {
		t.Errorf("Name = %q, expected %q", req.Name, "Updated Name")
	}
	if req.Type != "" {
		t.Errorf("Type should be empty, got %q", req.Type)
	}
	if req.IsActive == nil || *req.IsActive != false {
		t.Error("IsActive should be false")
	}
	if req.ErrorNotify == nil || *req.ErrorNotify != true {
		t.Error("ErrorNotify should be true")
	}
	if req.DailyReportEnabled != nil {
		t.Error("DailyReportEnabled should be nil")
	}
}

func TestIMBotTypes(t *testing.T) {
	validTypes := []string{
		"wechat_work",
		"dingtalk",
		"feishu",
		"slack",
		"discord",
		"teams",
		"telegram",
	}

	for _, botType := range validTypes {
		req := &CreateIMBotRequest{
			Name:    "Test Bot",
			Type:    botType,
			Webhook: "https://example.com/webhook",
		}

		if req.Type != botType {
			t.Errorf("Type = %q, expected %q", req.Type, botType)
		}
		if req.Name != "Test Bot" {
			t.Errorf("Name = %q, expected %q", req.Name, "Test Bot")
		}
		if req.Webhook != "https://example.com/webhook" {
			t.Errorf("Webhook = %q, expected %q", req.Webhook, "https://example.com/webhook")
		}
	}
}
