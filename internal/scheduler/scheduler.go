package scheduler

import (
	"clustercron/internal/schedule"
	"clustercron/internal/storage"
	"clustercron/internal/worker"
	"context"
	"log"
	"time"
)

type Scheduler struct {
	db       *storage.DB
	executor *worker.Executor
	interval time.Duration
}

func New(db *storage.DB, executor *worker.Executor) *Scheduler {
	return &Scheduler{
		db:       db,
		executor: executor,
		interval: 5 * time.Second,
	}
}

func (s *Scheduler) Run(ctx context.Context) {
	log.Printf("[scheduler] starting, tick every %s", s.interval)
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	s.tick(ctx)

	for {
		select {
		case <-ticker.C:
			s.tick(ctx)
		case <-ctx.Done():
			log.Printf("[scheduler] stopped")
			return
		}
	}
}

func (s *Scheduler) tick(ctx context.Context) {
	jobs, err := s.db.GetDueJobs(ctx, time.Now(), 100)
	if err != nil {
		log.Printf("[scheduler] ERROR get due jobs: %v", err)
		return
	}
	if len(jobs) == 0 {
		return
	}

	for _, job := range jobs {
		// Check if context is done between jobs.
		if ctx.Err() != nil {
			return
		}
		s.processJob(ctx, job)
	}
}

func (s *Scheduler) processJob(ctx context.Context, job *storage.Job) {
	runID := worker.BuildRunID(job)

	// Delegate execution to the executor.
	s.executor.Execute(ctx, job, runID)

	// Advance the schedule regardless of execution result.
	// Even if the webhook failed, we don't want to re-fire the same
	// scheduled time — retries are a separate mechanism (Phase 4).
	nextFire, err := schedule.NextFireTime(job.Schedule, time.Now())
	if err != nil {
		log.Printf("[scheduler] ERROR compute next fire for %s: %v", job.Name, err)
		return
	}

	if err := s.db.UpdateNextFireTime(ctx, job.ID, nextFire); err != nil {
		log.Printf("[scheduler] ERROR update next_fire_at for %s: %v", job.Name, err)
	}
}
