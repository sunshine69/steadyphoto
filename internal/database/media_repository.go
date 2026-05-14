package database

import (
	"context"
	"fmt"
	"os"
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
		INSERT INTO media (id, user_id, path, filename, hash, size_bytes, width, height, captured_at, media_type, metadata, video_metadata, created_at, updated_at, tags)
		VALUES (:id, :user_id, :path, :filename, :hash, :size_bytes, :width, :height, :captured_at, :media_type, :metadata, :video_metadata, :created_at, :updated_at, :tags)
	`
	_, err := r.db.NamedExecContext(ctx, query, media)
	return err
}

func (r *PostgresMediaRepository) GetByID(ctx context.Context, id uuid.UUID, userID *uuid.UUID) (*domain.Media, error) {
	var media domain.Media
	query := `SELECT * FROM media WHERE id = $1`
	args := []interface{}{id}

	if userID != nil {
		query += ` AND user_id = $2`
		args = append(args, *userID)
	}

	err := r.db.GetContext(ctx, &media, query, args...)
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(os.Stderr, "[DEBUG] GetById %v\n", media)
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
		WHERE id = :id AND user_id = :user_id
	`
	result, err := r.db.NamedExecContext(ctx, query, media)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no record updated (id not found or ownership mismatch)")
	}

	return nil
}

func (r *PostgresMediaRepository) Delete(ctx context.Context, id uuid.UUID, userID *uuid.UUID) error {
	query := `DELETE FROM media WHERE id = $1`
	args := []interface{}{id}

	if userID != nil {
		query += ` AND user_id = $2`
		args = append(args, *userID)
	}

	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *PostgresMediaRepository) DeleteByMediaID(ctx context.Context, mediaID uuid.UUID) error {
	query := `DELETE FROM faces WHERE media_id = $1`
	_, err := r.db.ExecContext(ctx, query, mediaID)
	return err
}

func (r *PostgresMediaRepository) List(ctx context.Context, limit, offset int, userID *uuid.UUID) ([]*domain.Media, int, error) {
	var mediaList []*domain.Media
	var total int

	// Get total count for pagination
	countQuery := `SELECT COUNT(*) FROM media`
	argsCount := []interface{}{}

	if userID != nil {
		countQuery += ` WHERE user_id = $1`
		argsCount = append(argsCount, *userID)
	}

	err := r.db.GetContext(ctx, &total, countQuery, argsCount...)
	if err != nil {
		return nil, 0, err
	}

	// Get paginated list
	listQuery := `SELECT * FROM media`
	argsList := []interface{}{}

	if userID != nil {
		listQuery += ` WHERE user_id = $1`
		argsList = append(argsList, *userID)
	}

	listQuery += fmt.Sprintf(` ORDER BY captured_at DESC LIMIT $%d OFFSET $%d`, len(argsList)+1, len(argsList)+2)
	argsList = append(argsList, limit, offset)

	err = r.db.SelectContext(ctx, &mediaList, listQuery, argsList...)
	if err != nil {
		return nil, 0, err
	}

	return mediaList, total, nil
}

func (r *PostgresMediaRepository) ListByType(ctx context.Context, mediaType domain.MediaType, limit, offset int, userID *uuid.UUID) ([]*domain.Media, int, error) {
	var mediaList []*domain.Media
	var total int

	// Get total count for pagination
	countQuery := `SELECT COUNT(*) FROM media WHERE media_type = $1`
	argsCount := []interface{}{mediaType}

	if userID != nil {
		countQuery += ` AND user_id = $2`
		argsCount = append(argsCount, *userID)
	}

	err := r.db.GetContext(ctx, &total, countQuery, argsCount...)
	if err != nil {
		return nil, 0, err
	}

	// Get paginated list
	listQuery := `SELECT * FROM media WHERE media_type = $1`
	argsList := []interface{}{mediaType}

	if userID != nil {
		listQuery += ` AND user_id = $2`
		argsList = append(argsList, *userID)
	}

	listQuery += fmt.Sprintf(` ORDER BY captured_at DESC LIMIT $%d OFFSET $%d`, len(argsList)+1, len(argsList)+2)
	argsList = append(argsList, limit, offset)

	err = r.db.SelectContext(ctx, &mediaList, listQuery, argsList...)
	if err != nil {
		return nil, 0, err
	}

	return mediaList, total, nil
}

func (r *PostgresMediaRepository) SearchByTags(ctx context.Context, tags string, userID *uuid.UUID) ([]*domain.Media, error) {
	var mediaList []*domain.Media
	query := `SELECT * FROM media WHERE tags LIKE $1`
	args := []interface{}{"%" + tags + "%"}

	if userID != nil {
		query += ` AND user_id = $2`
		args = append(args, *userID)
	}

	err := r.db.SelectContext(ctx, &mediaList, query, args...)
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

func (r *PostgresAlbumRepository) Create(ctx context.Context, name string, userID uuid.UUID) (uuid.UUID, error) {
	id := uuid.New()
	query := `INSERT INTO albums (id, name, user_id, created_at) VALUES ($1, $2, $3, $4)`
	_, err := r.db.ExecContext(ctx, query, id, name, userID, time.Now())
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

func (r *PostgresAlbumRepository) GetMedia(ctx context.Context, albumID uuid.UUID, userID *uuid.UUID) ([]*domain.Media, error) {
	var mediaList []*domain.Media
	query := `
		SELECT m.* FROM media m
		JOIN album_photos ap ON m.id = ap.media_id
		JOIN albums a ON a.id = ap.album_id
		WHERE ap.album_id = $1
	`
	args := []interface{}{albumID}

	if userID != nil {
		query += ` AND a.user_id = $2`
		args = append(args, *userID)
	}

	err := r.db.SelectContext(ctx, &mediaList, query, args...)
	if err != nil {
		return nil, err
	}
	return mediaList, nil
}
