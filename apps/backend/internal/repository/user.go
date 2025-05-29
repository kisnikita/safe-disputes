package repository

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"go.uber.org/zap"

	"github.com/kisnikita/safe-disputes/backend/internal/models"
)

func (repo *Repository) GetUserByUsername(ctx context.Context, username string) (models.User, error) {
	var user models.User
	err := handleNotFoundError(repo.db.QueryRowContext(ctx, `
	SELECT id, username, created_at, notification_enabled, dispute_readiness, minimum_dispute_amount, rating, chat_id 
	FROM users WHERE username = $1`, username).Scan(
		&user.ID,
		&user.Username,
		&user.CreatedAt,
		&user.NotificationEnabled,
		&user.DisputeReadiness,
		&user.MinimumDisputeAmount,
		&user.Rating,
		&user.ChatID,
	))
	if err != nil {
		return models.User{}, fmt.Errorf("failed to get user by username: %w", err)
	}
	return user, nil
}

func (repo *Repository) GetUserByID(ctx context.Context, id uuid.UUID) (models.User, error) {
	var user models.User
	err := handleNotFoundError(repo.db.QueryRowContext(ctx, `
	SELECT id, username, created_at, notification_enabled, dispute_readiness, minimum_dispute_amount, rating, chat_id
	FROM users WHERE id = $1`, id).Scan(
		&user.ID,
		&user.Username,
		&user.CreatedAt,
		&user.NotificationEnabled,
		&user.DisputeReadiness,
		&user.MinimumDisputeAmount,
		&user.Rating,
		&user.ChatID,
	))
	if err != nil {
		return models.User{}, fmt.Errorf("failed to get user by username: %w", err)
	}
	return user, nil
}

func (repo *Repository) ExistByUsername(ctx context.Context, username string) (bool, error) {
	var exists bool
	err := repo.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)",
		username).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check existence of user by username: %w", err)
	}
	return exists, nil
}

func (repo *Repository) InsertUser(ctx context.Context, user models.User) error {
	repo.logger.Info("creating user", zap.String("username", user.Username))
	_, err := repo.db.ExecContext(ctx, `
	INSERT INTO users (id, username, chat_id, notification_enabled) 
	VALUES ($1, $2, $3, $4)`,
		user.ID,
		user.Username,
		user.ChatID,
		user.NotificationEnabled,
	)
	if err != nil {
		return fmt.Errorf("failed to insert user: %w", err)
	}
	return nil
}

func (repo *Repository) UpdateUser(ctx context.Context, opts models.UserUpdateOpts) error {
	query := `
		UPDATE users
		SET
			notification_enabled = COALESCE($1, notification_enabled),
			dispute_readiness = COALESCE($2, dispute_readiness),
			minimum_dispute_amount = COALESCE($3, minimum_dispute_amount), 
			rating = COALESCE($4, rating)
		WHERE username = $5
	`

	_, err := repo.db.ExecContext(ctx, query,
		opts.NotificationEnabled,
		opts.DisputeReadiness,
		opts.MinimumDisputeAmount,
		opts.Rating,
		opts.Username,
	)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}

func (repo *Repository) UpdateChatID(ctx context.Context, chatID int64, username string) error {
	query := `
		UPDATE users
		SET 
		    chat_id = $1,
			notification_enabled = true
		WHERE username = $2
	`

	_, err := repo.db.ExecContext(ctx, query,
		chatID,
		username,
	)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}

func (repo *Repository) GetTotalUsers(ctx context.Context) (int, error) {
	var total int

	if err := repo.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&total); err != nil {
		return 0, fmt.Errorf("failed to update user: %w", err)
	}
	return total, nil
}

func (repo *Repository) GetUsers(ctx context.Context, ids []uuid.UUID) ([]models.User, error) {
	var users []models.User

	query := `
		SELECT id, username, created_at, notification_enabled, dispute_readiness, minimum_dispute_amount, rating, chat_id
		FROM users
		WHERE id = ANY($1)
	`

	rows, err := repo.db.QueryContext(ctx, query, pq.Array(ids))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch users: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var user models.User
		if err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.CreatedAt,
			&user.NotificationEnabled,
			&user.DisputeReadiness,
			&user.MinimumDisputeAmount,
			&user.Rating,
			&user.ChatID,
		); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over users: %w", err)
	}

	return users, nil
}

func (repo *Repository) GetTopUsers(ctx context.Context, limit int) ([]models.User, error) {
	var users []models.User

	query := `
		SELECT username, rating
		FROM users
		ORDER BY rating DESC
		LIMIT $1
	`

	rows, err := repo.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch top users: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var user models.User
		if err := rows.Scan(
			&user.Username,
			&user.Rating,
		); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over users: %w", err)
	}

	return users, nil
}

func (repo *Repository) EarnWinnerRating(ctx context.Context, ids []uuid.UUID) error {
	query := `
		UPDATE users
		SET rating = rating + 3
		WHERE id = ANY($1)
	`

	_, err := repo.db.ExecContext(ctx, query, pq.Array(ids))
	if err != nil {
		return fmt.Errorf("failed to update winner ratings: %w", err)
	}
	return nil
}
