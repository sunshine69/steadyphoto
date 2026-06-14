package database

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"steadyphoto/internal/domain"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// PostgresPublicShareRepository implements PublicShareRepository
type PostgresPublicShareRepository struct {
	db *sqlx.DB
}

func NewPostgresPublicShareRepository(db *sqlx.DB) *PostgresPublicShareRepository {
	return &PostgresPublicShareRepository{db: db}
}

// generateToken generates a random URL-safe token for public share links.
func generateToken() (string, error) {
	bytes := make([]byte, 16) // 32 hex characters = 16 bytes
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random token: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

func (r *PostgresPublicShareRepository) CreatePublicShare(ctx context.Context, sharerUserID uuid.UUID, resourceType string, resourceID uuid.UUID, passwordHash *string, expiresAt *time.Time) (*domain.PublicShare, error) {
	token, err := generateToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	ps := &domain.PublicShare{
		ID:             uuid.New(),
		Token:          token,
		SharerUserID:   sharerUserID,
		ResourceType:   resourceType,
		ResourceID:     resourceID,
		PasswordHash:   passwordHash, // May be nil if not set
		ExpiresAt:      expiresAt,    // May be nil if no expiration
		CreatedAt:      time.Now(),
	}

	query := `INSERT INTO public_shares (id, token, sharer_user_id, resource_type, resource_id, password_hash, expires_at, created_at) VALUES (:id, :token, :sharer_user_id, :resource_type, :resource_id, :password_hash, :expires_at, :created_at)`
	_, err = r.db.NamedExecContext(ctx, query, ps)
	if err != nil {
		return nil, fmt.Errorf("failed to create public share: %w", err)
	}

	return ps, nil
}

func (r *PostgresPublicShareRepository) GetByToken(ctx context.Context, token string) (*domain.PublicShare, error) {
	var ps domain.PublicShare
	query := `SELECT * FROM public_shares WHERE token = $1`
	err := r.db.GetContext(ctx, &ps, query, token)
	if err != nil {
		return nil, fmt.Errorf("public share not found: %w", err)
	}

	return &ps, nil
}

func (r *PostgresPublicShareRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.PublicShare, error) {
	var ps domain.PublicShare
	query := `SELECT * FROM public_shares WHERE id = $1`
	err := r.db.GetContext(ctx, &ps, query, id)
	if err != nil {
		return nil, fmt.Errorf("public share not found: %w", err)
	}

	return &ps, nil
}

func (r *PostgresPublicShareRepository) DeleteByToken(ctx context.Context, token string) error {
	query := `DELETE FROM public_shares WHERE token = $1`
	result, err := r.db.ExecContext(ctx, query, token)
	if err != nil {
		return fmt.Errorf("failed to delete public share: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("public share not found for token: %s", token)
	}

	return nil
}

func (r *PostgresPublicShareRepository) ListBySharer(ctx context.Context, sharerUserID uuid.UUID) ([]*domain.PublicShareWithSharer, error) {
	query := `
		SELECT ps.*, u.email as sharer_email
		FROM public_shares ps
		JOIN users u ON ps.sharer_user_id = u.id
		WHERE ps.sharer_user_id = $1
		ORDER BY ps.created_at DESC
	`

	var results []struct {
		domain.PublicShare `db:",inline"`
		SharerEmail        string `db:"sharer_email" json:"-"` // Not used in response, but available for enrichment
	}

	err := r.db.SelectContext(ctx, &results, query, sharerUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to list public shares: %w", err)
	}

	items := make([]*domain.PublicShareWithSharer, len(results))
	for i, r := range results {
		items[i] = &domain.PublicShareWithSharer{
			ID:             r.ID,
			Token:          r.Token,
			SharerUserID:   r.SharerUserID,
			ResourceType:   r.ResourceType,
			ResourceID:     r.ResourceID,
			PasswordProtected: r.PasswordHash != nil && *r.PasswordHash != "",
			ExpiresAt:      r.ExpiresAt,
			CreatedAt:      r.CreatedAt,
			AccessCount:    r.AccessCount,
		}
	}

	return items, nil
}

func (r *PostgresPublicShareRepository) IncrementAccessCount(ctx context.Context, publicShareID uuid.UUID) error {
	query := `UPDATE public_shares SET access_count = access_count + 1 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, publicShareID)
	if err != nil {
		return fmt.Errorf("failed to increment access count: %w", err)
	}

	return nil
}

// GetSharedMediaByToken retrieves a media item by its public share token.
func (r *PostgresPublicShareRepository) GetSharedMediaByToken(ctx context.Context, token string) (*domain.PublicShareWithSharerInfo, error) {
	query := `
		SELECT m.*, ps.sharer_user_id, ps.token
		FROM public_shares ps
		JOIN media m ON ps.resource_type = 'media' AND ps.resource_id = m.id AND m.deleted_at IS NULL
		WHERE ps.token = $1
	`

	var result struct {
		domain.Media   `db:",inline"`
		SharerUserID   uuid.UUID       `db:"sharer_user_id" json:"-"`
		Token          string          `db:"token" json:"-"`
	}

	err := r.db.GetContext(ctx, &result, query, token)
	if err != nil {
		return nil, fmt.Errorf("shared media not found: %w", err)
	}

	return &domain.PublicShareWithSharerInfo{
		Media:        &result.Media,
		Token:        result.Token,
		SharerUserID: result.SharerUserID,
	}, nil
}

// GetSharedAlbumByToken retrieves an album and its media items by its public share token.
func (r *PostgresPublicShareRepository) GetSharedAlbumByToken(ctx context.Context, token string) (*domain.PublicShareAlbumWithSharerInfo, error) {
	query := `
		SELECT a.*, ps.sharer_user_id, ps.token
		FROM public_shares ps
		JOIN albums a ON ps.resource_type = 'album' AND ps.resource_id = a.id
		WHERE ps.token = $1
	`

	var albumResult struct {
		domain.Album   `db:",inline"`
		SharerUserID   uuid.UUID       `db:"sharer_user_id" json:"-"`
		Token          string          `db:"token" json:"-"`
	}

	err := r.db.GetContext(ctx, &albumResult, query, token)
	if err != nil {
		return nil, fmt.Errorf("shared album not found: %w", err)
	}

	// Get media items in the album (position sorted)
	var mediaResults []struct {
		domain.Media `db:",inline"`
	}

	mediaQuery := `
		SELECT m.* FROM media m
		JOIN album_photos ap ON m.id = ap.media_id
		WHERE ap.album_id = $1 AND m.deleted_at IS NULL
		ORDER BY ap.position ASC, m.captured_at DESC
	`

	err = r.db.SelectContext(ctx, &mediaResults, mediaQuery, albumResult.Album.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get media for shared album: %w", err)
	}

	mediaItems := make([]*domain.Media, len(mediaResults))
	for i, r := range mediaResults {
		mediaItems[i] = &r.Media
	}

	return &domain.PublicShareAlbumWithSharerInfo{
		Album:      &albumResult.Album,
		Token:      albumResult.Token,
		MediaItems: mediaItems,
	}, nil
}
