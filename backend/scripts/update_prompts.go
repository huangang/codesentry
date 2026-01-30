package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/huangang/codesentry/backend/internal/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type PromptTemplate struct {
	ID      uint   `gorm:"primaryKey"`
	Name    string `gorm:"size:100"`
	Content string `gorm:"type:text"`
}

func (PromptTemplate) TableName() string { return "prompt_templates" }

func main() {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	db, err := gorm.Open(mysql.Open(cfg.Database.DSN), &gorm.Config{})
	if err != nil {
		fmt.Printf("Failed to connect to database: %v\n", err)
		os.Exit(1)
	}

	var prompts []PromptTemplate
	if err := db.Order("id").Find(&prompts).Error; err != nil {
		fmt.Printf("Failed to read prompts: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Found %d prompt templates:\n\n", len(prompts))

	for _, p := range prompts {
		fmt.Printf("=== ID: %d, Name: %s ===\n", p.ID, p.Name)
		fmt.Printf("Content:\n%s\n\n", p.Content)
		fmt.Println(strings.Repeat("-", 80))
	}

	if len(os.Args) > 1 && os.Args[1] == "--update" {
		fmt.Println("\n>>> Updating prompts to use conditional {{#if_file_context}} blocks...\n")

		for _, p := range prompts {
			newContent := updatePromptContent(p.Content)
			if newContent != p.Content {
				if err := db.Model(&PromptTemplate{}).Where("id = ?", p.ID).Update("content", newContent).Error; err != nil {
					fmt.Printf("Failed to update prompt %d: %v\n", p.ID, err)
				} else {
					fmt.Printf("Updated prompt ID %d: %s\n", p.ID, p.Name)
				}
			} else {
				fmt.Printf("Skipped prompt ID %d (already updated or no changes needed): %s\n", p.ID, p.Name)
			}
		}

		fmt.Println("\n>>> Done!")
	} else {
		fmt.Println("\nTo update prompts, run: go run scripts/update_prompts.go --update")
	}
}

func updatePromptContent(content string) string {
	if strings.Contains(content, "{{#if_file_context}}") {
		return content
	}

	hasOldFileContext := strings.Contains(content, "{{file_context}}")
	hasOldRule := strings.Contains(content, "当提供了完整文件上下文时") || strings.Contains(content, "When file context is provided")

	if !hasOldFileContext && !hasOldRule {
		return content
	}

	content = strings.ReplaceAll(content, "- **当提供了完整文件上下文时，请结合上下文理解代码变更的完整背景，避免仅根据 diff 片段做出片面判断。文件内容中 `»` 标记的行是本次修改的行。**\n", "")
	content = strings.ReplaceAll(content, "- **When file context is provided, use it to understand the full picture before judging the code changes. Lines marked with `»` are the modified lines.**\n", "")

	fileContextBlockZh := "{{#if_file_context}}\n**完整文件上下文**（`»` 标记的行是本次修改的行，请结合上下文理解代码变更）:\n{{file_context}}\n\n{{/if_file_context}}"

	fileContextBlockEn := "{{#if_file_context}}\n**Full File Context** (Lines marked with » are modified in this change, use context to understand the changes):\n{{file_context}}\n\n{{/if_file_context}}"

	if strings.Contains(content, "{{file_context}}\n代码变更内容:") {
		content = strings.Replace(content, "{{file_context}}\n代码变更内容:", fileContextBlockZh+"**代码变更内容**:", 1)
	} else if strings.Contains(content, "{{file_context}}\n代码变更内容：") {
		content = strings.Replace(content, "{{file_context}}\n代码变更内容：", fileContextBlockZh+"**代码变更内容**：", 1)
	} else if strings.Contains(content, "{{file_context}}\n**代码变更内容**:") {
		content = strings.Replace(content, "{{file_context}}\n**代码变更内容**:", fileContextBlockZh+"**代码变更内容**:", 1)
	} else if strings.Contains(content, "{{file_context}}\n**代码变更内容**：") {
		content = strings.Replace(content, "{{file_context}}\n**代码变更内容**：", fileContextBlockZh+"**代码变更内容**：", 1)
	} else if strings.Contains(content, "{{file_context}}\n**Code Changes**:") {
		content = strings.Replace(content, "{{file_context}}\n**Code Changes**:", fileContextBlockEn+"**Code Changes**:", 1)
	} else if strings.Contains(content, "{{file_context}}\n{{diffs}}") {
		if strings.Contains(content, "代码变更") || strings.Contains(content, "总分") {
			content = strings.Replace(content, "{{file_context}}\n{{diffs}}", fileContextBlockZh+"{{diffs}}", 1)
		} else {
			content = strings.Replace(content, "{{file_context}}\n{{diffs}}", fileContextBlockEn+"{{diffs}}", 1)
		}
	}

	return content
}
