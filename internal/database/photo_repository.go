package database

import (
	"context"
	"time"

	"steadyphoto/internal/domain"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type PostgresPhotoRepository struct {
	db *sqlx.DB
}

func NewPostgresPhotoRepository(db *sqlx.DB) *PostgresPhotoRepository {
	return &PostgresPhotoRepository{db: db}
}

func (r *PostgresPhotoRepository) Create(ctx context.Context, photo *domain.Photo) error {
	query := `
		INSERT INTO photos (id, path, filename, hash, size_bytes, width, height, captured_at, metadata, created_at, updated_at)
		VALUES (:id, :path, :filename, :hash, :size_bytes, :width, :height, :captured_at, :metadata, :created_at, :updated_at)
	`
	_, err := r.db.NamedExecContext(ctx, query, photo)
	return err
}

func (r *PostgresPhotoRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Photo, error) {
	var photo domain.Photo
	query := `SELECT * FROM photos WHERE id = $1`
	err := r.db.GetContext(ctx, &photo, query, id)
	if err != nil {
		return nil, err
	}
	return &photo, nil
}

func (r *PostgresPhotoRepository) GetByHash(ctx context.Context, hash string) (*domain.Photo, error) {
	var photo domain.Photo
	query := `SELECT * FROM photos WHERE hash = $1`
	err := r.db.GetContext(ctx, &photo, query, hash)
	if err != nil {
		return nil, err
	}
	return &photo, nil
}

func (r *PostgresPhotoRepository) Update(ctx context.Context, photo *domain.Photo) error {
	query := `
		UPDATE photos 
		SET path = :path, filename = :filename, size_bytes = :size_bytes, width = :width, height = :height, metadata = :metadata, updated_at = :updated_at
		WHERE id = :id
	`
	_, err := r.db.NamedExecContext(ctx, query, photo)
	return err
}

func (r *PostgresPhotoRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM photos WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *PostgresPhotoRepository) DeleteByPhotoID(ctx context.Context, photoID uuid.UUID) error {
	query := `DELETE FROM faces WHERE photo_id = $1`
	_, err := r.db.ExecContext(ctx, query, photoID)
	return err
}

func (r *PostgresPhotoRepository) List(ctx context.Context, limit, offset int) ([]*domain.Photo, int, error) {
	var photos []*domain.Photo
	var total int

	// Get total count for pagination
	countQuery := `SELECT COUNT(*) FROM photos`
	err := r.db.GetContext(ctx, &total, countQuery)
	if err != nil {
		return nil, 0, err
	}

	// Get paginated list
	listQuery := `SELECT * FROM photos ORDER BY captured_at DESC LIMIT $1 OFFSET $2`
	err = r.db.SelectContext(ctx, &photos, listQuery, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	return photos, total, nil
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

func (r *PostgresAlbumRepository) AddPhoto(ctx context.Context, albumID uuid.UUID, photoID uuid.UUID) error {
	query := `INSERT INTO album_photos (album_id, photo_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`
	_, err := r.db.ExecContext(ctx, query, albumID, photoID)
	return err
}

func (r *PostgresAlbumRepository) RemovePhoto(ctx context.Context, albumID uuid.UUID, photoID uuid.UUID) error {
	query := `DELETE FROM album_photos WHERE album_id = $1 AND photo_id = $2`
	_, err := r.db.ExecContext(ctx, query, albumID, photoID)
	return err
}

func (r *PostgresAlbumRepository) GetPhotos(ctx context.Context, albumID uuid.UUID) ([]*domain.Photo, error) {
	var photos []*domain.Photo
	query := `
		SELECT p.* FROM photos p
		JOIN album_photos ap ON p.id = ap.photo_id
		WHERE ap.album_id = $1
	`
	err := r.db.SelectContext(ctx, &photos, query, albumID)
	if err != nil {
		return nil, err
	}
	return photos, nil
}
