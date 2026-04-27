package repository

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestGetUserByUsernameNotFound(t *testing.T) {
	repo := newTestRepo(t, &stubDB{
		queryFn: func(string, []driver.NamedValue) (driver.Rows, error) {
			return newRows([]string{"id", "username", "photo_url", "created_at", "notification_enabled", "dispute_readiness", "minimum_dispute_amount_nano", "rating", "chat_id"}), nil
		},
	})

	_, err := repo.GetUserByUsername(context.Background(), "alice")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGetUserByUsernameSuccess(t *testing.T) {
	id := uuid.New()
	now := time.Now()
	repo := newTestRepo(t, &stubDB{
		queryFn: func(string, []driver.NamedValue) (driver.Rows, error) {
			return newRows(
				[]string{"id", "username", "photo_url", "created_at", "notification_enabled", "dispute_readiness", "minimum_dispute_amount_nano", "rating", "chat_id"},
				[]driver.Value{id.String(), "alice", "https://t.me/i/userpic/320/x.png", now, true, true, int64(100_000_000_000), 5, int64(123)},
			), nil
		},
	})

	user, err := repo.GetUserByUsername(context.Background(), "alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.ID != id || user.Username != "alice" || user.ChatID != 123 {
		t.Fatalf("unexpected user: %#v", user)
	}
	if user.PhotoUrl == nil || *user.PhotoUrl == "" {
		t.Fatalf("expected photo url to be set: %#v", user)
	}
}

func TestExistByUsername(t *testing.T) {
	repo := newTestRepo(t, &stubDB{
		queryFn: func(string, []driver.NamedValue) (driver.Rows, error) {
			return newRows([]string{"exists"}, []driver.Value{true}), nil
		},
	})

	exists, err := repo.ExistByUsername(context.Background(), "alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Fatal("expected true")
	}
}

func TestGetTopUsers(t *testing.T) {
	repo := newTestRepo(t, &stubDB{
		queryFn: func(string, []driver.NamedValue) (driver.Rows, error) {
			return newRows([]string{"username", "rating"}, []driver.Value{"alice", 10}, []driver.Value{"bob", 7}), nil
		},
	})

	users, err := repo.GetTopUsers(context.Background(), 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(users) != 2 || users[0].Username != "alice" || users[1].Rating != 7 {
		t.Fatalf("unexpected users: %#v", users)
	}
}

func TestHandleNotFoundError(t *testing.T) {
	err := handleNotFoundError(sql.ErrNoRows)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
