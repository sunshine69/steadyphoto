package database

import (
	"context"
	"fmt"
	"strings"

	"steadyphoto/internal/domain"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type PostgresUserRepository struct {
	db *sqlx.DB
}

func NewPostgresUserRepository(db *sqlx.DB) *PostgresUserRepository {
	return &PostgresUserRepository{db: db}
}

func (r *PostgresUserRepository) Create(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (id, email, password_hash, status, role, created_at, updated_at)
		VALUES (:id, :email, :password_hash, :status, :role, :created_at, :updated_at)
	`
	_, err := r.db.NamedExecContext(ctx, query, user)
	return err
}

func (r *PostgresUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	query := `SELECT * FROM users WHERE email = $1`
	err := r.db.GetContext(ctx, &user, query, email)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *PostgresUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var user domain.User
	query := `SELECT * FROM users WHERE id = $1`
	err := r.db.GetContext(ctx, &user, query, id)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *PostgresUserRepository) Update(ctx context.Context, user *domain.User) error {
	query := `
		UPDATE users
		SET email = :email, password_hash = :password_hash, status = :status, role = :role, updated_at = :updated_at
		WHERE id = :id
	`
	_, err := r.db.NamedExecContext(ctx, query, user)
	return err
}

func (r *PostgresUserRepository) ListUsers(ctx context.Context, status string) ([]*domain.User, error) {
	var users []*domain.User
	query := "SELECT * FROM users"
	args := []interface{}{}

	if status != "" && status != "all" {
		query += " WHERE status = $1"
		args = append(args, status)
	}

	query += " ORDER BY created_at DESC"
	err := r.db.SelectContext(ctx, &users, query, args...)
	if err != nil {
		return nil, err
	}
	return users, nil
}

func (r *PostgresUserRepository) BulkUpdateStatus(ctx context.Context, userIDs []uuid.UUID, newStatus string) error {
	if len(userIDs) == 0 || newStatus == "" {
		return nil
	}

	placeholders := make([]string, len(userIDs))
	args := make([]interface{}, len(userIDs)+1)
	args[0] = newStatus

	for i, id := range userIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args[i+1] = id
	}

	query := fmt.Sprintf("UPDATE users SET status = $1, updated_at = NOW() WHERE id IN (%s)", strings.Join(placeholders, ", "))
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *PostgresUserRepository) BulkDeleteUsers(ctx context.Context, userIDs []uuid.UUID) error {
	if len(userIDs) == 0 {
		return nil
	}

	placeholders := make([]string, len(userIDs))
	args := make([]interface{}, len(userIDs))

	for i, id := range userIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf("DELETE FROM users WHERE id IN (%s)", strings.Join(placeholders, ", "))
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *PostgresUserRepository) GetByUsernameOrEmail(ctx context.Context, identifier string) (*domain.User, error) {
	var user domain.User
	query := `SELECT * FROM users WHERE email = $1 OR id::text = $1`
	err := r.db.GetContext(ctx, &user, query, identifier)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

type PostgresSessionRepository struct {
	db *sqlx.DB
}

func NewPostgresSessionRepository(db *sqlx.DB) *PostgresSessionRepository {
	return &PostgresSessionRepository{db: db}
}

// HashToken is a helper to hash the opaque token before storing it in DB (deprecated - use security package)
func (r *PostgresSessionRepository) HashToken(token string) string {
	return "" 
}

func (r *PostgresSessionRepository) RevokeAllByUserID(ctx context.Context, userID uuid.UUID) error {
	query := `UPDATE user_sessions SET is_revoked = TRUE WHERE user_id = $1`
	_, err := r.db.ExecContext(ctx, query, userID)
	return err
}

func (r *PostgresSessionRepository) CreateSession(ctx context.Context, session *domain.UserSession) error {
	query := `
		INSERT INTO user_sessions (id, user_id, refresh_token_hash, expires_at, is_revoked, created_at)
		VALUES (:id, :user_id, :refresh_token_hash, :expires_at, :is_revoked, :created_at)
	`
	_, err := r.db.NamedExecContext(ctx, query, session)
	return err
}

func (r *PostgresSessionRepository) GetByRefreshTokenHash(ctx context.Context, hash string) (*domain.UserSession, error) {
	var session domain.UserSession
	query := `SELECT * FROM user_sessions WHERE refresh_token_hash = $1 AND is_revoked = FALSE`
	err := r.db.GetContext(ctx, &session, query, hash)
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *PostgresSessionRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.UserSession, error) {
	var session domain.UserSession
	query := `SELECT * FROM user_sessions WHERE id = $1`
	err := r.db.GetContext(ctx, &session, query, id)
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *PostgresSessionRepository) UpdateSession(ctx context.Context, session *domain.UserSession) error {
	query := `
		UPDATE user_sessions
		SET is_revoked = :is_revoked, expires_at = :expires_at
		WHERE id = :id
	`
	_, err := r.db.NamedExecContext(ctx, query, session)
	return err
}

func (r *PostgresSessionRepository) RevokeSession(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE user_sessions SET is_revoked = TRUE WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}
