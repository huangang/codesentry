package services

import (
	"context"
	"encoding/json"
	"github.com/huangang/codesentry/backend/pkg/logger"
	"sync"

	"github.com/hibiken/asynq"
	"github.com/huangang/codesentry/backend/internal/config"
)

// Worker processes async tasks from the queue
type Worker struct {
	server    *asynq.Server
	mux       *asynq.ServeMux
	processor func(context.Context, *ReviewTask) error
	wg        sync.WaitGroup
	running   bool
	mu        sync.Mutex
}

// NewWorker creates a new worker instance
func NewWorker(cfg *config.RedisConfig) *Worker {
	if !cfg.Enabled {
		return nil
	}

	redisOpt := asynq.RedisClientOpt{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	}

	server := asynq.NewServer(
		redisOpt,
		asynq.Config{
			Concurrency: 10,
			Queues: map[string]int{
				"default": 1,
			},
			ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
				logger.Infof("[Worker] Error processing task %s: %v", task.Type(), err)
			}),
		},
	)

	return &Worker{
		server: server,
		mux:    asynq.NewServeMux(),
	}
}

// SetProcessor sets the function to process review tasks
func (w *Worker) SetProcessor(processor func(context.Context, *ReviewTask) error) {
	w.processor = processor
}

// Start begins processing tasks
func (w *Worker) Start() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.running {
		return nil
	}

	// Register handler for review tasks
	w.mux.HandleFunc(TaskTypeReview, w.handleReviewTask)

	w.running = true
	w.wg.Add(1)

	go func() {
		defer w.wg.Done()
		logger.Infof("[Worker] Starting async worker...")
		if err := w.server.Run(w.mux); err != nil {
			logger.Infof("[Worker] Server error: %v", err)
		}
	}()

	return nil
}

// Stop gracefully shuts down the worker
func (w *Worker) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.running {
		return
	}

	logger.Infof("[Worker] Shutting down...")
	w.server.Shutdown()
	w.running = false
	w.wg.Wait()
	logger.Infof("[Worker] Shutdown complete")
}

// handleReviewTask processes a single review task
func (w *Worker) handleReviewTask(ctx context.Context, t *asynq.Task) error {
	var task ReviewTask
	if err := json.Unmarshal(t.Payload(), &task); err != nil {
		logger.Infof("[Worker] Failed to unmarshal task: %v", err)
		return err
	}

	logger.Infof("[Worker] Processing review task: review_log_id=%d, project_id=%d, commit=%s",
		task.ReviewLogID, task.ProjectID, task.CommitSHA)

	if w.processor == nil {
		logger.Infof("[Worker] Warning: no processor set")
		return nil
	}

	return w.processor(ctx, &task)
}

// Global worker instance
var (
	globalWorker *Worker
	workerOnce   sync.Once
)

// InitWorker initializes the global worker
func InitWorker(cfg *config.RedisConfig) *Worker {
	workerOnce.Do(func() {
		globalWorker = NewWorker(cfg)
	})
	return globalWorker
}

// GetWorker returns the global worker instance
func GetWorker() *Worker {
	return globalWorker
}
