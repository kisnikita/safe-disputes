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
	ListInvestigationCards(ctx context.Context, actorUsername string, opts models.InvestigationListOpts) ([]models.InvestigationCard, error)
	GetInvestigationDetails(ctx context.Context, invID uuid.UUID, actorUsername string) (models.InvestigationDetails, error)
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

type JurorSeener interface {
	MarkJurorsSeen(ctx context.Context, actorUsername string, investigationIDs []uuid.UUID) error
}

type InvestigationService struct {
	logger                  log.Logger
	investigationCreator    InvestigationCreator
	investigationFinder     InvestigationFinder
	investigationReadFinder InvestigationReadFinder
	investigationUpdater    InvestigationUpdater
	investigationDeleter    InvestigationDeleter
	userFinder              UserFinder
	userUpdater             UserUpdater
	participantUpdater      ParticipantUpdater
	participantGetter       ParticipantGetter
	jurorFinder             JurorFinder
	jurorUpdater            JurorUpdater
	jurorSeener             JurorSeener
	disputeFinder           DisputeFinder
	msgSender               MessageSender
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
		logger:                  log,
		investigationCreator:    repo,
		investigationFinder:     repo,
		investigationReadFinder: repo,
		investigationUpdater:    repo,
		investigationDeleter:    repo,
		userFinder:              repo,
		userUpdater:             repo,
		participantUpdater:      repo,
		participantGetter:       repo,
		jurorFinder:             repo,
		jurorUpdater:            repo,
		jurorSeener:             repo,
		disputeFinder:           repo,
		msgSender:               msgSender,
	}, nil
}

func (s InvestigationService) ListInvestigation(ctx context.Context, opts models.InvestigationListOpts,
	actorUsername string,
) ([]models.InvestigationCard, error) {
	investigations, err := s.investigationReadFinder.ListInvestigationCards(ctx, actorUsername, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list investigations: %w", err)
	}

	if len(investigations) == 0 {
		s.logger.Info("no investigations found", zap.String("actor", actorUsername))
		return []models.InvestigationCard{}, nil
	}

	return investigations, nil
}

func (s InvestigationService) GetInvestigation(ctx context.Context, id, actorUsername string,
) (models.InvestigationDetails, error) {
	invUUID, err := uuid.Parse(id)
	if err != nil {
		return models.InvestigationDetails{}, fmt.Errorf("invalid investigation ID format: %w", err)
	}

	investigation, err := s.investigationReadFinder.GetInvestigationDetails(ctx, invUUID, actorUsername)
	if err != nil {
		return models.InvestigationDetails{}, fmt.Errorf("failed to get investigation: %w", err)
	}

	return investigation, nil
}

func (s InvestigationService) VoteInvestigation(ctx context.Context, investigationID, username, vote string) error {
	user, err := s.userFinder.GetUserByUsername(ctx, username)
	if err != nil {
		return fmt.Errorf("failed to get user by username: %w", err)
	}

	invUUID, err := uuid.Parse(investigationID)
	if err != nil {
		return fmt.Errorf("invalid investigation ID format: %w", err)
	}

	participant, err := s.jurorFinder.GetJuror(ctx, invUUID, user.ID)
	if err != nil {
		return fmt.Errorf("failed to get investigation: %w", err)
	}

	opts := models.JurorUpdateOpts{
		ID: participant.ID, 
		Vote: &vote,
		Result: new(models.InvestigationResultSent),
		SeenAt: new(true),
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

	switch vote {
	case "p1":
		investigation.P1 += 1
	case "p2":
		investigation.P2 += 1
	default:
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

	s.logger.Info("investigation vote added", zap.String("investigation_id", investigationID), zap.String("username", username))

	invUpdateOpts.Status = new(models.InvestigationStatusPassed)
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
		participantUpdateOpts := models.ParticipantUpdateOpts{
			ID:     participantP1.ID,
			Status: new(models.DisputesStatusPassed),
			Result: new(models.DisputesResultDraw),
			Claim:  new(true),
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
		participantUpdateOpts := models.ParticipantUpdateOpts{
			ID:     participantP1.ID,
			Status: new(models.DisputesStatusPassed),
			Result: new(models.DisputesResultWin),
			Claim: new(true),
		}
		if err = s.participantUpdater.UpdateParticipant(ctx, participantUpdateOpts); err != nil {
			return fmt.Errorf("failed to update participants: %w", err)
		}

		participantUpdateOpts.ID = participantP2.ID
		participantUpdateOpts.Result = new(models.DisputesResultLose)
		participantUpdateOpts.Claim = new(false)
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

	// res == p2
	participantUpdateOpts := models.ParticipantUpdateOpts{
		ID:     participantP2.ID,
		Status: new(models.DisputesStatusPassed),
		Result: new(models.DisputesResultWin),
		Claim:  new(true),
	}
	if err = s.participantUpdater.UpdateParticipant(ctx, participantUpdateOpts); err != nil {
		return fmt.Errorf("failed to update participants: %w", err)
	}

	participantUpdateOpts.ID = participantP1.ID
	participantUpdateOpts.Result = new(models.DisputesResultLose)
	participantUpdateOpts.Claim = new(false)
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
		u, err := s.userFinder.GetUserByID(ctx, id)
		if err != nil {
			return fmt.Errorf("failed to get user by ID: %w", err)
		}
		if u.NotificationEnabled {
			msg := fmt.Sprintf("Вы верно рассмотрели расследование %s выиграли расследование", dispute.Title)
			if err = s.msgSender.SendMessage(u.ChatID, msg); err != nil {
				return fmt.Errorf("failed to send message to user: %w", err)
			}
		}
	}

	s.logger.Info("vote added to investigation", zap.String("investigation_id", investigationID), zap.String("username", username))
	return nil
}

func (s InvestigationService) MarkInvestigationsSeen(ctx context.Context, actorUsername string, investigationIDs []string,
) error {
	ids := make([]uuid.UUID, 0, len(investigationIDs))
	for _, rawID := range investigationIDs {
		id, err := uuid.Parse(rawID)
		if err != nil {
			return fmt.Errorf("invalid investigation ID format: %w", err)
		}
		ids = append(ids, id)
	}
	if err := s.jurorSeener.MarkJurorsSeen(ctx, actorUsername, ids); err != nil {
		return fmt.Errorf("failed to mark investigations seen: %w", err)
	}
	return nil
}
