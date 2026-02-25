package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/huangang/codesentry/backend/internal/models"
	"github.com/huangang/codesentry/backend/internal/services"
)

// HealthHandler provides enhanced health check endpoints.
type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// CheckHealth returns the health status of all subsystems.
func (h *HealthHandler) CheckHealth(c *gin.Context) {
	overall := "healthy"

	// Database check
	dbStatus := "ok"
	sqlDB, err := models.GetDB().DB()
	if err != nil {
		dbStatus = "error: " + err.Error()
		overall = "unhealthy"
	} else if err := sqlDB.Ping(); err != nil {
		dbStatus = "error: " + err.Error()
		overall = "unhealthy"
	}

	// Queue mode
	taskQueue := services.GetTaskQueue()
	queueMode := "sync"
	if taskQueue != nil && taskQueue.IsAsync() {
		queueMode = "async (Redis)"
	}

	// SSE connections
	sseClients := services.GetSSEHub().ClientCount()

	// Pending/analyzing review count
	var pendingCount int64
	models.GetDB().Model(&models.ReviewLog{}).
		Where("review_status IN ?", []string{"pending", "analyzing"}).
		Count(&pendingCount)

	c.JSON(200, gin.H{
		"status":  overall,
		"service": "codesentry",
		"components": gin.H{
			"database":        dbStatus,
			"queue_mode":      queueMode,
			"sse_clients":     sseClients,
			"pending_reviews": pendingCount,
		},
	})
}
