package database

import (
	"context"
	"fmt"

	"steadyphoto/internal/domain"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// PostgresAlbumShareRepository implements AlbumShareRepository
type PostgresAlbumShareRepository struct {
	db *sqlx.DB
}

func NewPostgresAlbumShareRepository(db *sqlx.DB) *PostgresAlbumShareRepository {
	return &PostgresAlbumShareRepository{db: db}
}

func (r *PostgresAlbumShareRepository) CreateAlbumShare(ctx context.Context, shareID uuid.UUID, albumID uuid.UUID) error {
	// Check for duplicate first to avoid UNIQUE constraint violations
	var existing domain.AlbumShare
	query := `SELECT id FROM album_shares WHERE share_id = $1 AND album_id = $2`
	err := r.db.GetContext(ctx, &existing, query, shareID, albumID)
	if err == nil {
		return nil // Already exists, skip duplicate
	}

	as := &domain.AlbumShare{
		ID:      uuid.New(),
		ShareID: shareID,
		AlbumID: albumID,
	}

	query = `INSERT INTO album_shares (id, share_id, album_id) VALUES (:id, :share_id, :album_id)`
	_, err = r.db.NamedExecContext(ctx, query, as)
	if err != nil {
		return fmt.Errorf("failed to create album share: %w", err)
	}

	return nil
}

// ListSharedAlbumsForUser returns all albums that have been shared with the given user.
func (r *PostgresAlbumShareRepository) ListSharedAlbumsForUser(ctx context.Context, shareeUserID uuid.UUID) ([]*domain.AlbumWithSharerInfo, error) {
	query := `
		SELECT a.*, s.sharer_user_id
		FROM album_shares albs
		JOIN shares s ON albs.share_id = s.id
		JOIN albums a ON albs.album_id = a.id
		WHERE s.sharee_user_id = $1
		ORDER BY a.created_at DESC
	`

	var results []struct {
		domain.Album `db:",inline"`
		SharerUserID uuid.UUID `db:"sharer_user_id" json:"-"`
	}

	err := r.db.SelectContext(ctx, &results, query, shareeUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to list shared albums: %w", err)
	}

	items := make([]*domain.AlbumWithSharerInfo, len(results))
	for i, r := range results {
		items[i] = &domain.AlbumWithSharerInfo{
			Album:        &r.Album,
			SharerUserID: r.SharerUserID,
		}
	}

	return items, nil
}

// GetOutgoingShareGroupAlbumIDs returns the album IDs in an outgoing share group (shares made BY a user).
func (r *PostgresAlbumShareRepository) GetOutgoingShareGroupAlbumIDs(ctx context.Context, shareID uuid.UUID) ([]uuid.UUID, error) {
	query := `SELECT album_id FROM album_shares WHERE share_id = $1`

	var albumIDs []uuid.UUID
	err := r.db.SelectContext(ctx, &albumIDs, query, shareID)
	if err != nil {
		return nil, fmt.Errorf("failed to get outgoing share group album IDs: %w", err)
	}

	return albumIDs, nil
}
