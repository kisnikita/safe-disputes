package repository

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/kisnikita/safe-disputes/backend/internal/models"
)

func (repo *Repository) InsertParticipant(ctx context.Context, participant models.Participant) error {
	_, err := repo.db.ExecContext(ctx, `
	INSERT INTO participants (id, user_id, dispute_id, result, status, claim) 
	VALUES ($1, $2, $3, $4, $5, $6)`,
		participant.ID,
		participant.UserID,
		participant.DisputeID,
		participant.Result,
		participant.Status,
		participant.Claim,
	)
	if err != nil {
		return fmt.Errorf("failed to insert participants: %w", err)
	}

	return nil
}

func (repo *Repository) GetOpponentID(ctx context.Context, disputeID uuid.UUID, actorID uuid.UUID) (uuid.UUID, error) {
	var opponentID uuid.UUID
	if err := repo.db.QueryRowContext(ctx, `
	SELECT user_id 
	FROM participants
	WHERE dispute_id = $1 AND user_id != $2`,
		disputeID, actorID,
	).Scan(&opponentID); err != nil {
		return uuid.Nil, fmt.Errorf("failed to get opponent: %w", err)
	}
	return opponentID, nil
}

func (repo *Repository) GetParticipant(ctx context.Context, disputeID uuid.UUID, userID uuid.UUID) (models.Participant, error) {
	var participant models.Participant
	if err := repo.db.QueryRowContext(ctx, `
	SELECT id, user_id, dispute_id, status, result, vote, claim
	FROM participants
	WHERE dispute_id = $1 AND user_id = $2`,
		disputeID, userID,
	).Scan(
		&participant.ID,
		&participant.UserID,
		&participant.DisputeID,
		&participant.Status,
		&participant.Result,
		&participant.Vote,
		&participant.Claim,
	); err != nil {
		return models.Participant{}, fmt.Errorf("failed to get participants: %w", err)
	}
	return participant, nil
}

func (repo *Repository) UpdateParticipant(ctx context.Context, opts models.ParticipantUpdateOpts) error {
	query := `
		UPDATE participants
		SET status = COALESCE($1, status),
			result = COALESCE($2, result),
			vote = COALESCE($3, vote),
			claim = COALESCE($4, claim)
		WHERE id = $5
	`
	_, err := repo.db.ExecContext(ctx, query, opts.Status, opts.Result, opts.Vote, opts.Claim, opts.ID)
	if err != nil {
		return fmt.Errorf("failed to update participants: %w", err)
	}
	return nil
}
