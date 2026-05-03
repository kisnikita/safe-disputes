package repository

import (
	"context"
	"database/sql/driver"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kisnikita/safe-disputes/backend/internal/models"
)

func TestGetDisputeByID(t *testing.T) {
	dID := uuid.New()
	actorID := uuid.New()
	now := time.Now()
	repo := newTestRepo(t, &stubDB{
		queryFn: func(string, []driver.NamedValue) (driver.Rows, error) {
			return newRows(
				[]string{"id", "title", "description", "created_at", "updated_at", "cryptocurrency", "amount_nano", "image_data", "image_type", "ends_at", "next_deadline", "contract_address"},
				[]driver.Value{dID.String(), "t", "d", now, now, "TON", int64(100_000_000_000), []byte{1}, "image/png", now.Add(2 * time.Hour), now.Add(1 * time.Hour), "addr"},
			), nil
		},
	})

	d, err := repo.GetDisputeByID(context.Background(), dID, actorID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.ID != dID || d.ContractAddress != "addr" || d.AmountNano != 100_000_000_000 {
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

	now := time.Now()
	err := repo.InsertDispute(context.Background(), models.Dispute{
		ID:              uuid.New(),
		Title:           "t",
		Description:     "d",
		CreatedAt:       now,
		UpdatedAt:       now,
		Cryptocurrency:  "TON",
		AmountNano:      1_000_000_000,
		ContractAddress: "a",
		EndsAt:          now.Add(24 * time.Hour),
		NextDeadline:    now.Add(12 * time.Hour),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if execCalls != 1 {
		t.Fatalf("expected 1 exec, got %d", execCalls)
	}
}
