package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kisnikita/safe-disputes/backend/internal/models"
	"github.com/kisnikita/safe-disputes/backend/internal/repository"
	"github.com/kisnikita/safe-disputes/backend/pkg/log"
	"go.uber.org/zap"
)

type DisputeFinder interface {
	GetDisputeByID(ctx context.Context, disputeID uuid.UUID) (models.Dispute, error)
	GetDisputeForEvidence(ctx context.Context, disputeID uuid.UUID) (models.Dispute, error)
}

type DisputeReadFinder interface {
	ListDisputeCards(ctx context.Context, actorUsername string, opts models.DisputeListOpts) ([]models.DisputeCard, error)
	GetDisputeDetailsByID(ctx context.Context, disputeID uuid.UUID, actorUsername string) (models.DisputeDetails, error)
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

type ParticipantSeener interface {
	MarkParticipantsSeen(ctx context.Context, actorUsername string, disputeIDs []uuid.UUID) error
}

type MessageSender interface {
	SendMessage(chatID int64, text string) error
}

type TransactionMonitor interface {
	WaitForSuccess(ctx context.Context, boc string) error
}

type DisputeService struct {
	logger log.Logger

	disputeCreator     DisputeCreator
	disputeFinder      DisputeFinder
	disputeReadFinder  DisputeReadFinder
	participantCreator ParticipantCreator
	participantGetter  ParticipantGetter
	participantUpdater ParticipantUpdater
	participantSeener  ParticipantSeener
	opponentGetter     OpponentGetter
	userFinder         UserFinder
	msgSender          MessageSender
	txMonitor          TransactionMonitor
}

func NewDisputeService(repo *repository.Repository, log log.Logger, msgSender MessageSender) (DisputeService, error) {
	if repo == nil {
		return DisputeService{}, fmt.Errorf("repository is nil")
	}
	if log == nil {
		return DisputeService{}, fmt.Errorf("logger is nil")
	}
	return DisputeService{
		logger:             log,
		disputeCreator:     repo,
		disputeFinder:      repo,
		disputeReadFinder:  repo,
		participantCreator: repo,
		participantGetter:  repo,
		participantUpdater: repo,
		participantSeener:  repo,
		opponentGetter:     repo,
		userFinder:         repo,
		msgSender:          msgSender,
	}, nil
}

func (s DisputeService) WithTransactionMonitor(txMonitor TransactionMonitor) DisputeService {
	s.txMonitor = txMonitor
	return s
}

func (s DisputeService) ensureTxSuccess(ctx context.Context, boc string) error {
	if s.txMonitor == nil {
		return fmt.Errorf("%w: tx monitor is not configured", ErrTxMonitorUnavailable)
	}
	return s.txMonitor.WaitForSuccess(ctx, boc)
}

func (s DisputeService) CreateDispute(ctx context.Context, req models.CreateDisputeReq, creatorUsername string) error {
	if err := s.ensureTxSuccess(ctx, req.Boc); err != nil {
		return err
	}

	opponent, err := s.userFinder.GetUserByUsername(ctx, req.Opponent)
	if err != nil {
		return fmt.Errorf("failed to check if opponent exists: %w", ErrUserNotFound)
	}

	dispute, err := models.NewDispute(req)
	switch {
	case errors.Is(err, models.ErrDisputeValidation):
		return fmt.Errorf("%w: %s", ErrValidation, err)
	case err != nil:
		return fmt.Errorf("failed to build dispute model %w", err)
	}
	if err = s.disputeCreator.InsertDispute(ctx, dispute); err != nil {
		return fmt.Errorf("failed to create dispute: %w", err)
	}

	participantOpponent := models.NewParticipant(opponent.ID, dispute.ID, models.DisputesResultNew, false)
	if err = s.participantCreator.InsertParticipant(ctx, participantOpponent); err != nil {
		return fmt.Errorf("failed to create participants for opponent: %w", err)
	}

	creator, err := s.userFinder.GetUserByUsername(ctx, creatorUsername)
	if err != nil {
		return fmt.Errorf("failed to get actor user: %w", err)
	}
	participantCreator := models.NewParticipant(creator.ID, dispute.ID, models.DisputesResultSent, true)
	if err = s.participantCreator.InsertParticipant(ctx, participantCreator); err != nil {
		return fmt.Errorf("failed to create participants for creator: %w", err)
	}

	if opponent.NotificationEnabled {
		if err = s.msgSender.SendMessage(opponent.ChatID,
			fmt.Sprintf("Пользователь %s вызвает вас на пари %s", creator.Username, dispute.Title)); err != nil {
			return err
		}
	}
	return nil
}

func (s DisputeService) PrecheckCreateDispute(ctx context.Context, opponent string, amountNano int64,
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
) ([]models.DisputeCard, error) {
	disputes, err := s.disputeReadFinder.ListDisputeCards(ctx, actorUsername, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list disputes: %w", err)
	}

	if len(disputes) == 0 {
		s.logger.Info("no disputes found", zap.String("actor", actorUsername))
		return []models.DisputeCard{}, nil
	}
	return disputes, nil
}

func (s DisputeService) MarkDisputesSeen(ctx context.Context, actorUsername string, disputeIDs []string,
) error {
	ids := make([]uuid.UUID, 0, len(disputeIDs))
	for _, rawID := range disputeIDs {
		id, err := uuid.Parse(rawID)
		if err != nil {
			return fmt.Errorf("invalid dispute ID format: %w", err)
		}
		ids = append(ids, id)
	}

	err := s.participantSeener.MarkParticipantsSeen(ctx, actorUsername, ids)
	if err != nil {
		return fmt.Errorf("failed to mark disputes seen: %w", err)
	}
	return nil
}

func (s DisputeService) GetDispute(ctx context.Context, disputeID string, actorUsername string,
) (models.DisputeDetails, error) {
	disputeUUID, err := uuid.Parse(disputeID)
	if err != nil {
		return models.DisputeDetails{}, fmt.Errorf("invalid dispute ID format: %w", err)
	}

	dispute, err := s.disputeReadFinder.GetDisputeDetailsByID(ctx, disputeUUID, actorUsername)
	if err != nil {
		return models.DisputeDetails{}, fmt.Errorf("failed to get dispute: %w", err)
	}
	return dispute, nil
}

func (s DisputeService) AcceptDispute(ctx context.Context, disputeID string, acceptorUsername string, boc string) error {
	if err := s.ensureTxSuccess(ctx, boc); err != nil {
		return err
	}

	acceptor, err := s.userFinder.GetUserByUsername(ctx, acceptorUsername)
	if err != nil {
		return fmt.Errorf("failed to get acceptor user: %w", err)
	}

	disputeUUID, err := uuid.Parse(disputeID)
	if err != nil {
		return fmt.Errorf("invalid dispute ID format: %w", err)
	}

	participantAccepter, err := s.participantGetter.GetParticipant(ctx, disputeUUID, acceptor.ID)
	if err != nil {
		return err
	}

	if participantAccepter.Status != models.DisputesStatusNew {
		return fmt.Errorf("user2duspite %s is not in new status", participantAccepter.ID)
	}

	opts := models.ParticipantUpdateOpts{
		ID:     participantAccepter.ID,
		Status: new(models.DisputesStatusCurrent),
		Result: new(models.DisputesResultProcessed),
	}
	if err = s.participantUpdater.UpdateParticipant(ctx, opts); err != nil {
		return fmt.Errorf("failed to update acceptor dispute status: %w", err)
	}

	opID, err := s.opponentGetter.GetOpponentID(ctx, disputeUUID, acceptor.ID)
	if err != nil {
		return fmt.Errorf("failed to get opponent ID: %w", err)
	}

	participantOpponent, err := s.participantGetter.GetParticipant(ctx, disputeUUID, opID)
	if err != nil {
		return err
	}
	opts.ID = participantOpponent.ID
	if err = s.participantUpdater.UpdateParticipant(ctx, opts); err != nil {
		return fmt.Errorf("failed to update opponent dispute status: %w", err)
	}

	// Notify opponent
	opponent, err := s.userFinder.GetUserByID(ctx, opID)
	if err != nil {
		return fmt.Errorf("failed to get opponent user: %w", err)
	}

	if opponent.NotificationEnabled {
		dispute, err := s.disputeFinder.GetDisputeByID(ctx, disputeUUID)
		if err != nil {
			return fmt.Errorf("failed to get dispute: %w", err)
		}
		msg := fmt.Sprintf("Ваше пари %s было принято пользователем %s", dispute.Title, acceptor.Username)
		if err = s.msgSender.SendMessage(opponent.ChatID, msg); err != nil {
			return err
		}
	}
	return nil
}

func (s DisputeService) RejectDispute(ctx context.Context, disputeID string, rejectorUsername string) error {
	rejector, err := s.userFinder.GetUserByUsername(ctx, rejectorUsername)
	if err != nil {
		return fmt.Errorf("failed to get rejector user: %w", err)
	}

	disputeUUID, err := uuid.Parse(disputeID)
	if err != nil {
		return fmt.Errorf("invalid dispute ID format: %w", err)
	}

	participantRejector, err := s.participantGetter.GetParticipant(ctx, disputeUUID, rejector.ID)
	if err != nil {
		return err
	}

	if participantRejector.Status != models.DisputesStatusNew {
		return fmt.Errorf("user2duspite %s is not in new status", participantRejector.ID)
	}

	opID, err := s.opponentGetter.GetOpponentID(ctx, disputeUUID, rejector.ID)
	if err != nil {
		return fmt.Errorf("failed to get opponent ID: %w", err)
	}
	participantOpponent, err := s.participantGetter.GetParticipant(ctx, disputeUUID, opID)
	if err != nil {
		return err
	}

	creatorID := participantRejector.UserID
	if participantRejector.Result != models.DisputesResultSent {
		creatorID = participantOpponent.UserID
	}

	switch {
	// Cancellation is only valid before acceptance, while one side has "sent" and the other has "new".
	case participantRejector.Result == models.DisputesResultNew && participantOpponent.Result == models.DisputesResultSent:
		// ok
	case participantRejector.Result == models.DisputesResultSent && participantOpponent.Result == models.DisputesResultNew:
		//ok
	default:
		return fmt.Errorf("%w: bad status: rejector: %s, opponent: %s",
			ErrValidation, participantRejector.Result, participantOpponent.Result)
	}

	opts := models.ParticipantUpdateOpts{
		ID:     participantRejector.ID,
		Status: new(models.DisputesStatusPassed),
		Result: new(models.DisputesResultRejected),
		// claimable only for creator
		IsClaimable: new(participantRejector.UserID == creatorID),
		// remove mark for rejector who didn't create bet
		Seen: new(participantRejector.UserID != creatorID),
	}
	if err = s.participantUpdater.UpdateParticipant(ctx, opts); err != nil {
		return fmt.Errorf("failed to update rejector dispute status: %w", err)
	}

	opts.ID = participantOpponent.ID
	opts.IsClaimable = new(participantOpponent.UserID == creatorID)
	// always mark reject for opponent
	opts.Seen = new(false)
	if err = s.participantUpdater.UpdateParticipant(ctx, opts); err != nil {
		return fmt.Errorf("failed to update opponent dispute status: %w", err)
	}

	// Notify opponent
	opponent, err := s.userFinder.GetUserByID(ctx, opID)
	if err != nil {
		return fmt.Errorf("failed to get opponent user: %w", err)
	}

	if opponent.NotificationEnabled {
		dispute, err := s.disputeFinder.GetDisputeByID(ctx, disputeUUID)
		if err != nil {
			return fmt.Errorf("failed to get dispute: %w", err)
		}
		format := "Пользователь %s отменил пари %s"
		if opponent.ID == creatorID {
			format = "Пользователь %s отклонил ваш вызов на пари %s. Вы можете вернуть вашу ставку и депозит!"
		}
		if err = s.msgSender.SendMessage(opponent.ChatID, fmt.Sprintf(format, rejector.Username, dispute.Title)); err != nil {
			return err
		}
	}
	return nil
}

func (s DisputeService) ClaimDispute(ctx context.Context, disputeID string, claimerUsername string, boc string) error {
	if err := s.ensureTxSuccess(ctx, boc); err != nil {
		return err
	}

	claimer, err := s.userFinder.GetUserByUsername(ctx, claimerUsername)
	if err != nil {
		return fmt.Errorf("failed to get claimer user: %w", err)
	}

	disputeUUID, err := uuid.Parse(disputeID)
	if err != nil {
		return fmt.Errorf("invalid dispute ID format: %w", err)
	}

	participantClaimer, err := s.participantGetter.GetParticipant(ctx, disputeUUID, claimer.ID)
	if err != nil {
		return err
	}

	if !participantClaimer.CanClaim() {
		return fmt.Errorf("%w: participant can't claime", ErrValidation)
	}

	opts := models.ParticipantUpdateOpts{
		ID:          participantClaimer.ID,
		IsClaimable: new(false),
		Seen:        new(true),
	}
	if err = s.participantUpdater.UpdateParticipant(ctx, opts); err != nil {
		return fmt.Errorf("failed to update claimer dispute status: %w", err)
	}
	return nil
}

func (s DisputeService) VoteDispute(ctx context.Context, disputeID string, voterUsername string, vote bool, boc string,
) error {
	if err := s.ensureTxSuccess(ctx, boc); err != nil {
		return err
	}

	if vote {
		return s.winDispute(ctx, disputeID, voterUsername)
	}
	return s.loseDispute(ctx, disputeID, voterUsername)
}

func (s DisputeService) winDispute(ctx context.Context, disputeID string, winnerUsername string) error {
	winner, err := s.userFinder.GetUserByUsername(ctx, winnerUsername)
	if err != nil {
		return fmt.Errorf("failed to get voter user: %w", err)
	}
	disputeUUID, err := uuid.Parse(disputeID)
	if err != nil {
		return fmt.Errorf("invalid dispute ID format: %w", err)
	}
	participantWinner, err := s.participantGetter.GetParticipant(ctx, disputeUUID, winner.ID)
	if err != nil {
		return err
	}

	opID, err := s.opponentGetter.GetOpponentID(ctx, disputeUUID, winner.ID)
	if err != nil {
		return fmt.Errorf("failed to get opponent ID: %w", err)
	}
	participantOpponent, err := s.participantGetter.GetParticipant(ctx, disputeUUID, opID)
	if err != nil {
		return err
	}

	var opts models.ParticipantUpdateOpts
	// first win
	if participantOpponent.Result == models.DisputesResultProcessed {
		opts.ID = participantWinner.ID
		opts.IsWin = new(true)
		opts.Result = new(models.DisputesResultAnswered)
		opts.Seen = new(true)
		err := s.participantUpdater.UpdateParticipant(ctx, opts)
		if err != nil {
			return fmt.Errorf("failed to update voter dispute status: %w", err)
		}
		return nil
	}

	// -- Opponent already voted ---
	var format string

	// -- win --
	if !participantOpponent.IsWin {
		opts.ID = participantOpponent.ID
		opts.Result = new(models.DisputesResultLose)
		opts.Status = new(models.DisputesStatusPassed)
		opts.IsClaimable = new(true)
		if err := s.participantUpdater.UpdateParticipant(ctx, opts); err != nil {
			return fmt.Errorf("failed to update opponent dispute status: %w", err)
		}
		format = "Ваше пари %s c пользователем %s завершилось поражением. Вы можете вернуть ваш депозит!"

		opts.ID = participantWinner.ID
		opts.Result = new(models.DisputesResultWin)
		opts.IsWin = new(true)
		if err := s.participantUpdater.UpdateParticipant(ctx, opts); err != nil {
			return fmt.Errorf("failed to update voter dispute status: %w", err)
		}
	}

	// -- investigation --
	if participantOpponent.IsWin {
		opts.ID = participantOpponent.ID
		opts.IsWin = new(true)
		opts.Result = new(models.DisputesResultEvidence)
		if err := s.participantUpdater.UpdateParticipant(ctx, opts); err != nil {
			return fmt.Errorf("failed to update opponent dispute status: %w", err)
		}
		format = "Ваше пари %s c пользователем %s требует доказательств. Внесите их в течении 24 часов."

		opts.ID = participantWinner.ID
		opts.Seen = new(true)
		if err := s.participantUpdater.UpdateParticipant(ctx, opts); err != nil {
			return fmt.Errorf("failed to update voter dispute status: %w", err)
		}
		nextDeadline := time.Now().Add(24 * time.Hour)
		if err := s.disputeCreator.UpdateDisputeNextDeadline(ctx, disputeUUID, nextDeadline); err != nil {
			return fmt.Errorf("failed to set next deadline for evidence stage: %w", err)
		}
	}

	// -- notification --
	opponent, err := s.userFinder.GetUserByID(ctx, opID)
	if err != nil {
		return fmt.Errorf("failed to get opponent user: %w", err)
	}
	if opponent.NotificationEnabled {
		dispute, err := s.disputeFinder.GetDisputeByID(ctx, disputeUUID)
		if err != nil {
			return fmt.Errorf("failed to get dispute: %w", err)
		}
		return s.msgSender.SendMessage(opponent.ChatID, fmt.Sprintf(format, dispute.Title, winner.Username))
	}
	return nil
}

func (s DisputeService) loseDispute(ctx context.Context, disputeID string, loserUsername string) error {
	loser, err := s.userFinder.GetUserByUsername(ctx, loserUsername)
	if err != nil {
		return fmt.Errorf("failed to get submitter user: %w", err)
	}
	disputeUUID, err := uuid.Parse(disputeID)
	if err != nil {
		return fmt.Errorf("invalid dispute ID format: %w", err)
	}
	participantLoser, err := s.participantGetter.GetParticipant(ctx, disputeUUID, loser.ID)
	if err != nil {
		return fmt.Errorf("failed to get submitter participant: %w", err)
	}

	opID, err := s.opponentGetter.GetOpponentID(ctx, disputeUUID, loser.ID)
	if err != nil {
		return fmt.Errorf("failed to get opponent ID: %w", err)
	}
	participantOpponent, err := s.participantGetter.GetParticipant(ctx, disputeUUID, opID)
	if err != nil {
		return fmt.Errorf("failed to get opponent participant: %w", err)
	}

	var opts models.ParticipantUpdateOpts
	// first lose
	if participantOpponent.Result == models.DisputesResultProcessed {
		opts.ID = participantLoser.ID
		opts.Result = new(models.DisputesResultAnswered)
		opts.Seen = new(true)
		if err = s.participantUpdater.UpdateParticipant(ctx, opts); err != nil {
			return fmt.Errorf("failed to update submitter participant: %w", err)
		}
		return nil
	}

	// -- Opponent already voted --
	var format string

	// lose
	if participantOpponent.IsWin {
		opts.ID = participantLoser.ID
		opts.Result = new(models.DisputesResultLose)
		opts.Status = new(models.DisputesStatusPassed)
		opts.IsClaimable = new(true)
		if err = s.participantUpdater.UpdateParticipant(ctx, opts); err != nil {
			return fmt.Errorf("failed to update submitter participant: %w", err)
		}
		opts.ID = participantOpponent.ID
		opts.Result = new(models.DisputesResultWin)
		if err = s.participantUpdater.UpdateParticipant(ctx, opts); err != nil {
			return fmt.Errorf("failed to update opponent participant: %w", err)
		}
		format = "Ваше пари %s c пользователем %s завершилось победой. Вы можете забрать свою награду!"
	}

	// draw
	if !participantOpponent.IsWin {
		opts.ID = participantLoser.ID
		opts.Result = new(models.DisputesResultDraw)
		opts.Status = new(models.DisputesStatusPassed)
		if err = s.participantUpdater.UpdateParticipant(ctx, opts); err != nil {
			return fmt.Errorf("failed to update submitter participant: %w", err)
		}
		opts.ID = participantOpponent.ID
		if err = s.participantUpdater.UpdateParticipant(ctx, opts); err != nil {
			return fmt.Errorf("failed to update opponent participant: %w", err)
		}
		format = "Ваше пари %s c пользователем %s завершилось вничью. Вы можете вернуть свою ставку и депозит!"
	}

	// -- notification --
	opponent, err := s.userFinder.GetUserByID(ctx, opID)
	if err != nil {
		return fmt.Errorf("failed to get opponent user: %w", err)
	}

	if opponent.NotificationEnabled {
		dispute, err := s.disputeFinder.GetDisputeByID(ctx, disputeUUID)
		if err != nil {
			return fmt.Errorf("failed to get dispute: %w", err)
		}
		return s.msgSender.SendMessage(opponent.ChatID, fmt.Sprintf(format, dispute.Title, loser.Username))
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
