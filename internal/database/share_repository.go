package database

import (
	"context"
	"fmt"
	"time"

	"steadyphoto/internal/domain"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// PostgresShareRepository implements ShareRepository
type PostgresShareRepository struct {
	db *sqlx.DB
}

func NewPostgresShareRepository(db *sqlx.DB) *PostgresShareRepository {
	return &PostgresShareRepository{db: db}
}

func (r *PostgresShareRepository) CreateShare(ctx context.Context, sharerUserID uuid.UUID, shareeUserID uuid.UUID) (*domain.Share, error) {
	share := &domain.Share{
		ID:           uuid.New(),
		SharerUserID: sharerUserID,
		ShareeUserID: shareeUserID,
		SharedAt:     time.Now(),
	}

	query := `INSERT INTO shares (id, sharer_user_id, sharee_user_id, shared_at) VALUES (:id, :sharer_user_id, :sharee_user_id, :shared_at)`
	_, err := r.db.NamedExecContext(ctx, query, share)
	if err != nil {
		return nil, fmt.Errorf("failed to create share: %w", err)
	}

	return share, nil
}

func (r *PostgresShareRepository) GetBySharerAndSharee(ctx context.Context, sharerUserID uuid.UUID, shareeUserID uuid.UUID) (*domain.Share, error) {
	var share domain.Share
	query := `SELECT * FROM shares WHERE sharer_user_id = $1 AND sharee_user_id = $2`
	err := r.db.GetContext(ctx, &share, query, sharerUserID, shareeUserID)
	if err != nil {
		return nil, fmt.Errorf("share not found: %w", err)
	}

	return &share, nil
}
