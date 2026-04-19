package worker

import (
	"clustercron/internal/storage"
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
)

type Executor struct {
	db     *storage.DB
	client *http.Client
	nodeID string
}

func NewExecutor(db *storage.DB, nodeID string) *Executor {
	return &Executor{
		db:     db,
		nodeID: nodeID,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (e *Executor) Execute(ctx context.Context, job *storage.Job, runID string) bool {
	now := time.Now()
	run := storage.JobRun{
		RunID:       runID,
		JobID:       job.ID,
		Status:      storage.StatusRunning,
		Attempt:     0,
		ScheduledAt: *job.NextFireAt,
		StartedAt:   &now,
		WorkerID:    &e.nodeID,
	}
	claimed, err := e.db.InsertJobRun(ctx, &run)
	if err != nil {
		log.Printf("[executor] ERROR insert run %s: %v", runID, err)
		return false
	}
	if !claimed {
		log.Printf("[executor] run %s already claimed, skipping", runID)
		return false
	}
	log.Printf("[executor] firing job %s (%s) → %s %s",
		job.Name, job.ID, job.HTTPMethod, job.WebhookURL)

	httpStatus, errMsg := e.fireWebhook(ctx, job)

	// 3. Record the result.
	var runStatus storage.RunStatus
	if errMsg != nil {
		runStatus = storage.StatusFailed
		log.Printf("[executor] job %s FAILED: %s", job.Name, *errMsg)
	} else {
		runStatus = storage.StatusSuccess
		log.Printf("[executor] job %s OK (HTTP %d)", job.Name, *httpStatus)
	}

	if err := e.db.CompleteJobRun(ctx, runID, runStatus, httpStatus, errMsg); err != nil {
		log.Printf("[executor] ERROR complete run %s: %v", runID, err)
	}

	return true
}

// fireWebhook makes the HTTP request and returns the status code and/or error.
func (e *Executor) fireWebhook(ctx context.Context, job *storage.Job) (httpStatus *int, errMsg *string) {
	timeout := time.Duration(job.TimeoutSeconds) * time.Second
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, job.HTTPMethod, job.WebhookURL, nil)
	if err != nil {
		msg := fmt.Sprintf("build request: %v", err)
		return nil, &msg
	}

	req.Header.Set("User-Agent", "clustercron/1.0")
	req.Header.Set("X-ClusterCron-Job-ID", job.ID.String())
	req.Header.Set("X-ClusterCron-Job-Name", job.Name)

	resp, err := e.client.Do(req)
	if err != nil {
		msg := fmt.Sprintf("request failed: %v", err)
		return nil, &msg
	}
	defer resp.Body.Close()

	code := resp.StatusCode
	if code >= 200 && code < 300 {
		return &code, nil
	}

	msg := fmt.Sprintf("webhook returned HTTP %d", code)
	return &code, &msg
}
