package repository

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/kisnikita/safe-disputes/backend/internal/models"
)

func (repo *Repository) InsertEvidence(ctx context.Context, evidence models.Evidence) error {
	_, err := repo.db.ExecContext(ctx, `
	INSERT INTO evidence (id, user_id, dispute_id, description, image_data, image_type) 
	VALUES ($1, $2, $3, $4, $5, $6)`,
		evidence.ID,
		evidence.UserID,
		evidence.DisputeID,
		evidence.Description,
		evidence.ImageData,
		evidence.ImageType,
	)
	if err != nil {
		return fmt.Errorf("failed to insert evidence: %w", err)
	}
	return nil
}

func (repo *Repository) IsFirstEvidence(ctx context.Context, disputeID string) (bool, error) {
	var count int
	err := repo.db.QueryRowContext(ctx, `
	SELECT COUNT(*) FROM evidence WHERE dispute_id = $1`, disputeID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to count evidence: %w", err)
	}
	return count == 0, nil
}

func (repo *Repository) GetEvidences(ctx context.Context, disputeID uuid.UUID) ([]models.Evidence, error) {
	var evidences []models.Evidence
	rows, err := repo.db.QueryContext(ctx, `
	SELECT id, user_id, dispute_id, description, image_data, image_type 
	FROM evidence WHERE dispute_id = $1
	ORDER BY created_at`, disputeID)
	if err != nil {
		return nil, fmt.Errorf("failed to query evidences: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var e models.Evidence
		if err := rows.Scan(&e.ID, &e.UserID, &e.DisputeID, &e.Description, &e.ImageData, &e.ImageType); err != nil {
			return nil, fmt.Errorf("failed to scan evidence: %w", err)
		}
		evidences = append(evidences, e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return evidences, nil
}
