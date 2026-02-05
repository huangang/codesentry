package services

import (
	"sync"
)

// ReviewEvent represents a real-time review status update event
type ReviewEvent struct {
	ID        uint     `json:"id"`
	ProjectID uint     `json:"project_id"`
	CommitSHA string   `json:"commit_sha"`
	Status    string   `json:"status"` // pending, analyzing, completed, failed
	Score     *float64 `json:"score,omitempty"`
	Error     string   `json:"error,omitempty"`
}

// ImportEvent represents a commit import completion event
type ImportEvent struct {
	ProjectID   uint   `json:"project_id"`
	ProjectName string `json:"project_name"`
	Imported    int    `json:"imported"`
	Skipped     int    `json:"skipped"`
	Error       string `json:"error,omitempty"`
}

// SSEHub manages SSE client connections and event broadcasting
type SSEHub struct {
	clients map[string]chan ReviewEvent
	mu      sync.RWMutex
}

// NewSSEHub creates a new SSE hub instance
func NewSSEHub() *SSEHub {
	return &SSEHub{
		clients: make(map[string]chan ReviewEvent),
	}
}

// Subscribe registers a new client and returns a channel for receiving events
func (h *SSEHub) Subscribe(clientID string) <-chan ReviewEvent {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Create buffered channel to prevent blocking
	ch := make(chan ReviewEvent, 100)
	h.clients[clientID] = ch
	return ch
}

// Unsubscribe removes a client from the hub
func (h *SSEHub) Unsubscribe(clientID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if ch, ok := h.clients[clientID]; ok {
		close(ch)
		delete(h.clients, clientID)
	}
}

// Publish broadcasts an event to all connected clients
func (h *SSEHub) Publish(event ReviewEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, ch := range h.clients {
		// Non-blocking send - drop event if client buffer is full
		select {
		case ch <- event:
		default:
			// Client is slow, skip this event
		}
	}
}

// ClientCount returns the number of connected clients
func (h *SSEHub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// Global SSE Hub instance
var globalSSEHub *SSEHub
var sseHubOnce sync.Once

// GetSSEHub returns the global SSE hub singleton
func GetSSEHub() *SSEHub {
	sseHubOnce.Do(func() {
		globalSSEHub = NewSSEHub()
	})
	return globalSSEHub
}

// PublishReviewEvent is a convenience function to publish review events
func PublishReviewEvent(id uint, projectID uint, commitSHA, status string, score *float64, errMsg string) {
	GetSSEHub().Publish(ReviewEvent{
		ID:        id,
		ProjectID: projectID,
		CommitSHA: commitSHA,
		Status:    status,
		Score:     score,
		Error:     errMsg,
	})
}

// ImportEventHub manages import event subscribers
type ImportEventHub struct {
	clients map[string]chan ImportEvent
	mu      sync.RWMutex
}

var globalImportHub *ImportEventHub
var importHubOnce sync.Once

func GetImportHub() *ImportEventHub {
	importHubOnce.Do(func() {
		globalImportHub = &ImportEventHub{
			clients: make(map[string]chan ImportEvent),
		}
	})
	return globalImportHub
}

func (h *ImportEventHub) Subscribe(clientID string) <-chan ImportEvent {
	h.mu.Lock()
	defer h.mu.Unlock()
	ch := make(chan ImportEvent, 10)
	h.clients[clientID] = ch
	return ch
}

func (h *ImportEventHub) Unsubscribe(clientID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if ch, ok := h.clients[clientID]; ok {
		close(ch)
		delete(h.clients, clientID)
	}
}

func (h *ImportEventHub) Publish(event ImportEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, ch := range h.clients {
		select {
		case ch <- event:
		default:
		}
	}
}

func PublishImportEvent(projectID uint, projectName string, imported, skipped int, errMsg string) {
	GetImportHub().Publish(ImportEvent{
		ProjectID:   projectID,
		ProjectName: projectName,
		Imported:    imported,
		Skipped:     skipped,
		Error:       errMsg,
	})
}
