package database

import (
	"context"
	"fmt"

	"steadyphoto/internal/domain"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// PostgresPublicShareAccessRepository implements PublicShareAccessRepository
type PostgresPublicShareAccessRepository struct {
	db *sqlx.DB
}

func NewPostgresPublicShareAccessRepository(db *sqlx.DB) *PostgresPublicShareAccessRepository {
	return &PostgresPublicShareAccessRepository{db: db}
}

func (r *PostgresPublicShareAccessRepository) CreateAccessLog(ctx context.Context, publicShareID uuid.UUID, ipAddress string) error {
	query := `INSERT INTO public_share_accesses (id, public_share_id, ip_address, accessed_at) VALUES (:id, :public_share_id, :ip_address, NOW())`

	access := &domain.PublicShareAccess{
		ID:            uuid.New(),
		PublicShareID: publicShareID,
		IPAddress:     ipAddress,
	}

	if _, err := r.db.NamedExecContext(ctx, query, access); err != nil {
		return fmt.Errorf("failed to create access log: %w", err)
	}

	return nil
}
