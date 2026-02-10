package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/huangang/codesentry/backend/internal/services"
	"github.com/huangang/codesentry/backend/internal/utils"
	"github.com/huangang/codesentry/backend/pkg/logger"
	"github.com/huangang/codesentry/backend/pkg/response"
)

// SSEHandler handles Server-Sent Events for real-time updates
type SSEHandler struct {
	hub       *services.SSEHub
	importHub *services.ImportEventHub
}

// NewSSEHandler creates a new SSE handler
func NewSSEHandler(hub *services.SSEHub) *SSEHandler {
	return &SSEHandler{
		hub:       hub,
		importHub: services.GetImportHub(),
	}
}

// StreamReviewEvents handles SSE connections for review status updates
func (h *SSEHandler) StreamReviewEvents(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		authHeader := c.GetHeader("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	if token == "" {
		response.Unauthorized(c, "Unauthorized")
		return
	}

	if _, err := utils.ParseToken(token); err != nil {
		response.Unauthorized(c, "Invalid token")
		return
	}

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Header("Access-Control-Allow-Origin", "*")

	clientID := uuid.New().String()

	events := h.hub.Subscribe(clientID)
	defer h.hub.Unsubscribe(clientID)

	logger.Info().Str("client_id", clientID).Int("total", h.hub.ClientCount()).Msg("SSE client connected")

	c.Stream(func(w io.Writer) bool {
		select {
		case event, ok := <-events:
			if !ok {
				return false
			}
			data, err := json.Marshal(event)
			if err != nil {
				logger.Error().Err(err).Msg("SSE marshal error")
				return true
			}
			fmt.Fprintf(w, "data: %s\n\n", data)
			c.Writer.Flush()
			return true
		case <-c.Request.Context().Done():
			logger.Info().Str("client_id", clientID).Msg("SSE client disconnected")
			return false
		}
	})
}

func (h *SSEHandler) StreamImportEvents(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		authHeader := c.GetHeader("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	if token == "" {
		response.Unauthorized(c, "Unauthorized")
		return
	}

	if _, err := utils.ParseToken(token); err != nil {
		response.Unauthorized(c, "Invalid token")
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Header("Access-Control-Allow-Origin", "*")

	clientID := uuid.New().String()
	events := h.importHub.Subscribe(clientID)
	defer h.importHub.Unsubscribe(clientID)

	logger.Info().Str("client_id", clientID).Msg("Import SSE client connected")

	c.Stream(func(w io.Writer) bool {
		select {
		case event, ok := <-events:
			if !ok {
				return false
			}
			data, err := json.Marshal(event)
			if err != nil {
				logger.Error().Err(err).Msg("Import SSE marshal error")
				return true
			}
			fmt.Fprintf(w, "data: %s\n\n", data)
			c.Writer.Flush()
			return true
		case <-c.Request.Context().Done():
			logger.Info().Str("client_id", clientID).Msg("Import SSE client disconnected")
			return false
		}
	})
}
