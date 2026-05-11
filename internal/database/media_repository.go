package database

import (
	"context"
	"time"

	"steadyphoto/internal/domain"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type PostgresMediaRepository struct {
	db *sqlx.DB
}

func NewPostgresMediaRepository(db *sqlx.DB) *PostgresMediaRepository {
	return &PostgresMediaRepository{db: db}
}

func (r *PostgresMediaRepository) Create(ctx context.Context, media *domain.Media) error {
	query := `
		INSERT INTO media (id, path, filename, hash, size_bytes, width, height, captured_at, media_type, metadata, video_metadata, created_at, updated_at, tags)
		VALUES (:id, :path, :filename, :hash, :size_bytes, :width, :height, :captured_at, :media_type, :metadata, :video_metadata, :created_at, :updated_at, :tags)
	`
	_, err := r.db.NamedExecContext(ctx, query, media)
	return err
}

func (r *PostgresMediaRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Media, error) {
	var media domain.Media
	query := `SELECT * FROM media WHERE id = $1`
	err := r.db.GetContext(ctx, &media, query, id)
	if err != nil {
		return nil, err
	}
	return &media, nil
}

func (r *PostgresMediaRepository) GetByHash(ctx context.Context, hash string) (*domain.Media, error) {
	var media domain.Media
	query := `SELECT * FROM media WHERE hash = $1`
	err := r.db.GetContext(ctx, &media, query, hash)
	if err != nil {
		return nil, err
	}
	return &media, nil
}

func (r *PostgresMediaRepository) Update(ctx context.Context, media *domain.Media) error {
	query := `
		UPDATE media 
		SET path = :path, filename = :filename, size_bytes = :size_bytes, width = :width, height = :height, media_type = :media_type, metadata = :metadata, video_metadata = :video_metadata, updated_at = :updated_at, tags = :tags
		WHERE id = :id
	`
	_, err := r.db.NamedExecContext(ctx, query, media)
	return err
}

func (r *PostgresMediaRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM media WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *PostgresMediaRepository) DeleteByMediaID(ctx context.Context, mediaID uuid.UUID) error {
	query := `DELETE FROM faces WHERE media_id = $1`
	_, err := r.db.ExecContext(ctx, query, mediaID)
	return err
}

func (r *PostgresMediaRepository) List(ctx context.Context, limit, offset int) ([]*domain.Media, int, error) {
	var mediaList []*domain.Media
	var total int

	// Get total count for pagination
	countQuery := `SELECT COUNT(*) FROM media`
	err := r.db.GetContext(ctx, &total, countQuery)
	if err != nil {
		return nil, 0, err
	}

	// Get paginated list
	listQuery := `SELECT * FROM media ORDER BY captured_at DESC LIMIT $1 OFFSET $2`
	err = r.db.SelectContext(ctx, &mediaList, listQuery, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	return mediaList, total, nil
}

func (r *PostgresMediaRepository) ListByType(ctx context.Context, mediaType domain.MediaType, limit, offset int) ([]*domain.Media, int, error) {
	var mediaList []*domain.Media
	var total int

	// Get total count for pagination
	countQuery := `SELECT COUNT(*) FROM media WHERE media_type = $1`
	err := r.db.GetContext(ctx, &total, countQuery, mediaType)
	if err != nil {
		return nil, 0, err
	}

	// Get paginated list
	listQuery := `SELECT * FROM media WHERE media_type = $1 ORDER BY captured_at DESC LIMIT $2 OFFSET $3`
	err = r.db.SelectContext(ctx, &mediaList, listQuery, mediaType, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	return mediaList, total, nil
}

func (r *PostgresMediaRepository) SearchByTags(ctx context.Context, tags string) ([]*domain.Media, error) {
	var mediaList []*domain.Media
	query := `SELECT * FROM media WHERE tags LIKE $1`
	searchPattern := "%" + tags + "%"
	err := r.db.SelectContext(ctx, &mediaList, query, searchPattern)
	if err != nil {
		return nil, err
	}
	return mediaList, nil
}

// Album Repository Implementation

type PostgresAlbumRepository struct {
	db *sqlx.DB
}

func NewPostgresAlbumRepository(db *sqlx.DB) *PostgresAlbumRepository {
	return &PostgresAlbumRepository{db: db}
}

func (r *PostgresAlbumRepository) Create(ctx context.Context, name string) (uuid.UUID, error) {
	id := uuid.New()
	query := `INSERT INTO albums (id, name, created_at) VALUES ($1, $2, $3)`
	_, err := r.db.ExecContext(ctx, query, id, name, time.Now())
	return id, err
}

func (r *PostgresAlbumRepository) AddMedia(ctx context.Context, albumID uuid.UUID, mediaID uuid.UUID) error {
	query := `INSERT INTO album_photos (album_id, media_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`
	_, err := r.db.ExecContext(ctx, query, albumID, mediaID)
	return err
}

func (r *PostgresAlbumRepository) RemoveMedia(ctx context.Context, albumID uuid.UUID, mediaID uuid.UUID) error {
	query := `DELETE FROM album_photos WHERE album_id = $1 AND media_id = $2`
	_, err := r.db.ExecContext(ctx, query, albumID, mediaID)
	return err
}

func (r *PostgresAlbumRepository) GetMedia(ctx context.Context, albumID uuid.UUID) ([]*domain.Media, error) {
	var mediaList []*domain.Media
	query := `
		SELECT m.* FROM media m
		JOIN album_photos ap ON m.id = ap.media_id
		WHERE ap.album_id = $1
	`
	err := r.db.SelectContext(ctx, &mediaList, query, albumID)
	if err != nil {
		return nil, err
	}
	return mediaList, nil
}
