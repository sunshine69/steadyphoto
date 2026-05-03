package domain

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Metadata is a custom type for EXIF data to support JSONB scanning/valuing
type Metadata map[string]string

// Value implements the driver.Valuer interface for JSONB serialization
func (m Metadata) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	return json.Marshal(m)
}

// Scan implements the sql.Scanner interface for JSONB deserialization
func (m *Metadata) Scan(value interface{}) error {
	if value == nil {
		*m = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("type assertion to []byte failed: %T", value)
	}

	return json.Unmarshal(bytes, m)
}

// Photo represents a single image/video asset
type Photo struct {
	ID         uuid.UUID `db:"id"`
	Path       string    `db:"path"`
	Filename   string    `db:"filename"`
	Hash       string    `db:"hash"`
	SizeBytes  int64     `db:"size_bytes"`
	Width      int       `db:"width"`
	Height     int       `db:"height"`
	CapturedAt time.Time `db:"captured_at"`
	Metadata   Metadata  `db:"metadata"`
	CreatedAt  time.Time `db:"created_at"`
	UpdatedAt  time.Time `db:"updated_at"`
}

// Face represents a detected face in a photo
type Face struct {
	ID          uuid.UUID   `db:"id"`
	PhotoID     uuid.UUID   `db:"photo_id"`
	BoundingBox BoundingBox `db:"bounding_box"`
	Embedding   []float32   `db:"embedding"` // For vector search
	CreatedAt   time.Time   `db:"created_at"`
}

// BoundingBox defines the location of a face in an image
type BoundingBox struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"w"`
	Height float64 `json:"h"`
}

// PhotoRepository defines the interface for photo storage
type PhotoRepository interface {
	Create(ctx context.Context, photo *Photo) error
	GetByID(ctx context.Context, id uuid.UUID) (*Photo, error)
	GetByHash(ctx context.Context, hash string) (*Photo, error)
	Update(ctx context.Context, photo *Photo) error
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteByPhotoID(ctx context.Context, photoID uuid.UUID) error
	List(ctx context.Context, limit, offset int) ([]*Photo, int, error)
}

// FaceRepository defines the interface for face storage
type FaceRepository interface {
	Create(ctx context.Context, face *Face) error
	GetByPhotoID(ctx context.Context, photoID uuid.UUID) ([]*Face, error)
	DeleteByPhotoID(ctx context.Context, photoID uuid.UUID) error
}

// AlbumRepository defines the interface for album storage
type AlbumRepository interface {
	Create(ctx context.Context, name string) (uuid.UUID, error)
	AddPhoto(ctx context.Context, albumID uuid.UUID, photoID uuid.UUID) error
	RemovePhoto(ctx context.Context, albumID uuid.UUID, photoID uuid.UUID) error
	GetPhotos(ctx context.Context, albumID uuid.UUID) ([]*Photo, error)
}
