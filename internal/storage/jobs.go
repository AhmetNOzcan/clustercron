package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var ErrNotFound = errors.New("not found")

func (db *DB) CreateJob(ctx context.Context, j *Job) error {
	if j.ID == uuid.Nil {
		j.ID = uuid.New()
	}
	if j.HTTPMethod == "" {
		j.HTTPMethod = "POST"
	}
	if j.TimeoutSeconds == 0 {
		j.TimeoutSeconds = 30
	}
	const q = `INSERT INTO jobs
		(id, name, schedule, webhook_url, http_method, timeout_seconds,
		enabled, next_fire_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at, updated_at
	`
	err := db.pool.QueryRow(ctx, q,
		j.ID, j.Name, j.Schedule, j.WebhookURL, j.HTTPMethod,
		j.TimeoutSeconds, j.Enabled, j.NextFireAt).Scan(&j.CreatedAt, &j.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create job: %w", err)
	}
	return nil
}

func (db *DB) GetJob(ctx context.Context, id uuid.UUID) (*Job, error) {
	const q = `
		SELECT id, name, schedule, webhook_url, http_method,
		       timeout_seconds, enabled, next_fire_at, last_fire_at,
		       created_at, updated_at
		FROM jobs
		WHERE id = $1
	`

	var j Job
	err := db.pool.QueryRow(ctx, q, id).Scan(
		&j.ID, &j.Name, &j.Schedule, &j.WebhookURL, &j.HTTPMethod,
		&j.TimeoutSeconds, &j.Enabled, &j.NextFireAt, &j.LastFireAt,
		&j.CreatedAt, &j.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get job: %w", err)
	}
	return &j, nil
}

func (db *DB) ListJobs(ctx context.Context) ([]*Job, error) {
	const q = `
		SELECT id, name, schedule, webhook_url, http_method,
		       timeout_seconds, enabled, next_fire_at, last_fire_at,
		       created_at, updated_at
		FROM jobs
		ORDER BY created_at DESC
	`
	rows, err := db.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list jobs: %w", err)
	}

	defer rows.Close()
	var jobs []*Job

	for rows.Next() {
		var j Job
		if err := rows.Scan(
			&j.ID, &j.Name, &j.Schedule, &j.WebhookURL, &j.HTTPMethod,
			&j.TimeoutSeconds, &j.Enabled, &j.NextFireAt, &j.LastFireAt,
			&j.CreatedAt, &j.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan job: %w", err)
		}
		jobs = append(jobs, &j)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate jobs: %w", err)
	}
	return jobs, nil
}

func (db *DB) DeleteJob(ctx context.Context, id uuid.UUID) error {
	const q = `DELETE FROM jobs WHERE id = $1`
	tag, err := db.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("delete job: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (db *DB) GetDueJobs(ctx context.Context, now time.Time, limit int) ([]*Job, error) {
	const q = `
		SELECT id, name, schedule, webhook_url, http_method,
		       timeout_seconds, enabled, next_fire_at, last_fire_at,
		       created_at, updated_at
		FROM jobs
		WHERE enabled = true
		  AND next_fire_at IS NOT NULL
		  AND next_fire_at <= $1
		ORDER BY next_fire_at ASC
		LIMIT $2
	`
	rows, err := db.pool.Query(ctx, q, now, limit)
	if err != nil {
		return nil, fmt.Errorf("get due jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*Job
	for rows.Next() {
		var j Job
		if err := rows.Scan(
			&j.ID, &j.Name, &j.Schedule, &j.WebhookURL, &j.HTTPMethod,
			&j.TimeoutSeconds, &j.Enabled, &j.NextFireAt, &j.LastFireAt,
			&j.CreatedAt, &j.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan due job: %w", err)
		}
		jobs = append(jobs, &j)
	}
	return jobs, rows.Err()
}

func (db *DB) UpdateNextFireTime(ctx context.Context, id uuid.UUID, next time.Time) error {
	const q = `
		UPDATE jobs
		SET next_fire_at = $1,
		    last_fire_at = now(),
		    updated_at   = now()
		WHERE id = $2
	`
	_, err := db.pool.Exec(ctx, q, next, id)
	if err != nil {
		return fmt.Errorf("update next_fire_at: %w", err)
	}
	return nil
}
