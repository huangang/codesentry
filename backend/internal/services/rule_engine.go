package services

import (
	"strings"

	"github.com/huangang/codesentry/backend/internal/models"
	"github.com/huangang/codesentry/backend/pkg/logger"
	"gorm.io/gorm"
)

// RuleEngineService evaluates review rules and returns enforcement decisions.
type RuleEngineService struct {
	db *gorm.DB
}

func NewRuleEngineService(db *gorm.DB) *RuleEngineService {
	return &RuleEngineService{db: db}
}

// RuleResult represents the outcome of evaluating a single rule.
type RuleResult struct {
	RuleID      uint   `json:"rule_id"`
	RuleName    string `json:"rule_name"`
	Condition   string `json:"condition"`
	Action      string `json:"action"`
	ActionValue string `json:"action_value"`
	Triggered   bool   `json:"triggered"`
	Message     string `json:"message"`
}

// EvaluationResult is the aggregate result of all rule evaluations for a review.
type EvaluationResult struct {
	Blocked  bool         `json:"blocked"`
	Warnings []string     `json:"warnings"`
	Results  []RuleResult `json:"results"`
}

// Evaluate runs all applicable rules against a review log and returns the combined result.
func (s *RuleEngineService) Evaluate(reviewLog *models.ReviewLog) *EvaluationResult {
	var rules []models.ReviewRule
	s.db.Where("is_active = ? AND (project_id IS NULL OR project_id = ?)", true, reviewLog.ProjectID).
		Order("priority DESC").Find(&rules)

	result := &EvaluationResult{}

	for _, rule := range rules {
		rr := s.evaluateRule(&rule, reviewLog)
		result.Results = append(result.Results, rr)

		if rr.Triggered {
			logger.Infof("[RuleEngine] Rule '%s' triggered for review %d: %s", rule.Name, reviewLog.ID, rr.Message)
			switch rule.Action {
			case "block":
				result.Blocked = true
				result.Warnings = append(result.Warnings, rr.Message)
			case "warn":
				result.Warnings = append(result.Warnings, rr.Message)
			}
		}
	}

	return result
}

func (s *RuleEngineService) evaluateRule(rule *models.ReviewRule, log *models.ReviewLog) RuleResult {
	rr := RuleResult{
		RuleID:      rule.ID,
		RuleName:    rule.Name,
		Condition:   rule.Condition,
		Action:      rule.Action,
		ActionValue: rule.ActionValue,
	}

	switch rule.Condition {
	case "score_below":
		if log.Score != nil && *log.Score < rule.Threshold {
			rr.Triggered = true
			rr.Message = "Review score below threshold"
		}

	case "files_changed_above":
		if log.FilesChanged > int(rule.Threshold) {
			rr.Triggered = true
			rr.Message = "Too many files changed"
		}

	case "has_keyword":
		if rule.Keyword != "" {
			keywords := strings.Split(rule.Keyword, ",")
			for _, kw := range keywords {
				kw = strings.TrimSpace(kw)
				if kw != "" && (strings.Contains(strings.ToLower(log.ReviewResult), strings.ToLower(kw)) ||
					strings.Contains(strings.ToLower(log.CommitMessage), strings.ToLower(kw))) {
					rr.Triggered = true
					rr.Message = "Found keyword: " + kw
					break
				}
			}
		}

	case "additions_above":
		if log.Additions > int(rule.Threshold) {
			rr.Triggered = true
			rr.Message = "Too many additions"
		}
	}

	return rr
}

// --- CRUD ---

func (s *RuleEngineService) List(projectID *uint) ([]models.ReviewRule, error) {
	var rules []models.ReviewRule
	query := s.db.Model(&models.ReviewRule{})
	if projectID != nil {
		query = query.Where("project_id IS NULL OR project_id = ?", *projectID)
	}
	if err := query.Order("priority DESC").Find(&rules).Error; err != nil {
		return nil, err
	}
	return rules, nil
}

func (s *RuleEngineService) Create(rule *models.ReviewRule) error {
	return s.db.Create(rule).Error
}

func (s *RuleEngineService) Update(id uint, updates map[string]interface{}) (*models.ReviewRule, error) {
	var rule models.ReviewRule
	if err := s.db.First(&rule, id).Error; err != nil {
		return nil, err
	}
	if err := s.db.Model(&rule).Updates(updates).Error; err != nil {
		return nil, err
	}
	s.db.First(&rule, id)
	return &rule, nil
}

func (s *RuleEngineService) Delete(id uint) error {
	return s.db.Delete(&models.ReviewRule{}, id).Error
}
