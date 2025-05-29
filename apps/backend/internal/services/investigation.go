package services

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/kisnikita/safe-disputes/backend/internal/models"
	"github.com/kisnikita/safe-disputes/backend/internal/repository"
	"github.com/kisnikita/safe-disputes/backend/pkg/log"
	"go.uber.org/zap"
)

type InvestigationCreator interface {
	InsertInvestigation(ctx context.Context, investigation models.Investigation) error
}

type InvestigationFinder interface {
	ListInvestigations(ctx context.Context, opts models.InvestigationListOpts) ([]models.Investigation, error)
	GetInvestigation(ctx context.Context, invID, userID uuid.UUID) (models.Investigation, error)
}

type InvestigationUpdater interface {
	UpdateInvestigation(ctx context.Context, opts models.InvestigationUpdateOpts) error
}

type InvestigationDeleter interface {
	DeleteUsersWithoutVote(ctx context.Context, invID uuid.UUID) error
}

type U2IFinder interface {
	GetUser2Investigation(ctx context.Context, invID, userID uuid.UUID) (models.User2Investigation, error)
	GetWinnersIDs(ctx context.Context, invID uuid.UUID, winner string) ([]uuid.UUID, error)
	GetDisputesUsers(ctx context.Context, invID uuid.UUID) ([]models.User, error)
}

type U2IUpdater interface {
	UpdateUser2Investigation(ctx context.Context, opts models.U2IUpdateOpts) error
	UpdateWinnersResult(ctx context.Context, invID uuid.UUID, ids []uuid.UUID) error
}

type InvestigationService struct {
	logger               log.Logger
	investigationCreator InvestigationCreator
	investigationFinder  InvestigationFinder
	investigationUpdater InvestigationUpdater
	investigationDeleter InvestigationDeleter
	userFinder           UserFinder
	userUpdater          UserUpdater
	u2dUpdater           User2DisputeUpdater
	u2dGetter            User2DisputeGetter
	u2iFinder            U2IFinder
	u2iUpdater           U2IUpdater
	disputeFinder        DisputeFinder
	msgSender            MessageSender
}

func NewInvestigationService(repo *repository.Repository, log log.Logger, msgSender MessageSender,
) (InvestigationService, error) {
	if repo == nil {
		return InvestigationService{}, fmt.Errorf("repository is nil")
	}
	if log == nil {
		return InvestigationService{}, fmt.Errorf("logger is nil")
	}

	return InvestigationService{
		logger:               log,
		investigationCreator: repo,
		investigationFinder:  repo,
		investigationUpdater: repo,
		investigationDeleter: repo,
		userFinder:           repo,
		userUpdater:          repo,
		u2dUpdater:           repo,
		u2dGetter:            repo,
		u2iFinder:            repo,
		u2iUpdater:           repo,
		disputeFinder:        repo,
		msgSender:            msgSender,
	}, nil
}

func (s InvestigationService) ListInvestigation(ctx context.Context, opts models.InvestigationListOpts, username string,
) ([]models.Investigation, error) {

	user, err := s.userFinder.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("failed to get creator user: %w", err)
	}

	opts.UserID = user.ID

	investigations, err := s.investigationFinder.ListInvestigations(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list investigations: %w", err)
	}

	if len(investigations) == 0 {
		s.logger.Info("no investigations found", zap.String("creator", user.Username))
		return []models.Investigation{}, nil
	}

	return investigations, nil
}

func (s InvestigationService) GetInvestigation(ctx context.Context, id, username string) (models.Investigation, error) {
	user, err := s.userFinder.GetUserByUsername(ctx, username)
	if err != nil {
		return models.Investigation{}, fmt.Errorf("failed to get creator user: %w", err)
	}

	invUUID, err := uuid.Parse(id)
	if err != nil {
		return models.Investigation{}, fmt.Errorf("invalid investigation ID format: %w", err)
	}

	investigation, err := s.investigationFinder.GetInvestigation(ctx, invUUID, user.ID)
	if err != nil {
		return models.Investigation{}, fmt.Errorf("failed to get investigation: %w", err)
	}

	return investigation, nil
}

func (s InvestigationService) VoteInvestigation(ctx context.Context, id, username, vote string) error {
	user, err := s.userFinder.GetUserByUsername(ctx, username)
	if err != nil {
		return fmt.Errorf("failed to get user by username: %w", err)
	}

	invUUID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid investigation ID format: %w", err)
	}

	u2i, err := s.u2iFinder.GetUser2Investigation(ctx, invUUID, user.ID)
	if err != nil {
		return fmt.Errorf("failed to get investigation: %w", err)
	}

	result := models.InvestigationResultSent
	opts := models.U2IUpdateOpts{
		ID: u2i.ID, Vote: &vote, Result: &result,
	}
	err = s.u2iUpdater.UpdateUser2Investigation(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to update user2investigation: %w", err)
	}
	rating := user.Rating + 1
	usrUpdOpts := models.UserUpdateOpts{
		Username: user.Username, Rating: &rating,
	}
	err = s.userUpdater.UpdateUser(ctx, usrUpdOpts)
	if err != nil {
		return fmt.Errorf("failed to update user rating: %w", err)
	}

	investigation, err := s.investigationFinder.GetInvestigation(ctx, invUUID, user.ID)
	if err != nil {
		return fmt.Errorf("failed to get investigation: %w", err)
	}

	if vote == "p1" {
		investigation.P1 += 1
	} else if vote == "p2" {
		investigation.P2 += 1
	} else {
		investigation.Draw += 1
	}

	invUpdateOpts := models.InvestigationUpdateOpts{
		ID:   investigation.ID,
		P1:   &investigation.P1,
		P2:   &investigation.P2,
		Draw: &investigation.Draw,
	}
	if err = s.investigationUpdater.UpdateInvestigation(ctx, invUpdateOpts); err != nil {
		return fmt.Errorf("failed to update investigation: %w", err)
	}
	if investigation.P1+investigation.P2+investigation.Draw != investigation.Total {
		return nil
	}

	s.logger.Info("investigation vote added", zap.String("investigation_id", id), zap.String("username", username))

	st := models.InvestigationStatusPassed
	invUpdateOpts.Status = &st
	if err = s.investigationUpdater.UpdateInvestigation(ctx, invUpdateOpts); err != nil {
		return fmt.Errorf("failed to update investigation: %w", err)
	}

	if err = s.investigationDeleter.DeleteUsersWithoutVote(ctx, investigation.ID); err != nil {
		return fmt.Errorf("failed to delete users without vote: %w", err)
	}

	var res string
	if investigation.P1 > investigation.P2 && investigation.P1 > investigation.Draw {
		res = "p1"
	} else if investigation.P2 > investigation.P1 && investigation.P2 > investigation.Draw {
		res = "p2"
	} else {
		res = "draw"
	}

	winnerIDs, err := s.u2iFinder.GetWinnersIDs(ctx, investigation.ID, res)
	if err != nil {
		return fmt.Errorf("failed to get winners IDs: %w", err)
	}
	if err = s.userUpdater.EarnWinnerRating(ctx, winnerIDs); err != nil {
		return fmt.Errorf("failed to update winner rating: %w", err)
	}
	if err = s.u2iUpdater.UpdateWinnersResult(ctx, investigation.ID, winnerIDs); err != nil {
		return fmt.Errorf("failed to update winners result: %w", err)
	}
	users, err := s.u2iFinder.GetDisputesUsers(ctx, investigation.ID)
	if err != nil {
		return fmt.Errorf("failed to get users from investigation: %w", err)
	}

	u2dP1, err := s.u2dGetter.GetUser2Dispute(ctx, investigation.DisputeID, users[0].ID)
	if err != nil {
		return fmt.Errorf("failed to get user2dispute for user1: %w", err)
	}
	u2dP2, err := s.u2dGetter.GetUser2Dispute(ctx, investigation.DisputeID, users[1].ID)
	if err != nil {
		return fmt.Errorf("failed to get user2dispute for user2: %w", err)
	}

	dispute, err := s.disputeFinder.GetDisputeByID(ctx, investigation.DisputeID, users[0].ID)
	if err != nil {
		return fmt.Errorf("failed to get dispute by ID: %w", err)
	}
	if res == "draw" {
		result := models.DisputesResultDraw
		status := models.DisputesStatusPassed
		tr := true

		u2dUpdateOpts := models.U2DUpdateOpts{
			ID:     u2dP1.ID,
			Status: &status,
			Result: &result,
			Claim:  &tr,
		}
		if err = s.u2dUpdater.UpdateUser2Dispute(ctx, u2dUpdateOpts); err != nil {
			return fmt.Errorf("failed to update user2dispute: %w", err)
		}
		u2dUpdateOpts.ID = u2dP2.ID
		if err = s.u2dUpdater.UpdateUser2Dispute(ctx, u2dUpdateOpts); err != nil {
			return fmt.Errorf("failed to update user2dispute: %w", err)
		}
		if users[0].NotificationEnabled {
			msg := fmt.Sprintf("Расследование %s завершилось ничьей, вы можете забрать свою ставку!", dispute.Title)
			if err = s.msgSender.SendMessage(users[0].ChatID, msg); err != nil {
				return fmt.Errorf("failed to send message to user: %w", err)
			}
		}
		if users[1].NotificationEnabled {
			msg := fmt.Sprintf("Расследование %s завершилось ничьей, вы можете забрать свою ставку!", dispute.Title)
			if err = s.msgSender.SendMessage(users[1].ChatID, msg); err != nil {
				return fmt.Errorf("failed to send message to user: %w", err)
			}
		}
		return nil
	}

	if res == "p1" {
		result := models.DisputesResultWin
		status := models.DisputesStatusPassed
		tr := true

		u2dUpdateOpts := models.U2DUpdateOpts{
			ID:     u2dP1.ID,
			Status: &status,
			Result: &result,
			Claim:  &tr,
		}
		if err = s.u2dUpdater.UpdateUser2Dispute(ctx, u2dUpdateOpts); err != nil {
			return fmt.Errorf("failed to update user2dispute: %w", err)
		}
		result = models.DisputesResultLose
		fl := false
		u2dUpdateOpts.ID = u2dP2.ID
		u2dUpdateOpts.Result = &result
		u2dUpdateOpts.Claim = &fl
		if err = s.u2dUpdater.UpdateUser2Dispute(ctx, u2dUpdateOpts); err != nil {
			return fmt.Errorf("failed to update user2dispute: %w", err)
		}
		if users[0].NotificationEnabled {
			msg := fmt.Sprintf("Расследование %s завершилось победой, вы можете забрать свою ставку!", dispute.Title)
			if err = s.msgSender.SendMessage(users[0].ChatID, msg); err != nil {
				return fmt.Errorf("failed to send message to user: %w", err)
			}
		}
		return nil
	}

	r := models.DisputesResultWin
	status := models.DisputesStatusPassed
	tr := true

	u2dUpdateOpts := models.U2DUpdateOpts{
		ID:     u2dP2.ID,
		Status: &status,
		Result: &r,
		Claim:  &tr,
	}
	if err = s.u2dUpdater.UpdateUser2Dispute(ctx, u2dUpdateOpts); err != nil {
		return fmt.Errorf("failed to update user2dispute: %w", err)
	}
	r = models.DisputesResultLose
	fl := false
	u2dUpdateOpts.ID = u2dP1.ID
	u2dUpdateOpts.Result = &r
	u2dUpdateOpts.Claim = &fl
	if err = s.u2dUpdater.UpdateUser2Dispute(ctx, u2dUpdateOpts); err != nil {
		return fmt.Errorf("failed to update user2dispute: %w", err)
	}
	if users[1].NotificationEnabled {
		msg := fmt.Sprintf("Расследование %s завершилось победой, вы можете забрать свою ставку!", dispute.Title)
		if err = s.msgSender.SendMessage(users[1].ChatID, msg); err != nil {
			return fmt.Errorf("failed to send message to user: %w", err)
		}
	}

	for _, id := range winnerIDs {
		user, err := s.userFinder.GetUserByID(ctx, id)
		if err != nil {
			return fmt.Errorf("failed to get user by ID: %w", err)
		}
		if user.NotificationEnabled {
			msg := fmt.Sprintf("Вы верно рассмотрели расследование %s выиграли расследование", dispute.Title)
			if err = s.msgSender.SendMessage(user.ChatID, msg); err != nil {
				return fmt.Errorf("failed to send message to user: %w", err)
			}
		}
	}

	s.logger.Info("vote added to investigation", zap.String("investigation_id", id), zap.String("username", username))
	return nil
}
