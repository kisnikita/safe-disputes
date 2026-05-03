package services

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/kisnikita/safe-disputes/backend/internal/models"
	"github.com/kisnikita/safe-disputes/backend/internal/repository"
	"github.com/kisnikita/safe-disputes/backend/pkg/log"
	"go.uber.org/zap"
)

type DisputeFinder interface {
	GetDisputeByID(ctx context.Context, disputeID uuid.UUID, actorID uuid.UUID) (models.Dispute, error)
	GetDisputeForEvidence(ctx context.Context, disputeID uuid.UUID) (models.Dispute, error)
}

type DisputeReadFinder interface {
	ListDisputeReads(ctx context.Context, actorUsername string, opts models.DisputeListOpts) ([]models.DisputeRead, error)
	GetDisputeReadByID(ctx context.Context, disputeID uuid.UUID, actorUsername string) (models.DisputeRead, error)
}

type DisputeCreator interface {
	InsertDispute(ctx context.Context, dispute models.Dispute) error
	UpdateDisputeNextDeadline(ctx context.Context, disputeID uuid.UUID, nextDeadline time.Time) error
}

type ParticipantCreator interface {
	InsertParticipant(ctx context.Context, participant models.Participant) error
}

type OpponentGetter interface {
	GetOpponentID(ctx context.Context, disputeID uuid.UUID, actorID uuid.UUID) (uuid.UUID, error)
}

type ParticipantGetter interface {
	GetParticipant(ctx context.Context, disputeID uuid.UUID, userID uuid.UUID) (models.Participant, error)
}

type ParticipantUpdater interface {
	UpdateParticipant(ctx context.Context, opts models.ParticipantUpdateOpts) error
}

type MessageSender interface {
	SendMessage(chatID int64, text string) error
}

type TransactionMonitor interface {
	WaitForSuccess(ctx context.Context, boc string) error
}

type DisputeService struct {
	logger log.Logger

	disputeCreator            DisputeCreator
	disputeFinder             DisputeFinder
	disputeReadFinder         DisputeReadFinder
	participantCreator ParticipantCreator
	participantGetter  ParticipantGetter
	participantUpdater ParticipantUpdater
	opponentGetter            OpponentGetter
	userFinder                UserFinder
	msgSender                 MessageSender
	txMonitor                 TransactionMonitor
}

func minDisputeDeadline(base, endsAt time.Time) time.Time {
	if endsAt.Before(base) {
		return endsAt
	}
	return base
}

func NewDisputeService(repo *repository.Repository, log log.Logger, msgSender MessageSender) (DisputeService, error) {
	if repo == nil {
		return DisputeService{}, fmt.Errorf("repository is nil")
	}
	if log == nil {
		return DisputeService{}, fmt.Errorf("logger is nil")
	}
	return DisputeService{
		logger:                    log,
		disputeCreator:            repo,
		disputeFinder:             repo,
		disputeReadFinder:         repo,
		participantCreator: repo,
		participantGetter:  repo,
		participantUpdater: repo,
		opponentGetter:            repo,
		userFinder:                repo,
		msgSender:                 msgSender,
	}, nil
}

func (s DisputeService) WithTransactionMonitor(txMonitor TransactionMonitor) DisputeService {
	s.txMonitor = txMonitor
	return s
}

func (s DisputeService) CreateDispute(ctx context.Context, req models.CreateDisputeReq, actorUsername string) error {
	if s.txMonitor == nil {
		return fmt.Errorf("%w: tx monitor is not configured", ErrTxMonitorUnavailable)
	}
	if err := s.txMonitor.WaitForSuccess(ctx, req.Boc); err != nil {
		return err
	}

	opponent, err := s.userFinder.GetUserByUsername(ctx, req.Opponent)
	if err != nil {
		return fmt.Errorf("failed to check if opponent exists: %w", ErrUserNotFound)
	}

	dispute, err := models.NewDispute(req)
	if err != nil {
		return fmt.Errorf("failed to build dispute model %w", err)
	}
	if err = s.disputeCreator.InsertDispute(ctx, dispute); err != nil {
		return fmt.Errorf("failed to create dispute: %w", err)
	}

	participantOpponent := models.NewParticipant(opponent.ID, dispute.ID, models.DisputesResultNew)
	if err = s.participantCreator.InsertParticipant(ctx, participantOpponent); err != nil {
		return fmt.Errorf("failed to create participants for opponent: %w", err)
	}

	actor, err := s.userFinder.GetUserByUsername(ctx, actorUsername)
	if err != nil {
		return fmt.Errorf("failed to get actor user: %w", err)
	}
	participantCreator := models.NewParticipant(actor.ID, dispute.ID, models.DisputesResultSent)
	if err = s.participantCreator.InsertParticipant(ctx, participantCreator); err != nil {
		return fmt.Errorf("failed to create participants for actor: %w", err)
	}

	if opponent.NotificationEnabled {
		hoursLeft := max(int(math.Ceil(time.Until(dispute.NextDeadline).Hours())), 1)
		if err = s.msgSender.SendMessage(opponent.ChatID,
			fmt.Sprintf("У вас новое пари от %s. Примите его в течение %d часов",
				actor.Username, hoursLeft)); err != nil {
			return err
		}
	}
	return nil
}

func (s DisputeService) PrecheckCreateDispute(
	ctx context.Context,
	opponent string,
	amountNano int64,
	actorUsername string,
) error {
	if opponent == "" || amountNano <= 0 {
		return fmt.Errorf("invalid data for disute precheck")
	}

	if actorUsername == opponent {
		return fmt.Errorf("creator and opponent must be different: %w", ErrSelfOpponent)
	}

	opponentUser, err := s.userFinder.GetUserByUsername(ctx, opponent)
	if err != nil {
		return fmt.Errorf("failed to check if opponent exists: %w", ErrUserNotFound)
	}

	if !opponentUser.DisputeReadiness {
		return fmt.Errorf("opponent %s %w", opponentUser.Username, ErrUnready)
	}

	if opponentUser.MinimumDisputeAmountNano > amountNano {
		return fmt.Errorf("%d %w", amountNano, ErrMinimalAmount)
	}

	return nil
}

func (s DisputeService) ListDisputes(ctx context.Context, opts models.DisputeListOpts, actorUsername string,
) ([]models.DisputeRead, error) {
	disputes, err := s.disputeReadFinder.ListDisputeReads(ctx, actorUsername, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list disputes: %w", err)
	}

	if len(disputes) == 0 {
		s.logger.Info("no disputes found", zap.String("actor", actorUsername))
		return []models.DisputeRead{}, nil
	}
	return disputes, nil
}

func (s DisputeService) GetDispute(ctx context.Context, disputeID string, actorUsername string,
) (models.DisputeRead, error) {
	disputeUUID, err := uuid.Parse(disputeID)
	if err != nil {
		return models.DisputeRead{}, fmt.Errorf("invalid dispute ID format: %w", err)
	}

	dispute, err := s.disputeReadFinder.GetDisputeReadByID(ctx, disputeUUID, actorUsername)
	if err != nil {
		return models.DisputeRead{}, fmt.Errorf("failed to get dispute: %w", err)
	}
	return dispute, nil
}

func (s DisputeService) AcceptDispute(ctx context.Context, disputeID string, acceptorUsername string) error {
	// Get acceptor and update
	acceptor, err := s.userFinder.GetUserByUsername(ctx, acceptorUsername)
	if err != nil {
		return fmt.Errorf("failed to get acceptor user: %w", err)
	}

	disputeUUID, err := uuid.Parse(disputeID)
	if err != nil {
		return fmt.Errorf("invalid dispute ID format: %w", err)
	}

	participant, err := s.participantGetter.GetParticipant(ctx, disputeUUID, acceptor.ID)
	if err != nil {
		return err
	}

	if participant.Status != models.DisputesStatusNew {
		return fmt.Errorf("user2duspite %s is not in new status", participant.ID)
	}

	status := models.DisputesStatusCurrent
	result := models.DisputesResultProcessed
	opts := models.ParticipantUpdateOpts{
		ID:     participant.ID,
		Status: &status,
		Result: &result,
	}
	if err = s.participantUpdater.UpdateParticipant(ctx, opts); err != nil {
		return fmt.Errorf("failed to update rejector dispute status: %w", err)
	}

	// Get opponent and update
	opID, err := s.opponentGetter.GetOpponentID(ctx, disputeUUID, acceptor.ID)
	if err != nil {
		return fmt.Errorf("failed to get opponent ID: %w", err)
	}

	participantOp, err := s.participantGetter.GetParticipant(ctx, disputeUUID, opID)
	if err != nil {
		return err
	}
	opts.ID = participantOp.ID
	if err = s.participantUpdater.UpdateParticipant(ctx, opts); err != nil {
		return fmt.Errorf("failed to update opponent dispute status: %w", err)
	}

	dispute, err := s.disputeFinder.GetDisputeByID(ctx, disputeUUID, acceptor.ID)
	if err != nil {
		return fmt.Errorf("failed to get dispute: %w", err)
	}
	if err = s.disputeCreator.UpdateDisputeNextDeadline(ctx, disputeUUID, dispute.EndsAt); err != nil {
		return fmt.Errorf("failed to set next deadline for accepted dispute: %w", err)
	}

	// Notify opponent
	opponent, err := s.userFinder.GetUserByID(ctx, opID)
	if err != nil {
		return fmt.Errorf("failed to get opponent user: %w", err)
	}

	if opponent.NotificationEnabled {
		msg := fmt.Sprintf("Ваше пари %s было принято %s", dispute.Title, acceptor.Username)
		if err = s.msgSender.SendMessage(opponent.ChatID, msg); err != nil {
			return err
		}
	}
	return nil
}

func (s DisputeService) RejectDispute(ctx context.Context, disputeID string, rejectorUsername string) error {
	// Get rejector and update
	acceptor, err := s.userFinder.GetUserByUsername(ctx, rejectorUsername)
	if err != nil {
		return fmt.Errorf("failed to get acceptor user: %w", err)
	}

	disputeUUID, err := uuid.Parse(disputeID)
	if err != nil {
		return fmt.Errorf("invalid dispute ID format: %w", err)
	}

	participant, err := s.participantGetter.GetParticipant(ctx, disputeUUID, acceptor.ID)
	if err != nil {
		return err
	}

	if participant.Status != models.DisputesStatusNew {
		return fmt.Errorf("user2duspite %s is not in new status", participant.ID)
	}

	status := models.DisputesStatusPassed
	result := models.DisputesResultRejected
	opts := models.ParticipantUpdateOpts{
		ID:     participant.ID,
		Status: &status,
		Result: &result,
	}
	if err = s.participantUpdater.UpdateParticipant(ctx, opts); err != nil {
		return fmt.Errorf("failed to update rejector dispute status: %w", err)
	}

	// Get opponent and update
	opID, err := s.opponentGetter.GetOpponentID(ctx, disputeUUID, acceptor.ID)
	if err != nil {
		return fmt.Errorf("failed to get opponent ID: %w", err)
	}

	participantOp, err := s.participantGetter.GetParticipant(ctx, disputeUUID, opID)
	if err != nil {
		return err
	}
	opts.ID = participantOp.ID
	tr := true
	opts.Claim = &tr
	if err = s.participantUpdater.UpdateParticipant(ctx, opts); err != nil {
		return fmt.Errorf("failed to update opponent dispute status: %w", err)
	}

	// Notify opponent
	opponent, err := s.userFinder.GetUserByID(ctx, opID)
	if err != nil {
		return fmt.Errorf("failed to get opponent user: %w", err)
	}

	dispute, err := s.disputeFinder.GetDisputeByID(ctx, disputeUUID, acceptor.ID)
	if err != nil {
		return fmt.Errorf("failed to get dispute: %w", err)
	}

	if opponent.NotificationEnabled {
		msg := fmt.Sprintf("Ваше пари %s было отменено %s", dispute.Title, acceptor.Username)
		if err = s.msgSender.SendMessage(opponent.ChatID, msg); err != nil {
			return err
		}
	}
	return nil
}

func (s DisputeService) ClaimDispute(ctx context.Context, disputeID string, claimerUsername string) error {
	// Get claimer and update
	claimer, err := s.userFinder.GetUserByUsername(ctx, claimerUsername)
	if err != nil {
		return fmt.Errorf("failed to get claimer user: %w", err)
	}

	disputeUUID, err := uuid.Parse(disputeID)
	if err != nil {
		return fmt.Errorf("invalid dispute ID format: %w", err)
	}

	participant, err := s.participantGetter.GetParticipant(ctx, disputeUUID, claimer.ID)
	if err != nil {
		return err
	}

	if participant.Status != models.DisputesStatusPassed {
		return fmt.Errorf("user2duspite %s is not in current status", participant.ID)
	}

	fl := false
	opts := models.ParticipantUpdateOpts{
		ID:    participant.ID,
		Claim: &fl,
	}
	if err = s.participantUpdater.UpdateParticipant(ctx, opts); err != nil {
		return fmt.Errorf("failed to update claimer dispute status: %w", err)
	}
	return nil
}

func (s DisputeService) VoteDispute(ctx context.Context, disputeID string, claimerUsername string,
	win bool) error {
	voter, err := s.userFinder.GetUserByUsername(ctx, claimerUsername)
	if err != nil {
		return fmt.Errorf("failed to get claimer user: %w", err)
	}

	disputeUUID, err := uuid.Parse(disputeID)
	if err != nil {
		return fmt.Errorf("invalid dispute ID format: %w", err)
	}

	participant, err := s.participantGetter.GetParticipant(ctx, disputeUUID, voter.ID)
	if err != nil {
		return err
	}

	opID, err := s.opponentGetter.GetOpponentID(ctx, disputeUUID, voter.ID)
	if err != nil {
		return fmt.Errorf("failed to get opponent ID: %w", err)
	}

	opponent, err := s.userFinder.GetUserByID(ctx, opID)
	if err != nil {
		return fmt.Errorf("failed to get opponent user: %w", err)
	}

	participantOp, err := s.participantGetter.GetParticipant(ctx, disputeUUID, opID)
	if err != nil {
		return err
	}

	dispute, err := s.disputeFinder.GetDisputeByID(ctx, disputeUUID, voter.ID)
	if err != nil {
		return fmt.Errorf("failed to get dispute: %w", err)
	}

	var opts models.ParticipantUpdateOpts
	tr := true
	status := models.DisputesStatusPassed

	// --- Opponent not voted yet ---
	if participantOp.Result == models.DisputesResultProcessed {
		opts.ID = participant.ID
		opts.Vote = &win
		res := models.DisputesResultAnswered
		opts.Result = &res
		err := s.participantUpdater.UpdateParticipant(ctx, opts)
		if err != nil {
			return fmt.Errorf("failed to update voter dispute status: %w", err)
		}
		return nil
	}

	// -- Opponent already voted ---

	// -- draw --
	if !participantOp.Vote && !win {
		res := models.DisputesResultDraw
		opts.ID = participantOp.ID
		opts.Result = &res
		opts.Status = &status
		opts.Claim = &tr
		err := s.participantUpdater.UpdateParticipant(ctx, opts)
		if err != nil {
			return fmt.Errorf("failed to update opponent dispute status: %w", err)
		}
		opts.ID = participant.ID
		if err = s.participantUpdater.UpdateParticipant(ctx, opts); err != nil {
			return fmt.Errorf("failed to update voter dispute status: %w", err)
		}

		if opponent.NotificationEnabled {
			msg := fmt.Sprintf("Ваше пари %s c игроком %s завершилось вничью. Заберите награду!",
				dispute.Title, voter.Username)
			if err = s.msgSender.SendMessage(opponent.ChatID, msg); err != nil {
				return err
			}
		}
		return nil
	}

	// -- win --
	if !participantOp.Vote && win {
		res := models.DisputesResultLose
		opts.ID = participantOp.ID
		opts.Result = &res
		opts.Status = &status
		if err := s.participantUpdater.UpdateParticipant(ctx, opts); err != nil {
			return fmt.Errorf("failed to update opponent dispute status: %w", err)
		}

		res = models.DisputesResultWin
		opts.ID = participant.ID
		opts.Result = &res
		opts.Claim = &tr
		opts.Vote = &win
		if err := s.participantUpdater.UpdateParticipant(ctx, opts); err != nil {
			return fmt.Errorf("failed to update voter dispute status: %w", err)
		}
		if opponent.NotificationEnabled {
			msg := fmt.Sprintf("Ваше пари %s c игроком %s завершилось поражением.", dispute.Title, voter.Username)
			if err = s.msgSender.SendMessage(opponent.ChatID, msg); err != nil {
				return err
			}
		}
		return nil
	}

	// -- lose --
	if participantOp.Vote && !win {
		res := models.DisputesResultLose
		opts.ID = participant.ID
		opts.Result = &res
		opts.Status = &status
		if err := s.participantUpdater.UpdateParticipant(ctx, opts); err != nil {
			return fmt.Errorf("failed to update voter dispute status: %w", err)
		}

		res = models.DisputesResultWin
		opts.ID = participantOp.ID
		opts.Result = &res
		opts.Claim = &tr
		if err := s.participantUpdater.UpdateParticipant(ctx, opts); err != nil {
			return fmt.Errorf("failed to update opponent dispute status: %w", err)
		}
		if opponent.NotificationEnabled {
			msg := fmt.Sprintf("Ваше пари %s c игроком %s завершилось победой. Заберите награду!",
				dispute.Title, voter.Username)
			if err = s.msgSender.SendMessage(opponent.ChatID, msg); err != nil {
				return err
			}
		}
		return nil
	}

	// -- investigation --
	if participantOp.Vote && win {
		res := models.DisputesResultEvidence
		opts.ID = participant.ID
		opts.Vote = &win
		opts.Result = &res
		if err := s.participantUpdater.UpdateParticipant(ctx, opts); err != nil {
			return fmt.Errorf("failed to update voter dispute status: %w", err)
		}

		opts.ID = participantOp.ID
		if err := s.participantUpdater.UpdateParticipant(ctx, opts); err != nil {
			return fmt.Errorf("failed to update voter dispute status: %w", err)
		}
		nextDeadline := minDisputeDeadline(time.Now().Add(24*time.Hour), dispute.EndsAt)
		if err := s.disputeCreator.UpdateDisputeNextDeadline(ctx, disputeUUID, nextDeadline); err != nil {
			return fmt.Errorf("failed to set next deadline for evidence stage: %w", err)
		}
		if opponent.NotificationEnabled {
			msg := fmt.Sprintf("Ваше пари %s c игроком %s требует доказательств.",
				dispute.Title, voter.Username)
			if err = s.msgSender.SendMessage(opponent.ChatID, msg); err != nil {
				return err
			}
		}
		return nil
	}
	return nil
}

func (s DisputeService) GetDisputeForEvidence(ctx context.Context, disputeID string) (models.Dispute, error) {
	disputeUUID, err := uuid.Parse(disputeID)
	if err != nil {
		return models.Dispute{}, fmt.Errorf("invalid dispute ID format: %w", err)
	}

	dispute, err := s.disputeFinder.GetDisputeForEvidence(ctx, disputeUUID)
	if err != nil {
		return models.Dispute{}, fmt.Errorf("failed to get dispute for evidence: %w", err)
	}

	return dispute, nil
}
