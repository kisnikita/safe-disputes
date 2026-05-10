package services

import (
	"context"
	"fmt"
	"time"

	"github.com/kisnikita/safe-disputes/backend/internal/models"
	"github.com/kisnikita/safe-disputes/backend/internal/repository"
	"github.com/kisnikita/safe-disputes/backend/pkg/log"
)

type ChangesReader interface {
	ListChanges(ctx context.Context, actorUsername string, since time.Time) (models.ChangesList, error)
	GetUnreadCounts(ctx context.Context, actorUsername string) (models.ChangesUnreadCounts, error)
}

type ChangesService struct {
	logger log.Logger
	reader ChangesReader
}

func NewChangesService(repo *repository.Repository, logger log.Logger) (ChangesService, error) {
	if repo == nil {
		return ChangesService{}, fmt.Errorf("repository is nil")
	}
	if logger == nil {
		return ChangesService{}, fmt.Errorf("logger is nil")
	}
	return ChangesService{logger: logger, reader: repo}, nil
}

func (s ChangesService) ListChanges(ctx context.Context, since time.Time, actorUsername string,
) (models.ChangesList, models.ChangesUnreadCounts, error) {
	changes, err := s.reader.ListChanges(ctx, actorUsername, since)
	if err != nil {
		return models.ChangesList{}, models.ChangesUnreadCounts{}, fmt.Errorf("failed to list changes: %w", err)
	}
	counts, err := s.reader.GetUnreadCounts(ctx, actorUsername)
	if err != nil {
		return models.ChangesList{}, models.ChangesUnreadCounts{}, fmt.Errorf("failed to get unread counts: %w", err)
	}
	return changes, counts, nil
}
