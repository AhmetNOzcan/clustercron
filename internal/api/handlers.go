package api

import (
	"clustercron/internal/heartbeat"
	"clustercron/internal/schedule"
	"clustercron/internal/storage"
	"errors"
	"log"
	"net/http"
	"time"
)

type Handler struct {
	db *storage.DB
	hb *heartbeat.Monitor
}

func NewHandler(db *storage.DB, hb *heartbeat.Monitor) *Handler {
	return &Handler{
		db: db,
		hb: hb,
	}
}

func (h *Handler) CreateJob(w http.ResponseWriter, r *http.Request) {
	var req CreateJobRequest

	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
	}

	// Validate required fields.
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Schedule == "" {
		writeError(w, http.StatusBadRequest, "schedule is required")
		return
	}
	if req.WebhookURL == "" {
		writeError(w, http.StatusBadRequest, "webhook_url is required")
		return
	}

	// Validate the cron expression.
	if err := schedule.Validate(req.Schedule); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Compute the first fire time.
	nextFire, err := schedule.NextFireTime(req.Schedule, time.Now())
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	job := &storage.Job{
		Name:           req.Name,
		Schedule:       req.Schedule,
		WebhookURL:     req.WebhookURL,
		HTTPMethod:     req.HTTPMethod,
		TimeoutSeconds: req.TimeoutSeconds,
		Enabled:        true,
		NextFireAt:     &nextFire,
	}

	if err := h.db.CreateJob(r.Context(), job); err != nil {
		log.Printf("ERROR crate job: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create job")
		return
	}
	writeJSON(w, http.StatusCreated, toJobResponse(job))
}

func (h *Handler) ListJobs(w http.ResponseWriter, r *http.Request) {
	jobs, err := h.db.ListJobs(r.Context())
	if err != nil {
		log.Printf("ERROR list jobs: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list jobs")
		return
	}

	resp := make([]JobResponse, len(jobs))
	for i, j := range jobs {
		resp[i] = toJobResponse(j)
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) GetJob(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid job id")
		return
	}
	job, err := h.db.GetJob(r.Context(), id)

	if errors.Is(err, storage.ErrNotFound) {
		writeError(w, http.StatusNotFound, "job not found")
		return
	}

	if err != nil {
		log.Printf("ERROR get job: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get job")
		return
	}

	writeJSON(w, http.StatusOK, toJobResponse(job))
}

// DeleteJob handles DELETE /api/jobs/{id}.
func (h *Handler) DeleteJob(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid job id")
		return
	}

	err = h.db.DeleteJob(r.Context(), id)
	if errors.Is(err, storage.ErrNotFound) {
		writeError(w, http.StatusNotFound, "job not found")
		return
	}
	if err != nil {
		log.Printf("ERROR delete job: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to delete job")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListJobRuns(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid job id")
		return
	}

	// Verify the job exists.
	_, err = h.db.GetJob(r.Context(), id)
	if errors.Is(err, storage.ErrNotFound) {
		writeError(w, http.StatusNotFound, "job not found")
		return
	}
	if err != nil {
		log.Printf("ERROR get job for runs: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get job")
		return
	}

	runs, err := h.db.ListJobRuns(r.Context(), id, 50)
	if err != nil {
		log.Printf("ERROR list runs: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list runs")
		return
	}

	resp := make([]JobRunResponse, len(runs))
	for i, run := range runs {
		resp[i] = toJobRunResponse(run)
	}
	writeJSON(w, http.StatusOK, resp)
}
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
func (h *Handler) ListWorkers(w http.ResponseWriter, r *http.Request) {
	workers, err := h.hb.LiveWorkers(r.Context())
	if err != nil {
		log.Printf("ERROR list workers: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list workers")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"workers": workers,
		"count":   len(workers),
	})
}
