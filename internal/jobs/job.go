package jobs

import (
	"time"

	"github.com/annismckenzie/x-article-exporter/internal/pipeline"
)

// Status represents the state of an export job.
type Status string

const (
	StatusProcessing Status = "processing"
	StatusComplete   Status = "complete"
	StatusFailed     Status = "failed"
)

// Job represents a single article export job.
type Job struct {
	ID        string
	URL       string
	Translate string
	DarkMode  bool
	Status    Status
	CreatedAt time.Time
	UpdatedAt time.Time
	Result    *pipeline.Result
	Error     string
}
