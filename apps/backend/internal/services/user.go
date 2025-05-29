package services

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/kisnikita/safe-disputes/backend/internal/models"
	"github.com/kisnikita/safe-disputes/backend/internal/repository"
	"github.com/kisnikita/safe-disputes/backend/pkg/log"
)

type UserFinder interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (models.User, error)
	GetUserByUsername(ctx context.Context, username string) (models.User, error)
	ExistByUsername(ctx context.Context, username string) (bool, error)
	GetTotalUsers(ctx context.Context) (int, error)
	GetUsers(ctx context.Context, ids []uuid.UUID) ([]models.User, error)
	GetTopUsers(ctx context.Context, limit int) ([]models.User, error)
}

type UserCreator interface {
	InsertUser(ctx context.Context, user models.User) error
}

type UserUpdater interface {
	UpdateUser(ctx context.Context, opts models.UserUpdateOpts) error
	EarnWinnerRating(ctx context.Context, ids []uuid.UUID) error
}

type UserService struct {
	logger log.Logger

	userFinder  UserFinder
	userCreator UserCreator
	userUpdater UserUpdater
}

func NewUserService(repo *repository.Repository, log log.Logger) (UserService, error) {
	if repo == nil {
		return UserService{}, fmt.Errorf("repository is nil")
	}
	if log == nil {
		return UserService{}, fmt.Errorf("logger is nil")
	}
	return UserService{
		logger: log,

		userFinder:  repo,
		userCreator: repo,
		userUpdater: repo,
	}, nil
}

func (s UserService) GetByUsername(ctx context.Context, username string) (models.User, error) {
	user, err := s.userFinder.GetUserByUsername(ctx, username)
	switch {
	case errors.Is(err, repository.ErrNotFound):
		s.logger.Info("user not found", zap.String("username", username))
		return models.User{}, ErrUserNotFound
	case err != nil:
		s.logger.Error("failed to get user by username", zap.String("username", username), zap.Error(err))
		return models.User{}, fmt.Errorf("failed to get user by username: %w", err)
	}
	return user, nil
}

func (s UserService) CreateIfNotExist(ctx context.Context, username string) error {
	s.logger.Info("checking if user exists", zap.String("username", username))
	exist, err := s.userFinder.ExistByUsername(ctx, username)
	if err != nil {
		s.logger.Error("failed to check existence of user by username", zap.String("username", username), zap.Error(err))
		return err
	}
	if exist {
		s.logger.Info("user already exists", zap.String("username", username))
		return nil
	}
	user := models.NewUser(username)
	err = s.userCreator.InsertUser(ctx, user)
	if err != nil {
		s.logger.Error("failed to create user", zap.String("username", username), zap.Error(err))
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

func (s UserService) UpdateByUsername(ctx context.Context, opts models.UserUpdateOpts) error {
	err := s.userUpdater.UpdateUser(ctx, opts)
	if err != nil {
		s.logger.Error("failed to update user", zap.String("username", opts.Username), zap.Error(err))
		return fmt.Errorf("failed to update user: %w", err)
	}
	s.logger.Info("user updated", zap.String("username", opts.Username))
	return nil
}

func (s UserService) GetTop(ctx context.Context, limit int) ([]models.User, error) {
	users, err := s.userFinder.GetTopUsers(ctx, limit)
	if err != nil {
		s.logger.Error("failed to get top users", zap.Int("limit", limit), zap.Error(err))
		return nil, fmt.Errorf("failed to get top users: %w", err)
	}
	if len(users) == 0 {
		s.logger.Info("no users found", zap.Int("limit", limit))
		return nil, repository.ErrNotFound
	}
	s.logger.Info("top users retrieved", zap.Int("count", len(users)), zap.Int("limit", limit))
	return users, nil
}
