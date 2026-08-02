package database

import (
	"context"
	"fmt"
	"time"

	"steadyphoto/internal/domain"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type PostgresAlbumRepository struct {
	db *sqlx.DB
}

func NewPostgresAlbumRepository(db *sqlx.DB) *PostgresAlbumRepository {
	return &PostgresAlbumRepository{db: db}
}

func (r *PostgresAlbumRepository) Create(ctx context.Context, name string, userID uuid.UUID) (*domain.Album, error) {
	album := &domain.Album{
		ID:        uuid.New(),
		UserID:    userID,
		Name:      name,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	query := `INSERT INTO albums (id, name, user_id, created_at, updated_at) VALUES (:id, :name, :user_id, :created_at, :updated_at)`
	_, err := r.db.NamedExecContext(ctx, query, album)
	if err != nil {
		return nil, err
	}

	return album, nil
}

func (r *PostgresAlbumRepository) GetByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*domain.Album, error) {
	var album domain.Album
	query := `SELECT * FROM albums WHERE id = $1 AND user_id = $2`
	err := r.db.GetContext(ctx, &album, query, id, userID)
	if err != nil {
		return nil, err
	}
	return &album, nil
}

func (r *PostgresAlbumRepository) Update(ctx context.Context, album *domain.Album) error {
	query := `UPDATE albums SET name = :name, description = :description, updated_at = :updated_at WHERE id = :id AND user_id = :user_id`
	result, err := r.db.NamedExecContext(ctx, query, album)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("album not found or ownership mismatch")
	}
	return nil
}

func (r *PostgresAlbumRepository) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	query := `DELETE FROM albums WHERE id = $1 AND user_id = $2`
	result, err := r.db.ExecContext(ctx, query, id, userID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("album not found or ownership mismatch")
	}
	return nil
}

func (r *PostgresAlbumRepository) List(ctx context.Context, userID uuid.UUID) ([]*domain.Album, error) {
	var albums []*domain.Album
	query := `SELECT * FROM albums WHERE user_id = $1 ORDER BY created_at DESC`
	err := r.db.SelectContext(ctx, &albums, query, userID)
	if err != nil {
		return nil, err
	}
	return albums, nil
}

func (r *PostgresAlbumRepository) AddMedia(ctx context.Context, albumID uuid.UUID, mediaIDs []uuid.UUID) error {
	// We use a transaction to ensure all or nothing is added
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for i, mediaID := range mediaIDs {
		// Position logic: simple incremental placement based on current batch order/count if needed
		// For now, we just insert with default position 0 or use the index in slice as a suggestion
		query := `INSERT INTO album_photos (album_id, media_id, position) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`
		_, err := tx.ExecContext(ctx, query, albumID, mediaID, i) // Using index as simple order
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *PostgresAlbumRepository) RemoveMedia(ctx context.Context, albumID uuid.UUID, mediaID uuid.UUID) error {
	query := `DELETE FROM album_photos WHERE album_id = $1 AND media_id = $2`
	_, err := r.db.ExecContext(ctx, query, albumID, mediaID)
	return err
}

func (r *PostgresAlbumRepository) BulkRemoveMedia(ctx context.Context, albumID uuid.UUID, mediaIDs []uuid.UUID) error {
	if len(mediaIDs) == 0 {
		return nil
	}
	query := `DELETE FROM album_photos WHERE album_id = $1 AND media_id = ANY($2)`
	_, err := r.db.ExecContext(ctx, query, albumID, mediaIDs)
	return err
}

func (r *PostgresAlbumRepository) GetMedia(ctx context.Context, albumID uuid.UUID, userID uuid.UUID) ([]*domain.Media, error) {
	var mediaList []*domain.Media
	query := `
		SELECT m.* FROM media m
		JOIN album_photos ap ON m.id = ap.media_id
		JOIN albums a ON a.id = ap.album_id
		WHERE ap.album_id = $1 AND a.user_id = $2
		ORDER BY ap.position ASC, m.captured_at ASC
	`
	err := r.db.SelectContext(ctx, &mediaList, query, albumID, userID)
	if err != nil {
		return nil, err
	}
	return mediaList, nil
}

func (r *PostgresAlbumRepository) GetMediaPaginated(ctx context.Context, albumID uuid.UUID, userID uuid.UUID, limit int, offset int) ([]*domain.Media, int, error) {
	var totalItems int
	countQuery := `SELECT COUNT(*) FROM album_photos ap JOIN albums a ON a.id = ap.album_id WHERE ap.album_id = $1 AND a.user_id = $2`
	err := r.db.GetContext(ctx, &totalItems, countQuery, albumID, userID)
	if err != nil {
		return nil, 0, err
	}

	var mediaList []*domain.Media
	query := `
		SELECT m.* FROM media m
		JOIN album_photos ap ON m.id = ap.media_id
		JOIN albums a ON a.id = ap.album_id
		WHERE ap.album_id = $1 AND a.user_id = $2
		ORDER BY ap.position ASC, m.captured_at ASC
		LIMIT $3 OFFSET $4
	`
	err = r.db.SelectContext(ctx, &mediaList, query, albumID, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	return mediaList, totalItems, nil
}

// GetMediaPaginatedBefore returns media items OLDER than the given timestamp (for presentation mode loading previous page)
// Items are returned oldest-first (ascending by captured_at)
func (r *PostgresAlbumRepository) GetMediaPaginatedBefore(ctx context.Context, albumID uuid.UUID, userID uuid.UUID, limit int, beforeTimestamp string) ([]*domain.Media, int, error) {
	// Parse the timestamp to ensure it's valid
	if beforeTimestamp == "" {
		return nil, 0, fmt.Errorf("before timestamp is required")
	}

	// First, get the total count for albums the user has access to
	var totalItems int
	countQuery := `SELECT COUNT(*) FROM album_photos ap JOIN albums a ON a.id = ap.album_id WHERE ap.album_id = $1 AND a.user_id = $2`
	err := r.db.GetContext(ctx, &totalItems, countQuery, albumID, userID)
	if err != nil {
		return nil, 0, err
	}

	var mediaList []*domain.Media
	// Query items OLDER than the given timestamp, ordered oldest-first
	// We use m.captured_at < $4 to find items older than the cursor
	query := `
		SELECT m.* FROM media m
		JOIN album_photos ap ON m.id = ap.media_id
		JOIN albums a ON a.id = ap.album_id
		WHERE ap.album_id = $1 AND a.user_id = $2 AND m.captured_at < to_timestamp($4::double precision / 1000.0)
		ORDER BY m.captured_at ASC, m.id ASC
		LIMIT $3
	`
	err = r.db.SelectContext(ctx, &mediaList, query, albumID, userID, limit, beforeTimestamp)
	if err != nil {
		return nil, 0, err
	}

	return mediaList, totalItems, nil
}

// GetMediaPaginatedAfter returns media items NEWER than the given timestamp (for presentation mode loading next page)
// Items are returned oldest-first (ascending by captured_at)
func (r *PostgresAlbumRepository) GetMediaPaginatedAfter(ctx context.Context, albumID uuid.UUID, userID uuid.UUID, limit int, afterTimestamp string) ([]*domain.Media, int, error) {
	// Parse the timestamp to ensure it's valid
	if afterTimestamp == "" {
		return nil, 0, fmt.Errorf("after timestamp is required")
	}

	// First, get the total count for albums the user has access to
	var totalItems int
	countQuery := `SELECT COUNT(*) FROM album_photos ap JOIN albums a ON a.id = ap.album_id WHERE ap.album_id = $1 AND a.user_id = $2`
	err := r.db.GetContext(ctx, &totalItems, countQuery, albumID, userID)
	if err != nil {
		return nil, 0, err
	}

	var mediaList []*domain.Media
	// Query items NEWER than the given timestamp, ordered oldest-first
	// We use m.captured_at > $4 to find items newer than the cursor
	query := `
		SELECT m.* FROM media m
		JOIN album_photos ap ON m.id = ap.media_id
		JOIN albums a ON a.id = ap.album_id
		WHERE ap.album_id = $1 AND a.user_id = $2 AND m.captured_at > to_timestamp($4::double precision / 1000.0)
		ORDER BY m.captured_at ASC, m.id ASC
		LIMIT $3
	`
	err = r.db.SelectContext(ctx, &mediaList, query, albumID, userID, limit, afterTimestamp)
	if err != nil {
		return nil, 0, err
	}

	return mediaList, totalItems, nil
}
