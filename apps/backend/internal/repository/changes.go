package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/kisnikita/safe-disputes/backend/internal/models"
)

func (repo *Repository) ListChanges(ctx context.Context, actorUsername string, since time.Time) (models.ChangesList, error) {
	res := models.ChangesList{Disputes: make([]models.DisputeChange, 0), Investigations: make([]models.InvestigationChange, 0)}

	dRows, err := repo.db.QueryContext(ctx, `
		SELECT p.dispute_id, p.status, p.updated_at
		FROM participants p
		JOIN users me ON me.id = p.user_id
		WHERE me.username = $1 AND p.updated_at > $2
		ORDER BY p.updated_at DESC
	`, actorUsername, since)
	if err != nil {
		return res, fmt.Errorf("failed to list dispute changes: %w", err)
	}
	defer dRows.Close()

	maxUpdated := since
	for dRows.Next() {
		var id string
		var status models.Status
		var updated time.Time
		if err = dRows.Scan(&id, &status, &updated); err != nil {
			return res, fmt.Errorf("failed to scan dispute change: %w", err)
		}
		if updated.After(maxUpdated) {
			maxUpdated = updated
		}
		res.Disputes = append(res.Disputes, models.DisputeChange{DisputeID: id, Status: status})
	}
	if err = dRows.Err(); err != nil {
		return res, fmt.Errorf("failed to iterate dispute changes: %w", err)
	}

	iRows, err := repo.db.QueryContext(ctx, `
		SELECT j.investigation_id, i.status, j.updated_at
		FROM jurors j
		JOIN users me ON me.id = j.user_id
		JOIN investigations i ON i.id = j.investigation_id
		WHERE me.username = $1 AND j.updated_at > $2
		ORDER BY j.updated_at DESC
	`, actorUsername, since)
	if err != nil {
		return res, fmt.Errorf("failed to list investigation changes: %w", err)
	}
	defer iRows.Close()

	for iRows.Next() {
		var id string
		var status models.InvestigationStatus
		var updated time.Time
		if err = iRows.Scan(&id, &status, &updated); err != nil {
			return res, fmt.Errorf("failed to scan investigation change: %w", err)
		}
		if updated.After(maxUpdated) {
			maxUpdated = updated
		}
		res.Investigations = append(res.Investigations, models.InvestigationChange{InvestigationID: id, Status: status})
	}
	if err = iRows.Err(); err != nil {
		return res, fmt.Errorf("failed to iterate investigation changes: %w", err)
	}

	res.MaxUpdatedAt = maxUpdated.Format(time.RFC3339Nano)
	return res, nil
}

func (repo *Repository) GetUnreadCounts(ctx context.Context, actorUsername string) (models.ChangesUnreadCounts, error) {
	res := models.ChangesUnreadCounts{}

	dRows, err := repo.db.QueryContext(ctx, `
		SELECT p.status, COUNT(*)::int AS cnt
		FROM participants p
		JOIN users me ON me.id = p.user_id
		WHERE me.username = $1
		  AND (p.seen_at IS NULL OR p.updated_at > p.seen_at)
		GROUP BY p.status
	`, actorUsername)
	if err != nil {
		return res, fmt.Errorf("failed to get dispute unread counts: %w", err)
	}
	defer dRows.Close()
	for dRows.Next() {
		var status models.Status
		var cnt int
		if err = dRows.Scan(&status, &cnt); err != nil {
			return res, fmt.Errorf("failed to scan dispute unread count: %w", err)
		}
		switch status {
		case models.DisputesStatusNew:
			res.Disputes.New = cnt
		case models.DisputesStatusCurrent:
			res.Disputes.Current = cnt
		case models.DisputesStatusPassed:
			res.Disputes.Passed = cnt
		}
	}
	if err = dRows.Err(); err != nil {
		return res, fmt.Errorf("failed to iterate dispute unread counts: %w", err)
	}

	iRows, err := repo.db.QueryContext(ctx, `
		SELECT i.status, COUNT(*)::int AS cnt
		FROM jurors j
		JOIN users me ON me.id = j.user_id
		JOIN investigations i ON i.id = j.investigation_id
		WHERE me.username = $1
		  AND (j.seen_at IS NULL OR j.updated_at > j.seen_at)
		GROUP BY i.status
	`, actorUsername)
	if err != nil {
		return res, fmt.Errorf("failed to get investigation unread counts: %w", err)
	}
	defer iRows.Close()
	for iRows.Next() {
		var status models.InvestigationStatus
		var cnt int
		if err = iRows.Scan(&status, &cnt); err != nil {
			return res, fmt.Errorf("failed to scan investigation unread count: %w", err)
		}
		switch status {
		case models.InvestigationStatusCurrent:
			res.Investigations.Current = cnt
		case models.InvestigationStatusPassed:
			res.Investigations.Passed = cnt
		}
	}
	if err = iRows.Err(); err != nil {
		return res, fmt.Errorf("failed to iterate investigation unread counts: %w", err)
	}

	return res, nil
}
