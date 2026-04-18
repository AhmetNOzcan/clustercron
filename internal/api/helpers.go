package api

import (
	"clustercron/internal/storage"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, ErrorResponse{Error: msg})
}

func decodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func parseUUID(r *http.Request, param string) (uuid.UUID, error) {
	raw := chi.URLParam(r, param)
	return uuid.Parse(raw)
}

func toJobResponse(j *storage.Job) JobResponse {
	return JobResponse{
		ID:             j.ID,
		Name:           j.Name,
		Schedule:       j.Schedule,
		WebhookURL:     j.WebhookURL,
		HTTPMethod:     j.HTTPMethod,
		TimeoutSeconds: j.TimeoutSeconds,
		Enabled:        j.Enabled,
		NextFireAt:     j.NextFireAt,
		LastFireAt:     j.LastFireAt,
		CreatedAt:      j.CreatedAt,
		UpdatedAt:      j.UpdatedAt,
	}
}

func toJobRunResponse(r *storage.JobRun) JobRunResponse {
	return JobRunResponse{
		RunID:        r.RunID,
		JobID:        r.JobID,
		Status:       string(r.Status),
		Attempt:      r.Attempt,
		ScheduledAt:  r.ScheduledAt,
		StartedAt:    r.StartedAt,
		FinishedAt:   r.FinishedAt,
		WorkerID:     r.WorkerID,
		HTTPStatus:   r.HTTPStatus,
		ErrorMessage: r.ErrorMessage,
	}
}
