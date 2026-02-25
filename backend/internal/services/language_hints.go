package services

import (
	"path/filepath"
	"strings"
)

// languageHints maps programming languages to specific review focus points.
var languageHints = map[string]string{
	"go": `Go-specific checks:
- Check for unhandled errors (err != nil patterns)
- Verify proper defer/close usage for resources
- Check goroutine leaks and race conditions
- Ensure proper context.Context propagation
- Validate struct tag correctness`,

	"python": `Python-specific checks:
- Check for proper exception handling (avoid bare except)
- Verify type hints consistency
- Check for mutable default arguments
- Validate proper resource cleanup (with statements)
- Check for potential injection vulnerabilities in string formatting`,

	"javascript": `JavaScript/TypeScript-specific checks:
- Check for potential XSS vulnerabilities
- Verify proper async/await and Promise error handling
- Check for memory leaks (event listeners, intervals)
- Validate proper null/undefined checks
- Check for unused imports and variables`,

	"typescript": `JavaScript/TypeScript-specific checks:
- Check for proper TypeScript type safety (avoid 'any')
- Verify proper async/await and Promise error handling
- Check for memory leaks (event listeners, intervals)
- Validate proper null/undefined checks
- Check for unused imports and variables`,

	"java": `Java-specific checks:
- Check for proper exception handling and resource management (try-with-resources)
- Verify null safety (Optional usage, @Nullable annotations)
- Check for thread safety issues
- Validate proper equals/hashCode implementations
- Check for potential SQL injection in query construction`,

	"rust": `Rust-specific checks:
- Check for proper error handling (Result/Option usage)
- Verify ownership and borrowing patterns
- Check for unsafe blocks necessity
- Validate proper lifetime annotations
- Check for potential panics (unwrap usage)`,

	"ruby": `Ruby-specific checks:
- Check for proper exception handling
- Verify security of eval/send usage
- Check for N+1 queries in ActiveRecord
- Validate input sanitization
- Check for proper use of symbols vs strings`,

	"php": `PHP-specific checks:
- Check for SQL injection vulnerabilities
- Verify proper input validation and sanitization
- Check for XSS vulnerabilities
- Validate proper error handling
- Check for type safety issues`,

	"swift": `Swift-specific checks:
- Check for proper optional handling (avoid force unwrapping)
- Verify memory management (retain cycles, weak references)
- Check for proper error handling with do-catch
- Validate thread safety with actors/locks
- Check for proper Codable implementations`,

	"kotlin": `Kotlin-specific checks:
- Check for proper null safety usage
- Verify coroutine scope and cancellation handling
- Check for proper sealed class/when exhaustiveness
- Validate data class usage
- Check for potential Java interop issues`,

	"c": `C/C++-specific checks:
- Check for memory leaks and buffer overflows
- Verify pointer safety and null dereferences
- Check for integer overflow vulnerabilities
- Validate proper resource cleanup
- Check for undefined behavior`,

	"cpp": `C/C++-specific checks:
- Check for memory leaks and smart pointer usage
- Verify RAII patterns for resource management
- Check for buffer overflows and bounds checking
- Validate exception safety guarantees
- Check for thread safety issues`,
}

// extensionToLanguage maps file extensions to language keys.
var extensionToLanguage = map[string]string{
	".go":     "go",
	".py":     "python",
	".pyw":    "python",
	".js":     "javascript",
	".jsx":    "javascript",
	".ts":     "typescript",
	".tsx":    "typescript",
	".mjs":    "javascript",
	".cjs":    "javascript",
	".java":   "java",
	".rs":     "rust",
	".rb":     "ruby",
	".php":    "php",
	".swift":  "swift",
	".kt":     "kotlin",
	".kts":    "kotlin",
	".c":      "c",
	".h":      "c",
	".cpp":    "cpp",
	".cc":     "cpp",
	".cxx":    "cpp",
	".hpp":    "cpp",
	".cs":     "java", // C# shares similar patterns with Java
	".scala":  "java",
	".vue":    "javascript",
	".svelte": "javascript",
}

// DetectLanguagesFromDiff parses a unified diff to extract file extensions
// and returns a deduplicated list of detected languages.
func DetectLanguagesFromDiff(diff string) []string {
	seen := make(map[string]bool)
	var languages []string

	for _, line := range strings.Split(diff, "\n") {
		// Match diff headers: "diff --git a/path/to/file.go b/path/to/file.go"
		// or "+++ b/path/to/file.go"
		var filePath string
		if strings.HasPrefix(line, "diff --git ") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				filePath = strings.TrimPrefix(parts[len(parts)-1], "b/")
			}
		} else if strings.HasPrefix(line, "+++ b/") {
			filePath = strings.TrimPrefix(line, "+++ b/")
		}

		if filePath == "" {
			continue
		}

		ext := strings.ToLower(filepath.Ext(filePath))
		if lang, ok := extensionToLanguage[ext]; ok {
			if !seen[lang] {
				seen[lang] = true
				languages = append(languages, lang)
			}
		}
	}

	return languages
}

// GenerateLanguageHints creates a prompt section with language-specific review
// guidance based on the detected languages in the diff.
func GenerateLanguageHints(diff string) string {
	languages := DetectLanguagesFromDiff(diff)
	if len(languages) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("\n\n--- Language-Specific Review Guidelines ---\n")

	for _, lang := range languages {
		if hint, ok := languageHints[lang]; ok {
			b.WriteString("\n")
			b.WriteString(hint)
			b.WriteString("\n")
		}
	}

	return b.String()
}
