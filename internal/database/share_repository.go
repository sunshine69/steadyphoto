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
// ListOutgoingShareGroups lists all outgoing share groups created by a user, enriched with sharer/sharee names.
func (r *PostgresShareRepository) ListOutgoingShareGroups(ctx context.Context, sharerUserID uuid.UUID) ([]*domain.ShareWithSharerName, error) {
	query := `
		SELECT s.id, s.sharer_user_id, s.sharee_user_id, s.shared_at, u.email as sharer_name, su.email as sharee_name
		FROM shares s
		JOIN users u ON s.sharer_user_id = u.id
		JOIN users su ON s.sharee_user_id = su.id
		WHERE s.sharer_user_id = $1
		ORDER BY s.shared_at DESC
	`

	var results []struct {
		ID         uuid.UUID `db:"id"`
		SharerUserID uuid.UUID `db:"sharer_user_id"`
		ShareeUserID uuid.UUID `db:"sharee_user_id"`
		SharedAt   time.Time `db:"shared_at"`
		SharerName   string       `db:"sharer_name"`

		ShareeName  string  `db:"sharee_name"`
	}

	err := r.db.SelectContext(ctx, &results, query, sharerUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to list outgoing share groups: %w", err)
	}

	shares := make([]*domain.ShareWithSharerName, len(results))
	for i, row := range results {
		shares[i] = &domain.ShareWithSharerName{
			ID:           row.ID,
			SharerUserID: row.SharerUserID,
			ShareeUserID: row.ShareeUserID,
			SharedAt:     row.SharedAt,
			SharerName:   row.SharerName,
			ShareeName:   row.ShareeName,
		}
	}

	return shares, nil
}

// ListOutgoingShares lists all outgoing share groups with details (media + albums).
func (r *PostgresShareRepository) ListOutgoingShares(ctx context.Context, sharerUserID uuid.UUID) ([]*domain.ShareWithDetails, error) {
	query := `
		SELECT s.id, s.sharer_user_id, s.sharee_user_id, s.shared_at
		FROM shares s
		WHERE s.sharer_user_id = $1
		ORDER BY s.shared_at DESC
	`

	var shares []*domain.ShareWithDetails

	rows, err := r.db.QueryContext(ctx, query, sharerUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to list outgoing shares: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		sd := &domain.ShareWithDetails{}
		err = rows.Scan(&sd.ID, &sd.SharerUserID, &sd.ShareeUserID, &sd.SharedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan share row: %w", err)
		}

		// Get media for this share group
		mediaQuery := `
			SELECT m.id, m.filename, m.path, m.media_type
			FROM media_shares ms
			JOIN media m ON ms.media_id = m.id
			WHERE ms.share_id = $1 AND m.deleted_at IS NULL
		`

		var mediaRows []struct {
			ID        uuid.UUID     `db:"id"`
			Filename  string        `db:"filename"`
			Path      *string       `db:"path"`
			MediaType domain.MediaType `db:"media_type"`
		}

		err = r.db.SelectContext(ctx, &mediaRows, mediaQuery, sd.ID)
		if err == nil { // Don't fail if no media found for this share group
			for _, m := range mediaRows {
				sd.Media = append(sd.Media, domain.OutgoingMediaInShare{
					ID:        m.ID,
					Filename:  m.Filename,
					Path:      m.Path,
					MediaType: m.MediaType,
				})
			}
		}

		// Get albums for this share group
		albumsQuery := `
			SELECT a.id, a.name, a.description
			FROM album_shares albs
			JOIN albums a ON albs.album_id = a.id
			WHERE albs.share_id = $1
		`

		var albumRows []struct {
			ID          uuid.UUID `db:"id"`
			Name        string    `db:"name"`
			Description *string   `db:"description"`
		}

		err = r.db.SelectContext(ctx, &albumRows, albumsQuery, sd.ID)
		if err == nil { // Don't fail if no albums found for this share group
			for _, a := range albumRows {
				sd.Albums = append(sd.Albums, domain.OutgoingAlbumInShare{
					ID:          a.ID,
					Name:        a.Name,
					Description: a.Description,
				})
			}
		}

		shares = append(shares, sd)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return shares, nil
}

// DeleteShareGroup deletes a share group along with its media_shares and album_shares entries.
func (r *PostgresShareRepository) DeleteShareGroup(ctx context.Context, shareID uuid.UUID) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		}
	}()

	// Delete media shares
	_, err = tx.ExecContext(ctx, "DELETE FROM media_shares WHERE share_id = $1", shareID)
	if err != nil {
		return fmt.Errorf("failed to delete media_shares: %w", err)
	}

	// Delete album shares
	_, err = tx.ExecContext(ctx, "DELETE FROM album_shares WHERE share_id = $1", shareID)
	if err != nil {
		return fmt.Errorf("failed to delete album_shares: %w", err)
	}

	// Delete the share group itself
	_, err = tx.ExecContext(ctx, "DELETE FROM shares WHERE id = $1", shareID)
	if err != nil {
		return fmt.Errorf("failed to delete share: %w", err)
	}

	return tx.Commit()
}
