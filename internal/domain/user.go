package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// User represents a registered user of the system
type User struct {
	ID           uuid.UUID `db:"id"`
	Email        string    `db:"email"`
	PasswordHash string    `db:"password_hash"`
	Status       string    `db:"status"` // pending, active, disabled, rejected
	Role         string    `db:"role"`   // admin or user
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
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
