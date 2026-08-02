package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Share represents a group of user-to-user shares together
type Share struct {
	ID           uuid.UUID `db:"id" json:"id"`
	SharerUserID uuid.UUID `db:"sharer_user_id" json:"-"`
	ShareeUserID uuid.UUID `db:"sharee_user_id" json:"-"`
	SharedAt     time.Time `db:"shared_at" json:"sharedAt"`
}

// MediaShare represents an individual media item shared with a user
type MediaShare struct {
	ID      uuid.UUID `db:"id" json:"-"`
	ShareID uuid.UUID `db:"share_id" json:"-"`
	MediaID uuid.UUID `db:"media_id" json:"-"`
}

// AlbumShare represents an album shared with a user
type AlbumShare struct {
	ID       uuid.UUID `db:"id" json:"-"`
	ShareID  uuid.UUID `db:"share_id" json:"-"`
	AlbumID  uuid.UUID `db:"album_id" json:"-"`
}

// PublicShare represents a public share link for media or albums
type PublicShare struct {
	ID             uuid.UUID `db:"id" json:"-"`
	Token          string    `db:"token" json:"token"`
	SharerUserID   uuid.UUID `db:"sharer_user_id" json:"-"`
	ResourceType   string    `db:"resource_type" json:"resourceType"`
	ResourceID     uuid.UUID `db:"resource_id" json:"-"`
	PasswordHash   *string   `db:"password_hash" json:"-"` // Pointer allows NULL values from DB
	ExpiresAt      *time.Time `db:"expires_at" json:"-"`    // Pointer allows NULL values from DB
	CreatedAt      time.Time `db:"created_at" json:"createdAt"`
	AccessCount    int       `db:"access_count" json:"accessCount"`
}

// PublicShareWithSharer is a PublicShare enriched with sharer's info for API responses
type PublicShareWithSharer struct {
	ID             uuid.UUID     `json:"-"`
	Token          string        `json:"token"`
	SharerUserID   uuid.UUID     `json:"sharerUserId"`
	SharerName     string        `json:"sharerName,omitempty"` // Populated by handler, not from DB directly
	ResourceType   string        `json:"resourceType"`
	ResourceID     uuid.UUID     `json:"-"`
	PasswordProtected bool       `json:"passwordProtected"`
	ExpiresAt      *time.Time    `json:"expiresAt"`
	CreatedAt      time.Time     `json:"createdAt"`
	AccessCount    int           `json:"accessCount"`
}

// PublicShareAccess represents an access log for a public share link
type PublicShareAccess struct {
	ID             uuid.UUID   `db:"id" json:"-"`
	PublicShareID  uuid.UUID   `db:"public_share_id" json:"-"`
	IPAddress      string      `db:"ip_address" json:"-"`
	AccessedAt     time.Time   `db:"accessed_at" json:"-"`
}

// OutgoingMediaInShare is a media item that belongs to an outgoing user-to-user share group.
type OutgoingMediaInShare struct {
	ID        uuid.UUID    `db:"media_id" json:"id"`
	Filename  string       `db:"filename" json:"filename"`
	Path      *string      `db:"path" json:"-"`
	MediaType MediaType     `db:"media_type" json:"mediaType"`
}

// OutgoingAlbumInShare is an album that belongs to an outgoing user-to-user share group.
type OutgoingAlbumInShare struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	Description *string    `json:"description"`
	CoverMediaID *uuid.UUID `db:"cover_media_id" json:"-"`
}

// ShareWithDetails represents an outgoing share group with its media and albums, plus the sharer/sharee names.
type ShareWithDetails struct {
	ID           uuid.UUID              `json:"id"`
	SharerUserID uuid.UUID              `json:"sharerUserId"`
	ShareeUserID uuid.UUID              `json:"shareeUserId"`
	SharedAt     time.Time              `json:"sharedAt"`
	Media        []OutgoingMediaInShare  `json:"media"`
	Albums       []OutgoingAlbumInShare  `json:"albums"`
}

// ShareWithSharerName enriches a Share with sharer and sharee display names.
type ShareWithSharerName struct {
	ID           uuid.UUID   `json:"id"`
	SharerUserID uuid.UUID   `json:"sharerUserId"`
	ShareeUserID uuid.UUID   `json:"shareeUserId"`
	SharedAt     time.Time   `json:"sharedAt"`
	SharerName   string      `json:"sharerName,omitempty"`
	ShareeName   string      `json:"shareeName,omitempty"`
}

// ShareRepository defines the interface for share storage and retrieval
type ShareRepository interface {
	CreateShare(ctx context.Context, sharerUserID uuid.UUID, shareeUserID uuid.UUID) (*Share, error)
	GetBySharerAndSharee(ctx context.Context, sharerUserID uuid.UUID, shareeUserID uuid.UUID) (*Share, error)

	// Outgoing shares — list all groups created BY a user (the sharer).
	ListOutgoingShares(ctx context.Context, sharerUserID uuid.UUID) ([]*ShareWithDetails, error)
	ListOutgoingShareGroups(ctx context.Context, sharerUserID uuid.UUID) ([]*ShareWithSharerName, error)

	// Revoke — delete a share group and its media_shares/album_shares.
	DeleteShareGroup(ctx context.Context, shareID uuid.UUID) error
}

// MediaShareRepository defines the interface for media share storage and retrieval
type MediaShareRepository interface {
	CreateMediaShare(ctx context.Context, shareID uuid.UUID, mediaID uuid.UUID) error
	GetByShareAndMedia(ctx context.Context, shareID uuid.UUID, mediaID uuid.UUID) (*MediaShare, error)
	BulkCreateMediaShares(ctx context.Context, shareID uuid.UUID, mediaIDs []uuid.UUID) error

	// Get shared media for a user (incoming shares)
	ListSharedMedia(ctx context.Context, shareeUserID uuid.UUID, limit int, offset int) ([]*MediaWithSharerInfo, int, error)
	GetSharedMediaByID(ctx context.Context, mediaID uuid.UUID, shareeUserID uuid.UUID) (*MediaWithSharerInfo, error)

	// Get shared albums for a user (incoming shares)
	ListSharedAlbums(ctx context.Context, shareeUserID uuid.UUID, limit int, offset int) ([]*AlbumWithSharerInfo, int, error)
	GetSharedAlbumByID(ctx context.Context, albumID uuid.UUID, shareeUserID uuid.UUID) (*AlbumWithSharerInfo, error)

	// Get media in a shared album for a user (incoming shares)
	ListMediaInSharedAlbum(ctx context.Context, albumID uuid.UUID, shareeUserID uuid.UUID, limit int, offset int) ([]*MediaWithSharerInfo, int, error)
	// Cursor-based pagination for presentation mode - older items (before timestamp)
	ListMediaInSharedAlbumBefore(ctx context.Context, albumID uuid.UUID, shareeUserID uuid.UUID, limit int, beforeTimestamp string) ([]*MediaWithSharerInfo, int, error)
	// Cursor-based pagination for presentation mode - newer items (after timestamp)
	ListMediaInSharedAlbumAfter(ctx context.Context, albumID uuid.UUID, shareeUserID uuid.UUID, limit int, afterTimestamp string) ([]*MediaWithSharerInfo, int, error)

	// Get media by ID that may be in a shared album (for individual media detail/thumb requests)
	GetSharedMediaFromAlbum(ctx context.Context, mediaID uuid.UUID, shareeUserID uuid.UUID) (*MediaWithSharerInfo, error)

	// Get media items in an outgoing share group (shares made BY the current user).
	GetOutgoingShareGroupMediaIDs(ctx context.Context, shareID uuid.UUID) ([]uuid.UUID, error)
}

// AlbumShareRepository defines the interface for album share storage and retrieval
type AlbumShareRepository interface {
	CreateAlbumShare(ctx context.Context, shareID uuid.UUID, albumID uuid.UUID) error

	// Get shared albums for a user (incoming shares)
	ListSharedAlbumsForUser(ctx context.Context, shareeUserID uuid.UUID) ([]*AlbumWithSharerInfo, error)

	// Get albums in an outgoing share group.
	GetOutgoingShareGroupAlbumIDs(ctx context.Context, shareID uuid.UUID) ([]uuid.UUID, error)
}

// PublicShareRepository defines the interface for public share storage and retrieval
type PublicShareRepository interface {
	CreatePublicShare(ctx context.Context, sharerUserID uuid.UUID, resourceType string, resourceID uuid.UUID, passwordHash *string, expiresAt *time.Time) (*PublicShare, error)
	GetByToken(ctx context.Context, token string) (*PublicShare, error)
	GetByID(ctx context.Context, id uuid.UUID) (*PublicShare, error)
	DeleteByToken(ctx context.Context, token string) error
	ListBySharer(ctx context.Context, sharerUserID uuid.UUID) ([]*PublicShareWithSharer, error)
	IncrementAccessCount(ctx context.Context, publicShareID uuid.UUID) error

	// Get shared media by token for public access (no auth required)
	GetSharedMediaByToken(ctx context.Context, token string) (*PublicShareWithSharerInfo, error)

	// Get shared album by token for public access (no auth required)
	GetSharedAlbumByToken(ctx context.Context, token string) (*PublicShareAlbumWithSharerInfo, error)

	// Get original file path by token for public serving (no auth required)
	GetOriginalFileByToken(ctx context.Context, token string) (*Media, error)

	// Get thumbnail file path by token for public serving (no auth required)
	GetThumbnailFileByToken(ctx context.Context, token string) (*Media, error)
}

// PublicShareAccessRepository defines the interface for public share access logging
type PublicShareAccessRepository interface {
	CreateAccessLog(ctx context.Context, publicShareID uuid.UUID, ipAddress string) error
}

// MediaWithSharerInfo is a Media enriched with sharer's info for API responses
type MediaWithSharerInfo struct {
	Media        *Media `json:"-"` // The base media metadata
	SharerUserID uuid.UUID `db:"sharer_user_id" json:"sharerUserId"`
}

// AlbumWithSharerInfo is an Album enriched with sharer's info for API responses
type AlbumWithSharerInfo struct {
	Album          *Album `json:"-"` // The base album metadata
	SharerUserID   uuid.UUID `db:"sharer_user_id" json:"sharerUserId"`
	FirstPhotoPath *string `db:"first_photo_path" json:"-"` // First photo path for thumbnail generation
	FirstMediaID   uuid.UUID `db:"first_media_id" json:"-"`  // First media item ID for thumbnail URL
}

// PublicShareWithSharerInfo is a Media enriched with public share info for API responses
type PublicShareWithSharerInfo struct {
	Media        *Media `json:"-"` // The base media metadata
	Token        string `json:"token"`
	SharerUserID uuid.UUID `db:"sharer_user_id" json:"sharerUserId"`
}

// PublicShareAlbumWithSharerInfo is an Album with media list enriched for public share API responses
type PublicShareAlbumWithSharerInfo struct {
	Album        *Album `json:"-"` // The base album metadata
	Token        string `json:"token"`
	MediaItems   []*Media `json:"media_items"`
}
