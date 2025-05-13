package repository

import (
	"context"
	"fmt"

	"github.com/kisnikita/safe-disputes/backend/internal/models"
)

func (repo *Repository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	var user models.User
	err := handleNotFoundError(repo.db.QueryRowContext(ctx, `
	SELECT id, username, created_at, refresh_token, notification_enabled, dispute_readiness 
	FROM users WHERE username = $1`, username).Scan(
		&user.ID,
		&user.Username,
		&user.CreatedAt,
		&user.RefreshToken,
		&user.NotificationEnabled,
		&user.DisputeReadiness,
	))
	if err != nil {
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}
	return &user, nil
}

func (repo *Repository) ExistByUsername(ctx context.Context, username string) (bool, error) {
	var exists bool
	err := handleNotFoundError(repo.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)",
		username).Scan(&exists))
	if err != nil {
		return false, fmt.Errorf("failed to check existence of user by username: %w", err)
	}
	return exists, nil
}

func (repo *Repository) Insert(ctx context.Context, user *models.User) error {
	_, err := repo.db.ExecContext(ctx, `
	INSERT INTO users (id, username, created_at, refresh_token, notification_enabled, dispute_readiness) 
	VALUES ($1, $2, $3, $4, $5, $6)`,
		user.ID,
		user.Username,
		user.CreatedAt,
		user.RefreshToken,
		user.NotificationEnabled,
		user.DisputeReadiness,
	)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}
