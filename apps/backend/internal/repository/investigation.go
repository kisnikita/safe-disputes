package repository

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/kisnikita/safe-disputes/backend/internal/models"
	"strings"
	"time"
)

func (repo *Repository) InsertInvestigation(ctx context.Context, investigation models.Investigation) error {
	_, err := repo.db.ExecContext(ctx, `
	INSERT INTO investigations (id, dispute_id, total, p1, p2, draw, status, ends_at, title) 
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		investigation.ID,
		investigation.DisputeID,
		investigation.Total,
		investigation.P1,
		investigation.P2,
		investigation.Draw,
		investigation.Status,
		investigation.EndsAt,
		investigation.Title,
	)
	if err != nil {
		return fmt.Errorf("failed to insert investigation: %w", err)
	}
	return nil
}

func (repo *Repository) ListInvestigations(
	ctx context.Context,
	opts models.InvestigationListOpts,
) ([]models.Investigation, error) {
	const maxLimit = 100

	// --- Dynamic WHERE clauses and parameters ---
	var (
		clauses []string
		args    []interface{}
		idx     = 1
	)

	// Filter by creator (user_id in user2investigation)
	clauses = append(clauses, fmt.Sprintf("u.user_id = $%d", idx))
	args = append(args, opts.UserID)
	idx++

	// Filter by status (column u.status)
	if opts.Status != nil {
		clauses = append(clauses, fmt.Sprintf("i.status = $%d", idx))
		args = append(args, *opts.Status)
		idx++
	}

	// Cursor-based pagination: fetch only older records by created_at
	if opts.Cursor != "" {
		t, err := time.Parse(time.RFC3339Nano, opts.Cursor)
		if err != nil {
			return nil, fmt.Errorf("invalid cursor format: %w", err)
		}
		clauses = append(clauses, fmt.Sprintf("i.created_at <= $%d", idx))
		args = append(args, t)
		idx++
	}

	whereSQL := ""
	if len(clauses) > 0 {
		whereSQL = "WHERE " + strings.Join(clauses, " AND ")
	}

	// Limit +1 for nextCursor calculation
	limit := opts.Limit
	if limit <= 0 || limit > maxLimit {
		limit = maxLimit
	}
	args = append(args, limit+1)

	// --- Build final SQL: JOIN INVESTIGATIONS i and USER2INVESTIGATION u ---
	query := fmt.Sprintf(`
        SELECT
          i.id, i.dispute_id, i.title,
          i.status,  i.ends_at,
        u.result, u.vote
        FROM investigations i
        JOIN user2investigation u ON i.id = u.investigation_id
        %s
        ORDER BY i.created_at DESC
        LIMIT $%d
    `, whereSQL, idx)

	rows, err := repo.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute ListInvestigations query: %w", err)
	}
	defer rows.Close()

	var investigations []models.Investigation
	for rows.Next() {
		var i models.Investigation
		if err := rows.Scan(
			&i.ID,
			&i.DisputeID,
			&i.Title,
			&i.Status,
			&i.EndsAt,
			&i.Result,
			&i.Vote,
		); err != nil {
			return nil, fmt.Errorf("failed to scan investigation row: %w", err)
		}
		investigations = append(investigations, i)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over rows: %w", err)
	}

	return investigations, nil
}

func (repo *Repository) GetInvestigation(ctx context.Context, invID, userID uuid.UUID) (models.Investigation, error) {
	query := `
		SELECT
		  i.id, i.dispute_id, i.title,
		  i.status,  i.ends_at,
		  u.result, u.vote
		FROM investigations i
		JOIN user2investigation u ON i.id = u.investigation_id
		WHERE i.id = $1 AND u.user_id = $2
	`

	row := repo.db.QueryRowContext(ctx, query, invID, userID)

	var investigation models.Investigation
	if err := row.Scan(
		&investigation.ID,
		&investigation.DisputeID,
		&investigation.Title,
		&investigation.Status,
		&investigation.EndsAt,
		&investigation.Result,
		&investigation.Vote,
	); err != nil {
		return models.Investigation{}, fmt.Errorf("failed to scan investigation: %w", err)
	}

	return investigation, nil
}

func (repo *Repository) UpdateInvestigation(ctx context.Context, opts models.InvestigationUpdateOpts) error {
	query := `
		UPDATE investigations
		SET status = COALESCE($1, status),
			p1 = COALESCE($2, p1),
			p2 = COALESCE($3, p2),
			draw = COALESCE($4, draw),
			total = COALESCE($5, total)
		WHERE id = $6
	`
	_, err := repo.db.ExecContext(ctx, query, opts.Status, opts.P1, opts.P2, opts.Draw, opts.Total, opts.ID)
	if err != nil {
		return fmt.Errorf("failed to update investigation: %w", err)
	}
	return nil
}
