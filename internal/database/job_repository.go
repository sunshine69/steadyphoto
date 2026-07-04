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
		INSERT INTO jobs (id, job_type, status, media_id, created_at, updated_at, error_message)
		VALUES (:id, :job_type, :status, :media_id, :created_at, :updated_at, :error_message)
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
		SET status = $1, error_message = $2, updated_at = $3 
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

// GetJobsByMediaID returns all jobs for a given media ID regardless of status
func (r *PostgresJobRepository) GetJobsByMediaID(ctx context.Context, mediaID uuid.UUID) ([]*domain.Job, error) {
	var jobs []*domain.Job
	query := `SELECT * FROM jobs WHERE media_id = $1 ORDER BY created_at DESC`
	err := r.db.SelectContext(ctx, &jobs, query, mediaID)
	if err != nil {
		return nil, err
	}
	return jobs, nil
}

// ResetJobsAll resets all jobs across all users back to pending status
// This is used for global thumbnail regeneration
func (r *PostgresJobRepository) ResetJobsAll(ctx context.Context) (int, error) {
	query := `UPDATE jobs SET status = 'pending', error_message = NULL, updated_at = $1`
	result, err := r.db.ExecContext(ctx, query, time.Now())
	if err != nil {
		return 0, err
	}
	rowsAffected, _ := result.RowsAffected()
	return int(rowsAffected), nil
}

// ResetJobsByUserID sets all jobs for a user's media back to pending status
// This is used for force thumbnail regeneration
func (r *PostgresJobRepository) ResetJobsByUserID(ctx context.Context, userID uuid.UUID) (int, error) {
	query := `
		UPDATE jobs 
		SET status = 'pending', error_message = NULL, updated_at = $2 
		WHERE media_id IN (SELECT id FROM media WHERE user_id = $1)
	`
	result, err := r.db.ExecContext(ctx, query, userID, time.Now())
	if err != nil {
		return 0, err
	}
	rowsAffected, _ := result.RowsAffected()
	return int(rowsAffected), nil
}
