package database

import (
	"context"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"steadyphoto/internal/domain"
)

type PostgresFaceRepository struct {
	db *sqlx.DB
}

func NewPostgresFaceRepository(db *sqlx.DB) *PostgresFaceRepository {
	return &PostgresFaceRepository{db: db}
}

func (r *PostgresFaceRepository) Create(ctx context.Context, face *domain.Face) error {
	query := `
		INSERT INTO faces (id, photo_id, bounding_box, embedding, created_at)
		VALUES (:id, :photo_id, :bounding_box, :embedding, :created_at)
	`
	// Note: bounding_box and embedding will be handled by JSONB/pgvector via the domain model
	_, err := r.db.NamedExecContext(ctx, query, face)
	return err
}

func (r *PostgresFaceRepository) GetByPhotoID(ctx context.Context, photoID uuid.UUID) ([]*domain.Face, error) {
	var faces []*domain.Face
	query := `SELECT * FROM faces WHERE photo_id = $1`
	err := r.db.SelectContext(ctx, &faces, query, photoID)
	if err != nil {
		return nil, err
	}
	return faces, nil
}

func (r *PostgresFaceRepository) DeleteByPhotoID(ctx context.Context, photoID uuid.UUID) error {
	query := `DELETE FROM faces WHERE photo_id = $1`
	_, err := r.db.ExecContext(ctx, query, photoID)
	return err
}
