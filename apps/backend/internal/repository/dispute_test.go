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

func TestListDisputesInvalidCursor(t *testing.T) {
	repo := newTestRepo(t, &stubDB{})
	status := models.DisputesStatusCurrent
	_, err := repo.ListDisputes(context.Background(), models.DisputeListOpts{Status: &status, Cursor: "bad"})
	if err == nil || !strings.Contains(err.Error(), "invalid cursor format") {
		t.Fatalf("expected cursor error, got %v", err)
	}
}

func TestGetDisputeByID(t *testing.T) {
	dID := uuid.New()
	creatorID := uuid.New()
	now := time.Now()
	repo := newTestRepo(t, &stubDB{
		queryFn: func(string, []driver.NamedValue) (driver.Rows, error) {
			return newRows(
				[]string{"id", "title", "description", "created_at", "updated_at", "cryptocurrency", "amount", "image_data", "image_type", "result", "claim", "vote", "contract_address"},
				[]driver.Value{dID.String(), "t", "d", now, now, "TON", 100, []byte{1}, "image/png", string(models.DisputesResultWin), true, true, "addr"},
			), nil
		},
	})

	d, err := repo.GetDisputeByID(context.Background(), dID, creatorID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.ID != dID || d.ContractAddress != "addr" || d.Amount != 100 {
		t.Fatalf("unexpected dispute: %#v", d)
	}
}

func TestInsertDispute(t *testing.T) {
	execCalls := 0
	repo := newTestRepo(t, &stubDB{
		execFn: func(string, []driver.NamedValue) (driver.Result, error) {
			execCalls++
			return driver.RowsAffected(1), nil
		},
	})

	err := repo.InsertDispute(context.Background(), models.Dispute{DisputeDB: models.DisputeDB{ID: uuid.New(), Title: "t", Description: "d", CreatedAt: time.Now(), UpdatedAt: time.Now(), Cryptocurrency: "TON", Amount: 1, ContractAddress: "a"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if execCalls != 1 {
		t.Fatalf("expected 1 exec, got %d", execCalls)
	}
}
