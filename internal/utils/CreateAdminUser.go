package utils

import (
	"context"
	"errors"
	"fmt"
	"github.com/jbrodriguez/mlog"
	"os"
	"time"

	"steadyphoto/internal/domain"
	"steadyphoto/internal/security"

	"github.com/google/uuid"
)

// CreateAdminUser inserts the initial admin user into the database using credentials from environment variables.
// If an admin with the same email already exists, it updates their password and role to ensure idempotency.
func CreateAdminUser(ctx context.Context, userRepo domain.UserRepository) error {
	email := os.Getenv("ADMIN_EMAIL")
	password := os.Getenv("ADMIN_PASSWORD")

	if email == "" || password == "" {
		return errors.New("ADMIN_EMAIL and ADMIN_PASSWORD environment variables must be set")
	}

	// 1. Check if admin user already exists
	existingUser, err := userRepo.GetByEmail(ctx, email)

	passwordHash, errHash := security.HashPassword(password)
	if errHash != nil {
		return fmt.Errorf("failed to hash admin password: %w", errHash)
	}

	now := time.Now()

	// Handle the case where user is found (Update) or not found (Create/Error handling)
	if existingUser != nil && err == nil {
		// 2. Update existing user (Idempotency: update credentials/role if they changed)
		mlog.Info("[INFO] Admin user with email %s already exists. Updating credentials.", email)
		existingUser.PasswordHash = passwordHash
		existingUser.Role = domain.UserRoleAdmin
		existingUser.Status = domain.UserStatusActive // Ensure they are active
		existingUser.UpdatedAt = now

		if err := userRepo.Update(ctx, existingUser); err != nil {
			return fmt.Errorf("failed to update admin user: %w", err)
		}
	} else if err == nil && existingUser == nil {
		// This case is unlikely with most drivers but for completeness...
	} else {
		if existingUser == nil {
			mlog.Info("[INFO] Creating new admin user: %s", email)
			adminUser := &domain.User{
				ID:           uuid.New(),
				Email:        email,
				PasswordHash: passwordHash,
				Status:       domain.UserStatusActive,
				Role:         domain.UserRoleAdmin,
				CreatedAt:    now,
				UpdatedAt:    now,
			}

			if err := userRepo.Create(ctx, adminUser); err != nil {
				return fmt.Errorf("failed to create admin user: %w", err)
			}
		} else if err != nil {
			// This is a real database error (e.g. connection issues, etc.)
			return fmt.Errorf("database error while checking existing admin: %w", err)
		}
	}

	mlog.Info("[INFO] Admin user setup complete for: %s", email)
	return nil
}
