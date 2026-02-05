package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/huangang/codesentry/backend/internal/services"
	"github.com/huangang/codesentry/backend/internal/utils"
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
// GET /api/events/reviews
func (h *SSEHandler) StreamReviewEvents(c *gin.Context) {
	// Verify authentication - check token from query param or header
	token := c.Query("token")
	if token == "" {
		authHeader := c.GetHeader("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	if token == "" {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}

	// Validate token
	_, err := utils.ParseToken(token)
	if err != nil {
		c.JSON(401, gin.H{"error": "Invalid token"})
		return
	}

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // Disable Nginx buffering
	c.Header("Access-Control-Allow-Origin", "*")

	// Generate unique client ID
	clientID := uuid.New().String()

	// Subscribe to events
	events := h.hub.Subscribe(clientID)
	defer h.hub.Unsubscribe(clientID)

	log.Printf("SSE client connected: %s (total: %d)", clientID, h.hub.ClientCount())

	// Send initial connection message
	c.Stream(func(w io.Writer) bool {
		select {
		case event, ok := <-events:
			if !ok {
				// Channel closed, exit
				return false
			}
			data, err := json.Marshal(event)
			if err != nil {
				log.Printf("SSE marshal error: %v", err)
				return true
			}
			fmt.Fprintf(w, "data: %s\n\n", data)
			c.Writer.Flush()
			return true
		case <-c.Request.Context().Done():
			// Client disconnected
			log.Printf("SSE client disconnected: %s", clientID)
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
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}

	if _, err := utils.ParseToken(token); err != nil {
		c.JSON(401, gin.H{"error": "Invalid token"})
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

	log.Printf("Import SSE client connected: %s", clientID)

	c.Stream(func(w io.Writer) bool {
		select {
		case event, ok := <-events:
			if !ok {
				return false
			}
			data, err := json.Marshal(event)
			if err != nil {
				log.Printf("Import SSE marshal error: %v", err)
				return true
			}
			fmt.Fprintf(w, "data: %s\n\n", data)
			c.Writer.Flush()
			return true
		case <-c.Request.Context().Done():
			log.Printf("Import SSE client disconnected: %s", clientID)
			return false
		}
	})
}
