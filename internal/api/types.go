package api

import (
	"time"

	"github.com/google/uuid"
)

type CreateJobRequest struct {
	Name           string `json:"name"`
	Schedule       string `json:"schedule"`
	WebhookURL     string `json:"webhook_url"`
	HTTPMethod     string `json:"http_method,omitempty"`
	TimeoutSeconds int    `json:"timeout_seconds,omitempty"`
}

type JobResponse struct {
	ID             uuid.UUID  `json:"id"`
	Name           string     `json:"name"`
	Schedule       string     `json:"schedule"`
	WebhookURL     string     `json:"webhook_url"`
	HTTPMethod     string     `json:"http_method"`
	TimeoutSeconds int        `json:"timeout_seconds"`
	Enabled        bool       `json:"enabled"`
	NextFireAt     *time.Time `json:"next_fire_at"`
	LastFireAt     *time.Time `json:"last_fire_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type JobRunResponse struct {
	RunID        string     `json:"run_id"`
	JobID        uuid.UUID  `json:"job_id"`
	Status       string     `json:"status"`
	Attempt      int        `json:"attempt"`
	ScheduledAt  time.Time  `json:"scheduled_at"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
	WorkerID     *string    `json:"worker_id,omitempty"`
	HTTPStatus   *int       `json:"http_status,omitempty"`
	ErrorMessage *string    `json:"error_message,omitempty"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
