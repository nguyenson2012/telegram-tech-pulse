package memory_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ai-tech-pulse/digest/internal/adapter/queue/memory"
	"github.com/ai-tech-pulse/digest/internal/domain/service"
)

func TestChannelQueue_EnqueueAndProcess(t *testing.T) {
	queue := memory.NewChannelJobQueue(3, 20)

	var processedCount int32
	testJobType := service.JobType("test:job")

	queue.RegisterHandler(testJobType, func(ctx context.Context, job *service.Job) error {
		atomic.AddInt32(&processedCount, 1)
		return nil
	})

	ctx := context.Background()
	if err := queue.Start(ctx); err != nil {
		t.Fatalf("failed to start queue: %v", err)
	}

	// Enqueue 5 jobs
	for i := 1; i <= 5; i++ {
		err := queue.Enqueue(ctx, &service.Job{
			ID:        "job-1",
			Type:      testJobType,
			Payload:   []byte(`{"test": true}`),
			CreatedAt: time.Now(),
		})
		if err != nil {
			t.Fatalf("failed to enqueue: %v", err)
		}
	}

	// Allow workers to process
	time.Sleep(100 * time.Millisecond)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := queue.Stop(shutdownCtx); err != nil {
		t.Fatalf("failed to stop queue: %v", err)
	}

	if val := atomic.LoadInt32(&processedCount); val != 5 {
		t.Errorf("expected 5 processed jobs, got %d", val)
	}
}
