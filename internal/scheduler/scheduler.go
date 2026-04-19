package scheduler

import (
	"clustercron/internal/schedule"
	"clustercron/internal/storage"
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
)

type Scheduler struct {
	db       *storage.DB
	client   *http.Client
	nodeID   string
	interval time.Duration
}

func New(db *storage.DB, nodeID string) *Scheduler {
	return &Scheduler{
		db:       db,
		nodeID:   nodeID,
		interval: 5 * time.Second,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
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
	runID := buildRunID(job)
	now := time.Now()

	run := &storage.JobRun{
		RunID:       runID,
		JobID:       job.ID,
		Status:      storage.StatusRunning,
		Attempt:     0,
		ScheduledAt: *job.NextFireAt,
		StartedAt:   &now,
		WorkerID:    &s.nodeID,
	}
	claimed, err := s.db.InsertJobRun(ctx, run)
	if err != nil {
		log.Printf("[scheduler] ERROR insert run %s: %v", runID, err)
		return
	}
	if !claimed {
		// Another node already claimed this run — skip.
		log.Printf("[scheduler] run %s already claimed, skipping", runID)
		return
	}

	log.Printf("[scheduler] firing job %s (%s) → %s %s",
		job.Name, job.ID, job.HTTPMethod, job.WebhookURL)

	status, errMsg := s.fireWebhook(ctx, job)

	var runStatus storage.RunStatus

	if errMsg != nil {
		runStatus = storage.StatusFailed
		log.Printf("[scheduler] job %s FAILED: %s", job.Name, *errMsg)
	} else {
		runStatus = storage.StatusSuccess
		log.Printf("[scheduler] job %s OK (HTTP %d)", job.Name, *status)
	}

	if err := s.db.CompleteJobRun(ctx, runID, runStatus, status, errMsg); err != nil {
		log.Printf("[scheduler] ERROR complete run %s: %v", runID, err)
	}
	// 4. Advance the schedule.
	nextFire, err := schedule.NextFireTime(job.Schedule, time.Now())
	if err != nil {
		log.Printf("[scheduler] ERROR compute next fire for %s: %v", job.Name, err)
		return
	}

	if err := s.db.UpdateNextFireTime(ctx, job.ID, nextFire); err != nil {
		log.Printf("[scheduler] ERROR update next_fire_at for %s: %v", job.Name, err)
	}
}

// fireWebhook makes the HTTP request and returns the status code and/or error.
func (s *Scheduler) fireWebhook(ctx context.Context, job *storage.Job) (httpStatus *int, errMsg *string) {
	// Build the request with the job's timeout.
	timeout := time.Duration(job.TimeoutSeconds) * time.Second
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, job.HTTPMethod, job.WebhookURL, nil)
	if err != nil {
		msg := fmt.Sprintf("build request: %v", err)
		return nil, &msg
	}

	// Add headers so the receiver knows who's calling.
	req.Header.Set("User-Agent", "clustercron/1.0")
	req.Header.Set("X-ClusterCron-Job-ID", job.ID.String())
	req.Header.Set("X-ClusterCron-Job-Name", job.Name)

	resp, err := s.client.Do(req)
	if err != nil {
		msg := fmt.Sprintf("request failed: %v", err)
		return nil, &msg
	}
	defer resp.Body.Close()

	code := resp.StatusCode

	// Treat 2xx as success, anything else as failure.
	if code >= 200 && code < 300 {
		return &code, nil
	}

	msg := fmt.Sprintf("webhook returned HTTP %d", code)
	return &code, &msg
}

func buildRunID(job *storage.Job) string {
	return fmt.Sprintf("%s:%d", job.ID.String(), job.NextFireAt.Unix())
}
