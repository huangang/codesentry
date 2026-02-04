package services

import (
	"context"
	"testing"
)

func TestTaskTypeReview_Constant(t *testing.T) {
	if TaskTypeReview != "review:process" {
		t.Errorf("TaskTypeReview = %q, expected %q", TaskTypeReview, "review:process")
	}
}

func TestReviewTask_Structure(t *testing.T) {
	mrNum := 42
	task := ReviewTask{
		ReviewLogID:     1,
		ProjectID:       10,
		CommitSHA:       "abc123",
		EventType:       "push",
		Branch:          "main",
		Author:          "developer",
		AuthorEmail:     "dev@example.com",
		AuthorAvatar:    "https://avatar.url",
		CommitMessage:   "feat: add feature",
		Diff:            "diff content",
		CommitURL:       "https://github.com/org/repo/commit/abc123",
		MRNumber:        &mrNum,
		MRURL:           "https://github.com/org/repo/pull/42",
		GitLabProjectID: 123,
	}

	if task.ReviewLogID != 1 {
		t.Errorf("ReviewLogID = %d, expected 1", task.ReviewLogID)
	}
	if task.ProjectID != 10 {
		t.Errorf("ProjectID = %d, expected 10", task.ProjectID)
	}
	if task.CommitSHA != "abc123" {
		t.Errorf("CommitSHA = %q, expected %q", task.CommitSHA, "abc123")
	}
	if task.EventType != "push" {
		t.Errorf("EventType = %q, expected %q", task.EventType, "push")
	}
	if task.Branch != "main" {
		t.Errorf("Branch = %q, expected %q", task.Branch, "main")
	}
	if task.Author != "developer" {
		t.Errorf("Author = %q, expected %q", task.Author, "developer")
	}
	if task.AuthorEmail != "dev@example.com" {
		t.Errorf("AuthorEmail = %q, expected %q", task.AuthorEmail, "dev@example.com")
	}
	if task.AuthorAvatar != "https://avatar.url" {
		t.Errorf("AuthorAvatar = %q, expected %q", task.AuthorAvatar, "https://avatar.url")
	}
	if task.CommitMessage != "feat: add feature" {
		t.Errorf("CommitMessage = %q, expected %q", task.CommitMessage, "feat: add feature")
	}
	if task.Diff != "diff content" {
		t.Errorf("Diff = %q, expected %q", task.Diff, "diff content")
	}
	if task.CommitURL != "https://github.com/org/repo/commit/abc123" {
		t.Errorf("CommitURL = %q, expected %q", task.CommitURL, "https://github.com/org/repo/commit/abc123")
	}
	if task.MRNumber == nil || *task.MRNumber != 42 {
		t.Error("MRNumber should be 42")
	}
	if task.MRURL != "https://github.com/org/repo/pull/42" {
		t.Errorf("MRURL = %q, expected %q", task.MRURL, "https://github.com/org/repo/pull/42")
	}
	if task.GitLabProjectID != 123 {
		t.Errorf("GitLabProjectID = %d, expected 123", task.GitLabProjectID)
	}
}

func TestReviewTask_MergeRequest(t *testing.T) {
	mrNum := 99
	task := ReviewTask{
		EventType: "merge_request",
		MRNumber:  &mrNum,
		MRURL:     "https://gitlab.com/org/repo/-/merge_requests/99",
	}

	if task.EventType != "merge_request" {
		t.Errorf("EventType = %q, expected %q", task.EventType, "merge_request")
	}
	if task.MRNumber == nil || *task.MRNumber != 99 {
		t.Error("MRNumber should be 99")
	}
	if task.MRURL != "https://gitlab.com/org/repo/-/merge_requests/99" {
		t.Errorf("MRURL = %q, expected %q", task.MRURL, "https://gitlab.com/org/repo/-/merge_requests/99")
	}
}

func TestSyncQueue_New(t *testing.T) {
	queue := NewSyncQueue()
	if queue == nil {
		t.Error("NewSyncQueue should not return nil")
	}
}

func TestSyncQueue_IsAsync(t *testing.T) {
	queue := NewSyncQueue()
	if queue.IsAsync() {
		t.Error("SyncQueue.IsAsync() should return false")
	}
}

func TestSyncQueue_Close(t *testing.T) {
	queue := NewSyncQueue()
	err := queue.Close()
	if err != nil {
		t.Errorf("SyncQueue.Close() should return nil, got %v", err)
	}
}

func TestSyncQueue_EnqueueWithoutProcessor(t *testing.T) {
	queue := NewSyncQueue()
	task := &ReviewTask{
		ReviewLogID: 1,
		ProjectID:   1,
	}

	err := queue.Enqueue(task)
	if err != nil {
		t.Errorf("Enqueue without processor should not error, got %v", err)
	}
}

func TestSyncQueue_SetProcessor(t *testing.T) {
	queue := NewSyncQueue()

	queue.SetProcessor(func(ctx context.Context, task *ReviewTask) error {
		return nil
	})

	if queue.processor == nil {
		t.Error("processor should be set")
	}
}

func TestAsyncQueue_IsAsync(t *testing.T) {
	queue := &AsyncQueue{}
	if !queue.IsAsync() {
		t.Error("AsyncQueue.IsAsync() should return true")
	}
}
