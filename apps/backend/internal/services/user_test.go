package services

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/kisnikita/safe-disputes/backend/internal/models"
	"github.com/kisnikita/safe-disputes/backend/internal/repository"
)

type fakeUserRepo struct {
	userByUsername   models.User
	usersTop         []models.User
	errByUsername    error
	errExist         error
	errInsert        error
	errUpdate        error
	errGetTop        error
	exists           bool
	inserted         bool
	updated          bool
	insertedUser     models.User
	updatedOpts      models.UserUpdateOpts
	gotTopLimit      int
	getByUsernameCnt int
}

func (f *fakeUserRepo) GetUserByID(context.Context, uuid.UUID) (models.User, error) {
	return models.User{}, nil
}
func (f *fakeUserRepo) GetUserByUsername(context.Context, string) (models.User, error) {
	f.getByUsernameCnt++
	if f.errByUsername != nil {
		return models.User{}, f.errByUsername
	}
	return f.userByUsername, nil
}
func (f *fakeUserRepo) ExistByUsername(context.Context, string) (bool, error) {
	if f.errExist != nil {
		return false, f.errExist
	}
	return f.exists, nil
}
func (f *fakeUserRepo) GetTotalUsers(context.Context) (int, error)                   { return 0, nil }
func (f *fakeUserRepo) GetUsers(context.Context, []uuid.UUID) ([]models.User, error) { return nil, nil }
func (f *fakeUserRepo) GetTopUsers(_ context.Context, limit int) ([]models.User, error) {
	f.gotTopLimit = limit
	if f.errGetTop != nil {
		return nil, f.errGetTop
	}
	return f.usersTop, nil
}
func (f *fakeUserRepo) InsertUser(_ context.Context, user models.User) error {
	if f.errInsert != nil {
		return f.errInsert
	}
	f.inserted = true
	f.insertedUser = user
	return nil
}
func (f *fakeUserRepo) UpdateUser(_ context.Context, opts models.UserUpdateOpts) error {
	if f.errUpdate != nil {
		return f.errUpdate
	}
	f.updated = true
	f.updatedOpts = opts
	return nil
}
func (f *fakeUserRepo) UpdateUserPhotoURL(_ context.Context, _ string, _ *string) error { return nil }
func (f *fakeUserRepo) EarnWinnerRating(context.Context, []uuid.UUID) error             { return nil }

func TestUserServiceGetByUsername(t *testing.T) {
	svc := UserService{logger: noopLogger{}, userFinder: &fakeUserRepo{errByUsername: repository.ErrNotFound}}

	_, err := svc.GetByUsername(context.Background(), "alice")
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestUserServiceCreateIfNotExist(t *testing.T) {
	t.Run("does not create existing user", func(t *testing.T) {
		repo := &fakeUserRepo{exists: true}
		svc := UserService{logger: noopLogger{}, userFinder: repo, userCreator: repo, userUpdater: repo}

		if err := svc.CreateIfNotExist(context.Background(), "alice", nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo.inserted {
			t.Fatal("expected no insert")
		}
	})

	t.Run("creates missing user", func(t *testing.T) {
		repo := &fakeUserRepo{}
		svc := UserService{logger: noopLogger{}, userFinder: repo, userCreator: repo, userUpdater: repo}

		if err := svc.CreateIfNotExist(context.Background(), "alice", nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !repo.inserted {
			t.Fatal("expected insert")
		}
		if repo.insertedUser.Username != "alice" {
			t.Fatalf("expected username alice, got %q", repo.insertedUser.Username)
		}
	})
}

func TestUserServiceUpdateAndTop(t *testing.T) {
	repo := &fakeUserRepo{usersTop: []models.User{{Username: "alice"}}}
	svc := UserService{logger: noopLogger{}, userFinder: repo, userUpdater: repo}

	rating := 10
	err := svc.UpdateByUsername(context.Background(), models.UserUpdateOpts{Username: "alice", Rating: &rating})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.updated {
		t.Fatal("expected update call")
	}

	users, err := svc.GetTop(context.Background(), 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(users) != 1 || users[0].Username != "alice" {
		t.Fatalf("unexpected top users: %#v", users)
	}
	if repo.gotTopLimit != 5 {
		t.Fatalf("expected limit 5, got %d", repo.gotTopLimit)
	}
}
