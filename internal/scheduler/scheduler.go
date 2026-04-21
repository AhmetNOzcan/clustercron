package scheduler

import (
	"clustercron/internal/broker"
	"clustercron/internal/hashring"
	"clustercron/internal/heartbeat"
	"clustercron/internal/schedule"
	"clustercron/internal/storage"
	"clustercron/internal/worker"
	"context"
	"log"
	"time"
)

type Scheduler struct {
	db       *storage.DB
	broker   *broker.Redis
	hb       *heartbeat.Monitor
	ring     *hashring.Ring
	interval time.Duration
}

func New(db *storage.DB, broker *broker.Redis, hb *heartbeat.Monitor) *Scheduler {
	return &Scheduler{
		db:       db,
		broker:   broker,
		hb:       hb,
		ring:     hashring.New(),
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

	s.refreshRing(ctx)

	if s.ring.Size() == 0 {
		log.Printf("[scheduler] no live workers, skipping tick")
		return
	}

	jobs, err := s.db.GetDueJobs(ctx, time.Now(), 100)
	if err != nil {
		log.Printf("[scheduler] ERROR get due jobs: %v", err)
		return
	}
	if len(jobs) == 0 {
		return
	}

	log.Printf("[scheduler] found %d due job(s), %d workers in ring",
		len(jobs), s.ring.Size())

	for _, job := range jobs {
		// Check if context is done between jobs.
		if ctx.Err() != nil {
			return
		}
		s.processJob(ctx, job)
	}
}

func (s *Scheduler) refreshRing(ctx context.Context) {
	workers, err := s.hb.LiveWorkers(ctx)
	if err != nil {
		log.Printf("[scheduler] ERROR get live workers: %v", err)
		return
	}

	currentMembers := s.ring.Members()

	// Quick check: if the member list hasn't changed, skip the rebuild.
	if slicesEqual(currentMembers, workers) {
		return
	}

	log.Printf("[scheduler] ring changed: %v → %v", currentMembers, workers)
	newRing := hashring.NewWithMembers(workers)
	s.ring = newRing
}

func (s *Scheduler) processJob(ctx context.Context, job *storage.Job) {
	runID := worker.BuildRunID(job)
	targetWorker := s.ring.GetWorker(job.ID.String())

	if targetWorker == "" {
		log.Printf("[scheduler] no worker available for job %s", job.Name)
		return
	}

	queue := broker.WorkerQueue(targetWorker)

	msg := &broker.JobMessage{
		RunID:          runID,
		JobID:          job.ID,
		JobName:        job.Name,
		WebhookURL:     job.WebhookURL,
		HTTPMethod:     job.HTTPMethod,
		TimeoutSeconds: job.TimeoutSeconds,
		ScheduledAt:    *job.NextFireAt,
	}

	data, err := msg.Encode()
	if err != nil {
		log.Printf("[scheduler] ERROR encode message for %s: %v", job.Name, err)
		return
	}

	if err := s.broker.Push(ctx, queue, data); err != nil {
		log.Printf("[scheduler] ERROR enqueue job %s: %v", job.Name, err)
		return
	}

	log.Printf("[scheduler] enqueued job %s (run %s)", job.Name, runID)

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

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
