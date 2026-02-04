package services

import (
	"testing"
)

func TestParseDiffToFiles(t *testing.T) {
	tests := []struct {
		name     string
		diff     string
		expected int // expected number of files
	}{
		{
			name:     "empty diff",
			diff:     "",
			expected: 0,
		},
		{
			name:     "whitespace only",
			diff:     "   \n\t\n  ",
			expected: 0,
		},
		{
			name: "single file diff",
			diff: `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,3 +1,4 @@
 package main
+import "fmt"
 func main() {}
`,
			expected: 1,
		},
		{
			name: "multiple files diff",
			diff: `diff --git a/file1.go b/file1.go
--- a/file1.go
+++ b/file1.go
@@ -1,2 +1,3 @@
 package main
+// comment
diff --git a/file2.go b/file2.go
--- a/file2.go
+++ b/file2.go
@@ -1 +1,2 @@
 package util
+func Helper() {}
`,
			expected: 2,
		},
		{
			name: "diff with renamed file",
			diff: `diff --git a/old.go b/new.go
--- a/old.go
+++ b/new.go
@@ -1 +1 @@
-old content
+new content
`,
			expected: 1,
		},
		{
			name:     "non-standard format returns single file",
			diff:     "some random text without diff markers",
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := ParseDiffToFiles(tt.diff)
			if len(files) != tt.expected {
				t.Errorf("ParseDiffToFiles() returned %d files, expected %d", len(files), tt.expected)
			}
		})
	}
}

func TestParseDiffToFiles_FileDetails(t *testing.T) {
	diff := `diff --git a/src/main.go b/src/main.go
--- a/src/main.go
+++ b/src/main.go
@@ -1,5 +1,7 @@
 package main
 
+import "fmt"
+
 func main() {
-    // old
+    fmt.Println("hello")
 }
`
	files := ParseDiffToFiles(diff)
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	file := files[0]
	if file.FilePath != "src/main.go" {
		t.Errorf("FilePath = %q, expected %q", file.FilePath, "src/main.go")
	}
	if file.OldPath != "src/main.go" {
		t.Errorf("OldPath = %q, expected %q", file.OldPath, "src/main.go")
	}
	if file.NewPath != "src/main.go" {
		t.Errorf("NewPath = %q, expected %q", file.NewPath, "src/main.go")
	}
	if file.Additions != 3 {
		t.Errorf("Additions = %d, expected 3", file.Additions)
	}
	if file.Deletions != 1 {
		t.Errorf("Deletions = %d, expected 1", file.Deletions)
	}
	if file.TokenEstimate <= 0 {
		t.Errorf("TokenEstimate should be positive, got %d", file.TokenEstimate)
	}
}

func TestCreateBatches(t *testing.T) {
	tests := []struct {
		name              string
		files             []FileDiff
		maxTokensPerBatch int
		expectedBatches   int
	}{
		{
			name:              "empty files",
			files:             nil,
			maxTokensPerBatch: 1000,
			expectedBatches:   0,
		},
		{
			name: "single small file",
			files: []FileDiff{
				{FilePath: "a.go", TokenEstimate: 100},
			},
			maxTokensPerBatch: 1000,
			expectedBatches:   1,
		},
		{
			name: "multiple files fit in one batch",
			files: []FileDiff{
				{FilePath: "a.go", TokenEstimate: 100},
				{FilePath: "b.go", TokenEstimate: 200},
				{FilePath: "c.go", TokenEstimate: 300},
			},
			maxTokensPerBatch: 1000,
			expectedBatches:   1,
		},
		{
			name: "files split into multiple batches",
			files: []FileDiff{
				{FilePath: "a.go", TokenEstimate: 400},
				{FilePath: "b.go", TokenEstimate: 400},
				{FilePath: "c.go", TokenEstimate: 400},
			},
			maxTokensPerBatch: 500,
			expectedBatches:   3,
		},
		{
			name: "oversized file gets own batch",
			files: []FileDiff{
				{FilePath: "small.go", TokenEstimate: 100},
				{FilePath: "huge.go", TokenEstimate: 2000},
				{FilePath: "another.go", TokenEstimate: 100},
			},
			maxTokensPerBatch: 500,
			expectedBatches:   3,
		},
		{
			name: "default max tokens when zero",
			files: []FileDiff{
				{FilePath: "a.go", TokenEstimate: 100},
			},
			maxTokensPerBatch: 0,
			expectedBatches:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			batches := CreateBatches(tt.files, tt.maxTokensPerBatch)
			if len(batches) != tt.expectedBatches {
				t.Errorf("CreateBatches() returned %d batches, expected %d", len(batches), tt.expectedBatches)
			}
		})
	}
}

func TestCreateBatches_SortsByDirectory(t *testing.T) {
	files := []FileDiff{
		{FilePath: "z/file.go", TokenEstimate: 100},
		{FilePath: "a/file.go", TokenEstimate: 100},
		{FilePath: "a/other.go", TokenEstimate: 100},
	}

	batches := CreateBatches(files, 10000)
	if len(batches) != 1 {
		t.Fatalf("expected 1 batch, got %d", len(batches))
	}

	// Files should be sorted by directory then filename
	expectedOrder := []string{"a/file.go", "a/other.go", "z/file.go"}
	for i, file := range batches[0].Files {
		if file.FilePath != expectedOrder[i] {
			t.Errorf("file[%d] = %q, expected %q", i, file.FilePath, expectedOrder[i])
		}
	}
}

func TestAggregateResults(t *testing.T) {
	tests := []struct {
		name          string
		results       []BatchResult
		expectedScore float64
		expectedCount int
	}{
		{
			name:          "empty results",
			results:       nil,
			expectedScore: 0,
			expectedCount: 0,
		},
		{
			name: "single result",
			results: []BatchResult{
				{BatchIndex: 0, Score: 80, Weight: 10, Files: []string{"a.go"}, Content: "Good code"},
			},
			expectedScore: 80,
			expectedCount: 1,
		},
		{
			name: "weighted average",
			results: []BatchResult{
				{BatchIndex: 0, Score: 100, Weight: 10, Files: []string{"a.go"}, Content: "Perfect"},
				{BatchIndex: 1, Score: 50, Weight: 10, Files: []string{"b.go"}, Content: "Needs work"},
			},
			expectedScore: 75, // (100*10 + 50*10) / 20 = 75
			expectedCount: 2,
		},
		{
			name: "weighted average with different weights",
			results: []BatchResult{
				{BatchIndex: 0, Score: 100, Weight: 30, Files: []string{"big.go"}, Content: "Perfect"},
				{BatchIndex: 1, Score: 50, Weight: 10, Files: []string{"small.go"}, Content: "Needs work"},
			},
			expectedScore: 87.5, // (100*30 + 50*10) / 40 = 87.5
			expectedCount: 2,
		},
		{
			name: "zero weight treated as 1",
			results: []BatchResult{
				{BatchIndex: 0, Score: 80, Weight: 0, Files: []string{"a.go"}, Content: "Good"},
			},
			expectedScore: 80,
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AggregateResults(tt.results)
			if result.BatchCount != tt.expectedCount {
				t.Errorf("BatchCount = %d, expected %d", result.BatchCount, tt.expectedCount)
			}
			if result.Score != tt.expectedScore {
				t.Errorf("Score = %.2f, expected %.2f", result.Score, tt.expectedScore)
			}
		})
	}
}

func TestAggregateResults_ContentFormat(t *testing.T) {
	results := []BatchResult{
		{BatchIndex: 0, Score: 80, Weight: 10, Files: []string{"a.go", "b.go"}, Content: "Review 1"},
		{BatchIndex: 1, Score: 90, Weight: 5, Files: []string{"c.go"}, Content: "Review 2"},
	}

	aggregated := AggregateResults(results)

	// Check content contains expected sections
	if aggregated.Content == "" {
		t.Error("Content should not be empty")
	}
	if !contains(aggregated.Content, "Chunked Code Review Summary") {
		t.Error("Content should contain summary header")
	}
	if !contains(aggregated.Content, "Total Batches") {
		t.Error("Content should contain batch count")
	}
	if !contains(aggregated.Content, "a.go") {
		t.Error("Content should list files")
	}
	if !contains(aggregated.Content, "Batch 1 Review") {
		t.Error("Content should contain batch 1 review")
	}
}

func TestReconstructDiff(t *testing.T) {
	files := []FileDiff{
		{Content: "diff --git a/a.go b/a.go\n+line1\n"},
		{Content: "diff --git a/b.go b/b.go\n+line2"},
	}

	result := ReconstructDiff(files)

	if !contains(result, "diff --git a/a.go") {
		t.Error("Result should contain first diff")
	}
	if !contains(result, "diff --git a/b.go") {
		t.Error("Result should contain second diff")
	}
	// Check trailing newlines are handled
	if result[len(result)-1] != '\n' {
		t.Error("Result should end with newline")
	}
}

func TestGetBatchFileNames(t *testing.T) {
	batch := ReviewBatch{
		Files: []FileDiff{
			{FilePath: "a.go"},
			{FilePath: "b.go"},
			{FilePath: "c.go"},
		},
	}

	names := GetBatchFileNames(batch)

	if len(names) != 3 {
		t.Errorf("expected 3 names, got %d", len(names))
	}
	expectedNames := []string{"a.go", "b.go", "c.go"}
	for i, name := range names {
		if name != expectedNames[i] {
			t.Errorf("name[%d] = %q, expected %q", i, name, expectedNames[i])
		}
	}
}

func TestGetBatchWeight(t *testing.T) {
	tests := []struct {
		name     string
		batch    ReviewBatch
		expected int
	}{
		{
			name: "sum of additions and deletions",
			batch: ReviewBatch{
				Files: []FileDiff{
					{Additions: 10, Deletions: 5},
					{Additions: 20, Deletions: 10},
				},
			},
			expected: 45,
		},
		{
			name: "minimum weight is 1",
			batch: ReviewBatch{
				Files: []FileDiff{
					{Additions: 0, Deletions: 0},
				},
			},
			expected: 1,
		},
		{
			name:     "empty batch has minimum weight",
			batch:    ReviewBatch{},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			weight := GetBatchWeight(tt.batch)
			if weight != tt.expected {
				t.Errorf("GetBatchWeight() = %d, expected %d", weight, tt.expected)
			}
		})
	}
}

// helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
