package services

import (
	"testing"
	"time"
)

func TestSSEHub_NewSSEHub(t *testing.T) {
	hub := NewSSEHub()
	if hub == nil {
		t.Fatal("NewSSEHub should not return nil")
	}
	if hub.clients == nil {
		t.Error("clients map should be initialized")
	}
	if hub.ClientCount() != 0 {
		t.Errorf("new hub should have 0 clients, got %d", hub.ClientCount())
	}
}

func TestSSEHub_Subscribe(t *testing.T) {
	hub := NewSSEHub()

	ch := hub.Subscribe("client1")
	if ch == nil {
		t.Error("Subscribe should return a channel")
	}
	if hub.ClientCount() != 1 {
		t.Errorf("expected 1 client, got %d", hub.ClientCount())
	}

	ch2 := hub.Subscribe("client2")
	if ch2 == nil {
		t.Error("Subscribe should return a channel")
	}
	if hub.ClientCount() != 2 {
		t.Errorf("expected 2 clients, got %d", hub.ClientCount())
	}
}

func TestSSEHub_Unsubscribe(t *testing.T) {
	hub := NewSSEHub()

	hub.Subscribe("client1")
	hub.Subscribe("client2")

	if hub.ClientCount() != 2 {
		t.Fatalf("expected 2 clients, got %d", hub.ClientCount())
	}

	hub.Unsubscribe("client1")
	if hub.ClientCount() != 1 {
		t.Errorf("expected 1 client after unsubscribe, got %d", hub.ClientCount())
	}

	hub.Unsubscribe("nonexistent")
	if hub.ClientCount() != 1 {
		t.Errorf("unsubscribing nonexistent should not affect count, got %d", hub.ClientCount())
	}

	hub.Unsubscribe("client2")
	if hub.ClientCount() != 0 {
		t.Errorf("expected 0 clients, got %d", hub.ClientCount())
	}
}

func TestSSEHub_Publish(t *testing.T) {
	hub := NewSSEHub()

	ch := hub.Subscribe("client1")

	score := 85.0
	event := ReviewEvent{
		ID:        1,
		ProjectID: 10,
		CommitSHA: "abc123",
		Status:    "completed",
		Score:     &score,
	}

	hub.Publish(event)

	select {
	case received := <-ch:
		if received.ID != event.ID {
			t.Errorf("ID = %d, expected %d", received.ID, event.ID)
		}
		if received.Status != "completed" {
			t.Errorf("Status = %q, expected %q", received.Status, "completed")
		}
		if *received.Score != 85.0 {
			t.Errorf("Score = %f, expected 85.0", *received.Score)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timed out waiting for event")
	}
}

func TestSSEHub_PublishMultipleClients(t *testing.T) {
	hub := NewSSEHub()

	ch1 := hub.Subscribe("client1")
	ch2 := hub.Subscribe("client2")

	event := ReviewEvent{
		ID:     1,
		Status: "pending",
	}

	hub.Publish(event)

	for i, ch := range []<-chan ReviewEvent{ch1, ch2} {
		select {
		case received := <-ch:
			if received.ID != 1 {
				t.Errorf("client%d: ID = %d, expected 1", i+1, received.ID)
			}
		case <-time.After(100 * time.Millisecond):
			t.Errorf("client%d: timed out waiting for event", i+1)
		}
	}
}

func TestSSEHub_NonBlockingPublish(t *testing.T) {
	hub := NewSSEHub()

	hub.Subscribe("slow_client")

	for i := 0; i < 200; i++ {
		hub.Publish(ReviewEvent{ID: uint(i)})
	}
}

func TestReviewEvent_Structure(t *testing.T) {
	score := 75.5
	event := ReviewEvent{
		ID:        42,
		ProjectID: 10,
		CommitSHA: "def456",
		Status:    "analyzing",
		Score:     &score,
		Error:     "",
	}

	if event.ID != 42 {
		t.Errorf("ID = %d, expected 42", event.ID)
	}
	if event.ProjectID != 10 {
		t.Errorf("ProjectID = %d, expected 10", event.ProjectID)
	}
	if event.CommitSHA != "def456" {
		t.Errorf("CommitSHA = %q, expected %q", event.CommitSHA, "def456")
	}
	if event.Status != "analyzing" {
		t.Errorf("Status = %q, expected %q", event.Status, "analyzing")
	}
	if event.Score == nil || *event.Score != 75.5 {
		t.Error("Score should be 75.5")
	}
	if event.Error != "" {
		t.Errorf("Error should be empty, got %q", event.Error)
	}
}

func TestReviewEvent_WithError(t *testing.T) {
	event := ReviewEvent{
		ID:     1,
		Status: "failed",
		Error:  "API timeout",
	}

	if event.ID != 1 {
		t.Errorf("ID = %d, expected 1", event.ID)
	}
	if event.Status != "failed" {
		t.Errorf("Status = %q, expected %q", event.Status, "failed")
	}
	if event.Error != "API timeout" {
		t.Errorf("Error = %q, expected %q", event.Error, "API timeout")
	}
}

func TestGetSSEHub_Singleton(t *testing.T) {
	hub1 := GetSSEHub()
	hub2 := GetSSEHub()

	if hub1 != hub2 {
		t.Error("GetSSEHub should return the same instance")
	}
}
