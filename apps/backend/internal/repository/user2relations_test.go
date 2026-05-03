package repository

import (
	"context"
	"database/sql/driver"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/kisnikita/safe-disputes/backend/internal/models"
)

func TestGetOpponentID(t *testing.T) {
	opID := uuid.New()
	repo := newTestRepo(t, &stubDB{
		queryFn: func(string, []driver.NamedValue) (driver.Rows, error) {
			return newRows([]string{"user_id"}, []driver.Value{opID.String()}), nil
		},
	})

	got, err := repo.GetOpponentID(context.Background(), uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != opID {
		t.Fatalf("expected %s, got %s", opID, got)
	}
}

func TestUpdateParticipant(t *testing.T) {
	execCalls := 0
	repo := newTestRepo(t, &stubDB{
		execFn: func(string, []driver.NamedValue) (driver.Result, error) {
			execCalls++
			return driver.RowsAffected(1), nil
		},
	})

	err := repo.UpdateParticipant(context.Background(), models.ParticipantUpdateOpts{ID: uuid.New()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if execCalls != 1 {
		t.Fatalf("expected 1 exec, got %d", execCalls)
	}
}

func TestBroadcastInvestigationSkipsDisputeUsers(t *testing.T) {
	p1 := uuid.New()
	p2 := uuid.New()
	eligible := uuid.New()
	execCalls := 0
	queryChecked := false
	repo := newTestRepo(t, &stubDB{
		queryFn: func(query string, _ []driver.NamedValue) (driver.Rows, error) {
			if !strings.Contains(query, "investigation_readiness = TRUE") {
				t.Fatalf("expected investigation readiness filter in query: %s", query)
			}
			queryChecked = true
			return newRows(
				[]string{"id"},
				[]driver.Value{p1.String()},
				[]driver.Value{p2.String()},
				[]driver.Value{eligible.String()},
			), nil
		},
		execFn: func(string, []driver.NamedValue) (driver.Result, error) {
			execCalls++
			return driver.RowsAffected(1), nil
		},
	})

	ids, err := repo.BroadcastInvestigation(context.Background(), models.NewJuror(uuid.New(), uuid.Nil), p1, p2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 1 || ids[0] != eligible {
		t.Fatalf("unexpected ids: %#v", ids)
	}
	if !queryChecked {
		t.Fatal("expected query check to run")
	}
	if execCalls != 2 {
		t.Fatalf("expected 2 exec calls (insert + total update), got %d", execCalls)
	}
}

func TestUpdateJurorAndDelete(t *testing.T) {
	execCalls := 0
	repo := newTestRepo(t, &stubDB{
		execFn: func(string, []driver.NamedValue) (driver.Result, error) {
			execCalls++
			return driver.RowsAffected(1), nil
		},
	})

	vote := "p1"
	res := models.InvestigationResultSent
	err := repo.UpdateJuror(context.Background(), models.JurorUpdateOpts{ID: uuid.New(), Vote: &vote, Result: &res})
	if err != nil {
		t.Fatalf("unexpected update error: %v", err)
	}
	err = repo.DeleteUsersWithoutVote(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected delete error: %v", err)
	}
	if execCalls != 2 {
		t.Fatalf("expected 2 execs, got %d", execCalls)
	}
}
