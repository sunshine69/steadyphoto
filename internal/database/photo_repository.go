package database

import (
	"context"
	"fmt"

	"steadyphoto/internal/domain"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type PostgresPhotoRepository struct {
	db *sqlx.DB
}

func NewPostgresPhotoRepository(db *sqlx.DB) *PostgresPhotoRepository {
	return &PostgresPhotoRepository{db: db}
}

func (r *PostgresPhotoRepository) Create(ctx context.Context, p *domain.Photo) error {
	query := `
		INSERT INTO photos (hash, filename, path, size_bytes, width, height, captured_at, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at
	`
	err := r.db.QueryRowxContext(ctx, query,
		p.Hash,
		p.Filename,
		p.Path,
		p.SizeBytes,
		p.Width,
		p.Height,
		p.CapturedAt,
		p.Metadata,
	).Scan(&p.ID, &p.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create photo: %w", err)
	}
	return nil
}

func (r *PostgresPhotoRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Photo, error) {
	var p domain.Photo
	query := `SELECT id, hash, filename, path, size_bytes, width, height, captured_at, metadata, created_at FROM photos WHERE id = $1`
	err := r.db.GetContext(ctx, &p, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get photo by id: %w", err)
	}
	return &p, nil
}

func (r *PostgresPhotoRepository) GetByHash(ctx context.Context, hash string) (*domain.Photo, error) {
	var p domain.Photo
	query := `SELECT id, hash, filename, path, size_bytes, width, height, captured_at, metadata, created_at FROM photos WHERE hash = $1`
	err := r.db.GetContext(ctx, &p, query, hash)
	if err != nil {
		return nil, fmt.Errorf("failed to get photo by hash: %w", err)
	}
	return &p, nil
}

func (r *PostgresPhotoRepository) List(ctx context.Context, limit, offset int) ([]*domain.Photo, error) {
	var photos []*domain.Photo
	query := `SELECT id, hash, filename, path, size_bytes, width, height, captured_at, metadata, created_at
			  FROM photos
			  ORDER BY captured_at DESC
			  LIMIT $1 OFFSET $2`
	err := r.db.SelectContext(ctx, &photos, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list photos: %w", err)
	}
	return photos, nil
}

func (r *PostgresPhotoRepository) Update(ctx context.Context, p *domain.Photo) error {
	query := `
		UPDATE photos
		SET filename = $1, path = $2, size_bytes = $3, width = $4, height = $5, captured_at = $6, metadata = $7
		WHERE id = $8
	`
	_, err := r.db.ExecContext(ctx, query,
		p.Filename,
		p.Path,
		p.SizeBytes,
		p.Width,
		p.Height,
		p.CapturedAt,
		p.Metadata,
		p.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update photo: %w", err)
	}
	return nil
}

// --- Face Repository Implementation ---

type PostgresFaceRepository struct {
	db *sqlx.DB
}

func NewPostgresFaceRepository(db *sqlx.DB) *PostgresFaceRepository {
	return &PostgresFaceRepository{db: db}
}

func (r *PostgresFaceRepository) Create(ctx context.Context, f *domain.Face) error {
	query := `
		INSERT INTO faces (photo_id, bounding_box, embedding)
		VALUES ($1, $2, $3)
		RETURNING id
	`
	err := r.db.QueryRowxContext(ctx, query,
		f.PhotoID,
		f.BoundingBox,
		f.Embedding,
	).Scan(&f.ID)

	if err != nil {
		return fmt.Errorf("failed to create face: %w", err)
	}
	return nil
}

func (r *PostgresFaceRepository) GetByPhotoID(ctx context.Context, photoID uuid.UUID) ([]*domain.Face, error) {
	var faces []*domain.Face
	query := `SELECT id, photo_id, bounding_box, embedding FROM faces WHERE photo_id = $1`
	err := r.db.SelectContext(ctx, &faces, query, photoID)
	if err != nil {
		return nil, fmt.Errorf("failed to get faces by photo id: %w", err)
	}
	return faces, nil
}

// --- Job Repository Implementation ---

type PostgresJobRepository struct {
	db *sqlx.DB
}

func NewPostgresJobRepository(db *sqlx.DB) *PostgresJobRepository {
	return &PostgresJobRepository{db: db}
}

func (r *PostgresJobRepository) Create(ctx context.Context, j *domain.Job) error {
	query := `
		INSERT INTO jobs (photo_id, job_type, status)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`
	err := r.db.QueryRowxContext(ctx, query,
		j.PhotoID,
		string(j.Type),
		string(j.Status),
	).Scan(&j.ID, &j.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create job: %w", err)
	}
	return nil
}

func (r *PostgresJobRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Job, error) {
	var j domain.Job
	query := `SELECT id, photo_id, job_type, status, error_message, created_at FROM jobs WHERE id = $1`
	err := r.db.GetContext(ctx, &j, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get job by id: %w", err)
	}
	// Convert string back to domain types
	j.Type = domain.JobType(j.Type)
	j.Status = domain.JobStatus(j.Status)
	return &j, nil
}

func (r *PostgresJobRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.JobStatus, errMsg *string) error {
	query := `UPDATE jobs SET status = $1, error_message = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $3`
	_, err := r.db.ExecContext(ctx, query, string(status), errMsg, id)
	if err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}
	return nil
}

func (r *PostgresJobRepository) GetPending(ctx context.Context, limit int) ([]*domain.Job, error) {
	var jobs []*domain.Job
	query := `SELECT id, photo_id, job_type, status, error_message, created_at FROM jobs
			  WHERE status = $1
			  ORDER BY created_at ASC
			  LIMIT $2`
	err := r.db.SelectContext(ctx, &jobs, query, string(domain.JobStatusPending), limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending jobs: %w", err)
	}

	for _, j := range jobs {
		j.Type = domain.JobType(j.Type)
		j.Status = domain.JobStatus(j.Status)
	}
	return jobs, nil
}
