package repository

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/kisnikita/safe-disputes/backend/internal/models"
	"log"
)

func (repo *Repository) InsertUser2Dispute(ctx context.Context, u2d models.User2Dispute) error {
	log.Printf("Inserting User2Dispute: %+v", u2d) // Log the data being inserted

	_, err := repo.db.ExecContext(ctx, `
	INSERT INTO user2dispute (id, user_id, dispute_id, result, status, claim) 
	VALUES ($1, $2, $3, $4, $5, $6)`,
		u2d.ID,
		u2d.UserID,
		u2d.DisputeID,
		u2d.Result,
		u2d.Status,
		u2d.Claim,
	)
	if err != nil {
		return fmt.Errorf("failed to insert user2dispute: %w", err)
	}

	return nil
}

func (repo *Repository) GetOpponentID(ctx context.Context, disputeID uuid.UUID, creatorID uuid.UUID) (uuid.UUID, error) {
	var opponentID uuid.UUID
	if err := repo.db.QueryRowContext(ctx, `
	SELECT user_id 
	FROM user2dispute
	WHERE dispute_id = $1 AND user_id != $2`,
		disputeID, creatorID,
	).Scan(&opponentID); err != nil {
		return uuid.Nil, fmt.Errorf("failed to get opponent: %w", err)
	}
	return opponentID, nil
}

func (repo *Repository) GetUser2Dispute(ctx context.Context, disputeID uuid.UUID, userID uuid.UUID) (models.User2Dispute, error) {
	var u2d models.User2Dispute
	if err := repo.db.QueryRowContext(ctx, `
	SELECT id, user_id, dispute_id, status, result, vote, claim
	FROM user2dispute
	WHERE dispute_id = $1 AND user_id = $2`,
		disputeID, userID,
	).Scan(
		&u2d.ID,
		&u2d.UserID,
		&u2d.DisputeID,
		&u2d.Status,
		&u2d.Result,
		&u2d.Vote,
		&u2d.Claim,
	); err != nil {
		return models.User2Dispute{}, fmt.Errorf("failed to get user2dispute: %w", err)
	}
	return u2d, nil
}

func (repo *Repository) UpdateUser2Dispute(ctx context.Context, opts models.U2DUpdateOpts) error {
	query := `
		UPDATE user2dispute
		SET status = COALESCE($1, status),
			result = COALESCE($2, result),
			vote = COALESCE($3, vote),
			claim = COALESCE($4, false)
		WHERE id = $5
	`
	_, err := repo.db.ExecContext(ctx, query, opts.Status, opts.Result, opts.Vote, opts.Claim, opts.ID)
	if err != nil {
		return fmt.Errorf("failed to update user2dispute: %w", err)
	}
	return nil
}
