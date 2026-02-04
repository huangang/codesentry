package services

import (
	"strings"
	"testing"
)

func TestBuildMessage(t *testing.T) {
	service := &NotificationService{}

	tests := []struct {
		name          string
		notification  *ReviewNotification
		shouldContain []string
	}{
		{
			name: "high score",
			notification: &ReviewNotification{
				ProjectName:   "TestProject",
				Branch:        "main",
				Author:        "john",
				CommitMessage: "feat: add feature",
				Score:         90,
				ReviewResult:  "Code looks good",
				EventType:     "push",
			},
			shouldContain: []string{"🟢", "TestProject", "main", "john", "90/100", "Code looks good"},
		},
		{
			name: "medium score",
			notification: &ReviewNotification{
				ProjectName:   "TestProject",
				Branch:        "develop",
				Author:        "jane",
				CommitMessage: "fix: bug fix",
				Score:         70,
				ReviewResult:  "Some improvements needed",
				EventType:     "push",
			},
			shouldContain: []string{"🟡", "70/100"},
		},
		{
			name: "low score",
			notification: &ReviewNotification{
				ProjectName:   "TestProject",
				Branch:        "feature",
				Author:        "bob",
				CommitMessage: "wip",
				Score:         50,
				ReviewResult:  "Major issues",
				EventType:     "push",
			},
			shouldContain: []string{"🔴", "50/100"},
		},
		{
			name: "merge request event",
			notification: &ReviewNotification{
				ProjectName: "TestProject",
				Branch:      "main",
				Author:      "dev",
				Score:       85,
				EventType:   "merge_request",
				MRURL:       "https://gitlab.com/test/mr/1",
			},
			shouldContain: []string{"Merge Request", "View MR/PR", "https://gitlab.com/test/mr/1"},
		},
		{
			name: "long commit message truncated",
			notification: &ReviewNotification{
				ProjectName:   "TestProject",
				Branch:        "main",
				Author:        "dev",
				CommitMessage: strings.Repeat("a", 150),
				Score:         80,
				EventType:     "push",
			},
			shouldContain: []string{"..."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.buildMessage(tt.notification)
			for _, expected := range tt.shouldContain {
				if !strings.Contains(result, expected) {
					t.Errorf("buildMessage() should contain %q, got:\n%s", expected, result)
				}
			}
		})
	}
}

func TestSplitMessage(t *testing.T) {
	service := &NotificationService{}

	tests := []struct {
		name          string
		msg           string
		maxLen        int
		expectedParts int
	}{
		{
			name:          "short message no split",
			msg:           "short message",
			maxLen:        100,
			expectedParts: 1,
		},
		{
			name:          "exact length no split",
			msg:           "12345",
			maxLen:        5,
			expectedParts: 1,
		},
		{
			name:          "split into two parts",
			msg:           "1234567890",
			maxLen:        5,
			expectedParts: 2,
		},
		{
			name:          "split at newline",
			msg:           "line1\nline2\nline3",
			maxLen:        10,
			expectedParts: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts := service.splitMessage(tt.msg, tt.maxLen)
			if len(parts) != tt.expectedParts {
				t.Errorf("splitMessage() returned %d parts, expected %d", len(parts), tt.expectedParts)
			}
			for _, part := range parts {
				if len(part) > tt.maxLen && tt.expectedParts > 1 {
					t.Errorf("part length %d exceeds maxLen %d", len(part), tt.maxLen)
				}
			}
		})
	}
}

func TestSplitMessage_PreservesContent(t *testing.T) {
	service := &NotificationService{}
	original := "This is a test message that should be split into multiple parts for testing purposes."
	maxLen := 30

	parts := service.splitMessage(original, maxLen)

	reconstructed := strings.Join(parts, "")
	if reconstructed != original {
		t.Errorf("reconstructed message differs from original\noriginal: %q\nreconstructed: %q", original, reconstructed)
	}
}

func TestReviewNotification_Structure(t *testing.T) {
	n := &ReviewNotification{
		ProjectName:   "codesentry",
		Branch:        "main",
		Author:        "developer",
		CommitMessage: "feat: new feature",
		Score:         85.5,
		ReviewResult:  "Good code quality",
		EventType:     "push",
		MRURL:         "",
	}

	if n.ProjectName != "codesentry" {
		t.Errorf("ProjectName = %q, expected %q", n.ProjectName, "codesentry")
	}
	if n.Branch != "main" {
		t.Errorf("Branch = %q, expected %q", n.Branch, "main")
	}
	if n.Author != "developer" {
		t.Errorf("Author = %q, expected %q", n.Author, "developer")
	}
	if n.CommitMessage != "feat: new feature" {
		t.Errorf("CommitMessage = %q, expected %q", n.CommitMessage, "feat: new feature")
	}
	if n.Score != 85.5 {
		t.Errorf("Score = %f, expected %f", n.Score, 85.5)
	}
	if n.ReviewResult != "Good code quality" {
		t.Errorf("ReviewResult = %q, expected %q", n.ReviewResult, "Good code quality")
	}
	if n.EventType != "push" {
		t.Errorf("EventType = %q, expected %q", n.EventType, "push")
	}
	if n.MRURL != "" {
		t.Errorf("MRURL = %q, expected empty", n.MRURL)
	}
}

func TestDingTalkSign(t *testing.T) {
	service := &NotificationService{}

	timestamp := int64(1699999999999)
	secret := "testsecret"

	sign := service.dingTalkSign(timestamp, secret)

	if sign == "" {
		t.Error("dingTalkSign should not return empty string")
	}
	if len(sign) < 20 {
		t.Errorf("dingTalkSign result seems too short: %s", sign)
	}

	sign2 := service.dingTalkSign(timestamp, secret)
	if sign != sign2 {
		t.Error("dingTalkSign should be deterministic")
	}

	sign3 := service.dingTalkSign(timestamp, "different")
	if sign == sign3 {
		t.Error("different secrets should produce different signatures")
	}
}

func TestFeishuSign(t *testing.T) {
	service := &NotificationService{}

	timestamp := int64(1699999999)
	secret := "testsecret"

	sign := service.feishuSign(timestamp, secret)

	if sign == "" {
		t.Error("feishuSign should not return empty string")
	}

	sign2 := service.feishuSign(timestamp, secret)
	if sign != sign2 {
		t.Error("feishuSign should be deterministic")
	}

	sign3 := service.feishuSign(timestamp, "different")
	if sign == sign3 {
		t.Error("different secrets should produce different signatures")
	}
}
