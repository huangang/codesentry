package main

import (
	"context"
	"embed"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/huangang/codesentry/backend/internal/config"
	"github.com/huangang/codesentry/backend/internal/models"
	"github.com/huangang/codesentry/backend/pkg/logger"
)

//go:embed static/*
var staticFiles embed.FS

func maskDSN(dsn string) string {
	if idx := strings.Index(dsn, "@"); idx > 0 {
		return "***@" + dsn[idx+1:]
	}
	return "***"
}

func main() {
	// Load configuration
	cfg, err := config.Load(os.Getenv("CONFIG_PATH"))
	if err != nil {
		logger.Fatalf("Failed to load config: %v", err)
	}

	// Initialize structured logger
	logLevel := "info"
	if cfg.Server.Mode == "debug" {
		logLevel = "debug"
	}
	logger.Init(logLevel)

	logger.Info().Str("driver", cfg.Database.Driver).Str("dsn", maskDSN(cfg.Database.DSN)).Msg("Config loaded")

	// Bootstrap all services
	svc := bootstrap(cfg)

	// Set Gin mode and create router
	gin.SetMode(cfg.Server.Mode)
	r := gin.New()

	// Register all routes
	registerRoutes(r, svc)

	// Serve static files (embedded frontend)
	serveStaticFiles(r)

	// Start server with graceful shutdown
	addr := cfg.Server.Host + ":" + cfg.Server.Port
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	go func() {
		logger.Info().Str("addr", addr).Msg("Server starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info().Msg("Shutting down server...")

	// Shutdown services
	svc.shutdown()

	// Graceful shutdown with 30 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatalf("Server forced to shutdown: %v", err)
	}

	// Close database connection
	if sqlDB, err := models.GetDB().DB(); err == nil {
		sqlDB.Close()
		logger.Info().Msg("Database connection closed")
	}

	logger.Info().Msg("Server exited gracefully")
}

// serveStaticFiles configures embedded frontend file serving and SPA fallback.
func serveStaticFiles(r *gin.Engine) {
	staticFS, staticErr := fs.Sub(staticFiles, "static")
	if staticErr != nil {
		return
	}

	// Helper function to serve index.html
	serveIndex := func(c *gin.Context) {
		data, readErr := fs.ReadFile(staticFS, "index.html")
		if readErr != nil {
			c.String(404, "index.html not found")
			return
		}
		c.Data(200, "text/html; charset=utf-8", data)
	}

	// Serve index.html for root path
	r.GET("/", serveIndex)

	r.NoRoute(func(c *gin.Context) {
		// Try to serve static file
		path := c.Request.URL.Path[1:] // Remove leading /

		data, readErr := fs.ReadFile(staticFS, path)
		if readErr != nil {
			// Fallback to index.html for SPA routing
			serveIndex(c)
			return
		}

		// Determine content type using standard library
		ext := filepath.Ext(path)
		contentType := mime.TypeByExtension(ext)
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		c.Data(200, contentType, data)
	})
}
