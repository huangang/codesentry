package services

import (
	"encoding/json"
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
	db *gorm.DB
}

func NewSystemLogService(db *gorm.DB) *SystemLogService {
	return &SystemLogService{db: db}
}

type SystemLogListRequest struct {
	Page      int    `form:"page" binding:"min=1"`
	PageSize  int    `form:"page_size" binding:"min=1,max=100"`
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
	var cfg models.SystemConfig
	if err := s.db.Where("config_key = ?", "log_retention_days").First(&cfg).Error; err != nil {
		return 30 // default 30 days
	}

	days, err := strconv.Atoi(cfg.Value)
	if err != nil {
		return 30
	}
	return days
}

// SetRetentionDays sets the log retention days in system config
func (s *SystemLogService) SetRetentionDays(days int) error {
	return s.db.Model(&models.SystemConfig{}).
		Where("config_key = ?", "log_retention_days").
		Update("value", strconv.Itoa(days)).Error
}

// StartCleanupScheduler starts a goroutine that periodically cleans up old logs
func StartLogCleanupScheduler(db *gorm.DB) {
	go func() {
		service := NewSystemLogService(db)

		// Run cleanup immediately on startup
		runCleanup(service)

		// Then run every 24 hours
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			runCleanup(service)
		}
	}()
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
