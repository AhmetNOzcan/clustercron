package broker

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type JobMessage struct {
	RunID          string    `json:"run_id"`
	JobID          uuid.UUID `json:"job_id"`
	JobName        string    `json:"job_name"`
	WebhookURL     string    `json:"webhook_url"`
	HTTPMethod     string    `json:"http_method"`
	TimeoutSeconds int       `json:"timeout_seconds"`
	ScheduledAt    time.Time `json:"scheduled_at"`
}

// Encode serializes the message to JSON bytes.
func (m *JobMessage) Encode() ([]byte, error) {
	data, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("encode job message: %w", err)
	}
	return data, nil
}

// DecodeJobMessage deserializes JSON bytes into a JobMessage.
func DecodeJobMessage(data []byte) (*JobMessage, error) {
	var m JobMessage
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("decode job message: %w", err)
	}
	return &m, nil
}
