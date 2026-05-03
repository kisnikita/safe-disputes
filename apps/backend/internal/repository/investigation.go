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

func (repo *Repository) ListInvestigationReads(ctx context.Context, actorUsername string, 
	opts models.InvestigationListOpts,
) ([]models.InvestigationRead, error) {
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
		clauses = append(clauses, fmt.Sprintf("i.status = $%d", idx))
		args = append(args, *opts.Status)
		idx++
	}

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

	limit := opts.Limit
	if limit <= 0 || limit > maxLimit {
		limit = maxLimit
	}
	args = append(args, limit+1)

	query := fmt.Sprintf(`
		SELECT
			i.id, i.dispute_id, i.total, i.p1, i.p2, i.draw, i.status, i.created_at, i.ends_at, i.title,
			u.result, u.vote
		FROM investigations i
		JOIN jurors u ON i.id = u.investigation_id
		JOIN users me ON me.id = u.user_id
		%s
		ORDER BY i.created_at DESC
		LIMIT $%d
	`, whereSQL, idx)

	rows, err := repo.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute ListInvestigationReads query: %w", err)
	}
	defer rows.Close()

	var investigations []models.InvestigationRead
	for rows.Next() {
		var i models.InvestigationRead
		if err := rows.Scan(
			&i.ID,
			&i.DisputeID,
			&i.Total,
			&i.P1,
			&i.P2,
			&i.Draw,
			&i.Status,
			&i.CreatedAt,
			&i.EndsAt,
			&i.Title,
			&i.Result,
			&i.Vote,
		); err != nil {
			return nil, fmt.Errorf("failed to scan investigation read row: %w", err)
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
		  i.total, i.p1, i.p2, i.draw,
		  i.status, i.created_at, i.ends_at,
		FROM investigations i
		JOIN jurors u ON i.id = u.investigation_id
		WHERE i.id = $1 AND u.user_id = $2
	`

	row := repo.db.QueryRowContext(ctx, query, invID, userID)

	var investigation models.Investigation
	if err := row.Scan(
		&investigation.ID,
		&investigation.DisputeID,
		&investigation.Title,
		&investigation.Total,
		&investigation.P1,
		&investigation.P2,
		&investigation.Draw,
		&investigation.Status,
		&investigation.CreatedAt,
		&investigation.EndsAt,
	); err != nil {
		return models.Investigation{}, fmt.Errorf("failed to scan investigation: %w", err)
	}

	return investigation, nil
}

func (repo *Repository) GetInvestigationRead(ctx context.Context, id uuid.UUID, actorUsername string,
) (models.InvestigationRead, error) {
	query := `
		SELECT
		  i.id, i.dispute_id, i.total, i.p1, i.p2, i.draw, i.status, i.created_at, i.ends_at, i.title,
		  u.result, u.vote
		FROM investigations i
		JOIN jurors u ON i.id = u.investigation_id
		JOIN users me ON me.id = u.user_id
		WHERE i.id = $1 AND me.username = $2
	`

	row := repo.db.QueryRowContext(ctx, query, id, actorUsername)

	var investigation models.InvestigationRead
	if err := row.Scan(
		&investigation.ID,
		&investigation.DisputeID,
		&investigation.Total,
		&investigation.P1,
		&investigation.P2,
		&investigation.Draw,
		&investigation.Status,
		&investigation.CreatedAt,
		&investigation.EndsAt,
		&investigation.Title,
		&investigation.Result,
		&investigation.Vote,
	); err != nil {
		return models.InvestigationRead{}, fmt.Errorf("failed to scan investigation read: %w", err)
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
