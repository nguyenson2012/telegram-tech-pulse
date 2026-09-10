package service

import (
	"context"
	"time"
)

// JobType identifies the category of asynchronous background work.
type JobType string

const (
	JobTypeDailyPipeline  JobType = "pipeline:daily"
	JobTypeProcessFeed    JobType = "feed:process"
	JobTypeProcessArticle JobType = "article:process"
)

// Job represents a unit of background work passed to the queue.
type Job struct {
	ID        string    `json:"id"`
	Type      JobType   `json:"type"`
	Payload   []byte    `json:"payload"`
	CreatedAt time.Time `json:"created_at"`
}

// JobHandler handles the execution of a dequeued Job.
type JobHandler func(ctx context.Context, job *Job) error

// JobQueue defines the port for asynchronous background job scheduling and execution.
// Designed to be backed by Go channels (in-memory) for MVP or Redis (e.g. Asynq) in production.
type JobQueue interface {
	// Enqueue pushes a job to the queue for background processing.
	Enqueue(ctx context.Context, job *Job) error

	// RegisterHandler binds a handler function to a specific JobType.
	RegisterHandler(jobType JobType, handler JobHandler)

	// Start launches queue worker goroutines and begins processing.
	Start(ctx context.Context) error

	// Stop gracefully drains the queue and waits for in-flight tasks to complete.
	Stop(ctx context.Context) error
}
