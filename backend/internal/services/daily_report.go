package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/huangang/codesentry/backend/internal/models"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

type DailyReportService struct {
	db                  *gorm.DB
	aiService           *AIService
	notificationService *NotificationService
	cronScheduler       *cron.Cron
	currentEntryID      cron.EntryID
}

func NewDailyReportService(db *gorm.DB, aiService *AIService, notificationService *NotificationService) *DailyReportService {
	return &DailyReportService{
		db:                  db,
		aiService:           aiService,
		notificationService: notificationService,
	}
}

type ReportStats struct {
	TotalProjects  int     `json:"total_projects"`
	TotalCommits   int     `json:"total_commits"`
	TotalAuthors   int     `json:"total_authors"`
	TotalAdditions int     `json:"total_additions"`
	TotalDeletions int     `json:"total_deletions"`
	AverageScore   float64 `json:"average_score"`
	PassedCount    int     `json:"passed_count"`
	FailedCount    int     `json:"failed_count"`
	PendingCount   int     `json:"pending_count"`
}

type ProjectStat struct {
	Name        string  `json:"name"`
	CommitCount int     `json:"commit_count"`
	AvgScore    float64 `json:"avg_score"`
}

type AuthorStat struct {
	Name        string  `json:"name"`
	CommitCount int     `json:"commit_count"`
	AvgScore    float64 `json:"avg_score"`
}

type LowScoreReview struct {
	Project string  `json:"project"`
	Author  string  `json:"author"`
	Score   float64 `json:"score"`
	Summary string  `json:"summary"`
}

func (s *DailyReportService) StartScheduler() {
	s.cronScheduler = cron.New()

	s.updateSchedule()

	s.cronScheduler.Start()
	log.Println("[DailyReport] Scheduler started")
}

func (s *DailyReportService) StopScheduler() {
	if s.cronScheduler != nil {
		s.cronScheduler.Stop()
	}
}

func (s *DailyReportService) updateSchedule() {
	if s.currentEntryID != 0 {
		s.cronScheduler.Remove(s.currentEntryID)
	}

	reportTime := s.getReportTime()
	parts := strings.Split(reportTime, ":")
	hour := "18"
	minute := "0"
	if len(parts) == 2 {
		hour = parts[0]
		minute = parts[1]
	}

	cronExpr := fmt.Sprintf("%s %s * * *", minute, hour)

	entryID, err := s.cronScheduler.AddFunc(cronExpr, func() {
		s.GenerateAndSendReport()
	})
	if err != nil {
		log.Printf("[DailyReport] Failed to add cron job: %v", err)
		return
	}

	s.currentEntryID = entryID
	log.Printf("[DailyReport] Scheduled at %s (cron: %s)", reportTime, cronExpr)
}

func (s *DailyReportService) getReportTime() string {
	var config models.SystemConfig
	if err := s.db.Where("`key` = ?", "daily_report_time").First(&config).Error; err != nil {
		return "18:00"
	}
	return config.Value
}

func (s *DailyReportService) isEnabled() bool {
	var config models.SystemConfig
	if err := s.db.Where("`key` = ?", "daily_report_enabled").First(&config).Error; err != nil {
		return false
	}
	return config.Value == "true"
}

func (s *DailyReportService) getLowScoreThreshold() float64 {
	var config models.SystemConfig
	if err := s.db.Where("`key` = ?", "daily_report_low_score").First(&config).Error; err != nil {
		return 60
	}
	threshold, err := strconv.ParseFloat(config.Value, 64)
	if err != nil {
		return 60
	}
	return threshold
}

func (s *DailyReportService) getLLMConfigID() uint {
	var config models.SystemConfig
	if err := s.db.Where("`key` = ?", "daily_report_llm_config_id").First(&config).Error; err != nil {
		return 0
	}
	id, err := strconv.ParseUint(config.Value, 10, 64)
	if err != nil {
		return 0
	}
	return uint(id)
}

func (s *DailyReportService) GenerateAndSendReport() error {
	report, err := s.GenerateReport()
	if err != nil {
		return err
	}

	if err := s.sendNotifications(report); err != nil {
		report.NotifyError = err.Error()
		s.db.Save(report)
		return err
	}

	now := time.Now()
	report.NotifiedAt = &now
	s.db.Save(report)

	log.Printf("[DailyReport] Report generated and sent successfully (ID: %d)", report.ID)
	return nil
}

func (s *DailyReportService) GenerateReport() (*models.DailyReport, error) {
	log.Println("[DailyReport] Generating daily report...")

	today := time.Now()
	startOfDay := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	report, err := s.generateReport(startOfDay, endOfDay)
	if err != nil {
		log.Printf("[DailyReport] Failed to generate report: %v", err)
		return nil, err
	}

	var existingReport models.DailyReport
	if err := s.db.Where("report_date = ?", startOfDay).First(&existingReport).Error; err == nil {
		report.ID = existingReport.ID
		report.CreatedAt = existingReport.CreatedAt
		report.NotifiedAt = existingReport.NotifiedAt
		if err := s.db.Save(report).Error; err != nil {
			log.Printf("[DailyReport] Failed to update report: %v", err)
			return nil, err
		}
		log.Printf("[DailyReport] Updated existing report (ID: %d)", report.ID)
	} else {
		if err := s.db.Create(report).Error; err != nil {
			log.Printf("[DailyReport] Failed to save report: %v", err)
			return nil, err
		}
		log.Printf("[DailyReport] Created new report (ID: %d)", report.ID)
	}

	return report, nil
}

func (s *DailyReportService) generateReport(startTime, endTime time.Time) (*models.DailyReport, error) {
	stats := s.collectStats(startTime, endTime)
	topProjects := s.getTopProjects(startTime, endTime, 5)
	topAuthors := s.getTopAuthors(startTime, endTime, 5)
	lowScoreReviews := s.getLowScoreReviews(startTime, endTime)

	topProjectsJSON, _ := json.Marshal(topProjects)
	topAuthorsJSON, _ := json.Marshal(topAuthors)
	lowScoreReviewsJSON, _ := json.Marshal(lowScoreReviews)

	aiAnalysis, modelUsed := s.generateAIAnalysis(stats, topProjects, topAuthors, lowScoreReviews)

	report := &models.DailyReport{
		ReportDate:      startTime,
		ReportType:      "daily",
		TotalProjects:   stats.TotalProjects,
		TotalCommits:    stats.TotalCommits,
		TotalAuthors:    stats.TotalAuthors,
		TotalAdditions:  stats.TotalAdditions,
		TotalDeletions:  stats.TotalDeletions,
		AverageScore:    stats.AverageScore,
		PassedCount:     stats.PassedCount,
		FailedCount:     stats.FailedCount,
		PendingCount:    stats.PendingCount,
		TopProjects:     string(topProjectsJSON),
		TopAuthors:      string(topAuthorsJSON),
		LowScoreReviews: string(lowScoreReviewsJSON),
		AIAnalysis:      aiAnalysis,
		AIModelUsed:     modelUsed,
	}

	return report, nil
}

func (s *DailyReportService) collectStats(startTime, endTime time.Time) ReportStats {
	var stats ReportStats

	var totalProjects int64
	s.db.Model(&models.ReviewLog{}).
		Where("created_at BETWEEN ? AND ?", startTime, endTime).
		Distinct("project_id").
		Count(&totalProjects)
	stats.TotalProjects = int(totalProjects)

	var totalCommits int64
	s.db.Model(&models.ReviewLog{}).
		Where("created_at BETWEEN ? AND ?", startTime, endTime).
		Count(&totalCommits)
	stats.TotalCommits = int(totalCommits)

	var totalAuthors int64
	s.db.Model(&models.ReviewLog{}).
		Where("created_at BETWEEN ? AND ?", startTime, endTime).
		Distinct("author").
		Count(&totalAuthors)
	stats.TotalAuthors = int(totalAuthors)

	s.db.Model(&models.ReviewLog{}).
		Where("created_at BETWEEN ? AND ?", startTime, endTime).
		Select("COALESCE(SUM(additions), 0)").
		Scan(&stats.TotalAdditions)

	s.db.Model(&models.ReviewLog{}).
		Where("created_at BETWEEN ? AND ?", startTime, endTime).
		Select("COALESCE(SUM(deletions), 0)").
		Scan(&stats.TotalDeletions)

	s.db.Model(&models.ReviewLog{}).
		Where("created_at BETWEEN ? AND ? AND score IS NOT NULL", startTime, endTime).
		Select("COALESCE(AVG(score), 0)").
		Scan(&stats.AverageScore)

	threshold := s.getLowScoreThreshold()

	var passedCount int64
	s.db.Model(&models.ReviewLog{}).
		Where("created_at BETWEEN ? AND ? AND score IS NOT NULL AND score >= ?", startTime, endTime, threshold).
		Count(&passedCount)
	stats.PassedCount = int(passedCount)

	var failedCount int64
	s.db.Model(&models.ReviewLog{}).
		Where("created_at BETWEEN ? AND ? AND score IS NOT NULL AND score < ?", startTime, endTime, threshold).
		Count(&failedCount)
	stats.FailedCount = int(failedCount)

	var pendingCount int64
	s.db.Model(&models.ReviewLog{}).
		Where("created_at BETWEEN ? AND ? AND review_status = ?", startTime, endTime, "pending").
		Count(&pendingCount)
	stats.PendingCount = int(pendingCount)

	return stats
}

func (s *DailyReportService) getTopProjects(startTime, endTime time.Time, limit int) []ProjectStat {
	var results []struct {
		ProjectID   uint
		CommitCount int
		AvgScore    float64
	}

	s.db.Model(&models.ReviewLog{}).
		Select("project_id, COUNT(*) as commit_count, COALESCE(AVG(score), 0) as avg_score").
		Where("created_at BETWEEN ? AND ?", startTime, endTime).
		Group("project_id").
		Order("commit_count DESC").
		Limit(limit).
		Scan(&results)

	var stats []ProjectStat
	for _, r := range results {
		var project models.Project
		if err := s.db.First(&project, r.ProjectID).Error; err == nil {
			stats = append(stats, ProjectStat{
				Name:        project.Name,
				CommitCount: r.CommitCount,
				AvgScore:    r.AvgScore,
			})
		}
	}

	return stats
}

func (s *DailyReportService) getTopAuthors(startTime, endTime time.Time, limit int) []AuthorStat {
	var results []struct {
		Author      string
		CommitCount int
		AvgScore    float64
	}

	s.db.Model(&models.ReviewLog{}).
		Select("author, COUNT(*) as commit_count, COALESCE(AVG(score), 0) as avg_score").
		Where("created_at BETWEEN ? AND ?", startTime, endTime).
		Group("author").
		Order("commit_count DESC").
		Limit(limit).
		Scan(&results)

	var stats []AuthorStat
	for _, r := range results {
		stats = append(stats, AuthorStat{
			Name:        r.Author,
			CommitCount: r.CommitCount,
			AvgScore:    r.AvgScore,
		})
	}

	return stats
}

func (s *DailyReportService) getLowScoreReviews(startTime, endTime time.Time) []LowScoreReview {
	threshold := s.getLowScoreThreshold()

	var reviews []models.ReviewLog
	s.db.Preload("Project").
		Where("created_at BETWEEN ? AND ? AND score IS NOT NULL AND score < ?", startTime, endTime, threshold).
		Order("score ASC").
		Limit(10).
		Find(&reviews)

	var lowScores []LowScoreReview
	for _, r := range reviews {
		projectName := ""
		if r.Project != nil {
			projectName = r.Project.Name
		}

		summary := r.CommitMessage
		if len(summary) > 50 {
			summary = summary[:50] + "..."
		}

		lowScores = append(lowScores, LowScoreReview{
			Project: projectName,
			Author:  r.Author,
			Score:   *r.Score,
			Summary: summary,
		})
	}

	return lowScores
}

func (s *DailyReportService) generateAIAnalysis(stats ReportStats, topProjects []ProjectStat, topAuthors []AuthorStat, lowScores []LowScoreReview) (string, string) {
	if s.aiService == nil {
		return s.buildDefaultSummary(stats, topProjects, topAuthors, lowScores), ""
	}

	lowScoreThreshold := s.getLowScoreThreshold()

	contextData := map[string]interface{}{
		"report_type":         "daily_summary",
		"date":                time.Now().Format("2006-01-02"),
		"metrics":             stats,
		"top_projects":        topProjects,
		"top_authors":         topAuthors,
		"low_scores":          lowScores,
		"low_score_threshold": lowScoreThreshold,
	}

	contextJSON, _ := json.Marshal(contextData)

	prompt := fmt.Sprintf(`你是一位技术团队经理，请根据以下代码审查数据生成一份简洁的日报摘要。

数据：
%s

说明：
- 低分阈值为 %.0f 分，低于此分数的提交需要特别关注
- passed_count 表示分数 >= %.0f 的提交数
- failed_count 表示分数 < %.0f 的提交数

请生成一份 Markdown 格式的日报，包含：
1. 今日概览（审查数、通过率、平均分、贡献者数）
2. Top 活跃项目（最多5个）
3. 需要关注的低分提交（分数 < %.0f，如果有）
4. 1-2 条简短的 AI 洞察/建议

注意：输出要简洁，适合在 IM 群里阅读，总字数控制在 500 字以内。`, string(contextJSON), lowScoreThreshold, lowScoreThreshold, lowScoreThreshold, lowScoreThreshold)

	llmConfigID := s.getLLMConfigID()
	content, modelName, err := s.aiService.CallWithConfig(context.Background(), llmConfigID, prompt)

	if err != nil {
		log.Printf("[DailyReport] AI analysis failed: %v", err)
		return s.buildDefaultSummary(stats, topProjects, topAuthors, lowScores), ""
	}

	return content, modelName
}

func (s *DailyReportService) buildDefaultSummary(stats ReportStats, topProjects []ProjectStat, topAuthors []AuthorStat, lowScores []LowScoreReview) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("## 📊 CodeSentry 日报 - %s\n\n", time.Now().Format("2006-01-02")))

	passRate := 0.0
	if stats.TotalCommits > 0 {
		passRate = float64(stats.PassedCount) / float64(stats.PassedCount+stats.FailedCount) * 100
	}

	sb.WriteString("### 今日概览\n")
	sb.WriteString(fmt.Sprintf("- 🔍 审查数：%d（通过 %d / 未通过 %d）\n", stats.TotalCommits, stats.PassedCount, stats.FailedCount))
	sb.WriteString(fmt.Sprintf("- 📈 平均分：%.1f 分 | 通过率：%.0f%%\n", stats.AverageScore, passRate))
	sb.WriteString(fmt.Sprintf("- 👥 贡献者：%d 人\n", stats.TotalAuthors))
	sb.WriteString(fmt.Sprintf("- 📁 活跃项目：%d 个\n\n", stats.TotalProjects))

	if len(topProjects) > 0 {
		sb.WriteString("### 🏆 Top 活跃项目\n")
		for i, p := range topProjects {
			sb.WriteString(fmt.Sprintf("%d. %s - %d 次提交，均分 %.0f\n", i+1, p.Name, p.CommitCount, p.AvgScore))
		}
		sb.WriteString("\n")
	}

	if len(lowScores) > 0 {
		sb.WriteString("### ⚠️ 需关注\n")
		for _, l := range lowScores {
			sb.WriteString(fmt.Sprintf("- %s (%s): %.0f 分\n", l.Project, l.Author, l.Score))
		}
	}

	return sb.String()
}

func (s *DailyReportService) getIMBotIDs() []uint {
	var config models.SystemConfig
	if err := s.db.Where("`key` = ?", "daily_report_im_bot_ids").First(&config).Error; err != nil {
		return nil
	}
	if config.Value == "" {
		return nil
	}
	var ids []uint
	for _, idStr := range strings.Split(config.Value, ",") {
		idStr = strings.TrimSpace(idStr)
		if id, err := strconv.ParseUint(idStr, 10, 64); err == nil {
			ids = append(ids, uint(id))
		}
	}
	return ids
}

func (s *DailyReportService) sendNotifications(report *models.DailyReport) error {
	var bots []models.IMBot

	botIDs := s.getIMBotIDs()
	if len(botIDs) > 0 {
		if err := s.db.Where("id IN ? AND is_active = ?", botIDs, true).Find(&bots).Error; err != nil {
			return err
		}
	} else {
		if err := s.db.Where("is_active = ? AND daily_report_enabled = ?", true, true).Find(&bots).Error; err != nil {
			return err
		}
	}

	if len(bots) == 0 {
		log.Println("[DailyReport] No bots enabled for daily report")
		return nil
	}

	message := report.AIAnalysis
	if message == "" {
		message = s.buildDefaultSummary(
			ReportStats{
				TotalProjects: report.TotalProjects,
				TotalCommits:  report.TotalCommits,
				TotalAuthors:  report.TotalAuthors,
				AverageScore:  report.AverageScore,
				PassedCount:   report.PassedCount,
				FailedCount:   report.FailedCount,
			},
			nil, nil, nil,
		)
	}

	var lastErr error
	successCount := 0
	for _, bot := range bots {
		if err := s.notificationService.SendErrorNotification(&bot, message); err != nil {
			log.Printf("[DailyReport] Failed to send to bot %s: %v", bot.Name, err)
			lastErr = err
		} else {
			log.Printf("[DailyReport] Sent to bot %s", bot.Name)
			successCount++
		}
	}

	if successCount == 0 && lastErr != nil {
		return lastErr
	}
	return nil
}

func (s *DailyReportService) List(page, pageSize int) ([]models.DailyReport, int64, error) {
	var reports []models.DailyReport
	var total int64

	s.db.Model(&models.DailyReport{}).Count(&total)

	offset := (page - 1) * pageSize
	if err := s.db.Order("report_date DESC").Offset(offset).Limit(pageSize).Find(&reports).Error; err != nil {
		return nil, 0, err
	}

	return reports, total, nil
}

func (s *DailyReportService) GetByID(id uint) (*models.DailyReport, error) {
	var report models.DailyReport
	if err := s.db.First(&report, id).Error; err != nil {
		return nil, err
	}
	return &report, nil
}

func (s *DailyReportService) ResendNotification(id uint) error {
	report, err := s.GetByID(id)
	if err != nil {
		return err
	}

	if err := s.sendNotifications(report); err != nil {
		report.NotifyError = err.Error()
		s.db.Save(report)
		return err
	}

	now := time.Now()
	report.NotifiedAt = &now
	report.NotifyError = ""
	s.db.Save(report)

	return nil
}
