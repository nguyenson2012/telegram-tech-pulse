package cron

import (
	"context"
	"fmt"
	"log"

	"github.com/ai-tech-pulse/digest/internal/usecase"
	"github.com/robfig/cron/v3"
)

// Scheduler manages scheduled execution of the daily AI Tech Pulse pipeline.
type Scheduler struct {
	cron            *cron.Cron
	pipelineUseCase *usecase.PipelineUseCase
	schedule        string
}

// NewScheduler creates a new cron scheduler instance.
func NewScheduler(schedule string, pipelineUseCase *usecase.PipelineUseCase) *Scheduler {
	if schedule == "" {
		schedule = "0 8 * * *" // Daily at 08:00 AM UTC
	}

	return &Scheduler{
		cron:            cron.New(cron.WithSeconds()), // or standard 5-spec
		pipelineUseCase: pipelineUseCase,
		schedule:        schedule,
	}
}

// Start registers the pipeline cron job and begins the background timer.
func (s *Scheduler) Start(ctx context.Context) error {
	c := cron.New()

	_, err := c.AddFunc(s.schedule, func() {
		log.Printf("[Cron] Triggering scheduled daily pipeline (%s)...", s.schedule)
		if err := s.pipelineUseCase.TriggerPipelineAsync(ctx); err != nil {
			log.Printf("[Cron] Failed to enqueue daily pipeline job: %v", err)
		}
	})
	if err != nil {
		return fmt.Errorf("invalid cron schedule '%s': %w", s.schedule, err)
	}

	s.cron = c
	s.cron.Start()
	log.Printf("[Cron] Daily pipeline scheduler started with schedule: '%s'", s.schedule)
	return nil
}

// Stop terminates scheduled jobs.
func (s *Scheduler) Stop() {
	if s.cron != nil {
		ctx := s.cron.Stop()
		<-ctx.Done()
		log.Println("[Cron] Scheduler stopped.")
	}
}
