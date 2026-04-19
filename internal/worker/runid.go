package worker

import (
	"clustercron/internal/storage"
	"fmt"
)

func BuildRunID(job *storage.Job) string {
	return fmt.Sprintf("%s:%d", job.ID.String(), job.NextFireAt.Unix())
}
