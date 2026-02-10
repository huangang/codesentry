package services

import (
	"context"
	"encoding/json"
	"github.com/huangang/codesentry/backend/pkg/logger"
	"sync"

	"github.com/hibiken/asynq"
	"github.com/huangang/codesentry/backend/internal/config"
)

const (
	TaskTypeReview = "review:process"
)

// ReviewTask represents a review job to be processed
type ReviewTask struct {
	ReviewLogID   uint   `json:"review_log_id"`
	ProjectID     uint   `json:"project_id"`
	CommitSHA     string `json:"commit_sha"`
	EventType     string `json:"event_type"` // push, merge_request
	Branch        string `json:"branch"`
	Author        string `json:"author"`
	AuthorEmail   string `json:"author_email"`
	AuthorAvatar  string `json:"author_avatar"`
	CommitMessage string `json:"commit_message"`
	Diff          string `json:"diff"`
	CommitURL     string `json:"commit_url"`
	MRNumber      *int   `json:"mr_number,omitempty"`
	MRURL         string `json:"mr_url,omitempty"`
	// GitLab specific
	GitLabProjectID int `json:"gitlab_project_id,omitempty"`
}

// TaskQueue defines the interface for review task processing
type TaskQueue interface {
	// Enqueue adds a task to the queue
	Enqueue(task *ReviewTask) error
	// IsAsync returns true if queue processes tasks asynchronously
	IsAsync() bool
	// Close gracefully shuts down the queue
	Close() error
}

// Global task queue instance
var (
	globalTaskQueue TaskQueue
	taskQueueOnce   sync.Once
)

// InitTaskQueue initializes the global task queue based on config
func InitTaskQueue(cfg *config.Config) TaskQueue {
	taskQueueOnce.Do(func() {
		if cfg.Redis.Enabled {
			queue, err := NewAsyncQueue(&cfg.Redis)
			if err != nil {
				logger.Infof("[TaskQueue] Redis unavailable, falling back to sync mode: %v", err)
				globalTaskQueue = NewSyncQueue()
			} else {
				logger.Infof("[TaskQueue] Async queue initialized with Redis at %s", cfg.Redis.Addr)
				globalTaskQueue = queue
			}
		} else {
			logger.Infof("[TaskQueue] Sync queue initialized (Redis disabled)")
			globalTaskQueue = NewSyncQueue()
		}
	})
	return globalTaskQueue
}

// GetTaskQueue returns the global task queue instance
func GetTaskQueue() TaskQueue {
	return globalTaskQueue
}

// AsyncQueue implements TaskQueue using asynq (Redis-based)
type AsyncQueue struct {
	client *asynq.Client
}

// NewAsyncQueue creates a new Redis-based async queue
func NewAsyncQueue(cfg *config.RedisConfig) (*AsyncQueue, error) {
	redisOpt := asynq.RedisClientOpt{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	}

	client := asynq.NewClient(redisOpt)

	// Test connection by pinging Redis
	inspector := asynq.NewInspector(redisOpt)
	defer inspector.Close()

	// Try to get queue info to verify connection
	_, err := inspector.Queues()
	if err != nil {
		client.Close()
		return nil, err
	}

	return &AsyncQueue{client: client}, nil
}

// Enqueue adds a review task to the async queue
func (q *AsyncQueue) Enqueue(task *ReviewTask) error {
	payload, err := json.Marshal(task)
	if err != nil {
		return err
	}

	t := asynq.NewTask(TaskTypeReview, payload)
	info, err := q.client.Enqueue(t,
		asynq.Queue("default"),
		asynq.MaxRetry(3),
	)
	if err != nil {
		return err
	}

	logger.Infof("[AsyncQueue] Task enqueued: id=%s, queue=%s", info.ID, info.Queue)
	return nil
}

// IsAsync returns true for async queue
func (q *AsyncQueue) IsAsync() bool {
	return true
}

// Close closes the async queue client
func (q *AsyncQueue) Close() error {
	return q.client.Close()
}

// SyncQueue implements TaskQueue with synchronous processing (no Redis)
type SyncQueue struct {
	processor func(context.Context, *ReviewTask) error
}

// NewSyncQueue creates a new synchronous queue
func NewSyncQueue() *SyncQueue {
	return &SyncQueue{}
}

// SetProcessor sets the function to process tasks synchronously
func (q *SyncQueue) SetProcessor(processor func(context.Context, *ReviewTask) error) {
	q.processor = processor
}

// Enqueue processes the task immediately in the current goroutine
func (q *SyncQueue) Enqueue(task *ReviewTask) error {
	if q.processor == nil {
		logger.Infof("[SyncQueue] Warning: no processor set, task will be dropped")
		return nil
	}

	// Process in a goroutine to not block the webhook response
	go func() {
		ctx := context.Background()
		if err := q.processor(ctx, task); err != nil {
			logger.Infof("[SyncQueue] Task processing failed: %v", err)
		}
	}()

	return nil
}

// IsAsync returns false for sync queue
func (q *SyncQueue) IsAsync() bool {
	return false
}

// Close is a no-op for sync queue
func (q *SyncQueue) Close() error {
	return nil
}
