package repository

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/kisnikita/safe-disputes/backend/internal/models"
	"github.com/lib/pq"
)

func (repo *Repository) BroadcastInvestigation(ctx context.Context, u2i models.User2Investigation, p1, p2 uuid.UUID,
) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	rows, err := repo.db.QueryContext(ctx, `SELECT id FROM users`)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch users: %w", err)
	}
	defer rows.Close()

	// Insert investigation for each user
	for rows.Next() {
		var (
			userID uuid.UUID
		)
		if err := rows.Scan(&userID); err != nil {
			return nil, fmt.Errorf("failed to scan user ID: %w", err)
		}
		if userID == p1 || userID == p2 {
			continue
		}

		_, err := repo.db.ExecContext(ctx, `
			INSERT INTO user2investigation (id, user_id, investigation_id, result, vote)
			VALUES ($1, $2, $3, $4, $5)`,
			uuid.New(), userID, u2i.InvestigationID, u2i.Result, u2i.Vote,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to insert user2investigation: %w", err)
		}
		ids = append(ids, userID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over users: %w", err)
	}

	return ids, nil
}

func (repo *Repository) GetUser2Investigation(ctx context.Context, invID, userID uuid.UUID) (models.User2Investigation, error) {
	var u2i models.User2Investigation
	if err := repo.db.QueryRowContext(ctx, `
		SELECT id, investigation_id, user_id, vote, result
		FROM user2investigation
		WHERE investigation_id = $1 AND user_id = $2`,
		invID, userID,
	).Scan(
		&u2i.ID,
		&u2i.InvestigationID,
		&u2i.UserID,
		&u2i.Vote,
		&u2i.Result,
	); err != nil {
		return models.User2Investigation{}, fmt.Errorf("failed to get user2investigation: %w", err)
	}
	return u2i, nil
}

func (repo *Repository) UpdateUser2Investigation(ctx context.Context, opts models.U2IUpdateOpts) error {
	query := `
		UPDATE user2investigation
		SET vote = COALESCE($1, vote),
			result = COALESCE($2, result)
		WHERE id = $3
	`
	_, err := repo.db.ExecContext(ctx, query, opts.Vote, opts.Result, opts.ID)
	if err != nil {
		return fmt.Errorf("failed to update user2investigation: %w", err)
	}
	return nil
}

func (repo *Repository) DeleteUsersWithoutVote(ctx context.Context, invID uuid.UUID) error {
	query := `
		DELETE FROM user2investigation
		WHERE investigation_id = $1 AND vote = ''
	`
	_, err := repo.db.ExecContext(ctx, query, invID)
	if err != nil {
		return fmt.Errorf("failed to delete users without vote: %w", err)
	}
	return nil
}

func (repo *Repository) GetWinnersIDs(ctx context.Context, invID uuid.UUID, winner string) ([]uuid.UUID, error) {
	var ids []uuid.UUID

	rows, err := repo.db.QueryContext(ctx, `
		SELECT user_id
		FROM user2investigation
		WHERE investigation_id = $1 AND vote = $2`,
		invID, winner,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch winner user IDs: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var userID uuid.UUID
		if err := rows.Scan(&userID); err != nil {
			return nil, fmt.Errorf("failed to scan user ID: %w", err)
		}
		ids = append(ids, userID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over winner user IDs: %w", err)
	}

	return ids, nil
}

func (repo *Repository) UpdateWinnersResult(ctx context.Context, invID uuid.UUID, ids []uuid.UUID) error {
	queryCorrect := `
		UPDATE user2investigation
		SET result = $1
		WHERE investigation_id = $2 AND user_id = ANY($3)
	`
	_, err := repo.db.ExecContext(ctx, queryCorrect, models.InvestigationResultCorrect, invID, pq.Array(ids))
	if err != nil {
		return fmt.Errorf("failed to update result to correct: %w", err)
	}

	queryIncorrect := `
		UPDATE user2investigation
		SET result = $1
		WHERE investigation_id = $2 AND user_id != ALL($3)
	`
	_, err = repo.db.ExecContext(ctx, queryIncorrect, models.InvestigationResultInCorrect, invID, pq.Array(ids))
	if err != nil {
		return fmt.Errorf("failed to update result to incorrect: %w", err)
	}

	return nil
}

func (repo *Repository) GetDisputesUsers(ctx context.Context, invID uuid.UUID) ([]models.User, error) {
	query := `
  SELECT u.id, u.username, u.chat_id, u.notification_enabled, u.rating
  FROM evidence e
  JOIN users u ON e.user_id = u.id
  WHERE e.dispute_id IN (
    SELECT dispute_id
    FROM investigations
    WHERE id = $1
  )
  ORDER BY e.created_at
 `

	rows, err := repo.db.QueryContext(ctx, query, invID)
	if err != nil {
		return nil, fmt.Errorf("failed to execute GetDisputesUsers query: %w", err)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		if err := rows.Scan(&user.ID, &user.Username, &user.ChatID, &user.NotificationEnabled, &user.Rating); err != nil {
			return nil, fmt.Errorf("failed to scan user row: %w", err)
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over rows: %w", err)
	}

	return users, nil
}
