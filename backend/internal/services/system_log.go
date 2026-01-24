package services

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/huangang/codesentry/backend/internal/models"
	"gorm.io/gorm"
)

var globalDB *gorm.DB

func InitSystemLogger(db *gorm.DB) {
	globalDB = db
}

func LogInfo(module, action, message string, userID *uint, ip, userAgent string, extra interface{}) {
	writeLog("info", module, action, message, userID, ip, userAgent, extra)
}

func LogWarning(module, action, message string, userID *uint, ip, userAgent string, extra interface{}) {
	writeLog("warning", module, action, message, userID, ip, userAgent, extra)
}

func LogError(module, action, message string, userID *uint, ip, userAgent string, extra interface{}) {
	writeLog("error", module, action, message, userID, ip, userAgent, extra)
	go sendErrorNotification(module, action, message, extra)
}

func sendErrorNotification(module, action, message string, extra interface{}) {
	if globalDB == nil {
		return
	}

	var bots []models.IMBot
	if err := globalDB.Where("is_active = ? AND error_notify = ?", true, true).Find(&bots).Error; err != nil {
		log.Printf("[SystemLog] Failed to get error notify bots: %v", err)
		return
	}

	if len(bots) == 0 {
		return
	}

	notificationService := NewNotificationService(globalDB)
	errorMsg := buildErrorMessage(module, action, message, extra)

	for _, bot := range bots {
		if err := notificationService.SendErrorNotification(&bot, errorMsg); err != nil {
			log.Printf("[SystemLog] Failed to send error notification to bot %s: %v", bot.Name, err)
		}
	}
}

func buildErrorMessage(module, action, message string, extra interface{}) string {
	msg := fmt.Sprintf(`🚨 **System Error Alert**

**Module**: %s
**Action**: %s
**Message**: %s
**Time**: %s`, module, action, message, time.Now().Format("2006-01-02 15:04:05"))

	if extra != nil {
		if b, err := json.Marshal(extra); err == nil && len(b) > 2 {
			extraStr := string(b)
			if len(extraStr) > 500 {
				extraStr = extraStr[:500] + "..."
			}
			msg += fmt.Sprintf("\n\n**Extra**: ```%s```", extraStr)
		}
	}

	return msg
}

func writeLog(level, module, action, message string, userID *uint, ip, userAgent string, extra interface{}) {
	if globalDB == nil {
		return
	}

	var extraStr string
	if extra != nil {
		if b, err := json.Marshal(extra); err == nil {
			extraStr = string(b)
		}
	}

	log := &models.SystemLog{
		Level:     level,
		Module:    module,
		Action:    action,
		Message:   message,
		UserID:    userID,
		IP:        ip,
		UserAgent: userAgent,
		Extra:     extraStr,
		CreatedAt: time.Now(),
	}
	globalDB.Create(log)
}

type SystemLogService struct {
	db            *gorm.DB
	configService *SystemConfigService
}

func NewSystemLogService(db *gorm.DB) *SystemLogService {
	return &SystemLogService{
		db:            db,
		configService: NewSystemConfigService(db),
	}
}

type SystemLogListRequest struct {
	Page      int    `form:"page" binding:"omitempty,min=1"`
	PageSize  int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	Level     string `form:"level"`
	Module    string `form:"module"`
	Action    string `form:"action"`
	StartDate string `form:"start_date"`
	EndDate   string `form:"end_date"`
	Search    string `form:"search"`
}

type SystemLogListResponse struct {
	Total    int64              `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"page_size"`
	Items    []models.SystemLog `json:"items"`
}

func (s *SystemLogService) List(req *SystemLogListRequest) (*SystemLogListResponse, error) {
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 20
	}

	var logs []models.SystemLog
	var total int64

	query := s.db.Model(&models.SystemLog{})

	if req.Level != "" {
		query = query.Where("level = ?", req.Level)
	}
	if req.Module != "" {
		query = query.Where("module = ?", req.Module)
	}
	if req.Action != "" {
		query = query.Where("action LIKE ?", "%"+req.Action+"%")
	}
	if req.StartDate != "" {
		query = query.Where("created_at >= ?", req.StartDate)
	}
	if req.EndDate != "" {
		query = query.Where("created_at <= ?", req.EndDate+" 23:59:59")
	}
	if req.Search != "" {
		query = query.Where("message LIKE ?", "%"+req.Search+"%")
	}

	query.Count(&total)

	offset := (req.Page - 1) * req.PageSize
	if err := query.Offset(offset).Limit(req.PageSize).Order("created_at DESC").Find(&logs).Error; err != nil {
		return nil, err
	}

	return &SystemLogListResponse{
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
		Items:    logs,
	}, nil
}

func (s *SystemLogService) GetModules() ([]string, error) {
	var modules []string
	if err := s.db.Model(&models.SystemLog{}).Distinct("module").Pluck("module", &modules).Error; err != nil {
		return nil, err
	}
	return modules, nil
}

func (s *SystemLogService) Create(log *models.SystemLog) error {
	return s.db.Create(log).Error
}

// CleanupOldLogs deletes logs older than the specified number of days
// Returns the number of deleted records
func (s *SystemLogService) CleanupOldLogs(retentionDays int) (int64, error) {
	if retentionDays <= 0 {
		return 0, nil
	}

	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)
	result := s.db.Where("created_at < ?", cutoffTime).Delete(&models.SystemLog{})
	if result.Error != nil {
		return 0, result.Error
	}

	return result.RowsAffected, nil
}

// GetRetentionDays gets the log retention days from system config
func (s *SystemLogService) GetRetentionDays() int {
	days, err := strconv.Atoi(s.configService.GetWithDefault("log_retention_days", "30"))
	if err != nil {
		return 30
	}
	return days
}

// SetRetentionDays sets the log retention days in system config
func (s *SystemLogService) SetRetentionDays(days int) error {
	return s.configService.Set("log_retention_days", strconv.Itoa(days))
}

var logCleanupStopChan chan struct{}

// StartLogCleanupScheduler starts a goroutine that periodically cleans up old logs
func StartLogCleanupScheduler(db *gorm.DB) {
	logCleanupStopChan = make(chan struct{})
	go func() {
		service := NewSystemLogService(db)

		// Run cleanup immediately on startup
		runCleanup(service)

		// Then run every 24 hours
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				runCleanup(service)
			case <-logCleanupStopChan:
				log.Println("[SystemLog] Log cleanup scheduler stopped")
				return
			}
		}
	}()
}

// StopLogCleanupScheduler stops the log cleanup scheduler
func StopLogCleanupScheduler() {
	if logCleanupStopChan != nil {
		close(logCleanupStopChan)
	}
}

func runCleanup(service *SystemLogService) {
	retentionDays := service.GetRetentionDays()
	if retentionDays <= 0 {
		log.Println("[SystemLog] Log cleanup disabled (retention_days <= 0)")
		return
	}

	deleted, err := service.CleanupOldLogs(retentionDays)
	if err != nil {
		log.Printf("[SystemLog] Failed to cleanup old logs: %v", err)
		return
	}

	if deleted > 0 {
		log.Printf("[SystemLog] Cleaned up %d logs older than %d days", deleted, retentionDays)
	}
}
