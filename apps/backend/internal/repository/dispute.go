package repository

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/kisnikita/safe-disputes/backend/internal/models"
	"strings"
	"time"
)

func (repo *Repository) InsertDispute(ctx context.Context, dispute models.Dispute) error {
	_, err := repo.db.ExecContext(ctx, `
	INSERT INTO disputes (id, title, description, created_at, updated_at, cryptocurrency, amount, image_data, image_type) 
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		dispute.ID,
		dispute.Title,
		dispute.Description,
		dispute.CreatedAt,
		dispute.UpdatedAt,
		dispute.Cryptocurrency,
		dispute.Amount,
		dispute.ImageData,
		dispute.ImageType,
	)
	if err != nil {
		return fmt.Errorf("failed to insert dispute: %w", err)
	}
	return nil
}

func (repo *Repository) ListDisputes(
	ctx context.Context,
	opts models.DisputeListOpts,
) ([]models.Dispute, error) {
	const maxLimit = 100

	// --- динамические WHERE-клаузулы и параметры ---
	var (
		clauses []string
		args    []interface{}
		idx     = 1
	)

	// Обязательно фильтруем по creator (user_id в user2dispute)
	clauses = append(clauses, fmt.Sprintf("u.user_id = $%d", idx))
	args = append(args, opts.Creator)
	idx++

	// Фильтрация по статусу (столбец u.status)
	if opts.Status != nil {
		clauses = append(clauses, fmt.Sprintf("u.status = $%d", idx))
		args = append(args, *opts.Status)
		idx++
	}

	// Фильтрация по результату (столбец u.result)
	if opts.Result != nil {
		clauses = append(clauses, fmt.Sprintf("u.result = $%d", idx))
		args = append(args, *opts.Result)
		idx++
	}

	// Cursor-based pagination: берем только более старые записи по created_at
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

	// Лимит +1 для расчёта nextCursor
	limit := opts.Limit
	if limit <= 0 || limit > maxLimit {
		limit = maxLimit
	}
	args = append(args, limit+1)

	// --- Собираем итоговый SQL: JOIN DISPUTES d и USER2DISPUTE u ---
	query := fmt.Sprintf(`
        SELECT
          d.id, d.title, d.description,
        d.created_at, d.updated_at,
          d.cryptocurrency, d.amount,
          d.image_data, d.image_type,
          u.result, u.claim
        FROM disputes d
        JOIN user2dispute u ON d.id = u.dispute_id
        %s
        ORDER BY d.created_at DESC
        LIMIT $%d
    `, whereSQL, idx)

	rows, err := repo.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute ListDisputes query: %w", err)
	}
	defer rows.Close()

	var disputes []models.Dispute
	for rows.Next() {
		var d models.Dispute
		if err := rows.Scan(
			&d.ID,
			&d.Title,
			&d.Description,
			&d.CreatedAt,
			&d.UpdatedAt,
			&d.Cryptocurrency,
			&d.Amount,
			&d.ImageData,
			&d.ImageType,
			&d.Result,
			&d.Claim,
		); err != nil {
			return nil, fmt.Errorf("failed to scan dispute row: %w", err)
		}
		disputes = append(disputes, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over rows: %w", err)
	}

	return disputes, nil
}

func (repo *Repository) GetDisputeByID(ctx context.Context, disputeID uuid.UUID, creatorID uuid.UUID) (models.Dispute, error) {
	var d models.Dispute
	err := repo.db.QueryRowContext(ctx, `
		SELECT 
			d.id, d.title, d.description, 
			d.created_at, d.updated_at, 
			d.cryptocurrency, d.amount, 
			d.image_data, d.image_type,
			u.result, u.claim, u.vote
		FROM disputes d
		JOIN user2dispute u ON d.id = u.dispute_id
		WHERE d.id = $1 AND u.user_id = $2`,
		disputeID, creatorID,
	).Scan(
		&d.ID,
		&d.Title,
		&d.Description,
		&d.CreatedAt,
		&d.UpdatedAt,
		&d.Cryptocurrency,
		&d.Amount,
		&d.ImageData,
		&d.ImageType,
		&d.Result,
		&d.Claim,
		&d.Vote,
	)
	if err != nil {
		return models.Dispute{}, fmt.Errorf("failed to get dispute by ID: %w", err)
	}
	return d, nil
}

func (repo *Repository) GetDisputeForEvidence(ctx context.Context, disputeID uuid.UUID) (models.Dispute, error) {
	var d models.Dispute
	err := repo.db.QueryRowContext(ctx, `
		SELECT 
			d.id, d.title, d.description, 
			d.image_data, d.image_type
		FROM disputes d
		WHERE d.id = $1`,
		disputeID,
	).Scan(
		&d.ID,
		&d.Title,
		&d.Description,
		&d.ImageData,
		&d.ImageType,
	)
	if err != nil {
		return models.Dispute{}, fmt.Errorf("failed to get dispute by ID: %w", err)
	}
	return d, nil
}
