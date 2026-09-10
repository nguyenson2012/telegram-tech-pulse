package memory

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ai-tech-pulse/digest/internal/domain/service"
)

var (
	ErrQueueClosed = errors.New("job queue is closed")
	ErrQueueFull   = errors.New("job queue buffer is full")
)

type channelQueue struct {
	workersCount int
	bufferSize   int
	jobsChan     chan *service.Job
	handlers     map[service.JobType]service.JobHandler
	mu           sync.RWMutex
	wg           sync.WaitGroup
	ctx          context.Context
	cancel       context.CancelFunc
	isRunning    bool
}

// NewChannelJobQueue creates an in-memory channel-based JobQueue with worker pool.
func NewChannelJobQueue(workersCount, bufferSize int) service.JobQueue {
	if workersCount <= 0 {
		workersCount = 2
	}
	if bufferSize <= 0 {
		bufferSize = 100
	}

	return &channelQueue{
		workersCount: workersCount,
		bufferSize:   bufferSize,
		jobsChan:     make(chan *service.Job, bufferSize),
		handlers:     make(map[service.JobType]service.JobHandler),
	}
}

// RegisterHandler binds a handler function to a job type.
func (q *channelQueue) RegisterHandler(jobType service.JobType, handler service.JobHandler) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.handlers[jobType] = handler
}

// Enqueue pushes a new job to the buffered channel.
func (q *channelQueue) Enqueue(ctx context.Context, job *service.Job) error {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if !q.isRunning {
		return ErrQueueClosed
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case q.jobsChan <- job:
		log.Printf("[Queue] Enqueued job %s (Type: %s)", job.ID, job.Type)
		return nil
	default:
		return ErrQueueFull
	}
}

// Start launches worker goroutines to process jobs from the channel.
func (q *channelQueue) Start(parentCtx context.Context) error {
	q.mu.Lock()
	if q.isRunning {
		q.mu.Unlock()
		return nil
	}

	q.ctx, q.cancel = context.WithCancel(parentCtx)
	q.isRunning = true
	q.mu.Unlock()

	log.Printf("[Queue] Starting %d background worker goroutines (Buffer: %d)...", q.workersCount, q.bufferSize)

	for i := 1; i <= q.workersCount; i++ {
		q.wg.Add(1)
		go q.worker(i)
	}

	return nil
}

// Stop gracefully signals workers to stop and waits for all active jobs to complete.
func (q *channelQueue) Stop(ctx context.Context) error {
	q.mu.Lock()
	if !q.isRunning {
		q.mu.Unlock()
		return nil
	}
	q.isRunning = false
	q.cancel()
	close(q.jobsChan)
	q.mu.Unlock()

	done := make(chan struct{})
	go func() {
		q.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("[Queue] All worker goroutines stopped gracefully.")
		return nil
	case <-ctx.Done():
		return fmt.Errorf("queue shutdown timed out: %w", ctx.Err())
	}
}

func (q *channelQueue) worker(workerID int) {
	defer q.wg.Done()
	log.Printf("[Queue:Worker-%d] Worker started", workerID)

	for job := range q.jobsChan {
		q.processJob(workerID, job)
	}

	log.Printf("[Queue:Worker-%d] Worker stopped", workerID)
}

func (q *channelQueue) processJob(workerID int, job *service.Job) {
	q.mu.RLock()
	handler, exists := q.handlers[job.Type]
	q.mu.RUnlock()

	if !exists {
		log.Printf("[Queue:Worker-%d] No handler registered for job type: %s", workerID, job.Type)
		return
	}

	start := time.Now()
	log.Printf("[Queue:Worker-%d] Processing job %s (Type: %s)", workerID, job.ID, job.Type)

	// Create job context with reasonable timeout (e.g. 5 minutes for full pipeline)
	jobCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := handler(jobCtx, job); err != nil {
		log.Printf("[Queue:Worker-%d] Job %s failed after %v: %v", workerID, job.ID, time.Since(start), err)
	} else {
		log.Printf("[Queue:Worker-%d] Job %s succeeded in %v", workerID, job.ID, time.Since(start))
	}
}
