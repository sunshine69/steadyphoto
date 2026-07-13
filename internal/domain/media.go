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
	Duration      float64   `json:"duration"`
	Width         int       `json:"width"`
	Height        int       `json:"height"`
	Bitrate       int64     `json:"bitrate"`
	VideoCodec    string    `json:"video_codec"`
	AudioCodec    string    `json:"audio_codec"`
	FrameRate     float64   `json:"frame_rate"`
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

// ClientSource represents the source of an upload (e.g., "web", "android", "immich-migrate")
type ClientSource string

const (
	ClientSourceWeb        ClientSource = "web"
	ClientSourceAndroid    ClientSource = "android"
	ClientSourceiOS        ClientSource = "ios"
	ClientSourceImmichMigrate ClientSource = "immich-migrate"
	ClientSourceUnknown    ClientSource = "unknown"
)

// Media represents a single image/video asset
type Media struct {
	ID            uuid.UUID      `db:"id" json:"id"`
	Path          string         `db:"path" json:"path"`
	Filename      string         `db:"filename" json:"filename"`
	Hash          string         `db:"hash" json:"hash"`
	SizeBytes     int64          `db:"size_bytes" json:"sizeBytes"`
	Width         int            `db:"width" json:"width"`
	Height        int            `db:"height" json:"height"`
	CapturedAt    time.Time      `db:"captured_at" json:"capturedAt"`
	MediaType     MediaType      `db:"media_type" json:"mediaType"`
	Metadata      Metadata       `db:"metadata" json:"metadata"`
	VideoMetadata VideoMetadata  `db:"video_metadata" json:"videoMetadata"`
	CreatedAt     time.Time      `db:"created_at" json:"createdAt"`
	UpdatedAt     time.Time      `db:"updated_at" json:"updatedAt"`
	Tags          string             `db:"tags" json:"tags"`
	UserID        uuid.UUID          `db:"user_id" json:"userId"`
	ClientSource  ClientSource     `db:"client_source" json:"clientSource"`
	DeletedAt     *time.Time         `db:"deleted_at" json:"-"` // Soft delete timestamp, not exposed in JSON API
}

// Face represents a detected face in a media item
type Face struct {
	ID          uuid.UUID   `json:"id"`
	MediaID     uuid.UUID   `json:"mediaId"`
	BoundingBox BoundingBox `json:"boundingBox"`
	Embedding   []float32   `json:"embedding"` // For vector search
	CreatedAt   time.Time   `json:"createdAt"`
}

// BoundingBox defines the location of a face in an image
type BoundingBox struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"w"`
	Height float64 `json:"h"`
}

// Album represents a logical grouping of media assets
type Album struct {
	ID          uuid.UUID `db:"id" json:"id"`
	UserID      uuid.UUID `db:"user_id" json:"userId"`
	Name        string    `db:"name" json:"name"`
	Description *string   `db:"description" json:"description"` // Pointer allows handling NULL values from the DB
	CreatedAt   time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt   time.Time `db:"updated_at" json:"updatedAt"`
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

	// Search performs a full-text search across filename, tags, metadata, and path
	// with pagination support. scope: "all" | "name" | "tags"
	// startDate/endDate can be in formats: dd/mm/yyyy, yyyy/mm/dd, dd/mm/yyyy hh:mm:ss, etc.
	// If only one date is given, both startDate and endDate will be the same (single date search).
	// If startDate > endDate, they are swapped.
	// ExpressionResult contains include/exclude terms for advanced search (e.g., "holiday -beach")
	Search(ctx context.Context, query string, scope string, limit, offset int, userID *uuid.UUID, startDate string, endDate string, exprResult *ExpressionResult) ([]*Media, int, error)

	// Trash operations for soft-delete and permanent delete functionality
	ListTrashed(ctx context.Context, limit, offset int, userID uuid.UUID) ([]*Media, int, error)
	GetTrashedMedia(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*Media, error)
	RestoreMedia(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	PermanentlyDeleteMedia(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
}

// FaceRepository defines the interface for face storage
type FaceRepository interface {
	Create(ctx context.Context, face *Face) error
	GetByMediaID(ctx context.Context, mediaID uuid.UUID) ([]*Face, error)
	DeleteByMediaID(ctx context.Context, mediaID uuid.UUID) error
}

// AlbumRepository defines the interface for album management and membership
type AlbumRepository interface {
	Create(ctx context.Context, name string, userID uuid.UUID) (*Album, error)
	GetByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*Album, error)
	Update(ctx context.Context, album *Album) error
	Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	List(ctx context.Context, userID uuid.UUID) ([]*Album, error)

	// Media Association (Handles many-to-many relationships with position support)
	AddMedia(ctx context.Context, albumID uuid.UUID, mediaIDs []uuid.UUID) error
	RemoveMedia(ctx context.Context, albumID uuid.UUID, mediaID uuid.UUID) error
	BulkRemoveMedia(ctx context.Context, albumID uuid.UUID, mediaIDs []uuid.UUID) error
	GetMedia(ctx context.Context, albumID uuid.UUID, userID uuid.UUID) ([]*Media, error)
	GetMediaPaginated(ctx context.Context, albumID uuid.UUID, userID uuid.UUID, limit int, offset int) ([]*Media, int, error)
}
