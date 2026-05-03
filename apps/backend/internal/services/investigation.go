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
	GetInvestigation(ctx context.Context, invID, userID uuid.UUID) (models.Investigation, error)
}

type InvestigationReadFinder interface {
	ListInvestigationReads(ctx context.Context, actorUsername string, opts models.InvestigationListOpts) ([]models.InvestigationRead, error)
	GetInvestigationRead(ctx context.Context, invID uuid.UUID, actorUsername string) (models.InvestigationRead, error)
}

type InvestigationUpdater interface {
	UpdateInvestigation(ctx context.Context, opts models.InvestigationUpdateOpts) error
}

type InvestigationDeleter interface {
	DeleteUsersWithoutVote(ctx context.Context, invID uuid.UUID) error
}

type JurorFinder interface {
	GetJuror(ctx context.Context, invID, userID uuid.UUID) (models.Juror, error)
	GetWinnersIDs(ctx context.Context, invID uuid.UUID, winner string) ([]uuid.UUID, error)
	GetDisputesUsers(ctx context.Context, invID uuid.UUID) ([]models.User, error)
}

type JurorUpdater interface {
	UpdateJuror(ctx context.Context, opts models.JurorUpdateOpts) error
	UpdateWinnersResult(ctx context.Context, invID uuid.UUID, ids []uuid.UUID) error
}

type InvestigationService struct {
	logger                    log.Logger
	investigationCreator      InvestigationCreator
	investigationFinder       InvestigationFinder
	investigationReadFinder   InvestigationReadFinder
	investigationUpdater      InvestigationUpdater
	investigationDeleter      InvestigationDeleter
	userFinder                UserFinder
	userUpdater               UserUpdater
	participantUpdater ParticipantUpdater
	participantGetter  ParticipantGetter
	jurorFinder  JurorFinder
	jurorUpdater JurorUpdater
	disputeFinder             DisputeFinder
	msgSender                 MessageSender
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
		logger:                    log,
		investigationCreator:      repo,
		investigationFinder:       repo,
		investigationReadFinder:   repo,
		investigationUpdater:      repo,
		investigationDeleter:      repo,
		userFinder:                repo,
		userUpdater:               repo,
		participantUpdater: repo,
		participantGetter:  repo,
		jurorFinder:  repo,
		jurorUpdater: repo,
		disputeFinder:             repo,
		msgSender:                 msgSender,
	}, nil
}

func (s InvestigationService) ListInvestigation(ctx context.Context, opts models.InvestigationListOpts, actorUsername string,
) ([]models.InvestigationRead, error) {
	investigations, err := s.investigationReadFinder.ListInvestigationReads(ctx, actorUsername, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list investigations: %w", err)
	}

	if len(investigations) == 0 {
		s.logger.Info("no investigations found", zap.String("actor", actorUsername))
		return []models.InvestigationRead{}, nil
	}

	return investigations, nil
}

func (s InvestigationService) GetInvestigation(ctx context.Context, id, actorUsername string) (models.InvestigationRead, error) {
	invUUID, err := uuid.Parse(id)
	if err != nil {
		return models.InvestigationRead{}, fmt.Errorf("invalid investigation ID format: %w", err)
	}

	investigation, err := s.investigationReadFinder.GetInvestigationRead(ctx, invUUID, actorUsername)
	if err != nil {
		return models.InvestigationRead{}, fmt.Errorf("failed to get investigation: %w", err)
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

	participant, err := s.jurorFinder.GetJuror(ctx, invUUID, user.ID)
	if err != nil {
		return fmt.Errorf("failed to get investigation: %w", err)
	}

	result := models.InvestigationResultSent
	opts := models.JurorUpdateOpts{
		ID: participant.ID, Vote: &vote, Result: &result,
	}
	err = s.jurorUpdater.UpdateJuror(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to update jurors: %w", err)
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

	winnerIDs, err := s.jurorFinder.GetWinnersIDs(ctx, investigation.ID, res)
	if err != nil {
		return fmt.Errorf("failed to get winners IDs: %w", err)
	}
	if err = s.userUpdater.EarnWinnerRating(ctx, winnerIDs); err != nil {
		return fmt.Errorf("failed to update winner rating: %w", err)
	}
	if err = s.jurorUpdater.UpdateWinnersResult(ctx, investigation.ID, winnerIDs); err != nil {
		return fmt.Errorf("failed to update winners result: %w", err)
	}
	users, err := s.jurorFinder.GetDisputesUsers(ctx, investigation.ID)
	if err != nil {
		return fmt.Errorf("failed to get users from investigation: %w", err)
	}

	participantP1, err := s.participantGetter.GetParticipant(ctx, investigation.DisputeID, users[0].ID)
	if err != nil {
		return fmt.Errorf("failed to get participants for user1: %w", err)
	}
	participantP2, err := s.participantGetter.GetParticipant(ctx, investigation.DisputeID, users[1].ID)
	if err != nil {
		return fmt.Errorf("failed to get participants for user2: %w", err)
	}

	dispute, err := s.disputeFinder.GetDisputeByID(ctx, investigation.DisputeID, users[0].ID)
	if err != nil {
		return fmt.Errorf("failed to get dispute by ID: %w", err)
	}
	if res == "draw" {
		result := models.DisputesResultDraw
		status := models.DisputesStatusPassed
		tr := true

		participantUpdateOpts := models.ParticipantUpdateOpts{
			ID:     participantP1.ID,
			Status: &status,
			Result: &result,
			Claim:  &tr,
		}
		if err = s.participantUpdater.UpdateParticipant(ctx, participantUpdateOpts); err != nil {
			return fmt.Errorf("failed to update participants: %w", err)
		}
		participantUpdateOpts.ID = participantP2.ID
		if err = s.participantUpdater.UpdateParticipant(ctx, participantUpdateOpts); err != nil {
			return fmt.Errorf("failed to update participants: %w", err)
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

		participantUpdateOpts := models.ParticipantUpdateOpts{
			ID:     participantP1.ID,
			Status: &status,
			Result: &result,
			Claim:  &tr,
		}
		if err = s.participantUpdater.UpdateParticipant(ctx, participantUpdateOpts); err != nil {
			return fmt.Errorf("failed to update participants: %w", err)
		}
		result = models.DisputesResultLose
		fl := false
		participantUpdateOpts.ID = participantP2.ID
		participantUpdateOpts.Result = &result
		participantUpdateOpts.Claim = &fl
		if err = s.participantUpdater.UpdateParticipant(ctx, participantUpdateOpts); err != nil {
			return fmt.Errorf("failed to update participants: %w", err)
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

	participantUpdateOpts := models.ParticipantUpdateOpts{
		ID:     participantP2.ID,
		Status: &status,
		Result: &r,
		Claim:  &tr,
	}
	if err = s.participantUpdater.UpdateParticipant(ctx, participantUpdateOpts); err != nil {
		return fmt.Errorf("failed to update participants: %w", err)
	}
	r = models.DisputesResultLose
	fl := false
	participantUpdateOpts.ID = participantP1.ID
	participantUpdateOpts.Result = &r
	participantUpdateOpts.Claim = &fl
	if err = s.participantUpdater.UpdateParticipant(ctx, participantUpdateOpts); err != nil {
		return fmt.Errorf("failed to update participants: %w", err)
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
