package services

import (
	"testing"
)

func TestExtractScore(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected float64
	}{
		{
			name:     "chinese total score",
			content:  "代码质量良好\n总分：85分",
			expected: 85,
		},
		{
			name:     "chinese total score no suffix",
			content:  "总分: 90",
			expected: 90,
		},
		{
			name:     "english total score",
			content:  "Good code\nTotal Score: 75",
			expected: 75,
		},
		{
			name:     "score with /100",
			content:  "Score: 88/100",
			expected: 88,
		},
		{
			name:     "score /100 chinese",
			content:  "评分结果: 92/100分",
			expected: 92,
		},
		{
			name:     "chinese rating",
			content:  "评分: 70",
			expected: 70,
		},
		{
			name:     "score in markdown",
			content:  "## Review\n\n**Total Score: 65**\n\nDetails...",
			expected: 65,
		},
		{
			name:     "no score found",
			content:  "This is just some text without any score",
			expected: 0,
		},
		{
			name:     "score out of range high",
			content:  "Total Score: 150",
			expected: 0,
		},
		{
			name:     "score out of range negative",
			content:  "Total Score: -10",
			expected: 0,
		},
		{
			name:     "zero score is valid",
			content:  "Total Score: 0/100",
			expected: 0,
		},
		{
			name:     "100 is valid",
			content:  "Perfect! Total Score: 100",
			expected: 100,
		},
		{
			name:     "case insensitive",
			content:  "total score: 80",
			expected: 80,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractScore(tt.content)
			if result != tt.expected {
				t.Errorf("extractScore() = %.0f, expected %.0f", result, tt.expected)
			}
		})
	}
}

func TestContainsScoringInstruction(t *testing.T) {
	tests := []struct {
		name     string
		prompt   string
		expected bool
	}{
		{
			name:     "contains chinese total score",
			prompt:   "请给出代码的总分",
			expected: true,
		},
		{
			name:     "contains english score",
			prompt:   "Please provide a Total Score",
			expected: true,
		},
		{
			name:     "contains x/100",
			prompt:   "Rate the code from 0 to X/100",
			expected: true,
		},
		{
			name:     "contains rating keyword",
			prompt:   "Please rate the code quality",
			expected: true,
		},
		{
			name:     "contains scoring",
			prompt:   "Use the following scoring criteria",
			expected: true,
		},
		{
			name:     "no scoring instruction",
			prompt:   "Review the following code for bugs",
			expected: false,
		},
		{
			name:     "case insensitive",
			prompt:   "TOTAL SCORE required",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsScoringInstruction(tt.prompt)
			if result != tt.expected {
				t.Errorf("containsScoringInstruction() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestAppendScoringInstruction(t *testing.T) {
	original := "Review this code"
	result := appendScoringInstruction(original)

	if len(result) <= len(original) {
		t.Error("result should be longer than original")
	}
	if !containsSubstring(result, original) {
		t.Error("result should contain original prompt")
	}
	if !containsSubstring(result, "Total Score") {
		t.Error("result should contain scoring instruction")
	}
	if !containsSubstring(result, "X/100") {
		t.Error("result should contain score format")
	}
}

func TestProcessFileContextBlock(t *testing.T) {
	service := &AIService{}

	tests := []struct {
		name        string
		prompt      string
		fileContext string
		shouldKeep  bool
	}{
		{
			name:        "with file context - block kept",
			prompt:      "Review:\n{{#if_file_context}}Context: {{file_context}}{{/if_file_context}}\nDiff: {{diffs}}",
			fileContext: "file content here",
			shouldKeep:  true,
		},
		{
			name:        "without file context - block removed",
			prompt:      "Review:\n{{#if_file_context}}Context: {{file_context}}{{/if_file_context}}\nDiff: {{diffs}}",
			fileContext: "",
			shouldKeep:  false,
		},
		{
			name:        "whitespace only context - block removed",
			prompt:      "{{#if_file_context}}Has context{{/if_file_context}}",
			fileContext: "   \n\t  ",
			shouldKeep:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.processFileContextBlock(tt.prompt, tt.fileContext)

			if tt.shouldKeep {
				if containsSubstring(result, "{{#if_file_context}}") {
					t.Error("conditional block markers should be removed")
				}
				if !containsSubstring(result, tt.fileContext) {
					t.Error("file context should be present")
				}
			} else {
				if containsSubstring(result, "{{file_context}}") {
					t.Error("file_context placeholder should be removed")
				}
				if containsSubstring(result, "Has context") && tt.fileContext == "" {
					t.Error("conditional content should be removed when no context")
				}
			}
		})
	}
}
