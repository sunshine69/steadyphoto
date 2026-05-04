package database

import (
	"context"
	"time"

	"steadyphoto/internal/domain"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type PostgresJobRepository struct {
	db *sqlx.DB
}

func NewPostgresJobRepository(db *sqlx.DB) *PostgresJobRepository {
	return &PostgresJobRepository{db: db}
}

func (r *PostgresJobRepository) Create(ctx context.Context, job *domain.Job) error {
	query := `
		INSERT INTO jobs (id, type, status, photo_id, created_at, updated_at, error)
		VALUES (:id, :type, :status, :photo_id, :created_at, :updated_at, :error)
	`
	_, err := r.db.NamedExecContext(ctx, query, job)
	return err
}

func (r *PostgresJobRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Job, error) {
	var job domain.Job
	query := `SELECT * FROM jobs WHERE id = $1`
	err := r.db.GetContext(ctx, &job, query, id)
	if err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *PostgresJobRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.JobStatus, errStr string) error {
	query := `
		UPDATE jobs 
		SET status = $1, error = $2, updated_at = $3 
		WHERE id = $4
	`
	_, err := r.db.ExecContext(ctx, query, status, errStr, time.Now(), id)
	return err
}

func (r *PostgresJobRepository) GetPending(ctx context.Context, limit int) ([]*domain.Job, error) {
	var jobs []*domain.Job
	query := `SELECT * FROM jobs WHERE status = 'pending' ORDER BY created_at ASC LIMIT $1`
	err := r.db.SelectContext(ctx, &jobs, query, limit)
	if err != nil {
		return nil, err
	}
	return jobs, nil
}
