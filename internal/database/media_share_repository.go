package database

import (
	"context"
	"fmt"

	"steadyphoto/internal/domain"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// PostgresMediaShareRepository implements MediaShareRepository
type PostgresMediaShareRepository struct {
	db *sqlx.DB
}

func NewPostgresMediaShareRepository(db *sqlx.DB) *PostgresMediaShareRepository {
	return &PostgresMediaShareRepository{db: db}
}

func (r *PostgresMediaShareRepository) CreateMediaShare(ctx context.Context, shareID uuid.UUID, mediaID uuid.UUID) error {
	// Check for duplicate first to avoid UNIQUE constraint violations
	var existing domain.MediaShare
	query := `SELECT id FROM media_shares WHERE share_id = $1 AND media_id = $2`
	err := r.db.GetContext(ctx, &existing, query, shareID, mediaID)
	if err == nil {
		return nil // Already exists, skip duplicate
	}

	ms := &domain.MediaShare{
		ID:      uuid.New(),
		ShareID: shareID,
		MediaID: mediaID,
	}

	query = `INSERT INTO media_shares (id, share_id, media_id) VALUES (:id, :share_id, :media_id)`
	_, err = r.db.NamedExecContext(ctx, query, ms)
	if err != nil {
		return fmt.Errorf("failed to create media share: %w", err)
	}

	return nil
}

func (r *PostgresMediaShareRepository) BulkCreateMediaShares(ctx context.Context, shareID uuid.UUID, mediaIDs []uuid.UUID) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		}
	}()

	for _, mediaID := range mediaIDs {
		// Check for duplicate first to avoid UNIQUE constraint violations
		var existing domain.MediaShare
		query := `SELECT id FROM media_shares WHERE share_id = $1 AND media_id = $2`
		err = tx.GetContext(ctx, &existing, query, shareID, mediaID)
		if err == nil {
			continue // Already exists, skip duplicate
		}

		ms := &domain.MediaShare{
			ID:      uuid.New(),
			ShareID: shareID,
			MediaID: mediaID,
		}

		query = `INSERT INTO media_shares (id, share_id, media_id) VALUES (:id, :share_id, :media_id)`
		if _, err = tx.NamedExecContext(ctx, query, ms); err != nil {
			return fmt.Errorf("failed to create media share: %w", err)
		}
	}

	return tx.Commit()
}

func (r *PostgresMediaShareRepository) GetByShareAndMedia(ctx context.Context, shareID uuid.UUID, mediaID uuid.UUID) (*domain.MediaShare, error) {
	var ms domain.MediaShare
	query := `SELECT * FROM media_shares WHERE share_id = $1 AND media_id = $2`
	err := r.db.GetContext(ctx, &ms, query, shareID, mediaID)
	if err != nil {
		return nil, fmt.Errorf("media share not found: %w", err)
	}

	return &ms, nil
}

// ListSharedMedia returns all media items that have been shared with the given user.
func (r *PostgresMediaShareRepository) ListSharedMedia(ctx context.Context, shareeUserID uuid.UUID, limit int, offset int) ([]*domain.MediaWithSharerInfo, int, error) {
	var totalItems int
	countQuery := `
		SELECT COUNT(DISTINCT ms.media_id)
		FROM media_shares ms
		JOIN shares s ON ms.share_id = s.id
		WHERE s.sharee_user_id = $1 AND ms.media_id NOT IN (SELECT id FROM media WHERE deleted_at IS NOT NULL)
	`
	err := r.db.GetContext(ctx, &totalItems, countQuery, shareeUserID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count shared media: %w", err)
	}

	query := `
		SELECT m.*, s.sharer_user_id
		FROM media_shares ms
		JOIN shares s ON ms.share_id = s.id
		JOIN media m ON ms.media_id = m.id
		WHERE s.sharee_user_id = $1 AND m.deleted_at IS NULL
		ORDER BY m.captured_at DESC
		LIMIT $2 OFFSET $3
	`

	var results []struct {
		domain.Media `db:",inline"`
		SharerUserID uuid.UUID `db:"sharer_user_id" json:"-"`
	}

	err = r.db.SelectContext(ctx, &results, query, shareeUserID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list shared media: %w", err)
	}

	items := make([]*domain.MediaWithSharerInfo, len(results))
	for i, r := range results {
		items[i] = &domain.MediaWithSharerInfo{
			Media:        &r.Media,
			SharerUserID: r.SharerUserID,
		}
	}

	return items, totalItems, nil
}

// GetSharedMediaByID returns a single media item that has been shared with the given user.
func (r *PostgresMediaShareRepository) GetSharedMediaByID(ctx context.Context, mediaID uuid.UUID, shareeUserID uuid.UUID) (*domain.MediaWithSharerInfo, error) {
	query := `
		SELECT m.*, s.sharer_user_id
		FROM media_shares ms
		JOIN shares s ON ms.share_id = s.id
		JOIN media m ON ms.media_id = m.id
		WHERE ms.media_id = $1 AND s.sharee_user_id = $2 AND m.deleted_at IS NULL
	`

	var result struct {
		domain.Media `db:",inline"`
		SharerUserID uuid.UUID `db:"sharer_user_id" json:"-"`
	}

	err := r.db.GetContext(ctx, &result, query, mediaID, shareeUserID)
	if err != nil {
		return nil, fmt.Errorf("shared media not found: %w", err)
	}

	return &domain.MediaWithSharerInfo{
		Media:        &result.Media,
		SharerUserID: result.SharerUserID,
	}, nil
}

// ListSharedAlbums returns all albums that have been shared with the given user.
func (r *PostgresMediaShareRepository) ListSharedAlbums(ctx context.Context, shareeUserID uuid.UUID, limit int, offset int) ([]*domain.AlbumWithSharerInfo, int, error) {
	var totalItems int
	countQuery := `
		SELECT COUNT(DISTINCT albs.album_id)
		FROM album_shares albs
		JOIN shares s ON albs.share_id = s.id
		WHERE s.sharee_user_id = $1
	`
	err := r.db.GetContext(ctx, &totalItems, countQuery, shareeUserID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count shared albums: %w", err)
	}

	query := `
		SELECT a.*, s.sharer_user_id,
		       fp.id AS first_media_id,
		       fp.path AS first_photo_path
		FROM album_shares albs
		JOIN shares s ON albs.share_id = s.id
		JOIN albums a ON albs.album_id = a.id
		LEFT JOIN LATERAL (
		    SELECT m.id, m.path
		    FROM album_photos ap
		    JOIN media m ON m.id = ap.media_id AND m.deleted_at IS NULL
		    WHERE ap.album_id = a.id
		    ORDER BY ap.position ASC
		    LIMIT 1
		) fp ON TRUE
		WHERE s.sharee_user_id = $1
		ORDER BY a.created_at DESC
		LIMIT $2 OFFSET $3
	`

	var results []struct {
		domain.Album       `db:",inline"`
		SharerUserID       uuid.UUID `db:"sharer_user_id" json:"-"`
		FirstPhotoPath     *string   `db:"first_photo_path" json:"-"`
		FirstMediaID       uuid.UUID `db:"first_media_id" json:"-"`
	}

	err = r.db.SelectContext(ctx, &results, query, shareeUserID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list shared albums: %w", err)
	}

	items := make([]*domain.AlbumWithSharerInfo, len(results))
	for i, r := range results {
		items[i] = &domain.AlbumWithSharerInfo{
			Album:          &r.Album,
			SharerUserID:   r.SharerUserID,
			FirstPhotoPath: r.FirstPhotoPath,
			FirstMediaID:   r.FirstMediaID,
		}
	}

	return items, totalItems, nil
}

// GetSharedAlbumByID returns a single album that has been shared with the given user.
func (r *PostgresMediaShareRepository) GetSharedAlbumByID(ctx context.Context, albumID uuid.UUID, shareeUserID uuid.UUID) (*domain.AlbumWithSharerInfo, error) {
	query := `
		SELECT a.*, s.sharer_user_id,
		       fp.id AS first_media_id,
		       fp.path AS first_photo_path
		FROM album_shares albs
		JOIN shares s ON albs.share_id = s.id
		JOIN albums a ON albs.album_id = a.id
		LEFT JOIN LATERAL (
		    SELECT m.id, m.path
		    FROM album_photos ap
		    JOIN media m ON m.id = ap.media_id AND m.deleted_at IS NULL
		    WHERE ap.album_id = a.id
		    ORDER BY ap.position ASC
		    LIMIT 1
		) fp ON TRUE
		WHERE albs.album_id = $1 AND s.sharee_user_id = $2
	`

	var result struct {
		domain.Album       `db:",inline"`
		SharerUserID       uuid.UUID `db:"sharer_user_id" json:"-"`
		FirstPhotoPath     *string   `db:"first_photo_path" json:"-"`
		FirstMediaID       uuid.UUID `db:"first_media_id" json:"-"`
	}

	err := r.db.GetContext(ctx, &result, query, albumID, shareeUserID)
	if err != nil {
		return nil, fmt.Errorf("shared album not found: %w", err)
	}

	return &domain.AlbumWithSharerInfo{
		Album:          &result.Album,
		SharerUserID:   result.SharerUserID,
		FirstPhotoPath: result.FirstPhotoPath,
		FirstMediaID:   result.FirstMediaID,
	}, nil
}

// ListMediaInSharedAlbum returns all media items in a shared album for the given user.
func (r *PostgresMediaShareRepository) ListMediaInSharedAlbum(ctx context.Context, albumID uuid.UUID, shareeUserID uuid.UUID, limit int, offset int) ([]*domain.MediaWithSharerInfo, int, error) {
	var totalItems int
	countQuery := `
		SELECT COUNT(*) FROM album_photos ap
		JOIN albums a ON a.id = ap.album_id
		JOIN media m ON m.id = ap.media_id AND m.deleted_at IS NULL
		JOIN album_shares albs ON albs.album_id = a.id
		JOIN shares s ON albs.share_id = s.id
		WHERE ap.album_id = $1 AND s.sharee_user_id = $2
	`
	err := r.db.GetContext(ctx, &totalItems, countQuery, albumID, shareeUserID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count media in shared album: %w", err)
	}

	query := `
		SELECT m.*
		FROM album_photos ap
		JOIN albums a ON a.id = ap.album_id
		JOIN media m ON m.id = ap.media_id AND m.deleted_at IS NULL
		JOIN album_shares albs ON albs.album_id = a.id
		JOIN shares s ON albs.share_id = s.id
		WHERE ap.album_id = $1 AND s.sharee_user_id = $2
		ORDER BY ap.position ASC, m.captured_at DESC
		LIMIT $3 OFFSET $4
	`

	var results []struct {
		domain.Media `db:",inline"`
	}

	err = r.db.SelectContext(ctx, &results, query, albumID, shareeUserID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list media in shared album: %w", err)
	}

	items := make([]*domain.MediaWithSharerInfo, len(results))
	for i, r := range results {
		items[i] = &domain.MediaWithSharerInfo{
			Media:        &r.Media,
			SharerUserID: uuid.Nil, // Not applicable for album media - sharer is the album owner
		}
	}

	return items, totalItems, nil
}

// GetSharedMediaFromAlbum returns a single media item that is in a shared album for the given user.
// This is used as a fallback when GetSharedMediaByID fails because the media is shared via an album,
// not directly as a media share.
func (r *PostgresMediaShareRepository) GetSharedMediaFromAlbum(ctx context.Context, mediaID uuid.UUID, shareeUserID uuid.UUID) (*domain.MediaWithSharerInfo, error) {
	query := `
		SELECT m.*, s.sharer_user_id
		FROM album_photos ap
		JOIN albums a ON a.id = ap.album_id
		JOIN media m ON m.id = ap.media_id AND m.deleted_at IS NULL
		JOIN album_shares albs ON albs.album_id = a.id
		JOIN shares s ON albs.share_id = s.id
		WHERE ap.media_id = $1 AND s.sharee_user_id = $2
		LIMIT 1
	`

	var result struct {
		domain.Media `db:",inline"`
		SharerUserID uuid.UUID `db:"sharer_user_id" json:"-"`
	}

	err := r.db.GetContext(ctx, &result, query, mediaID, shareeUserID)
	if err != nil {
		return nil, fmt.Errorf("shared media from album not found: %w", err)
	}

	return &domain.MediaWithSharerInfo{
		Media:        &result.Media,
		SharerUserID: result.SharerUserID,
	}, nil
}

// GetOutgoingShareGroupMediaIDs returns the media IDs in an outgoing share group (shares made BY a user).
func (r *PostgresMediaShareRepository) GetOutgoingShareGroupMediaIDs(ctx context.Context, shareID uuid.UUID) ([]uuid.UUID, error) {
	query := `SELECT media_id FROM media_shares WHERE share_id = $1`

	var mediaIDs []uuid.UUID
	err := r.db.SelectContext(ctx, &mediaIDs, query, shareID)
	if err != nil {
		return nil, fmt.Errorf("failed to get outgoing share group media IDs: %w", err)
	}

	return mediaIDs, nil
}
