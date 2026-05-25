package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// User represents a registered user of the system
type User struct {
	ID           uuid.UUID `json:"id" db:"id"`
	Email        string    `json:"email" db:"email"`
	PasswordHash string    `json:"-" db:"password_hash"` // Never expose password hash in JSON
	Status       string    `json:"status" db:"status"`   // pending, active, disabled, rejected
	Role         string    `json:"role" db:"role"`       // admin or user
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

const (
	UserStatusPending  = "pending"
	UserStatusActive   = "active"
	UserStatusDisabled = "disabled"
	UserStatusRejected = "rejected"

	UserRoleAdmin = "admin"
	UserRoleUser  = "user"
)

// UserSession represents an active authentication session for a user (Opaque Token)
type UserSession struct {
	ID               uuid.UUID `db:"id"`
	UserID           uuid.UUID `db:"user_id"`
	RefreshTokenHash string    `db:"refresh_token_hash"` // SHA256 of the long-lived token
	ExpiresAt        time.Time `db:"expires_at"`
	IsRevoked        bool      `db:"is_revoked"`
	CreatedAt        time.Time `db:"created_at"`
}

// UserRepository defines the interface for user storage and retrieval
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	Update(ctx context.Context, user *User) error
	
	// Admin methods for user management
	ListUsers(ctx context.Context, status string) ([]*User, error)
	GetByUsernameOrEmail(ctx context.Context, identifier string) (*User, error)
	
	// Bulk operations for admin management
	BulkUpdateStatus(ctx context.Context, userIDs []uuid.UUID, newStatus string) error
	BulkDeleteUsers(ctx context.Context, userIDs []uuid.UUID) error
}

// SessionRepository defines the interface for session lifecycle management
type SessionRepository interface {
	CreateSession(ctx context.Context, session *UserSession) error
	GetByRefreshTokenHash(ctx context.Context, hash string) (*UserSession, error)
	GetByID(ctx context.Context, id uuid.UUID) (*UserSession, error)
	UpdateSession(ctx context.Context, session *UserSession) error
	RevokeSession(ctx context.Context, id uuid.UUID) error
	// RevokeAllByUserID revokes all sessions for a given user (useful during account deletion/disabling)
	RevokeAllByUserID(ctx context.Context, userID uuid.UUID) error
}

// ... existing Media/Face/Album types and interfaces follow below in the real file
