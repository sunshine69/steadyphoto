package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// JobType defines the category of background work to be performed.
type JobType string

const (
	JobTypeFaceDetection JobType = "face_detection"
	JobTypeThumbnail     JobType = "thumbnail_generation"
)

// JobStatus defines the lifecycle of a background job.
type JobStatus string

const (
	JobStatusPending    JobStatus = "pending"
	JobStatusProcessing JobStatus = "processing"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
)

// Job represents a unit of work to be processed by the background worker.
type Job struct {
	ID        uuid.UUID `json:"id"`
	Type      JobType   `json:"type"`
	Status    JobStatus `json:"status"`
	PhotoID   uuid.UUID `json:"photo_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Error     string    `json:"error,omitempty"` // Stores error message if status is Failed
}

// JobRepository defines the interface for job persistence and queue management.
type JobRepository interface {
	Create(ctx context.Context, job *Job) error
	GetByID(ctx context.Context, id uuid.UUID) (*Job, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status JobStatus, errStr string) error
	GetPending(ctx context.Context, limit int) ([]*Job, error)
}
