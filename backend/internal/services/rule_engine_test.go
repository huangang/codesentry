package services

import (
	"testing"

	"github.com/huangang/codesentry/backend/internal/models"
)

func TestRuleEngineService_evaluateRule_ScoreBelow(t *testing.T) {
	svc := &RuleEngineService{}

	score := 45.0
	log := &models.ReviewLog{Score: &score, FilesChanged: 5}

	rule := &models.ReviewRule{
		ID:        1,
		Name:      "Low Score Block",
		Condition: "score_below",
		Threshold: 60,
		Action:    "block",
	}

	result := svc.evaluateRule(rule, log)
	if !result.Triggered {
		t.Error("Rule should be triggered for score 45 < 60")
	}
	if result.Action != "block" {
		t.Errorf("Action should be 'block', got %q", result.Action)
	}

	// Score above threshold — should NOT trigger
	score2 := 80.0
	log2 := &models.ReviewLog{Score: &score2}
	result2 := svc.evaluateRule(rule, log2)
	if result2.Triggered {
		t.Error("Rule should NOT be triggered for score 80 >= 60")
	}
}

func TestRuleEngineService_evaluateRule_FilesChangedAbove(t *testing.T) {
	svc := &RuleEngineService{}

	log := &models.ReviewLog{FilesChanged: 50}
	rule := &models.ReviewRule{
		ID:        2,
		Name:      "Too Many Files",
		Condition: "files_changed_above",
		Threshold: 20,
		Action:    "warn",
	}

	result := svc.evaluateRule(rule, log)
	if !result.Triggered {
		t.Error("Rule should be triggered for 50 files > 20")
	}

	log2 := &models.ReviewLog{FilesChanged: 10}
	result2 := svc.evaluateRule(rule, log2)
	if result2.Triggered {
		t.Error("Rule should NOT be triggered for 10 files <= 20")
	}
}

func TestRuleEngineService_evaluateRule_HasKeyword(t *testing.T) {
	svc := &RuleEngineService{}

	log := &models.ReviewLog{
		ReviewResult:  "Found potential SQL injection vulnerability",
		CommitMessage: "fix: update query",
	}

	rule := &models.ReviewRule{
		ID:        3,
		Name:      "Security Alert",
		Condition: "has_keyword",
		Keyword:   "SQL injection, XSS",
		Action:    "block",
	}

	result := svc.evaluateRule(rule, log)
	if !result.Triggered {
		t.Error("Rule should be triggered for keyword 'SQL injection'")
	}

	// No matching keyword
	log2 := &models.ReviewLog{
		ReviewResult:  "Code looks good",
		CommitMessage: "chore: cleanup",
	}
	result2 := svc.evaluateRule(rule, log2)
	if result2.Triggered {
		t.Error("Rule should NOT be triggered when no keywords match")
	}
}

func TestRuleEngineService_evaluateRule_AdditionsAbove(t *testing.T) {
	svc := &RuleEngineService{}

	log := &models.ReviewLog{Additions: 1500}
	rule := &models.ReviewRule{
		ID:        4,
		Name:      "Large Change",
		Condition: "additions_above",
		Threshold: 1000,
		Action:    "warn",
	}

	result := svc.evaluateRule(rule, log)
	if !result.Triggered {
		t.Error("Rule should be triggered for 1500 additions > 1000")
	}
}

func TestRuleEngineService_evaluateRule_NilScore(t *testing.T) {
	svc := &RuleEngineService{}

	log := &models.ReviewLog{Score: nil}
	rule := &models.ReviewRule{
		Condition: "score_below",
		Threshold: 60,
		Action:    "block",
	}

	result := svc.evaluateRule(rule, log)
	if result.Triggered {
		t.Error("Rule should NOT trigger when score is nil")
	}
}
