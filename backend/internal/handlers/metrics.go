package handlers

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/huangang/codesentry/backend/internal/models"
	"github.com/huangang/codesentry/backend/internal/services"
)

var startTime = time.Now()

// Metrics returns Prometheus-compatible text format metrics.
func Metrics(c *gin.Context) {
	var b strings.Builder

	// -- Runtime metrics --
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	writeGauge(&b, "codesentry_uptime_seconds", "Time since server start in seconds", float64(time.Since(startTime).Seconds()))
	writeGauge(&b, "codesentry_goroutines", "Number of active goroutines", float64(runtime.NumGoroutine()))
	writeGauge(&b, "codesentry_memory_alloc_bytes", "Current heap allocation in bytes", float64(m.Alloc))
	writeGauge(&b, "codesentry_memory_sys_bytes", "Total memory obtained from OS in bytes", float64(m.Sys))
	writeGauge(&b, "codesentry_gc_runs_total", "Total number of GC runs", float64(m.NumGC))

	// -- Database metrics --
	db := models.GetDB()
	if db != nil {
		if sqlDB, err := db.DB(); err == nil {
			stats := sqlDB.Stats()
			writeGauge(&b, "codesentry_db_open_connections", "Number of open DB connections", float64(stats.OpenConnections))
			writeGauge(&b, "codesentry_db_in_use_connections", "Number of in-use DB connections", float64(stats.InUse))
			writeGauge(&b, "codesentry_db_idle_connections", "Number of idle DB connections", float64(stats.Idle))
		}
	}

	// -- SSE metrics --
	sseHub := services.GetSSEHub()
	if sseHub != nil {
		writeGauge(&b, "codesentry_sse_active_clients", "Number of active SSE connections", float64(sseHub.ClientCount()))
	}

	// -- Queue metrics --
	taskQueue := services.GetTaskQueue()
	queueAsync := 0.0
	if taskQueue != nil && taskQueue.IsAsync() {
		queueAsync = 1.0
	}
	writeGauge(&b, "codesentry_queue_async_enabled", "Whether async queue (Redis) is enabled (1=yes, 0=no)", queueAsync)

	// -- Review metrics --
	if db != nil {
		var totalReviews, pendingReviews, analyzingReviews, completedReviews, failedReviews int64
		db.Model(&models.ReviewLog{}).Where("deleted_at IS NULL").Count(&totalReviews)
		db.Model(&models.ReviewLog{}).Where("review_status = ? AND deleted_at IS NULL", "pending").Count(&pendingReviews)
		db.Model(&models.ReviewLog{}).Where("review_status = ? AND deleted_at IS NULL", "analyzing").Count(&analyzingReviews)
		db.Model(&models.ReviewLog{}).Where("review_status = ? AND deleted_at IS NULL", "completed").Count(&completedReviews)
		db.Model(&models.ReviewLog{}).Where("review_status = ? AND deleted_at IS NULL", "failed").Count(&failedReviews)

		writeGauge(&b, "codesentry_reviews_total", "Total number of review logs", float64(totalReviews))
		writeGauge(&b, "codesentry_reviews_pending", "Number of pending reviews", float64(pendingReviews))
		writeGauge(&b, "codesentry_reviews_analyzing", "Number of currently analyzing reviews", float64(analyzingReviews))
		writeGauge(&b, "codesentry_reviews_completed", "Number of completed reviews", float64(completedReviews))
		writeGauge(&b, "codesentry_reviews_failed", "Number of failed reviews", float64(failedReviews))

		// Projects & Users
		var projectCount, userCount int64
		db.Model(&models.Project{}).Where("deleted_at IS NULL").Count(&projectCount)
		db.Model(&models.User{}).Where("deleted_at IS NULL AND is_active = ?", true).Count(&userCount)

		writeGauge(&b, "codesentry_projects_total", "Total number of active projects", float64(projectCount))
		writeGauge(&b, "codesentry_users_active", "Number of active users", float64(userCount))

		// AI Usage (last 24h)
		since24h := time.Now().Add(-24 * time.Hour)
		var aiCalls24h int64
		db.Model(&models.AIUsageLog{}).Where("created_at >= ?", since24h).Count(&aiCalls24h)
		writeGauge(&b, "codesentry_ai_calls_24h", "AI API calls in the last 24 hours", float64(aiCalls24h))
	}

	c.Data(200, "text/plain; version=0.0.4; charset=utf-8", []byte(b.String()))
}

func writeGauge(b *strings.Builder, name, help string, value float64) {
	fmt.Fprintf(b, "# HELP %s %s\n", name, help)
	fmt.Fprintf(b, "# TYPE %s gauge\n", name)
	fmt.Fprintf(b, "%s %g\n\n", name, value)
}
