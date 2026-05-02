package repository

import (
	"context"
	"database/sql/driver"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kisnikita/safe-disputes/backend/internal/models"
)

func TestListInvestigationsInvalidCursor(t *testing.T) {
	repo := newTestRepo(t, &stubDB{})
	status := models.InvestigationStatusCurrent
	_, err := repo.ListInvestigations(context.Background(), models.InvestigationListOpts{Status: &status, Cursor: "bad"})
	if err == nil || !strings.Contains(err.Error(), "invalid cursor format") {
		t.Fatalf("expected cursor error, got %v", err)
	}
}

func TestGetInvestigation(t *testing.T) {
	invID := uuid.New()
	disputeID := uuid.New()
	now := time.Now()
	repo := newTestRepo(t, &stubDB{
		queryFn: func(string, []driver.NamedValue) (driver.Rows, error) {
			return newRows(
				[]string{"id", "dispute_id", "title", "total", "p1", "p2", "draw", "status", "created_at", "ends_at", "result", "vote"},
				[]driver.Value{
					invID.String(),
					disputeID.String(),
					"INV",
					int64(1),
					int64(1),
					int64(0),
					int64(0),
					string(models.InvestigationStatusCurrent),
					now,
					now,
					string(models.InvestigationResultSent),
					"p1",
				},
			), nil
		},
	})

	inv, err := repo.GetInvestigation(context.Background(), invID, uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inv.ID != invID || inv.Title != "INV" {
		t.Fatalf("unexpected investigation: %#v", inv)
	}
}

func TestUpdateWinnersResult(t *testing.T) {
	execCalls := 0
	repo := newTestRepo(t, &stubDB{
		execFn: func(string, []driver.NamedValue) (driver.Result, error) {
			execCalls++
			return driver.RowsAffected(1), nil
		},
	})

	err := repo.UpdateWinnersResult(context.Background(), uuid.New(), []uuid.UUID{uuid.New()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if execCalls != 2 {
		t.Fatalf("expected 2 exec calls, got %d", execCalls)
	}
}

func TestGetDisputesUsers(t *testing.T) {
	repo := newTestRepo(t, &stubDB{
		queryFn: func(string, []driver.NamedValue) (driver.Rows, error) {
			return newRows(
				[]string{"id", "username", "chat_id", "notification_enabled", "rating"},
				[]driver.Value{uuid.NewString(), "alice", int64(10), true, 5},
				[]driver.Value{uuid.NewString(), "bob", int64(20), false, 7},
			), nil
		},
	})

	users, err := repo.GetDisputesUsers(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(users) != 2 || users[0].Username != "alice" || users[1].Rating != 7 {
		t.Fatalf("unexpected users: %#v", users)
	}
}
