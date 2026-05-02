package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Photo represents the core entity of a photograph in the system.
type Photo struct {
	ID          uuid.UUID         `json:"id"`
	Hash        string            `json:"hash"`         // SHA256 hash for deduplication
	Path        string            `json:"path"`         // Physical path on filesystem
	Filename    string            `json:"filename"`     // Original filename
	Size        int64             `json:"size"`         // Size in bytes
	Width       int               `json:"width"`        // Image width
	Height      int               `json:"height"`       // Image height
	CapturedAt  time.Time         `json:"captured_at"`  // EXIF capture time
	CreatedAt   time.Time         `json:"created_at"`   // Database insertion time
	Metadata    map[string]string `json:"metadata"`     // Flexible EXIF/other metadata
}

// Face represents a detected face within a photo.
type Face struct {
	ID          uuid.UUID `json:"id"`
	PhotoID     uuid.UUID `json:"photo_id"`
	BoundingBox BBox      `json:"bounding_box"`
	Embedding   []float32 `json:"embedding"` // For vector similarity search
}

// BBox defines the coordinates of a bounding box.
type BBox struct {
	Top    float64 `json:"top"`
	Left   float64 `json:"left"`
	Bottom float64 `json:"bottom"`
	Right  float64 `json:"right"`
}

// PhotoRepository defines the interface for photo data persistence.
type PhotoRepository interface {
	Create(ctx context.Context, photo *Photo) error
	GetByID(ctx context.Context, id uuid.UUID) (*Photo, error)
	GetByHash(ctx context.Context, hash string) (*Photo, error)
	List(ctx context.Context, limit, offset int) ([]*Photo, int, error)
	Update(ctx context.Context, photo *Photo) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// FaceRepository defines the interface for face data persistence.
type FaceRepository interface {
	Create(ctx context.Context, face *Face) error
	GetByPhotoID(ctx context.Context, photoID uuid.UUID) ([]*Face, error)
	DeleteByPhotoID(ctx context.Context, photoID uuid.UUID) error
}
