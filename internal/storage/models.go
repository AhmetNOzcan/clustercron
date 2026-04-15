package storage

import (
	"time"

	"github.com/google/uuid"
)

type Job struct {
	ID             uuid.UUID
	Name           string
	Schedule       string
	WebhookURL     string
	HTTPMethod     string
	TimeoutSeconds int
	Enabled        bool
	NextFireAt     *time.Time
	LastFireAt     *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type JobRun struct {
	RunID        string
	JobID        uuid.UUID
	Status       RunStatus
	Attempt      int
	ScheduledAt  time.Time
	StartedAt    *time.Time
	FinishedAt   *time.Time
	WorkerID     *string
	HTTPStatus   *int
	ErrorMessage *string
}

type RunStatus string

const (
	StatusRunning  RunStatus = "running"
	StatusSuccess  RunStatus = "success"
	StatusFailed   RunStatus = "failed"
	StatusTimedOut RunStatus = "timed_out"
)
