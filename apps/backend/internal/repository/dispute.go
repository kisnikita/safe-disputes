package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kisnikita/safe-disputes/backend/internal/models"
)

func (repo *Repository) InsertDispute(ctx context.Context, dispute models.Dispute) error {
	_, err := repo.db.ExecContext(ctx, `
	INSERT INTO disputes (
		id, title, description, created_at, updated_at, cryptocurrency, amount_nano, image_data, image_type,
		contract_address, ends_at, next_deadline
	) 
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		dispute.ID,
		dispute.Title,
		dispute.Description,
		dispute.CreatedAt,
		dispute.UpdatedAt,
		dispute.Cryptocurrency,
		dispute.AmountNano,
		dispute.ImageData,
		dispute.ImageType,
		dispute.ContractAddress,
		dispute.EndsAt,
		dispute.NextDeadline,
	)
	if err != nil {
		return fmt.Errorf("failed to insert dispute: %w", err)
	}
	return nil
}

func (repo *Repository) ListDisputeCards(ctx context.Context, actorUsername string, opts models.DisputeListOpts,
) ([]models.DisputeCard, error) {
	const maxLimit = 100

	var (
		clauses []string
		args    []interface{}
		idx     = 1
	)

	clauses = append(clauses, fmt.Sprintf("me.username = $%d", idx))
	args = append(args, actorUsername)
	idx++

	if opts.Status != nil {
		clauses = append(clauses, fmt.Sprintf("self.status = $%d", idx))
		args = append(args, *opts.Status)
		idx++
	}

	if opts.Result != nil {
		clauses = append(clauses, fmt.Sprintf("self.vote = $%d", idx))
		args = append(args, *opts.Result)
		idx++
	}

	if opts.Cursor != "" {
		t, err := time.Parse(time.RFC3339Nano, opts.Cursor)
		if err != nil {
			return nil, fmt.Errorf("invalid cursor format: %w", err)
		}
		clauses = append(clauses, fmt.Sprintf("d.created_at <= $%d", idx))
		args = append(args, t)
		idx++
	}

	whereSQL := ""
	if len(clauses) > 0 {
		whereSQL = "WHERE " + strings.Join(clauses, " AND ")
	}

	limit := opts.Limit
	if limit <= 0 || limit > maxLimit {
		limit = maxLimit
	}
	args = append(args, limit+1)

	query := fmt.Sprintf(`
		SELECT
			d.id, d.title,
			d.created_at, d.amount_nano,
			d.ends_at, d.next_deadline,
			opp_user.username AS opponent,
			opp_user.photo_url,
			self.result, self.vote, self.claim
		FROM disputes d
		JOIN participants self ON self.dispute_id = d.id
		JOIN users me ON me.id = self.user_id
		JOIN participants opp ON opp.dispute_id = d.id AND opp.user_id <> self.user_id
		JOIN users opp_user ON opp_user.id = opp.user_id
		%s
		ORDER BY d.created_at DESC
		LIMIT $%d
	`, whereSQL, idx)

	rows, err := repo.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute ListDisputeCards query: %w", err)
	}
	defer rows.Close()

	var disputes []models.DisputeCard
	for rows.Next() {
		var d models.DisputeCard
		if err := rows.Scan(
			&d.ID,
			&d.Title,
			&d.CreatedAt,
			&d.AmountNano,
			&d.EndsAt,
			&d.NextDeadline,
			&d.Opponent,
			&d.PhotoUrl,
			&d.Result,
			&d.Vote,
			&d.Claim,
		); err != nil {
			return nil, fmt.Errorf("failed to scan dispute read row: %w", err)
		}
		disputes = append(disputes, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over rows: %w", err)
	}

	return disputes, nil
}

func (repo *Repository) GetDisputeByID(ctx context.Context, disputeID uuid.UUID, actorID uuid.UUID,
) (models.Dispute, error) {
	var d models.Dispute
	err := repo.db.QueryRowContext(ctx, `
		SELECT 
			d.id, d.title, d.description, 
			d.created_at, d.updated_at, 
			d.cryptocurrency, d.amount_nano, 
			d.image_data, d.image_type, d.ends_at, d.next_deadline,
			d.contract_address
		FROM disputes d
		JOIN participants u ON d.id = u.dispute_id
		WHERE d.id = $1 AND u.user_id = $2`,
		disputeID, actorID,
	).Scan(
		&d.ID,
		&d.Title,
		&d.Description,
		&d.CreatedAt,
		&d.UpdatedAt,
		&d.Cryptocurrency,
		&d.AmountNano,
		&d.ImageData,
		&d.ImageType,
		&d.EndsAt,
		&d.NextDeadline,
		&d.ContractAddress,
	)
	if err != nil {
		return models.Dispute{}, fmt.Errorf("failed to get dispute by ID: %w", err)
	}
	return d, nil
}

func (repo *Repository) GetDisputeDetailsByID(ctx context.Context, disputeID uuid.UUID, actorUsername string,
) (models.DisputeDetails, error) {
	var d models.DisputeDetails
	err := repo.db.QueryRowContext(ctx, `
		SELECT
			d.id, d.title, d.description,
			d.created_at, d.updated_at,
			d.cryptocurrency, d.amount_nano,
			d.image_data, d.image_type,
			d.contract_address,
			d.ends_at, d.next_deadline,
			opp_user.username AS opponent,
			opp_user.photo_url,
			self.result, self.vote, self.claim
		FROM disputes d
		JOIN participants self ON self.dispute_id = d.id
		JOIN users me ON me.id = self.user_id
		JOIN participants opp ON opp.dispute_id = d.id AND opp.user_id <> self.user_id
		JOIN users opp_user ON opp_user.id = opp.user_id
		WHERE d.id = $1 AND me.username = $2
	`, disputeID, actorUsername).Scan(
		&d.ID,
		&d.Title,
		&d.Description,
		&d.CreatedAt,
		&d.UpdatedAt,
		&d.Cryptocurrency,
		&d.AmountNano,
		&d.ImageData,
		&d.ImageType,
		&d.ContractAddress,
		&d.EndsAt,
		&d.NextDeadline,
		&d.Opponent,
		&d.PhotoUrl,
		&d.Result,
		&d.Vote,
		&d.Claim,
	)
	if err != nil {
		return models.DisputeDetails{}, fmt.Errorf("failed to get dispute details by ID: %w", err)
	}
	return d, nil
}

func (repo *Repository) GetDisputeForEvidence(ctx context.Context, disputeID uuid.UUID) (models.Dispute, error) {
	var d models.Dispute
	err := repo.db.QueryRowContext(ctx, `
		SELECT 
			d.id, d.title, d.description, 
			d.image_data, d.image_type, d.ends_at, d.next_deadline
		FROM disputes d
		WHERE d.id = $1`,
		disputeID,
	).Scan(
		&d.ID,
		&d.Title,
		&d.Description,
		&d.ImageData,
		&d.ImageType,
		&d.EndsAt,
		&d.NextDeadline,
	)
	if err != nil {
		return models.Dispute{}, fmt.Errorf("failed to get dispute by ID: %w", err)
	}
	return d, nil
}

func (repo *Repository) UpdateDisputeNextDeadline(ctx context.Context, disputeID uuid.UUID, nextDeadline time.Time,
) error {
	_, err := repo.db.ExecContext(ctx, `
		UPDATE disputes
		SET next_deadline = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`, nextDeadline, disputeID)
	if err != nil {
		return fmt.Errorf("failed to update dispute next deadline: %w", err)
	}
	return nil
}
