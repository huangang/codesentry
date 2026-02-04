package services

import (
	"testing"
)

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		filePath string
		expected string
	}{
		{"main.go", "go"},
		{"app.js", "javascript"},
		{"component.tsx", "typescript"},
		{"utils.ts", "typescript"},
		{"script.py", "python"},
		{"App.java", "java"},
		{"main.c", "c"},
		{"lib.cpp", "cpp"},
		{"header.h", "c"},
		{"template.hpp", "cpp"},
		{"Program.cs", "csharp"},
		{"gem.rb", "ruby"},
		{"index.php", "php"},
		{"App.swift", "swift"},
		{"Main.kt", "kotlin"},
		{"lib.rs", "rust"},
		{"App.vue", "vue"},
		{"Component.svelte", "svelte"},
		{"unknown.xyz", "text"},
		{"no_extension", "text"},
		{"path/to/file.go", "go"},
		{"UPPER.GO", "go"},
	}

	for _, tt := range tests {
		t.Run(tt.filePath, func(t *testing.T) {
			result := detectLanguage(tt.filePath)
			if result != tt.expected {
				t.Errorf("detectLanguage(%q) = %q, expected %q", tt.filePath, result, tt.expected)
			}
		})
	}
}

func TestExtractModifiedRanges(t *testing.T) {
	tests := []struct {
		name     string
		diff     string
		expected []LineRange
	}{
		{
			name:     "empty diff",
			diff:     "",
			expected: nil,
		},
		{
			name:     "single hunk",
			diff:     "@@ -1,3 +1,5 @@\n content",
			expected: []LineRange{{Start: 1, End: 5}},
		},
		{
			name:     "multiple hunks",
			diff:     "@@ -10,5 +10,3 @@\n...\n@@ -50,2 +48,10 @@\n...",
			expected: []LineRange{{Start: 10, End: 12}, {Start: 48, End: 57}},
		},
		{
			name:     "hunk without count",
			diff:     "@@ -1 +1 @@\n single line change",
			expected: []LineRange{{Start: 1, End: 1}},
		},
		{
			name:     "new file",
			diff:     "@@ -0,0 +1,20 @@\n new file content",
			expected: []LineRange{{Start: 1, End: 20}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractModifiedRanges(tt.diff)
			if len(result) != len(tt.expected) {
				t.Errorf("got %d ranges, expected %d", len(result), len(tt.expected))
				return
			}
			for i, r := range result {
				if r.Start != tt.expected[i].Start || r.End != tt.expected[i].End {
					t.Errorf("range[%d] = {%d, %d}, expected {%d, %d}",
						i, r.Start, r.End, tt.expected[i].Start, tt.expected[i].End)
				}
			}
		})
	}
}

func TestExtractGoFunctions(t *testing.T) {
	content := `package main

func init() {
	setup()
}

func main() {
	fmt.Println("hello")
}

func helper(x int) int {
	return x * 2
}

type Service struct{}

func (s *Service) Method() {
	doSomething()
}
`
	lines := splitLines(content)

	tests := []struct {
		name           string
		modifiedRanges []LineRange
		expectedFuncs  []string
	}{
		{
			name:           "modification in main",
			modifiedRanges: []LineRange{{Start: 8, End: 8}},
			expectedFuncs:  []string{"main"},
		},
		{
			name:           "modification in helper",
			modifiedRanges: []LineRange{{Start: 12, End: 12}},
			expectedFuncs:  []string{"helper"},
		},
		{
			name:           "modification spans multiple functions",
			modifiedRanges: []LineRange{{Start: 4, End: 12}},
			expectedFuncs:  []string{"init", "main", "helper"},
		},
		{
			name:           "modification in method",
			modifiedRanges: []LineRange{{Start: 18, End: 18}},
			expectedFuncs:  []string{"Method"},
		},
		{
			name:           "no overlap",
			modifiedRanges: []LineRange{{Start: 15, End: 15}},
			expectedFuncs:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			functions := extractGoFunctions(lines, tt.modifiedRanges)
			if len(functions) != len(tt.expectedFuncs) {
				t.Errorf("got %d functions, expected %d", len(functions), len(tt.expectedFuncs))
				return
			}
			for i, fn := range functions {
				if fn.Name != tt.expectedFuncs[i] {
					t.Errorf("function[%d].Name = %q, expected %q", i, fn.Name, tt.expectedFuncs[i])
				}
				if fn.Language != "go" {
					t.Errorf("function[%d].Language = %q, expected %q", i, fn.Language, "go")
				}
			}
		})
	}
}

func TestExtractPythonFunctions(t *testing.T) {
	content := `def hello():
    print("hello")

async def fetch_data():
    return await api.get()

class MyClass:
    def method(self):
        pass
`
	lines := splitLines(content)

	tests := []struct {
		name           string
		modifiedRanges []LineRange
		expectedFuncs  []string
	}{
		{
			name:           "modification in hello",
			modifiedRanges: []LineRange{{Start: 2, End: 2}},
			expectedFuncs:  []string{"hello"},
		},
		{
			name:           "modification in async function",
			modifiedRanges: []LineRange{{Start: 5, End: 5}},
			expectedFuncs:  []string{"fetch_data"},
		},
		{
			name:           "modification in method",
			modifiedRanges: []LineRange{{Start: 9, End: 9}},
			expectedFuncs:  []string{"method"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			functions := extractPythonFunctions(lines, tt.modifiedRanges)
			if len(functions) != len(tt.expectedFuncs) {
				t.Errorf("got %d functions, expected %d", len(functions), len(tt.expectedFuncs))
				return
			}
			for i, fn := range functions {
				if fn.Name != tt.expectedFuncs[i] {
					t.Errorf("function[%d].Name = %q, expected %q", i, fn.Name, tt.expectedFuncs[i])
				}
			}
		})
	}
}

func TestExtractJSFunctions(t *testing.T) {
	content := `function greet() {
  console.log("hi");
}

const helper = () => {
  return 42;
}

export async function fetchData() {
  return await fetch(url);
}
`
	lines := splitLines(content)

	tests := []struct {
		name           string
		modifiedRanges []LineRange
		expectedFuncs  []string
	}{
		{
			name:           "modification in greet",
			modifiedRanges: []LineRange{{Start: 2, End: 2}},
			expectedFuncs:  []string{"greet"},
		},
		{
			name:           "modification in arrow function",
			modifiedRanges: []LineRange{{Start: 6, End: 6}},
			expectedFuncs:  []string{"helper"},
		},
		{
			name:           "modification in exported async function",
			modifiedRanges: []LineRange{{Start: 10, End: 10}},
			expectedFuncs:  []string{"fetchData"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			functions := extractJSFunctions(lines, tt.modifiedRanges)
			if len(functions) != len(tt.expectedFuncs) {
				t.Errorf("got %d functions, expected %d", len(functions), len(tt.expectedFuncs))
				return
			}
			for i, fn := range functions {
				if fn.Name != tt.expectedFuncs[i] {
					t.Errorf("function[%d].Name = %q, expected %q", i, fn.Name, tt.expectedFuncs[i])
				}
			}
		})
	}
}

func TestExtractGenericContext(t *testing.T) {
	lines := make([]string, 50)
	for i := range lines {
		lines[i] = "line content"
	}

	modifiedRanges := []LineRange{{Start: 25, End: 26}}
	functions := extractGenericContext(lines, modifiedRanges)

	if len(functions) != 1 {
		t.Fatalf("expected 1 context block, got %d", len(functions))
	}

	fn := functions[0]
	if fn.StartLine > 25-15 {
		t.Errorf("StartLine = %d, expected <= %d", fn.StartLine, 25-15)
	}
	if fn.EndLine < 26+15 {
		t.Errorf("EndLine = %d, expected >= %d", fn.EndLine, 26+15)
	}
}

func TestFormatFunctionDefinitions(t *testing.T) {
	functions := []FunctionDefinition{
		{
			Name:      "TestFunc",
			StartLine: 10,
			EndLine:   20,
			Content:   "func TestFunc() {\n  // body\n}",
			Language:  "go",
			FilePath:  "test.go",
		},
	}

	result := FormatFunctionDefinitions(functions, "test.go")

	if result == "" {
		t.Error("result should not be empty")
	}
	if !containsSubstring(result, "TestFunc") {
		t.Error("result should contain function name")
	}
	if !containsSubstring(result, "lines 10-20") {
		t.Error("result should contain line numbers")
	}
	if !containsSubstring(result, "```go") {
		t.Error("result should contain language fence")
	}
}

func TestFormatFunctionDefinitions_Empty(t *testing.T) {
	result := FormatFunctionDefinitions(nil, "test.go")
	if result != "" {
		t.Errorf("empty functions should return empty string, got %q", result)
	}
}

func TestFormatFileContexts(t *testing.T) {
	contexts := []FileContext{
		{
			FilePath:       "main.go",
			Content:        "package main\n\nfunc main() {}",
			ModifiedRanges: []LineRange{{Start: 3, End: 3}},
			Language:       "go",
		},
	}

	result := formatFileContexts(contexts)

	if !containsSubstring(result, "File Context") {
		t.Error("should contain header")
	}
	if !containsSubstring(result, "main.go") {
		t.Error("should contain file path")
	}
	if !containsSubstring(result, "lines 3-3") {
		t.Error("should contain modified range")
	}
	if !containsSubstring(result, "»") {
		t.Error("should mark modified lines")
	}
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
