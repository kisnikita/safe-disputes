package repository

import (
	"context"
	"database/sql/driver"
	"testing"

	"github.com/google/uuid"
	"github.com/kisnikita/safe-disputes/backend/internal/models"
)

func TestIsFirstEvidence(t *testing.T) {
	repo := newTestRepo(t, &stubDB{
		queryFn: func(string, []driver.NamedValue) (driver.Rows, error) {
			return newRows([]string{"count"}, []driver.Value{0}), nil
		},
	})

	isFirst, err := repo.IsFirstEvidence(context.Background(), uuid.NewString())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !isFirst {
		t.Fatal("expected true")
	}
}

func TestGetEvidences(t *testing.T) {
	dID := uuid.New()
	repo := newTestRepo(t, &stubDB{
		queryFn: func(string, []driver.NamedValue) (driver.Rows, error) {
			return newRows(
				[]string{"id", "user_id", "dispute_id", "description", "image_data", "image_type"},
				[]driver.Value{uuid.NewString(), uuid.NewString(), dID.String(), "one", []byte{1}, "image/png"},
				[]driver.Value{uuid.NewString(), uuid.NewString(), dID.String(), "two", []byte{2}, "image/jpeg"},
			), nil
		},
	})

	evidences, err := repo.GetEvidences(context.Background(), dID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(evidences) != 2 || evidences[0].Description != "one" {
		t.Fatalf("unexpected evidences: %#v", evidences)
	}
}

func TestInsertEvidence(t *testing.T) {
	execCalls := 0
	repo := newTestRepo(t, &stubDB{
		execFn: func(string, []driver.NamedValue) (driver.Result, error) {
			execCalls++
			return driver.RowsAffected(1), nil
		},
	})

	err := repo.InsertEvidence(context.Background(), models.Evidence{ID: uuid.New(), UserID: uuid.New(), DisputeID: uuid.New(), Description: "x"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if execCalls != 1 {
		t.Fatalf("expected 1 exec, got %d", execCalls)
	}
}
