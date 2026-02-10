package logger

import (
	"io"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

var log zerolog.Logger

// Init initializes the global logger with the specified level.
// level can be: "debug", "info", "warn", "error", "fatal"
// In development mode (debug level), output is human-friendly console format.
func Init(level string) {
	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		lvl = zerolog.InfoLevel
	}

	var writer io.Writer
	if lvl == zerolog.DebugLevel {
		writer = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: "15:04:05"}
	} else {
		writer = os.Stdout
	}

	log = zerolog.New(writer).
		Level(lvl).
		With().
		Timestamp().
		Caller().
		Logger()
}

func init() {
	// Default logger before Init() is called
	Init("info")
}

// --- Convenience functions ---

func Debug() *zerolog.Event { return log.Debug() }
func Info() *zerolog.Event  { return log.Info() }
func Warn() *zerolog.Event  { return log.Warn() }
func Error() *zerolog.Event { return log.Error() }
func Fatal() *zerolog.Event { return log.Fatal() }

// Infof provides printf-style logging at info level.
func Infof(format string, v ...interface{}) {
	log.Info().Msgf(format, v...)
}

// Errorf provides printf-style logging at error level.
func Errorf(format string, v ...interface{}) {
	log.Error().Msgf(format, v...)
}

// Warnf provides printf-style logging at warn level.
func Warnf(format string, v ...interface{}) {
	log.Warn().Msgf(format, v...)
}

// Fatalf provides printf-style logging at fatal level (calls os.Exit).
func Fatalf(format string, v ...interface{}) {
	log.Fatal().Msgf(format, v...)
}

// Get returns the underlying zerolog.Logger for advanced usage.
func Get() zerolog.Logger {
	return log
}

// GinLogger returns a Gin middleware that logs HTTP requests using zerolog.
func GinLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		event := log.Info()
		if status >= 500 {
			event = log.Error()
		} else if status >= 400 {
			event = log.Warn()
		}

		event.
			Int("status", status).
			Str("method", c.Request.Method).
			Str("path", path).
			Str("query", query).
			Str("ip", c.ClientIP()).
			Dur("latency", latency).
			Int("size", c.Writer.Size()).
			Msg("request")
	}
}

// GinRecovery returns a Gin recovery middleware that logs panics using zerolog.
func GinRecovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		log.Error().
			Interface("panic", recovered).
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Str("ip", c.ClientIP()).
			Msg("panic recovered")
		c.AbortWithStatus(500)
	})
}
