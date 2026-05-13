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

// VideoMetadata represents technical properties of a video file
type VideoMetadata struct {
	Duration      float64 `json:"duration"`
	Width         int     `json:"width"`
	Height        int     `json:"height"`
	Bitrate       int64   `json:"bitrate"`
	VideoCodec    string  `json:"video_codec"`
	AudioCodec    string  `json:"audio_codec"`
	FrameRate     float64 `json:"frame_rate"`
	CreatedAt     time.Time `json:"created_at"`
	ModifiedAt    time.Time `json:"modified_at"`
}

// Value implements the driver.Valuer interface for JSONB serialization
func (v VideoMetadata) Value() (driver.Value, error) {
	if v.Duration == 0 && v.Width == 0 {
		return nil, nil
	}
	return json.Marshal(v)
}

// Scan implements the sql.Scanner interface for JSONB deserialization
func (v *VideoMetadata) Scan(value interface{}) error {
	if value == nil {
		*v = VideoMetadata{}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("type assertion to []byte failed: %T", value)
	}

	return json.Unmarshal(bytes, v)
}

// MediaType represents the type of media
type MediaType string

const (
	MediaTypePhoto MediaType = "photo"
	MediaTypeVideo MediaType = "video"
)

// Scan implements the sql.Scanner interface to ensure proper deserialization
// from the database, handling potential NULL values or type mismatches.
func (m *MediaType) Scan(value interface{}) error {
	if value == nil {
		*m = ""
		return nil
	}
	switch v := value.(type) {
	case string:
		*m = MediaType(v)
	case []byte:
		*m = MediaType(string(v))
	default:
		return fmt.Errorf("type assertion failed: %T", value)
	}
	return nil
}

// Media represents a single image/video asset
type Media struct {
	ID            uuid.UUID      `db:"id"`
	Path          string         `db:"path"`
	Filename      string         `db:"filename"`
	Hash          string         `db:"hash"`
	SizeBytes     int64          `db:"size_bytes"`
	Width         int            `db:"width"`
	Height        int            `db:"height"`
	CapturedAt    time.Time      `db:"captured_at"`
	MediaType     MediaType      `db:"media_type"`
	Metadata      Metadata       `db:"metadata"`
	VideoMetadata VideoMetadata  `db:"video_metadata"`
	CreatedAt     time.Time      `db:"created_at"`
	UpdatedAt     time.Time      `db:"updated_at"`
	Tags          string         `db:"tags"`
	UserID        uuid.UUID      `db:"user_id"`
}

// Face represents a detected face in a media item
type Face struct {
	ID          uuid.UUID   `db:"id"`
	MediaID     uuid.UUID   `db:"media_id"`
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

// MediaRepository defines the interface for media storage
type MediaRepository interface {
	Create(ctx context.Context, media *Media) error
	GetByID(ctx context.Context, id uuid.UUID, userID *uuid.UUID) (*Media, error)
	GetByHash(ctx context.Context, hash string) (*Media, error)
	Update(ctx context.Context, media *Media) error
	Delete(ctx context.Context, id uuid.UUID, userID *uuid.UUID) error
	DeleteByMediaID(ctx context.Context, mediaID uuid.UUID) error
	List(ctx context.Context, limit, offset int, userID *uuid.UUID) ([]*Media, int, error)
	ListByType(ctx context.Context, mediaType MediaType, limit, offset int, userID *uuid.UUID) ([]*Media, int, error)
	SearchByTags(ctx context.Context, tags string, userID *uuid.UUID) ([]*Media, error)
}

// FaceRepository defines the interface for face storage
type FaceRepository interface {
	Create(ctx context.Context, face *Face) error
	GetByMediaID(ctx context.Context, mediaID uuid.UUID) ([]*Face, error)
	DeleteByMediaID(ctx context.Context, mediaID uuid.UUID) error
}

// AlbumRepository defines the interface for album storage
type AlbumRepository interface {
	Create(ctx context.Context, name string, userID uuid.UUID) (uuid.UUID, error)
	AddMedia(ctx context.Context, albumID uuid.UUID, mediaID uuid.UUID) error
	RemoveMedia(ctx context.Context, albumID uuid.UUID, mediaID uuid.UUID) error
	GetMedia(ctx context.Context, albumID uuid.UUID, userID *uuid.UUID) ([]*Media, error)
}
