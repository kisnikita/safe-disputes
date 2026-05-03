package repository

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/kisnikita/safe-disputes/backend/internal/models"
)

func (repo *Repository) InsertDisputeParticipant(ctx context.Context, disputeParticipant models.DisputeParticipant) error {
	_, err := repo.db.ExecContext(ctx, `
	INSERT INTO dispute_participants (id, user_id, dispute_id, result, status, claim) 
	VALUES ($1, $2, $3, $4, $5, $6)`,
		disputeParticipant.ID,
		disputeParticipant.UserID,
		disputeParticipant.DisputeID,
		disputeParticipant.Result,
		disputeParticipant.Status,
		disputeParticipant.Claim,
	)
	if err != nil {
		return fmt.Errorf("failed to insert dispute_participants: %w", err)
	}

	return nil
}

func (repo *Repository) GetOpponentID(ctx context.Context, disputeID uuid.UUID, actorID uuid.UUID) (uuid.UUID, error) {
	var opponentID uuid.UUID
	if err := repo.db.QueryRowContext(ctx, `
	SELECT user_id 
	FROM dispute_participants
	WHERE dispute_id = $1 AND user_id != $2`,
		disputeID, actorID,
	).Scan(&opponentID); err != nil {
		return uuid.Nil, fmt.Errorf("failed to get opponent: %w", err)
	}
	return opponentID, nil
}

func (repo *Repository) GetDisputeParticipant(ctx context.Context, disputeID uuid.UUID, userID uuid.UUID) (models.DisputeParticipant, error) {
	var disputeParticipant models.DisputeParticipant
	if err := repo.db.QueryRowContext(ctx, `
	SELECT id, user_id, dispute_id, status, result, vote, claim
	FROM dispute_participants
	WHERE dispute_id = $1 AND user_id = $2`,
		disputeID, userID,
	).Scan(
		&disputeParticipant.ID,
		&disputeParticipant.UserID,
		&disputeParticipant.DisputeID,
		&disputeParticipant.Status,
		&disputeParticipant.Result,
		&disputeParticipant.Vote,
		&disputeParticipant.Claim,
	); err != nil {
		return models.DisputeParticipant{}, fmt.Errorf("failed to get dispute_participants: %w", err)
	}
	return disputeParticipant, nil
}

func (repo *Repository) UpdateDisputeParticipant(ctx context.Context, opts models.DisputeParticipantUpdateOpts) error {
	query := `
		UPDATE dispute_participants
		SET status = COALESCE($1, status),
			result = COALESCE($2, result),
			vote = COALESCE($3, vote),
			claim = COALESCE($4, claim)
		WHERE id = $5
	`
	_, err := repo.db.ExecContext(ctx, query, opts.Status, opts.Result, opts.Vote, opts.Claim, opts.ID)
	if err != nil {
		return fmt.Errorf("failed to update dispute_participants: %w", err)
	}
	return nil
}
