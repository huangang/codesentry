package services

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// FileDiff represents a single file's diff content
type FileDiff struct {
	FilePath      string
	OldPath       string
	NewPath       string
	Content       string
	Additions     int
	Deletions     int
	TokenEstimate int // Rough estimate: len(Content) / 4
}

// ReviewBatch represents a batch of files to review together
type ReviewBatch struct {
	Files       []FileDiff
	TotalTokens int
}

// ChunkedReviewResult represents the aggregated result from multiple batches
type ChunkedReviewResult struct {
	Content      string
	Score        float64
	BatchCount   int
	BatchResults []BatchResult
}

// BatchResult represents the result from a single batch review
type BatchResult struct {
	BatchIndex int
	Files      []string
	Score      float64
	Content    string
	Weight     int // Weight for aggregation (based on additions + deletions)
}

// ParseDiffToFiles splits a unified diff string into individual file diffs
func ParseDiffToFiles(diff string) []FileDiff {
	var files []FileDiff

	// Split by "diff --git" blocks
	diffPattern := regexp.MustCompile(`(?m)^diff --git a/(.+?) b/(.+?)$`)
	indices := diffPattern.FindAllStringIndex(diff, -1)

	if len(indices) == 0 {
		// No standard diff format found, return single file
		if strings.TrimSpace(diff) != "" {
			return []FileDiff{{
				FilePath:      "unknown",
				Content:       diff,
				TokenEstimate: len(diff) / 4,
			}}
		}
		return nil
	}

	for i, idx := range indices {
		start := idx[0]
		end := len(diff)
		if i+1 < len(indices) {
			end = indices[i+1][0]
		}

		block := diff[start:end]
		matches := diffPattern.FindStringSubmatch(block)

		var oldPath, newPath string
		if len(matches) >= 3 {
			oldPath = matches[1]
			newPath = matches[2]
		}

		// Count additions and deletions
		additions := 0
		deletions := 0
		lines := strings.Split(block, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
				additions++
			} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
				deletions++
			}
		}

		filePath := newPath
		if filePath == "" {
			filePath = oldPath
		}

		files = append(files, FileDiff{
			FilePath:      filePath,
			OldPath:       oldPath,
			NewPath:       newPath,
			Content:       block,
			Additions:     additions,
			Deletions:     deletions,
			TokenEstimate: len(block) / 4,
		})
	}

	return files
}

// CreateBatches groups files into batches based on token limits
// maxTokensPerBatch default: 30000 (leaves room for prompt + response)
func CreateBatches(files []FileDiff, maxTokensPerBatch int) []ReviewBatch {
	if len(files) == 0 {
		return nil
	}

	if maxTokensPerBatch <= 0 {
		maxTokensPerBatch = 30000
	}

	// Sort files by directory to keep related files together
	sort.Slice(files, func(i, j int) bool {
		dirI := filepath.Dir(files[i].FilePath)
		dirJ := filepath.Dir(files[j].FilePath)
		if dirI != dirJ {
			return dirI < dirJ
		}
		return files[i].FilePath < files[j].FilePath
	})

	var batches []ReviewBatch
	var currentBatch ReviewBatch

	for _, file := range files {
		// If single file exceeds limit, put it in its own batch
		if file.TokenEstimate > maxTokensPerBatch {
			// Save current batch if not empty
			if len(currentBatch.Files) > 0 {
				batches = append(batches, currentBatch)
				currentBatch = ReviewBatch{}
			}
			// Add oversized file as its own batch
			batches = append(batches, ReviewBatch{
				Files:       []FileDiff{file},
				TotalTokens: file.TokenEstimate,
			})
			continue
		}

		// Check if adding this file exceeds the limit
		if currentBatch.TotalTokens+file.TokenEstimate > maxTokensPerBatch {
			// Save current batch and start new one
			if len(currentBatch.Files) > 0 {
				batches = append(batches, currentBatch)
			}
			currentBatch = ReviewBatch{}
		}

		currentBatch.Files = append(currentBatch.Files, file)
		currentBatch.TotalTokens += file.TokenEstimate
	}

	// Don't forget the last batch
	if len(currentBatch.Files) > 0 {
		batches = append(batches, currentBatch)
	}

	return batches
}

// AggregateResults combines multiple batch results into a final result
// Uses weighted average based on file changes (additions + deletions)
func AggregateResults(results []BatchResult) *ChunkedReviewResult {
	if len(results) == 0 {
		return &ChunkedReviewResult{
			Content:    "No review results available",
			Score:      0,
			BatchCount: 0,
		}
	}

	// Calculate weighted average score
	var totalWeight int
	var weightedScoreSum float64
	var contentBuilder strings.Builder

	for i, result := range results {
		weight := result.Weight
		if weight <= 0 {
			weight = 1 // Minimum weight
		}
		totalWeight += weight
		weightedScoreSum += result.Score * float64(weight)

		// Build combined content
		if i > 0 {
			contentBuilder.WriteString("\n\n---\n\n")
		}
		contentBuilder.WriteString(fmt.Sprintf("## Batch %d Review (%d files)\n\n", result.BatchIndex+1, len(result.Files)))
		contentBuilder.WriteString(fmt.Sprintf("**Files reviewed:** %s\n\n", strings.Join(result.Files, ", ")))
		contentBuilder.WriteString(result.Content)
	}

	var finalScore float64
	if totalWeight > 0 {
		finalScore = weightedScoreSum / float64(totalWeight)
	}

	// Add summary header
	var summaryBuilder strings.Builder
	summaryBuilder.WriteString("# Chunked Code Review Summary\n\n")
	summaryBuilder.WriteString(fmt.Sprintf("**Total Batches:** %d\n", len(results)))
	summaryBuilder.WriteString(fmt.Sprintf("**Aggregated Score:** %.0f/100\n\n", finalScore))

	// List all files reviewed
	summaryBuilder.WriteString("**All Files Reviewed:**\n")
	for _, result := range results {
		for _, file := range result.Files {
			summaryBuilder.WriteString(fmt.Sprintf("- %s\n", file))
		}
	}
	summaryBuilder.WriteString("\n---\n\n")

	// Add individual batch reviews
	summaryBuilder.WriteString(contentBuilder.String())

	return &ChunkedReviewResult{
		Content:      summaryBuilder.String(),
		Score:        finalScore,
		BatchCount:   len(results),
		BatchResults: results,
	}
}

// ReconstructDiff rebuilds a unified diff string from file diffs
func ReconstructDiff(files []FileDiff) string {
	var builder strings.Builder
	for _, file := range files {
		builder.WriteString(file.Content)
		if !strings.HasSuffix(file.Content, "\n") {
			builder.WriteString("\n")
		}
	}
	return builder.String()
}

// GetBatchFileNames returns a list of file names from a batch
func GetBatchFileNames(batch ReviewBatch) []string {
	names := make([]string, len(batch.Files))
	for i, f := range batch.Files {
		names[i] = f.FilePath
	}
	return names
}

// GetBatchWeight calculates the total weight (additions + deletions) for a batch
func GetBatchWeight(batch ReviewBatch) int {
	weight := 0
	for _, f := range batch.Files {
		weight += f.Additions + f.Deletions
	}
	if weight == 0 {
		weight = 1 // Minimum weight
	}
	return weight
}
