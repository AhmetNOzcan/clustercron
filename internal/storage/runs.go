package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (db *DB) InsertJobRun(ctx context.Context, r *JobRun) (bool, error) {
	const q = `
		INSERT INTO job_runs
			(run_id, job_id, status, attempt, scheduled_at,
			 started_at, worker_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (run_id) DO NOTHING
	`
	tag, err := db.pool.Exec(ctx, q,
		r.RunID, r.JobID, r.Status, r.Attempt, r.ScheduledAt,
		r.StartedAt, r.WorkerID,
	)
	if err != nil {
		return false, fmt.Errorf("insert job run: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

func (db *DB) CompleteJobRun(
	ctx context.Context,
	runID string,
	status RunStatus,
	httpStatus *int,
	errMsg *string,
) error {
	const q = `
		UPDATE job_runs
		SET status        = $1,
		    finished_at   = now(),
		    http_status   = $2,
		    error_message = $3
		WHERE run_id = $4
	`
	_, err := db.pool.Exec(ctx, q, status, httpStatus, errMsg, runID)
	if err != nil {
		return fmt.Errorf("complete job run: %w", err)
	}
	return nil
}

func (db *DB) ListJobRuns(ctx context.Context, jobID uuid.UUID, limit int) ([]*JobRun, error) {
	const q = `
		SELECT run_id, job_id, status, attempt, scheduled_at,
		       started_at, finished_at, worker_id, http_status, error_message
		FROM job_runs
		WHERE job_id = $1
		ORDER BY scheduled_at DESC
		LIMIT $2
	`
	rows, err := db.pool.Query(ctx, q, jobID, limit)
	if err != nil {
		return nil, fmt.Errorf("list job runs: %w", err)
	}
	defer rows.Close()

	var runs []*JobRun
	for rows.Next() {
		var r JobRun
		if err := rows.Scan(
			&r.RunID, &r.JobID, &r.Status, &r.Attempt, &r.ScheduledAt,
			&r.StartedAt, &r.FinishedAt, &r.WorkerID, &r.HTTPStatus, &r.ErrorMessage,
		); err != nil {
			return nil, fmt.Errorf("scan run: %w", err)
		}
		runs = append(runs, &r)
	}
	return runs, rows.Err()
}

// GetJobRun fetches a single run by its deterministic run_id.
func (db *DB) GetJobRun(ctx context.Context, runID string) (*JobRun, error) {
	const q = `
		SELECT run_id, job_id, status, attempt, scheduled_at,
		       started_at, finished_at, worker_id, http_status, error_message
		FROM job_runs
		WHERE run_id = $1
	`
	var r JobRun
	err := db.pool.QueryRow(ctx, q, runID).Scan(
		&r.RunID, &r.JobID, &r.Status, &r.Attempt, &r.ScheduledAt,
		&r.StartedAt, &r.FinishedAt, &r.WorkerID, &r.HTTPStatus, &r.ErrorMessage,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get job run: %w", err)
	}
	return &r, nil
}
