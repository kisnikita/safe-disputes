package services

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"

	"github.com/kisnikita/safe-disputes/backend/internal/models"
	"github.com/kisnikita/safe-disputes/backend/internal/repository"
	"github.com/kisnikita/safe-disputes/backend/pkg/log"
)

type UserFinder interface {
	GetByUsername(ctx context.Context, username string) (*models.User, error)
	ExistByUsername(ctx context.Context, username string) (bool, error)
}

type UserCreator interface {
	Insert(ctx context.Context, user *models.User) error
}

type UserService struct {
	logger log.Logger

	userFinder  UserFinder
	userCreator UserCreator
}

func NewUserService(repo *repository.Repository, log log.Logger) (*UserService, error) {
	if repo == nil {
		return nil, fmt.Errorf("repository is nil")
	}
	if log == nil {
		return nil, fmt.Errorf("logger is nil")
	}
	return &UserService{
		logger: log,

		userFinder:  repo,
		userCreator: repo,
	}, nil
}

func (s *UserService) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	user, err := s.userFinder.GetByUsername(ctx, username)
	switch {
	case errors.Is(err, repository.ErrNotFound):
		s.logger.Info("user not found", zap.String("username", username))
		return nil, ErrUserNotFound
	case err != nil:
		s.logger.Error("failed to get user by username", zap.String("username", username), zap.Error(err))
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}
	return user, nil
}

func (s *UserService) Insert(ctx context.Context, user *models.User) error {
	if user == nil {
		return fmt.Errorf("user is nil")
	}
	if user.Username == "" {
		return fmt.Errorf("username is empty")
	}
	if user.ID == "" {
		return fmt.Errorf("user ID is empty")
	}
	if user.CreatedAt.IsZero() {
		return fmt.Errorf("created_at is zero")
	}

	err := s.userCreator.Insert(ctx, user)
	if err != nil {
		s.logger.Error("failed to create user", zap.String("username", user.Username), zap.Error(err))
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

func (s *UserService) CreateIfNotExist(ctx context.Context, username string) error {
	exist, err := s.userFinder.ExistByUsername(ctx, username)
	if err != nil {
		s.logger.Error("failed to check existence of user by username", zap.String("username", username), zap.Error(err))
		return err
	}
	if exist {
		s.logger.Info("user already exists", zap.String("username", username))
		return nil
	}
	user := &models.User{}

}
